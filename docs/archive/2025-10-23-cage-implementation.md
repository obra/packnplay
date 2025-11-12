# Cage Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build packnplay, a CLI tool that launches commands (like Claude Code) in isolated Docker containers with automated worktree and dev container management.

**Architecture:** Single Go binary using cobra for CLI, shells out to docker/git commands, uses idmap mounts for UID translation, session-based container lifecycle with Docker labels for state tracking.

**Tech Stack:** Go 1.21+, cobra CLI framework, Docker CLI (via shell), git CLI (via shell)

---

## Task 1: Project Initialization

**Files:**
- Create: `go.mod`
- Create: `main.go`
- Create: `.gitignore`

**Step 1: Initialize Go module**

Run:
```bash
cd /home/jesse/git/claude-launcher/.worktrees/packnplay-impl
go mod init github.com/jessedrelick/packnplay
```

Expected: `go.mod` created with module declaration

**Step 2: Create initial .gitignore for Go**

Create `.gitignore`:
```
# Binaries
packnplay
*.exe
*.exe~
*.dll
*.so
*.dylib

# Test binary
*.test

# Output of go build
*.out

# Go workspace file
go.work

# Dependency directories
vendor/

# IDE
.vscode/
.idea/
*.swp
*.swo
*~
```

**Step 3: Create placeholder main.go**

Create `main.go`:
```go
package main

import "fmt"

func main() {
	fmt.Println("packnplay")
}
```

**Step 4: Test build**

Run: `go build -o packnplay .`

Expected: Binary `packnplay` created successfully

Run: `./packnplay`

Expected: Output "packnplay"

**Step 5: Commit**

```bash
git add go.mod main.go .gitignore
git commit -m "feat: initialize Go project with basic structure

- Initialize Go module
- Add Go-specific .gitignore
- Create placeholder main.go
- Verify build succeeds"
```

---

## Task 2: CLI Framework Setup with Cobra

**Files:**
- Modify: `go.mod`
- Modify: `main.go`
- Create: `cmd/root.go`
- Create: `cmd/run.go`
- Create: `cmd/attach.go`
- Create: `cmd/stop.go`
- Create: `cmd/list.go`

**Step 1: Add cobra dependency**

Run:
```bash
go get github.com/spf13/cobra@latest
```

Expected: `go.mod` updated with cobra dependency

**Step 2: Create root command**

Create `cmd/root.go`:
```go
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "packnplay",
	Short: "Launch commands in isolated Docker containers",
	Long: `Cage runs commands (like Claude Code) inside isolated Docker containers
with automated worktree and dev container management.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

**Step 3: Create run command with flags**

Create `cmd/run.go`:
```go
package cmd

import (
	"github.com/spf13/cobra"
)

var (
	runPath      string
	runWorktree  string
	runNoWorktree bool
	runEnv       []string
	runVerbose   bool
)

var runCmd = &cobra.Command{
	Use:   "run [flags] [command...]",
	Short: "Run command in container",
	Long:  `Start a container and execute the specified command inside it.`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// TODO: implement
		cmd.Println("run command not yet implemented")
	},
}

func init() {
	rootCmd.AddCommand(runCmd)

	runCmd.Flags().StringVar(&runPath, "path", "", "Project path (default: pwd)")
	runCmd.Flags().StringVar(&runWorktree, "worktree", "", "Worktree name (creates if needed)")
	runCmd.Flags().BoolVar(&runNoWorktree, "no-worktree", false, "Skip worktree, use directory directly")
	runCmd.Flags().StringSliceVar(&runEnv, "env", []string{}, "Additional env vars (KEY=value)")
	runCmd.Flags().BoolVar(&runVerbose, "verbose", false, "Show all docker/git commands")
}
```

**Step 4: Create attach command**

Create `cmd/attach.go`:
```go
package cmd

import (
	"github.com/spf13/cobra"
)

var (
	attachPath     string
	attachWorktree string
)

var attachCmd = &cobra.Command{
	Use:   "attach [flags]",
	Short: "Attach to running container",
	Long:  `Attach to an existing running container with an interactive shell.`,
	Run: func(cmd *cobra.Command, args []string) {
		// TODO: implement
		cmd.Println("attach command not yet implemented")
	},
}

func init() {
	rootCmd.AddCommand(attachCmd)

	attachCmd.Flags().StringVar(&attachPath, "path", "", "Project path (default: pwd)")
	attachCmd.Flags().StringVar(&attachWorktree, "worktree", "", "Worktree name")
}
```

**Step 5: Create stop command**

Create `cmd/stop.go`:
```go
package cmd

import (
	"github.com/spf13/cobra"
)

var (
	stopPath     string
	stopWorktree string
)

var stopCmd = &cobra.Command{
	Use:   "stop [flags]",
	Short: "Stop container",
	Long:  `Stop the container for the specified project/worktree.`,
	Run: func(cmd *cobra.Command, args []string) {
		// TODO: implement
		cmd.Println("stop command not yet implemented")
	},
}

func init() {
	rootCmd.AddCommand(stopCmd)

	stopCmd.Flags().StringVar(&stopPath, "path", "", "Project path (default: pwd)")
	stopCmd.Flags().StringVar(&stopWorktree, "worktree", "", "Worktree name")
}
```

**Step 6: Create list command**

Create `cmd/list.go`:
```go
package cmd

import (
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all packnplay-managed containers",
	Long:  `Display all running containers managed by packnplay.`,
	Run: func(cmd *cobra.Command, args []string) {
		// TODO: implement
		cmd.Println("list command not yet implemented")
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
```

**Step 7: Update main.go**

Modify `main.go`:
```go
package main

import "github.com/jessedrelick/packnplay/cmd"

func main() {
	cmd.Execute()
}
```

**Step 8: Test CLI structure**

Run: `go build -o packnplay .`

Expected: Build succeeds

Run: `./packnplay --help`

