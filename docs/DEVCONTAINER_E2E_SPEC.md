# DevContainer E2E Testing & Feature Completion Specification

## Executive Summary

This specification defines:
1. **End-to-End Test Coverage**: Comprehensive real Docker integration tests for all devcontainer features
2. **Missing Features**: Remaining devcontainer spec features to implement
3. **Implementation Plan**: Phased approach using TDD and subagent pattern

**Current Status**: 86+ unit tests (all mocked), 0 real Docker tests

**Goal**: Prove all devcontainer functionality works with real Docker daemon

## Phase 1: E2E Test Infrastructure (Priority 1)

### 1.1 Test Framework Setup

**File**: `pkg/runner/e2e_test.go`

**Requirements**:
- Skip tests if Docker daemon unavailable (`testing.Short()`)
- Clean up all containers/images after tests
- Isolated test environments (unique container names)
- Test timeout enforcement (prevent hanging tests)
- Helper functions for common operations

**Test Helpers Needed**:
```go
// Test helper functions
func skipIfNoDocker(t *testing.T)
func createTestProject(t *testing.T, files map[string]string) string
func cleanupContainer(t *testing.T, containerName string)
func waitForContainer(t *testing.T, containerName string, timeout time.Duration) error
func execInContainer(t *testing.T, containerName string, cmd []string) (string, error)
func inspectContainer(t *testing.T, containerName string) (map[string]interface{}, error)
```

**Test Lifecycle Pattern**:
```go
func TestE2E_FeatureName(t *testing.T) {
    skipIfNoDocker(t)

    projectDir := createTestProject(t, map[string]string{
        ".devcontainer/devcontainer.json": `{...}`,
        "test.txt": "content",
    })
    defer os.RemoveAll(projectDir)

    containerName := fmt.Sprintf("packnplay-e2e-%d", time.Now().UnixNano())
    defer cleanupContainer(t, containerName)

    // Run packnplay
    // Verify behavior
    // Assert results
}
```

### 1.2 Docker Daemon Detection

**Requirements**:
- Detect if Docker daemon is available
- Provide clear skip messages for developers without Docker
- Support both `docker` and `podman` (future)

**Implementation**:
```go
func isDockerAvailable() bool {
    cmd := exec.Command("docker", "info")
    return cmd.Run() == nil
}
```

### 1.3 Test Cleanup Strategy

**Requirements**:
- Always cleanup containers (even on test failure)
- Use `defer` for cleanup
- Label test containers for identification
- Provide manual cleanup command

**Cleanup Function**:
```go
func cleanupTestContainers() {
    // Remove all containers with label managed-by=packnplay-e2e
    exec.Command("docker", "ps", "-aq", "--filter", "label=managed-by=packnplay-e2e").Run()
}
```

## Phase 2: E2E Tests for Current Features (Priority 1)

Test all currently implemented features with real Docker.

### 2.1 Image Pull Tests

**Test**: `TestE2E_ImagePull`
- Create devcontainer.json with `"image": "alpine:latest"`
- Run packnplay
- Verify container created from alpine image
- Verify container runs specified command

**Assertions**:
```go
// Verify image was pulled
output := exec.Command("docker", "images", "alpine:latest", "-q").Output()
assert.NotEmpty(t, output)

// Verify container created from image
inspect := inspectContainer(t, containerName)
assert.Equal(t, "alpine:latest", inspect["Config"].(map[string]interface{})["Image"])
```

### 2.2 Dockerfile Build Tests

**Test**: `TestE2E_DockerfileBuild`
- Create test project with Dockerfile
- Create devcontainer.json with `"dockerfile": "Dockerfile"`
- Run packnplay
- Verify custom image built
- Verify container uses custom image

**Test Files**:
```
.devcontainer/Dockerfile:
  FROM alpine:latest
  RUN echo "custom-marker" > /custom-marker.txt

.devcontainer/devcontainer.json:
  {"dockerfile": "Dockerfile"}
```

**Assertions**:
```go
// Verify custom file exists in container
output := execInContainer(t, containerName, []string{"cat", "/custom-marker.txt"})
assert.Equal(t, "custom-marker\n", output)
```

### 2.3 Build Config Tests

**Test**: `TestE2E_BuildWithArgs`
- Dockerfile with ARG
- Build config with args
- Verify build arg applied

**Test**: `TestE2E_BuildWithTarget`
- Multi-stage Dockerfile
- Build config with target
- Verify correct stage built

**Test**: `TestE2E_BuildWithContext`
- Files in parent directory
- Build config with `"context": ".."`
- Verify files accessible during build

