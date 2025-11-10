# Devcontainer Mounts and RunArgs Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add custom mounts and runArgs support plus missing E2E test coverage for complete devcontainer feature set.

**Architecture:** Incremental addition to existing Config struct, integrate into current runner logic with full variable substitution support.

**Tech Stack:** Go, Docker CLI integration, existing devcontainer parsing, E2E test infrastructure

---

## Task 1: Add Missing E2E Tests for Existing Features

**Files:**
- Modify: `pkg/runner/e2e_test.go:1400+`

**Step 1: Write failing test for cacheFrom build feature**

```go
// TestE2E_BuildWithCacheFrom tests build cache functionality
func TestE2E_BuildWithCacheFrom(t *testing.T) {
	skipIfNoDocker(t)

	projectDir := createTestProject(t, map[string]string{
		".devcontainer/devcontainer.json": `{
			"build": {
				"dockerfile": "Dockerfile",
				"cacheFrom": ["alpine:latest"]
			}
		}`,
		".devcontainer/Dockerfile": `FROM alpine:latest
RUN echo "cached build test" > /cache-test.txt`,
	})
	defer os.RemoveAll(projectDir)

	containerName := getContainerNameForProject(projectDir)
	defer cleanupContainer(t, containerName)

	output, err := runPacknplayInDir(t, projectDir, "run", "--no-worktree", "cat", "/cache-test.txt")
	require.NoError(t, err, "Failed to run with cache: %s", output)
	require.Contains(t, output, "cached build test")
}
```

**Step 2: Write failing test for build options**

```go
// TestE2E_BuildWithOptions tests custom build options
func TestE2E_BuildWithOptions(t *testing.T) {
	skipIfNoDocker(t)

	projectDir := createTestProject(t, map[string]string{
		".devcontainer/devcontainer.json": `{
			"build": {
				"dockerfile": "Dockerfile",
				"options": ["--network=host"]
			}
		}`,
		".devcontainer/Dockerfile": `FROM alpine:latest
RUN echo "build options test" > /options-test.txt`,
	})
	defer os.RemoveAll(projectDir)

	containerName := getContainerNameForProject(projectDir)
	defer cleanupContainer(t, containerName)

	output, err := runPacknplayInDir(t, projectDir, "run", "--no-worktree", "cat", "/options-test.txt")
	require.NoError(t, err, "Failed to run with build options: %s", output)
	require.Contains(t, output, "build options test")
}
```

**Step 3: Run tests to verify they fail**

Run: `go test ./pkg/runner -run "TestE2E_BuildWithCacheFrom|TestE2E_BuildWithOptions" -v`
Expected: Both tests PASS (existing implementation should work)

**Step 4: Commit missing E2E tests**

```bash
git add pkg/runner/e2e_test.go
git commit -m "feat: add E2E tests for build cacheFrom and options

- Test cacheFrom array functionality with real Docker
- Test build options array with --network flag
- Verify existing BuildConfig implementation works end-to-end"
```

---

## Task 2: Add Config Fields for Mounts and RunArgs

**Files:**
- Modify: `pkg/devcontainer/config.go:12-25`
- Test: `pkg/devcontainer/config_test.go`

**Step 1: Add fields to Config struct**

```go
type Config struct {
	Image        string            `json:"image"`
	DockerFile   string            `json:"dockerFile"`
	Build        *BuildConfig      `json:"build,omitempty"`
	RemoteUser   string            `json:"remoteUser"`
	ContainerEnv map[string]string `json:"containerEnv,omitempty"`
	RemoteEnv    map[string]string `json:"remoteEnv,omitempty"`
	ForwardPorts []interface{}     `json:"forwardPorts,omitempty"` // int or string
	Mounts       []string          `json:"mounts,omitempty"`       // Docker mount syntax
	RunArgs      []string          `json:"runArgs,omitempty"`      // Additional docker run arguments

	// Lifecycle commands
	OnCreateCommand   *LifecycleCommand `json:"onCreateCommand,omitempty"`
	PostCreateCommand *LifecycleCommand `json:"postCreateCommand,omitempty"`
	PostStartCommand  *LifecycleCommand `json:"postStartCommand,omitempty"`
}
```