Expected: Shows root help with subcommands (run, attach, stop, list)

Run: `./packnplay run --help`

Expected: Shows run command help with flags

**Step 9: Commit**

```bash
go mod tidy
git add go.mod go.sum main.go cmd/
git commit -m "feat: add CLI framework with cobra

- Add cobra dependency
- Create root command with help text
- Add run command with all flags (path, worktree, no-worktree, env, verbose)
- Add attach, stop, list commands with appropriate flags
- Update main.go to use cmd.Execute()
- Verify CLI structure with --help"
```

---

## Task 3: Docker CLI Detection and Execution Utility

**Files:**
- Create: `pkg/docker/client.go`
- Create: `pkg/docker/client_test.go`

**Step 1: Write test for Docker CLI detection**

Create `pkg/docker/client_test.go`:
```go
package docker

import (
	"os"
	"testing"
)

func TestDetectDockerCLI(t *testing.T) {
	tests := []struct {
		name    string
		envVar  string
		want    string
		wantErr bool
	}{
		{
			name:    "detect docker in PATH",
			envVar:  "",
			want:    "docker",
			wantErr: false,
		},
		{
			name:    "use DOCKER_CMD override",
			envVar:  "podman",
			want:    "podman",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envVar != "" {
				os.Setenv("DOCKER_CMD", tt.envVar)
				defer os.Unsetenv("DOCKER_CMD")
			}

			client := &Client{}
			cmd, err := client.DetectCLI()

			if (err != nil) != tt.wantErr {
				t.Errorf("DetectCLI() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if cmd != tt.want {
				t.Errorf("DetectCLI() = %v, want %v", cmd, tt.want)
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/docker/... -v`

Expected: FAIL - package or types don't exist

**Step 3: Implement Docker client**

Create `pkg/docker/client.go`:
```go
package docker

import (
	"fmt"
	"os"
	"os/exec"
)

// Client handles Docker CLI interactions
type Client struct {
	cmd     string
	verbose bool
}

// NewClient creates a new Docker client
func NewClient(verbose bool) (*Client, error) {
	client := &Client{verbose: verbose}
	cmd, err := client.DetectCLI()
	if err != nil {
		return nil, err
	}
	client.cmd = cmd
	return client, nil
}

// DetectCLI finds the docker command to use
func (c *Client) DetectCLI() (string, error) {
	// Check for DOCKER_CMD environment variable
	if envCmd := os.Getenv("DOCKER_CMD"); envCmd != "" {
		if _, err := exec.LookPath(envCmd); err != nil {
			return "", fmt.Errorf("DOCKER_CMD=%s not found in PATH", envCmd)
		}
		return envCmd, nil
	}

	// Try docker first
	if _, err := exec.LookPath("docker"); err == nil {
		return "docker", nil
	}

	// Try podman as fallback
	if _, err := exec.LookPath("podman"); err == nil {
		return "podman", nil
	}

	return "", fmt.Errorf("no docker-compatible CLI found (tried: docker, podman)")
}

// Run executes a docker command
func (c *Client) Run(args ...string) (string, error) {
	cmd := exec.Command(c.cmd, args...)

	if c.verbose {
		fmt.Fprintf(os.Stderr, "+ %s %v\n", c.cmd, args)
	}

	output, err := cmd.CombinedOutput()

	if c.verbose && len(output) > 0 {
		fmt.Fprintf(os.Stderr, "%s\n", output)
	}

	return string(output), err
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./pkg/docker/... -v`

Expected: PASS

**Step 5: Commit**

```bash
git add pkg/docker/
git commit -m "feat: add Docker CLI detection and execution utility

- Implement Client type for Docker CLI interactions
- Add DetectCLI() to find docker/podman in PATH
- Support DOCKER_CMD environment variable override
- Add Run() method to execute docker commands with optional verbose output
- Add tests for CLI detection"
```

---

## Task 4: Git Worktree Management

**Files:**
- Create: `pkg/git/worktree.go`
- Create: `pkg/git/worktree_test.go`

**Step 1: Write test for worktree path determination**

Create `pkg/git/worktree_test.go`:
```go
package git

import (
	"testing"
)

func TestDetermineWorktreePath(t *testing.T) {
	tests := []struct {
		name          string
		projectPath   string
		worktreeName  string
		wantContains  string
	}{
		{
			name:         "basic worktree path",
			projectPath:  "/home/user/myproject",
			worktreeName: "feature-auth",
			wantContains: "myproject-feature-auth",
		},
		{
			name:         "sanitize slashes in branch name",
			projectPath:  "/home/user/myproject",
			worktreeName: "feature/auth",
			wantContains: "myproject-feature-auth",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetermineWorktreePath(tt.projectPath, tt.worktreeName)

			if !contains(got, tt.wantContains) {
				t.Errorf("DetermineWorktreePath() = %v, want to contain %v", got, tt.wantContains)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr || len(s) > len(substr) &&
		(s[0:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
		findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/git/... -v`

Expected: FAIL - package doesn't exist

**Step 3: Implement worktree management**

