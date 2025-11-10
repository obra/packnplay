package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// skipIfNoDocker skips the test if Docker daemon is not available
// or if running in short mode (go test -short)
func skipIfNoDocker(t *testing.T) {
	t.Helper()

	// Skip if running in short mode
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Check if Docker is available
	if !isDockerAvailable() {
		t.Skip("Docker daemon not available - skipping E2E test")
	}
}

// isDockerAvailable checks if Docker daemon is available
func isDockerAvailable() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", "info")
	return cmd.Run() == nil
}

// createTestProject creates a temporary test project with the given files
// Returns the absolute path to the project directory
func createTestProject(t *testing.T, files map[string]string) string {
	t.Helper()

	// Create temp directory
	projectDir, err := os.MkdirTemp("", "packnplay-e2e-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}

	// Create all files
	for relPath, content := range files {
		fullPath := filepath.Join(projectDir, relPath)

		// Create parent directory if needed
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			os.RemoveAll(projectDir)
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}

		// Write file
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			os.RemoveAll(projectDir)
			t.Fatalf("Failed to write file %s: %v", fullPath, err)
		}
	}

	return projectDir
}

// cleanupContainer removes a container by name
// Uses docker rm -f for fast, forceful removal (kills and removes in one step)
// This is appropriate for test cleanup where graceful shutdown is not required
func cleanupContainer(t *testing.T, containerName string) {
	t.Helper()

	// Use shorter timeout since we're using -f flag for immediate kill
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Use docker rm -f to kill and remove in one operation
	// This is much faster than docker stop (which waits up to 10s for SIGTERM)
	// followed by docker rm. For test cleanup, graceful shutdown is not needed.
	removeCmd := exec.CommandContext(ctx, "docker", "rm", "-f", containerName)
	if err := removeCmd.Run(); err != nil {
		// Only log if the error is not "no such container"
		if !strings.Contains(err.Error(), "No such container") {
			t.Logf("Warning: Failed to remove container %s: %v", containerName, err)
		}
	}
}

// waitForContainer waits for a container to be in running state
func waitForContainer(t *testing.T, containerName string, timeout time.Duration) error {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for container %s to start", containerName)
		case <-ticker.C:
			cmd := exec.CommandContext(ctx, "docker", "inspect", "--format", "{{.State.Running}}", containerName)
			output, err := cmd.Output()
			if err != nil {
				continue // Container might not exist yet
			}

			if strings.TrimSpace(string(output)) == "true" {
				return nil
			}
		}
	}
}

// execInContainer executes a command in a running container
// Returns the combined stdout and stderr output
func execInContainer(t *testing.T, containerName string, cmd []string) (string, error) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	args := append([]string{"exec", containerName}, cmd...)
	execCmd := exec.CommandContext(ctx, "docker", args...)
	output, err := execCmd.CombinedOutput()
	return string(output), err
}

// inspectContainer returns the full inspect output for a container
func inspectContainer(t *testing.T, containerName string) (map[string]interface{}, error) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", "inspect", containerName)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to inspect container: %w", err)
	}

	var inspectData []map[string]interface{}
	if err := json.Unmarshal(output, &inspectData); err != nil {
		return nil, fmt.Errorf("failed to parse inspect output: %w", err)
	}

	if len(inspectData) == 0 {
		return nil, fmt.Errorf("no inspect data returned for container %s", containerName)
	}

	return inspectData[0], nil
}

// getPacknplayBinary returns the path to the packnplay binary
// It builds it if necessary and caches the path
var packnplayBinaryPath string

func getPacknplayBinary(t *testing.T) string {
	t.Helper()

	// Return cached path if available
	if packnplayBinaryPath != "" {
		return packnplayBinaryPath
	}

	// Try to find packnplay in PATH
	if path, err := exec.LookPath("packnplay"); err == nil {
		packnplayBinaryPath = path
		return path
	}

	// Build packnplay to temp location
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	binaryPath := filepath.Join(os.TempDir(), fmt.Sprintf("packnplay-test-%d", os.Getpid()))

	// Get project root (assumes we're in pkg/runner/)
	projectRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("Failed to get project root: %v", err)
	}

	t.Logf("Building packnplay binary to %s...", binaryPath)
	buildCmd := exec.CommandContext(ctx, "go", "build", "-o", binaryPath, ".")
	buildCmd.Dir = projectRoot
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build packnplay: %v\nOutput: %s", err, output)
	}

	packnplayBinaryPath = binaryPath
	return binaryPath
}

// cleanupMetadata removes metadata files for a container
func cleanupMetadata(t *testing.T, containerID string) {
	t.Helper()

	// Metadata is stored at ~/.local/share/packnplay/metadata/<container-id>.json
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Logf("Warning: Failed to get home directory: %v", err)
		return
	}

	metadataDir := filepath.Join(homeDir, ".local", "share", "packnplay", "metadata")
	metadataFile := filepath.Join(metadataDir, containerID+".json")

	if err := os.RemoveAll(metadataFile); err != nil {
		t.Logf("Warning: Failed to remove metadata file %s: %v", metadataFile, err)
	}
}

// runPacknplay executes packnplay with the given arguments
func runPacknplay(t *testing.T, args ...string) (string, error) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	binary := getPacknplayBinary(t)
	cmd := exec.CommandContext(ctx, binary, args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// runPacknplayInDir changes to directory and runs packnplay
// This is needed because packnplay doesn't have a --project flag
func runPacknplayInDir(t *testing.T, dir string, args ...string) (string, error) {
	t.Helper()

	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Failed to chdir to %s: %v", dir, err)
	}
	defer os.Chdir(oldwd)

	return runPacknplay(t, args...)
}

// getContainerIDByName returns the container ID for a given container name
func getContainerIDByName(t *testing.T, containerName string) string {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", "inspect", "--format", "{{.Id}}", containerName)
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(output))
}

// getContainerNameForProject calculates the container name for a project directory
// This matches the naming logic used by packnplay: packnplay-{projectName}-no-worktree
func getContainerNameForProject(projectDir string) string {
	projectName := filepath.Base(projectDir)
	return fmt.Sprintf("packnplay-%s-no-worktree", projectName)
}

// readMetadata reads the metadata file for a container and returns it as a map
func readMetadata(t *testing.T, containerID string) map[string]interface{} {
	t.Helper()

	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("Failed to get home directory: %v", err)
	}

	metadataPath := filepath.Join(homeDir, ".local/share/packnplay/metadata", containerID+".json")

	data, err := os.ReadFile(metadataPath)
	if err != nil {
		return nil // Metadata doesn't exist yet
	}

	var metadata map[string]interface{}
	err = json.Unmarshal(data, &metadata)
	if err != nil {
		t.Fatalf("Failed to parse metadata JSON: %v", err)
	}

	return metadata
}

// parseLineCount parses the output from wc -l command and returns the line count
func parseLineCount(output string) int {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		// Look for number at start of line (from wc -l)
		fields := strings.Fields(line)
		if len(fields) > 0 {
			if count, err := strconv.Atoi(fields[0]); err == nil {
				return count
			}
		}
	}
	return 0
}

