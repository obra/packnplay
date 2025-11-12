# Cage: Sandboxed Container Launcher for Claude Code

## Overview

Cage launches Claude Code (and other tools) inside isolated Docker containers to sandbox their execution. The tool manages git worktrees, dev containers, environment setup, and container lifecycle to give Claude a safe workspace without risking the host system.

## Goals

- Sandbox Claude Code execution in Docker containers
- Automate worktree creation for safe project isolation
- Support both project-specific and default dev containers
- Handle environment proxying and configuration
- Make container management transparent and idiot-proof

## Architecture

### Core Components

**Single Go binary (`packnplay`):**
- Shells out to Docker CLI (not SDK) for compatibility with docker/podman/etc
- Simple command structure with flags
- No daemon or background processes
- Container state tracked via Docker itself (no separate state files)

**Container lifecycle:**
- Session-based: containers start with `packnplay run`, stop when command exits
- Multiple sessions can attach to running containers
- Containers persist after stopping (for log inspection)

**State management:**
- Docker labels track packnplay-managed containers
- `docker ps` is source of truth for running containers
- No complex state files or registries

### Command Structure

```bash
# Run command in container
packnplay run [flags] [command...]
  --path=<dir>           # Project path (default: pwd)
  --worktree=<name>      # Worktree name (creates if needed)
  --no-worktree          # Skip worktree, use directory directly
  --env KEY=value        # Additional env vars (repeatable)
  --verbose              # Show all docker/git commands

# Attach to running container (interactive shell)
packnplay attach [flags]

# Stop container
packnplay stop [flags]

# List all packnplay-managed containers
packnplay list
```

**Examples:**
```bash
packnplay run 'claude --dangerously-skip-permissions'
packnplay run --worktree=feature-auth --env DEBUG=1 claude
packnplay run --path=/home/jesse/myproject codex
packnplay run --no-worktree bash
packnplay attach --worktree=feature-auth
```

## Worktree Management

### Logic Flow

**With `--worktree=<name>`:**
1. Verify path is git repo (error if not)
2. Check if worktree exists: `git worktree list`
3. Create if missing: `git worktree add <repo>/../<project>-<name> -b <name>`
4. Mount worktree at `/workspace` in container

**With `--no-worktree`:**
- Use `--path` (or pwd) directly
- Mount at `/workspace`
- No git operations

**Default (no flags, git repo):**
1. Get current branch: `git branch --show-current`
2. Sanitize branch name for filesystem (replace `/` with `-`)
3. Check if worktree with that name exists
4. If exists: **ERROR** - "Worktree already exists. Use --worktree=<name> or --no-worktree"
5. If not: create and mount

**Default (no flags, not git repo):**
- Use directory directly (implicit `--no-worktree`)

### Worktree Naming

- Pattern: `<repo-dir>/../<project-name>-<worktree-name>`
- Example: `/home/jesse/myproject` + `feature-auth` → `/home/jesse/myproject-feature-auth`
- Sanitize special characters (replace `/` with `-`, remove other problematic chars)

## Dev Container Discovery

### Discovery Process

1. Check for `.devcontainer/devcontainer.json` in project path
2. If found:
   - Parse JSON for `image` or `dockerFile` field
   - Extract `remoteUser` (default to `devuser`)
   - Use project's dev container config
3. If not found: use packnplay's default dev container

### Image Handling

**For `image` field:**
1. Check if exists: `docker image inspect <image>`
2. If missing: `docker pull <image>` (show progress)
3. If pull fails: error with clear message

**For `dockerFile` field:**
1. Check for built image: `packnplay-<project-name>-devcontainer:latest`
2. If missing: `docker build -f .devcontainer/Dockerfile -t packnplay-<project-name>-devcontainer:latest .devcontainer`
3. If build fails: error with docker output

**Default container:**
- Reference: `mcr.microsoft.com/devcontainers/base:ubuntu` (or custom `packnplay-default:latest`)
- Auto-pull on first run
- Contains: git, curl, wget, build-essential, common dev tools
- Default user: `devuser` (UID 1000)

## Environment and Mounting

### Environment Variables

