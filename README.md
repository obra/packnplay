
# packnplay

![packnplay hero image](docs/hero.jpeg)

packnplay launches commands (like Claude Code, Codex, Gemini) inside isolated Docker containers with automated worktree and dev container management.



packnplay is a containerization wrapper for your coding agents. It doesn't provide any level of intpospection or access control, other than running your agents in their own containers.

# Leash is a more powerful tool

[Leash by StrongDM](https://github.com/strongdm/leash) provides *comprehensive* access control and introspection for your coding agents activities. It also provides tools to sandbox your coding agents on macOS *without* containerization.  You probably want to use Leash.

# About packnplay

I built packnplay as a lightweight container/worktree launcher for my coding agents.

## Features

- **Sandboxed Execution**: Run AI coding assistants in isolated Docker containers
- **Smart User Detection**: Automatically detects and uses the correct container user with intelligent caching
- **Docker-Compatible Port Mapping**: Expose container ports to host with familiar `-p` syntax
- **Automatic Worktree Management**: Creates git worktrees in XDG-compliant locations (`~/.local/share/packnplay/worktrees`)
- **Dev Container Support**: Uses project's `.devcontainer/devcontainer.json` or feature-rich default with AI CLIs pre-installed
- **Credential Management**: Interactive first-run setup for git, GitHub CLI, GPG, npm, and AWS credentials
- **AWS Credentials Support**: Intelligent handling of AWS credentials including SSO, credential_process (granted.dev, aws-vault), and static credentials
- **Clean Environment**: Only passes safe environment variables (terminal/locale), no host pollution
- **macOS Keychain Integration**: Automatically extracts Claude and GitHub CLI credentials from macOS Keychain

## Installation

### Homebrew (macOS and Linux)

```bash
brew install obra/tap/packnplay
```

### Build from Source

**Prerequisites:**
- Go 1.21 or later
- Git (for version information)

**Build with version information:**
```bash
git clone https://github.com/obra/packnplay.git
cd packnplay
make build
./packnplay version
```

**Quick install without Makefile:**
```bash
go install github.com/obra/packnplay@latest
```

Note: Building with `make build` includes proper version, commit, and build date information. Direct `go build` or `go install` will show default values.

## Quick Start

On first run, packnplay will prompt you to configure which credentials to mount (git, GitHub CLI, GPG, npm, AWS). Your choices are saved to `~/.config/packnplay/config.json`.

```bash
# Run Claude Code in a sandboxed container (creates worktree automatically)
packnplay run claude

# Run in a specific worktree
packnplay run --worktree=feature-auth claude

# Run with all credentials enabled
packnplay run --all-creds claude

# Run with port mapping (expose container port 3000 to host port 8080)
packnplay run -p 8080:3000 npm start

# Multiple port mappings
packnplay run -p 8080:3000 -p 9000:9001 npm dev

# List running containers
packnplay list

# Stop all containers
packnplay stop --all
```

## Usage

### Basic Commands

```bash
# Run command in container (auto-creates worktree from current branch)
packnplay run <command>

# Use specific worktree (creates if doesn't exist, uses if exists)
packnplay run --worktree=<name> <command>

# Skip worktree, use current directory
packnplay run --no-worktree <command>

# Pass arguments to the command
packnplay run bash -c "echo hello && ls"

# Attach to running container (runs postAttachCommand)
packnplay attach --worktree=<name>

# Stop specific container
packnplay stop --worktree=<name>

# Stop all packnplay containers
packnplay stop --all

# List all running containers
packnplay list
```

### Credential Flags

Override default credential settings per-invocation:

```bash
# Enable specific credentials
packnplay run --git-creds claude           # Mount git config (~/.gitconfig)
packnplay run --ssh-creds claude           # Mount SSH keys (~/.ssh)
packnplay run --gh-creds claude            # Mount GitHub CLI credentials
packnplay run --gpg-creds claude           # Mount GPG keys for signing
packnplay run --npm-creds claude           # Mount npm credentials
packnplay run --aws-creds claude           # Mount AWS credentials
packnplay run --all-creds claude           # Mount all available credentials
```

#### AWS Credentials

The `--aws-creds` flag provides intelligent AWS credential handling with multiple strategies:

**Priority Order:**
1. **Static credentials** from environment variables (if `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY` are set)
2. **Dynamic credentials** via `credential_process` (if `AWS_PROFILE` is set and profile has `credential_process` configured)
3. **All other AWS environment variables** (`AWS_REGION`, `AWS_DEFAULT_REGION`, etc.)

**What happens:**
- Mounts `~/.aws` directory (read-write for SSO token refresh and CLI caching)
- If `AWS_PROFILE` is set and no static credentials exist:
  - Parses `~/.aws/config` (or `$AWS_CONFIG_FILE` if set)
  - Executes `credential_process` command on the host
  - Injects credentials into container as environment variables
- Passes all `AWS_*` environment variables (excluding host-specific container metadata)

**Supported credential tools:**
- AWS SSO
- [granted.dev](https://granted.dev)
- [aws-vault](https://github.com/99designs/aws-vault)
- Any tool using AWS `credential_process` standard

**Example:**

```bash
# With granted.dev
export AWS_PROFILE=my-profile
packnplay run --aws-creds aws s3 ls

# With static credentials
export AWS_ACCESS_KEY_ID=AKIA...
export AWS_SECRET_ACCESS_KEY=...
packnplay run --aws-creds aws s3 ls

# Override credentials per invocation
packnplay run --aws-creds --env AWS_REGION=eu-west-1 aws ec2 describe-instances
```

**Notes:**
- `credential_process` executes on the host with a 30-second timeout
- Credentials from `credential_process` may expire (snapshot at container start, not refreshed)
- User can override any AWS variable using `--env` flags (they take precedence)

### Port Mapping

Expose container ports to host using Docker-compatible syntax:

```bash
# Basic port mapping (host:container)
packnplay run -p 8080:3000 npm start

# Bind to specific host IP
packnplay run -p 127.0.0.1:8080:3000 npm dev

# Multiple ports
packnplay run -p 8080:3000 -p 9000:9001 -p 5432:5432 npm dev

# Specify protocol (TCP is default)
packnplay run -p 8080:3000/tcp -p 5353:53/udp npm start

# Same port on both sides
packnplay run -p 3000:3000 npm start
```

### Environment Variables

```bash
# Set specific environment variable
packnplay run --env DEBUG=1 claude

# Pass through variable from host
packnplay run --env EDITOR bash

# Multiple variables
packnplay run --env DEBUG=1 --env EDITOR bash
```

### AI Agent Support

packnplay provides **first-class support for 7 major AI coding assistants** with automatic configuration and credential management.

**Supported AI Agents:**
- **Claude Code** (`claude`) - Anthropic's coding assistant with `.claude/` config mounting
- **OpenCode AI** (`opencode`) - Modern AI coding platform with `.config/opencode/` support
- **OpenAI Codex** (`codex`) - OpenAI's code generation model
- **Google Gemini** (`gemini`) - Google's AI coding assistant
- **GitHub Copilot** (`copilot`) - GitHub's AI pair programmer
- **Qwen Code** (`qwen`) - Alibaba's coding AI
- **Sourcegraph Amp** (`amp`) - Code search and navigation AI

**Automatic Configuration:**
- **Config Directories**: Mounts appropriate config directories (`.claude/`, `.config/opencode/`, etc.)
- **API Keys**: Passes through relevant environment variables (`ANTHROPIC_API_KEY`, `OPENCODE_API_KEY`, etc.)
- **Authentication**: Handles GitHub tokens, custom API endpoints
- **Persistence**: Preserves settings and authentication across container runs

**Example Usage:**
```bash
# Claude Code with automatic config mounting
packnplay run claude

# OpenCode AI with config and credential support
packnplay run opencode

# Any AI agent with custom environment
packnplay run --env OPENCODE_CONFIG=/custom/path opencode
```

## How It Works

### Smart User Detection

packnplay automatically detects the correct user for any Docker image:

**Detection Priority:**
1. **devcontainer.json**: Respects `remoteUser` field if specified
2. **Cached Results**: Fast lookup by Docker image ID (no repeated detection)
3. **Runtime Detection**: Asks container directly: `whoami && echo $HOME`
4. **Safe Fallback**: Uses `root` if detection fails

**Benefits:**
- **Universal compatibility**: Works with node, ubuntu, python, custom images
- **Performance optimized**: Caches results to avoid repeated container starts
- **No guessing**: Direct container interrogation eliminates assumptions
- **Standards compliant**: Honors devcontainer.json when present

### Worktree Management

Pack 'n Play creates git worktrees in XDG-compliant locations for isolation:

- **Location**: `~/.local/share/packnplay/worktrees/<project>/<worktree>` (or `$XDG_DATA_HOME/packnplay/worktrees`)
- **Auto-create**: If you're in a git repo without `--worktree` flag, uses current branch name
- **Explicit**: `--worktree=<name>` creates new or connects to existing worktree
- **Skip**: `--no-worktree` uses current directory without git worktree
- **Auto-connect**: If container already running for a worktree, automatically connects to it
- **Git integration**: Main repo's `.git` directory mounted so git commands work correctly

### Dev Container Support

packnplay achieves **97% Microsoft devcontainer specification compliance**—production-ready support for reproducible development environments.

**Quick Example:**
```json
{
  "name": "Node.js Development",
  "image": "mcr.microsoft.com/devcontainers/base:ubuntu",
  "features": {
    "ghcr.io/devcontainers/features/node:1": {
      "version": "20",
      "nodeGypDependencies": true
    },
    "ghcr.io/devcontainers/features/docker-in-docker:2": {
      "version": "latest",
      "enableNonRootDocker": true
    }
  },
  "workspaceFolder": "/workspace",
  "forwardPorts": [3000],
  "containerEnv": {
    "NODE_ENV": "development"
  },
  "mounts": [
    "type=tmpfs,target=/tmp/fast-storage"
  ],
  "onCreateCommand": "npm install",
  "postCreateCommand": "npm run build",
  "postStartCommand": "echo 'Development environment ready'"
}
```

#### **🎯 Supported Features**

**✅ Container Configuration:**
- `name` - Container display name
- `image` - Any Docker registry image
- `dockerfile` / `build` - Custom builds with args, target, context, cacheFrom
- `workspaceFolder`, `workspaceMount` - Workspace configuration
- `remoteUser` - User management (auto-detected when omitted)

**✅ Microsoft DevContainer Features:**
- **OCI Registry**: Full `ghcr.io/devcontainers/features/*` support
- **Local Features**: `.devcontainer/local-features/`
- **Option Validation**: String, boolean, number types with enum checking
- **Dependencies**: Complex chains via `dependsOn` object format
- **Properties**: `privileged`, `capAdd`, `securityOpt`, `init`, `entrypoint`, `mounts`
- **Variables**: `${devcontainerId}`, `${localWorkspaceFolder}`, `${containerWorkspaceFolder}`

**✅ Environment Variables:**
- `containerEnv` - Runtime variables with full substitution
- **Built-in**: `_REMOTE_USER`, `_REMOTE_USER_HOME`, `_CONTAINER_USER`
- **Patterns**: `${localEnv:VAR}`, `${containerWorkspaceFolder}`, `${devcontainerId}`

**✅ Port Forwarding:**
- `forwardPorts` - Secure localhost binding (matches Microsoft behavior)
- **Formats**: Integers (`3000`), strings (`"8080:3000"`), IP binding (`"127.0.0.1:8080:3000"`)

**✅ Lifecycle Commands:**
- `initializeCommand` - Runs on host before container creation
- `onCreateCommand` - Runs once on creation (tracks changes)
- `updateContentCommand` - Content change hooks
- `postCreateCommand` - Runs once after creation
- `postStartCommand` - Runs every container start
- `postAttachCommand` - Runs on `packnplay attach`
- **Formats**: String (shell), array (exec), object (parallel)
- **Execution order**: Feature commands precede user commands

**✅ Advanced Features:**
- `mounts` - Volume mounts with variable substitution
- `runArgs` - Additional Docker arguments
- **Signal Handling**: Graceful shutdown with SIGTERM
- **Feature Integration**: Complete metadata processing

#### **🚧 Minor Gaps (2% of specification)**

**Would complete specification:**
- ❌ `remoteEnv` - Container-computed environment variables
- ❌ Container restart (recreates stopped containers instead)

**Advanced features:**
- ❌ HTTPS tarball features (`https://example.com/feature.tgz`)
- ❌ Private feature authentication
- ❌ Lockfile support
- ❌ `portsAttributes` configuration

**See [GitHub Issues](https://github.com/obra/packnplay/issues?q=is%3Aissue+is%3Aopen+label%3Adevcontainer) for roadmap.**

**📖 Full Documentation:** See [DevContainer Guide](docs/DEVCONTAINER_GUIDE.md) for complete reference with examples.

**Fallback:**
If `.devcontainer/devcontainer.json` not found, uses `ghcr.io/obra/packnplay/devcontainer:latest`

**Default container architecture:**
- **Foundation**: Microsoft devcontainer features (reliable, maintained, consistent)
- **Languages**: Node.js LTS, Python 3.11+, Go latest, Rust latest
- **Cloud CLIs**: AWS, Azure, Google Cloud, GitHub
- **Development Tools**: Git, build-essential, vim, nano, jq, yq, curl, wget, make
- **Shell**: Zsh with oh-my-zsh (better developer experience)

**AI Coding Assistant Ecosystem:**
- **Claude Code** - Anthropic's premier coding assistant
- **OpenCode AI** - Modern AI coding platform with advanced features
- **OpenAI Codex** - OpenAI's foundational code model
- **Google Gemini** - Google's multimodal AI assistant
- **GitHub Copilot** - GitHub's AI pair programmer
- **Qwen Code** - Alibaba's efficient coding model
- **Sourcegraph Amp** - Code search and navigation AI

**Configuration Management:**
- **Automatic mounting**: Each AI tool's config directory mounted seamlessly
- **Credential passthrough**: API keys and tokens handled securely
- **Persistent settings**: Configuration survives container recreation
- **Environment flexibility**: Custom configs via environment variables

### Default Container Configuration

**Configurable Default Image:**
packnplay supports custom default containers - you're not limited to the packnplay default. Set any image from any registry:

```json
{
  "default_container": {
    "image": "my-company/dev-environment:latest",
    "check_for_updates": true,
    "auto_pull_updates": false,
    "check_frequency_hours": 24
  }
}
```

**Version Update Notifications:**
When enabled, packnplay checks for new versions and shows detailed notifications:

```bash
ℹ️  New version available: ghcr.io/obra/packnplay/devcontainer:latest
   Current: abc123de (2 days old)
   Latest:  xyz789gh (just released)

   To update: packnplay refresh-container
```

**Features:**
- **Smart notifications**: Only notifies once per version, respects frequency settings
- **Detailed version info**: Shows current vs latest with digests and age
- **User control**: Manual refresh command with `packnplay refresh-container`
- **Configurable checking**: Enable/disable update checking and auto-pull behavior
- **Non-intrusive**: Checking happens in background, notifications only when needed

## Rebuilding the Default Container

See [.devcontainer/README.md](.devcontainer/README.md) for instructions on building and publishing the default container image.

### Credential Handling

**Interactive Setup (first run):**
On first run, packnplay prompts you to choose which credentials to enable by default using a beautiful terminal UI.

**Credentials are mounted read-only for security:**
- **Git**: `~/.gitconfig` (git user configuration)
- **SSH**: `~/.ssh` (SSH keys for authentication to servers and repos)
- **GitHub CLI**: `~/.config/gh` (copied from Keychain on macOS, mounted on Linux)
- **GPG**: `~/.gnupg` (for commit signing)
- **npm**: `~/.npmrc` (for authenticated package operations)

**macOS Keychain Integration:**
- Claude credentials automatically extracted from Keychain (`Claude Code-credentials`)
- GitHub CLI credentials extracted and base64-decoded from Keychain (`gh:github.com`)
- Credentials copied into container (not mounted) to avoid file locking

### File Mounts

**Host Path Preservation:**
packnplay mounts your project at the **exact same path** inside the container as it exists on your host. This ensures absolute path consistency between host and container environments.

- `~/.claude` → mounted read-write (skills, plugins, history)
- `~/.claude.json` → copied into container (avoids file lock conflicts)
- **Project directory** → mounted at identical host path (no `/workspace` abstraction)
- Main repo `.git` → mounted at its real path (git commands work)

**Examples:**
```bash
# Host
/Users/jesse/Documents/GitHub/myproject

# Container (same path!)
/Users/jesse/Documents/GitHub/myproject
```

**Benefits:**
- Absolute paths work identically in host and container
- Git worktree references maintain correct paths
- IDE configurations with hardcoded paths work consistently
- Symlinks preserve correct relative relationships
- Cross-container workflows see consistent paths

### Environment Variables

**Safe whitelist approach:**
- Only `TERM`, `LANG`, `LC_*`, `COLORTERM` passed from host
- `HOME=/home/vscode` set in container
- `IS_SANDBOX=1` marker added
- `PATH` uses container default (not polluted from host)
- Use `--env KEY=value` or `--env KEY` to pass additional variables

### Container Lifecycle

- **Persistent containers**: Started with `packnplay run`, stay running after command exits
- **Auto-attach**: Running `packnplay run` again connects to existing container
- **Labeled**: All containers tagged with `managed-by=packnplay` for tracking
- **Clean**: Use `packnplay stop --all` to stop and remove all packnplay containers

## Requirements

- **Docker**: Docker Desktop on macOS, or Docker Engine on Linux
- **Git**: For worktree functionality
- **Go 1.23+**: For building from source
- **Optional**: GitHub CLI (`gh`) for GitHub operations

## Configuration

### Interactive Configuration

**Reconfigure packnplay safely:**
```bash
packnplay configure                    # Edit all settings in logical flow
```

**Safe Editing:**
- ✅ **Preserves custom settings** not shown in UI (env configs, advanced settings)
- ✅ **Shows current values** as defaults in forms
- ✅ **Complete visibility** - see all your current configuration
- ✅ **No data loss** - manual edits and version tracking are preserved
- ✅ **Logical flow** - runtime → credentials → default container → update settings

### Config File

`~/.config/packnplay/config.json` (XDG-compliant):

```json
{
  "container_runtime": "docker",
  "default_credentials": {
    "git": true,
    "ssh": true,
    "gh": true,
    "gpg": false,
    "npm": false
  },
  "env_configs": {
    "z.ai": {
      "name": "Z.AI Claude",
      "description": "Z.AI's Claude implementation with GLM models",
      "env_vars": {
        "ANTHROPIC_AUTH_TOKEN": "${Z_AI_API_KEY}",
        "ANTHROPIC_BASE_URL": "https://api.z.ai/api/anthropic",
        "API_TIMEOUT_MS": "3000000",
        "ANTHROPIC_DEFAULT_OPUS_MODEL": "GLM-4.6",
        "ANTHROPIC_DEFAULT_SONNET_MODEL": "GLM-4.6",
        "ANTHROPIC_DEFAULT_HAIKU_MODEL": "GLM-4.5-Air"
      }
    },
    "anthropic-work": {
      "name": "Anthropic API (Work)",
      "description": "Work API key with standard models",
      "env_vars": {
        "ANTHROPIC_API_KEY": "${ANTHROPIC_WORK_API_KEY}"
      }
    },
    "claude-personal": {
      "name": "Claude Personal",
      "description": "Personal API key setup",
      "env_vars": {
        "ANTHROPIC_API_KEY": "${ANTHROPIC_PERSONAL_API_KEY}",
        "ANTHROPIC_DEFAULT_SONNET_MODEL": "claude-3-5-sonnet-20241022"
      }
    }
  }
}
```

Created interactively on first run. Edit manually or delete to reconfigure.

### Environment Configurations

Environment configs let you define different API setups and switch between them:

```bash
# Use Z.AI endpoints and models
packnplay run --config=z.ai claude

# Use work API key
packnplay run --config=anthropic-work claude

# Use personal API key with specific model
packnplay run --config=claude-personal claude
```

**Variable substitution:** Use `${VAR_NAME}` in env_vars to substitute from host environment.

**Required host environment variables:**
```bash
export Z_AI_API_KEY="your-z-ai-key"
export ANTHROPIC_WORK_API_KEY="sk-ant-work-key"
export ANTHROPIC_PERSONAL_API_KEY="sk-ant-personal-key"
```

### Environment Variables

- `DOCKER_CMD`: Override docker command (e.g., `DOCKER_CMD=podman packnplay run ...`)

- `XDG_DATA_HOME`: Override data directory (default: `~/.local/share`)
- `XDG_CONFIG_HOME`: Override config directory (default: `~/.config`)

**Note:** Apple Container support was disabled due to incompatibilities. See [issue #1](https://github.com/obra/packnplay/issues/1) for details. Use Docker Desktop or Podman on macOS.

## Examples

```bash
# First run - interactive credential setup, then run Claude
packnplay run claude

# Run in specific worktree with all credentials
packnplay run --worktree=bug-fix --all-creds claude

# Run with custom environment variables
packnplay run --env DEBUG=1 --env EDITOR bash -c "echo \$EDITOR"

# Get a shell in the container
packnplay run --worktree=feature bash

# Run command in existing container (auto-connects)
packnplay run --worktree=feature npm test

# Attach with interactive shell
packnplay attach --worktree=feature

# List all running containers
packnplay list

# Stop specific container
packnplay stop --worktree=feature

# Stop all packnplay containers
packnplay stop --all
```

## Testing

packnplay has comprehensive test coverage with both unit tests and end-to-end integration tests.

### Running Tests

```bash
# Run all unit tests (fast, no Docker required)
go test -short ./...

# Run all tests including E2E tests (requires Docker daemon)
go test ./...

# Run only E2E tests
go test ./pkg/runner -run TestE2E

# Run specific E2E test
go test -v ./pkg/runner -run TestE2E_OnCreateCommand_RunsOnce
```

### Test Requirements

- **Unit Tests**: No dependencies, run with `-short` flag
- **E2E Tests**: Require Docker daemon running
  - Tests skip gracefully if Docker unavailable
  - Tests clean up all containers and metadata automatically
  - Tests use real Docker (no mocks)

For more details, see [Testing Guide](docs/TESTING.md).

## Credits

- The core ergonomics of the packnplay tool were heavily inspired by [StrongDM Leash](https://github.com/strongdm/leash), which has actual authorization and visibility features that make your use of agents safer, rather than just being wrapper around `docker` commandline invocations like this tool.
- Hero image contributed by [Dan Shapiro](https://github.com/danshapiro)

## License

MIT
