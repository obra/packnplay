# Devcontainer Full Test Coverage Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Achieve full test coverage for all implemented devcontainer features and implement the missing lifecycle command execution.

**Architecture:** Add E2E tests for feature security properties (privileged, capAdd, securityOpt, init, entrypoint, mounts) that are implemented but untested. Implement execution of updateContentCommand and postAttachCommand lifecycle hooks. Each test creates a temporary project with devcontainer.json, runs packnplay, and verifies the expected behavior.

**Tech Stack:** Go testing, Docker, existing E2E test patterns in `pkg/runner/e2e_test.go`

---

## Background

### Current State (from code audit)
- 42 E2E tests exist covering core functionality
- Feature security properties ARE implemented in `pkg/runner/runner.go:93-107` but have NO tests
- `updateContentCommand` and `postAttachCommand` are parsed into Config but NEVER executed
- `initializeCommand` runs on HOST - intentionally skipped (security)

### Files Overview
- **Main test file:** `pkg/runner/e2e_test.go` (2222 lines, 42 tests)
- **Feature properties applier:** `pkg/runner/runner.go` lines 80-120
- **Lifecycle executor:** `pkg/runner/lifecycle_executor.go`
- **Config struct:** `pkg/devcontainer/config.go` lines 11-40

---

## Part 1: Tests for Implemented-But-Untested Features

### Task 1: Test Feature Privileged Mode

**Files:**
- Modify: `pkg/runner/e2e_test.go` (append test)

**Step 1: Write the failing test**

Add to end of `pkg/runner/e2e_test.go`:

```go
// TestE2E_FeaturePrivilegedMode tests that features can request privileged mode
func TestE2E_FeaturePrivilegedMode(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx := context.Background()
	client, err := docker.NewClientWithOpts(docker.FromEnv, docker.WithAPIVersionNegotiation())
	if err != nil {
		t.Skip("Docker daemon not available - skipping E2E test")
	}
	defer client.Close()

	// Create temp project with local feature requesting privileged
	projectDir := t.TempDir()

	// Feature metadata requesting privileged mode
	featureMetadata := `{
		"id": "privileged-feature",
		"version": "1.0.0",
		"name": "Privileged Feature",
		"privileged": true
	}`

	installScript := `#!/bin/bash
set -e
echo "PRIVILEGED_FEATURE_INSTALLED=true" >> /etc/environment
touch /tmp/privileged-feature-installed
`

	devcontainerJSON := `{
		"image": "ubuntu:22.04",
		"features": {
			"./local-features/privileged-feature": {}
		}
	}`

	// Create directory structure
	files := map[string]string{
		".devcontainer/devcontainer.json":                              devcontainerJSON,
		".devcontainer/local-features/privileged-feature/devcontainer-feature.json": featureMetadata,
		".devcontainer/local-features/privileged-feature/install.sh":                installScript,
	}

	for path, content := range files {
		fullPath := filepath.Join(projectDir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0755); err != nil {
			t.Fatalf("Failed to write file %s: %v", path, err)
		}
	}

	// Initialize git repo (required by packnplay)
	cmd := exec.Command("git", "init")
	cmd.Dir = projectDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	containerName := fmt.Sprintf("test-privileged-%d", time.Now().UnixNano())

	// Clean up
	defer func() {
		_ = client.ContainerRemove(ctx, containerName, container.RemoveOptions{Force: true})
		// Clean up any images built
		images, _ := client.ImageList(ctx, image.ListOptions{})
		for _, img := range images {
			for _, tag := range img.RepoTags {
				if strings.Contains(tag, "test-privileged") {
					_, _ = client.ImageRemove(ctx, img.ID, image.RemoveOptions{Force: true})
				}
			}
		}
	}()

	// Run packnplay to build and start container
	runner := NewRunner(projectDir, containerName, "bash", []string{"-c", "cat /proc/1/status | grep CapEff"}, true)
	runner.NoWorktree = true

	launchInfo, err := runner.Run(ctx)
	if err != nil {
		t.Fatalf("Runner.Run() failed: %v", err)
	}

	// Verify container was created with privileged mode
	containerInfo, err := client.ContainerInspect(ctx, launchInfo.ContainerName)
	if err != nil {
		t.Fatalf("Failed to inspect container: %v", err)
	}

	if !containerInfo.HostConfig.Privileged {
		t.Errorf("Container should be privileged but HostConfig.Privileged = false")
	}
}
```