// TestE2E_Infrastructure tests the test helper infrastructure itself
func TestE2E_Infrastructure(t *testing.T) {
	skipIfNoDocker(t)

	t.Run("Docker availability check", func(t *testing.T) {
		if !isDockerAvailable() {
			t.Fatal("Docker should be available but isDockerAvailable() returned false")
		}
	})

	t.Run("Create test project", func(t *testing.T) {
		projectDir := createTestProject(t, map[string]string{
			"test.txt":                        "hello world",
			".devcontainer/devcontainer.json": `{"image": "alpine:latest"}`,
			"nested/dir/file.txt":             "nested content",
		})
		defer os.RemoveAll(projectDir)

		// Verify project directory exists
		if _, err := os.Stat(projectDir); os.IsNotExist(err) {
			t.Fatal("Project directory was not created")
		}

		// Verify files exist
		testFile := filepath.Join(projectDir, "test.txt")
		content, err := os.ReadFile(testFile)
		if err != nil {
			t.Fatalf("Failed to read test.txt: %v", err)
		}
		if string(content) != "hello world" {
			t.Errorf("test.txt content = %q, want %q", string(content), "hello world")
		}

		// Verify nested directory
		nestedFile := filepath.Join(projectDir, "nested/dir/file.txt")
		nestedContent, err := os.ReadFile(nestedFile)
		if err != nil {
			t.Fatalf("Failed to read nested/dir/file.txt: %v", err)
		}
		if string(nestedContent) != "nested content" {
			t.Errorf("nested/dir/file.txt content = %q, want %q", string(nestedContent), "nested content")
		}
	})

	t.Run("Container cleanup", func(t *testing.T) {
		// Create a test container
		containerName := fmt.Sprintf("packnplay-e2e-cleanup-%d", time.Now().UnixNano())

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Run a container that sleeps
		cmd := exec.CommandContext(ctx, "docker", "run",
			"-d",
			"--name", containerName,
			"--label", "managed-by=packnplay-e2e",
			"alpine:latest",
			"sleep", "3600",
		)
		if err := cmd.Run(); err != nil {
			t.Fatalf("Failed to create test container: %v", err)
		}

		// Verify container exists
		checkCmd := exec.CommandContext(ctx, "docker", "ps", "-q", "--filter", fmt.Sprintf("name=^%s$", containerName))
		output, err := checkCmd.Output()
		if err != nil {
			t.Fatalf("Failed to check for container: %v", err)
		}
		if len(strings.TrimSpace(string(output))) == 0 {
			t.Fatal("Container was not created")
		}

		// Clean up container
		cleanupContainer(t, containerName)

		// Verify container is removed
		checkCmd2 := exec.CommandContext(ctx, "docker", "ps", "-aq", "--filter", fmt.Sprintf("name=^%s$", containerName))
		output2, err := checkCmd2.Output()
		if err != nil {
			t.Fatalf("Failed to check for container after cleanup: %v", err)
		}
		if len(strings.TrimSpace(string(output2))) != 0 {
			t.Error("Container was not cleaned up properly")
		}
	})

	t.Run("Wait for container", func(t *testing.T) {
		containerName := fmt.Sprintf("packnplay-e2e-wait-%d", time.Now().UnixNano())
		defer cleanupContainer(t, containerName)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Start container
		cmd := exec.CommandContext(ctx, "docker", "run",
			"-d",
			"--name", containerName,
			"--label", "managed-by=packnplay-e2e",
			"alpine:latest",
			"sleep", "3600",
		)
		if err := cmd.Run(); err != nil {
			t.Fatalf("Failed to start container: %v", err)
		}

		// Wait for container to be running
		if err := waitForContainer(t, containerName, 10*time.Second); err != nil {
			t.Fatalf("waitForContainer failed: %v", err)
		}
	})

	t.Run("Exec in container", func(t *testing.T) {
		containerName := fmt.Sprintf("packnplay-e2e-exec-%d", time.Now().UnixNano())
		defer cleanupContainer(t, containerName)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Start container
		cmd := exec.CommandContext(ctx, "docker", "run",
			"-d",
			"--name", containerName,
			"--label", "managed-by=packnplay-e2e",
			"alpine:latest",
			"sleep", "3600",
		)
		if err := cmd.Run(); err != nil {
			t.Fatalf("Failed to start container: %v", err)
		}

		// Wait for container
		if err := waitForContainer(t, containerName, 10*time.Second); err != nil {
			t.Fatalf("Container failed to start: %v", err)
		}

		// Execute command
		output, err := execInContainer(t, containerName, []string{"echo", "hello from container"})
		if err != nil {
			t.Fatalf("execInContainer failed: %v", err)
		}

		expected := "hello from container\n"
		if output != expected {
			t.Errorf("execInContainer output = %q, want %q", output, expected)
		}
	})

	t.Run("Inspect container", func(t *testing.T) {
		containerName := fmt.Sprintf("packnplay-e2e-inspect-%d", time.Now().UnixNano())
		defer cleanupContainer(t, containerName)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Start container with specific label
		cmd := exec.CommandContext(ctx, "docker", "run",
			"-d",
			"--name", containerName,
			"--label", "managed-by=packnplay-e2e",
			"--label", "test-label=test-value",
			"alpine:latest",
			"sleep", "3600",
		)
		if err := cmd.Run(); err != nil {
			t.Fatalf("Failed to start container: %v", err)
		}

		// Wait for container
		if err := waitForContainer(t, containerName, 10*time.Second); err != nil {
			t.Fatalf("Container failed to start: %v", err)
		}

		// Inspect container
		inspect, err := inspectContainer(t, containerName)
		if err != nil {
			t.Fatalf("inspectContainer failed: %v", err)
		}

		// Verify inspect data contains expected fields
		if inspect["Name"] == nil {
			t.Error("Inspect data missing 'Name' field")
		}

		// Verify labels
		config, ok := inspect["Config"].(map[string]interface{})
		if !ok {
			t.Fatal("Inspect data missing 'Config' field")
		}

		labels, ok := config["Labels"].(map[string]interface{})
		if !ok {
			t.Fatal("Inspect data missing 'Config.Labels' field")
		}

		if labels["managed-by"] != "packnplay-e2e" {
			t.Errorf("Label 'managed-by' = %v, want 'packnplay-e2e'", labels["managed-by"])
		}
	})

	t.Run("packnplay binary available", func(t *testing.T) {
		binary := getPacknplayBinary(t)
		if binary == "" {
			t.Fatal("Should be able to locate or build packnplay binary")
		}

		// Verify binary is executable
		info, err := os.Stat(binary)
		if err != nil {
			t.Fatalf("Failed to stat binary: %v", err)
		}
		if info.Mode()&0111 == 0 {
			t.Fatal("Binary should be executable")
		}

		// Verify binary responds to --help
		cmd := exec.Command(binary, "--help")
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Binary should respond to --help: %v\nOutput: %s", err, output)
		}
		if !strings.Contains(string(output), "packnplay") {
			t.Errorf("Help output should mention packnplay, got: %s", output)
		}
	})
}

