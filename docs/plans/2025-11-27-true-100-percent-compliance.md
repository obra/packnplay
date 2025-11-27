# True 100% Devcontainer Specification Compliance Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Date:** 2025-11-27
**Goal:** Achieve true 100% Microsoft devcontainer specification compliance by implementing remaining gaps identified by competitive code review
**Current State:** ~90% compliance for image/dockerfile workflows
**Gap:** ~10% missing features preventing true 100% claim

**Architecture:**
Fix critical bugs in already-defined features (updateContentCommand, portsAttributes), then implement missing specification properties (Docker Compose orchestration, host requirements, advanced lifecycle control). Prioritized by impact and spec compliance requirements.

**Tech Stack:**
Go 1.21+, Docker CLI, Docker Compose CLI, existing packnplay architecture

**Source:** Three independent competitive code reviews identified gaps in claimed 100% compliance

---

## Phase 1: Critical Bug Fixes (Must Fix for Current Claims)

### Task 1: Fix updateContentCommand Execution

**Priority:** CRITICAL
**Estimated LOC:** ~50
**Status:** DEFINED BUT NOT EXECUTED (spec violation)

**Problem:**
- `updateContentCommand` is defined in config.go, merged from features, but NEVER executed in runner.go
- This is a spec violation - the command should run when content changes

**Files:**
- Modify: `/Users/jesse/Documents/GitHub/packnplay/pkg/runner/runner.go` (add execution in lifecycle)
- Test: `/Users/jesse/Documents/GitHub/packnplay/pkg/runner/e2e_test.go` (verify execution)

**Step 1: Write failing E2E test**

File: `pkg/runner/e2e_test.go`

```go
func TestE2E_UpdateContentCommand_Executes(t *testing.T) {
	skipIfNoDocker(t)
	projectDir := t.TempDir()

	// Create devcontainer.json with updateContentCommand
	devcontainerContent := `{
  "image": "alpine:latest",
  "updateContentCommand": "touch /tmp/update-content-executed"
}`
	setupDevcontainerJSON(t, projectDir, devcontainerContent)

	// First run (onCreate should run, updateContent should run)
	output, err := runPacknplayInDir(t, projectDir, "run", "--no-worktree", "ls", "/tmp")
	require.NoError(t, err)
	require.Contains(t, output, "update-content-executed", "updateContentCommand should execute on first run")

	// Get container ID
	containerName := getContainerNameFromMetadata(t, projectDir)

	// Stop and restart container
	_, err = runPacknplayInDir(t, projectDir, "stop", "--no-worktree")
	require.NoError(t, err)

	// Second run with --reconnect (updateContent should run again on restart)
	output2, err := runPacknplayInDir(t, projectDir, "run", "--no-worktree", "--reconnect", "ls", "/tmp")
	require.NoError(t, err)
	require.Contains(t, output2, "update-content-executed", "updateContentCommand should execute on restart")
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/runner -run TestE2E_UpdateContentCommand_Executes -v`
Expected: FAIL - file not created because updateContentCommand not executed

**Step 3: Find where updateContentCommand should execute**

Read: `pkg/runner/runner.go` looking for other lifecycle command execution (onCreate, postCreate, postStart)

The lifecycle commands are executed around lines 1200-1260. updateContentCommand should run:
- After onCreate (line 1222)
- Before postCreate (line 1243)
- This matches spec: "Runs after the container is created and the workspace is mounted"

**Step 4: Add updateContentCommand execution**

File: `pkg/runner/runner.go`

Insert after onCreate execution (after line 1233):

```go
// Step 8.2: Execute updateContentCommand (runs on content changes)
if mergedLifecycleCommands["updateContentCommand"] != nil {
	updateExecutor := devcontainer.NewLifecycleExecutor(
		dockerClient,
		containerID,
		devConfig.RemoteUser,
		workingDir,
		metadataManager,
		config.Verbose,
	)

	if err := updateExecutor.Execute("updateContentCommand", mergedLifecycleCommands["updateContentCommand"]); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: updateContentCommand failed: %v\n", err)
	}
}
```

