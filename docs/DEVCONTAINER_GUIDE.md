# DevContainer Support Guide

Packnplay provides comprehensive support for devcontainer.json configuration files, enabling reproducible development environments with intelligent lifecycle management.

## Quick Start

Create `.devcontainer/devcontainer.json` in your project:

```json
{
  "image": "mcr.microsoft.com/devcontainers/base:ubuntu",
  "forwardPorts": [3000, 8080],
  "containerEnv": {
    "NODE_ENV": "development",
    "API_URL": "http://localhost:3000"
  },
  "postCreateCommand": "npm install",
  "postStartCommand": "npm run dev"
}
```

Then run:
```bash
packnplay run bash
```

Packnplay will:
1. Pull/build the specified image
2. Forward ports 3000 and 8080
3. Set environment variables
4. Run `npm install` once on first creation
5. Run `npm run dev` every time the container starts

## Supported Fields

### Image Configuration

#### `image`
Pre-built Docker image to use.

```json
{
  "image": "node:18-bullseye"
}
```

#### `dockerfile` (or `build.dockerfile`)
Path to Dockerfile relative to `.devcontainer/` directory.

```json
{
  "dockerfile": "Dockerfile"
}
```

#### `build`
Advanced build configuration with arguments, targets, and caching.

```json
{
  "build": {
    "dockerfile": "Dockerfile.dev",
    "context": "..",
    "args": {
      "VARIANT": "18-bullseye",
      "NODE_VERSION": "18.16.0"
    },
    "target": "development",
    "cacheFrom": [
      "ghcr.io/myorg/cache:latest"
    ],
    "options": ["--pull"]
  }
}
```