// TestE2E_BasicImagePull tests pulling an alpine image and running a simple command
func TestE2E_BasicImagePull(t *testing.T) {
	skipIfNoDocker(t)

	// Create test project
	projectDir := createTestProject(t, map[string]string{
		".devcontainer/devcontainer.json": `{"image": "alpine:latest"}`,
	})
	defer os.RemoveAll(projectDir)

	containerName := fmt.Sprintf("packnplay-e2e-basic-%d", time.Now().UnixNano())
	defer cleanupContainer(t, containerName)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Pull alpine image if not already present
	t.Log("Pulling alpine:latest image...")
	pullCmd := exec.CommandContext(ctx, "docker", "pull", "alpine:latest")
	if output, err := pullCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to pull alpine:latest: %v\nOutput: %s", err, output)
	}

	// Verify image was pulled
	t.Log("Verifying alpine:latest image exists...")
	imagesCmd := exec.CommandContext(ctx, "docker", "images", "alpine:latest", "-q")
	imageOutput, err := imagesCmd.Output()
	if err != nil {
		t.Fatalf("Failed to list images: %v", err)
	}
	if len(strings.TrimSpace(string(imageOutput))) == 0 {
		t.Fatal("alpine:latest image not found after pull")
	}

	// Create and run a container from the image
	t.Logf("Creating container %s...", containerName)
	runCmd := exec.CommandContext(ctx, "docker", "run",
		"-d",
		"--name", containerName,
		"--label", "managed-by=packnplay-e2e",
		"alpine:latest",
		"sh", "-c", "echo 'hello from alpine' && sleep 3600",
	)
	if output, err := runCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to run container: %v\nOutput: %s", err, output)
	}

	// Wait for container to be running
	t.Log("Waiting for container to start...")
	if err := waitForContainer(t, containerName, 30*time.Second); err != nil {
		t.Fatalf("Container failed to start: %v", err)
	}

	// Inspect container to verify it's using alpine:latest
	t.Log("Inspecting container...")
	inspect, err := inspectContainer(t, containerName)
	if err != nil {
		t.Fatalf("Failed to inspect container: %v", err)
	}

	// Verify image
	config, ok := inspect["Config"].(map[string]interface{})
	if !ok {
		t.Fatal("Inspect data missing 'Config' field")
	}

	image, ok := config["Image"].(string)
	if !ok {
		t.Fatal("Inspect data missing 'Config.Image' field")
	}

	// Image might be alpine:latest or the full sha256
	if !strings.Contains(image, "alpine") && image != "alpine:latest" {
		t.Errorf("Container image = %q, expected to contain 'alpine' or be 'alpine:latest'", image)
	}

	// Execute a command in the container
	t.Log("Executing command in container...")
	output, err := execInContainer(t, containerName, []string{"echo", "test successful"})
	if err != nil {
		t.Fatalf("Failed to execute command in container: %v", err)
	}

	expected := "test successful\n"
	if output != expected {
		t.Errorf("Command output = %q, want %q", output, expected)
	}

	// Verify container has the correct label
	labels, ok := config["Labels"].(map[string]interface{})
	if !ok {
		t.Fatal("Inspect data missing 'Config.Labels' field")
	}

	if labels["managed-by"] != "packnplay-e2e" {
		t.Errorf("Label 'managed-by' = %v, want 'packnplay-e2e'", labels["managed-by"])
	}

	t.Log("Test completed successfully!")
}

// ============================================================================
// Section 2.1: Image Tests
// ============================================================================

// TestE2E_ImagePull tests pulling and using a pre-built image
func TestE2E_ImagePull(t *testing.T) {
	skipIfNoDocker(t)

	projectDir := createTestProject(t, map[string]string{
		".devcontainer/devcontainer.json": `{"image": "alpine:latest"}`,
	})
	defer os.RemoveAll(projectDir)

	containerName := getContainerNameForProject(projectDir)
	defer cleanupContainer(t, containerName)
	defer func() {
		containerID := getContainerIDByName(t, containerName)
		if containerID != "" {
			cleanupMetadata(t, containerID)
		}
	}()

	output, err := runPacknplayInDir(t, projectDir, "run", "--no-worktree", "echo", "image test success")
	require.NoError(t, err, "Failed to run packnplay: %s", output)
	require.Contains(t, output, "image test success")
}

// TestE2E_ImageAlreadyExists tests that packnplay skips pull if image exists locally
func TestE2E_ImageAlreadyExists(t *testing.T) {
	skipIfNoDocker(t)

	// Pre-pull the image
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	pullCmd := exec.CommandContext(ctx, "docker", "pull", "alpine:latest")
	require.NoError(t, pullCmd.Run(), "Failed to pre-pull alpine:latest")

	projectDir := createTestProject(t, map[string]string{
		".devcontainer/devcontainer.json": `{"image": "alpine:latest"}`,
	})
	defer os.RemoveAll(projectDir)

	containerName := getContainerNameForProject(projectDir)
	defer cleanupContainer(t, containerName)
	defer func() {
		containerID := getContainerIDByName(t, containerName)
		if containerID != "" {
			cleanupMetadata(t, containerID)
		}
	}()

	output, err := runPacknplayInDir(t, projectDir, "run", "--no-worktree", "echo", "using cached image")
	require.NoError(t, err, "Failed to run packnplay: %s", output)
	require.Contains(t, output, "using cached image")
}

// ============================================================================
// Section 2.2: Dockerfile Tests
// ============================================================================

// TestE2E_DockerfileBuild tests building from a Dockerfile
func TestE2E_DockerfileBuild(t *testing.T) {
	skipIfNoDocker(t)

	projectDir := createTestProject(t, map[string]string{
		".devcontainer/Dockerfile": `FROM alpine:latest
RUN echo "custom-marker" > /custom-marker.txt
RUN echo "built successfully" > /build-success.txt`,
		".devcontainer/devcontainer.json": `{"dockerfile": "Dockerfile"}`,
	})
	defer os.RemoveAll(projectDir)

	containerName := getContainerNameForProject(projectDir)
	defer cleanupContainer(t, containerName)
	defer func() {
		containerID := getContainerIDByName(t, containerName)
		if containerID != "" {
			cleanupMetadata(t, containerID)
		}
	}()

	output, err := runPacknplayInDir(t, projectDir, "run", "--no-worktree", "cat", "/custom-marker.txt")
	require.NoError(t, err, "Failed to run packnplay: %s", output)
	require.Contains(t, output, "custom-marker")
}

