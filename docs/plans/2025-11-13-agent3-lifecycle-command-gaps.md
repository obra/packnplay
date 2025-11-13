# Agent 3: Lifecycle Command and Execution Feature Gaps

**Analysis Date**: 2025-11-13
**Task**: Compare packnplay's lifecycle command and execution support with Microsoft's devcontainer-cli specification

## Executive Summary

Packnplay has implemented the **core lifecycle command execution patterns correctly**, but is missing several important features from the Microsoft specification:

1. **Missing `initializeCommand` lifecycle hook**
2. **Missing `waitFor` configuration support**
3. **Missing secrets integration for lifecycle commands**
4. **Missing output buffering for parallel commands**
5. **Incomplete container restart behavior**
6. **Missing postAttach lifecycle hook**

## Detailed Analysis

### 1. Lifecycle Hook Coverage

#### ✅ Implemented in packnplay
- `onCreateCommand` - Run once when container is created
- `updateContentCommand` - Run on create and optionally on rebuild
- `postCreateCommand` - Run after container creation completes
- `postStartCommand` - Run every time container starts

#### ❌ Missing from packnplay
- `initializeCommand` - Run **before** container is built (on host, not in container)
- `postAttachCommand` - Run when attaching to an existing container
- `waitFor` - Control which lifecycle hook to wait for before considering setup complete

**Microsoft Implementation**:
```typescript
// src/spec-common/injectHeadless.ts:128
export type DevContainerLifecycleHook =
  'initializeCommand' | 'onCreateCommand' | 'updateContentCommand' |
  'postCreateCommand' | 'postStartCommand' | 'postAttachCommand';

// src/spec-common/injectHeadless.ts:130
const defaultWaitFor: DevContainerLifecycleHook = 'updateContentCommand';

// src/spec-common/injectHeadless.ts:146
waitFor?: DevContainerLifecycleHook;
```

**Impact**:
- Cannot run pre-build commands (e.g., downloading dependencies before image build)
- Cannot run attach-specific commands (e.g., starting a file watcher when IDE connects)
- Cannot configure when container is "ready" for use

---

### 2. Execution Order and Sequencing

#### ✅ Correct in packnplay
- Feature commands execute **before** user commands (per spec)
- Commands within same hook execute **sequentially** by default
- Parallel execution uses goroutines correctly

**packnplay Implementation**:
```go
// pkg/devcontainer/lifecycle_merger.go:24-48
// First, add feature commands in installation order
for _, feature := range features {
    if featureCommand != nil {
        commands := featureCommand.ToStringSlice()
        mergedCommands = append(mergedCommands, commands...)
    }
}

// Then, add user commands
if userCommand, exists := userCommands[hookType]; exists && userCommand != nil {
    commands := userCommand.ToStringSlice()
    mergedCommands = append(mergedCommands, commands...)
}
```

**Microsoft Implementation**:
```typescript
// src/spec-common/injectHeadless.ts:453-462
async function runLifecycleCommands(...) {
    const commandsForHook = lifecycleCommandOriginMap[lifecycleHookName];
    for (const { command, origin } of commandsForHook) {
        const displayOrigin = origin ? (origin === 'devcontainer.json' ? origin : `Feature '${origin}'`) : '???';
        await runLifecycleCommand(...);
    }
}
```

#### ⚠️ Potential Issue: Merged Commands Execution
packnplay executes merged commands sequentially with early exit on error:
```go
// pkg/runner/lifecycle_executor.go:102-108
func (le *LifecycleExecutor) executeMergedCommands(commands []string) error {
    for _, cmd := range commands {
        if err := le.executeShellCommand(cmd); err != nil {
            return err  // ← Stops on first error
        }
    }
    return nil
}
```

This matches Microsoft's behavior (sequential with early exit).

---

### 3. Parallel Command Execution