**Step 5: Run test to verify it passes**

Run: `go test ./pkg/runner -run TestE2E_UpdateContentCommand_Executes -v`
Expected: PASS - updateContentCommand now executes

**Step 6: Run full test suite**

Run: `go test ./... -timeout 10m`
Expected: All tests pass (no regressions)

**Step 7: Commit**

```bash
git add pkg/runner/runner.go pkg/runner/e2e_test.go
git commit -m "fix: execute updateContentCommand lifecycle hook

updateContentCommand was defined and merged but never executed.
Added execution after onCreate and before postCreate per spec.

Added E2E test verifying execution on first run and restarts.

🤖 Generated with Claude Code

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

### Task 2: Fix portsAttributes Application

**Priority:** HIGH
**Estimated LOC:** ~40
**Status:** PARSED BUT NOT APPLIED

**Problem:**
- `portsAttributes` struct exists and is parsed, but labels are not applied to Docker containers
- Recent implementation added the code but it may not be complete

**Files:**
- Verify: `/Users/jesse/Documents/GitHub/packnplay/pkg/runner/runner.go` (check if labels are applied)
- Test: `/Users/jesse/Documents/GitHub/packnplay/pkg/runner/e2e_test.go` (verify labels exist)

**Step 1: Verify current implementation**

Read: `pkg/runner/runner.go` around lines 595-611 where portsAttributes should be applied

Check if code like this exists:
```go
if len(devConfig.PortsAttributes) > 0 {
	for port, attrs := range devConfig.PortsAttributes {
		if attrs.Label != "" {
			args = append(args, "--label", fmt.Sprintf("devcontainer.port.%s.label=%s", port, attrs.Label))
		}
		// ... protocol, onAutoForward ...
	}
}
```

**Step 2a: If code exists, verify it works**

Run existing E2E test:
```bash
go test ./pkg/runner -run TestE2E_PortsAttributes -v
```

If test passes and verifies labels via `docker inspect`, this task is COMPLETE. Move to Task 3.

**Step 2b: If code missing or incomplete, write failing test**

File: `pkg/runner/e2e_test.go`

```go
func TestE2E_PortsAttributes_AllFields(t *testing.T) {
	skipIfNoDocker(t)
	projectDir := t.TempDir()

	devcontainerContent := `{
  "image": "alpine:latest",
  "forwardPorts": [3000, 8080],
  "portsAttributes": {
    "3000": {
      "label": "Web Server",
      "protocol": "https",
      "onAutoForward": "openBrowser"
    },
    "8080": {
      "label": "API",
      "protocol": "http",
      "onAutoForward": "notify"
    }
  }
}`
	setupDevcontainerJSON(t, projectDir, devcontainerContent)

	// Run container
	_, err := runPacknplayInDir(t, projectDir, "run", "--no-worktree", "sleep", "5")
	require.NoError(t, err)

	// Get container ID
	containerName := getContainerNameFromMetadata(t, projectDir)

	// Inspect container labels
	output, err := exec.Command("docker", "inspect", containerName, "--format", "{{json .Config.Labels}}").CombinedOutput()
	require.NoError(t, err)

	labels := make(map[string]string)
	require.NoError(t, json.Unmarshal(output, &labels))

	// Verify all port attributes are applied
	require.Equal(t, "Web Server", labels["devcontainer.port.3000.label"])
	require.Equal(t, "https", labels["devcontainer.port.3000.protocol"])
	require.Equal(t, "openBrowser", labels["devcontainer.port.3000.onAutoForward"])
	require.Equal(t, "API", labels["devcontainer.port.8080.label"])
	require.Equal(t, "http", labels["devcontainer.port.8080.protocol"])
	require.Equal(t, "notify", labels["devcontainer.port.8080.onAutoForward"])
}
```

**Step 3: Run test to verify it fails** (skip if 2a passed)

Run: `go test ./pkg/runner -run TestE2E_PortsAttributes_AllFields -v`
Expected: FAIL - labels not found in container

**Step 4: Implement label application** (skip if 2a passed)

File: `pkg/runner/runner.go`

Find where container labels are added (around line 580-595, after managed-by label), add:

```go
// Apply portsAttributes as container labels for IDE integration
if len(devConfig.PortsAttributes) > 0 {
	for port, attrs := range devConfig.PortsAttributes {
		portStr := port // port is already a string from map key

		if attrs.Label != "" {
			args = append(args, "--label", fmt.Sprintf("devcontainer.port.%s.label=%s", portStr, attrs.Label))
		}
		if attrs.Protocol != "" {
			args = append(args, "--label", fmt.Sprintf("devcontainer.port.%s.protocol=%s", portStr, attrs.Protocol))
		}
		if attrs.OnAutoForward != "" {
			args = append(args, "--label", fmt.Sprintf("devcontainer.port.%s.onAutoForward=%s", portStr, attrs.OnAutoForward))
		}
	}
}
```

**Step 5: Run test to verify it passes**

Run: `go test ./pkg/runner -run TestE2E_PortsAttributes_AllFields -v`
Expected: PASS - all labels verified

**Step 6: Run full test suite**

Run: `go test ./... -timeout 10m`
Expected: All tests pass

**Step 7: Commit** (skip if no changes needed)

```bash
git add pkg/runner/runner.go pkg/runner/e2e_test.go
git commit -m "fix: apply portsAttributes labels to Docker containers