// TestE2E_DockerfileInDevcontainer tests Dockerfile in .devcontainer/
func TestE2E_DockerfileInDevcontainer(t *testing.T) {
	skipIfNoDocker(t)

	projectDir := createTestProject(t, map[string]string{
		".devcontainer/Dockerfile.dev": `FROM alpine:latest
RUN echo "devcontainer-build" > /devcontainer-marker.txt`,
		".devcontainer/devcontainer.json": `{"dockerfile": "Dockerfile.dev"}`,
	})
	defer os.RemoveAll(projectDir)

	containerName := getContainerNameForProject(projectDir)
	defer cleanupContainer(t, containerName)
	defer func() {
		containerID := getContainerIDByName(t, containerName)
		if containerID != "" {
			cleanupMetadata(t, containerID)
		}
	}()

	output, err := runPacknplayInDir(t, projectDir, "run", "--no-worktree", "cat", "/devcontainer-marker.txt")
	require.NoError(t, err, "Failed to run packnplay: %s", output)
	require.Contains(t, output, "devcontainer-build")
}

// ============================================================================
// Section 2.3: Build Config Tests
// ============================================================================

// TestE2E_BuildWithArgs tests build args substitution
func TestE2E_BuildWithArgs(t *testing.T) {
	skipIfNoDocker(t)

	projectDir := createTestProject(t, map[string]string{
		".devcontainer/Dockerfile": `ARG TEST_ARG=default
FROM alpine:latest
ARG TEST_ARG
RUN echo "arg value: ${TEST_ARG}" > /arg-test.txt`,
		".devcontainer/devcontainer.json": `{
  "build": {
    "dockerfile": "Dockerfile",
    "args": {
      "TEST_ARG": "custom_value"
    }
  }
}`,
	})
	defer os.RemoveAll(projectDir)

	containerName := getContainerNameForProject(projectDir)
	defer cleanupContainer(t, containerName)
	defer func() {
		containerID := getContainerIDByName(t, containerName)
		if containerID != "" {
			cleanupMetadata(t, containerID)
		}
	}()

	output, err := runPacknplayInDir(t, projectDir, "run", "--no-worktree", "cat", "/arg-test.txt")
	require.NoError(t, err, "Failed to run packnplay: %s", output)
	require.Contains(t, output, "custom_value")
}

// TestE2E_BuildWithTarget tests multi-stage build target
func TestE2E_BuildWithTarget(t *testing.T) {
	skipIfNoDocker(t)

	projectDir := createTestProject(t, map[string]string{
		".devcontainer/Dockerfile": `FROM alpine:latest AS base
RUN echo "base stage" > /stage.txt

FROM base AS development
RUN echo "development stage" > /stage.txt

FROM base AS production
RUN echo "production stage" > /stage.txt`,
		".devcontainer/devcontainer.json": `{
  "build": {
    "dockerfile": "Dockerfile",
    "target": "development"
  }
}`,
	})
	defer os.RemoveAll(projectDir)

	containerName := getContainerNameForProject(projectDir)
	defer cleanupContainer(t, containerName)
	defer func() {
		containerID := getContainerIDByName(t, containerName)
		if containerID != "" {
			cleanupMetadata(t, containerID)
		}
	}()

	output, err := runPacknplayInDir(t, projectDir, "run", "--no-worktree", "cat", "/stage.txt")
	require.NoError(t, err, "Failed to run packnplay: %s", output)
	require.Contains(t, output, "development stage")
}

// TestE2E_BuildWithContext tests build context outside .devcontainer
func TestE2E_BuildWithContext(t *testing.T) {
	skipIfNoDocker(t)

	projectDir := createTestProject(t, map[string]string{
		"shared-file.txt": "shared content from parent",
		".devcontainer/Dockerfile": `FROM alpine:latest
COPY shared-file.txt /shared.txt`,
		".devcontainer/devcontainer.json": `{
  "build": {
    "dockerfile": "Dockerfile",
    "context": ".."
  }
}`,
	})
	defer os.RemoveAll(projectDir)

	containerName := getContainerNameForProject(projectDir)
	defer cleanupContainer(t, containerName)
	defer func() {
		containerID := getContainerIDByName(t, containerName)
		if containerID != "" {
			cleanupMetadata(t, containerID)
		}
	}()

	output, err := runPacknplayInDir(t, projectDir, "run", "--no-worktree", "cat", "/shared.txt")
	require.NoError(t, err, "Failed to run packnplay: %s", output)
	require.Contains(t, output, "shared content from parent")
}

// ============================================================================
// Section 2.4: Environment Variable Tests
// ============================================================================

// TestE2E_ContainerEnv tests containerEnv sets environment variables
func TestE2E_ContainerEnv(t *testing.T) {
	skipIfNoDocker(t)

	projectDir := createTestProject(t, map[string]string{
		".devcontainer/devcontainer.json": `{
  "image": "alpine:latest",
  "containerEnv": {
    "TEST_VAR": "test_value",
    "ANOTHER_VAR": "another_value"
  }
}`,
	})
	defer os.RemoveAll(projectDir)

	containerName := getContainerNameForProject(projectDir)
	defer cleanupContainer(t, containerName)
	defer func() {
		containerID := getContainerIDByName(t, containerName)
		if containerID != "" {
			cleanupMetadata(t, containerID)
		}
	}()

	output, err := runPacknplayInDir(t, projectDir, "run", "--no-worktree", "sh", "-c", "echo $TEST_VAR")
	require.NoError(t, err, "Failed to run packnplay: %s", output)
	require.Contains(t, output, "test_value")
}

// TestE2E_RemoteEnv tests remoteEnv with references
func TestE2E_RemoteEnv(t *testing.T) {
	skipIfNoDocker(t)

	projectDir := createTestProject(t, map[string]string{
		".devcontainer/devcontainer.json": `{
  "image": "alpine:latest",
  "containerEnv": {
    "BASE_URL": "https://api.example.com"
  },
  "remoteEnv": {
    "API_ENDPOINT": "${containerEnv:BASE_URL}/v1"
  }
}`,
	})
	defer os.RemoveAll(projectDir)

	containerName := getContainerNameForProject(projectDir)
	defer cleanupContainer(t, containerName)
	defer func() {
		containerID := getContainerIDByName(t, containerName)
		if containerID != "" {
			cleanupMetadata(t, containerID)
		}
	}()

	output, err := runPacknplayInDir(t, projectDir, "run", "--no-worktree", "sh", "-c", "echo $API_ENDPOINT")
	require.NoError(t, err, "Failed to run packnplay: %s", output)
	require.Contains(t, output, "https://api.example.com/v1")
}

// TestE2E_EnvPriority tests CLI --env overrides devcontainer
func TestE2E_EnvPriority(t *testing.T) {
	skipIfNoDocker(t)

	projectDir := createTestProject(t, map[string]string{
		".devcontainer/devcontainer.json": `{
  "image": "alpine:latest",
  "containerEnv": {
    "TEST_VAR": "devcontainer_value"
  }
}`,
	})
	defer os.RemoveAll(projectDir)

	containerName := getContainerNameForProject(projectDir)
	defer cleanupContainer(t, containerName)
	defer func() {
		containerID := getContainerIDByName(t, containerName)
		if containerID != "" {
			cleanupMetadata(t, containerID)
		}
	}()

	output, err := runPacknplayInDir(t, projectDir, "run", "--no-worktree", "--env", "TEST_VAR=cli_override", "sh", "-c", "echo $TEST_VAR")
	require.NoError(t, err, "Failed to run packnplay: %s", output)
	require.Contains(t, output, "cli_override")
}

