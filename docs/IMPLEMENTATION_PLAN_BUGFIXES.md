# CRITICAL BUG FIXES for Implementation Plan

**Date:** 2025-11-07
**Status:** MUST READ BEFORE IMPLEMENTING

This document contains critical corrections to `/home/user/packnplay/docs/IMPLEMENTATION_PLAN.md` based on architecture review. These bugs would prevent compilation.

---

## Bug Fix #1: RunWithProgress Signature (BLOCKER)

**Location:** Task 1.1.1, ImageManager implementation

**Problem in Plan:**
```go
type DockerClient interface {
    RunWithProgress(args ...string) error  // ❌ WRONG
}
```

**Actual Signature** (from docker/client.go:132):
```go
func (c *Client) RunWithProgress(imageName string, args ...string) error
```

**Corrected DockerClient Interface:**
```go
// DockerClient interface for testing
// Note: imageName parameter is for progress tracking
type DockerClient interface {
    RunWithProgress(imageName string, args ...string) error
    Run(args ...string) (string, error)
}
```

**Corrected ImageManager Implementation:**
```go
// pullImage pulls a container image
func (im *ImageManager) pullImage(image string) error {
    if im.verbose {
        fmt.Printf("Pulling image: %s\n", image)
    }

    // CORRECT: Pass imageName as first parameter
    return im.client.RunWithProgress(image, "pull", image)
}

// buildImage builds a container image from Dockerfile
func (im *ImageManager) buildImage(devConfig *devcontainer.Config, projectPath string) error {
    tag := fmt.Sprintf("packnplay-%s-devcontainer:latest",
        filepath.Base(projectPath))

    buildArgs := []string{
        "build",
        "-t", tag,
        "-f", filepath.Join(projectPath, ".devcontainer", devConfig.DockerFile),
        filepath.Join(projectPath, ".devcontainer"),
    }

    if im.verbose {
        fmt.Printf("Building image: %s\n", tag)
    }

    // CORRECT: Pass tag as imageName for progress tracking
    return im.client.RunWithProgress(tag, buildArgs...)
}
```

---

## Bug Fix #2: Agent Mount Conversion (MAJOR)

**Location:** Task 1.1.2, MountBuilder.buildAgentMounts()

**Problem in Plan:**
```go
for _, mount := range mounts {
    args = append(args, "-v", mount.String())  // ❌ Mount has no String() method!
}
```

**Actual Mount struct** (from agent.go:17-21):
```go
type Mount struct {
    HostPath      string
    ContainerPath string
    ReadOnly      bool
}
```

**Corrected Implementation:**
```go
// buildAgentMounts constructs agent config directory mounts
// Use the Agent abstraction (fixes hardcoded issue)
func (mb *MountBuilder) buildAgentMounts() []string {
    var args []string

    for _, agent := range agents.GetSupportedAgents() {
        // Check if agent config exists on host
        agentPath := filepath.Join(mb.hostHomeDir, agent.ConfigDir())
        if !fileExists(agentPath) {
            continue
        }

        // Get mounts from agent
        mounts := agent.GetMounts(mb.hostHomeDir, mb.containerUser)
        for _, mount := range mounts {
            // Convert Mount struct to Docker -v format
            mountStr := fmt.Sprintf("%s:%s", mount.HostPath, mount.ContainerPath)
            if mount.ReadOnly {
                mountStr += ":ro"
            }
            args = append(args, "-v", mountStr)
        }
    }

    return args
}
```

---

## Bug Fix #3: Missing Imports (BLOCKER)

**Location:** Multiple files

**Corrected image_manager.go imports:**
```go
package runner

import (
    "fmt"
    "os"  // For fmt.Printf output
    "path/filepath"  // For filepath.Join, filepath.Base

    "github.com/obra/packnplay/pkg/devcontainer"
)
```

**Corrected mount_builder.go imports:**
```go
package runner

import (
    "fmt"  // For fmt.Sprintf
    "os"  // For os.Stat, fileExists
    "path/filepath"  // For filepath.Join

    "github.com/obra/packnplay/pkg/agents"
    "github.com/obra/packnplay/pkg/config"  // For config.Credentials
)
```