Create `pkg/git/worktree.go`:
```go
package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// DetermineWorktreePath calculates the path for a worktree
func DetermineWorktreePath(projectPath, worktreeName string) string {
	projectName := filepath.Base(projectPath)
	sanitizedName := sanitizeBranchName(worktreeName)

	parentDir := filepath.Dir(projectPath)
	return filepath.Join(parentDir, fmt.Sprintf("%s-%s", projectName, sanitizedName))
}

// sanitizeBranchName converts branch name to filesystem-safe name
func sanitizeBranchName(name string) string {
	// Replace slashes with dashes
	name = strings.ReplaceAll(name, "/", "-")
	// Remove other problematic characters
	name = strings.ReplaceAll(name, ":", "-")
	name = strings.ReplaceAll(name, " ", "-")
	return name
}

// IsGitRepo checks if a directory is a git repository
func IsGitRepo(path string) bool {
	cmd := exec.Command("git", "-C", path, "rev-parse", "--git-dir")
	return cmd.Run() == nil
}

// GetCurrentBranch returns the current branch name
func GetCurrentBranch(path string) (string, error) {
	cmd := exec.Command("git", "-C", path, "branch", "--show-current")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// WorktreeExists checks if a worktree with the given name exists
func WorktreeExists(worktreeName string) (bool, error) {
	cmd := exec.Command("git", "worktree", "list", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}

	// Parse worktree list output
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "branch ") {
			branch := strings.TrimPrefix(line, "branch refs/heads/")
			if branch == worktreeName {
				return true, nil
			}
		}
	}
	return false, nil
}

// CreateWorktree creates a new worktree
func CreateWorktree(path, branchName string, verbose bool) error {
	cmd := exec.Command("git", "worktree", "add", path, "-b", branchName)

	if verbose {
		fmt.Fprintf(os.Stderr, "+ git worktree add %s -b %s\n", path, branchName)
		cmd.Stdout = os.Stderr
		cmd.Stderr = os.Stderr
	}

	return cmd.Run()
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./pkg/git/... -v`

Expected: PASS

**Step 5: Commit**

```bash
git add pkg/git/
git commit -m "feat: add git worktree management utilities

- Implement DetermineWorktreePath() for calculating worktree locations
- Add sanitizeBranchName() to convert branches to filesystem-safe names
- Add IsGitRepo() to check if directory is a git repository
- Add GetCurrentBranch() to get current branch name
- Add WorktreeExists() to check if worktree already exists
- Add CreateWorktree() to create new worktrees
- Add tests for path determination and name sanitization"
```

---

## Task 5: Dev Container Discovery and Parsing

**Files:**
- Create: `pkg/devcontainer/config.go`
- Create: `pkg/devcontainer/config_test.go`

**Step 1: Write test for devcontainer parsing**

Create `pkg/devcontainer/config_test.go`:
```go
package devcontainer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Create temp dir with devcontainer.json
	tmpDir := t.TempDir()
	devcontainerDir := filepath.Join(tmpDir, ".devcontainer")
	os.Mkdir(devcontainerDir, 0755)

	configContent := `{
		"image": "mcr.microsoft.com/devcontainers/base:ubuntu",
		"remoteUser": "vscode"
	}`

	os.WriteFile(
		filepath.Join(devcontainerDir, "devcontainer.json"),
		[]byte(configContent),
		0644,
	)

	config, err := LoadConfig(tmpDir)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if config.Image != "mcr.microsoft.com/devcontainers/base:ubuntu" {
		t.Errorf("Image = %v, want mcr.microsoft.com/devcontainers/base:ubuntu", config.Image)
	}

	if config.RemoteUser != "vscode" {
		t.Errorf("RemoteUser = %v, want vscode", config.RemoteUser)
	}
}

func TestLoadConfig_NotFound(t *testing.T) {
	tmpDir := t.TempDir()

	config, err := LoadConfig(tmpDir)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v, want nil for missing config", err)
	}

	if config != nil {
		t.Errorf("LoadConfig() = %v, want nil for missing config", config)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/devcontainer/... -v`

Expected: FAIL - package doesn't exist

**Step 3: Implement devcontainer config parsing**

Create `pkg/devcontainer/config.go`:
```go
package devcontainer

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config represents a parsed devcontainer.json
type Config struct {
	Image       string `json:"image"`
	DockerFile  string `json:"dockerFile"`
	RemoteUser  string `json:"remoteUser"`
}

// LoadConfig loads and parses .devcontainer/devcontainer.json if it exists
func LoadConfig(projectPath string) (*Config, error) {
	configPath := filepath.Join(projectPath, ".devcontainer", "devcontainer.json")

	// Check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	// Set default remote user if not specified
	if config.RemoteUser == "" {
		config.RemoteUser = "devuser"
	}

	return &config, nil
}

// GetDefaultConfig returns the default devcontainer config
func GetDefaultConfig() *Config {
	return &Config{
		Image:      "mcr.microsoft.com/devcontainers/base:ubuntu",
		RemoteUser: "devuser",
	}
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./pkg/devcontainer/... -v`

Expected: PASS

**Step 5: Commit**

```bash
git add pkg/devcontainer/
git commit -m "feat: add devcontainer config discovery and parsing

- Implement Config type for devcontainer.json
- Add LoadConfig() to parse .devcontainer/devcontainer.json
- Support image and dockerFile fields
- Parse remoteUser with default fallback
- Add GetDefaultConfig() for projects without devcontainer
- Add tests for config parsing and missing config handling"
```

---

## Task 6: Container Name Generation and Label Management

**Files:**
- Create: `pkg/container/naming.go`
- Create: `pkg/container/naming_test.go`

**Step 1: Write test for container naming**

Create `pkg/container/naming_test.go`:
```go
package container

import (
	"testing"
)

func TestGenerateContainerName(t *testing.T) {
	tests := []struct {
		name         string
		projectPath  string
		worktreeName string
		want         string
	}{
		{
			name:         "basic naming",
			projectPath:  "/home/user/myproject",
			worktreeName: "main",
			want:         "packnplay-myproject-main",
		},
		{
			name:         "sanitized worktree name",
			projectPath:  "/home/user/myproject",
			worktreeName: "feature/auth",
			want:         "packnplay-myproject-feature-auth",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateContainerName(tt.projectPath, tt.worktreeName)
			if got != tt.want {
				t.Errorf("GenerateContainerName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGenerateLabels(t *testing.T) {
	labels := GenerateLabels("myproject", "feature-auth")

	if labels["managed-by"] != "packnplay" {
		t.Errorf("managed-by label = %v, want packnplay", labels["managed-by"])
	}

	if labels["packnplay-project"] != "myproject" {
		t.Errorf("packnplay-project label = %v, want myproject", labels["packnplay-project"])
	}

	if labels["packnplay-worktree"] != "feature-auth" {
		t.Errorf("packnplay-worktree label = %v, want feature-auth", labels["packnplay-worktree"])
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/container/... -v`