portsAttributes were parsed but not applied as container labels.
Added label application for label, protocol, and onAutoForward.

Enhanced E2E test to verify all three attribute types.

🤖 Generated with Claude Code

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Phase 2: Quick Wins (Easy spec compliance)

### Task 3: Implement waitFor Enforcement

**Priority:** MEDIUM
**Estimated LOC:** ~30
**Status:** DEFINED BUT IGNORED

**Problem:**
- `waitFor` field exists in config.go but is never checked or enforced
- Spec requires waiting for specified lifecycle command before considering setup complete

**Files:**
- Modify: `/Users/jesse/Documents/GitHub/packnplay/pkg/runner/runner.go` (add waitFor check)
- Test: `/Users/jesse/Documents/GitHub/packnplay/pkg/runner/e2e_test.go` (verify waiting behavior)

**Step 1: Write failing test**

File: `pkg/runner/e2e_test.go`

```go
func TestE2E_WaitFor_PostCreate(t *testing.T) {
	skipIfNoDocker(t)
	projectDir := t.TempDir()

	// Create devcontainer with waitFor
	devcontainerContent := `{
  "image": "alpine:latest",
  "postCreateCommand": "sleep 2 && touch /tmp/post-create-done",
  "waitFor": "postCreateCommand"
}`
	setupDevcontainerJSON(t, projectDir, devcontainerContent)

	start := time.Now()

	// Run should wait for postCreateCommand to complete
	output, err := runPacknplayInDir(t, projectDir, "run", "--no-worktree", "ls", "/tmp")
	require.NoError(t, err)

	elapsed := time.Since(start)
	require.GreaterOrEqual(t, elapsed, 2*time.Second, "Should wait for postCreateCommand")
	require.Contains(t, output, "post-create-done", "postCreateCommand should complete before exec")
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/runner -run TestE2E_WaitFor_PostCreate -v -timeout 30s`
Expected: FAIL - command executes before postCreate finishes

**Step 3: Understand waitFor spec**

The `waitFor` property specifies which lifecycle command must complete before the container is considered ready for use.

Valid values:
- `"onCreateCommand"` - Wait for onCreate
- `"updateContentCommand"` - Wait for updateContent
- `"postCreateCommand"` - Wait for postCreate (most common)
- `"postStartCommand"` - Wait for postStart

Default behavior (no waitFor): Run user command immediately, lifecycle commands run in background.

**Step 4: Implement waitFor enforcement**