**Corrected lifecycle.go imports:**
```go
package runner

import (
    "fmt"
    "os"
    "time"  // For time.Now(), time.Time

    "github.com/obra/packnplay/pkg/devcontainer"
)
```

**Corrected variables.go imports:**
```go
package devcontainer

import (
    "os"  // For os.Getenv
    "regexp"  // For variable substitution
)
```

**Corrected ports.go imports:**
```go
package devcontainer

import (
    "fmt"  // For fmt.Sprintf, fmt.Errorf
)
```

---

## Bug Fix #4: Config.GetDockerfile() Inconsistency

**Location:** Task 2.3, ImageManager must use GetDockerfile()

**Problem in Plan:** Creates helper but doesn't always use it

**Corrected ImageManager.buildImage():**
```go
func (im *ImageManager) buildImage(devConfig *devcontainer.Config, projectPath string) error {
    // ALWAYS use GetDockerfile() helper
    dockerfile := devConfig.GetDockerfile()
    if dockerfile == "" {
        return fmt.Errorf("no dockerfile specified")
    }

    tag := fmt.Sprintf("packnplay-%s-devcontainer:latest",
        filepath.Base(projectPath))

    buildArgs := []string{
        "build",
        "-t", tag,
        "-f", filepath.Join(projectPath, ".devcontainer", dockerfile),  // Use helper result
        filepath.Join(projectPath, ".devcontainer"),
    }

    if im.verbose {
        fmt.Printf("Building image: %s\n", tag)
    }

    return im.client.RunWithProgress(tag, buildArgs...)
}
```

---

## Bug Fix #5: Missing DockerClient Interface Extraction

**Location:** Task 1.1.1 - CRITICAL STEP MISSING

**Add to docker/client.go:**
```go
package docker

// DockerClient interface for testing and abstraction
type DockerClient interface {
    RunWithProgress(imageName string, args ...string) error
    Run(args ...string) (string, error)
    Command() string
}

// Compile-time check that *Client implements DockerClient
var _ DockerClient = (*Client)(nil)

// Command returns the docker command being used
func (c *Client) Command() string {
    return c.cmd
}
```

**Update runner imports:**
```go
import (
    "github.com/obra/packnplay/pkg/docker"
)

// Use docker.DockerClient everywhere instead of creating new interface:
func NewImageManager(client docker.DockerClient, verbose bool) *ImageManager {
    return &ImageManager{
        client:  client,
        verbose: verbose,
    }
}
```

---

## Bug Fix #6: MockDockerClient Improvements

**Location:** All test files

**Corrected Mock Implementation:**
```go
// mockDockerClient for testing with error injection and call tracking
type mockDockerClient struct {
    pullCalled  bool
    buildCalled bool
    execCalled  bool
    pullError   error  // Inject errors
    buildError  error
    execError   error
    calls       []string  // Track call order
    output      string    // Return value for Run()
}

func (m *mockDockerClient) RunWithProgress(imageName string, args ...string) error {
    m.calls = append(m.calls, args[0])

    switch args[0] {
    case "pull":
        m.pullCalled = true
        return m.pullError
    case "build":
        m.buildCalled = true
        return m.buildError
    default:
        return fmt.Errorf("unexpected command: %s", args[0])
    }
}

func (m *mockDockerClient) Run(args ...string) (string, error) {
    m.calls = append(m.calls, args[0])

    switch args[0] {
    case "exec":
        m.execCalled = true
        return m.output, m.execError
    default:
        return "", fmt.Errorf("unexpected command: %s", args[0])
    }
}

func (m *mockDockerClient) Command() string {
    return "docker"
}
```

---

## Bug Fix #7: Task Order - Merge Task 1.2 into 1.1.2

**Problem:** Task 1.2 "Use Agent Abstraction" is already done in Task 1.1.2

**Corrected Task List:**

### Phase 1 Tasks (UPDATED):
1. Task 1.1.1: Create ImageManager
2. Task 1.1.2: Create MountBuilder (includes fixing hardcoded agents)
3. Task 1.1.3: Integrate services into runner.Run()
4. ~~Task 1.2~~: **REMOVED** (already done in 1.1.2)
5. Task 1.3: Consolidate duplicate label parsing