#### ✅ Implemented correctly
packnplay uses object syntax for parallel execution:
```go
// pkg/runner/lifecycle_executor.go:134-194
func (le *LifecycleExecutor) executeParallelCommands(commands map[string]interface{}) error {
    var wg sync.WaitGroup
    errChan := make(chan error, len(commands))

    for name, cmd := range commands {
        wg.Add(1)
        go func(taskName string, taskCmd interface{}) {
            defer wg.Done()
            // Execute command...
        }(name, cmd)
    }

    wg.Wait()
    // Collect all errors...
}
```

#### ❌ Missing: Output Buffering
Microsoft buffers parallel command output to prevent interleaving:
```typescript
// src/spec-common/injectHeadless.ts:510-521
async function runSingleCommand(postCommand: string | string[], name?: string) {
    // If we have a command name then the command is running in parallel and
    // we need to hold output until the command is done so that the output
    // doesn't get interleaved with the output of other commands.
    const printMode = name ? 'off' : 'continuous';  // ← Key difference
    const env = { ...(await remoteEnv), ...(await secrets) };
    try {
        const { cmdOutput } = await runRemoteCommand(..., { print: printMode });

        // 'name' is set when parallel execution syntax is used.
        if (name) {
            infoOutput.raw(`\x1b[1mRunning ${name} of ${lifecycleHookName}...\x1b[0m\r\n${cmdOutput}\r\n`);
        }
    }
}
```

**Impact**: When running parallel commands in packnplay, output from different tasks can interleave, making logs hard to read.

**packnplay's current behavior**:
- Parallel tasks print directly to stdout as they run
- No buffering or synchronization of output streams

---

### 4. Secrets Support

#### ❌ Completely Missing
Microsoft passes secrets as environment variables to lifecycle commands:

```typescript
// src/spec-common/injectHeadless.ts:514
const env = { ...(await remoteEnv), ...(await secrets) };

// src/spec-common/injectHeadless.ts:364
export async function runLifecycleHooks(
    params: ResolverParameters,
    lifecycleHooksInstallMap: LifecycleHooksInstallMap,
    containerProperties: ContainerProperties,
    config: CommonMergedDevContainerConfig,
    remoteEnv: Promise<Record<string, string>>,
    secrets: Promise<Record<string, string>>,  // ← Secrets parameter
    stopForPersonalization: boolean
)
```

**Test Evidence** (from lifecycleHooks.test.ts:108-194):
- Secrets are loaded from `--secrets-file` JSON file
- Available to all lifecycle hooks during `up` and `run-user-commands`
- Secret values are masked in logs (e.g., "cycle" → "******")

**packnplay Implementation**:
- No secrets parameter in `LifecycleExecutor`
- No environment variable injection from secrets
- No secrets file support

**Impact**:
- Cannot securely pass credentials to lifecycle commands
- Cannot use CI/CD secrets in container setup
- Users must hardcode sensitive values or use workarounds

---

### 5. Container Restart and Resume Behavior

#### ✅ Correct: Metadata Tracking
packnplay correctly tracks lifecycle execution state:
```go
// pkg/runner/metadata.go:151-187
func (m *ContainerMetadata) ShouldRun(commandType string, cmd *devcontainer.LifecycleCommand) bool {
    if cmd == nil {
        return false
    }

    // postStart always runs (no tracking)
    if commandType == "postStart" {
        return true
    }

    // Check if command has been executed before
    state, exists := m.LifecycleRan[commandType]
    if !exists {
        return true  // First time
    }

    // Command has been executed before - check if it changed
    currentHash := HashCommand(cmd)
    if currentHash != state.CommandHash {
        return true  // Command changed
    }

    return false  // Already executed
}
```

#### ❌ Missing: postAttachCommand
Microsoft runs `postAttachCommand` on **every attach**, not tracked by markers:

```typescript
// src/spec-common/injectHeadless.ts:448-450
async function runPostAttachCommand(...) {
    await runLifecycleCommands(..., 'postAttachCommand', ..., true);  // ← Always runs (doRun=true)
}
```

**Microsoft's lifecycle on container resume**:
1. Container stopped
2. `devcontainer up` on existing container
3. Runs `postStartCommand` (based on marker file)
4. Runs `postAttachCommand` (always)

**packnplay's current behavior**:
- Only implements `postStartCommand`
- No `postAttachCommand` hook
- Missing the "attach" lifecycle event entirely

**Test Evidence** (lifecycleHooks.test.ts:41-93):
```typescript
// After stopping and restarting container:
assert.match(outputOfExecCommand, /15.panda.postStartCommand.testMarker/);
assert.match(outputOfExecCommand, /18.panda.postAttachCommand.testMarker/);
```

---

### 6. Marker File Strategy

#### ✅ Similar Approach
Both implementations use marker files to track execution.

**Microsoft** (in-container marker files):
```typescript
// src/spec-common/injectHeadless.ts:427-431
async function runPostCreateCommand(...) {
    const markerFile = path.posix.join(containerProperties.userDataFolder, `.${postCommandName}Marker`);
    const doRun = !!containerProperties.createdAt &&
                  await updateMarkerFile(containerProperties.shellServer, markerFile, containerProperties.createdAt) ||
                  rerun;
    await runLifecycleCommands(...);
}

// src/spec-common/injectHeadless.ts:439-446
async function updateMarkerFile(shellServer: ShellServer, location: string, content: string) {
    try {
        await shellServer.exec(`mkdir -p '${path.posix.dirname(location)}' && CONTENT="$(cat '${location}' 2>/dev/null || echo ENOENT)" && [ "\${CONTENT:-${content}}" != '${content}' ] && echo '${content}' > '${location}'`);
        return true;
    } catch (err) {
        return false;
    }
}
```

Marker files are stored **inside the container** at `${userDataFolder}/.{hookName}Marker`

**packnplay** (host-side metadata):
```go
// pkg/runner/metadata.go:31-53
// GetMetadataPath returns the path where metadata for a container should be stored.
// Location: ${XDG_DATA_HOME}/packnplay/metadata/{container-id}.json
// or ~/.local/share/packnplay/metadata/{container-id}.json
func GetMetadataPath(containerID string) (string, error) {
    dataHome := os.Getenv("XDG_DATA_HOME")
    if dataHome == "" {
        homeDir, err := os.UserHomeDir()
        dataHome = filepath.Join(homeDir, ".local", "share")
    }

    metadataDir := filepath.Join(dataHome, "packnplay", "metadata")
    os.MkdirAll(metadataDir, 0755)

    return filepath.Join(metadataDir, containerID+".json"), nil
}
```

Metadata is stored **on the host** in `~/.local/share/packnplay/metadata/{container-id}.json`

#### ⚠️ Architectural Difference
- **Microsoft**: Marker files travel with the container (if volumes persist)
- **packnplay**: Metadata tied to host machine (doesn't follow container across hosts)

**Implications**:
- ✅ packnplay approach is simpler and more reliable (no container file system dependencies)
- ❌ packnplay metadata doesn't follow containers if they're exported/imported
- ✅ packnplay can track state even if container volumes are destroyed

---

### 7. Command Type Support

#### ✅ All Three Types Supported
Both implementations support:

1. **String** - Shell command
   ```json
   "postCreateCommand": "npm install"
   ```

2. **Array** - Direct execution (no shell)
   ```json
   "postCreateCommand": ["npm", "install"]
   ```

3. **Object** - Parallel tasks
   ```json
   "postCreateCommand": {
       "server": "npm start",
       "watch": "npm run watch"
   }
   ```

**packnplay Implementation**:
```go
// pkg/devcontainer/lifecycle.go:47-100
func (lc *LifecycleCommand) AsString() (string, bool)
func (lc *LifecycleCommand) AsArray() ([]string, bool)
func (lc *LifecycleCommand) AsObject() (map[string]interface{}, bool)
```

**Microsoft Implementation**:
```typescript
// src/spec-common/injectHeadless.ts:132
export type LifecycleCommand = string | string[] | { [key: string]: string | string[] };
```

---

### 8. Error Handling

#### ✅ Correct Behavior
Both implementations:
- Stop on first error in sequential execution
- Collect all errors in parallel execution
- Return descriptive error messages

**packnplay**:
```go
// pkg/runner/lifecycle_executor.go:174-193
// Collect all errors
var errors []error
for err := range errChan {
    errors = append(errors, err)
}

if len(errors) == 1 {
    return errors[0]
}

// Multiple errors - combine them
errMsg := "multiple tasks failed:"
for _, err := range errors {
    errMsg += fmt.Sprintf("\n  - %s", err.Error())
}
return fmt.Errorf("%s", errMsg)
```

**Microsoft**:
```typescript
// src/spec-common/injectHeadless.ts:553-557
const results = await Promise.allSettled(commands);
const rejection = results.find(p => p.status === 'rejected');
if (rejection) {
    throw (rejection as PromiseRejectedResult).reason;
}
```

---

### 9. Environment Setup

#### ❌ Missing: remoteEnv and Secrets Injection
Microsoft injects environment variables into every lifecycle command:

```typescript
// src/spec-common/injectHeadless.ts:514
const env = { ...(await remoteEnv), ...(await secrets) };

// src/spec-common/injectHeadless.ts:516
const { cmdOutput } = await runRemoteCommand(
    { ...lifecycleHook, output: infoOutput },
    containerProperties,
    typeof postCommand === 'string' ? ['/bin/sh', '-c', postCommand] : postCommand,
    remoteCwd,
    { remoteEnv: env, pty: true, print: printMode }
);
```

**packnplay**:
- Uses container's default environment
- No additional environment variable injection
- No support for secrets or remoteEnv configuration

---

### 10. Command Context and Working Directory

#### ✅ Correct: Uses Container Exec
Both implementations execute commands in the running container:

**packnplay**:
```go
// pkg/runner/lifecycle_executor.go:84-97
func (le *LifecycleExecutor) executeShellCommand(cmd string) error {
    args := []string{
        "exec",
        "-u", le.containerUser,
        le.containerName,
        "sh", "-c", cmd,
    }

    output, err := le.client.Run(args...)
    return err
}
```

**Microsoft**:
```typescript
// Uses runRemoteCommand with containerProperties which includes:
// - remoteWorkspaceFolder (working directory)
// - shellServer (command execution interface)
// - containerUser (execution user)
```

#### ❌ Missing: Working Directory Configuration
Microsoft executes commands in the workspace folder:
```typescript
// src/spec-common/injectHeadless.ts:501
const remoteCwd = containerProperties.remoteWorkspaceFolder || containerProperties.homeFolder;
```

packnplay doesn't explicitly set working directory (uses container default).

---

## Feature Gap Summary

### Critical Missing Features

| Feature | Priority | Complexity | Spec Compliance |
|---------|----------|------------|-----------------|
| `initializeCommand` | HIGH | Medium | Required for pre-build commands |
| `postAttachCommand` | HIGH | Low | Required for IDE attach scenarios |
| `waitFor` configuration | MEDIUM | Low | Controls when setup is "done" |
| Secrets support | HIGH | Medium | Required for secure credentials |
| Output buffering (parallel) | LOW | Low | Quality of life improvement |
| Working directory control | MEDIUM | Low | May affect command behavior |

### Behavioral Differences

| Behavior | packnplay | Microsoft | Impact |
|----------|-----------|-----------|--------|
| Metadata storage | Host filesystem | Container filesystem | Different portability characteristics |
| Parallel output | Interleaved | Buffered | Log readability |
| Environment injection | None | remoteEnv + secrets | Feature parity gap |

---

## Test Coverage Comparison

### Microsoft's Test Suite (lifecycleHooks.test.ts)

1. **lifecycle-hooks-inline-commands**
   - Tests all 5 lifecycle hooks execute in order
   - Validates feature commands run before user commands
   - Confirms postStart/postAttach run on container resume

2. **lifecycle-hooks-inline-commands with secrets**
   - Secrets available during `up` command
   - Secrets available during `run-user-commands`
   - Secret masking in log output

3. **lifecycle-hooks-alternative-order**
   - Tests different `installsAfter` orderings
   - Validates stable execution order

4. **lifecycle-hooks-resume-existing-container**
   - Container stop → restart behavior
   - postStart/postAttach re-execution

5. **lifecycle-hooks-advanced**
   - Scripts bundled with features
   - Parallel postCreateCommand execution
   - Commands installed by feature install.sh

### packnplay's Test Suite

1. **lifecycle_executor_test.go**
   - String command execution ✅
   - Array command execution ✅
   - Object (parallel) command execution ✅
   - Error handling ✅
   - Multiple parallel errors ✅

2. **lifecycle_merger_test.go** (assumed to exist)
   - Feature + user command merging
   - Order preservation

**Gap**: No end-to-end tests for:
- Container restart behavior
- Secrets integration
- initializeCommand
- postAttachCommand
- waitFor configuration

---

## Recommendations

### Phase 1: Critical Gaps (Required for Spec Compliance)

1. **Add `postAttachCommand` support**
   - Extend lifecycle_merger.go to handle postAttachCommand
   - Add to LifecycleExecutor (always runs, no tracking)
   - Add to Config struct in devcontainer package

2. **Add `initializeCommand` support**
   - Runs on **host** before container build
   - Not in LifecycleExecutor (different execution context)
   - Needs integration in container build flow

3. **Add secrets support**
   - Add `--secrets-file` flag to CLI
   - Pass secrets to LifecycleExecutor
   - Inject as environment variables in executeShellCommand

4. **Add `waitFor` configuration**
   - Add to Config struct
   - Implement in container orchestration logic
   - Document default behavior (updateContentCommand)

### Phase 2: Quality Improvements

5. **Buffer parallel command output**
   - Modify executeParallelCommands to capture output
   - Print buffered output after completion
   - Improve log readability

6. **Add working directory control**
   - Pass workspace folder to LifecycleExecutor
   - Add `-w` flag to docker exec commands
   - Match Microsoft's remoteCwd behavior

### Phase 3: Testing

7. **Add E2E tests**
   - Container restart scenarios
   - Secrets integration tests
   - Feature + user command ordering
   - Parallel execution with output validation

---

## Code References

### packnplay Files
- `/home/jesse/git/packnplay/pkg/devcontainer/lifecycle.go` - LifecycleCommand type
- `/home/jesse/git/packnplay/pkg/devcontainer/lifecycle_merger.go` - Command merging
- `/home/jesse/git/packnplay/pkg/runner/lifecycle_executor.go` - Command execution
- `/home/jesse/git/packnplay/pkg/runner/metadata.go` - Execution state tracking

### Microsoft devcontainer-cli Files
- `vendor/devcontainer-cli/src/spec-common/injectHeadless.ts:364-559` - Lifecycle execution
- `vendor/devcontainer-cli/src/spec-node/imageMetadata.ts:123-143` - Command merging
- `vendor/devcontainer-cli/src/test/container-features/lifecycleHooks.test.ts` - Test suite

---

## Conclusion

packnplay has implemented the **core lifecycle command execution pattern correctly**:
- ✅ Sequential execution with early exit on error
- ✅ Parallel execution with error collection
- ✅ Feature commands before user commands
- ✅ Three command types (string, array, object)
- ✅ Metadata tracking for one-time commands

**However**, it is missing several features required for full Microsoft specification compliance:
- ❌ `initializeCommand` (pre-build hook)
- ❌ `postAttachCommand` (attach hook)
- ❌ `waitFor` configuration
- ❌ Secrets integration
- ❌ Output buffering for parallel commands
- ❌ Working directory control

The most critical gaps for users are:
1. **Secrets support** - Security concern, needed for real-world use
2. **initializeCommand** - Common pattern in Microsoft examples
3. **postAttachCommand** - Important for IDE integration

These gaps should be prioritized in the implementation roadmap.
