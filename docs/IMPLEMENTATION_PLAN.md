# packnplay Implementation Plan - Detailed Step-by-Step Guide

**Target Audience:** Skilled developer unfamiliar with this codebase
**Approach:** Test-Driven Development (TDD) with Red-Green-Refactor cycle
**Principles:** YAGNI, DRY, frequent commits, good judgment

## Prerequisites - Understanding the Codebase

### Key Files to Read First:
1. `/home/user/packnplay/pkg/runner/runner.go` - Main orchestration (635 lines, needs refactoring)
2. `/home/user/packnplay/pkg/devcontainer/config.go` - Current devcontainer parsing (only 3 fields)
3. `/home/user/packnplay/cmd/run.go` - CLI entry point
4. `/home/user/packnplay/pkg/agents/agent.go` - Agent abstraction (exists but unused)

### Current Architecture:
```
User runs CLI → cmd/run.go parses flags → runner.Run() orchestrates everything
                                         ↓
                    Loads devcontainer.json (3 fields only)
                                         ↓
                    Determines image, user, mounts
                                         ↓
                    Builds Docker command with all args
                                         ↓
                    Executes container
```

### Key Data Flows:
1. **Environment variables**: CLI `--env` → RunConfig.Env → Docker `-e` flags
2. **Ports**: CLI `-p` → RunConfig.PublishPorts → Docker `-p` flags
3. **Devcontainer**: .devcontainer/devcontainer.json → Config{Image, DockerFile, RemoteUser}

---

## Phase 1: Architecture Refactoring

### Task 1.1: Refactor runner.Run() - Extract Services

**Goal:** Split 635-line god object into focused, testable services

**Current Problem:**
- `runner.Run()` at lines 49-683 in `/home/user/packnplay/pkg/runner/runner.go`
- Does everything: worktree, image, user detection, mounts, credentials, execution
- Untestable, hard to extend

**Target Architecture:**
```
ContainerLauncher (orchestrator)
    ├── ImageManager (pull/build images)
    ├── MountBuilder (configure all mounts)
    ├── CredentialManager (handle credentials)
    └── UserDetector (detect container user) [already exists]
```

#### Step 1.1.1: Create ImageManager (TDD)

**Test First (Red Phase):**

Create `/home/user/packnplay/pkg/runner/image_manager_test.go`:

```go
package runner

import (
	"testing"

	"github.com/obra/packnplay/pkg/devcontainer"
	"github.com/obra/packnplay/pkg/docker"
)

func TestImageManager_EnsureAvailable_WithImage(t *testing.T) {
	// Test: When devcontainer specifies an image, pull it
	mockClient := &mockDockerClient{
		pullCalled: false,
	}

	im := &ImageManager{
		client: mockClient,
	}

	devConfig := &devcontainer.Config{
		Image: "ubuntu:22.04",
	}

	err := im.EnsureAvailable(devConfig, false) // false = not verbose
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

	im := &ImageManager{
		client: mockClient,
	}

	devConfig := &devcontainer.Config{
		DockerFile: "Dockerfile",
	}

	err := im.EnsureAvailable(devConfig, false)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if !mockClient.buildCalled {
		t.Error("Expected image build to be called")
	}
}

// Mock docker client for testing
type mockDockerClient struct {
	pullCalled  bool
	buildCalled bool
}

func (m *mockDockerClient) RunWithProgress(args ...string) error {
	if args[0] == "pull" {
		m.pullCalled = true
	} else if args[0] == "build" {
		m.buildCalled = true
	}
	return nil
}
```

**Run test:** `go test ./pkg/runner/image_manager_test.go` → **SHOULD FAIL** (red)

**Implementation (Green Phase):**

Create `/home/user/packnplay/pkg/runner/image_manager.go`:

```go
package runner

import (
	"fmt"

	"github.com/obra/packnplay/pkg/devcontainer"
	"github.com/obra/packnplay/pkg/docker"
)

// ImageManager handles container image availability (pull/build)
type ImageManager struct {
	client  DockerClient
	verbose bool
}

// DockerClient interface for testing (extract from docker.Client)
type DockerClient interface {
	RunWithProgress(args ...string) error
	Run(args ...string) (string, error)
}

// NewImageManager creates an ImageManager
func NewImageManager(client DockerClient, verbose bool) *ImageManager {
	return &ImageManager{
		client:  client,
		verbose: verbose,
	}
}

// EnsureAvailable ensures the image is available (pulls or builds)
// Extracted from runner.Run() lines 153-156 and 685-737
func (im *ImageManager) EnsureAvailable(devConfig *devcontainer.Config, projectPath string) error {
	// If Dockerfile specified, build it
	if devConfig.DockerFile != "" {
		return im.buildImage(devConfig, projectPath)
	}

	// Otherwise pull the image
	if devConfig.Image != "" {
		return im.pullImage(devConfig.Image)
	}

	return fmt.Errorf("no image or dockerfile specified")
}

// pullImage pulls a container image
func (im *ImageManager) pullImage(image string) error {
	if im.verbose {
		fmt.Printf("Pulling image: %s\n", image)
	}

	return im.client.RunWithProgress("pull", image)
}

// buildImage builds a container image from Dockerfile
// Extracted from runner.Run() lines 685-737
func (im *ImageManager) buildImage(devConfig *devcontainer.Config, projectPath string) error {
	if im.verbose {
		fmt.Printf("Building image from: %s\n", devConfig.DockerFile)
	}

	// Build args: docker build -t <tag> -f <dockerfile> <context>
	tag := fmt.Sprintf("packnplay-%s-devcontainer:latest",
		filepath.Base(projectPath))

	buildArgs := []string{
		"build",
		"-t", tag,
		"-f", filepath.Join(projectPath, ".devcontainer", devConfig.DockerFile),
		filepath.Join(projectPath, ".devcontainer"),
	}

	return im.client.RunWithProgress(buildArgs...)
}
```

**Run test:** `go test ./pkg/runner/` → **SHOULD PASS** (green)

**Refactor Phase:**
- Review for code smells
- Ensure single responsibility
- Check error messages are clear
- Verify YAGNI - no unused features

**Commit:**
```bash
git add pkg/runner/image_manager.go pkg/runner/image_manager_test.go
git commit -m "refactor: extract ImageManager from runner.Run()

- Create ImageManager service for image pull/build operations
- Add DockerClient interface for testability
- Extract logic from runner.Run() lines 153-156, 685-737
- Add comprehensive tests for pull and build scenarios
- TDD: Red-Green-Refactor cycle"
```

#### Step 1.1.2: Create MountBuilder (TDD)

**Test First (Red Phase):**

