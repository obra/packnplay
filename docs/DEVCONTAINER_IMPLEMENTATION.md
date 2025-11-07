# DevContainer Implementation Plan - Research-Driven TDD

**Based on:** Official Microsoft devcontainers/cli (MIT License)
**Research Date:** 2025-11-07
**Implementation Approach:** Rigorous TDD with test adaptation

## Attribution

Test cases and patterns adapted from:
- **Repository:** https://github.com/devcontainers/cli
- **License:** MIT License
- **Copyright:** (c) Microsoft Corporation. All rights reserved.
- **Test Source:** https://github.com/devcontainers/cli/blob/main/src/test/variableSubstitution.test.ts

## Library Structure Decision

**Chosen Approach:** Single package initially

```
pkg/devcontainer/
├── config.go              # Main configuration struct
├── config_test.go         # Configuration parsing tests
├── types.go               # Type definitions
├── variables.go           # ⭐ Variable substitution engine
├── variables_test.go      # ⭐ 13+ test cases (adapted from official)
├── lifecycle.go           # Command execution
├── lifecycle_test.go      # Lifecycle tests
└── testdata/
    ├── parallel-commands.json
    ├── env-substitution.json
    └── ...
```

**Rationale:**
- Start simple (YAGNI)
- Easier to navigate
- Can split later if needed

---

## Implementation Order

### Phase 1: Variable Substitution Engine (PRIORITY 1)

**Why First:** Most complex, used by all other features

#### Supported Patterns (from spec):

| Pattern | Description | Example |
|---------|-------------|---------|
| `${env:VAR}` | Local environment | `${env:HOME}` |
| `${localEnv:VAR}` | Explicit local env | `${localEnv:HOME}` |
| `${localEnv:VAR:default}` | With default | `${localEnv:MISSING:fallback}` |
| `${containerEnv:VAR}` | Container env | `${containerEnv:PATH}` |
| `${containerEnv:VAR:default}` | With default | `${containerEnv:NODE_ENV:development}` |
| `${localWorkspaceFolder}` | Local project path | `/Users/jesse/projects/myapp` |
| `${localWorkspaceFolderBasename}` | Folder name | `myapp` |
| `${containerWorkspaceFolder}` | Container path | `/workspace` |
| `${containerWorkspaceFolderBasename}` | Container folder name | `workspace` |
| `${devcontainerId}` | Container ID | SHA-256 based (52 chars) |

#### Test Cases to Adapt (from official impl):

```go
// Test cases adapted from devcontainers/cli (MIT License)
// Original: https://github.com/devcontainers/cli/blob/main/src/test/variableSubstitution.test.ts

func TestSubstituteEnvironmentVariable(t *testing.T)
func TestSubstituteWithDefault(t *testing.T)
func TestSubstituteMissingVariable(t *testing.T)
func TestSubstituteWorkspaceFolder(t *testing.T)
func TestSubstituteRecursive(t *testing.T)
func TestSubstituteDevContainerID(t *testing.T)
func TestSubstituteMultipleColonsInDefault(t *testing.T)
func TestSubstituteContainerEnv(t *testing.T)
func TestSubstitutePlatformSpecific(t *testing.T)
func TestSubstituteArray(t *testing.T)
func TestSubstituteObject(t *testing.T)
func TestSubstitutePreservesNonStrings(t *testing.T)
func TestDevContainerIDDeterministic(t *testing.T)
```

#### Implementation Strategy:

**Step 1: Types**
```go
// SubstituteContext holds all variables for substitution
type SubstituteContext struct {
    Platform                     string
    LocalWorkspaceFolder         string
    ContainerWorkspaceFolder     string
    LocalEnv                     map[string]string
    ContainerEnv                 map[string]string
    Labels                       map[string]string // For devcontainerId
}

// Substitute performs variable substitution on any JSON-compatible value
func Substitute(ctx *SubstituteContext, value interface{}) interface{}
```