// ============================================================================
// Section 2.5: Variable Substitution Tests
// ============================================================================

// TestE2E_LocalEnvSubstitution tests ${localEnv:VAR}
func TestE2E_LocalEnvSubstitution(t *testing.T) {
	skipIfNoDocker(t)

	// Set local environment variable
	os.Setenv("TEST_LOCAL_VAR", "local_value_123")
	defer os.Unsetenv("TEST_LOCAL_VAR")

	projectDir := createTestProject(t, map[string]string{
		".devcontainer/devcontainer.json": `{
  "image": "alpine:latest",
  "containerEnv": {
    "MY_VAR": "${localEnv:TEST_LOCAL_VAR}"
  }
}`,
	})
	defer os.RemoveAll(projectDir)

	containerName := getContainerNameForProject(projectDir)
	defer cleanupContainer(t, containerName)
	defer func() {
		containerID := getContainerIDByName(t, containerName)
		if containerID != "" {
			cleanupMetadata(t, containerID)
		}
	}()

	output, err := runPacknplayInDir(t, projectDir, "run", "--no-worktree", "sh", "-c", "echo $MY_VAR")
	require.NoError(t, err, "Failed to run packnplay: %s", output)
	require.Contains(t, output, "local_value_123")
}

// TestE2E_WorkspaceVariables tests ${localWorkspaceFolder} and ${containerWorkspaceFolder}
func TestE2E_WorkspaceVariables(t *testing.T) {
	skipIfNoDocker(t)

	projectDir := createTestProject(t, map[string]string{
		".devcontainer/devcontainer.json": `{
  "image": "alpine:latest",
  "containerEnv": {
    "PROJECT_NAME": "${localWorkspaceFolderBasename}",
    "CONTAINER_WS": "${containerWorkspaceFolder}"
  }
}`,
	})
	defer os.RemoveAll(projectDir)

	containerName := getContainerNameForProject(projectDir)
	defer cleanupContainer(t, containerName)
	defer func() {
		containerID := getContainerIDByName(t, containerName)
		if containerID != "" {
			cleanupMetadata(t, containerID)
		}
	}()

	output, err := runPacknplayInDir(t, projectDir, "run", "--no-worktree", "sh", "-c", "echo $PROJECT_NAME")
	require.NoError(t, err, "Failed to run packnplay: %s", output)

	// Should contain the base name of the temp directory
	assert.NotEmpty(t, strings.TrimSpace(output), "Expected project name from workspace folder basename")
}

// TestE2E_DefaultValues tests ${localEnv:VAR:default}
func TestE2E_DefaultValues(t *testing.T) {
	skipIfNoDocker(t)

	// Make sure variable doesn't exist
	os.Unsetenv("NONEXISTENT_VAR_12345")

	projectDir := createTestProject(t, map[string]string{
		".devcontainer/devcontainer.json": `{
  "image": "alpine:latest",
  "containerEnv": {
    "MY_VAR": "${localEnv:NONEXISTENT_VAR_12345:default_value}"
  }
}`,
	})
	defer os.RemoveAll(projectDir)

	containerName := getContainerNameForProject(projectDir)
	defer cleanupContainer(t, containerName)
	defer func() {
		containerID := getContainerIDByName(t, containerName)
		if containerID != "" {
			cleanupMetadata(t, containerID)
		}
	}()

	output, err := runPacknplayInDir(t, projectDir, "run", "--no-worktree", "sh", "-c", "echo $MY_VAR")
	require.NoError(t, err, "Failed to run packnplay: %s", output)
	require.Contains(t, output, "default_value")
}

// ============================================================================
// Section 2.6: Port Forwarding Tests
// ============================================================================

// TestE2E_PortForwarding tests basic port mapping
func TestE2E_PortForwarding(t *testing.T) {
	skipIfNoDocker(t)

	projectDir := createTestProject(t, map[string]string{
		".devcontainer/devcontainer.json": `{
  "image": "alpine:latest",
  "forwardPorts": [33001, 33002]
}`,
	})
	defer os.RemoveAll(projectDir)

	containerName := getContainerNameForProject(projectDir)
	defer cleanupContainer(t, containerName)
	defer func() {
		containerID := getContainerIDByName(t, containerName)
		if containerID != "" {
			cleanupMetadata(t, containerID)
		}
	}()

	// Start container (runs sleep infinity in background)
	output, err := runPacknplayInDir(t, projectDir, "run", "--no-worktree", "echo", "started")
	require.NoError(t, err, "Failed to start: %s", output)

	// Container is running - verify ports
	portOut, err := exec.Command("docker", "port", containerName, "33001").CombinedOutput()
	require.NoError(t, err, "docker port should work on running container: %s", portOut)
	require.Contains(t, string(portOut), ":33001")

	portOut2, err := exec.Command("docker", "port", containerName, "33002").CombinedOutput()
	require.NoError(t, err, "docker port should work on running container: %s", portOut2)
	require.Contains(t, string(portOut2), ":33002")
}

// TestE2E_PortFormats tests different port format types
func TestE2E_PortFormats(t *testing.T) {
	skipIfNoDocker(t)

	projectDir := createTestProject(t, map[string]string{
		".devcontainer/devcontainer.json": `{
  "image": "alpine:latest",
  "forwardPorts": [33003, "33004:33005", "127.0.0.1:33006:33006"]
}`,
	})
	defer os.RemoveAll(projectDir)

	containerName := getContainerNameForProject(projectDir)
	defer cleanupContainer(t, containerName)
	defer func() {
		containerID := getContainerIDByName(t, containerName)
		if containerID != "" {
			cleanupMetadata(t, containerID)
		}
	}()

	// Start container (runs sleep infinity in background)
	output, err := runPacknplayInDir(t, projectDir, "run", "--no-worktree", "echo", "multiple port formats")
	require.NoError(t, err, "Failed to start: %s", output)

	// Verify integer format (33003 -> 33003:33003)
	portOutput33003, err := exec.Command("docker", "port", containerName, "33003").CombinedOutput()
	require.NoError(t, err, "docker port should work on running container: %s", portOutput33003)
	require.Contains(t, string(portOutput33003), ":33003")

	// Verify string format ("33004:33005" means host:33004 -> container:33005)
	portOutput33005, err := exec.Command("docker", "port", containerName, "33005").CombinedOutput()
	require.NoError(t, err, "docker port should work on running container: %s", portOutput33005)
	require.Contains(t, string(portOutput33005), ":33004")

	// Verify IP binding format ("127.0.0.1:33006:33006")
	portOutput33006, err := exec.Command("docker", "port", containerName, "33006").CombinedOutput()
	require.NoError(t, err, "docker port should work on running container: %s", portOutput33006)
	require.Contains(t, string(portOutput33006), "127.0.0.1:33006")
}