**Step 2: Write unit tests for new fields parsing**

```go
func TestConfig_MountsAndRunArgs(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		wantMounts []string
		wantRunArgs []string
	}{
		{
			name: "mounts and runArgs present",
			json: `{
				"image": "alpine:latest",
				"mounts": [
					"source=/var/run/docker.sock,target=/var/run/docker.sock,type=bind",
					"source=my-volume,target=/data,type=volume"
				],
				"runArgs": ["--memory=2g", "--cpus=2"]
			}`,
			wantMounts: []string{
				"source=/var/run/docker.sock,target=/var/run/docker.sock,type=bind",
				"source=my-volume,target=/data,type=volume",
			},
			wantRunArgs: []string{"--memory=2g", "--cpus=2"},
		},
		{
			name: "mounts and runArgs absent",
			json: `{"image": "alpine:latest"}`,
			wantMounts: nil,
			wantRunArgs: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var config Config
			err := json.Unmarshal([]byte(tt.json), &config)
			require.NoError(t, err)

			assert.Equal(t, tt.wantMounts, config.Mounts)
			assert.Equal(t, tt.wantRunArgs, config.RunArgs)
		})
	}
}
```

**Step 3: Run tests to verify parsing works**

Run: `go test ./pkg/devcontainer -run TestConfig_MountsAndRunArgs -v`
Expected: PASS

**Step 4: Commit config changes**

```bash
git add pkg/devcontainer/config.go pkg/devcontainer/config_test.go
git commit -m "feat: add mounts and runArgs fields to devcontainer Config

- Add Mounts []string for Docker mount syntax support
- Add RunArgs []string for custom docker run arguments
- Both fields optional with omitempty tags for backward compatibility
- Add comprehensive unit tests for parsing"
```

---

## Task 3: Implement Mounts Processing in Runner

**Files:**
- Modify: `pkg/runner/runner.go:635-650` (around Docker args building)

**Step 1: Add mount processing after existing volume mounts**

Find this section in `runner.go` around line 635:
```go
// Add port mappings (devcontainer ports + CLI -p flags)
for _, port := range publishPorts {
    args = append(args, "-p", port)
}
```

Add after port mappings:
```go
// Add custom mounts from devcontainer.json
for _, mount := range devConfig.Mounts {
    // Apply variable substitution to mount string
    substitutedMount, err := devcontainer.SubstituteVariables(mount, map[string]interface{}{
        "localWorkspaceFolder":     workDir,
        "containerWorkspaceFolder": "/workspace",
        "containerWorkspaceFolderBasename": filepath.Base(workDir),
    }, containerEnv)
    if err != nil {
        if config.Verbose {
            fmt.Fprintf(os.Stderr, "Warning: failed to substitute variables in mount %s: %v\n", mount, err)
        }
        substitutedMount = mount // Use original if substitution fails
    }

    // Add as Docker mount flag
    args = append(args, "--mount", substitutedMount)
}
```

**Step 2: Run existing tests to ensure no regression**

Run: `go test ./pkg/runner -run TestE2E_BasicImagePull -v`
Expected: PASS (no behavior change yet)

**Step 3: Commit mounts implementation**

```bash
git add pkg/runner/runner.go
git commit -m "feat: implement custom mounts processing in runner

- Process devcontainer mounts field in Docker container creation
- Apply variable substitution to mount strings
- Use --mount flag for full Docker mount syntax support
- Add graceful fallback if variable substitution fails"
```

---

## Task 4: Implement RunArgs Processing in Runner

**Files:**
- Modify: `pkg/runner/runner.go:635-650` (same area as Task 3)

**Step 1: Add runArgs processing before image name**

Find this section (after adding mounts):
```go
// Add image
imageName := devConfig.Image
if devConfig.HasDockerfile() {
    // Docker image names must be lowercase
    imageName = fmt.Sprintf("packnplay-%s-devcontainer:latest", strings.ToLower(projectName))
}
args = append(args, imageName)
```