Expected: FAIL - package doesn't exist

**Step 3: Implement container naming**

Create `pkg/container/naming.go`:
```go
package container

import (
	"fmt"
	"path/filepath"
	"strings"
)

// GenerateContainerName creates a container name from project and worktree
func GenerateContainerName(projectPath, worktreeName string) string {
	projectName := filepath.Base(projectPath)
	sanitizedWorktree := sanitizeName(worktreeName)
	return fmt.Sprintf("packnplay-%s-%s", projectName, sanitizedWorktree)
}

// sanitizeName converts a name to docker-compatible format
func sanitizeName(name string) string {
	// Docker container names: [a-zA-Z0-9][a-zA-Z0-9_.-]*
	name = strings.ReplaceAll(name, "/", "-")
	name = strings.ReplaceAll(name, " ", "-")
	name = strings.ReplaceAll(name, ":", "-")
	return name
}

// GenerateLabels creates Docker labels for packnplay-managed containers
func GenerateLabels(projectName, worktreeName string) map[string]string {
	return map[string]string{
		"managed-by":    "packnplay",
		"packnplay-project":  projectName,
		"packnplay-worktree": worktreeName,
	}
}

// LabelsToArgs converts label map to docker --label args
func LabelsToArgs(labels map[string]string) []string {
	args := make([]string, 0, len(labels)*2)
	for k, v := range labels {
		args = append(args, "--label", fmt.Sprintf("%s=%s", k, v))
	}
	return args
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./pkg/container/... -v`

Expected: PASS

**Step 5: Commit**

```bash
git add pkg/container/
git commit -m "feat: add container naming and label management

- Implement GenerateContainerName() for packnplay-<project>-<worktree> pattern
- Add sanitizeName() to ensure Docker-compatible names
- Add GenerateLabels() to create packnplay management labels
- Add LabelsToArgs() to convert labels to docker CLI args
- Add tests for name generation and label creation"
```

---

## Task 7: Implement `packnplay run` Command Logic

**Files:**
- Modify: `cmd/run.go`
- Create: `pkg/runner/runner.go`

**Step 1: Write test for run logic**

This step involves integration testing which is complex. We'll implement the runner and test manually.

**Step 2: Implement runner package**