**Step 2: Core Engine**
```go
func substituteString(ctx *SubstituteContext, s string) string {
    re := regexp.MustCompile(`\$\{(.*?)\}`)

    return re.ReplaceAllStringFunc(s, func(match string) string {
        // Remove ${ and }
        inner := match[2 : len(match)-1]

        // Split on first colon: varType:varName:default:...
        parts := strings.SplitN(inner, ":", 3)

        varType := parts[0]  // env, localEnv, containerEnv, etc.
        var varName, defaultVal string
        if len(parts) > 1 {
            varName = parts[1]
        }
        if len(parts) > 2 {
            defaultVal = parts[2]  // Can contain colons
        }

        // Lookup based on type
        switch varType {
        case "env", "localEnv":
            if val, ok := ctx.LocalEnv[varName]; ok {
                return val
            }
            return defaultVal

        case "containerEnv":
            if val, ok := ctx.ContainerEnv[varName]; ok {
                return val
            }
            return defaultVal

        case "localWorkspaceFolder":
            return ctx.LocalWorkspaceFolder

        case "containerWorkspaceFolder":
            // May contain ${localWorkspaceFolderBasename} - recurse
            return substituteString(ctx, ctx.ContainerWorkspaceFolder)

        case "devcontainerId":
            return generateDevContainerID(ctx.Labels)

        // ... other cases
        }

        return match // Preserve if unknown
    })
}

func generateDevContainerID(labels map[string]string) string {
    // Sort labels for determinism
    keys := make([]string, 0, len(labels))
    for k := range labels {
        keys = append(keys, k)
    }
    sort.Strings(keys)

    // Build sorted string
    var builder strings.Builder
    for _, k := range keys {
        builder.WriteString(k)
        builder.WriteString("=")
        builder.WriteString(labels[k])
        builder.WriteString("\n")
    }

    // SHA-256 hash
    hash := sha256.Sum256([]byte(builder.String()))

    // Base32 encode (52 chars)
    encoded := base32.StdEncoding.EncodeToString(hash[:])
    return strings.ToLower(encoded)[:52]
}
```

**Step 3: Recursive Traversal**
```go
func Substitute(ctx *SubstituteContext, value interface{}) interface{} {
    switch v := value.(type) {
    case string:
        return substituteString(ctx, v)

    case []interface{}:
        result := make([]interface{}, len(v))
        for i, item := range v {
            result[i] = Substitute(ctx, item)
        }
        return result

    case map[string]interface{}:
        result := make(map[string]interface{}, len(v))
        for k, val := range v {
            result[k] = Substitute(ctx, val)
        }
        return result

    default:
        // Preserve non-string types (numbers, booleans, null)
        return v
    }
}
```

---

### Phase 2: Environment Variables (containerEnv, remoteEnv)

#### Configuration Structure:

```go
type Config struct {
    Image        string            `json:"image,omitempty"`
    DockerFile   string            `json:"dockerFile,omitempty"`
    Build        *BuildConfig      `json:"build,omitempty"`
    RemoteUser   string            `json:"remoteUser,omitempty"`

    // NEW: Environment variables
    ContainerEnv map[string]string `json:"containerEnv,omitempty"`
    RemoteEnv    map[string]string `json:"remoteEnv,omitempty"`
}

// GetResolvedEnvironment returns environment variables after substitution
func (c *Config) GetResolvedEnvironment(ctx *SubstituteContext) map[string]string {
    result := make(map[string]string)

    // Apply substitution to containerEnv
    for k, v := range c.ContainerEnv {
        resolved := substituteString(ctx, v)
        result[k] = resolved
        // Add to context for future containerEnv: references
        ctx.ContainerEnv[k] = resolved
    }

    // Apply substitution to remoteEnv (can reference containerEnv)
    for k, v := range c.RemoteEnv {
        if v == "" {
            // null value removes variable
            delete(result, k)
        } else {
            result[k] = substituteString(ctx, v)
        }
    }

    return result
}
```