// ============================================================================
// Section 2.7: Lifecycle Command Tests (CRITICAL!)
// ============================================================================

// TestE2E_OnCreateCommand_RunsOnce tests that onCreate runs only once
func TestE2E_OnCreateCommand_RunsOnce(t *testing.T) {
	skipIfNoDocker(t)

	projectDir := createTestProject(t, map[string]string{
		".devcontainer/devcontainer.json": `{
  "image": "alpine:latest",
  "onCreateCommand": "echo 'onCreate executed' > /tmp/onCreate-ran.txt"
}`,
	})
	defer os.RemoveAll(projectDir)

	containerName := getContainerNameForProject(projectDir)
	defer cleanupContainer(t, containerName)

	// First run - creates container with sleep infinity, runs onCreate
	output1, err := runPacknplayInDir(t, projectDir, "run", "--no-worktree", "cat", "/tmp/onCreate-ran.txt")
	require.NoError(t, err, "First run failed: %s", output1)
	require.Contains(t, output1, "onCreate executed")

	// Container is still running - get its ID
	containerID := getContainerIDByName(t, containerName)
	require.NotEmpty(t, containerID, "Container should exist")
	defer cleanupMetadata(t, containerID)

	// Verify metadata shows onCreate executed
	metadata := readMetadata(t, containerID)
	require.NotNil(t, metadata, "Metadata should exist")

	lifecycleRan, ok := metadata["lifecycleRan"].(map[string]interface{})
	require.True(t, ok, "Should have lifecycleRan")

	onCreate, ok := lifecycleRan["onCreate"].(map[string]interface{})
	require.True(t, ok, "Should have onCreate")
	require.True(t, onCreate["executed"].(bool), "onCreate should be marked executed")

	firstHash := onCreate["commandHash"].(string)
	require.NotEmpty(t, firstHash, "Should have command hash")

	// Second run - use --reconnect to exec into SAME container
	output2, err := runPacknplayInDir(t, projectDir, "run", "--no-worktree", "--reconnect", "cat", "/tmp/onCreate-ran.txt")
	require.NoError(t, err, "Second run failed: %s", output2)
	require.Contains(t, output2, "onCreate executed", "File should still exist")

	// Verify onCreate didn't run again (hash unchanged)
	metadata2 := readMetadata(t, containerID)
	onCreate2 := metadata2["lifecycleRan"].(map[string]interface{})["onCreate"].(map[string]interface{})
	secondHash := onCreate2["commandHash"].(string)
	require.Equal(t, firstHash, secondHash, "Hash should not change (onCreate didn't re-run)")
}

// TestE2E_PostCreateCommand_RunsOnce tests that postCreate runs only once
func TestE2E_PostCreateCommand_RunsOnce(t *testing.T) {
	skipIfNoDocker(t)

	projectDir := createTestProject(t, map[string]string{
		".devcontainer/devcontainer.json": `{
  "image": "alpine:latest",
  "postCreateCommand": "echo 'postCreate executed' > /tmp/postCreate-ran.txt"
}`,
	})
	defer os.RemoveAll(projectDir)

	containerName := getContainerNameForProject(projectDir)
	defer cleanupContainer(t, containerName)

	// First run
	output1, err := runPacknplayInDir(t, projectDir, "run", "--no-worktree", "cat", "/tmp/postCreate-ran.txt")
	require.NoError(t, err, "First run failed: %s", output1)
	require.Contains(t, output1, "postCreate executed")

	// Verify metadata was created and postCreate was tracked
	containerID := getContainerIDByName(t, containerName)
	require.NotEmpty(t, containerID, "Container should exist")
	defer cleanupMetadata(t, containerID)

	metadata := readMetadata(t, containerID)
	require.NotNil(t, metadata, "Metadata should exist")

	lifecycleRan, ok := metadata["lifecycleRan"].(map[string]interface{})
	require.True(t, ok, "Should have lifecycleRan")

	postCreate, ok := lifecycleRan["postCreate"].(map[string]interface{})
	require.True(t, ok, "Should have postCreate")
	require.True(t, postCreate["executed"].(bool), "postCreate should be marked executed")

	firstHash := postCreate["commandHash"].(string)
	require.NotEmpty(t, firstHash, "Should have command hash")

	// Second run - use --reconnect, postCreate should NOT execute again
	output2, err := runPacknplayInDir(t, projectDir, "run", "--no-worktree", "--reconnect", "cat", "/tmp/postCreate-ran.txt")
	require.NoError(t, err, "Second run failed: %s", output2)
	require.Contains(t, output2, "postCreate executed", "File should persist")

	// Verify postCreate didn't run again (hash unchanged)
	metadata2 := readMetadata(t, containerID)
	postCreate2 := metadata2["lifecycleRan"].(map[string]interface{})["postCreate"].(map[string]interface{})
	secondHash := postCreate2["commandHash"].(string)
	require.Equal(t, firstHash, secondHash, "Hash should not change (postCreate didn't re-run)")
}

// TestE2E_PostStartCommand_RunsEveryTime tests that postStart runs every time
func TestE2E_PostStartCommand_RunsEveryTime(t *testing.T) {
	skipIfNoDocker(t)

	projectDir := createTestProject(t, map[string]string{
		".devcontainer/devcontainer.json": `{
  "image": "alpine:latest",
  "postStartCommand": "date >> /tmp/postStart-runs.txt"
}`,
	})
	defer os.RemoveAll(projectDir)

	containerName := getContainerNameForProject(projectDir)
	defer cleanupContainer(t, containerName)
	defer func() {
		containerID := getContainerIDByName(t, containerName)
		if containerID != "" {
			cleanupMetadata(t, containerID)
		}
	}()

	// First run
	output1, err := runPacknplayInDir(t, projectDir, "run", "--no-worktree", "wc", "-l", "/tmp/postStart-runs.txt")
	require.NoError(t, err, "First run failed: %s", output1)

	count1 := parseLineCount(output1)
	require.GreaterOrEqual(t, count1, 1, "First run should have at least one line")

	// Second run - use --reconnect, postStart should run again and append
	output2, err := runPacknplayInDir(t, projectDir, "run", "--no-worktree", "--reconnect", "wc", "-l", "/tmp/postStart-runs.txt")
	require.NoError(t, err, "Second run failed: %s", output2)

	count2 := parseLineCount(output2)
	require.Greater(t, count2, count1, "postStart should run every time, count should increase")

	// Third run - verify postStart continues to run
	output3, err := runPacknplayInDir(t, projectDir, "run", "--no-worktree", "--reconnect", "wc", "-l", "/tmp/postStart-runs.txt")
	require.NoError(t, err, "Third run failed: %s", output3)

	count3 := parseLineCount(output3)
	require.Greater(t, count3, count2, "postStart should run on third time too")

	t.Logf("postStart ran successfully: run1=%d lines, run2=%d lines, run3=%d lines", count1, count2, count3)
}