**Test Files**:
```
.devcontainer/Dockerfile:
  ARG VARIANT=3.11
  FROM python:${VARIANT}-slim
  RUN python --version > /python-version.txt

.devcontainer/devcontainer.json:
  {
    "build": {
      "dockerfile": "Dockerfile",
      "args": {"VARIANT": "3.12"}
    }
  }
```

**Assertions**:
```go
output := execInContainer(t, containerName, []string{"cat", "/python-version.txt"})
assert.Contains(t, output, "3.12")
```

### 2.4 Environment Variable Tests

**Test**: `TestE2E_ContainerEnv`
- Create devcontainer.json with containerEnv
- Run packnplay
- Verify env vars set in container

**Test**: `TestE2E_RemoteEnv`
- Create devcontainer.json with containerEnv + remoteEnv
- Verify remoteEnv can reference containerEnv
- Verify substitution works correctly

**Test**: `TestE2E_EnvPriority`
- Set devcontainer env vars
- Override with CLI `--env` flag
- Verify CLI takes precedence

**Test Files**:
```json
{
  "image": "alpine:latest",
  "containerEnv": {
    "BASE_URL": "https://api.example.com"
  },
  "remoteEnv": {
    "API_ENDPOINT": "${containerEnv:BASE_URL}/v1"
  }
}
```

**Assertions**:
```go
output := execInContainer(t, containerName, []string{"sh", "-c", "echo $API_ENDPOINT"})
assert.Equal(t, "https://api.example.com/v1\n", output)
```

### 2.5 Variable Substitution Tests

**Test**: `TestE2E_LocalEnvSubstitution`
- Set local env var
- Reference in devcontainer.json
- Verify substituted in container

**Test**: `TestE2E_WorkspaceVariables`
- Use ${localWorkspaceFolder}, ${containerWorkspaceFolder}
- Verify correct paths

**Test**: `TestE2E_DefaultValues`
- Use ${localEnv:VAR:default}
- Verify default used when VAR not set

**Environment Setup**:
```go
os.Setenv("TEST_API_KEY", "secret123")
defer os.Unsetenv("TEST_API_KEY")
```

**Assertions**:
```go
output := execInContainer(t, containerName, []string{"sh", "-c", "echo $API_KEY"})
assert.Equal(t, "secret123\n", output)
```

### 2.6 Port Forwarding Tests

**Test**: `TestE2E_PortForwarding`
- Create devcontainer.json with forwardPorts
- Run packnplay
- Verify ports mapped on host

**Test**: `TestE2E_PortFormats`
- Integer format: `3000`
- String format: `"8080:80"`
- IP binding: `"127.0.0.1:9000:9000"`
- Verify all formats work

**Assertions**:
```go
output := exec.Command("docker", "port", containerName, "3000").Output()
assert.Contains(t, string(output), "0.0.0.0:3000")
```

### 2.7 Lifecycle Command Tests

**Test**: `TestE2E_OnCreateCommand_RunsOnce`
- Create devcontainer with onCreate that creates file
- Run packnplay twice
- Verify command only ran first time

**Test**: `TestE2E_PostCreateCommand_RunsOnce`
- Similar to onCreate test
- Verify runs after onCreate

**Test**: `TestE2E_PostStartCommand_RunsEveryTime`
- Create devcontainer with postStart
- Run packnplay multiple times
- Verify runs every time

**Test**: `TestE2E_CommandFormatString`
- String command with shell features (pipes, &&)
- Verify shell command executes

**Test**: `TestE2E_CommandFormatArray`
- Array command format
- Verify direct execution (no shell)

**Test**: `TestE2E_CommandFormatObject`
- Object format with parallel tasks
- Verify all tasks execute

**Test**: `TestE2E_CommandChangeDetection`
- Run with onCreate command
- Change command content
- Run again
- Verify re-executes

**Test Files**:
```json
{
  "image": "alpine:latest",
  "onCreateCommand": "touch /tmp/created-once.txt",
  "postStartCommand": "date > /tmp/started-at.txt"
}
```

**Assertions**:
```go
// First run
runPacknplay(t, projectDir)
output1 := execInContainer(t, containerName, []string{"cat", "/tmp/created-once.txt"})
assert.Empty(t, output1) // file exists but empty

// Second run (attach to same container)
runPacknplay(t, projectDir)
// Verify onCreate didn't run again (file still exists, not recreated)
```

### 2.8 User Detection Tests

**Test**: `TestE2E_RemoteUser`
- Create devcontainer with `"remoteUser": "nobody"`
- Run packnplay
- Verify commands run as nobody

**Test**: `TestE2E_UserAutoDetection`
- No remoteUser specified
- Verify auto-detection works
- Check user in container

**Assertions**:
```go
output := execInContainer(t, containerName, []string{"whoami"})
assert.Equal(t, "nobody\n", output)
```

### 2.9 Integration Tests