Add before `args = append(args, imageName)`:
```go
// Add custom Docker run arguments from devcontainer.json
for _, runArg := range devConfig.RunArgs {
    // Apply variable substitution to run argument
    substitutedArg, err := devcontainer.SubstituteVariables(runArg, map[string]interface{}{
        "localWorkspaceFolder":     workDir,
        "containerWorkspaceFolder": "/workspace",
        "containerWorkspaceFolderBasename": filepath.Base(workDir),
    }, containerEnv)
    if err != nil {
        if config.Verbose {
            fmt.Fprintf(os.Stderr, "Warning: failed to substitute variables in runArg %s: %v\n", runArg, err)
        }
        substitutedArg = runArg // Use original if substitution fails
    }

    // Add to Docker run command
    args = append(args, substitutedArg)
}
```

**Step 2: Run existing tests to ensure no regression**

Run: `go test ./pkg/runner -run TestE2E_BasicImagePull -v`
Expected: PASS (no behavior change with empty runArgs)

**Step 3: Commit runArgs implementation**

```bash
git add pkg/runner/runner.go
git commit -m "feat: implement custom runArgs processing in runner

- Process devcontainer runArgs field in Docker container creation
- Apply variable substitution to run arguments
- Insert args before image name in docker run command
- Add graceful fallback if variable substitution fails"
```

---

## Task 5: Add E2E Tests for Custom Mounts

**Files:**
- Modify: `pkg/runner/e2e_test.go:1450+`

**Step 1: Write failing test for bind mounts**

```go
// TestE2E_CustomMounts tests custom mount configurations
func TestE2E_CustomMounts(t *testing.T) {
	skipIfNoDocker(t)

	// Create test directory with content
	testDataDir, err := os.MkdirTemp("", "packnplay-mount-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(testDataDir)

	testFile := filepath.Join(testDataDir, "mounted-file.txt")
	err = os.WriteFile(testFile, []byte("mount test content"), 0644)
	require.NoError(t, err)

	projectDir := createTestProject(t, map[string]string{
		".devcontainer/devcontainer.json": fmt.Sprintf(`{
			"image": "alpine:latest",
			"mounts": [
				"source=%s,target=/mounted-data,type=bind"
			]
		}`, testDataDir),
	})
	defer os.RemoveAll(projectDir)

	containerName := getContainerNameForProject(projectDir)
	defer cleanupContainer(t, containerName)

	output, err := runPacknplayInDir(t, projectDir, "run", "--no-worktree", "cat", "/mounted-data/mounted-file.txt")
	require.NoError(t, err, "Failed to access mounted file: %s", output)
	require.Contains(t, output, "mount test content")
}
```

**Step 2: Write test for mount variable substitution**

```go
// TestE2E_MountVariableSubstitution tests variable substitution in mounts
func TestE2E_MountVariableSubstitution(t *testing.T) {
	skipIfNoDocker(t)

	projectDir := createTestProject(t, map[string]string{
		".devcontainer/devcontainer.json": `{
			"image": "alpine:latest",
			"mounts": [
				"source=${localWorkspaceFolder}/test-data,target=/workspace-data,type=bind"
			]
		}`,
	})
	defer os.RemoveAll(projectDir)

	// Create test data in project
	testDataDir := filepath.Join(projectDir, "test-data")
	err := os.MkdirAll(testDataDir, 0755)
	require.NoError(t, err)

	testFile := filepath.Join(testDataDir, "variable-test.txt")
	err = os.WriteFile(testFile, []byte("variable substitution works"), 0644)
	require.NoError(t, err)

	containerName := getContainerNameForProject(projectDir)
	defer cleanupContainer(t, containerName)

	output, err := runPacknplayInDir(t, projectDir, "run", "--no-worktree", "cat", "/workspace-data/variable-test.txt")
	require.NoError(t, err, "Failed to access mount with variable: %s", output)
	require.Contains(t, output, "variable substitution works")
}
```

**Step 3: Run tests to verify they fail initially**

Run: `go test ./pkg/runner -run TestE2E_CustomMounts -v`
Expected: FAIL (mounts field not processed yet)

**Step 4: Run tests after Task 3 implementation**