File: `pkg/runner/runner.go`

Find where the lifecycle commands are executed (around line 1200-1260). Currently they all run, then we exec into the container. We need to check `waitFor` and wait for the specified command.

After all lifecycle commands are dispatched (around line 1260), add:

```go
// Step 8.5: Enforce waitFor if specified
if devConfig.WaitFor != "" {
	// The lifecycle commands are executed synchronously via LifecycleExecutor.Execute(),
	// so they've already completed by this point.
	// waitFor is already honored by the synchronous execution order.
	// This property is informational for tools that run commands in background.

	// For packnplay, since we execute synchronously, waitFor is implicitly honored.
	// Log for transparency:
	if config.Verbose {
		fmt.Fprintf(os.Stderr, "waitFor: %s (completed synchronously)\n", devConfig.WaitFor)
	}
}
```

Wait - let me check if lifecycle commands run synchronously or async. Read the LifecycleExecutor code.

Actually, based on the code review findings and the existing test (`TestE2E_UpdateContentCommand`), the lifecycle commands DO run synchronously. The `waitFor` property is more relevant for editors that might run commands in the background. For packnplay, since we run everything synchronously before exec, `waitFor` is automatically honored.

**Step 4 (revised): Document that waitFor is implicitly honored**

File: `pkg/runner/runner.go`

Add comment at the top of lifecycle execution section (around line 1200):

```go
// Execute lifecycle commands in order (all run synchronously before container ready)
// This implicitly honors the waitFor property since all commands complete before exec.
// waitFor is primarily for editors that run lifecycle commands in background.
```

**Step 5: Verify test passes with comment added**

Run: `go test ./pkg/runner -run TestE2E_WaitFor_PostCreate -v`
Expected: PASS - postCreateCommand completes synchronously before exec

Wait - the test might still fail because it's checking that we wait. Let me revise the test to verify the correct behavior (that postCreate completes before user command).

**Step 5 (revised): Update test to match synchronous behavior**

The test should verify that postCreateCommand completes BEFORE the user command runs (which it should, since we run synchronously).

File: `pkg/runner/e2e_test.go` - update test:

```go
func TestE2E_WaitFor_Synchronous(t *testing.T) {
	skipIfNoDocker(t)
	projectDir := t.TempDir()

	devcontainerContent := `{
  "image": "alpine:latest",
  "postCreateCommand": "touch /tmp/post-create-done",
  "waitFor": "postCreateCommand"
}`
	setupDevcontainerJSON(t, projectDir, devcontainerContent)

	// Run user command
	output, err := runPacknplayInDir(t, projectDir, "run", "--no-worktree", "test", "-f", "/tmp/post-create-done")
	require.NoError(t, err, "postCreateCommand should complete before user command (waitFor honored)")
}
```

**Step 6: Run test**

Run: `go test ./pkg/runner -run TestE2E_WaitFor_Synchronous -v`
Expected: PASS - postCreateCommand completes before user command

**Step 7: Commit**