---

## Bug Fix #8: Lifecycle Metadata Migration

**Location:** Task 3.1, LifecycleExecutor.ShouldRunOnCreate()

**Problem:** Existing containers have no metadata, will run onCreate every time

**Corrected Implementation:**
```go
// ShouldRunOnCreate determines if onCreate should run
func (le *LifecycleExecutor) ShouldRunOnCreate(devConfig *devcontainer.Config) bool {
    if devConfig.OnCreateCommand == nil {
        return false
    }

    cmdHash := HashCommand(devConfig.OnCreateCommand)

    // Check if we have any metadata at all (existing container migration)
    if le.metadata.CreatedAt.IsZero() {
        // No metadata exists - this is an existing container
        // Initialize metadata and mark onCreate as already run
        le.metadata.CreatedAt = time.Now()
        le.metadata.LifecycleRan = map[string]LifecycleRun{
            "onCreate": {
                Executed:    true,
                Timestamp:   time.Now(),
                ExitCode:    0,
                CommandHash: cmdHash,
            },
        }
        // Don't run onCreate for existing containers
        return false
    }

    // Check if onCreate has run before with this command hash
    if run, exists := le.metadata.LifecycleRan["onCreate"]; exists {
        // Command has run before - only re-run if command changed
        return run.CommandHash != cmdHash
    }

    // Never run before - should run
    return true
}
```

---

## Bug Fix #9: Lifecycle Timeout Handling

**Location:** Task 3.1, LifecycleExecutor.ExecuteLifecycle()

**Problem:** Long-running scripts block indefinitely

**Add Timeout Support:**
```go
import (
    "context"
    "time"
)

// ExecuteLifecycle runs all appropriate lifecycle scripts with timeout
func (le *LifecycleExecutor) ExecuteLifecycle(devConfig *devcontainer.Config, verbose bool) error {
    // Create timeout context (5 minutes default)
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
    defer cancel()

    // 1. Execute onCreateCommand (only if not run before)
    if devConfig.OnCreateCommand != nil {
        if le.ShouldRunOnCreate(devConfig) {
            if verbose {
                fmt.Println("Running onCreateCommand...")
            }
            if err := le.ExecuteCommandWithContext(ctx, devConfig.OnCreateCommand, verbose); err != nil {
                return fmt.Errorf("onCreateCommand failed: %w", err)
            }
            le.MarkCommandRun("onCreate", devConfig.OnCreateCommand)
        }
    }

    // ... rest of lifecycle commands with ctx

    // Save metadata
    return SaveMetadata(le.containerName, le.metadata)
}

// ExecuteCommandWithContext executes with timeout
func (le *LifecycleExecutor) ExecuteCommandWithContext(ctx context.Context, cmd interface{}, verbose bool) error {
    // Create channel for result
    errChan := make(chan error, 1)

    go func() {
        errChan <- le.ExecuteCommand(cmd, verbose)
    }()

    select {
    case err := <-errChan:
        return err
    case <-ctx.Done():
        return fmt.Errorf("lifecycle command timed out after 5 minutes")
    }
}
```

---

## Bug Fix #10: Security Comment for Lifecycle Commands

**Location:** Task 3.1, LifecycleExecutor.executeShellCommand()

**Add Security Documentation:**
```go
// executeShellCommand executes a single shell command in the container
//
// SECURITY NOTE: Command comes from devcontainer.json (user's own config file).
// This is executed in the user's own container with their own credentials.
// No privilege escalation occurs. The user is running their own commands
// in their own environment, so command injection is not a concern here.
func (le *LifecycleExecutor) executeShellCommand(cmd string, verbose bool) error {
    // Use docker exec to run command in container
    args := []string{
        "exec",
        "-u", le.containerUser,
        le.containerName,
        "sh", "-c", cmd,  // Safe - user's own command in their own container
    }

    output, err := le.client.Run(args...)
    if verbose || err != nil {
        fmt.Println(output)
    }

    return err
}
```

---

## Bug Fix #11: Build Args Security Warning

**Location:** Task 2.3, ImageManager.buildImage()