**Step 2: Run test to verify behavior**

Run: `go test -v ./pkg/runner -run TestE2E_FeaturePrivilegedMode -timeout 5m`

Expected: Test should pass if privileged mode is correctly applied, fail if not.

**Step 3: Commit**

```bash
git add pkg/runner/e2e_test.go
git commit -m "test: add E2E test for feature privileged mode"
```

---

### Task 2: Test Feature capAdd

**Files:**
- Modify: `pkg/runner/e2e_test.go` (append test)

**Step 1: Write the failing test**

Add to end of `pkg/runner/e2e_test.go`:

```go
// TestE2E_FeatureCapAdd tests that features can request Linux capabilities
func TestE2E_FeatureCapAdd(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx := context.Background()
	client, err := docker.NewClientWithOpts(docker.FromEnv, docker.WithAPIVersionNegotiation())
	if err != nil {
		t.Skip("Docker daemon not available - skipping E2E test")
	}
	defer client.Close()

	projectDir := t.TempDir()

	// Feature metadata requesting NET_ADMIN capability
	featureMetadata := `{
		"id": "capadd-feature",
		"version": "1.0.0",
		"name": "CapAdd Feature",
		"capAdd": ["NET_ADMIN", "SYS_PTRACE"]
	}`

	installScript := `#!/bin/bash
touch /tmp/capadd-feature-installed
`

	devcontainerJSON := `{
		"image": "ubuntu:22.04",
		"features": {
			"./local-features/capadd-feature": {}
		}
	}`

	files := map[string]string{
		".devcontainer/devcontainer.json":                            devcontainerJSON,
		".devcontainer/local-features/capadd-feature/devcontainer-feature.json": featureMetadata,
		".devcontainer/local-features/capadd-feature/install.sh":                installScript,
	}

	for path, content := range files {
		fullPath := filepath.Join(projectDir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0755); err != nil {
			t.Fatalf("Failed to write file %s: %v", path, err)
		}
	}

	cmd := exec.Command("git", "init")
	cmd.Dir = projectDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	containerName := fmt.Sprintf("test-capadd-%d", time.Now().UnixNano())

	defer func() {
		_ = client.ContainerRemove(ctx, containerName, container.RemoveOptions{Force: true})
	}()

	runner := NewRunner(projectDir, containerName, "bash", []string{"-c", "echo done"}, true)
	runner.NoWorktree = true

	launchInfo, err := runner.Run(ctx)
	if err != nil {
		t.Fatalf("Runner.Run() failed: %v", err)
	}

	containerInfo, err := client.ContainerInspect(ctx, launchInfo.ContainerName)
	if err != nil {
		t.Fatalf("Failed to inspect container: %v", err)
	}

	// Check that capabilities were added
	capAdd := containerInfo.HostConfig.CapAdd
	hasNetAdmin := false
	hasSysPtrace := false
	for _, cap := range capAdd {
		if cap == "NET_ADMIN" {
			hasNetAdmin = true
		}
		if cap == "SYS_PTRACE" {
			hasSysPtrace = true
		}
	}

	if !hasNetAdmin {
		t.Errorf("Container should have NET_ADMIN capability, got: %v", capAdd)
	}
	if !hasSysPtrace {
		t.Errorf("Container should have SYS_PTRACE capability, got: %v", capAdd)
	}
}
```

**Step 2: Run test**

Run: `go test -v ./pkg/runner -run TestE2E_FeatureCapAdd -timeout 5m`

**Step 3: Commit**

```bash
git add pkg/runner/e2e_test.go
git commit -m "test: add E2E test for feature capAdd"
```

---

### Task 3: Test Feature securityOpt

**Files:**
- Modify: `pkg/runner/e2e_test.go` (append test)

**Step 1: Write the failing test**

Add to end of `pkg/runner/e2e_test.go`:

```go
// TestE2E_FeatureSecurityOpt tests that features can request security options
func TestE2E_FeatureSecurityOpt(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx := context.Background()
	client, err := docker.NewClientWithOpts(docker.FromEnv, docker.WithAPIVersionNegotiation())
	if err != nil {
		t.Skip("Docker daemon not available - skipping E2E test")
	}
	defer client.Close()

	projectDir := t.TempDir()

	// Feature metadata requesting security options
	featureMetadata := `{
		"id": "secopt-feature",
		"version": "1.0.0",
		"name": "SecurityOpt Feature",
		"securityOpt": ["seccomp=unconfined"]
	}`

	installScript := `#!/bin/bash
touch /tmp/secopt-feature-installed
`

	devcontainerJSON := `{
		"image": "ubuntu:22.04",
		"features": {
			"./local-features/secopt-feature": {}
		}
	}`

	files := map[string]string{
		".devcontainer/devcontainer.json":                            devcontainerJSON,
		".devcontainer/local-features/secopt-feature/devcontainer-feature.json": featureMetadata,
		".devcontainer/local-features/secopt-feature/install.sh":                installScript,
	}

	for path, content := range files {
		fullPath := filepath.Join(projectDir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0755); err != nil {
			t.Fatalf("Failed to write file %s: %v", path, err)
		}
	}

	cmd := exec.Command("git", "init")
	cmd.Dir = projectDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	containerName := fmt.Sprintf("test-secopt-%d", time.Now().UnixNano())

	defer func() {
		_ = client.ContainerRemove(ctx, containerName, container.RemoveOptions{Force: true})
	}()

	runner := NewRunner(projectDir, containerName, "bash", []string{"-c", "echo done"}, true)
	runner.NoWorktree = true

	launchInfo, err := runner.Run(ctx)
	if err != nil {
		t.Fatalf("Runner.Run() failed: %v", err)
	}

	containerInfo, err := client.ContainerInspect(ctx, launchInfo.ContainerName)
	if err != nil {
		t.Fatalf("Failed to inspect container: %v", err)
	}

	// Check that security options were added
	secOpts := containerInfo.HostConfig.SecurityOpt
	hasSeccompUnconfined := false
	for _, opt := range secOpts {
		if opt == "seccomp=unconfined" {
			hasSeccompUnconfined = true
		}
	}

	if !hasSeccompUnconfined {
		t.Errorf("Container should have seccomp=unconfined, got: %v", secOpts)
	}
}
```

**Step 2: Run test**

Run: `go test -v ./pkg/runner -run TestE2E_FeatureSecurityOpt -timeout 5m`

**Step 3: Commit**

```bash
git add pkg/runner/e2e_test.go
git commit -m "test: add E2E test for feature securityOpt"
```

---

### Task 4: Test Feature Init

**Files:**
- Modify: `pkg/runner/e2e_test.go` (append test)

**Step 1: Write the failing test**

Add to end of `pkg/runner/e2e_test.go`:

```go
// TestE2E_FeatureInit tests that features can request --init for proper signal handling
func TestE2E_FeatureInit(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx := context.Background()
	client, err := docker.NewClientWithOpts(docker.FromEnv, docker.WithAPIVersionNegotiation())
	if err != nil {
		t.Skip("Docker daemon not available - skipping E2E test")
	}
	defer client.Close()

	projectDir := t.TempDir()

	// Feature metadata requesting init process
	featureMetadata := `{
		"id": "init-feature",
		"version": "1.0.0",
		"name": "Init Feature",
		"init": true
	}`

	installScript := `#!/bin/bash