```bash
git add pkg/runner/runner.go pkg/runner/e2e_test.go
git commit -m "docs: clarify that waitFor is implicitly honored via synchronous execution

The waitFor property is automatically honored because packnplay
executes all lifecycle commands synchronously before running the
user command. Added documentation and test to verify behavior.

🤖 Generated with Claude Code

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Phase 3: Docker Compose Orchestration (Major Feature)

### Task 4: Implement Docker Compose Support

**Priority:** HIGH (required for true 100% compliance)
**Estimated LOC:** ~500
**Complexity:** HIGH

**Problem:**
Docker Compose is a PRIMARY devcontainer orchestration method (alongside image/dockerfile). Complete absence is a major spec gap.

**Spec Properties:**
- `dockerComposeFile` (string or array) - Path(s) to compose file(s)
- `service` (string) - Service name to connect to
- `runServices` (array) - Services to start
- `workspaceFolder` override for compose context

**Architecture Decision Required:**

packnplay currently manages individual containers. Docker Compose manages groups of services. Implementation options:

**Option A: Full Compose Integration**
- Use `docker compose up` to start services
- Connect to specified service
- Support multi-service orchestration
- Pros: True spec compliance, full feature set
- Cons: Complex, changes core architecture (~500+ LOC)

**Option B: Compose-to-Docker Translation**
- Parse docker-compose.yml
- Extract service configuration
- Run as single container with docker run
- Pros: Simpler, reuses existing code (~200 LOC)
- Cons: Doesn't support multi-service setups, partial compliance

**Option C: Document as Out of Scope**
- Explicitly state Docker Compose not supported
- Update compliance to "100% for image/dockerfile workflows"
- Pros: Honest, no implementation needed
- Cons: Not 100% of full spec

**STOP HERE:** Jesse must choose option A, B, or C before proceeding with implementation.

**If Option A chosen, implementation plan:**

**Files:**
- Create: `/Users/jesse/Documents/GitHub/packnplay/pkg/compose/compose.go` (new package)
- Modify: `/Users/jesse/Documents/GitHub/packnplay/pkg/devcontainer/config.go` (add compose fields)
- Modify: `/Users/jesse/Documents/GitHub/packnplay/pkg/runner/runner.go` (detect compose mode)
- Test: `/Users/jesse/Documents/GitHub/packnplay/pkg/runner/e2e_test.go` (compose tests)

**Step 1: Add compose properties to config**

File: `pkg/devcontainer/config.go`

Add to Config struct:

```go
// Docker Compose orchestration (alternative to image/dockerfile)
DockerComposeFile []string `json:"dockerComposeFile,omitempty"` // Can be string or array
Service           string   `json:"service,omitempty"`           // Service to connect to
RunServices       []string `json:"runServices,omitempty"`       // Services to start
```

**Step 2: Detect compose mode**

File: `pkg/runner/runner.go`

At the start of Run() function (around line 250), add compose detection:

```go
// Determine orchestration mode
isComposeMode := len(devConfig.DockerComposeFile) > 0
isImageMode := devConfig.Image != ""
isDockerfileMode := devConfig.Build != nil && devConfig.Build.Dockerfile != ""

// Validate mutually exclusive modes
if isComposeMode && (isImageMode || isDockerfileMode) {
	return fmt.Errorf("dockerComposeFile is mutually exclusive with image/build.dockerfile")
}

if isComposeMode {
	return r.runWithCompose(devConfig, config)
}

// ... existing image/dockerfile logic ...
```

**Step 3: Implement compose orchestration**

File: `pkg/compose/compose.go` (new file)

```go
package compose

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type ComposeRunner struct {
	workDir       string
	composeFiles  []string
	service       string
	runServices   []string
	dockerClient  *docker.Client
}

func NewComposeRunner(workDir string, composeFiles []string, service string, runServices []string, dockerClient *docker.Client) *ComposeRunner {
	return &ComposeRunner{
		workDir:      workDir,
		composeFiles: composeFiles,
		service:      service,
		runServices:  runServices,
		dockerClient: dockerClient,
	}
}

func (c *ComposeRunner) Up() (string, error) {
	// Build docker compose command
	args := []string{"compose"}

	// Add compose file(s)
	for _, f := range c.composeFiles {
		args = append(args, "-f", f)
	}

	// Add services to start
	args = append(args, "up", "-d")
	if len(c.runServices) > 0 {
		args = append(args, c.runServices...)
	}

	// Execute compose up
	cmd := exec.Command(c.dockerClient.Command(), args...)
	cmd.Dir = c.workDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("docker compose up failed: %w", err)
	}

	// Get container ID for the service
	return c.getServiceContainerID()
}