#### Test Cases:

```go
func TestContainerEnvBasic(t *testing.T) {
    // Basic environment variable
}

func TestContainerEnvWithSubstitution(t *testing.T) {
    // containerEnv: { "API_KEY": "${localEnv:API_KEY}" }
}

func TestRemoteEnvReferencesContainerEnv(t *testing.T) {
    // remoteEnv: { "PATH": "${containerEnv:PATH}:/custom" }
}

func TestRemoteEnvNull(t *testing.T) {
    // remoteEnv: { "REMOVED": null } removes variable
}
```

---

### Phase 3: Port Forwarding (forwardPorts)

#### Decision: Implement Despite CLI Limitation

Official CLI doesn't implement, but we can use Docker `-p` directly.

```go
type Config struct {
    // ...
    ForwardPorts []interface{} `json:"forwardPorts,omitempty"` // int | string
}

type PortSpec struct {
    Host      string // Default: "0.0.0.0"
    HostPort  int
    Container int
    Protocol  string // Default: "tcp"
}

func parseForwardPorts(ports []interface{}) ([]PortSpec, error) {
    var result []PortSpec

    for _, port := range ports {
        switch v := port.(type) {
        case float64:
            // JSON numbers are float64
            p := int(v)
            result = append(result, PortSpec{
                HostPort:  p,
                Container: p,
                Protocol:  "tcp",
            })

        case string:
            // Parse "3000" or "8080:80" or "127.0.0.1:8080:80/tcp"
            spec, err := parsePortString(v)
            if err != nil {
                return nil, err
            }
            result = append(result, spec)
        }
    }

    return result, nil
}

func parsePortString(s string) (PortSpec, error) {
    // Handle: "3000", "8080:80", "127.0.0.1:8080:80", "8080:80/tcp"
    // ...
}
```

---

### Phase 4: Build Configuration (build)

#### Configuration:

```go
type BuildConfig struct {
    Dockerfile string            `json:"dockerfile"`
    Context    string            `json:"context,omitempty"`
    Args       map[string]string `json:"args,omitempty"`
    Target     string            `json:"target,omitempty"`
    CacheFrom  []string          `json:"cacheFrom,omitempty"` // string | string[]
    Options    []string          `json:"options,omitempty"`
}

func (b *BuildConfig) ToDockerBuildArgs() []string {
    args := []string{"build"}

    // Context
    context := b.Context
    if context == "" {
        context = "."
    }

    // Dockerfile
    args = append(args, "-f", b.Dockerfile)

    // Build args
    for k, v := range b.Args {
        args = append(args, "--build-arg", fmt.Sprintf("%s=%s", k, v))
    }

    // Target
    if b.Target != "" {
        args = append(args, "--target", b.Target)
    }

    // Cache from
    for _, cache := range b.CacheFrom {
        args = append(args, "--cache-from", cache)
    }

    // Additional options
    args = append(args, b.Options...)

    // Context at end
    args = append(args, context)

    return args
}
```

---

### Phase 5: Lifecycle Scripts

#### Type Definition:

```go
type LifecycleCommand struct {
    raw interface{} // string | []string | map[string]interface{}
}

// UnmarshalJSON handles three formats
func (lc *LifecycleCommand) UnmarshalJSON(data []byte) error {
    // Try string
    var s string
    if err := json.Unmarshal(data, &s); err == nil {
        lc.raw = s
        return nil
    }

    // Try array
    var arr []string
    if err := json.Unmarshal(data, &arr); err == nil {
        lc.raw = arr
        return nil
    }

    // Try object
    var obj map[string]interface{}
    if err := json.Unmarshal(data, &obj); err == nil {
        lc.raw = obj
        return nil
    }

    return fmt.Errorf("invalid lifecycle command format")
}

type Config struct {
    // ...
    OnCreateCommand   *LifecycleCommand `json:"onCreateCommand,omitempty"`
    PostCreateCommand *LifecycleCommand `json:"postCreateCommand,omitempty"`
    PostStartCommand  *LifecycleCommand `json:"postStartCommand,omitempty"`
}
```