touch /tmp/init-feature-installed
`

	devcontainerJSON := `{
		"image": "ubuntu:22.04",
		"features": {
			"./local-features/init-feature": {}
		}
	}`

	files := map[string]string{
		".devcontainer/devcontainer.json":                          devcontainerJSON,
		".devcontainer/local-features/init-feature/devcontainer-feature.json": featureMetadata,
		".devcontainer/local-features/init-feature/install.sh":                installScript,
	}

	for path, content := range files {
		fullPath := filepath.Join(projectDir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0755); err != nil {
			t.Fatalf("Failed to write file %s: %v", path, err)
		}
	}

	cmd := exec.Command("git", "init")
	cmd.Dir = projectDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	containerName := fmt.Sprintf("test-init-%d", time.Now().UnixNano())

	defer func() {
		_ = client.ContainerRemove(ctx, containerName, container.RemoveOptions{Force: true})
	}()

	runner := NewRunner(projectDir, containerName, "bash", []string{"-c", "echo done"}, true)
	runner.NoWorktree = true

	launchInfo, err := runner.Run(ctx)
	if err != nil {
		t.Fatalf("Runner.Run() failed: %v", err)
	}

	containerInfo, err := client.ContainerInspect(ctx, launchInfo.ContainerName)
	if err != nil {
		t.Fatalf("Failed to inspect container: %v", err)
	}

	// Check that init was enabled
	if containerInfo.HostConfig.Init == nil || !*containerInfo.HostConfig.Init {
		t.Errorf("Container should have Init=true, got: %v", containerInfo.HostConfig.Init)
	}
}
```

**Step 2: Run test**

Run: `go test -v ./pkg/runner -run TestE2E_FeatureInit -timeout 5m`

**Step 3: Commit**

```bash
git add pkg/runner/e2e_test.go
git commit -m "test: add E2E test for feature init"
```

---

### Task 5: Test Feature Entrypoint

**Files:**
- Modify: `pkg/runner/e2e_test.go` (append test)

**Step 1: Write the failing test**

Add to end of `pkg/runner/e2e_test.go`:

```go
// TestE2E_FeatureEntrypoint tests that features can override container entrypoint
func TestE2E_FeatureEntrypoint(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx := context.Background()
	client, err := docker.NewClientWithOpts(docker.FromEnv, docker.WithAPIVersionNegotiation())
	if err != nil {
		t.Skip("Docker daemon not available - skipping E2E test")
	}
	defer client.Close()

	projectDir := t.TempDir()

	// Feature metadata requesting custom entrypoint
	featureMetadata := `{
		"id": "entrypoint-feature",
		"version": "1.0.0",
		"name": "Entrypoint Feature",
		"entrypoint": ["/bin/sh", "-c"]
	}`

	installScript := `#!/bin/bash
touch /tmp/entrypoint-feature-installed
`

	devcontainerJSON := `{
		"image": "ubuntu:22.04",
		"features": {
			"./local-features/entrypoint-feature": {}
		}
	}`

	files := map[string]string{
		".devcontainer/devcontainer.json":                                devcontainerJSON,
		".devcontainer/local-features/entrypoint-feature/devcontainer-feature.json": featureMetadata,
		".devcontainer/local-features/entrypoint-feature/install.sh":                installScript,
	}

	for path, content := range files {
		fullPath := filepath.Join(projectDir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0755); err != nil {
			t.Fatalf("Failed to write file %s: %v", path, err)
		}
	}

	cmd := exec.Command("git", "init")
	cmd.Dir = projectDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	containerName := fmt.Sprintf("test-entrypoint-%d", time.Now().UnixNano())

	defer func() {
		_ = client.ContainerRemove(ctx, containerName, container.RemoveOptions{Force: true})
	}()

	runner := NewRunner(projectDir, containerName, "bash", []string{"-c", "echo done"}, true)
	runner.NoWorktree = true

	launchInfo, err := runner.Run(ctx)
	if err != nil {
		t.Fatalf("Runner.Run() failed: %v", err)
	}

	containerInfo, err := client.ContainerInspect(ctx, launchInfo.ContainerName)
	if err != nil {
		t.Fatalf("Failed to inspect container: %v", err)
	}

	// Check that entrypoint was set
	// Note: The entrypoint might be modified by how docker stores it
	entrypoint := containerInfo.Config.Entrypoint
	if len(entrypoint) == 0 {
		t.Errorf("Container should have custom entrypoint, got empty")
	}
}
```

**Step 2: Run test**

Run: `go test -v ./pkg/runner -run TestE2E_FeatureEntrypoint -timeout 5m`

**Step 3: Commit**

```bash
git add pkg/runner/e2e_test.go
git commit -m "test: add E2E test for feature entrypoint"
```

---

### Task 6: Test Feature Mounts

**Files:**
- Modify: `pkg/runner/e2e_test.go` (append test)

