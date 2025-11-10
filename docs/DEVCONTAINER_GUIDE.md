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
User to run commands as in the container.

```json
{
  "remoteUser": "node"
}
```

If not specified, packnplay auto-detects the appropriate user.

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

### Lifecycle Commands

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

packnplay focuses on **core devcontainer functionality for AI coding agents** while maintaining compatibility with the devcontainer specification. The following features are intentionally not supported:

### Out of Scope (VS Code-Specific)

1. **`features`**: Devcontainer features are VS Code-specific and require complex installation system
   - **Why**: Large implementation surface area, tightly coupled to VS Code
   - **Alternative**: Use custom Dockerfile with pre-installed tools

2. **`customizations`**: Editor-specific extensions and settings
   - **Why**: Editor-agnostic tool, not tied to VS Code
   - **Alternative**: Configure your editor separately

### Out of Scope (Security/Complexity)

3. **`initializeCommand`**: Runs on host before container starts
   - **Why**: Security concern (arbitrary host code execution)
   - **Alternative**: Use shell scripts or Makefile on host

4. **`updateContentCommand`**: Runs when container content changes
   - **Why**: Requires content change detection system
   - **Alternative**: Use `postCreateCommand` for one-time setup

5. **`postAttachCommand`**: Runs after attaching to container
   - **Why**: Requires attach detection infrastructure
   - **Alternative**: Run commands manually after attach

### Future Enhancements

6. **Lifecycle command timeouts**: Commands don't timeout
   - **Status**: Future enhancement planned
   - **Alternative**: Use timeout command in shell: `timeout 60 npm install`

### Technical Limitations

7. **Metadata per container**: Rebuilding image creates new container ID, new metadata
   - **Impact**: Lifecycle commands re-run after rebuild
   - **Workaround**: None needed (expected behavior)

### Compatibility Notes

- packnplay implements a **useful subset** of the devcontainer specification
- devcontainer.json files are portable between packnplay and VS Code Remote Containers
- Unsupported fields are silently ignored (no errors)
- See comparison table in "Comparison with VS Code Remote Containers" section above

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

Packnplay implements a **subset** of the devcontainer specification:

| Feature | Packnplay | VS Code |
|---------|-----------|---------|
| `image` | ✅ | ✅ |
| `dockerfile` / `build` | ✅ | ✅ |
| `remoteUser` | ✅ | ✅ |
| `containerEnv` / `remoteEnv` | ✅ | ✅ |
| `forwardPorts` | ✅ | ✅ |
| `mounts` | ✅ | ✅ |
| `runArgs` | ✅ | ✅ |
| `onCreateCommand` | ✅ | ✅ |
| `postCreateCommand` | ✅ | ✅ |
| `postStartCommand` | ✅ | ✅ |
| Variable substitution | ✅ (subset) | ✅ (full) |
| `features` | ❌ | ✅ |
| `customizations` | ❌ | ✅ |

Packnplay focuses on **core devcontainer functionality** for AI coding agents while maintaining compatibility with the specification.

## See Also

- [DevContainer Specification](https://containers.dev/implementors/json_reference/)
- [Variable Substitution Reference](https://containers.dev/implementors/json_reference/#variables-in-devcontainerjson)
- [Packnplay README](../README.md)