**Test**: `TestE2E_FullStack`
- Complete devcontainer.json with all features
- Verify everything works together

**Test**: `TestE2E_NodeJSProject`
- Real-world Node.js example
- Dockerfile + build config
- npm install in onCreate
- npm run dev in postStart
- Port forwarding for app

**Test**: `TestE2E_PythonProject`
- Real-world Python example
- Requirements.txt install
- Virtual environment setup

## Phase 3: Missing Devcontainer Features (Priority 2)

Features from the spec not yet implemented.

### 3.1 Additional Lifecycle Commands

**Missing Commands**:
- `initializeCommand` - Runs on host before container starts
- `updateContentCommand` - Runs when container content updated
- `postAttachCommand` - Runs after attaching to container

**Implementation Effort**: Medium

**Why Not Implemented Yet**:
- initializeCommand requires host execution (security concern)
- updateContentCommand requires content change detection
- postAttachCommand requires attach detection

**Decision**: Skip for now (YAGNI principle)
- Core onCreate/postCreate/postStart covers 90% of use cases
- Can add later if users request

### 3.2 Mounts

**Missing Feature**: `mounts` field for additional volume mounts

**Example**:
```json
{
  "mounts": [
    "source=/var/run/docker.sock,target=/var/run/docker.sock,type=bind"
  ]
}
```

**Implementation Effort**: Low

**Value**: Medium (useful for Docker-in-Docker scenarios)

**Decision**: Implement if time allows, otherwise defer

### 3.3 Features

**Missing Feature**: Devcontainer features (pre-packaged tools/configuration)

**Example**:
```json
{
  "features": {
    "ghcr.io/devcontainers/features/node:1": {
      "version": "18"
    }
  }
}
```

**Implementation Effort**: Very High
- Requires feature download/installation system
- Complex specification
- Large surface area

**Decision**: Explicitly out of scope
- Features are VS Code-specific
- packnplay focuses on core container functionality
- Document as known limitation

### 3.4 Customizations

**Missing Feature**: Editor-specific customizations (VS Code extensions, settings)

**Example**:
```json
{
  "customizations": {
    "vscode": {
      "extensions": ["dbaeumer.vscode-eslint"]
    }
  }
}
```

**Implementation Effort**: N/A (editor-specific)

**Decision**: Explicitly out of scope
- packnplay is editor-agnostic
- Document as known limitation

### 3.5 Additional Variable Types

**Currently Supported**:
- `${localEnv:VAR}`
- `${env:VAR}`
- `${containerEnv:VAR}`
- `${localWorkspaceFolder}`
- `${containerWorkspaceFolder}`
- `${localWorkspaceFolderBasename}`
- `${devcontainerId}`

**Missing from Spec**:
- `${userHome}` - User's home directory
- `${localEnv:HOME}` - Already works via localEnv
- Additional platform-specific variables

**Decision**: Current coverage sufficient
- Core variables implemented
- Can add more if users request

## Phase 4: Implementation Plan

### Task Breakdown

**Phase 1: E2E Infrastructure (4 hours)**
- [ ] Task 1.1: Create e2e_test.go with framework setup
- [ ] Task 1.2: Implement test helper functions
- [ ] Task 1.3: Implement Docker detection
- [ ] Task 1.4: Implement cleanup strategy
- [ ] Task 1.5: Write example e2e test as proof of concept

**Phase 2: E2E Tests (8 hours)**
- [ ] Task 2.1: Image pull tests (1 hour)
- [ ] Task 2.2: Dockerfile build tests (1 hour)
- [ ] Task 2.3: Build config tests (1 hour)
- [ ] Task 2.4: Environment variable tests (1 hour)
- [ ] Task 2.5: Variable substitution tests (1 hour)
- [ ] Task 2.6: Port forwarding tests (1 hour)
- [ ] Task 2.7: Lifecycle command tests (1.5 hours)
- [ ] Task 2.8: User detection tests (0.5 hours)
- [ ] Task 2.9: Integration tests (1 hour)

**Phase 3: Optional Features (4 hours)**
- [ ] Task 3.1: Mounts support (if requested)
- [ ] Task 3.2: Additional variable types (if requested)
- [ ] Task 3.3: Update documentation with limitations

**Total Estimated Effort**: 12-16 hours

### Subagent Assignment

**Agent 1 (Implementation)**:
- Create e2e test infrastructure
- Write e2e tests following TDD
- Fix any issues found

**Agent 2 (Code Review)**:
- Review e2e test coverage
- Verify tests actually use Docker (not mocks)
- Check for test isolation issues
- Ensure cleanup is robust

**Agent 3 (Documentation)**:
- Update DEVCONTAINER_GUIDE.md with testing info
- Document known limitations
- Add troubleshooting for e2e tests

## Phase 5: Success Criteria