#### Execution:

```go
func (lc *LifecycleCommand) Execute(ctx context.Context, containerName, user string) error {
    if lc == nil {
        return nil
    }

    switch v := lc.raw.(type) {
    case string:
        // Run through shell
        return executeShell(ctx, containerName, user, v)

    case []string:
        // Direct exec
        return executeDirect(ctx, containerName, user, v)

    case map[string]interface{}:
        // Parallel execution
        return executeParallel(ctx, containerName, user, v)
    }

    return nil
}

func executeShell(ctx context.Context, containerName, user, command string) error {
    cmd := exec.CommandContext(ctx, "docker", "exec", "-u", user, containerName, "sh", "-c", command)
    output, err := cmd.CombinedOutput()
    if err != nil {
        return fmt.Errorf("command failed: %s: %w", string(output), err)
    }
    return nil
}

func executeParallel(ctx context.Context, containerName, user string, commands map[string]interface{}) error {
    var wg sync.WaitGroup
    errChan := make(chan error, len(commands))

    for name, cmd := range commands {
        wg.Add(1)
        go func(n string, c interface{}) {
            defer wg.Done()

            // Buffer output
            var buf bytes.Buffer

            // Execute command
            err := executeCommand(ctx, containerName, user, c, &buf)
            if err != nil {
                errChan <- fmt.Errorf("%s: %w", n, err)
            }

            // Print buffered output
            fmt.Print(buf.String())
        }(name, cmd)
    }

    wg.Wait()
    close(errChan)

    // Check for errors
    for err := range errChan {
        return err // Return first error
    }

    return nil
}
```

---

## Test Data (Adapted from Official Fixtures)

### testdata/parallel-commands.json

```json
{
  "build": {
    "dockerfile": "Dockerfile",
    "args": {
      "VARIANT": "16-bullseye"
    }
  },
  "postCreateCommand": {
    "post-create-1": "echo 'post create 1'",
    "post-create-2": "echo 'post create 2'"
  },
  "postStartCommand": {
    "post-start-1": "echo 'post start 1'",
    "post-start-2": "echo 'post start 2'"
  }
}
```

### testdata/env-substitution.json

```json
{
  "image": "ubuntu:22.04",
  "containerEnv": {
    "NODE_ENV": "development",
    "API_URL": "http://localhost:${localEnv:API_PORT:3000}"
  },
  "remoteEnv": {
    "PATH": "${containerEnv:PATH}:/custom/bin",
    "PROJECT": "${localWorkspaceFolderBasename}"
  }
}
```

---

## Implementation Timeline

### Week 1: Variable Substitution
- Day 1: Write 13 test cases (adapted from official)
- Day 2-3: Implement core substitution engine
- Day 4: Recursive resolution
- Day 5: DevContainer ID generation

### Week 2: Environment & Ports
- Day 1-2: containerEnv/remoteEnv parsing and substitution
- Day 3: Port forwarding parsing
- Day 4-5: Integration with runner

### Week 3: Build & Lifecycle
- Day 1-2: Build configuration parsing
- Day 3-4: Lifecycle command execution
- Day 5: Metadata tracking (onCreate runs once)

---

## Success Criteria

For each phase:
- [ ] All tests from official impl adapted with attribution
- [ ] Additional edge case tests written
- [ ] Test fixtures created
- [ ] Implementation passes all tests
- [ ] Integration with runner.Run() complete
- [ ] Documentation updated
- [ ] Examples added

---

## References

- Official Spec: https://containers.dev/implementors/spec/
- TypeScript Implementation: https://github.com/devcontainers/cli
- Go Implementation: https://github.com/kontainment/devcontainers-go
- Metadata Reference: https://containers.dev/implementors/json_reference/