**Fields:**
- `dockerfile` - Path to Dockerfile
- `context` - Build context path (default: `.devcontainer`)
- `args` - Build-time variables (⚠️ don't use for secrets!)
- `target` - Multi-stage build target
- `cacheFrom` - Images to use for layer caching (string or array)
- `options` - Additional docker build flags

⚠️ **Security Warning**: Build args are persisted in image metadata. Use `containerEnv` with variable substitution for secrets.

### User Configuration

#### `remoteUser`
User to run commands as in the container (used for `docker exec`).

```json
{
  "remoteUser": "node"
}
```

If not specified, packnplay auto-detects the appropriate user.

#### `containerUser`
User for container creation (used for `docker run --user`). Different from `remoteUser`.

```json
{
  "containerUser": "root",
  "remoteUser": "node"
}
```

**Use Case:** Run container as root for setup, but exec commands as non-root user.

#### `updateRemoteUserUID`
Sync container user's UID/GID to match host user (Linux only).

```json
{
  "remoteUser": "vscode",
  "updateRemoteUserUID": true
}
```

**Behavior:**
- Only applies on Linux hosts (Docker Desktop handles this automatically)
- Updates container user's UID/GID to match host user
- Fixes file permission issues when sharing volumes

#### `userEnvProbe`
How to probe the user's environment for shell configuration.

```json
{
  "userEnvProbe": "loginInteractiveShell"
}
```

**Values:**
- `none` - No shell probing
- `loginShell` - Use login shell (`-l`)
- `interactiveShell` - Use interactive shell (`-i`)
- `loginInteractiveShell` - Use both (`-li`, default)

### Environment Variables

#### `containerEnv`
Environment variables set when container is created.

```json
{
  "containerEnv": {
    "DATABASE_URL": "postgres://localhost:5432/db",
    "LOG_LEVEL": "debug"
  }
}
```

#### `remoteEnv`
Environment variables that can reference `containerEnv` values.

```json
{
  "containerEnv": {
    "API_HOST": "api.example.com"
  },
  "remoteEnv": {
    "API_URL": "https://${containerEnv:API_HOST}/v1"
  }
}
```

**Priority Order** (lowest to highest):
1. Default environment (TERM, LANG, etc.)
2. Agent API keys (ANTHROPIC_API_KEY, etc.)
3. AWS credentials
4. **devcontainer.json env vars** ⬅️ from containerEnv/remoteEnv
5. CLI `--env` flags ⬅️ highest priority, overrides all

### Workspace Configuration

#### `workspaceFolder`
Path inside the container where the workspace should be mounted.

```json
{
  "workspaceFolder": "/workspace"
}
```

If not specified, defaults to `/workspace`.

#### `workspaceMount`
Custom mount string for the workspace using Docker `--mount` syntax.

```json
{
  "workspaceFolder": "/workspace",
  "workspaceMount": "source=${localWorkspaceFolder},target=/workspace,type=bind,consistency=cached"
}
```

**Notes:**
- Requires `workspaceFolder` to be set
- Supports variable substitution
- Use for advanced mount options (consistency, caching)

### Port Forwarding

#### `forwardPorts`
Ports to expose from the container.

```json
{
  "forwardPorts": [3000, 8080, "5432:5432"]
}
```

**Formats:**
- Integer: `3000` → maps to `"3000:3000"`
- String: Full Docker syntax
  - `"8080:80"` - Map container port 80 to host port 8080
  - `"127.0.0.1:9000:9000"` - Bind to specific IP
  - `"3000-3010:3000-3010"` - Port ranges
  - `"8080:80/tcp"` - Specify protocol

**Priority Order:**
1. **devcontainer.json ports** (applied first)
2. CLI `-p` flags (applied second, can override)

#### `portsAttributes`
Per-port configuration for labels, protocols, and behavior.

```json
{
  "forwardPorts": [3000, 8080],
  "portsAttributes": {
    "3000": {
      "label": "Frontend",
      "protocol": "http",
      "onAutoForward": "notify"
    },
    "8080": {
      "label": "API",
      "protocol": "https",
      "requireLocalPort": true,
      "elevateIfNeeded": false
    }
  }
}
```

**Fields:**
- `label` - Display name for the port
- `protocol` - `http` or `https`
- `onAutoForward` - `notify`, `openBrowser`, `openBrowserOnce`, `openPreview`, `silent`, `ignore`
- `requireLocalPort` - Fail if the specific local port is unavailable
- `elevateIfNeeded` - Elevate permissions to bind privileged ports

#### `otherPortsAttributes`
Default attributes for ports not explicitly listed in `portsAttributes`.

```json
{
  "portsAttributes": {
    "3000": { "label": "App" }
  },
  "otherPortsAttributes": {
    "onAutoForward": "silent"
  }
}
```

### Docker Compose Orchestration

Use Docker Compose instead of a single image/dockerfile.

#### `dockerComposeFile`
Path to Docker Compose file(s).

```json
{
  "dockerComposeFile": "docker-compose.yml",
  "service": "app"
}
```

Or multiple files:
```json
{
  "dockerComposeFile": ["docker-compose.yml", "docker-compose.dev.yml"],
  "service": "app"
}
```

#### `service`
Which service to connect to (required with `dockerComposeFile`).

```json
{
  "dockerComposeFile": "docker-compose.yml",
  "service": "web"
}
```

#### `runServices`
Which services to start (defaults to all).

```json
{
  "dockerComposeFile": "docker-compose.yml",
  "service": "app",
  "runServices": ["app", "db", "redis"]
}
```

**Notes:**
- `dockerComposeFile` is mutually exclusive with `image`/`dockerfile`
- Features are not supported with Docker Compose (install in your service image)
- `shutdownAction: "stopCompose"` stops all services on exit

### Lifecycle Commands

#### `initializeCommand`
Runs **on the host** before the container is created.

```json
{
  "initializeCommand": "npm install"
}
```

**Behavior:**
- Executes on the host machine (not in container)
- Runs before container creation
- Ideal for: downloading dependencies, generating files, pre-build setup
- **Security warning**: Executes code from devcontainer.json on your host

**Important Notes:**
- This command runs with your host user permissions
- Use with caution as it executes arbitrary commands on your host system
- The working directory is the project directory (where devcontainer.json is located)

#### `onCreateCommand`
Runs **once** when the container is first created.

```json
{
  "onCreateCommand": "npm install"
}
```

**Behavior:**
- Executes on first `packnplay run`
- Skipped on subsequent runs
- Re-runs if command content changes
- Ideal for: dependency installation, initial setup

#### `postCreateCommand`
Runs **once** after the container is created.

```json
{
  "postCreateCommand": "npm run build"
}
```

**Behavior:**
- Executes after `onCreateCommand`
- Skipped on subsequent runs
- Re-runs if command content changes
- Ideal for: building assets, database migrations

#### `updateContentCommand`
Runs when container content is updated (similar to onCreate but for content updates).

```json
{
  "updateContentCommand": "npm run generate"
}
```

**Behavior:**
- Executes after onCreateCommand
- Skipped on subsequent runs unless content changes
- Re-runs if command content changes
- Ideal for: code generation, content synchronization

#### `postStartCommand`
Runs **every time** the container starts.

```json
{
  "postStartCommand": "npm run dev"
}
```

**Behavior:**
- Executes every time
- Not tracked (always runs)
- Ideal for: starting development servers, watch mode

#### `postAttachCommand`
Runs when attaching to the container (via `packnplay attach`).

```json
{
  "postAttachCommand": "echo 'Welcome back!'"
}
```

**Behavior:**
- Executes on every `packnplay attach` invocation
- Runs before entering the container shell
- Ideal for: status messages, environment refresh

#### Command Formats

All lifecycle commands support three formats:

**String (Shell):**
```json
{
  "postCreateCommand": "npm install && npm run build"
}
```
Executed via `sh -c`, supports shell features (pipes, &&, etc.)

**Array (Direct Exec):**
```json
{
  "postCreateCommand": ["npm", "install", "--production"]
}
```
Executed directly without shell, safer for complex arguments.

**Object (Parallel):**
```json
{
  "postStartCommand": {
    "server": "npm run server",
    "watch": "npm run watch",
    "docs": ["python", "-m", "http.server", "8000"]
  }
}
```
Executes multiple commands in parallel. Values can be strings or arrays.

**Note:** All lifecycle commands support parallel execution via object format, including `initializeCommand` which runs parallel tasks on the host.

### Lifecycle Control

#### `waitFor`
Which lifecycle command to wait for before considering the container ready.

```json
{
  "waitFor": "postCreateCommand"
}
```

**Values:**
- `onCreateCommand`
- `updateContentCommand`
- `postCreateCommand`
- `postStartCommand`

**Note:** All commands run synchronously before the user command, so this is primarily for compatibility and documentation.

#### `overrideCommand`
Whether to override the container's CMD with the user command.

```json
{
  "overrideCommand": true
}
```

**Behavior:**
- `true` (default): User command replaces container CMD
- `false`: Container runs its default CMD

#### `shutdownAction`
What to do when the user exits the container.

```json
{
  "shutdownAction": "stopContainer"
}
```

**Values:**
- `none` (default): Leave container running
- `stopContainer`: Stop the container on exit
- `stopCompose`: Stop all Docker Compose services on exit

### Host Requirements

#### `hostRequirements`
Advisory minimum host system requirements.

```json
{
  "hostRequirements": {
    "cpus": 4,
    "memory": "8gb",
    "storage": "32gb",
    "gpu": true
  }
}
```

**Fields:**
- `cpus` - Minimum CPU cores
- `memory` - Minimum RAM (e.g., "8gb")
- `storage` - Minimum disk space (e.g., "32gb")
- `gpu` - Requires GPU

**Note:** These are advisory only. Packnplay validates and warns but doesn't enforce.

### Custom Mounts

#### `mounts`
Additional volume mounts beyond the workspace mount.

```json
{
  "mounts": [
    "source=/var/run/docker.sock,target=/var/run/docker.sock,type=bind",
    "source=my-volume,target=/data,type=volume",
    "type=tmpfs,target=/tmp"
  ]
}
```

**Mount Syntax:**
Uses Docker's `--mount` syntax (not `-v`):
- `source=<host-path>,target=<container-path>,type=bind` - Bind mount
- `source=<volume-name>,target=<container-path>,type=volume` - Named volume
- `type=tmpfs,target=<container-path>` - Temporary filesystem

**Variable Substitution:**
Mount paths support variable substitution:
```json
{
  "mounts": [
    "source=${localWorkspaceFolder}/config,target=/app/config,type=bind"
  ]
}
```

**Common Use Cases:**
- Docker socket access for Docker-in-Docker
- Persistent data volumes
- Configuration file sharing
- Temporary high-performance storage (tmpfs)

### Custom Run Arguments

#### `runArgs`
Additional Docker run arguments for container creation.

```json
{
  "runArgs": ["--memory=2g", "--cpus=2", "--label", "env=dev"]
}
```

**Common Use Cases:**
- Resource limits: `--memory=2g`, `--cpus=2`
- Custom labels: `--label key=value`
- Security options: `--cap-add=SYS_PTRACE`
- Network configuration: `--network=host`

**Variable Substitution:**
RunArgs support variable substitution:
```json
{
  "runArgs": ["--label", "project=${containerWorkspaceFolderBasename}"]
}
```

**Precedence:**
runArgs are added before the image name in the `docker run` command, allowing you to pass any Docker CLI flags.

### Features

#### `features`
Install pre-packaged development tools and configurations from community-maintained features.

```json
{
  "image": "mcr.microsoft.com/devcontainers/base:ubuntu",
  "features": {
    "ghcr.io/devcontainers/features/node:1": {
      "version": "18"
    },
    "ghcr.io/devcontainers/features/docker-in-docker:2": {},
    "ghcr.io/devcontainers/features/common-utils:2": {
      "installZsh": true,
      "installOhMyZsh": true
    }
  }
}
```

**Feature Syntax:**
Features are referenced by their container registry path with semantic versioning:
- `ghcr.io/devcontainers/features/<feature-name>:<major-version>`
- Pin to major: `:1`, minor: `:1.0`, or patch: `:1.0.0`
- Omit version for `:latest` tag

**Common Community Features:**

**Node.js (`node:1`):**
```json
{
  "features": {
    "ghcr.io/devcontainers/features/node:1": {
      "version": "18",
      "nodeGypDependencies": true
    }
  }
}
```
Installs Node.js, npm, and optionally build dependencies.

**Docker-in-Docker (`docker-in-docker:2`):**
```json
{
  "features": {
    "ghcr.io/devcontainers/features/docker-in-docker:2": {
      "version": "latest",
      "moby": true
    }
  }
}
```
Enables running Docker commands inside the container.

**Common Utilities (`common-utils:2`):**
```json
{
  "features": {
    "ghcr.io/devcontainers/features/common-utils:2": {
      "installZsh": true,
      "installOhMyZsh": true,
      "upgradePackages": true
    }
  }
}
```
Installs common CLI tools, shells, and utilities.

**Local Features:**
Reference local feature directories:
```json
{
  "features": {
    "/path/to/local/feature": {},
    "./local-feature": {}
  }
}
```

**Local Feature Structure:**
```
my-feature/
├── devcontainer-feature.json  # Metadata
└── install.sh                 # Installation script
```

**Feature Discovery:**
Browse available features:
- Official features: https://github.com/devcontainers/features
- Community registry: https://containers.dev/features

#### Private Features

Packnplay supports private features from authenticated OCI registries. Authentication uses standard Docker credentials, requiring no additional configuration.

**Setup:**
```bash
# Authenticate to your private registry
docker login ghcr.io
# or
docker login myregistry.com
```

**Usage:**
```json
{
  "features": {
    "ghcr.io/myorg/private-feature:1": {
      "option": "value"
    }
  }
}
```

**How It Works:**
- Packnplay uses ORAS to pull OCI features
- ORAS automatically inherits Docker credentials from `~/.docker/config.json`
- Credential helpers (Docker Desktop, cloud provider helpers) are automatically supported
- No additional configuration or environment variables needed

**Supported Registries:**
- GitHub Container Registry (ghcr.io)
- Docker Hub
- Azure Container Registry
- Google Container Registry
- AWS ECR (with credential helper)
- Any OCI-compliant registry

If you encounter authentication issues, ensure you've logged in to the registry:
```bash
docker login <registry-url>
```

#### `overrideFeatureInstallOrder`

Override the automatic dependency-based installation order for features. This allows manual control of feature installation sequence, bypassing dependency resolution.

**Usage:**
```json
{
  "features": {
    "ghcr.io/devcontainers/features/common-utils:2": {},
    "ghcr.io/devcontainers/features/node:1": {},
    "ghcr.io/devcontainers/features/docker-in-docker:2": {}
  },
  "overrideFeatureInstallOrder": [
    "ghcr.io/devcontainers/features/docker-in-docker:2",
    "ghcr.io/devcontainers/features/common-utils:2",
    "ghcr.io/devcontainers/features/node:1"
  ]
}
```

**Behavior:**
- If specified, features are installed in the exact order listed
- Features not in the override list are installed after specified features
- If not specified or empty, uses automatic dependency resolution based on `dependsOn` and `installsAfter` metadata
- Warning is printed if the override order doesn't include all features

**Use Cases:**
- Working around feature dependency issues
- Optimizing build time by reordering features
- Testing different installation sequences
- Forcing specific installation order when automatic resolution is incorrect

**Example with Partial Order:**
```json
{
  "features": {
    "feature-a": {},
    "feature-b": {},
    "feature-c": {}
  },
  "overrideFeatureInstallOrder": ["feature-c", "feature-a"]
}
```
Result: `feature-c` → `feature-a` → `feature-b` (feature-b appended at end)

#### Feature Options Processing

Packnplay fully supports the devcontainer features specification for option processing. Feature options are automatically converted to environment variables that the feature's install script can use.

**How Options Work:**
1. Options defined in `devcontainer-feature.json` specify available configuration
2. User-provided values override defaults from the feature metadata
3. Options are converted to environment variables per specification (uppercase, dashes to underscores)
4. Environment variables are available during feature installation

**Example:**
```json
{
  "features": {
    "ghcr.io/devcontainers/features/node:1": {
      "version": "18.20.0",
      "nodeGypDependencies": true
    }
  }
}
```

During installation, the node feature receives:
- `VERSION=18.20.0`
- `NODEGYPDEPENDENCIES=true`

#### Feature Lifecycle Hooks

Features can contribute lifecycle commands that execute at specific points in the container lifecycle. **Feature commands always execute before user commands**, ensuring features can set up the environment properly.

**Execution Order:**
1. Feature `onCreateCommand` (all features, in installation order)
2. User `onCreateCommand`
3. Feature `postCreateCommand` (all features, in installation order)
4. User `postCreateCommand`
5. Feature `postStartCommand` (all features, in installation order)
6. User `postStartCommand`

**Example Feature with Lifecycle Hook:**
```json
// devcontainer-feature.json
{
  "id": "my-feature",
  "version": "1.0.0",
  "postCreateCommand": "echo 'Feature setup complete' > /tmp/feature-status.txt"
}
```

When combined with user commands:
```json
{
  "features": {
    "./local-features/my-feature": {}
  },
  "postCreateCommand": "cat /tmp/feature-status.txt && echo 'User setup complete'"
}
```

The feature command runs first, then the user command can read its output.

#### Feature Container Properties

Features can contribute container configuration properties that are merged into the final container:

- **Security Options:** `privileged`, `capAdd`, `securityOpt`
- **Environment Variables:** `containerEnv`
- **Mounts:** Feature-specific volume mounts
- **Init System:** `init` for proper signal handling

**Example:**
```json
// Feature metadata
{
  "id": "docker-feature",
  "privileged": true,
  "capAdd": ["NET_ADMIN"],
  "containerEnv": {
    "DOCKER_FEATURE_ENABLED": "true"
  }
}
```

These properties are automatically applied when the feature is used.

#### Complete Specification Support

packnplay supports 100% of the devcontainer features specification:
- **Feature options with validation**: Options defined in `devcontainer-feature.json` are automatically converted to environment variables with proper type validation (string, boolean, enum)
- **All lifecycle hooks**: Features can contribute `onCreateCommand`, `postCreateCommand`, and `postStartCommand` that execute before user commands
- **Container properties**: Features can configure security settings (`privileged`, `capAdd`, `securityOpt`), environment variables, mounts, and init systems
- **VS Code compatibility**: Full interoperability with VS Code devcontainers specification

**Example - Multi-Feature Configuration:**
```json
{
  "image": "mcr.microsoft.com/devcontainers/base:ubuntu",
  "features": {
    "ghcr.io/devcontainers/features/node:1": {
      "version": "18",
      "nodeGypDependencies": true
    },
    "ghcr.io/devcontainers/features/docker-in-docker:2": {
      "version": "latest",
      "moby": true
    },
    "ghcr.io/devcontainers/features/common-utils:2": {
      "installZsh": true,
      "installOhMyZsh": true,
      "upgradePackages": true
    }
  },
  "postCreateCommand": "npm install",
  "postStartCommand": "npm run dev"
}
```

This configuration:
1. Installs Node.js 18 with build dependencies
2. Enables Docker-in-Docker with Moby engine
3. Installs Zsh with Oh My Zsh
4. Runs all feature lifecycle hooks before user commands
5. Applies all feature container properties automatically

### Variable Substitution

Use variable substitution in `containerEnv`, `remoteEnv`, `mounts`, and `runArgs` values.

#### Supported Variables

**Local Environment:**
```json
{
  "containerEnv": {
    "HOME_DIR": "${localEnv:HOME}",
    "USER_NAME": "${localEnv:USER}",
    "API_KEY": "${localEnv:MY_API_KEY:default_key}"
  }
}
```

**Container Environment:**
```json
{
  "containerEnv": {
    "BASE_URL": "https://api.example.com"
  },
  "remoteEnv": {
    "API_ENDPOINT": "${containerEnv:BASE_URL}/v1"
  }
}
```

**Workspace Paths:**
```json
{
  "containerEnv": {
    "PROJECT_ROOT": "${localWorkspaceFolder}",
    "CONTAINER_ROOT": "${containerWorkspaceFolder}",
    "PROJECT_NAME": "${localWorkspaceFolderBasename}"
  }
}
```

**Container ID:**
```json
{
  "containerEnv": {
    "CONTAINER_ID": "${devcontainerId}"
  }
}
```

**Default Values:**
```json
{
  "containerEnv": {
    "NODE_ENV": "${localEnv:NODE_ENV:development}",
    "PORT": "${localEnv:PORT:3000}"
  }
}
```

**Variable Types:**
- `${localEnv:VAR}` or `${env:VAR}` - Host environment variable
- `${containerEnv:VAR}` - Container environment variable (from containerEnv)
- `${localWorkspaceFolder}` - Host project path
- `${containerWorkspaceFolder}` - Container workspace path
- `${localWorkspaceFolderBasename}` - Project directory name
- `${devcontainerId}` - Deterministic container identifier (SHA-256 of labels)

All variables support default values: `${localEnv:VAR:default}`

## Complete Example

```json
{
  "name": "My Node.js Dev Environment",
  "build": {
    "dockerfile": "Dockerfile.dev",
    "args": {
      "NODE_VERSION": "18"
    },
    "target": "development"
  },
  "remoteUser": "node",
  "containerEnv": {
    "NODE_ENV": "development",
    "PROJECT_ROOT": "${containerWorkspaceFolder}",
    "HOME_DIR": "${localEnv:HOME}",
    "DATABASE_URL": "postgres://localhost:5432/devdb"
  },
  "remoteEnv": {
    "PATH": "${containerEnv:PROJECT_ROOT}/node_modules/.bin:${containerEnv:PATH}"
  },
  "forwardPorts": [3000, 5432, 8080],
  "mounts": [
    "source=${localWorkspaceFolder}/config,target=/app/config,type=bind",
    "type=tmpfs,target=/tmp"
  ],
  "runArgs": ["--memory=2g", "--cpus=2"],
  "onCreateCommand": {
    "install": "npm ci",
    "prepare": "npm run prepare"
  },
  "postCreateCommand": "npm run build",
  "postStartCommand": {
    "server": "npm run dev",
    "types": "npm run types:watch"
  }
}
```

## Troubleshooting

### Environment Variables Not Set

**Problem:** Variables don't appear in container.

**Solutions:**
1. Check devcontainer.json syntax (valid JSON)
2. Verify variable names are correct
3. Use `docker exec <container> env` to inspect environment
4. Remember: CLI `--env` flags override devcontainer values

### Ports Not Forwarded

**Problem:** Cannot access forwarded ports.

**Solutions:**
1. Verify port format: integer or string
2. Check for port conflicts on host
3. Use `docker port <container>` to see mappings
4. CLI `-p` flags override devcontainer ports

### Lifecycle Commands Not Running

**Problem:** onCreate/postCreate/postStart don't execute.

**Solutions:**
1. Check command syntax (valid string, array, or object)
2. View container logs: `docker logs <container>`
3. Enable verbose mode: `packnplay run --verbose bash`
4. Lifecycle errors are warnings (don't fail startup)

### Commands Run Every Time

**Problem:** onCreate runs on every container start.

**Diagnosis:** Check metadata:
```bash
cat ~/.local/share/packnplay/metadata/<container-id>.json
```

**Solutions:**
1. Metadata tracks execution by command hash
2. Changing command content triggers re-run
3. Delete metadata file to force re-run: `rm ~/.local/share/packnplay/metadata/<container-id>.json`

### Variable Substitution Not Working

**Problem:** Variables show as `${...}` literally.

**Solutions:**
1. Variables only work in `containerEnv` and `remoteEnv` values
2. Check variable syntax: `${type:name:default}`
3. Verify variable exists (e.g., `echo $HOME` on host)
4. Unknown variable types are preserved (check spelling)

## CLI Override Behavior

CLI flags take precedence over devcontainer.json:

```bash
# Override environment variables
packnplay run --env NODE_ENV=production bash

# Override port forwarding
packnplay run -p 8080:80 bash

# Override image (not recommended)
# devcontainer.json image is ignored
```

**Best Practice:** Use devcontainer.json for team-shared config, CLI for personal overrides.

## Metadata Storage

Lifecycle execution state is stored in:
```
~/.local/share/packnplay/metadata/<container-id>.json
```

**Format:**
```json
{
  "containerId": "abc123...",
  "createdAt": "2024-01-15T10:30:00Z",
  "updatedAt": "2024-01-15T10:35:00Z",
  "lifecycleRan": {
    "onCreate": {
      "executed": true,
      "timestamp": "2024-01-15T10:30:00Z",
      "commandHash": "sha256:..."
    },
    "postCreate": {
      "executed": true,
      "timestamp": "2024-01-15T10:31:00Z",
      "commandHash": "sha256:..."
    }
  }
}
```

You can manually inspect or delete metadata files.

## Known Limitations

packnplay achieves **100% Microsoft devcontainer specification compliance** for core functionality. The only intentional exclusion:

### Out of Scope (VS Code-Specific)

1. **`customizations`**: Editor-specific extensions and settings
   - **Why**: Editor-agnostic CLI tool, not tied to VS Code
   - **Alternative**: Configure your editor separately

### Technical Notes

2. **Metadata per container**: Rebuilding image creates new container ID, new metadata
   - **Impact**: Lifecycle commands re-run after rebuild
   - **Workaround**: None needed (expected behavior)

3. **Lifecycle command timeouts**: Commands don't have built-in timeouts
   - **Alternative**: Use timeout command in shell: `timeout 60 npm install`

### Compatibility Notes

- devcontainer.json files are fully portable between packnplay and VS Code Remote Containers
- Unsupported fields (`customizations`) are silently ignored (no errors)
- All other specification properties are fully supported

## Examples

### Python Data Science

```json
{
  "image": "python:3.11-slim",
  "forwardPorts": [8888],
  "containerEnv": {
    "JUPYTER_ENABLE_LAB": "yes"
  },
  "onCreateCommand": "pip install -r requirements.txt",
  "postStartCommand": "jupyter lab --ip=0.0.0.0 --port=8888 --no-browser"
}
```

### Go Development

```json
{
  "build": {
    "dockerfile": "Dockerfile",
    "args": {
      "GO_VERSION": "1.21"
    }
  },
  "remoteUser": "vscode",
  "containerEnv": {
    "GOPATH": "/go",
    "CGO_ENABLED": "0"
  },
  "forwardPorts": [8080],
  "postCreateCommand": "go mod download",
  "postStartCommand": "go run main.go"
}
```

### Full-Stack Application

```json
{
  "build": {
    "dockerfile": "Dockerfile.dev",
    "context": ".."
  },
  "containerEnv": {
    "DATABASE_URL": "postgres://postgres:postgres@db:5432/appdb",
    "REDIS_URL": "redis://redis:6379",
    "API_PORT": "3000",
    "WEB_PORT": "8080"
  },
  "forwardPorts": [3000, 8080],
  "onCreateCommand": {
    "backend": "cd backend && npm ci",
    "frontend": "cd frontend && npm ci"
  },
  "postCreateCommand": {
    "db": "npm run db:migrate",
    "build": "npm run build"
  },
  "postStartCommand": {
    "api": "npm run api:dev",
    "web": "npm run web:dev"
  }
}
```

### Docker-in-Docker Development

```json
{
  "image": "docker:24-dind",
  "mounts": [
    "source=/var/run/docker.sock,target=/var/run/docker.sock,type=bind"
  ],
  "runArgs": ["--privileged"],
  "containerEnv": {
    "DOCKER_HOST": "unix:///var/run/docker.sock"
  },
  "postCreateCommand": "apk add --no-cache git bash",
  "postStartCommand": "docker info"
}
```

### Resource-Limited Testing Environment

```json
{
  "image": "node:18-alpine",
  "runArgs": [
    "--memory=512m",
    "--cpus=1",
    "--label", "env=${localEnv:ENV:development}"
  ],
  "mounts": [
    "type=tmpfs,target=/tmp,tmpfs-size=100000000"
  ],
  "containerEnv": {
    "NODE_OPTIONS": "--max-old-space-size=384"
  },
  "postCreateCommand": "npm ci --prefer-offline"
}
```

## Comparison with VS Code Remote Containers

Packnplay achieves **100% devcontainer specification compliance** (excluding VS Code-specific features):

| Feature | Packnplay | VS Code |
|---------|-----------|---------|
| `image` | ✅ | ✅ |
| `dockerfile` / `build` | ✅ | ✅ |
| `dockerComposeFile` / `service` | ✅ | ✅ |
| `remoteUser` / `containerUser` | ✅ | ✅ |
| `updateRemoteUserUID` | ✅ | ✅ |
| `containerEnv` / `remoteEnv` | ✅ | ✅ |
| `forwardPorts` | ✅ | ✅ |
| `portsAttributes` | ✅ | ✅ |
| `mounts` / `workspaceMount` | ✅ | ✅ |
| `runArgs` | ✅ | ✅ |
| All lifecycle commands | ✅ | ✅ |
| `waitFor` / `overrideCommand` | ✅ | ✅ |
| `shutdownAction` | ✅ | ✅ |
| `hostRequirements` | ✅ | ✅ |
| Variable substitution | ✅ | ✅ |
| `features` (OCI, local, HTTPS) | ✅ | ✅ |
| `customizations` | ❌ (out of scope) | ✅ |

Packnplay is a CLI tool focused on **AI coding agents**, so VS Code-specific `customizations` is intentionally excluded.

## See Also

- [DevContainer Specification](https://containers.dev/implementors/json_reference/)
- [Variable Substitution Reference](https://containers.dev/implementors/json_reference/#variables-in-devcontainerjson)
- [Packnplay README](../README.md)