// TestE2E_CommandFormatString tests string command with shell features
func TestE2E_CommandFormatString(t *testing.T) {
	skipIfNoDocker(t)

	projectDir := createTestProject(t, map[string]string{
		".devcontainer/devcontainer.json": `{
  "image": "alpine:latest",
  "onCreateCommand": "echo 'part1' > /tmp/test.txt && echo 'part2' >> /tmp/test.txt"
}`,
	})
	defer os.RemoveAll(projectDir)

	containerName := getContainerNameForProject(projectDir)
	defer cleanupContainer(t, containerName)
	defer func() {
		containerID := getContainerIDByName(t, containerName)
		if containerID != "" {
			cleanupMetadata(t, containerID)
		}
	}()

	output, err := runPacknplayInDir(t, projectDir, "run", "--no-worktree", "cat", "/tmp/test.txt")
	require.NoError(t, err, "Failed to run: %s", output)
	require.Contains(t, output, "part1")
	require.Contains(t, output, "part2")
}

// TestE2E_CommandFormatArray tests array command format (direct exec)
func TestE2E_CommandFormatArray(t *testing.T) {
	skipIfNoDocker(t)

	projectDir := createTestProject(t, map[string]string{
		".devcontainer/devcontainer.json": `{
  "image": "alpine:latest",
  "onCreateCommand": ["sh", "-c", "echo 'array format' > /tmp/array-test.txt"]
}`,
	})
	defer os.RemoveAll(projectDir)

	containerName := getContainerNameForProject(projectDir)
	defer cleanupContainer(t, containerName)
	defer func() {
		containerID := getContainerIDByName(t, containerName)
		if containerID != "" {
			cleanupMetadata(t, containerID)
		}
	}()

	output, err := runPacknplayInDir(t, projectDir, "run", "--no-worktree", "cat", "/tmp/array-test.txt")
	require.NoError(t, err, "Failed to run: %s", output)
	require.Contains(t, output, "array format")
}

// TestE2E_CommandFormatObject tests object format with parallel tasks
func TestE2E_CommandFormatObject(t *testing.T) {
	skipIfNoDocker(t)

	projectDir := createTestProject(t, map[string]string{
		".devcontainer/devcontainer.json": `{
  "image": "alpine:latest",
  "onCreateCommand": {
    "task1": "echo 'task1' > /tmp/task1.txt",
    "task2": "echo 'task2' > /tmp/task2.txt",
    "task3": "echo 'task3' > /tmp/task3.txt"
  }
}`,
	})
	defer os.RemoveAll(projectDir)

	containerName := getContainerNameForProject(projectDir)
	defer cleanupContainer(t, containerName)
	defer func() {
		containerID := getContainerIDByName(t, containerName)
		if containerID != "" {
			cleanupMetadata(t, containerID)
		}
	}()

	// Run and verify all tasks executed
	output1, err := runPacknplayInDir(t, projectDir, "run", "--no-worktree", "cat", "/tmp/task1.txt")
	require.NoError(t, err, "Failed to read task1: %s", output1)
	require.Contains(t, output1, "task1")

	// Use --reconnect for subsequent runs
	output2, err := runPacknplayInDir(t, projectDir, "run", "--no-worktree", "--reconnect", "cat", "/tmp/task2.txt")
	require.NoError(t, err, "Failed to read task2: %s", output2)
	require.Contains(t, output2, "task2")

	output3, err := runPacknplayInDir(t, projectDir, "run", "--no-worktree", "--reconnect", "cat", "/tmp/task3.txt")
	require.NoError(t, err, "Failed to read task3: %s", output3)
	require.Contains(t, output3, "task3")
}

// TestE2E_CommandChangeDetection tests re-execution when command changes
func TestE2E_CommandChangeDetection(t *testing.T) {
	skipIfNoDocker(t)

	projectDir := createTestProject(t, map[string]string{
		".devcontainer/devcontainer.json": `{
  "image": "alpine:latest",
  "onCreateCommand": "echo 'version1' > /tmp/version.txt"
}`,
	})
	defer os.RemoveAll(projectDir)

	containerName := getContainerNameForProject(projectDir)
	defer cleanupContainer(t, containerName)

	// First run with version1
	output1, err := runPacknplayInDir(t, projectDir, "run", "--no-worktree", "cat", "/tmp/version.txt")
	require.NoError(t, err, "First run failed: %s", output1)
	require.Contains(t, output1, "version1")

	// Verify metadata was created with first command hash
	containerID := getContainerIDByName(t, containerName)
	require.NotEmpty(t, containerID, "Container should exist")
	defer cleanupMetadata(t, containerID)

	metadata := readMetadata(t, containerID)
	require.NotNil(t, metadata, "Metadata should exist")

	lifecycleRan := metadata["lifecycleRan"].(map[string]interface{})
	onCreate := lifecycleRan["onCreate"].(map[string]interface{})
	commandHash1 := onCreate["commandHash"].(string)
	require.NotEmpty(t, commandHash1, "Should have command hash")

	// Stop and remove container to test re-creation with changed command
	cleanupContainer(t, containerName)

	// Modify the devcontainer.json with different command
	newConfig := `{
  "image": "alpine:latest",
  "onCreateCommand": "echo 'version2' > /tmp/version.txt"
}`
	configPath := filepath.Join(projectDir, ".devcontainer", "devcontainer.json")
	require.NoError(t, os.WriteFile(configPath, []byte(newConfig), 0644))

	// Second run with changed command - should create new container and re-execute
	output2, err := runPacknplayInDir(t, projectDir, "run", "--no-worktree", "cat", "/tmp/version.txt")
	require.NoError(t, err, "Second run failed: %s", output2)

	// Get new container ID
	containerID2 := getContainerIDByName(t, containerName)
	require.NotEmpty(t, containerID2, "New container should exist")
	defer cleanupMetadata(t, containerID2)

	// Verify metadata was updated with new command hash
	metadata2 := readMetadata(t, containerID2)
	require.NotNil(t, metadata2, "Metadata should exist")

	onCreate2 := metadata2["lifecycleRan"].(map[string]interface{})["onCreate"].(map[string]interface{})
	commandHash2 := onCreate2["commandHash"].(string)
	require.NotEmpty(t, commandHash2, "Should have command hash")

	// CRITICAL: Verify command hash changed when command content changed
	require.NotEqual(t, commandHash1, commandHash2, "Command hash should change when command content changes")

	// CRITICAL: Verify command re-executed with new content
	require.Contains(t, output2, "version2", "Command should re-execute with new content")
}

// ============================================================================
// Section 2.8: User Detection Tests
// ============================================================================

// TestE2E_RemoteUser tests respecting remoteUser setting
func TestE2E_RemoteUser(t *testing.T) {
	skipIfNoDocker(t)

	projectDir := createTestProject(t, map[string]string{
		".devcontainer/devcontainer.json": `{
  "image": "alpine:latest",
  "remoteUser": "nobody"
}`,
	})
	defer os.RemoveAll(projectDir)

	containerName := getContainerNameForProject(projectDir)
	defer cleanupContainer(t, containerName)
	defer func() {
		containerID := getContainerIDByName(t, containerName)
		if containerID != "" {
			cleanupMetadata(t, containerID)
		}
	}()

	output, err := runPacknplayInDir(t, projectDir, "run", "--no-worktree", "whoami")
	require.NoError(t, err, "Failed to run whoami: %s", output)
	require.Contains(t, output, "nobody")
}