Create `pkg/runner/runner.go`:
```go
package runner

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/jessedrelick/packnplay/pkg/container"
	"github.com/jessedrelick/packnplay/pkg/devcontainer"
	"github.com/jessedrelick/packnplay/pkg/docker"
	"github.com/jessedrelick/packnplay/pkg/git"
)

type RunConfig struct {
	Path        string
	Worktree    string
	NoWorktree  bool
	Env         []string
	Verbose     bool
	Command     []string
}

func Run(config *RunConfig) error {
	// Step 1: Determine working directory
	workDir := config.Path
	if workDir == "" {
		var err error
		workDir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}
	}

	// Make absolute
	workDir, err := filepath.Abs(workDir)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	// Step 2: Handle worktree logic
	var mountPath string
	var worktreeName string

	if config.NoWorktree {
		// Use directory directly
		mountPath = workDir
		worktreeName = "no-worktree"
	} else {
		// Check if git repo
		if !git.IsGitRepo(workDir) {
			if config.Worktree != "" {
				return fmt.Errorf("--worktree specified but %s is not a git repository", workDir)
			}
			// Not a git repo and no worktree flag: use directly
			mountPath = workDir
			worktreeName = "no-worktree"
		} else {
			// Is a git repo
			if config.Worktree != "" {
				worktreeName = config.Worktree
			} else {
				// Auto-detect from current branch
				branch, err := git.GetCurrentBranch(workDir)
				if err != nil {
					return fmt.Errorf("failed to get current branch: %w", err)
				}
				worktreeName = branch
			}

			// Check if worktree exists
			exists, err := git.WorktreeExists(worktreeName)
			if err != nil {
				return fmt.Errorf("failed to check worktree: %w", err)
			}

			if exists {
				return fmt.Errorf("worktree '%s' already exists. Use --worktree=%s to connect, or --no-worktree to use directory directly", worktreeName, worktreeName)
			}

			// Create worktree
			mountPath = git.DetermineWorktreePath(workDir, worktreeName)
			if config.Verbose {
				fmt.Fprintf(os.Stderr, "Creating worktree at %s\n", mountPath)
			}

			if err := git.CreateWorktree(mountPath, worktreeName, config.Verbose); err != nil {
				return fmt.Errorf("failed to create worktree: %w", err)
			}
		}
	}

	// Step 3: Load devcontainer config
	devConfig, err := devcontainer.LoadConfig(mountPath)
	if err != nil {
		return fmt.Errorf("failed to load devcontainer config: %w", err)
	}
	if devConfig == nil {
		devConfig = devcontainer.GetDefaultConfig()
	}

	// Step 4: Initialize Docker client
	dockerClient, err := docker.NewClient(config.Verbose)
	if err != nil {
		return fmt.Errorf("failed to initialize docker: %w", err)
	}

	// Step 5: Ensure image available
	if err := ensureImage(dockerClient, devConfig, mountPath, config.Verbose); err != nil {
		return err
	}

	// Step 6: Generate container name and labels
	projectName := filepath.Base(workDir)
	containerName := container.GenerateContainerName(workDir, worktreeName)
	labels := container.GenerateLabels(projectName, worktreeName)

	// Step 7: Check if container already running
	if isRunning, err := containerIsRunning(dockerClient, containerName); err != nil {
		return fmt.Errorf("failed to check container status: %w", err)
	} else if isRunning {
		return fmt.Errorf("container already running. Use 'packnplay attach --worktree=%s' or 'packnplay stop --worktree=%s'", worktreeName, worktreeName)
	}

	// Step 8: Build docker run command
	args := []string{"run", "--rm", "-it"}

	// Add labels
	args = append(args, container.LabelsToArgs(labels)...)

	// Add name
	args = append(args, "--name", containerName)

	// Get current user for idmap
	currentUser, err := user.Current()
	if err != nil {
		return fmt.Errorf("failed to get current user: %w", err)
	}

	// Add mounts with idmap
	homeDir := currentUser.HomeDir

	// Mount .claude directory
	args = append(args, "--mount", fmt.Sprintf(
		"type=bind,source=%s/.claude,target=/home/%s/.claude,idmap=uids=%s-%s-1000:gids=%s-%s-1000",
		homeDir, devConfig.RemoteUser, currentUser.Uid, currentUser.Uid, currentUser.Gid, currentUser.Gid,
	))

	// Mount workspace
	args = append(args, "--mount", fmt.Sprintf(
		"type=bind,source=%s,target=/workspace,idmap=uids=%s-%s-1000:gids=%s-%s-1000",
		mountPath, currentUser.Uid, currentUser.Uid, currentUser.Gid, currentUser.Gid,
	))

	// Set working directory
	args = append(args, "-w", "/workspace")

	// Add environment variables
	// Copy host environment
	for _, env := range os.Environ() {
		args = append(args, "-e", env)
	}

	// Add IS_SANDBOX
	args = append(args, "-e", "IS_SANDBOX=1")

	// Add custom env vars
	for _, env := range config.Env {
		args = append(args, "-e", env)
	}

	// Add image
	imageName := devConfig.Image
	if devConfig.DockerFile != "" {
		imageName = fmt.Sprintf("packnplay-%s-devcontainer:latest", projectName)
	}
	args = append(args, imageName)

	// Add command to run
	args = append(args, config.Command...)

	// Step 9: Run container
	if config.Verbose {
		fmt.Fprintf(os.Stderr, "Starting container %s\n", containerName)
	}

	// Execute docker run - this blocks until container exits
	// We need to run this interactively, not via dockerClient.Run()
	// because we need to preserve stdin/stdout/stderr
	return execDocker(dockerClient, args)
}

func ensureImage(dockerClient *docker.Client, config *devcontainer.Config, projectPath string, verbose bool) error {
	var imageName string

	if config.DockerFile != "" {
		// Need to build from Dockerfile
		projectName := filepath.Base(projectPath)
		imageName = fmt.Sprintf("packnplay-%s-devcontainer:latest", projectName)

		// Check if already built
		_, err := dockerClient.Run("image", "inspect", imageName)
		if err != nil {
			// Need to build
			if verbose {
				fmt.Fprintf(os.Stderr, "Building image from %s\n", config.DockerFile)
			}

			dockerfilePath := filepath.Join(projectPath, ".devcontainer", config.DockerFile)
			contextPath := filepath.Join(projectPath, ".devcontainer")

			_, err := dockerClient.Run("build", "-f", dockerfilePath, "-t", imageName, contextPath)
			if err != nil {
				return fmt.Errorf("failed to build image: %w", err)
			}
		}
	} else {
		// Use pre-built image
		imageName = config.Image

		// Check if exists locally
		_, err := dockerClient.Run("image", "inspect", imageName)
		if err != nil {
			// Need to pull
			if verbose {
				fmt.Fprintf(os.Stderr, "Pulling image %s\n", imageName)
			}

			_, err := dockerClient.Run("pull", imageName)
			if err != nil {
				return fmt.Errorf("failed to pull image: %w", err)
			}
		}
	}

	return nil
}

func containerIsRunning(dockerClient *docker.Client, name string) (bool, error) {
	output, err := dockerClient.Run("ps", "--filter", fmt.Sprintf("name=%s", name), "--format", "{{.Names}}")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(output) == name, nil
}

func execDocker(dockerClient *docker.Client, args []string) error {
	// Import exec and syscall for direct execution
	// This is a workaround - we need to run docker interactively
	// For now, use a simple approach with os/exec
	// TODO: This needs refinement for proper interactive mode
	return fmt.Errorf("interactive docker execution not yet implemented - need to use syscall.Exec")
}
```

**Step 3: Update cmd/run.go to use runner**

Modify `cmd/run.go`:
```go
package cmd

import (
	"fmt"
	"os"

	"github.com/jessedrelick/packnplay/pkg/runner"
	"github.com/spf13/cobra"
)

var (
	runPath       string
	runWorktree   string
	runNoWorktree bool
	runEnv        []string
	runVerbose    bool
)

var runCmd = &cobra.Command{
	Use:   "run [flags] [command...]",
	Short: "Run command in container",
	Long:  `Start a container and execute the specified command inside it.`,
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		config := &runner.RunConfig{
			Path:       runPath,
			Worktree:   runWorktree,
			NoWorktree: runNoWorktree,
			Env:        runEnv,
			Verbose:    runVerbose,
			Command:    args,
		}

		if err := runner.Run(config); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return err
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(runCmd)

	runCmd.Flags().StringVar(&runPath, "path", "", "Project path (default: pwd)")
	runCmd.Flags().StringVar(&runWorktree, "worktree", "", "Worktree name (creates if needed)")
	runCmd.Flags().BoolVar(&runNoWorktree, "no-worktree", false, "Skip worktree, use directory directly")
	runCmd.Flags().StringSliceVar(&runEnv, "env", []string{}, "Additional env vars (KEY=value)")
	runCmd.Flags().BoolVar(&runVerbose, "verbose", false, "Show all docker/git commands")
}
```

**Step 4: Fix interactive docker execution**

We need to properly handle interactive execution. Modify `pkg/runner/runner.go` - add the execDocker function properly:

```go
// Add to imports
import (
	"os/exec"
	"syscall"
)

// Replace execDocker function
func execDocker(dockerClient *docker.Client, args []string) error {
	// Get docker command path
	cmdPath, err := exec.LookPath(dockerClient.Command())
	if err != nil {
		return fmt.Errorf("failed to find docker command: %w", err)
	}

	// Use syscall.Exec to replace current process
	// This preserves stdin/stdout/stderr properly for interactive mode
	argv := append([]string{filepath.Base(cmdPath)}, args...)
	return syscall.Exec(cmdPath, argv, os.Environ())
}
```

Also need to add a method to docker.Client:

Modify `pkg/docker/client.go`:
```go
// Add this method
func (c *Client) Command() string {
	return c.cmd
}
```

**Step 5: Build and test manually**

Run: `go build -o packnplay .`

Expected: Build succeeds (may have import errors to fix with `go mod tidy`)

Run: `go mod tidy`

**Step 6: Commit**

```bash
git add cmd/run.go pkg/runner/ pkg/docker/client.go
git commit -m "feat: implement packnplay run command logic

- Create runner package with Run() function
- Implement worktree management logic (auto-create, check existence, no-worktree mode)
- Add devcontainer config loading with fallback to default
- Implement Docker image handling (pull for image, build for dockerFile)
- Generate container names and labels
- Check for running containers to prevent conflicts
- Build docker run command with idmap mounts for .claude and workspace
- Copy host environment and add IS_SANDBOX=1
- Use syscall.Exec for proper interactive mode
- Wire up cmd/run.go to use runner package"
```

---

## Task 8: Implement `packnplay attach` Command

**Files:**
- Modify: `cmd/attach.go`

**Step 1: Implement attach command**

Modify `cmd/attach.go`:
```go
package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/jessedrelick/packnplay/pkg/container"
	"github.com/jessedrelick/packnplay/pkg/docker"
	"github.com/spf13/cobra"
)

var (
	attachPath     string
	attachWorktree string
)

var attachCmd = &cobra.Command{
	Use:   "attach [flags]",
	Short: "Attach to running container",
	Long:  `Attach to an existing running container with an interactive shell.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Determine working directory
		workDir := attachPath
		if workDir == "" {
			var err error
			workDir, err = os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get working directory: %w", err)
			}
		}

		workDir, err := filepath.Abs(workDir)
		if err != nil {
			return fmt.Errorf("failed to resolve path: %w", err)
		}

		// Determine worktree name
		worktreeName := attachWorktree
		if worktreeName == "" {
			return fmt.Errorf("--worktree flag is required for attach")
		}

		// Generate container name
		containerName := container.GenerateContainerName(workDir, worktreeName)

		// Initialize Docker client
		dockerClient, err := docker.NewClient(false)
		if err != nil {
			return fmt.Errorf("failed to initialize docker: %w", err)
		}

		// Check if container is running
		output, err := dockerClient.Run("ps", "--filter", fmt.Sprintf("name=%s", containerName), "--format", "{{.Names}}")
		if err != nil {
			return fmt.Errorf("failed to check container status: %w", err)
		}

		if strings.TrimSpace(output) != containerName {
			return fmt.Errorf("no running container found for worktree '%s'", worktreeName)
		}

		// Execute docker exec with interactive shell
		cmdPath, err := exec.LookPath(dockerClient.Command())
		if err != nil {
			return fmt.Errorf("failed to find docker command: %w", err)
		}

		argv := []string{
			filepath.Base(cmdPath),
			"exec",
			"-it",
			containerName,
			"/bin/bash",
		}

		return syscall.Exec(cmdPath, argv, os.Environ())
	},
}

func init() {
	rootCmd.AddCommand(attachCmd)

	attachCmd.Flags().StringVar(&attachPath, "path", "", "Project path (default: pwd)")
	attachCmd.Flags().StringVar(&attachWorktree, "worktree", "", "Worktree name")
}
```

**Step 2: Build and test**

Run: `go build -o packnplay .`

Expected: Build succeeds

**Step 3: Commit**

```bash
git add cmd/attach.go
git commit -m "feat: implement packnplay attach command

- Add attach command logic to connect to running containers
- Require --worktree flag to identify container
- Use docker exec with /bin/bash for interactive shell
- Use syscall.Exec for proper interactive mode
- Check container is running before attempting attach"
```

---

## Task 9: Implement `packnplay stop` Command

**Files:**
- Modify: `cmd/stop.go`

**Step 1: Implement stop command**

Modify `cmd/stop.go`:
```go
package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jessedrelick/packnplay/pkg/container"
	"github.com/jessedrelick/packnplay/pkg/docker"
	"github.com/spf13/cobra"
)

var (
	stopPath     string
	stopWorktree string
)