**Step 1: Write the failing test**

Add to end of `pkg/runner/e2e_test.go`:

```go
// TestE2E_FeatureMounts tests that features can contribute volume mounts
func TestE2E_FeatureMounts(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx := context.Background()
	client, err := docker.NewClientWithOpts(docker.FromEnv, docker.WithAPIVersionNegotiation())
	if err != nil {
		t.Skip("Docker daemon not available - skipping E2E test")
	}
	defer client.Close()

	projectDir := t.TempDir()

	// Feature metadata requesting a tmpfs mount
	featureMetadata := `{
		"id": "mount-feature",
		"version": "1.0.0",
		"name": "Mount Feature",
		"mounts": [
			{
				"type": "tmpfs",
				"target": "/feature-tmpfs"
			}
		]
	}`

	installScript := `#!/bin/bash
touch /tmp/mount-feature-installed
`

	devcontainerJSON := `{
		"image": "ubuntu:22.04",
		"features": {
			"./local-features/mount-feature": {}
		}
	}`

	files := map[string]string{
		".devcontainer/devcontainer.json":                           devcontainerJSON,
		".devcontainer/local-features/mount-feature/devcontainer-feature.json": featureMetadata,
		".devcontainer/local-features/mount-feature/install.sh":                installScript,
	}

	for path, content := range files {
		fullPath := filepath.Join(projectDir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0755); err != nil {
			t.Fatalf("Failed to write file %s: %v", path, err)
		}
	}

	cmd := exec.Command("git", "init")
	cmd.Dir = projectDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	containerName := fmt.Sprintf("test-mount-%d", time.Now().UnixNano())

	defer func() {
		_ = client.ContainerRemove(ctx, containerName, container.RemoveOptions{Force: true})
	}()

	// Run and check if the mount point exists
	runner := NewRunner(projectDir, containerName, "bash", []string{"-c", "test -d /feature-tmpfs && echo MOUNT_EXISTS"}, true)
	runner.NoWorktree = true

	launchInfo, err := runner.Run(ctx)
	if err != nil {
		t.Fatalf("Runner.Run() failed: %v", err)
	}

	// Inspect container to verify mount
	containerInfo, err := client.ContainerInspect(ctx, launchInfo.ContainerName)
	if err != nil {
		t.Fatalf("Failed to inspect container: %v", err)
	}

	// Check mounts for our feature tmpfs
	foundMount := false
	for _, mount := range containerInfo.Mounts {
		if mount.Destination == "/feature-tmpfs" && mount.Type == "tmpfs" {
			foundMount = true
			break
		}
	}

	if !foundMount {
		t.Errorf("Container should have /feature-tmpfs mount, mounts: %+v", containerInfo.Mounts)
	}
}
```

**Step 2: Run test**

Run: `go test -v ./pkg/runner -run TestE2E_FeatureMounts -timeout 5m`

**Step 3: Commit**

```bash
git add pkg/runner/e2e_test.go
git commit -m "test: add E2E test for feature mounts"
```

---

## Part 2: Implement Missing Lifecycle Command Execution

### Task 7: Implement updateContentCommand Execution

**Files:**
- Modify: `pkg/runner/runner.go`

**Step 1: Write the failing test**

Add to end of `pkg/runner/e2e_test.go`:

```go
// TestE2E_UpdateContentCommand tests that updateContentCommand is executed
func TestE2E_UpdateContentCommand(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx := context.Background()
	client, err := docker.NewClientWithOpts(docker.FromEnv, docker.WithAPIVersionNegotiation())
	if err != nil {
		t.Skip("Docker daemon not available - skipping E2E test")
	}
	defer client.Close()

	projectDir := t.TempDir()

	devcontainerJSON := `{
		"image": "ubuntu:22.04",
		"updateContentCommand": "touch /tmp/update-content-ran"
	}`

	files := map[string]string{
		".devcontainer/devcontainer.json": devcontainerJSON,
	}

	for path, content := range files {
		fullPath := filepath.Join(projectDir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write file %s: %v", path, err)
		}
	}

	cmd := exec.Command("git", "init")
	cmd.Dir = projectDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	containerName := fmt.Sprintf("test-updatecontent-%d", time.Now().UnixNano())

	defer func() {
		_ = client.ContainerRemove(ctx, containerName, container.RemoveOptions{Force: true})
	}()

	runner := NewRunner(projectDir, containerName, "bash", []string{"-c", "test -f /tmp/update-content-ran && echo UPDATE_CONTENT_RAN"}, true)
	runner.NoWorktree = true

	_, err = runner.Run(ctx)
	if err != nil {
		t.Fatalf("Runner.Run() failed: %v", err)
	}

	// Execute a command to check if the file exists
	execConfig := container.ExecOptions{
		Cmd:          []string{"test", "-f", "/tmp/update-content-ran"},
		AttachStdout: true,
		AttachStderr: true,
	}

	execResp, err := client.ContainerExecCreate(ctx, containerName, execConfig)
	if err != nil {
		t.Fatalf("Failed to create exec: %v", err)
	}

	err = client.ContainerExecStart(ctx, execResp.ID, container.ExecStartOptions{})
	if err != nil {
		t.Fatalf("Failed to start exec: %v", err)
	}

	// Check exit code
	inspectResp, err := client.ContainerExecInspect(ctx, execResp.ID)
	if err != nil {
		t.Fatalf("Failed to inspect exec: %v", err)
	}

	if inspectResp.ExitCode != 0 {
		t.Errorf("updateContentCommand should have created /tmp/update-content-ran, but file not found")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test -v ./pkg/runner -run TestE2E_UpdateContentCommand -timeout 5m`

Expected: FAIL - updateContentCommand not executed

**Step 3: Implement updateContentCommand execution**

In `pkg/runner/runner.go`, find the lifecycle command execution section (around line 1001-1010) and add updateContentCommand:

```go
// After postCreateCmd execution, add:
updateContentCmd := devConfig.UpdateContentCommand
```

Then in the execution block (around line 1040), add execution of updateContentCommand after onCreateCommand:

Find the section that executes lifecycle commands and add:

```go
// Execute updateContentCommand (after workspace is mounted)
if updateContentCmd != nil {
    if r.Verbose {
        fmt.Println("Running updateContentCommand...")
    }
    if err := executor.Execute("updateContent", updateContentCmd); err != nil {
        return nil, fmt.Errorf("updateContentCommand failed: %w", err)
    }
}
```

**Step 4: Run test to verify it passes**

Run: `go test -v ./pkg/runner -run TestE2E_UpdateContentCommand -timeout 5m`

Expected: PASS

**Step 5: Commit**

```bash
git add pkg/runner/runner.go pkg/runner/e2e_test.go
git commit -m "feat: implement updateContentCommand execution"
```

---

### Task 8: Implement postAttachCommand Execution

**Files:**
- Modify: `pkg/runner/runner.go`
- Modify: `pkg/runner/e2e_test.go`

**Step 1: Write the failing test**

Add to end of `pkg/runner/e2e_test.go`:

```go
// TestE2E_PostAttachCommand tests that postAttachCommand is executed
func TestE2E_PostAttachCommand(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx := context.Background()
	client, err := docker.NewClientWithOpts(docker.FromEnv, docker.WithAPIVersionNegotiation())
	if err != nil {
		t.Skip("Docker daemon not available - skipping E2E test")
	}
	defer client.Close()

	projectDir := t.TempDir()

	devcontainerJSON := `{
		"image": "ubuntu:22.04",
		"postAttachCommand": "touch /tmp/post-attach-ran"
	}`

	files := map[string]string{
		".devcontainer/devcontainer.json": devcontainerJSON,
	}

	for path, content := range files {
		fullPath := filepath.Join(projectDir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write file %s: %v", path, err)
		}
	}

	cmd := exec.Command("git", "init")
	cmd.Dir = projectDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	containerName := fmt.Sprintf("test-postattach-%d", time.Now().UnixNano())

	defer func() {
		_ = client.ContainerRemove(ctx, containerName, container.RemoveOptions{Force: true})
	}()

	runner := NewRunner(projectDir, containerName, "bash", []string{"-c", "echo done"}, true)
	runner.NoWorktree = true

	_, err = runner.Run(ctx)
	if err != nil {
		t.Fatalf("Runner.Run() failed: %v", err)
	}

	// Check if the file exists
	execConfig := container.ExecOptions{
		Cmd:          []string{"test", "-f", "/tmp/post-attach-ran"},
		AttachStdout: true,
		AttachStderr: true,
	}

	execResp, err := client.ContainerExecCreate(ctx, containerName, execConfig)
	if err != nil {
		t.Fatalf("Failed to create exec: %v", err)
	}

	err = client.ContainerExecStart(ctx, execResp.ID, container.ExecStartOptions{})
	if err != nil {
		t.Fatalf("Failed to start exec: %v", err)
	}

	inspectResp, err := client.ContainerExecInspect(ctx, execResp.ID)
	if err != nil {
		t.Fatalf("Failed to inspect exec: %v", err)
	}

	if inspectResp.ExitCode != 0 {
		t.Errorf("postAttachCommand should have created /tmp/post-attach-ran, but file not found")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test -v ./pkg/runner -run TestE2E_PostAttachCommand -timeout 5m`