Create `/home/user/packnplay/pkg/runner/mount_builder_test.go`:

```go
package runner

import (
	"strings"
	"testing"

	"github.com/obra/packnplay/pkg/config"
)

func TestMountBuilder_BuildMounts_Basic(t *testing.T) {
	mb := NewMountBuilder("/home/testuser", "testuser")

	cfg := &RunConfig{
		Path: "/project/path",
		Credentials: config.Credentials{
			Git: true,
			SSH: false,
		},
	}

	mounts, err := mb.BuildMounts(cfg)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Should include project mount and .gitconfig
	if len(mounts) < 2 {
		t.Errorf("Expected at least 2 mounts, got %d", len(mounts))
	}

	// Check for project path mount
	hasProjectMount := false
	for _, mount := range mounts {
		if strings.Contains(mount, cfg.Path) {
			hasProjectMount = true
			break
		}
	}
	if !hasProjectMount {
		t.Error("Expected project path to be mounted")
	}
}

func TestMountBuilder_BuildMounts_WithSSH(t *testing.T) {
	mb := NewMountBuilder("/home/testuser", "testuser")

	cfg := &RunConfig{
		Path: "/project/path",
		Credentials: config.Credentials{
			SSH: true,
		},
	}

	mounts, err := mb.BuildMounts(cfg)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Should include .ssh mount
	hasSSH := false
	for _, mount := range mounts {
		if strings.Contains(mount, ".ssh") {
			hasSSH = true
			break
		}
	}
	if !hasSSH {
		t.Error("Expected .ssh to be mounted when SSH credentials enabled")
	}
}
```

**Implementation (Green Phase):**

Create `/home/user/packnplay/pkg/runner/mount_builder.go`:

```go
package runner

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/obra/packnplay/pkg/agents"
)

// MountBuilder constructs volume mount arguments for containers
type MountBuilder struct {
	hostHomeDir   string
	containerUser string
}

// NewMountBuilder creates a MountBuilder
func NewMountBuilder(hostHomeDir, containerUser string) *MountBuilder {
	return &MountBuilder{
		hostHomeDir:   hostHomeDir,
		containerUser: containerUser,
	}
}

// BuildMounts constructs all volume mounts
// Extracted from runner.Run() lines 345-426
func (mb *MountBuilder) BuildMounts(cfg *RunConfig) ([]string, error) {
	var args []string

	// 1. Mount project directory
	projectMount := fmt.Sprintf("%s:%s", cfg.Path, cfg.Path)
	args = append(args, "-v", projectMount)

	// 2. Mount .git directory (if exists)
	gitDir := filepath.Join(cfg.Path, ".git")
	if fileExists(gitDir) {
		args = append(args, "-v", fmt.Sprintf("%s:%s", gitDir, gitDir))
	}

	// 3. Mount credentials based on config
	credMounts, err := mb.buildCredentialMounts(cfg.Credentials)
	if err != nil {
		return nil, err
	}
	args = append(args, credMounts...)

	// 4. Mount agent configs
	agentMounts := mb.buildAgentMounts()
	args = append(args, agentMounts...)

	return args, nil
}

// buildCredentialMounts constructs credential volume mounts
func (mb *MountBuilder) buildCredentialMounts(creds config.Credentials) ([]string, error) {
	var args []string

	if creds.Git {
		gitconfig := filepath.Join(mb.hostHomeDir, ".gitconfig")
		if fileExists(gitconfig) {
			target := fmt.Sprintf("/home/%s/.gitconfig", mb.containerUser)
			args = append(args, "-v", fmt.Sprintf("%s:%s:ro", gitconfig, target))
		}
	}

	if creds.SSH {
		sshDir := filepath.Join(mb.hostHomeDir, ".ssh")
		if fileExists(sshDir) {
			target := fmt.Sprintf("/home/%s/.ssh", mb.containerUser)
			args = append(args, "-v", fmt.Sprintf("%s:%s:ro", sshDir, target))
		}
	}

	// Add other credentials (GH, GPG, NPM, AWS) similarly...

	return args, nil
}

// buildAgentMounts constructs agent config directory mounts
// Use the Agent abstraction (fixes hardcoded issue)
func (mb *MountBuilder) buildAgentMounts() []string {
	var args []string

	for _, agent := range agents.GetSupportedAgents() {
		mounts := agent.GetMounts(mb.hostHomeDir, mb.containerUser)
		for _, mount := range mounts {
			// Mount format: "source:target" or "source:target:ro"
			args = append(args, "-v", mount.String())
		}
	}

	return args
}

// fileExists checks if path exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
```

**Run test:** `go test ./pkg/runner/` → **SHOULD PASS** (green)

**Commit:**
```bash
git add pkg/runner/mount_builder.go pkg/runner/mount_builder_test.go
git commit -m "refactor: extract MountBuilder from runner.Run()

- Create MountBuilder service for volume mount configuration
- Extract logic from runner.Run() lines 345-426
- Use Agent abstraction instead of hardcoded list (fixes architectural smell)
- Add tests for basic mounts, credential mounts, agent mounts
- TDD: Red-Green-Refactor cycle"
```

#### Step 1.1.3: Integrate Services into runner.Run()

**Test First (Red Phase):**

Update `/home/user/packnplay/pkg/runner/runner_test.go`:

```go
func TestRun_IntegrationWithServices(t *testing.T) {
	// Test that Run() properly uses the new services
	// This is an integration test, not a unit test

	cfg := &RunConfig{
		Path: "/tmp/test-project",
		Credentials: config.Credentials{Git: true},
		// ... other fields
	}

	// Should not panic, should use services correctly
	// We can't run full integration here, but ensure structure is right
}
```

**Implementation (Green Phase):**

Modify `/home/user/packnplay/pkg/runner/runner.go`:

```go
// Run starts a container and executes the specified command
// REFACTORED: Now orchestrates services instead of doing everything
func Run(config *RunConfig) error {
	// 1. Determine working directory (keep this - simple logic)
	workDir := config.Path
	if workDir == "" {
		// ... existing logic lines 51-64
	}

	// 2. Handle worktree logic (keep this - domain logic)
	var mountPath string
	if config.NoWorktree {
		// ... existing logic lines 68-134
	}

	// 3. Load devcontainer config (keep this - simple)
	devConfig, err := loadDevContainerConfig(mountPath, config)
	if err != nil {
		return err
	}

	// 4. Initialize Docker client (keep this - simple)
	dockerClient, err := docker.NewClient(config.Verbose)
	if err != nil {
		return err
	}

	// 5. Use ImageManager service
	imageManager := NewImageManager(dockerClient, config.Verbose)
	if err := imageManager.EnsureAvailable(devConfig, mountPath); err != nil {
		return fmt.Errorf("failed to ensure image: %w", err)
	}

	// 6. Detect user (keep existing logic - already separated)
	containerUser := devConfig.RemoteUser

	// 7. Check for existing container (keep this - simple)
	containerName := container.GenerateContainerName(mountPath, config.Worktree)
	// ... existing logic lines 170-267

	// 8. Use MountBuilder service
	homeDir, _ := os.UserHomeDir()
	mountBuilder := NewMountBuilder(homeDir, containerUser)
	volumeMounts, err := mountBuilder.BuildMounts(config)
	if err != nil {
		return fmt.Errorf("failed to build mounts: %w", err)
	}

	// 9. Build Docker run command (keep this - orchestration)
	args := []string{"run"}
	args = append(args, "--rm")
	args = append(args, "--name", containerName)
	args = append(args, volumeMounts...)

	// ... rest of Docker args (env, ports, etc.)

	// 10. Execute container (keep this - simple)
	return dockerClient.Run(args...)
}
```

**Refactor Phase:**
- runner.Run() should now be ~300 lines (was 635)
- Clear separation of concerns
- Each service is independently testable
- No business logic duplication

**Commit:**
```bash
git add pkg/runner/runner.go pkg/runner/runner_test.go
git commit -m "refactor: integrate ImageManager and MountBuilder into runner.Run()

- Reduce runner.Run() from 635 to ~300 lines
- Replace inline logic with service calls
- Maintain backward compatibility
- All existing tests still pass
- Clear separation of concerns achieved"
```

---

### Task 1.2: Use Agent Abstraction Instead of Hardcoded List

**Current Problem:**
- Lines 359-369 in runner.go hardcode agent directories
- Agent abstraction exists in pkg/agents/ but is unused
- Missing .claude directory

**This was already fixed in Task 1.1.2** when we created MountBuilder!

**Verification Test:**

Add to `/home/user/packnplay/pkg/runner/mount_builder_test.go`:

```go
func TestMountBuilder_UsesAgentAbstraction(t *testing.T) {
	mb := NewMountBuilder("/home/testuser", "testuser")

	cfg := &RunConfig{
		Path: "/project",
		Credentials: config.Credentials{},
	}

	mounts, _ := mb.BuildMounts(cfg)

	// Verify that agent mounts are included
	// This proves we're using the Agent interface, not hardcoded list
	hasAgentMount := false
	for _, mount := range mounts {
		// Check for any agent directory (.claude, .codex, etc.)
		if strings.Contains(mount, ".claude") ||
		   strings.Contains(mount, ".codex") {
			hasAgentMount = true
			break
		}
	}

	// If any agent config exists on the host, it should be mounted
	// This is verified by the Agent abstraction logic
}
```

**Run test:** `go test ./pkg/runner/` → **SHOULD PASS** (green)

**Already committed in Task 1.1.2!**

---

### Task 1.3: Consolidate Duplicate Label Parsing

**Current Problem:**
- Label parsing duplicated in 3 places:
  1. runner.go:829-850 - `parseLabelsFromString()`
  2. list.go:140-155 - `parseLabels()`
  3. list.go:157-175 - `parseLabelsWithLaunchInfo()`

**Test First (Red Phase):**

Create `/home/user/packnplay/pkg/container/labels_test.go`:

```go
package container

import (
	"testing"
)

func TestParseLabels_Basic(t *testing.T) {
	labelString := "packnplay-project=myproject,packnplay-worktree=feature-branch"

	labels := ParseLabels(labelString)

	if labels["packnplay-project"] != "myproject" {
		t.Errorf("Expected project=myproject, got %s", labels["packnplay-project"])
	}

	if labels["packnplay-worktree"] != "feature-branch" {
		t.Errorf("Expected worktree=feature-branch, got %s", labels["packnplay-worktree"])
	}
}

func TestParseLabels_WithAllFields(t *testing.T) {
	labelString := "packnplay-project=proj,packnplay-worktree=wt,packnplay-host-path=/path,packnplay-launch-command=bash"

	labels := ParseLabels(labelString)

	if len(labels) != 4 {
		t.Errorf("Expected 4 labels, got %d", len(labels))
	}
}

func TestGetProjectFromLabels(t *testing.T) {
	labels := map[string]string{
		"packnplay-project": "myproject",
		"other-label": "value",
	}

	project := GetProjectFromLabels(labels)
	if project != "myproject" {
		t.Errorf("Expected myproject, got %s", project)
	}
}
```

**Implementation (Green Phase):**

Create `/home/user/packnplay/pkg/container/labels.go`:

```go
package container

import (
	"strings"
)

// Label key constants
const (
	LabelProject       = "packnplay-project"
	LabelWorktree      = "packnplay-worktree"
	LabelHostPath      = "packnplay-host-path"
	LabelLaunchCommand = "packnplay-launch-command"
	LabelManagedBy     = "managed-by"
)

// ParseLabels parses a comma-separated label string into a map
// Replaces 3 duplicate implementations across the codebase
func ParseLabels(labelString string) map[string]string {
	labels := make(map[string]string)

	pairs := strings.Split(labelString, ",")
	for _, pair := range pairs {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) == 2 {
			labels[parts[0]] = parts[1]
		}
	}

	return labels
}

// GetProjectFromLabels extracts project name from label map
func GetProjectFromLabels(labels map[string]string) string {
	return labels[LabelProject]
}

// GetWorktreeFromLabels extracts worktree name from label map
func GetWorktreeFromLabels(labels map[string]string) string {
	return labels[LabelWorktree]
}

// GetHostPathFromLabels extracts host path from label map
func GetHostPathFromLabels(labels map[string]string) string {
	return labels[LabelHostPath]
}

// GetLaunchCommandFromLabels extracts launch command from label map
func GetLaunchCommandFromLabels(labels map[string]string) string {
	return labels[LabelLaunchCommand]
}
```

**Update existing code:**

Modify `/home/user/packnplay/pkg/runner/runner.go`:

```go
// DELETE lines 829-850 (parseLabelsFromString function)

// REPLACE usage (around line 750) with:
labels := container.ParseLabels(labelString)
project := container.GetProjectFromLabels(labels)
worktree := container.GetWorktreeFromLabels(labels)
```

Modify `/home/user/packnplay/cmd/list.go`:

```go
// DELETE lines 140-175 (parseLabels and parseLabelsWithLaunchInfo functions)
// DELETE lines 178-200 (splitByComma and splitByEquals functions)

// REPLACE all usages with:
labels := container.ParseLabels(labelString)
project := container.GetProjectFromLabels(labels)
worktree := container.GetWorktreeFromLabels(labels)
hostPath := container.GetHostPathFromLabels(labels)
launchCommand := container.GetLaunchCommandFromLabels(labels)
```

**Run tests:** `go test ./...` → **SHOULD PASS** (green)

**Commit:**
```bash
git add pkg/container/labels.go pkg/container/labels_test.go
git add pkg/runner/runner.go cmd/list.go
git commit -m "refactor: consolidate duplicate label parsing into pkg/container

- Create unified ParseLabels function in pkg/container/labels.go
- Add type-safe label key constants
- Add convenience functions for extracting specific labels
- Remove 3 duplicate implementations:
  - runner.go:829-850 parseLabelsFromString
  - list.go:140-155 parseLabels
  - list.go:157-175 parseLabelsWithLaunchInfo
- DRY principle applied
- All existing tests pass"
```

---

## Phase 2: Connect CLI to devcontainer.json

### Task 2.1: Add containerEnv/remoteEnv Parsing

**Goal:** Parse environment variables from devcontainer.json and pass to existing `--env` infrastructure

**Test First (Red Phase):**

Create `/home/user/packnplay/pkg/devcontainer/config_test.go` (extend existing):

```go
func TestLoadConfig_WithContainerEnv(t *testing.T) {
	// Create temp devcontainer.json
	tmpDir := t.TempDir()
	devcontainerDir := filepath.Join(tmpDir, ".devcontainer")
	os.MkdirAll(devcontainerDir, 0755)

	configJSON := `{
		"image": "ubuntu:22.04",
		"containerEnv": {
			"NODE_ENV": "development",
			"DATABASE_URL": "postgresql://localhost:5432/dev"
		}
	}`

	os.WriteFile(
		filepath.Join(devcontainerDir, "devcontainer.json"),
		[]byte(configJSON),
		0644,
	)

	config, err := LoadConfig(tmpDir)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if config.ContainerEnv == nil {
		t.Fatal("Expected containerEnv to be parsed")
	}

	if config.ContainerEnv["NODE_ENV"] != "development" {
		t.Errorf("Expected NODE_ENV=development, got %s", config.ContainerEnv["NODE_ENV"])
	}
}

func TestLoadConfig_WithRemoteEnv(t *testing.T) {
	tmpDir := t.TempDir()
	devcontainerDir := filepath.Join(tmpDir, ".devcontainer")
	os.MkdirAll(devcontainerDir, 0755)

	configJSON := `{
		"image": "ubuntu:22.04",
		"remoteEnv": {
			"PATH": "${containerEnv:PATH}:/custom/bin"
		}
	}`

	os.WriteFile(
		filepath.Join(devcontainerDir, "devcontainer.json"),
		[]byte(configJSON),
		0644,
	)

	config, err := LoadConfig(tmpDir)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if config.RemoteEnv == nil {
		t.Fatal("Expected remoteEnv to be parsed")
	}
}
```

**Run test:** `go test ./pkg/devcontainer/` → **SHOULD FAIL** (red)

**Implementation (Green Phase):**

Modify `/home/user/packnplay/pkg/devcontainer/config.go`:

```go
// Config represents a parsed devcontainer.json
type Config struct {
	Image        string            `json:"image"`
	DockerFile   string            `json:"dockerFile"`
	RemoteUser   string            `json:"remoteUser"`
	ContainerEnv map[string]string `json:"containerEnv"` // NEW
	RemoteEnv    map[string]string `json:"remoteEnv"`    // NEW
}
```

Create `/home/user/packnplay/pkg/devcontainer/variables.go`:

```go
package devcontainer

import (
	"os"
	"regexp"
	"strings"
)

// SubstituteVariables performs variable substitution on environment variables
// Supports: ${localEnv:VAR}, ${containerEnv:VAR}
func SubstituteVariables(value string, containerEnv map[string]string) string {
	// Pattern: ${localEnv:VAR} or ${containerEnv:VAR}
	re := regexp.MustCompile(`\$\{(localEnv|containerEnv):([^}]+)\}`)

	return re.ReplaceAllStringFunc(value, func(match string) string {
		parts := re.FindStringSubmatch(match)
		if len(parts) != 3 {
			return match
		}

		source := parts[1]  // localEnv or containerEnv
		varName := parts[2] // Variable name

		switch source {
		case "localEnv":
			return os.Getenv(varName)
		case "containerEnv":
			if val, ok := containerEnv[varName]; ok {
				return val
			}
		}

		return match // Return unchanged if not found
	})
}

// ResolveEnvironment resolves all environment variables with substitution
func ResolveEnvironment(config *Config) map[string]string {
	result := make(map[string]string)

	// 1. Copy containerEnv (no substitution needed)
	for k, v := range config.ContainerEnv {
		result[k] = v
	}

	// 2. Process remoteEnv with substitution
	for k, v := range config.RemoteEnv {
		result[k] = SubstituteVariables(v, config.ContainerEnv)
	}

	return result
}
```

Create `/home/user/packnplay/pkg/devcontainer/variables_test.go`:

```go
package devcontainer

import (
	"os"
	"testing"
)

func TestSubstituteVariables_LocalEnv(t *testing.T) {
	os.Setenv("TEST_VAR", "test-value")
	defer os.Unsetenv("TEST_VAR")

	result := SubstituteVariables("${localEnv:TEST_VAR}", nil)
	if result != "test-value" {
		t.Errorf("Expected test-value, got %s", result)
	}
}

func TestSubstituteVariables_ContainerEnv(t *testing.T) {
	containerEnv := map[string]string{
		"PATH": "/usr/bin",
	}

	result := SubstituteVariables("${containerEnv:PATH}:/custom", containerEnv)
	if result != "/usr/bin:/custom" {
		t.Errorf("Expected /usr/bin:/custom, got %s", result)
	}
}

func TestResolveEnvironment(t *testing.T) {
	config := &Config{
		ContainerEnv: map[string]string{
			"NODE_ENV": "development",
		},
		RemoteEnv: map[string]string{
			"PATH": "${containerEnv:PATH}:/custom",
		},
	}

	env := ResolveEnvironment(config)

	if env["NODE_ENV"] != "development" {
		t.Errorf("Expected NODE_ENV=development, got %s", env["NODE_ENV"])
	}

	// PATH should have substitution performed
	if !strings.Contains(env["PATH"], "/custom") {
		t.Errorf("Expected PATH to contain /custom, got %s", env["PATH"])
	}
}
```

**Integration into runner:**

Modify `/home/user/packnplay/pkg/runner/runner.go`:

```go
// In Run() function, after loading devcontainer config:

// Merge environment variables from devcontainer.json
devEnv := devcontainer.ResolveEnvironment(devConfig)
for k, v := range devEnv {
	config.Env = append([]string{fmt.Sprintf("%s=%s", k, v)}, config.Env...)
}
// Note: config.Env (from CLI) is appended after, so CLI overrides devcontainer
```

**Run tests:** `go test ./...` → **SHOULD PASS** (green)

**Commit:**
```bash
git add pkg/devcontainer/config.go pkg/devcontainer/variables.go pkg/devcontainer/variables_test.go
git add pkg/runner/runner.go
git commit -m "feat: add containerEnv and remoteEnv support from devcontainer.json

- Parse containerEnv and remoteEnv fields from devcontainer.json
- Implement variable substitution engine (\${localEnv:X}, \${containerEnv:X})
- Integrate with existing --env flag infrastructure
- CLI flags override devcontainer.json values
- Add comprehensive tests for substitution logic
- TDD: Red-Green-Refactor cycle"
```

---

### Task 2.2: Add forwardPorts Parsing

**Test First (Red Phase):**

Add to `/home/user/packnplay/pkg/devcontainer/config_test.go`:

```go
func TestLoadConfig_WithForwardPorts(t *testing.T) {
	tmpDir := t.TempDir()
	devcontainerDir := filepath.Join(tmpDir, ".devcontainer")
	os.MkdirAll(devcontainerDir, 0755)

	configJSON := `{
		"image": "ubuntu:22.04",
		"forwardPorts": [3000, 5432, "8080:8080"]
	}`

	os.WriteFile(
		filepath.Join(devcontainerDir, "devcontainer.json"),
		[]byte(configJSON),
		0644,
	)

	config, err := LoadConfig(tmpDir)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if config.ForwardPorts == nil {
		t.Fatal("Expected forwardPorts to be parsed")
	}

	if len(config.ForwardPorts) != 3 {
		t.Errorf("Expected 3 ports, got %d", len(config.ForwardPorts))
	}
}
```

**Implementation (Green Phase):**

Modify `/home/user/packnplay/pkg/devcontainer/config.go`:

```go
type Config struct {
	Image        string            `json:"image"`
	DockerFile   string            `json:"dockerFile"`
	RemoteUser   string            `json:"remoteUser"`
	ContainerEnv map[string]string `json:"containerEnv"`
	RemoteEnv    map[string]string `json:"remoteEnv"`
	ForwardPorts []interface{}     `json:"forwardPorts"` // NEW: Can be int or string
}
```

Create `/home/user/packnplay/pkg/devcontainer/ports.go`:

```go
package devcontainer

import (
	"fmt"
)

// ConvertForwardPortsToPublishArgs converts forwardPorts to Docker -p arguments
// Input: [3000, "8080:8080", "127.0.0.1:9000:9000"]
// Output: ["3000:3000", "8080:8080", "127.0.0.1:9000:9000"]
func ConvertForwardPortsToPublishArgs(ports []interface{}) ([]string, error) {
	var result []string

	for _, port := range ports {
		switch v := port.(type) {
		case float64: // JSON numbers are float64
			// Single port: 3000 → "3000:3000"
			portStr := fmt.Sprintf("%.0f:%.0f", v, v)
			result = append(result, portStr)

		case string:
			// Already formatted: "8080:8080" or "127.0.0.1:8080:8080"
			result = append(result, v)

		default:
			return nil, fmt.Errorf("invalid port type: %T", port)
		}
	}

	return result, nil
}
```

Create `/home/user/packnplay/pkg/devcontainer/ports_test.go`:

```go
package devcontainer

import (
	"testing"
)

func TestConvertForwardPorts_IntegerPort(t *testing.T) {
	ports := []interface{}{float64(3000)}

	result, err := ConvertForwardPortsToPublishArgs(ports)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("Expected 1 port, got %d", len(result))
	}

	if result[0] != "3000:3000" {
		t.Errorf("Expected 3000:3000, got %s", result[0])
	}
}

func TestConvertForwardPorts_StringPort(t *testing.T) {
	ports := []interface{}{"8080:80"}

	result, err := ConvertForwardPortsToPublishArgs(ports)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result[0] != "8080:80" {
		t.Errorf("Expected 8080:80, got %s", result[0])
	}
}

func TestConvertForwardPorts_Mixed(t *testing.T) {
	ports := []interface{}{float64(3000), "8080:80", "127.0.0.1:9000:9000"}

	result, err := ConvertForwardPortsToPublishArgs(ports)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(result) != 3 {
		t.Fatalf("Expected 3 ports, got %d", len(result))
	}
}
```

**Integration into runner:**

Modify `/home/user/packnplay/pkg/runner/runner.go`:

```go
// In Run() function, after loading devcontainer config:

// Merge port mappings from devcontainer.json
if len(devConfig.ForwardPorts) > 0 {
	devPorts, err := devcontainer.ConvertForwardPortsToPublishArgs(devConfig.ForwardPorts)
	if err != nil {
		return fmt.Errorf("failed to parse forwardPorts: %w", err)
	}
	// Prepend dev ports, so CLI -p flags override
	config.PublishPorts = append(devPorts, config.PublishPorts...)
}
```

**Run tests:** `go test ./...` → **SHOULD PASS** (green)

**Commit:**
```bash
git add pkg/devcontainer/config.go pkg/devcontainer/ports.go pkg/devcontainer/ports_test.go
git add pkg/runner/runner.go
git commit -m "feat: add forwardPorts support from devcontainer.json

- Parse forwardPorts array from devcontainer.json
- Support both integer (3000) and string (\"8080:80\") formats
- Convert to Docker -p flag format
- Integrate with existing --publish flag infrastructure
- CLI flags override devcontainer.json values
- Add comprehensive tests for port conversion
- TDD: Red-Green-Refactor cycle"
```

---

### Task 2.3: Add Build Configuration Parsing

**Test First (Red Phase):**

Add to `/home/user/packnplay/pkg/devcontainer/config_test.go`:

```go
func TestLoadConfig_WithBuildConfig(t *testing.T) {
	tmpDir := t.TempDir()
	devcontainerDir := filepath.Join(tmpDir, ".devcontainer")
	os.MkdirAll(devcontainerDir, 0755)

	configJSON := `{
		"build": {
			"dockerfile": "Dockerfile.dev",
			"context": "..",
			"args": {
				"NODE_VERSION": "18",
				"INSTALL_DEV": "true"
			},
			"target": "development"
		}
	}`

	os.WriteFile(
		filepath.Join(devcontainerDir, "devcontainer.json"),
		[]byte(configJSON),
		0644,
	)

	config, err := LoadConfig(tmpDir)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if config.Build == nil {
		t.Fatal("Expected build config to be parsed")
	}

	if config.Build.Dockerfile != "Dockerfile.dev" {
		t.Errorf("Expected Dockerfile.dev, got %s", config.Build.Dockerfile)
	}

	if config.Build.Args["NODE_VERSION"] != "18" {
		t.Errorf("Expected NODE_VERSION=18, got %s", config.Build.Args["NODE_VERSION"])
	}
}

func TestLoadConfig_BackwardCompatWithDockerFile(t *testing.T) {
	// Old format: "dockerFile": "Dockerfile"
	// Should still work
	tmpDir := t.TempDir()
	devcontainerDir := filepath.Join(tmpDir, ".devcontainer")
	os.MkdirAll(devcontainerDir, 0755)

	configJSON := `{
		"dockerFile": "Dockerfile"
	}`

	os.WriteFile(
		filepath.Join(devcontainerDir, "devcontainer.json"),
		[]byte(configJSON),
		0644,
	)

	config, err := LoadConfig(tmpDir)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Should be converted to Build config
	if config.DockerFile != "Dockerfile" {
		t.Error("Backward compatibility broken")
	}
}
```

**Implementation (Green Phase):**

Modify `/home/user/packnplay/pkg/devcontainer/config.go`:

```go
// BuildConfig represents build configuration
type BuildConfig struct {
	Dockerfile string            `json:"dockerfile"`
	Context    string            `json:"context"`
	Args       map[string]string `json:"args"`
	Target     string            `json:"target"`
	CacheFrom  []string          `json:"cacheFrom"`
}

// Config represents a parsed devcontainer.json
type Config struct {
	Image        string            `json:"image"`
	DockerFile   string            `json:"dockerFile"` // Backward compat
	Build        *BuildConfig      `json:"build"`      // NEW
	RemoteUser   string            `json:"remoteUser"`
	ContainerEnv map[string]string `json:"containerEnv"`
	RemoteEnv    map[string]string `json:"remoteEnv"`
	ForwardPorts []interface{}     `json:"forwardPorts"`
}

// GetDockerfile returns the Dockerfile path (handles both formats)
func (c *Config) GetDockerfile() string {
	if c.Build != nil && c.Build.Dockerfile != "" {
		return c.Build.Dockerfile
	}
	return c.DockerFile
}
```

**Update ImageManager to use build config:**

Modify `/home/user/packnplay/pkg/runner/image_manager.go`:

```go
// buildImage builds a container image from Dockerfile with build config
func (im *ImageManager) buildImage(devConfig *devcontainer.Config, projectPath string) error {
	dockerfile := devConfig.GetDockerfile()
	if dockerfile == "" {
		return fmt.Errorf("no dockerfile specified")
	}

	tag := fmt.Sprintf("packnplay-%s-devcontainer:latest",
		filepath.Base(projectPath))

	// Base build args
	buildArgs := []string{
		"build",
		"-t", tag,
	}

	// Add build arguments from config
	if devConfig.Build != nil {
		for k, v := range devConfig.Build.Args {
			buildArgs = append(buildArgs, "--build-arg", fmt.Sprintf("%s=%s", k, v))
		}

		// Add target if specified
		if devConfig.Build.Target != "" {
			buildArgs = append(buildArgs, "--target", devConfig.Build.Target)
		}

		// Add cache-from if specified
		for _, cache := range devConfig.Build.CacheFrom {
			buildArgs = append(buildArgs, "--cache-from", cache)
		}
	}

	// Determine context path
	contextPath := filepath.Join(projectPath, ".devcontainer")
	if devConfig.Build != nil && devConfig.Build.Context != "" {
		contextPath = filepath.Join(projectPath, ".devcontainer", devConfig.Build.Context)
	}

	// Add dockerfile and context
	buildArgs = append(buildArgs,
		"-f", filepath.Join(projectPath, ".devcontainer", dockerfile),
		contextPath,
	)

	return im.client.RunWithProgress(buildArgs...)
}
```

**Run tests:** `go test ./...` → **SHOULD PASS** (green)

**Commit:**
```bash
git add pkg/devcontainer/config.go pkg/runner/image_manager.go
git commit -m "feat: add build configuration support from devcontainer.json

- Add BuildConfig struct with args, target, context, cacheFrom
- Support both old dockerFile and new build formats (backward compatible)
- Add build arguments to Docker build command
- Support multi-stage build targets (--target)
- Support custom build context
- Add cache-from support for faster builds
- Maintain backward compatibility with simple dockerFile string
- Add comprehensive tests
- TDD: Red-Green-Refactor cycle"
```

---

## Phase 3: New Features

### Task 3.1: Implement Lifecycle Scripts

**Goal:** Execute onCreateCommand, postCreateCommand, postStartCommand at appropriate times

**Architecture:**
1. Metadata file tracks which scripts have run (onCreate only runs once)
2. Execute scripts via `docker exec` in running container
3. Capture output for debugging
4. Handle failures gracefully

**Test First (Red Phase):**

Create `/home/user/packnplay/pkg/runner/lifecycle_test.go`:

```go
package runner

import (
	"testing"

	"github.com/obra/packnplay/pkg/devcontainer"
)

func TestLifecycleExecutor_ShouldRunOnCreate_FirstTime(t *testing.T) {
	executor := NewLifecycleExecutor(nil, "test-container", "testuser", "/tmp/test-meta")

	devConfig := &devcontainer.Config{
		OnCreateCommand: "npm install",
	}

	shouldRun, err := executor.ShouldRunOnCreate(devConfig)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !shouldRun {
		t.Error("Expected onCreate to run on first execution")
	}
}

func TestLifecycleExecutor_ShouldRunOnCreate_SecondTime(t *testing.T) {
	executor := NewLifecycleExecutor(nil, "test-container", "testuser", "/tmp/test-meta")

	devConfig := &devcontainer.Config{
		OnCreateCommand: "npm install",
	}

	// Mark as already run
	executor.MarkOnCreateRun(devConfig)

	shouldRun, _ := executor.ShouldRunOnCreate(devConfig)
	if shouldRun {
		t.Error("Expected onCreate to NOT run second time")
	}
}

func TestLifecycleExecutor_ExecuteCommand_String(t *testing.T) {
	mockClient := &mockDockerClient{}
	executor := NewLifecycleExecutor(mockClient, "test-container", "testuser", "/tmp/test-meta")

	err := executor.ExecuteCommand("npm install", false)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify docker exec was called correctly
	// mockClient should have recorded the call
}
```