Run: `go test ./pkg/runner -run TestE2E_CustomMounts -v`
Expected: PASS

**Step 5: Commit mount tests**

```bash
git add pkg/runner/e2e_test.go
git commit -m "feat: add E2E tests for custom mounts functionality

- Test bind mounts with real host directories
- Test variable substitution in mount source paths
- Verify mounted content accessible inside container
- Use real Docker integration for comprehensive validation"
```

---

## Task 6: Add E2E Tests for Custom RunArgs

**Files:**
- Modify: `pkg/runner/e2e_test.go:1500+`

**Step 1: Write test for resource limits**

```go
// TestE2E_CustomRunArgs tests custom Docker run arguments
func TestE2E_CustomRunArgs(t *testing.T) {
	skipIfNoDocker(t)

	projectDir := createTestProject(t, map[string]string{
		".devcontainer/devcontainer.json": `{
			"image": "alpine:latest",
			"runArgs": ["--memory=256m", "--cpus=1"]
		}`,
	})
	defer os.RemoveAll(projectDir)

	containerName := getContainerNameForProject(projectDir)
	defer cleanupContainer(t, containerName)

	// Verify container starts with resource limits
	output, err := runPacknplayInDir(t, projectDir, "run", "--no-worktree", "echo", "runargs test success")
	require.NoError(t, err, "Failed to run with custom runArgs: %s", output)
	require.Contains(t, output, "runargs test success")

	// Verify memory limit was applied by inspecting container
	containerID := getContainerIDByName(t, containerName)
	require.NotEmpty(t, containerID, "Container ID should be found")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	inspectCmd := exec.CommandContext(ctx, "docker", "inspect", containerID, "--format", "{{.HostConfig.Memory}}")
	memoryOutput, err := inspectCmd.CombinedOutput()
	require.NoError(t, err, "Failed to inspect container memory")

	// Docker returns memory in bytes, 256m = 268435456 bytes
	require.Contains(t, string(memoryOutput), "268435456", "Memory limit should be applied")
}
```

**Step 2: Write test for runArgs variable substitution**

```go
// TestE2E_RunArgsVariableSubstitution tests variable substitution in runArgs
func TestE2E_RunArgsVariableSubstitution(t *testing.T) {
	skipIfNoDocker(t)

	projectDir := createTestProject(t, map[string]string{
		".devcontainer/devcontainer.json": `{
			"image": "alpine:latest",
			"runArgs": ["--label", "project=${containerWorkspaceFolderBasename}"]
		}`,
	})
	defer os.RemoveAll(projectDir)

	containerName := getContainerNameForProject(projectDir)
	defer cleanupContainer(t, containerName)

	output, err := runPacknplayInDir(t, projectDir, "run", "--no-worktree", "echo", "variable runargs success")
	require.NoError(t, err, "Failed to run with variable runArgs: %s", output)

	// Verify label was applied with substituted variable
	containerID := getContainerIDByName(t, containerName)
	require.NotEmpty(t, containerID)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	expectedLabel := fmt.Sprintf("project=%s", filepath.Base(projectDir))
	inspectCmd := exec.CommandContext(ctx, "docker", "inspect", containerID, "--format", "{{index .Config.Labels \"project\"}}")
	labelOutput, err := inspectCmd.CombinedOutput()
	require.NoError(t, err, "Failed to inspect container labels")
	require.Contains(t, string(labelOutput), filepath.Base(projectDir), "Variable substitution should work in runArgs")
}
```

**Step 3: Run tests to verify they fail initially**

Run: `go test ./pkg/runner -run TestE2E.*RunArgs -v`
Expected: FAIL (runArgs field not processed yet)

**Step 4: Run tests after Task 4 implementation**

Run: `go test ./pkg/runner -run TestE2E.*RunArgs -v`
Expected: PASS

**Step 5: Commit runArgs tests**

```bash
git add pkg/runner/e2e_test.go
git commit -m "feat: add E2E tests for custom runArgs functionality

- Test resource limits with memory and CPU constraints
- Test variable substitution in runArgs values
- Verify Docker container inspection shows applied arguments
- Use real Docker validation for comprehensive testing"
```

---