// TestE2E_UserAutoDetection tests auto-detection when not specified
func TestE2E_UserAutoDetection(t *testing.T) {
	skipIfNoDocker(t)

	projectDir := createTestProject(t, map[string]string{
		".devcontainer/devcontainer.json": `{
  "image": "alpine:latest"
}`,
	})
	defer os.RemoveAll(projectDir)

	containerName := getContainerNameForProject(projectDir)
	defer cleanupContainer(t, containerName)
	defer func() {
		containerID := getContainerIDByName(t, containerName)
		if containerID != "" {
			cleanupMetadata(t, containerID)
		}
	}()

	output, err := runPacknplayInDir(t, projectDir, "run", "--no-worktree", "whoami")
	require.NoError(t, err, "Failed to run whoami: %s", output)

	// Should return some user (root or auto-detected)
	assert.NotEmpty(t, strings.TrimSpace(output), "Expected a username from auto-detection")
	t.Logf("Auto-detected user: %s", strings.TrimSpace(output))
}

// ============================================================================
// Section 2.9: Integration Tests
// ============================================================================

// TestE2E_FullDevcontainer tests all features together
func TestE2E_FullDevcontainer(t *testing.T) {
	skipIfNoDocker(t)

	// Set local env for substitution
	os.Setenv("FULL_TEST_VAR", "from_local_env")
	defer os.Unsetenv("FULL_TEST_VAR")

	projectDir := createTestProject(t, map[string]string{
		".devcontainer/Dockerfile": `FROM alpine:latest
RUN echo "custom image" > /custom.txt`,
		".devcontainer/devcontainer.json": `{
  "build": {
    "dockerfile": "Dockerfile"
  },
  "containerEnv": {
    "BASE_VAR": "base_value",
    "LOCAL_VAR": "${localEnv:FULL_TEST_VAR}"
  },
  "remoteEnv": {
    "DERIVED_VAR": "${containerEnv:BASE_VAR}_derived"
  },
  "forwardPorts": [33007],
  "onCreateCommand": "echo 'setup complete' > /tmp/setup.txt",
  "remoteUser": "nobody"
}`,
	})
	defer os.RemoveAll(projectDir)

	containerName := getContainerNameForProject(projectDir)
	defer cleanupContainer(t, containerName)
	defer func() {
		containerID := getContainerIDByName(t, containerName)
		if containerID != "" {
			cleanupMetadata(t, containerID)
		}
	}()

	// Test 1: Verify custom build
	output1, err := runPacknplayInDir(t, projectDir, "run", "--no-worktree", "cat", "/custom.txt")
	require.NoError(t, err, "Failed to verify custom build: %s", output1)
	require.Contains(t, output1, "custom image")

	// Test 2: Verify environment variables (use --reconnect)
	output2, err := runPacknplayInDir(t, projectDir, "run", "--no-worktree", "--reconnect", "sh", "-c", "echo $BASE_VAR $LOCAL_VAR $DERIVED_VAR")
	require.NoError(t, err, "Failed to verify env vars: %s", output2)
	require.Contains(t, output2, "base_value")
	require.Contains(t, output2, "from_local_env")
	require.Contains(t, output2, "base_value_derived")

	// Test 3: Verify onCreate ran (use --reconnect)
	output3, err := runPacknplayInDir(t, projectDir, "run", "--no-worktree", "--reconnect", "cat", "/tmp/setup.txt")
	require.NoError(t, err, "Failed to verify onCreate: %s", output3)
	require.Contains(t, output3, "setup complete")

	// Test 4: Verify user (use --reconnect)
	output4, err := runPacknplayInDir(t, projectDir, "run", "--no-worktree", "--reconnect", "whoami")
	require.NoError(t, err, "Failed to verify user: %s", output4)
	require.Contains(t, output4, "nobody")

	t.Log("Full integration test passed!")
}

// TestE2E_RealWorldNodeJS tests a realistic Node.js project setup
func TestE2E_RealWorldNodeJS(t *testing.T) {
	skipIfNoDocker(t)

	projectDir := createTestProject(t, map[string]string{
		"package.json": `{
  "name": "test-app",
  "version": "1.0.0",
  "scripts": {
    "test": "echo 'tests passed'"
  }
}`,
		".devcontainer/devcontainer.json": `{
  "image": "node:18-alpine",
  "containerEnv": {
    "NODE_ENV": "development"
  },
  "forwardPorts": [33008],
  "onCreateCommand": "npm --version > /tmp/npm-version.txt",
  "postCreateCommand": "echo 'dependencies installed' > /tmp/deps.txt"
}`,
	})
	defer os.RemoveAll(projectDir)

	containerName := getContainerNameForProject(projectDir)
	defer cleanupContainer(t, containerName)
	defer func() {
		containerID := getContainerIDByName(t, containerName)
		if containerID != "" {
			cleanupMetadata(t, containerID)
		}
	}()

	// Test 1: Verify Node.js environment
	output1, err := runPacknplayInDir(t, projectDir, "run", "--no-worktree", "node", "--version")
	require.NoError(t, err, "Failed to run node: %s", output1)
	require.Contains(t, output1, "v18")

	// Test 2: Verify environment variable (use --reconnect)
	output2, err := runPacknplayInDir(t, projectDir, "run", "--no-worktree", "--reconnect", "sh", "-c", "echo $NODE_ENV")
	require.NoError(t, err, "Failed to check NODE_ENV: %s", output2)
	require.Contains(t, output2, "development")

	// Test 3: Verify onCreate ran (npm version check) (use --reconnect)
	output3, err := runPacknplayInDir(t, projectDir, "run", "--no-worktree", "--reconnect", "cat", "/tmp/npm-version.txt")
	require.NoError(t, err, "Failed to verify onCreate: %s", output3)
	// Should contain npm version number
	assert.NotEmpty(t, strings.TrimSpace(output3), "onCreate command (npm --version) should produce output")

	// Test 4: Verify postCreate ran (use --reconnect)
	output4, err := runPacknplayInDir(t, projectDir, "run", "--no-worktree", "--reconnect", "cat", "/tmp/deps.txt")
	require.NoError(t, err, "Failed to verify postCreate: %s", output4)
	require.Contains(t, output4, "dependencies installed")

	// Test 5: Verify package.json is accessible (use --reconnect)
	output5, err := runPacknplayInDir(t, projectDir, "run", "--no-worktree", "--reconnect", "cat", "package.json")
	require.NoError(t, err, "Failed to read package.json: %s", output5)
	require.Contains(t, output5, "test-app")

	t.Log("Real-world Node.js test passed!")
}