var stopCmd = &cobra.Command{
	Use:   "stop [flags]",
	Short: "Stop container",
	Long:  `Stop the container for the specified project/worktree.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Determine working directory
		workDir := stopPath
		if workDir == "" {
			var err error
			workDir, err = os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get working directory: %w", err)
			}
		}

		workDir, err := filepath.Abs(workDir)
		if err != nil {
			return fmt.Errorf("failed to resolve path: %w", err)
		}

		// Determine worktree name
		worktreeName := stopWorktree
		if worktreeName == "" {
			return fmt.Errorf("--worktree flag is required for stop")
		}

		// Generate container name
		containerName := container.GenerateContainerName(workDir, worktreeName)

		// Initialize Docker client
		dockerClient, err := docker.NewClient(false)
		if err != nil {
			return fmt.Errorf("failed to initialize docker: %w", err)
		}

		// Stop container
		fmt.Printf("Stopping container %s...\n", containerName)
		_, err = dockerClient.Run("stop", containerName)
		if err != nil {
			return fmt.Errorf("failed to stop container: %w", err)
		}

		// Remove container
		_, err = dockerClient.Run("rm", containerName)
		if err != nil {
			return fmt.Errorf("failed to remove container: %w", err)
		}

		fmt.Printf("Container %s stopped and removed\n", containerName)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(stopCmd)

	stopCmd.Flags().StringVar(&stopPath, "path", "", "Project path (default: pwd)")
	stopCmd.Flags().StringVar(&stopWorktree, "worktree", "", "Worktree name")
}
```

**Step 2: Build and test**

Run: `go build -o packnplay .`

Expected: Build succeeds

**Step 3: Commit**

```bash
git add cmd/stop.go
git commit -m "feat: implement packnplay stop command

- Add stop command to stop and remove containers
- Require --worktree flag to identify container
- Use docker stop followed by docker rm
- Provide user feedback on stop/remove progress"
```

---

## Task 10: Implement `packnplay list` Command

**Files:**
- Modify: `cmd/list.go`

**Step 1: Implement list command**

Modify `cmd/list.go`:
```go
package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/jessedrelick/packnplay/pkg/docker"
	"github.com/spf13/cobra"
)

type ContainerInfo struct {
	Names  string `json:"Names"`
	Status string `json:"Status"`
	Labels string `json:"Labels"`
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all packnplay-managed containers",
	Long:  `Display all running containers managed by packnplay.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Initialize Docker client
		dockerClient, err := docker.NewClient(false)
		if err != nil {
			return fmt.Errorf("failed to initialize docker: %w", err)
		}

		// Get all packnplay-managed containers
		output, err := dockerClient.Run(
			"ps",
			"--filter", "label=managed-by=packnplay",
			"--format", "{{json .}}",
		)
		if err != nil {
			return fmt.Errorf("failed to list containers: %w", err)
		}

		if output == "" {
			fmt.Println("No packnplay-managed containers running")
			return nil
		}

		// Parse JSON output
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "CONTAINER\tSTATUS\tPROJECT\tWORKTREE")

		// Docker outputs one JSON object per line
		lines := splitLines(output)
		for _, line := range lines {
			if line == "" {
				continue
			}

			var info ContainerInfo
			if err := json.Unmarshal([]byte(line), &info); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to parse container info: %v\n", err)
				continue
			}

			// Parse labels to extract project and worktree
			project, worktree := parseLabels(info.Labels)

			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
				info.Names,
				info.Status,
				project,
				worktree,
			)
		}

		w.Flush()
		return nil
	},
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func parseLabels(labels string) (project, worktree string) {
	// Labels format: "label1=value1,label2=value2"
	pairs := splitByComma(labels)
	for _, pair := range pairs {
		kv := splitByEquals(pair)
		if len(kv) == 2 {
			if kv[0] == "packnplay-project" {
				project = kv[1]
			} else if kv[0] == "packnplay-worktree" {
				worktree = kv[1]
			}
		}
	}
	return
}

func splitByComma(s string) []string {
	var parts []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == ',' {
			parts = append(parts, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		parts = append(parts, s[start:])
	}
	return parts
}

func splitByEquals(s string) []string {
	for i := 0; i < len(s); i++ {
		if s[i] == '=' {
			return []string{s[:i], s[i+1:]}
		}
	}
	return []string{s}
}

func init() {
	rootCmd.AddCommand(listCmd)
}
```

**Step 2: Build and test**

Run: `go build -o packnplay .`

Expected: Build succeeds

**Step 3: Commit**

```bash
git add cmd/list.go
git commit -m "feat: implement packnplay list command

- Add list command to show packnplay-managed containers
- Filter containers by managed-by=packnplay label
- Parse JSON output from docker ps
- Display table with container name, status, project, and worktree
- Handle empty results gracefully"
```

---

## Task 11: Fix ~/.claude.json Copy Logic

**Files:**
- Modify: `pkg/runner/runner.go`

**Step 1: Add ~/.claude.json copy after container starts**

The current implementation doesn't handle the ~/.claude.json copy. We need to revise the approach since we're using `syscall.Exec` which replaces the process. We need to copy the file before exec.

Actually, we have a problem: we can't copy a file into a container before it's running, and if we use `syscall.Exec`, we replace our process.

Solution: Don't use `--rm` flag, start container in background, copy file, then exec into it.

Modify the `Run` function in `pkg/runner/runner.go` - change the docker run approach:

```go
// Change Step 8 in Run() function
// Replace the entire Step 8 and Step 9 with:

	// Step 8: Build docker run command for background container
	args := []string{"run", "-d", "-it"} // -d for detached, keep -it for interactive

	// ... rest of args building stays the same until the image ...

	// Add image
	imageName := devConfig.Image
	if devConfig.DockerFile != "" {
		imageName = fmt.Sprintf("packnplay-%s-devcontainer:latest", projectName)
	}
	args = append(args, imageName)

	// Add a command that keeps container alive
	args = append(args, "sleep", "infinity")

	// Step 9: Start container in background
	if config.Verbose {
		fmt.Fprintf(os.Stderr, "Starting container %s\n", containerName)
	}

	containerID, err := dockerClient.Run(args...)
	if err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}
	containerID = strings.TrimSpace(containerID)

	// Step 10: Copy ~/.claude.json into container
	claudeConfigSrc := filepath.Join(homeDir, ".claude.json")
	claudeConfigDst := fmt.Sprintf("%s:/home/%s/.claude.json", containerID, devConfig.RemoteUser)

	if _, err := os.Stat(claudeConfigSrc); err == nil {
		if config.Verbose {
			fmt.Fprintf(os.Stderr, "Copying %s to container\n", claudeConfigSrc)
		}
		_, err = dockerClient.Run("cp", claudeConfigSrc, claudeConfigDst)
		if err != nil {
			// Clean up container on error
			dockerClient.Run("rm", "-f", containerID)
			return fmt.Errorf("failed to copy .claude.json: %w", err)
		}
	}

	// Step 11: Exec into container with user's command
	cmdPath, err := exec.LookPath(dockerClient.Command())
	if err != nil {
		return fmt.Errorf("failed to find docker command: %w", err)
	}

	execArgs := []string{
		filepath.Base(cmdPath),
		"exec",
		"-it",
		"-w", "/workspace",
		containerID,
	}
	execArgs = append(execArgs, config.Command...)

	// Use syscall.Exec to replace current process
	return syscall.Exec(cmdPath, execArgs, os.Environ())