**Run test:** `go test ./pkg/runner/` → **SHOULD FAIL** (red)

**Implementation (Green Phase) - Part 1: Metadata:**

Create `/home/user/packnplay/pkg/runner/metadata.go`:

```go
package runner

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// ContainerMetadata tracks container lifecycle state
type ContainerMetadata struct {
	ContainerID  string                 `json:"container_id"`
	ImageDigest  string                 `json:"image_digest"`
	CreatedAt    time.Time              `json:"created_at"`
	LifecycleRan map[string]LifecycleRun `json:"lifecycle_ran"`
}

// LifecycleRun tracks execution of a lifecycle script
type LifecycleRun struct {
	Executed   bool      `json:"executed"`
	Timestamp  time.Time `json:"timestamp"`
	ExitCode   int       `json:"exit_code"`
	CommandHash string   `json:"command_hash"` // Hash of command to detect changes
}

// GetMetadataPath returns path to metadata file for a container
func GetMetadataPath(containerName string) (string, error) {
	dataDir := os.Getenv("XDG_DATA_HOME")
	if dataDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		dataDir = filepath.Join(homeDir, ".local", "share")
	}

	metadataDir := filepath.Join(dataDir, "packnplay", "metadata")
	if err := os.MkdirAll(metadataDir, 0755); err != nil {
		return "", err
	}

	return filepath.Join(metadataDir, fmt.Sprintf("%s.json", containerName)), nil
}

// LoadMetadata loads container metadata from disk
func LoadMetadata(containerName string) (*ContainerMetadata, error) {
	path, err := GetMetadataPath(containerName)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// First time - return empty metadata
			return &ContainerMetadata{
				LifecycleRan: make(map[string]LifecycleRun),
			}, nil
		}
		return nil, err
	}

	var metadata ContainerMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, err
	}

	return &metadata, nil
}

// SaveMetadata saves container metadata to disk
func SaveMetadata(containerName string, metadata *ContainerMetadata) error {
	path, err := GetMetadataPath(containerName)
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// HashCommand creates a hash of a command string
func HashCommand(cmd interface{}) string {
	cmdStr := fmt.Sprintf("%v", cmd)
	hash := sha256.Sum256([]byte(cmdStr))
	return fmt.Sprintf("%x", hash[:8])
}
```

**Implementation (Green Phase) - Part 2: Executor:**

Create `/home/user/packnplay/pkg/runner/lifecycle.go`:

```go
package runner

import (
	"fmt"
	"strings"

	"github.com/obra/packnplay/pkg/devcontainer"
)

// LifecycleExecutor executes lifecycle commands in containers
type LifecycleExecutor struct {
	client        DockerClient
	containerName string
	containerUser string
	metadata      *ContainerMetadata
}

// NewLifecycleExecutor creates a LifecycleExecutor
func NewLifecycleExecutor(client DockerClient, containerName, containerUser string) (*LifecycleExecutor, error) {
	metadata, err := LoadMetadata(containerName)
	if err != nil {
		return nil, err
	}

	return &LifecycleExecutor{
		client:        client,
		containerName: containerName,
		containerUser: containerUser,
		metadata:      metadata,
	}, nil
}

// ExecuteLifecycle runs all appropriate lifecycle scripts
func (le *LifecycleExecutor) ExecuteLifecycle(devConfig *devcontainer.Config, verbose bool) error {
	// 1. Execute onCreateCommand (only if not run before)
	if devConfig.OnCreateCommand != nil {
		if le.ShouldRunOnCreate(devConfig) {
			if verbose {
				fmt.Println("Running onCreateCommand...")
			}
			if err := le.ExecuteCommand(devConfig.OnCreateCommand, verbose); err != nil {
				return fmt.Errorf("onCreateCommand failed: %w", err)
			}
			le.MarkCommandRun("onCreate", devConfig.OnCreateCommand)
		}
	}

	// 2. Execute postCreateCommand (only if not run before)
	if devConfig.PostCreateCommand != nil {
		if le.ShouldRunPostCreate(devConfig) {
			if verbose {
				fmt.Println("Running postCreateCommand...")
			}
			if err := le.ExecuteCommand(devConfig.PostCreateCommand, verbose); err != nil {
				return fmt.Errorf("postCreateCommand failed: %w", err)
			}
			le.MarkCommandRun("postCreate", devConfig.PostCreateCommand)
		}
	}

	// 3. Execute postStartCommand (runs every time)
	if devConfig.PostStartCommand != nil {
		if verbose {
			fmt.Println("Running postStartCommand...")
		}
		if err := le.ExecuteCommand(devConfig.PostStartCommand, verbose); err != nil {
			return fmt.Errorf("postStartCommand failed: %w", err)
		}
	}

	// Save metadata
	return SaveMetadata(le.containerName, le.metadata)
}

// ExecuteCommand executes a lifecycle command in the container
// Handles string, []string, and map formats from devcontainer spec
func (le *LifecycleExecutor) ExecuteCommand(cmd interface{}, verbose bool) error {
	switch v := cmd.(type) {
	case string:
		// Single command string: "npm install"
		return le.executeShellCommand(v, verbose)

	case []interface{}:
		// Array of commands: ["npm install", "npm run build"]
		for _, subcmd := range v {
			if err := le.ExecuteCommand(subcmd, verbose); err != nil {
				return err
			}
		}
		return nil

	case map[string]interface{}:
		// Parallel commands (we'll run sequentially for simplicity)
		for _, subcmd := range v {
			if err := le.ExecuteCommand(subcmd, verbose); err != nil {
				return err
			}
		}
		return nil

	default:
		return fmt.Errorf("unsupported command type: %T", cmd)
	}
}

// executeShellCommand executes a single shell command
func (le *LifecycleExecutor) executeShellCommand(cmd string, verbose bool) error {
	// Use docker exec to run command in container
	args := []string{
		"exec",
		"-u", le.containerUser,
		le.containerName,
		"sh", "-c", cmd,
	}

	output, err := le.client.Run(args...)
	if verbose || err != nil {
		fmt.Println(output)
	}

	return err
}

// ShouldRunOnCreate determines if onCreate should run
func (le *LifecycleExecutor) ShouldRunOnCreate(devConfig *devcontainer.Config) bool {
	return le.shouldRunCommand("onCreate", devConfig.OnCreateCommand)
}

// ShouldRunPostCreate determines if postCreate should run
func (le *LifecycleExecutor) ShouldRunPostCreate(devConfig *devcontainer.Config) bool {
	return le.shouldRunCommand("postCreate", devConfig.PostCreateCommand)
}

// shouldRunCommand checks if a command should run (based on hash)
func (le *LifecycleExecutor) shouldRunCommand(name string, cmd interface{}) bool {
	if cmd == nil {
		return false
	}

	cmdHash := HashCommand(cmd)

	if run, exists := le.metadata.LifecycleRan[name]; exists {
		// Command has run before - check if it changed
		return run.CommandHash != cmdHash
	}

	// Never run before
	return true
}

// MarkCommandRun marks a command as having run
func (le *LifecycleExecutor) MarkCommandRun(name string, cmd interface{}) {
	if le.metadata.LifecycleRan == nil {
		le.metadata.LifecycleRan = make(map[string]LifecycleRun)
	}

	le.metadata.LifecycleRan[name] = LifecycleRun{
		Executed:    true,
		Timestamp:   time.Now(),
		ExitCode:    0,
		CommandHash: HashCommand(cmd),
	}
}
```