- Copy host environment (all vars from host shell)
- Add: `IS_SANDBOX=1`
- Add: `--env` flag values (override host vars)
- Pass to container via `docker run -e`

### Volume Mounts (with idmap)

All mounts use `--mount type=bind,...,idmap=uids=$(id -u)-$(id -u)-1000:gids=$(id -g)-$(id -g)-1000`

This maps host UID to container UID 1000, allowing container to run as `devuser` while files have correct ownership on host.

**1. ~/.claude directory:**
```bash
--mount type=bind,source=$HOME/.claude,target=/home/devuser/.claude,idmap=...
```
- Read-write mount
- Contains skills, plugins, history, etc.

**2. ~/.claude.json:**
- COPY (not mount) to avoid file locking
- `docker cp $HOME/.claude.json <container>:/home/devuser/.claude.json`
- After container starts
- Changes discarded when container stops

**3. Worktree/project directory:**
```bash
--mount type=bind,source=<worktree-path>,target=/workspace,idmap=...
```
- Read-write mount
- Container starts with `PWD=/workspace`

### Container User

- Run as `devuser` (UID 1000) in container
- idmap handles permission translation
- Container can install packages, run as non-root
- Files appear with correct ownership on host

## Container Lifecycle

### Container Naming

Pattern: `packnplay-<project-name>-<worktree-name>`

Examples:
- `packnplay-myproject-feature-auth`
- `packnplay-claude-launcher-main`

### Container Labels

Add to all containers:
```bash
--label managed-by=packnplay
--label packnplay-project=<project-name>
--label packnplay-worktree=<worktree-name>
```

### On `packnplay run`

1. Generate container name from project + worktree
2. Check if running: `docker ps --filter name=<name>`
3. If running: **ERROR** - "Container already running. Use 'packnplay attach' or 'packnplay stop'"
4. If not running:
   - Ensure image available (pull/build if needed)
   - Create container with all mounts and env vars
   - Copy ~/.claude.json into container
   - Execute command
5. Container stops when command exits (no `--rm`)

### On `packnplay attach`

1. Find container by name (from --path and --worktree)
2. If not running: **ERROR** - "No running container found"
3. If running: `docker exec -it <container> /bin/bash`

### On `packnplay stop`

1. Find container by name
2. `docker stop <container>`
3. `docker rm <container>`

### On `packnplay list`

1. `docker ps --filter label=managed-by=packnplay --format json`
2. Parse and display: project, worktree, container name, uptime

## Error Handling

### Fail Fast with Clear Messages

**Git errors:**
- Show git error + hint ("Is this a git repository?")
- Worktree collision: show exact command to resolve

**Docker errors:**
- Show docker error + hint ("Is docker running?")
- Image pull/build failures: show full output

**Container state errors:**
- Already running: show `packnplay attach` and `packnplay stop` options
- Not running: suggest `packnplay run`

### Verbosity

**Default:**
- Show high-level progress ("Pulling image...", "Creating worktree...")
- Hide command details

**`--verbose` flag:**
- Show all docker/git commands
- Show full command output
- For debugging

### Docker CLI Detection

1. Check for `docker` in PATH
2. If not found: check for `podman`
3. Allow override: `DOCKER_CMD=podman packnplay run ...`
4. Error if no compatible CLI found

### Exit Codes

- 0: success
- 1: user error (bad flags, collision, etc.)
- 2: external tool error (docker/git failed)
- Pass through exit code of wrapped command when applicable

## Implementation Notes

### Assumptions

- Linux 6.0.8+ (idmap support)
- Docker 28.5.1+ (idmap support)
- Go binary shells out to docker CLI
- No docker SDK dependency

### Branch Name Sanitization

- Replace `/` with `-`
- Remove or replace other filesystem-unfriendly characters
- Preserve readability where possible

### Container Image Strategy

For MVP:
- Support `image` field in devcontainer.json
- Support `dockerFile` field in devcontainer.json
- Use `mcr.microsoft.com/devcontainers/base:ubuntu` as default
- Future: support more devcontainer features

### Future Enhancements

Not in MVP:
- `packnplay clean` command (remove stopped containers)
- Long-running container mode (keep alive between sessions)
- More devcontainer.json feature support
- Container resource limits (CPU, memory)
- Network isolation options