### Test Coverage Metrics

**Minimum Requirements**:
- ✅ At least 20 e2e tests with real Docker
- ✅ Every devcontainer field has e2e test
- ✅ All tests pass with real Docker daemon
- ✅ Tests skip gracefully when Docker unavailable
- ✅ No test containers left behind after test suite

### Quality Gates

**Before Merge**:
1. All unit tests pass (existing 86+)
2. All e2e tests pass (new)
3. Code coverage maintained or improved
4. No regressions in existing functionality
5. Documentation updated

### Performance Targets

**Test Suite Performance**:
- Unit tests: < 5 seconds (current)
- E2E tests: < 60 seconds (target)
- Total CI time: < 90 seconds

**Optimization Strategies**:
- Run e2e tests in parallel where possible
- Reuse base images (alpine, ubuntu)
- Clean up immediately after each test

## Phase 6: Risk Mitigation

### Risk 1: Flaky Tests

**Concern**: Real Docker tests can be flaky (timing, network)

**Mitigation**:
- Add retry logic for container operations
- Use explicit wait functions (waitForContainer)
- Set reasonable timeouts
- Make tests hermetic (no external dependencies)

### Risk 2: CI/CD Integration

**Concern**: CI environment may not have Docker

**Mitigation**:
- Tests skip gracefully with `testing.Short()`
- Provide Docker-in-Docker CI configuration
- Document CI requirements

### Risk 3: Test Cleanup Failures

**Concern**: Failed tests may leave containers running

**Mitigation**:
- Always use `defer` for cleanup
- Label test containers for easy identification
- Provide manual cleanup script
- Add cleanup to CI post-test step

### Risk 4: Platform Differences

**Concern**: Docker behaves differently on Linux/Mac/Windows

**Mitigation**:
- Test on multiple platforms
- Document platform-specific behavior
- Use platform-agnostic assertions

## Phase 7: Documentation Updates

### Files to Update

**`README.md`**:
- Add "Testing" section
- Document how to run e2e tests
- Explain Docker requirement

**`DEVCONTAINER_GUIDE.md`**:
- Add "Known Limitations" section
- Document features vs. customizations (not supported)
- Explain initializeCommand (not supported)

**New File**: `docs/TESTING.md`
```markdown
# Testing Guide

## Running Tests

### Unit Tests
```bash
go test ./...
```

### E2E Tests (require Docker)
```bash
go test ./pkg/runner -run TestE2E
```

### Skip E2E Tests
```bash
go test -short ./...
```

## Writing E2E Tests

[Guidelines for contributors]
```

## Appendix A: Test Environment Setup

### Local Development

```bash
# Install Docker
# macOS:
brew install --cask docker

# Linux:
curl -fsSL https://get.docker.com -o get-docker.sh
sh get-docker.sh

# Verify
docker info
```

### CI/CD (GitHub Actions)

```yaml
name: Tests
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      - name: Run unit tests
        run: go test -short ./...
      - name: Run e2e tests
        run: go test ./pkg/runner -run TestE2E
```

## Appendix B: Example E2E Test

```go
func TestE2E_BasicImagePull(t *testing.T) {
    skipIfNoDocker(t)

    // Create test project
    projectDir := createTestProject(t, map[string]string{
        ".devcontainer/devcontainer.json": `{"image": "alpine:latest"}`,
    })
    defer os.RemoveAll(projectDir)

    // Unique container name
    containerName := fmt.Sprintf("packnplay-e2e-%d", time.Now().UnixNano())
    defer cleanupContainer(t, containerName)

    // Run packnplay
    cmd := exec.Command("packnplay", "run", "--project", projectDir, "echo", "hello")
    output, err := cmd.CombinedOutput()
    require.NoError(t, err, "packnplay run failed: %s", output)

    // Verify container exists
    inspect, err := inspectContainer(t, containerName)
    require.NoError(t, err)

    // Verify image
    assert.Equal(t, "alpine:latest", inspect["Config"].(map[string]interface{})["Image"])

    // Verify output
    assert.Contains(t, string(output), "hello")
}
```

## Appendix C: Known Limitations (Post-Implementation)

After implementation, document these as known limitations:

1. **Features**: Not supported (VS Code-specific, out of scope)
2. **Customizations**: Not supported (editor-specific, out of scope)
3. **initializeCommand**: Not supported (host execution security concern)
4. **updateContentCommand**: Not supported (no content change detection)
5. **postAttachCommand**: Not supported (no attach detection)
6. **Mounts**: Not yet implemented (may add if requested)

These align with packnplay's focus: **core devcontainer functionality for AI coding agents**, not full VS Code compatibility.

---

**Document Version**: 1.0
**Last Updated**: 2025-11-07
**Status**: Ready for Implementation