**Add Warning Comment:**
```go
// buildImage builds a container image from Dockerfile with build config
//
// SECURITY WARNING: Build args are persisted in image metadata and can be
// inspected with `docker history`. Users should not put secrets in build args.
// For secrets, use containerEnv with ${localEnv:SECRET} variable substitution
// which injects secrets at runtime without persisting them in the image.
func (im *ImageManager) buildImage(devConfig *devcontainer.Config, projectPath string) error {
    // ... implementation
}
```

---

## Additional Corrections

### Task 1.1.1: Complete Test File

**Corrected image_manager_test.go with all imports:**
```go
package runner

import (
    "fmt"
    "testing"

    "github.com/obra/packnplay/pkg/devcontainer"
)

func TestImageManager_EnsureAvailable_WithImage(t *testing.T) {
    // Test: When devcontainer specifies an image, pull it
    mockClient := &mockDockerClient{
        pullCalled: false,
    }

    im := NewImageManager(mockClient, false)

    devConfig := &devcontainer.Config{
        Image: "ubuntu:22.04",
    }

    err := im.EnsureAvailable(devConfig, "/test/project")
    if err != nil {
        t.Fatalf("Expected no error, got: %v", err)
    }

    if !mockClient.pullCalled {
        t.Error("Expected image pull to be called")
    }
}

func TestImageManager_EnsureAvailable_WithDockerfile(t *testing.T) {
    // Test: When devcontainer specifies dockerfile, build it
    mockClient := &mockDockerClient{
        buildCalled: false,
    }

    im := NewImageManager(mockClient, false)

    devConfig := &devcontainer.Config{
        DockerFile: "Dockerfile",
    }

    err := im.EnsureAvailable(devConfig, "/test/project")
    if err != nil {
        t.Fatalf("Expected no error, got: %v", err)
    }

    if !mockClient.buildCalled {
        t.Error("Expected image build to be called")
    }
}

func TestImageManager_EnsureAvailable_NeitherImageNorDockerfile(t *testing.T) {
    // Test: Error when neither image nor dockerfile specified
    mockClient := &mockDockerClient{}
    im := NewImageManager(mockClient, false)

    devConfig := &devcontainer.Config{
        // Neither Image nor DockerFile set
    }

    err := im.EnsureAvailable(devConfig, "/test/project")
    if err == nil {
        t.Error("Expected error when no image or dockerfile specified")
    }
}

// mockDockerClient for testing (see Bug Fix #6 for full implementation)
type mockDockerClient struct {
    pullCalled  bool
    buildCalled bool
    pullError   error
    buildError  error
    calls       []string
}

func (m *mockDockerClient) RunWithProgress(imageName string, args ...string) error {
    m.calls = append(m.calls, args[0])

    if args[0] == "pull" {
        m.pullCalled = true
        return m.pullError
    } else if args[0] == "build" {
        m.buildCalled = true
        return m.buildError
    }

    return nil
}

func (m *mockDockerClient) Run(args ...string) (string, error) {
    return "", nil
}

func (m *mockDockerClient) Command() string {
    return "docker"
}
```

---

## Adjusted Timeline

**Phase 1:** 6-7 hours (was 4-5)
- +1 hour for interface extraction
- +1 hour for bug fixes and mock improvements

**Phase 2:** 2 hours (unchanged)

**Phase 3:** 3-4 hours (was 2)
- +1 hour for metadata migration
- +30min for timeout handling

**Total:** 12-15 hours (was 8-9 hours)

---

## Implementation Checklist

Before starting each task, verify:
- [ ] All imports are correct
- [ ] Interface signatures match actual code
- [ ] Mock implementations support error injection
- [ ] Tests include negative cases
- [ ] Security implications documented
- [ ] Run `go test ./...` before committing

---

## Order of Operations (Corrected)

1. Read this bugfixes document completely
2. Start with Phase 1, Task 1.1.1
3. Apply all bug fixes from this document
4. Write tests first (RED)
5. Implement with bugfixes (GREEN)
6. Refactor (REFACTOR)
7. Run all tests
8. Commit with clear message
9. Move to next task

**DO NOT** start implementation without reading all bugfixes first!