## Task 7: Add Error Handling and Timeout Tests

**Files:**
- Modify: `pkg/runner/e2e_test.go:1550+`

**Step 1: Write test for lifecycle command failure handling**

```go
// TestE2E_LifecycleCommandErrors tests error handling for failed lifecycle commands
func TestE2E_LifecycleCommandErrors(t *testing.T) {
	skipIfNoDocker(t)

	projectDir := createTestProject(t, map[string]string{
		".devcontainer/devcontainer.json": `{
			"image": "alpine:latest",
			"postCreateCommand": "exit 1"
		}`,
	})
	defer os.RemoveAll(projectDir)

	containerName := getContainerNameForProject(projectDir)
	defer cleanupContainer(t, containerName)

	// Lifecycle command failure should not prevent container startup
	output, err := runPacknplayInDir(t, projectDir, "run", "--no-worktree", "echo", "container still works")
	require.NoError(t, err, "Container should start despite lifecycle command failure")
	require.Contains(t, output, "container still works")

	// But should log the warning
	require.Contains(t, output, "postCreateCommand failed", "Should warn about lifecycle command failure")
}
```

**Step 2: Run test to verify current error handling**

Run: `go test ./pkg/runner -run TestE2E_LifecycleCommandErrors -v`
Expected: Should PASS if current error handling works correctly

**Step 3: Commit error handling test**

```bash
git add pkg/runner/e2e_test.go
git commit -m "feat: add E2E test for lifecycle command error handling

- Test that failed lifecycle commands don't prevent container startup
- Verify warning messages are displayed for failed commands
- Ensure container remains functional despite command failures"
```

---

## Task 8: Final Integration Testing and Verification

**Files:**
- Run comprehensive test suite

**Step 1: Run full devcontainer package tests**

Run: `go test ./pkg/devcontainer -v`
Expected: All tests PASS with new fields

**Step 2: Run full runner package tests**

Run: `go test ./pkg/runner -v`
Expected: All 28+ E2E tests PASS including new ones

**Step 3: Run complete test suite**

Run: `make test`
Expected: Full suite PASS with no regressions

**Step 4: Test real-world scenario**

Create manual test with complex devcontainer.json:
```json
{
  "image": "alpine:latest",
  "mounts": [
    "source=/var/run/docker.sock,target=/var/run/docker.sock,type=bind",
    "type=tmpfs,target=/tmp"
  ],
  "runArgs": ["--memory=512m", "--label", "test=integration"]
}
```

Run: `packnplay run --no-worktree echo "integration test"`
Expected: SUCCESS with mounts and args applied

**Step 5: Update documentation**

Add examples to existing documentation showing new features work.

**Step 6: Final commit**

```bash
git add -A
git commit -m "feat: complete devcontainer mounts and runArgs implementation

- Full support for custom Docker mount syntax
- Full support for custom Docker run arguments
- Variable substitution in both mounts and runArgs
- Comprehensive E2E test coverage
- Backward compatible with existing configurations
- Documentation updated with examples"
```

---

## Verification Commands

**Test new functionality:**
```bash
# Test parsing
go test ./pkg/devcontainer -run "Mounts|RunArgs" -v

# Test E2E mounts
go test ./pkg/runner -run TestE2E_CustomMounts -v

# Test E2E runArgs
go test ./pkg/runner -run TestE2E.*RunArgs -v

# Test full suite
make test
```

**Manual verification:**
```bash
# Create test with new features
mkdir -p test-new-features/.devcontainer
echo '{"image":"alpine:latest","mounts":["type=tmpfs,target=/tmp"],"runArgs":["--memory=256m"]}' > test-new-features/.devcontainer/devcontainer.json
cd test-new-features
packnplay run --no-worktree echo "new features work"
```

## Success Criteria

- ✅ All existing tests continue to pass (backward compatibility)
- ✅ New mounts field parsed and applied to Docker containers
- ✅ New runArgs field parsed and applied to Docker run command
- ✅ Variable substitution works in both new fields
- ✅ Comprehensive E2E test coverage for new functionality
- ✅ Real-world manual testing succeeds
- ✅ Documentation updated with examples