```

**Step 2: Test build**

Run: `go build -o packnplay .`

Expected: Build succeeds

**Step 3: Commit**

```bash
git add pkg/runner/runner.go
git commit -m "fix: properly handle ~/.claude.json copy and container lifecycle

- Start container in detached mode with sleep infinity
- Copy ~/.claude.json after container starts
- Exec into running container with user command
- Clean up container on copy failure
- This allows proper file copy before exec replaces process"
```

---

## Task 12: Add README and Usage Documentation

**Files:**
- Create: `README.md`

**Step 1: Create README**

Create `README.md`:
```markdown
# Cage

Cage launches commands (like Claude Code) inside isolated Docker containers with automated worktree and dev container management.

## Features

- **Sandboxed Execution**: Run commands in isolated Docker containers
- **Automatic Worktree Management**: Creates and manages git worktrees automatically
- **Dev Container Support**: Uses project's `.devcontainer/devcontainer.json` or sensible defaults
- **UID Mapping**: Proper file ownership with idmap mounts (Linux 6.0.8+, Docker 28.5.1+)
- **Environment Proxying**: Forwards host environment with `IS_SANDBOX=1` indicator

## Installation

```bash
go build -o packnplay .
sudo mv packnplay /usr/local/bin/
```

Or install directly:

```bash
go install github.com/jessedrelick/packnplay@latest
```

## Usage

### Run a command in a container

```bash
packnplay run 'claude --dangerously-skip-permissions'
```

### Specify a worktree

```bash
packnplay run --worktree=feature-auth claude
```

### Use current directory without worktree

```bash
packnplay run --no-worktree bash
```

### Add environment variables

```bash
packnplay run --env DEBUG=1 --env LOG_LEVEL=trace claude
```

### Attach to running container

```bash
packnplay attach --worktree=feature-auth
```

### Stop a container

```bash
packnplay stop --worktree=feature-auth
```

### List all containers

```bash
packnplay list
```

## How It Works

### Worktree Management

- **Auto-create**: If you're in a git repo, packnplay creates a worktree based on current branch
- **Explicit**: Use `--worktree=<name>` to specify or create a worktree
- **Skip**: Use `--no-worktree` to use directory directly
- **Collision detection**: Errors if worktree already exists (prevents accidents)

### Dev Container Discovery

1. Checks for `.devcontainer/devcontainer.json` in project
2. Falls back to `mcr.microsoft.com/devcontainers/base:ubuntu` if not found
3. Supports both `image` (pulls) and `dockerFile` (builds) fields
4. Auto-pulls/builds images as needed

### File Mounts

- `~/.claude` → mounted read-write (skills, plugins, history)
- `~/.claude.json` → copied (avoids file lock conflicts)
- Project/worktree → mounted at `/workspace` with idmap

### Container Lifecycle

- Session-based: container runs until command exits
- Labeled: all containers tagged with `managed-by=packnplay`
- Multiple sessions can attach to running containers

## Requirements

- Linux 6.0.8+ (for idmap support)
- Docker 28.5.1+ (for idmap support)
- Git (for worktree features)
- Go 1.21+ (for building)

## Environment Variables

- `DOCKER_CMD`: Override docker command (e.g., `DOCKER_CMD=podman packnplay run ...`)

## Examples

```bash
# Run Claude in auto-created worktree
cd ~/myproject
packnplay run claude

# Run in specific worktree with debug logging
packnplay run --worktree=bug-fix --env DEBUG=1 --verbose claude

# Get a shell in the container
packnplay run --worktree=feature bash

# Attach to running container
packnplay attach --worktree=feature

# List all running containers
packnplay list

# Stop container
packnplay stop --worktree=feature
```

## License

MIT
```

**Step 2: Commit**

```bash
git add README.md
git commit -m "docs: add README with usage and examples

- Document installation instructions
- Explain worktree management behavior
- Describe dev container discovery
- List file mounts and container lifecycle
- Provide usage examples for all commands
- Document requirements and environment variables"
```

---

## Task 13: Final Testing and Bug Fixes

**Files:**
- Various (as needed based on testing)

**Step 1: Build final binary**

Run: `go build -o packnplay .`

Expected: Clean build

**Step 2: Test basic functionality**

Test plan:
1. `./packnplay --help` - verify help output
2. `./packnplay run --help` - verify run command help
3. Create a test directory with git repo
4. Try `./packnplay run --no-worktree echo "hello"`
5. Check for any runtime errors

**Step 3: Fix any bugs discovered**

Address issues found during testing. Common issues might be:
- Import errors
- Missing error checks
- Path handling bugs
- Docker command parsing issues

**Step 4: Run go mod tidy**

Run: `go mod tidy`

**Step 5: Final commit**

```bash
git add .
git commit -m "test: verify packnplay functionality and fix bugs

- Build and test all commands
- Fix any runtime issues discovered
- Clean up dependencies with go mod tidy
- Verify basic workflows function correctly"
```

---

## Execution Notes

- Follow TDD principles where practical (integration testing is challenging here)
- Commit frequently after each working unit
- Test manually as you go since this is CLI tooling
- Use `--verbose` flag for debugging during development
- Remember: DRY, YAGNI, small commits

## Post-Implementation

After completing all tasks:
1. Test with real Claude Code installation
2. Verify worktree creation and management
3. Test dev container discovery
4. Verify idmap mounts work correctly
5. Test all commands (run, attach, stop, list)