Expected: FAIL - postAttachCommand not executed

**Step 3: Implement postAttachCommand execution**

In `pkg/runner/runner.go`, add postAttachCommand to the lifecycle variables section and add execution.

Add to variable declarations:
```go
postAttachCmd := devConfig.PostAttachCommand
```

Add to execution block (after postStartCommand):
```go
// Execute postAttachCommand (runs every time user attaches)
if postAttachCmd != nil {
    if r.Verbose {
        fmt.Println("Running postAttachCommand...")
    }
    if err := executor.Execute("postAttach", postAttachCmd); err != nil {
        return nil, fmt.Errorf("postAttachCommand failed: %w", err)
    }
}
```

**Step 4: Run test to verify it passes**

Run: `go test -v ./pkg/runner -run TestE2E_PostAttachCommand -timeout 5m`

Expected: PASS

**Step 5: Commit**

```bash
git add pkg/runner/runner.go pkg/runner/e2e_test.go
git commit -m "feat: implement postAttachCommand execution"
```

---

## Part 3: Update Skipped Microsoft Compliance Tests

### Task 9: Unskip and Fix Microsoft Compliance Tests

**Files:**
- Modify: `pkg/devcontainer/microsoft_compliance_test.go`

**Step 1: Review the skipped tests**

The file contains 3 skipped tests:
- `TestMicrosoftComplianceDependencyResolution`
- `TestMicrosoftComplianceLifecycleHooks`
- `TestMicrosoftComplianceE2EFeatures`

**Step 2: Implement TestMicrosoftComplianceDependencyResolution**

Replace the skipped test with:

```go
// TestMicrosoftComplianceDependencyResolution tests dependency resolution algorithm
func TestMicrosoftComplianceDependencyResolution(t *testing.T) {
	tests := []struct {
		name           string
		features       map[string]FeatureMetadata
		expectedOrder  []string
		shouldError    bool
		errorContains  string
	}{
		{
			name: "simple linear dependency A->B->C",
			features: map[string]FeatureMetadata{
				"feature-a": {ID: "feature-a", DependsOn: []string{"feature-b"}},
				"feature-b": {ID: "feature-b", DependsOn: []string{"feature-c"}},
				"feature-c": {ID: "feature-c"},
			},
			expectedOrder: []string{"feature-c", "feature-b", "feature-a"},
			shouldError:   false,
		},
		{
			name: "no dependencies - order preserved",
			features: map[string]FeatureMetadata{
				"feature-a": {ID: "feature-a"},
				"feature-b": {ID: "feature-b"},
			},
			expectedOrder: []string{"feature-a", "feature-b"},
			shouldError:   false,
		},
		{
			name: "installsAfter soft dependency",
			features: map[string]FeatureMetadata{
				"feature-a": {ID: "feature-a", InstallsAfter: []string{"feature-b"}},
				"feature-b": {ID: "feature-b"},
			},
			expectedOrder: []string{"feature-b", "feature-a"},
			shouldError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create ResolvedFeature list from metadata
			var features []*ResolvedFeature
			for id, meta := range tt.features {
				metaCopy := meta
				features = append(features, &ResolvedFeature{
					ID:       id,
					Metadata: &metaCopy,
				})
			}

			// Resolve dependencies
			ordered, err := ResolveDependencies(features)

			if tt.shouldError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
				return
			}

			assert.NoError(t, err)

			// Extract IDs from ordered result
			var orderedIDs []string
			for _, f := range ordered {
				orderedIDs = append(orderedIDs, f.ID)
			}

			assert.Equal(t, tt.expectedOrder, orderedIDs, "Dependency order should match")
		})
	}
}
```