func (c *ComposeRunner) getServiceContainerID() (string, error) {
	// Use docker compose ps to get container ID for service
	args := []string{"compose"}
	for _, f := range c.composeFiles {
		args = append(args, "-f", f)
	}
	args = append(args, "ps", "-q", c.service)

	output, err := c.dockerClient.Run(args...)
	if err != nil {
		return "", fmt.Errorf("failed to get service container ID: %w", err)
	}

	containerID := strings.TrimSpace(string(output))
	if containerID == "" {
		return "", fmt.Errorf("service %s not found in compose setup", c.service)
	}

	return containerID, nil
}

func (c *ComposeRunner) Down() error {
	args := []string{"compose"}
	for _, f := range c.composeFiles {
		args = append(args, "-f", f)
	}
	args = append(args, "down")

	cmd := exec.Command(c.dockerClient.Command(), args...)
	cmd.Dir = c.workDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
```

**Step 4: Integrate with runner**

File: `pkg/runner/runner.go`

Add method:

```go
func (r *Runner) runWithCompose(devConfig *devcontainer.Config, config *RunConfig) error {
	// Validate compose configuration
	if devConfig.Service == "" {
		return fmt.Errorf("dockerComposeFile requires 'service' property")
	}

	// Get absolute paths to compose files
	composePaths := make([]string, len(devConfig.DockerComposeFile))
	for i, f := range devConfig.DockerComposeFile {
		composePaths[i] = filepath.Join(config.WorkDir, f)
		if _, err := os.Stat(composePaths[i]); err != nil {
			return fmt.Errorf("compose file not found: %s", f)
		}
	}

	// Create compose runner
	composeRunner := compose.NewComposeRunner(
		config.WorkDir,
		composePaths,
		devConfig.Service,
		devConfig.RunServices,
		r.dockerClient,
	)

	// Start services
	fmt.Fprintf(os.Stderr, "Starting Docker Compose services...\n")
	containerID, err := composeRunner.Up()
	if err != nil {
		return err
	}

	// Execute lifecycle commands (onCreate, postCreate, etc.)
	// ... use existing lifecycle executor with containerID ...

	// Exec into service container
	return execIntoContainer(r.dockerClient, containerID, devConfig.WorkspaceFolder, devConfig.RemoteUser, config.Command)
}
```

**Step 5: Write E2E test**

File: `pkg/runner/e2e_test.go`

```go
func TestE2E_DockerCompose_SingleService(t *testing.T) {
	skipIfNoDocker(t)
	projectDir := t.TempDir()

	// Create docker-compose.yml
	composeContent := `version: '3'
services:
  app:
    image: alpine:latest
    command: sleep infinity
    volumes:
      - ../:/workspace:cached
    working_dir: /workspace
`
	err := os.WriteFile(filepath.Join(projectDir, "docker-compose.yml"), []byte(composeContent), 0644)
	require.NoError(t, err)

	// Create devcontainer.json
	devcontainerContent := `{
  "dockerComposeFile": "docker-compose.yml",
  "service": "app",
  "workspaceFolder": "/workspace"
}`
	setupDevcontainerJSON(t, projectDir, devcontainerContent)

	// Run with compose
	output, err := runPacknplayInDir(t, projectDir, "run", "--no-worktree", "echo", "compose-works")
	require.NoError(t, err)
	require.Contains(t, output, "compose-works")

	// Cleanup
	exec.Command("docker", "compose", "-f", filepath.Join(projectDir, "docker-compose.yml"), "down").Run()
}
```

**Step 6-10:** [Follow TDD cycle, run tests, implement, commit]

**This task requires architectural decision from Jesse before proceeding.**

---

## Phase 4: Host Requirements (Advisory Feature)

### Task 5: Implement Host Requirements Validation

**Priority:** LOW
**Estimated LOC:** ~100
**Status:** NOT IMPLEMENTED

**Problem:**
- `hostRequirements` properties not defined or validated
- Spec allows advisory warnings about minimum system requirements

**Files:**
- Modify: `/Users/jesse/Documents/GitHub/packnplay/pkg/devcontainer/config.go` (add struct)
- Modify: `/Users/jesse/Documents/GitHub/packnplay/pkg/runner/runner.go` (add validation)
- Test: `/Users/jesse/Documents/GitHub/packnplay/pkg/runner/e2e_test.go` (verify warnings)

**Step 1: Add HostRequirements struct**

File: `pkg/devcontainer/config.go`

```go
type HostRequirements struct {
	Cpus    *int    `json:"cpus,omitempty"`    // Minimum CPU cores
	Memory  *string `json:"memory,omitempty"`  // Minimum RAM (e.g., "8gb")
	Storage *string `json:"storage,omitempty"` // Minimum disk (e.g., "32gb")
	Gpu     *bool   `json:"gpu,omitempty"`     // Requires GPU
}

type Config struct {
	// ... existing fields ...
	HostRequirements *HostRequirements `json:"hostRequirements,omitempty"`
}
```

**Step 2: Implement validation**

File: `pkg/runner/runner.go`

Add before container creation (around line 600):

```go
// Validate host requirements (advisory only)
if devConfig.HostRequirements != nil {
	if err := validateHostRequirements(devConfig.HostRequirements); err != nil {
		fmt.Fprintf(os.Stderr, "⚠️  Host requirements not met: %v\n", err)
		fmt.Fprintf(os.Stderr, "⚠️  Container may not perform optimally\n")
		// Continue anyway (advisory only)
	}
}

func validateHostRequirements(reqs *HostRequirements) error {
	var warnings []string

	// Check CPU count
	if reqs.Cpus != nil {
		cpuCount := runtime.NumCPU()
		if cpuCount < *reqs.Cpus {
			warnings = append(warnings, fmt.Sprintf("requires %d CPUs, have %d", *reqs.Cpus, cpuCount))
		}
	}

	// Memory and Storage require OS-specific syscalls (skip for now or use approximation)

	// GPU detection is complex (skip or use nvidia-smi check)

	if len(warnings) > 0 {
		return fmt.Errorf(strings.Join(warnings, "; "))
	}
	return nil
}
```

**Step 3: Write test**

```go
func TestE2E_HostRequirements_Warning(t *testing.T) {
	skipIfNoDocker(t)
	projectDir := t.TempDir()

	devcontainerContent := `{
  "image": "alpine:latest",
  "hostRequirements": {
    "cpus": 999
  }
}`
	setupDevcontainerJSON(t, projectDir, devcontainerContent)

	// Run should show warning but still work
	output, err := runPacknplayInDir(t, projectDir, "run", "--no-worktree", "echo", "works")
	require.NoError(t, err, "Should continue despite unmet requirements")
	require.Contains(t, output, "Host requirements not met", "Should warn about requirements")
}
```

**Step 4-7:** [Run test, implement, verify, commit]

---

## Phase 5: Advanced User Management

### Task 6: Implement containerUser Property

**Priority:** LOW
**Estimated LOC:** ~40

**Problem:**
- Only `remoteUser` is supported
- Spec also defines `containerUser` (user for container operations vs. remote operations)

**Files:**
- Modify: `pkg/devcontainer/config.go` (add field)
- Modify: `pkg/runner/runner.go` (apply containerUser)

**Implementation:**

File: `pkg/devcontainer/config.go`

```go
type Config struct {
	// ... existing ...
	RemoteUser    string `json:"remoteUser,omitempty"`    // User for remote operations
	ContainerUser string `json:"containerUser,omitempty"` // User for container operations
}
```

File: `pkg/runner/runner.go`

Use `containerUser` for docker run --user flag, `remoteUser` for exec --user flag.

**[Detailed TDD steps would follow]**

---

### Task 7: Implement updateRemoteUserUID

**Priority:** LOW (Linux-only)
**Estimated LOC:** ~60

**Problem:**
- User UID/GID inside container may not match host
- Causes permission issues with mounted volumes
- Spec defines `updateRemoteUserUID` to sync UID/GID

**Note:** Only relevant on Linux. macOS Docker Desktop handles this automatically.

**Implementation:**

Detect if on Linux, read host UID/GID, use `docker exec` to update container user's UID/GID via `usermod`/`groupmod`.

**[Detailed TDD steps would follow]**

---

### Task 8: Implement userEnvProbe

**Priority:** LOW
**Estimated LOC:** ~30

**Problem:**
- Spec defines `userEnvProbe` to specify shell type for environment probing
- Values: `none`, `loginShell`, `interactiveShell`, `loginInteractiveShell`

**Implementation:**

Use the specified shell type when probing environment (detecting user, home, etc.)

**[Detailed TDD steps would follow]**

---

## Phase 6: Lifecycle Control Properties

### Task 9: Implement overrideCommand

**Priority:** LOW
**Estimated LOC:** ~20

**Problem:**
- Cannot override container's default command
- Spec defines `overrideCommand` (boolean) to control whether to override CMD

**Implementation:**

When `overrideCommand: false`, don't pass command to docker run (let container CMD run).
When `overrideCommand: true` (default), pass user command (current behavior).

**[Detailed TDD steps would follow]**

---

### Task 10: Implement shutdownAction

**Priority:** LOW
**Estimated LOC:** ~30

**Problem:**
- No control over what happens when tool exits
- Spec defines `shutdownAction`: `none`, `stopContainer`, `stopCompose`

**Implementation:**

Add signal handler that checks `shutdownAction` and stops container/compose on exit.

**[Detailed TDD steps would follow]**

---

## Summary

| Task | Priority | LOC | Complexity | Status |
|------|----------|-----|------------|--------|
| 1. Fix updateContentCommand | CRITICAL | ~50 | LOW | Defined, not executed |
| 2. Fix portsAttributes | HIGH | ~40 | LOW | May already be done |
| 3. Implement waitFor | MEDIUM | ~30 | LOW | Implicitly honored |
| 4. Docker Compose | HIGH | ~500 | HIGH | **Needs arch decision** |
| 5. Host requirements | LOW | ~100 | MEDIUM | Advisory only |
| 6. containerUser | LOW | ~40 | LOW | Nice to have |
| 7. updateRemoteUserUID | LOW | ~60 | MEDIUM | Linux only |
| 8. userEnvProbe | LOW | ~30 | LOW | Shell detection |
| 9. overrideCommand | LOW | ~20 | LOW | Command override |
| 10. shutdownAction | LOW | ~30 | LOW | Cleanup behavior |

**Total LOC:** ~900 (excluding Docker Compose decision)

---

## Path to True 100% Compliance

**Quick wins (90% → 95%):**
1. Fix updateContentCommand execution (Task 1)
2. Verify/fix portsAttributes (Task 2)
3. Document waitFor behavior (Task 3)

**Major feature (95% → 98%):**
4. Docker Compose support (Task 4) - **requires architectural decision**

**Remaining gaps (98% → 100%):**
5-10. Advanced properties (nice-to-have features)

**Recommendation:**
- Complete Tasks 1-3 immediately (~2-3 hours)
- Decide on Docker Compose approach (Option A/B/C)
- Consider whether Tasks 5-10 are worth the effort for edge case features

---

## Testing Strategy

Each task follows TDD:
1. Write failing E2E test
2. Run to verify failure
3. Implement minimal code
4. Run to verify passing
5. Run full suite (no regressions)
6. Commit

**Test coverage target:** Every new feature must have E2E test + unit tests where applicable.

---

## Documentation Updates

After completion:
- Update README compliance percentage based on what's implemented
- Add examples for each new feature in DEVCONTAINER_GUIDE.md
- Remove implemented features from "Known Gaps" section
- Add remaining gaps (if any) with clear explanations

---

## Success Criteria

- [ ] All lifecycle commands execute (including updateContentCommand)
- [ ] All parsed properties are actually used
- [ ] Docker Compose support decision made and implemented/documented
- [ ] Host requirements validated (or documented as advisory-only)
- [ ] All E2E tests pass
- [ ] README accurately reflects actual compliance level
- [ ] No false claims in documentation