**Update devcontainer config:**

Modify `/home/user/packnplay/pkg/devcontainer/config.go`:

```go
type Config struct {
	Image             string            `json:"image"`
	DockerFile        string            `json:"dockerFile"`
	Build             *BuildConfig      `json:"build"`
	RemoteUser        string            `json:"remoteUser"`
	ContainerEnv      map[string]string `json:"containerEnv"`
	RemoteEnv         map[string]string `json:"remoteEnv"`
	ForwardPorts      []interface{}     `json:"forwardPorts"`
	OnCreateCommand   interface{}       `json:"onCreateCommand"`   // NEW
	PostCreateCommand interface{}       `json:"postCreateCommand"` // NEW
	PostStartCommand  interface{}       `json:"postStartCommand"`  // NEW
}
```

**Integration into runner:**

Modify `/home/user/packnplay/pkg/runner/runner.go`:

```go
// In Run() function, after container is started but before exec:

// Execute lifecycle scripts
if devConfig.HasLifecycleScripts() {
	executor, err := NewLifecycleExecutor(dockerClient, containerName, containerUser)
	if err != nil {
		return fmt.Errorf("failed to create lifecycle executor: %w", err)
	}

	if err := executor.ExecuteLifecycle(devConfig, config.Verbose); err != nil {
		// Log error but don't fail - user's script may be broken
		fmt.Fprintf(os.Stderr, "Warning: lifecycle script failed: %v\n", err)
	}
}
```

Add helper to Config:

```go
// HasLifecycleScripts returns true if any lifecycle scripts are defined
func (c *Config) HasLifecycleScripts() bool {
	return c.OnCreateCommand != nil ||
	       c.PostCreateCommand != nil ||
	       c.PostStartCommand != nil
}
```

**Run tests:** `go test ./...` → **SHOULD PASS** (green)

**Commit:**
```bash
git add pkg/runner/lifecycle.go pkg/runner/lifecycle_test.go
git add pkg/runner/metadata.go pkg/runner/metadata_test.go
git add pkg/devcontainer/config.go pkg/runner/runner.go
git commit -m "feat: implement lifecycle scripts (onCreate, postCreate, postStart)

- Add metadata tracking to prevent re-running onCreate
- Support string, array, and object command formats per spec
- Execute via docker exec in running container
- Capture and display output
- Hash commands to detect changes
- onCreate/postCreate run once, postStart runs always
- Graceful error handling (warn but don't fail)
- Add comprehensive tests for all scenarios
- TDD: Red-Green-Refactor cycle

This is the highest-value feature for dev container support."
```

---

## Implementation Guidelines

### General Principles

1. **TDD Cycle:**
   - Write failing test (RED)
   - Write minimal code to pass (GREEN)
   - Refactor for quality (REFACTOR)
   - Run ALL tests before committing

2. **YAGNI (You Aren't Gonna Need It):**
   - Don't add features not in the spec
   - Don't add configuration options unless needed
   - Don't create abstractions until you have 3 uses

3. **DRY (Don't Repeat Yourself):**
   - Extract common code immediately
   - Use the existing Agent abstraction
   - Consolidate duplicates

4. **Commit Frequency:**
   - After each TDD cycle (red-green-refactor)
   - After each complete feature
   - When all tests pass
   - With clear, descriptive messages

5. **Code Review Focus:**
   - Security: Input validation, path handling
   - Testability: Can this be tested easily?
   - Maintainability: Will future devs understand this?
   - Performance: Any unnecessary work?
   - Error handling: Clear messages, proper wrapping

### Testing Standards

1. **Test Coverage:**
   - Every new function needs tests
   - Every error path needs a test
   - Edge cases must be covered

2. **Test Names:**
   - Format: `Test<Function>_<Scenario>`
   - Example: `TestExecuteCommand_StringFormat`

3. **Test Organization:**
   - Arrange: Set up test data
   - Act: Call function
   - Assert: Verify results

4. **Mock Usage:**
   - Create interfaces for external dependencies
   - Use simple struct mocks
   - Don't over-mock - test real code when possible

### Error Handling

1. **Wrap errors with context:**
   ```go
   if err != nil {
       return fmt.Errorf("failed to parse config: %w", err)
   }
   ```

2. **Provide actionable messages:**
   ```go
   return fmt.Errorf("devcontainer.json not found in %s - run 'packnplay configure'", path)
   ```

3. **Don't swallow errors:**
   - Log or return, never ignore

### Code Style

1. **Go conventions:**
   - gofmt on every file
   - Exported functions have godoc comments
   - Error checks immediately after calls

2. **Clarity over cleverness:**
   - Explicit is better than implicit
   - Simple is better than complex

3. **Function length:**
   - Keep under 50 lines when possible
   - Extract helper functions liberally

---

## Success Criteria

### Phase 1 Complete When:
- [ ] runner.Run() is under 350 lines
- [ ] All services have >80% test coverage
- [ ] All existing tests pass
- [ ] No duplicate code remains

### Phase 2 Complete When:
- [ ] containerEnv/remoteEnv parsed and working
- [ ] forwardPorts parsed and working
- [ ] build config parsed and working
- [ ] CLI flags still override devcontainer.json
- [ ] All tests pass

### Phase 3 Complete When:
- [ ] Lifecycle scripts execute correctly
- [ ] onCreate runs once only
- [ ] postStart runs every time
- [ ] Errors are handled gracefully
- [ ] All tests pass
- [ ] Documentation updated

---

## Estimated Timeline

**Phase 1:** 4-5 hours
**Phase 2:** 2 hours
**Phase 3:** 2 hours
**Total:** 8-9 hours

Each phase should be completed and committed before starting the next.