**Step 3: Implement TestMicrosoftComplianceLifecycleHooks**

Replace with actual test:

```go
// TestMicrosoftComplianceLifecycleHooks tests lifecycle command execution order
func TestMicrosoftComplianceLifecycleHooks(t *testing.T) {
	// Test that lifecycle merger correctly orders feature commands before user commands
	merger := NewLifecycleMerger()

	// Create feature lifecycle commands
	featureOnCreate := &LifecycleCommand{}
	featureOnCreate.SetString("echo feature-oncreate")

	userOnCreate := &LifecycleCommand{}
	userOnCreate.SetString("echo user-oncreate")

	// Feature metadata with lifecycle commands
	featureMeta := &FeatureMetadata{
		ID:              "test-feature",
		OnCreateCommand: featureOnCreate,
	}

	// User config with lifecycle commands
	userConfig := &Config{
		OnCreateCommand: userOnCreate,
	}

	// Merge commands
	merged := merger.MergeLifecycleCommands([]*FeatureMetadata{featureMeta}, userConfig)

	// Verify onCreate has both commands with feature first
	assert.NotNil(t, merged.OnCreate)

	// The merged command should be a MergedCommands type
	if merged.OnCreate.IsMerged() {
		commands, _ := merged.OnCreate.AsMerged()
		assert.Len(t, commands, 2, "Should have 2 commands merged")
		// Feature command should come first
		firstCmd, _ := commands[0].AsString()
		assert.Equal(t, "echo feature-oncreate", firstCmd)
	}
}
```

**Step 4: Run tests**

Run: `go test -v ./pkg/devcontainer -run TestMicrosoftCompliance`

**Step 5: Commit**

```bash
git add pkg/devcontainer/microsoft_compliance_test.go
git commit -m "test: implement Microsoft compliance tests for dependency resolution and lifecycle hooks"
```

---

## Part 4: Run Full Test Suite and Verify

### Task 10: Run All Tests and Document Coverage

**Step 1: Run full test suite**

Run: `go test -v ./... -timeout 30m 2>&1 | tee test-output.txt`

**Step 2: Count test coverage**

Run: `go test -cover ./pkg/runner ./pkg/devcontainer`

**Step 3: Document final coverage**

Create a summary of:
- Total E2E tests (should be 50+)
- Coverage percentage
- Any remaining gaps

**Step 4: Final commit**

```bash
git add -A
git commit -m "test: complete devcontainer test coverage implementation"
```

---

## Summary

| Task | Type | Estimated LOC |
|------|------|---------------|
| Task 1: Test privileged mode | Test | ~80 |
| Task 2: Test capAdd | Test | ~80 |
| Task 3: Test securityOpt | Test | ~70 |
| Task 4: Test init | Test | ~70 |
| Task 5: Test entrypoint | Test | ~70 |
| Task 6: Test feature mounts | Test | ~80 |
| Task 7: updateContentCommand | Test + Impl | ~100 |
| Task 8: postAttachCommand | Test + Impl | ~100 |
| Task 9: Microsoft compliance | Test | ~100 |
| Task 10: Verify coverage | Verification | ~20 |
| **Total** | | **~770 LOC** |

## Decision Points for Jesse

Before executing, please confirm:

1. **initializeCommand** - This runs on the HOST before container creation. Should we implement it? (Security implications - arbitrary host code execution from devcontainer.json)

2. **postAttachCommand behavior** - In VS Code, this runs when the IDE attaches. For CLI tool, should it run:
   - Every time `packnplay run` is called? (current plan)
   - Only on explicit `packnplay attach`?
   - Never? (document as intentional gap)

3. **Test isolation** - Should each test clean up images it builds, or leave them for caching?
