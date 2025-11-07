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
func cleanupContainer(t *testing.T, containerName string) {
	t.Helper()

	// Check if container exists
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	checkCmd := exec.CommandContext(ctx, "docker", "ps", "-aq", "--filter", fmt.Sprintf("name=^%s$", containerName))
	output, err := checkCmd.Output()
	if err != nil {
		t.Logf("Warning: Failed to check if container %s exists: %v", containerName, err)
		return
	}

	// If container doesn't exist, nothing to clean up
	if len(strings.TrimSpace(string(output))) == 0 {
		return
	}

	// Stop and remove container
	stopCmd := exec.CommandContext(ctx, "docker", "stop", containerName)
	if err := stopCmd.Run(); err != nil {
		t.Logf("Warning: Failed to stop container %s: %v", containerName, err)
	}

	removeCmd := exec.CommandContext(ctx, "docker", "rm", "-f", containerName)
	if err := removeCmd.Run(); err != nil {
		t.Logf("Warning: Failed to remove container %s: %v", containerName, err)
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

	// Note: We can't easily control container name with packnplay run
	// So we'll verify by checking logs and output
	output, err := runPacknplay(t, "run", "--project", projectDir, "echo", "image test success")
	if err != nil {
		t.Fatalf("Failed to run packnplay: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(output, "image test success") {
		t.Errorf("Expected output to contain 'image test success', got: %s", output)
	}
}

// TestE2E_ImageAlreadyExists tests that packnplay skips pull if image exists locally
func TestE2E_ImageAlreadyExists(t *testing.T) {
	skipIfNoDocker(t)

	// Pre-pull the image
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	pullCmd := exec.CommandContext(ctx, "docker", "pull", "alpine:latest")
	if err := pullCmd.Run(); err != nil {
		t.Fatalf("Failed to pre-pull alpine:latest: %v", err)
	}

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

	output, err := runPacknplay(t, "run", "--project", projectDir, "echo", "using cached image")
	if err != nil {
		t.Fatalf("Failed to run packnplay: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(output, "using cached image") {
		t.Errorf("Expected output to contain 'using cached image', got: %s", output)
	}
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

	output, err := runPacknplay(t, "run", "--project", projectDir, "cat", "/custom-marker.txt")
	if err != nil {
		t.Fatalf("Failed to run packnplay: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(output, "custom-marker") {
		t.Errorf("Expected custom marker file content, got: %s", output)
	}
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

	output, err := runPacknplay(t, "run", "--project", projectDir, "cat", "/devcontainer-marker.txt")
	if err != nil {
		t.Fatalf("Failed to run packnplay: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(output, "devcontainer-build") {
		t.Errorf("Expected devcontainer build marker, got: %s", output)
	}
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

	output, err := runPacknplay(t, "run", "--project", projectDir, "cat", "/arg-test.txt")
	if err != nil {
		t.Fatalf("Failed to run packnplay: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(output, "custom_value") {
		t.Errorf("Expected build arg value 'custom_value', got: %s", output)
	}
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

	output, err := runPacknplay(t, "run", "--project", projectDir, "cat", "/stage.txt")
	if err != nil {
		t.Fatalf("Failed to run packnplay: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(output, "development stage") {
		t.Errorf("Expected development stage marker, got: %s", output)
	}
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

	output, err := runPacknplay(t, "run", "--project", projectDir, "cat", "/shared.txt")
	if err != nil {
		t.Fatalf("Failed to run packnplay: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(output, "shared content from parent") {
		t.Errorf("Expected shared file content, got: %s", output)
	}
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

	output, err := runPacknplay(t, "run", "--project", projectDir, "sh", "-c", "echo $TEST_VAR")
	if err != nil {
		t.Fatalf("Failed to run packnplay: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(output, "test_value") {
		t.Errorf("Expected TEST_VAR=test_value, got: %s", output)
	}
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

	output, err := runPacknplay(t, "run", "--project", projectDir, "sh", "-c", "echo $API_ENDPOINT")
	if err != nil {
		t.Fatalf("Failed to run packnplay: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(output, "https://api.example.com/v1") {
		t.Errorf("Expected API_ENDPOINT with substitution, got: %s", output)
	}
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

	output, err := runPacknplay(t, "run", "--project", projectDir, "--env", "TEST_VAR=cli_override", "sh", "-c", "echo $TEST_VAR")
	if err != nil {
		t.Fatalf("Failed to run packnplay: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(output, "cli_override") {
		t.Errorf("Expected CLI override value, got: %s", output)
	}
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

	output, err := runPacknplay(t, "run", "--project", projectDir, "sh", "-c", "echo $MY_VAR")
	if err != nil {
		t.Fatalf("Failed to run packnplay: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(output, "local_value_123") {
		t.Errorf("Expected local env substitution, got: %s", output)
	}
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

	output, err := runPacknplay(t, "run", "--project", projectDir, "sh", "-c", "echo $PROJECT_NAME")
	if err != nil {
		t.Fatalf("Failed to run packnplay: %v\nOutput: %s", err, output)
	}

	// Should contain the base name of the temp directory
	if len(strings.TrimSpace(output)) == 0 {
		t.Errorf("Expected project name from workspace folder basename, got empty")
	}
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

	output, err := runPacknplay(t, "run", "--project", projectDir, "sh", "-c", "echo $MY_VAR")
	if err != nil {
		t.Fatalf("Failed to run packnplay: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(output, "default_value") {
		t.Errorf("Expected default value, got: %s", output)
	}
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
  "forwardPorts": [3000, 8080]
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

	// Run in background with a simple HTTP server simulation
	// Note: alpine doesn't have nc, so we'll just verify the container config
	output, err := runPacknplay(t, "run", "--project", projectDir, "echo", "port test")
	if err != nil {
		t.Fatalf("Failed to run packnplay: %v\nOutput: %s", err, output)
	}

	// For a proper port test, we'd need to inspect the running container
	// This is a simplified test that verifies the command succeeds
	if !strings.Contains(output, "port test") {
		t.Errorf("Expected output from container, got: %s", output)
	}

	// Verify port 3000 is mapped
	portOutput, err := exec.Command("docker", "port", containerName, "3000").CombinedOutput()
	if err == nil {
		if !strings.Contains(string(portOutput), ":3000") {
			t.Errorf("Port 3000 should be mapped to host, got: %s", portOutput)
		}
	} else {
		t.Logf("Note: docker port command failed (may need running container): %v", err)
	}

	// Verify port 8080 is mapped
	portOutput8080, err := exec.Command("docker", "port", containerName, "8080").CombinedOutput()
	if err == nil {
		if !strings.Contains(string(portOutput8080), ":8080") {
			t.Errorf("Port 8080 should be mapped to host, got: %s", portOutput8080)
		}
	} else {
		t.Logf("Note: docker port command failed (may need running container): %v", err)
	}
}

// TestE2E_PortFormats tests different port format types
func TestE2E_PortFormats(t *testing.T) {
	skipIfNoDocker(t)

	projectDir := createTestProject(t, map[string]string{
		".devcontainer/devcontainer.json": `{
  "image": "alpine:latest",
  "forwardPorts": [3000, "8080:80", "127.0.0.1:9000:9000"]
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

	output, err := runPacknplay(t, "run", "--project", projectDir, "echo", "multiple port formats")
	if err != nil {
		t.Fatalf("Failed to run packnplay: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(output, "multiple port formats") {
		t.Errorf("Expected output from container, got: %s", output)
	}

	// Verify integer format (3000)
	portOutput3000, err := exec.Command("docker", "port", containerName, "3000").CombinedOutput()
	if err == nil {
		if !strings.Contains(string(portOutput3000), ":3000") {
			t.Errorf("Port 3000 (integer format) should be mapped to host, got: %s", portOutput3000)
		}
	} else {
		t.Logf("Note: docker port command failed for port 3000 (may need running container): %v", err)
	}

	// Verify string format ("8080:80")
	portOutput80, err := exec.Command("docker", "port", containerName, "80").CombinedOutput()
	if err == nil {
		if !strings.Contains(string(portOutput80), ":8080") {
			t.Errorf("Port 80 (string format 8080:80) should be mapped to host port 8080, got: %s", portOutput80)
		}
	} else {
		t.Logf("Note: docker port command failed for port 80 (may need running container): %v", err)
	}

	// Verify IP binding format ("127.0.0.1:9000:9000")
	portOutput9000, err := exec.Command("docker", "port", containerName, "9000").CombinedOutput()
	if err == nil {
		if !strings.Contains(string(portOutput9000), "127.0.0.1:9000") {
			t.Errorf("Port 9000 should be mapped to 127.0.0.1:9000, got: %s", portOutput9000)
		}
	} else {
		t.Logf("Note: docker port command failed for port 9000 (may need running container): %v", err)
	}
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
	defer func() {
		containerID := getContainerIDByName(t, containerName)
		if containerID != "" {
			cleanupMetadata(t, containerID)
		}
	}()

	// First run - onCreate should execute
	output1, err := runPacknplay(t, "run", "--project", projectDir, "cat", "/tmp/onCreate-ran.txt")
	if err != nil {
		t.Fatalf("First run failed: %v\nOutput: %s", err, output1)
	}

	if !strings.Contains(output1, "onCreate executed") {
		t.Errorf("onCreate should have created file on first run, got: %s", output1)
	}

	// Verify metadata was created and onCreate was tracked
	containerID := getContainerIDByName(t, containerName)
	if containerID == "" {
		t.Fatal("Container ID should be available after first run")
	}

	metadata := readMetadata(t, containerID)
	if metadata == nil {
		t.Fatal("Metadata should exist after first run")
	}

	lifecycleRan, ok := metadata["lifecycleRan"].(map[string]interface{})
	if !ok {
		t.Fatal("Metadata should have lifecycleRan field")
	}

	onCreate, ok := lifecycleRan["onCreate"].(map[string]interface{})
	if !ok {
		t.Fatal("lifecycleRan should have onCreate field")
	}

	if executed, ok := onCreate["executed"].(bool); !ok || !executed {
		t.Error("onCreate should be marked as executed in metadata")
	}

	commandHash, ok := onCreate["commandHash"].(string)
	if !ok || commandHash == "" {
		t.Error("onCreate should have a command hash in metadata")
	}

	// Second run - onCreate should NOT execute again
	// We need to run with same container/project
	output2, err := runPacknplay(t, "run", "--project", projectDir, "test", "-f", "/tmp/onCreate-ran.txt")
	if err != nil {
		// If file doesn't exist, onCreate might have run again (bad)
		t.Logf("Second run output: %s", output2)
	}

	// File should still exist from first run
	output3, err := runPacknplay(t, "run", "--project", projectDir, "cat", "/tmp/onCreate-ran.txt")
	if err != nil {
		t.Fatalf("Failed to read onCreate file on second run: %v\nOutput: %s", err, output3)
	}

	if !strings.Contains(output3, "onCreate executed") {
		t.Errorf("onCreate file should persist from first run, got: %s", output3)
	}

	// Verify onCreate didn't run again by checking metadata hash is unchanged
	metadata2 := readMetadata(t, containerID)
	if metadata2 == nil {
		t.Fatal("Metadata should still exist after second run")
	}

	lifecycleRan2, ok := metadata2["lifecycleRan"].(map[string]interface{})
	if !ok {
		t.Fatal("Metadata should still have lifecycleRan field")
	}

	onCreate2, ok := lifecycleRan2["onCreate"].(map[string]interface{})
	if !ok {
		t.Fatal("lifecycleRan should still have onCreate field")
	}

	commandHash2, ok := onCreate2["commandHash"].(string)
	if !ok {
		t.Error("onCreate should still have command hash")
	}

	if commandHash != commandHash2 {
		t.Error("onCreate command hash should not change between runs")
	}
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
	defer func() {
		containerID := getContainerIDByName(t, containerName)
		if containerID != "" {
			cleanupMetadata(t, containerID)
		}
	}()

	// First run
	output1, err := runPacknplay(t, "run", "--project", projectDir, "cat", "/tmp/postCreate-ran.txt")
	if err != nil {
		t.Fatalf("First run failed: %v\nOutput: %s", err, output1)
	}

	if !strings.Contains(output1, "postCreate executed") {
		t.Errorf("postCreate should have created file on first run, got: %s", output1)
	}

	// Verify metadata was created and postCreate was tracked
	containerID := getContainerIDByName(t, containerName)
	if containerID == "" {
		t.Fatal("Container ID should be available after first run")
	}

	metadata := readMetadata(t, containerID)
	if metadata == nil {
		t.Fatal("Metadata should exist after first run")
	}

	lifecycleRan, ok := metadata["lifecycleRan"].(map[string]interface{})
	if !ok {
		t.Fatal("Metadata should have lifecycleRan field")
	}

	postCreate, ok := lifecycleRan["postCreate"].(map[string]interface{})
	if !ok {
		t.Fatal("lifecycleRan should have postCreate field")
	}

	if executed, ok := postCreate["executed"].(bool); !ok || !executed {
		t.Error("postCreate should be marked as executed in metadata")
	}

	commandHash, ok := postCreate["commandHash"].(string)
	if !ok || commandHash == "" {
		t.Error("postCreate should have a command hash in metadata")
	}

	// Second run - postCreate should NOT execute again
	output2, err := runPacknplay(t, "run", "--project", projectDir, "cat", "/tmp/postCreate-ran.txt")
	if err != nil {
		t.Fatalf("Second run failed: %v\nOutput: %s", err, output2)
	}

	if !strings.Contains(output2, "postCreate executed") {
		t.Errorf("postCreate file should persist from first run, got: %s", output2)
	}

	// Verify postCreate didn't run again by checking metadata hash is unchanged
	metadata2 := readMetadata(t, containerID)
	if metadata2 == nil {
		t.Fatal("Metadata should still exist after second run")
	}

	lifecycleRan2, ok := metadata2["lifecycleRan"].(map[string]interface{})
	if !ok {
		t.Fatal("Metadata should still have lifecycleRan field")
	}

	postCreate2, ok := lifecycleRan2["postCreate"].(map[string]interface{})
	if !ok {
		t.Fatal("lifecycleRan should still have postCreate field")
	}

	commandHash2, ok := postCreate2["commandHash"].(string)
	if !ok {
		t.Error("postCreate should still have command hash")
	}

	if commandHash != commandHash2 {
		t.Error("postCreate command hash should not change between runs")
	}
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
	output1, err := runPacknplay(t, "run", "--project", projectDir, "wc", "-l", "/tmp/postStart-runs.txt")
	if err != nil {
		t.Fatalf("First run failed: %v\nOutput: %s", err, output1)
	}

	// Parse and verify first run line count
	count1 := parseLineCount(output1)
	if count1 < 1 {
		t.Errorf("First run should have at least one line, got: %d (output: %s)", count1, output1)
	}

	// Second run - postStart should run again and append
	output2, err := runPacknplay(t, "run", "--project", projectDir, "wc", "-l", "/tmp/postStart-runs.txt")
	if err != nil {
		t.Fatalf("Second run failed: %v\nOutput: %s", err, output2)
	}

	// Parse and verify second run line count increased
	count2 := parseLineCount(output2)
	if count2 <= count1 {
		t.Errorf("postStart should run every time, count should increase. First: %d, Second: %d (output2: %s)", count1, count2, output2)
	}

	// Third run - verify postStart continues to run
	output3, err := runPacknplay(t, "run", "--project", projectDir, "wc", "-l", "/tmp/postStart-runs.txt")
	if err != nil {
		t.Fatalf("Third run failed: %v\nOutput: %s", err, output3)
	}

	// Parse and verify third run line count increased again
	count3 := parseLineCount(output3)
	if count3 <= count2 {
		t.Errorf("postStart should run on third time too, count should increase. Second: %d, Third: %d (output3: %s)", count2, count3, output3)
	}

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

	output, err := runPacknplay(t, "run", "--project", projectDir, "cat", "/tmp/test.txt")
	if err != nil {
		t.Fatalf("Failed to run: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(output, "part1") || !strings.Contains(output, "part2") {
		t.Errorf("Shell command with && should execute both parts, got: %s", output)
	}
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

	output, err := runPacknplay(t, "run", "--project", projectDir, "cat", "/tmp/array-test.txt")
	if err != nil {
		t.Fatalf("Failed to run: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(output, "array format") {
		t.Errorf("Array command format should work, got: %s", output)
	}
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
	output1, err := runPacknplay(t, "run", "--project", projectDir, "cat", "/tmp/task1.txt")
	if err != nil {
		t.Fatalf("Failed to read task1: %v\nOutput: %s", err, output1)
	}

	output2, err := runPacknplay(t, "run", "--project", projectDir, "cat", "/tmp/task2.txt")
	if err != nil {
		t.Fatalf("Failed to read task2: %v\nOutput: %s", err, output2)
	}

	output3, err := runPacknplay(t, "run", "--project", projectDir, "cat", "/tmp/task3.txt")
	if err != nil {
		t.Fatalf("Failed to read task3: %v\nOutput: %s", err, output3)
	}

	if !strings.Contains(output1, "task1") {
		t.Errorf("Task1 should have executed, got: %s", output1)
	}
	if !strings.Contains(output2, "task2") {
		t.Errorf("Task2 should have executed, got: %s", output2)
	}
	if !strings.Contains(output3, "task3") {
		t.Errorf("Task3 should have executed, got: %s", output3)
	}
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
	defer func() {
		containerID := getContainerIDByName(t, containerName)
		if containerID != "" {
			cleanupMetadata(t, containerID)
		}
	}()

	// First run with version1
	output1, err := runPacknplay(t, "run", "--project", projectDir, "cat", "/tmp/version.txt")
	if err != nil {
		t.Fatalf("First run failed: %v\nOutput: %s", err, output1)
	}

	if !strings.Contains(output1, "version1") {
		t.Errorf("First version should be 'version1', got: %s", output1)
	}

	// Verify metadata was created with first command hash
	containerID := getContainerIDByName(t, containerName)
	if containerID == "" {
		t.Fatal("Container ID should be available after first run")
	}

	metadata := readMetadata(t, containerID)
	if metadata == nil {
		t.Fatal("Metadata should exist after first run")
	}

	lifecycleRan, ok := metadata["lifecycleRan"].(map[string]interface{})
	if !ok {
		t.Fatal("Metadata should have lifecycleRan field")
	}

	onCreate, ok := lifecycleRan["onCreate"].(map[string]interface{})
	if !ok {
		t.Fatal("lifecycleRan should have onCreate field")
	}

	commandHash1, ok := onCreate["commandHash"].(string)
	if !ok || commandHash1 == "" {
		t.Fatal("onCreate should have a command hash in metadata")
	}

	// Modify the devcontainer.json with different command
	newConfig := `{
  "image": "alpine:latest",
  "onCreateCommand": "echo 'version2' > /tmp/version.txt"
}`
	configPath := filepath.Join(projectDir, ".devcontainer", "devcontainer.json")
	if err := os.WriteFile(configPath, []byte(newConfig), 0644); err != nil {
		t.Fatalf("Failed to update devcontainer.json: %v", err)
	}

	// Second run with changed command - should re-execute
	output2, err := runPacknplay(t, "run", "--project", projectDir, "cat", "/tmp/version.txt")
	if err != nil {
		t.Fatalf("Second run failed: %v\nOutput: %s", err, output2)
	}

	// Verify metadata was updated with new command hash
	metadata2 := readMetadata(t, containerID)
	if metadata2 == nil {
		t.Fatal("Metadata should exist after second run")
	}

	lifecycleRan2, ok := metadata2["lifecycleRan"].(map[string]interface{})
	if !ok {
		t.Fatal("Metadata should have lifecycleRan field after second run")
	}

	onCreate2, ok := lifecycleRan2["onCreate"].(map[string]interface{})
	if !ok {
		t.Fatal("lifecycleRan should have onCreate field after second run")
	}

	commandHash2, ok := onCreate2["commandHash"].(string)
	if !ok || commandHash2 == "" {
		t.Fatal("onCreate should have a command hash in metadata after second run")
	}

	// CRITICAL: Verify command hash changed when command content changed
	if commandHash1 == commandHash2 {
		t.Errorf("Command hash should change when command content changes, but both are: %s", commandHash1)
	}

	// CRITICAL: Verify command re-executed with new content
	if !strings.Contains(output2, "version2") {
		t.Errorf("Command should re-execute with new content when hash changes. Expected 'version2', got: %s", output2)
	}
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

	output, err := runPacknplay(t, "run", "--project", projectDir, "whoami")
	if err != nil {
		t.Fatalf("Failed to run whoami: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(output, "nobody") {
		t.Errorf("Expected user 'nobody', got: %s", output)
	}
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

	output, err := runPacknplay(t, "run", "--project", projectDir, "whoami")
	if err != nil {
		t.Fatalf("Failed to run whoami: %v\nOutput: %s", err, output)
	}

	// Should return some user (root or auto-detected)
	if len(strings.TrimSpace(output)) == 0 {
		t.Errorf("Expected a username from auto-detection, got empty")
	}
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
  "forwardPorts": [3000],
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
	output1, err := runPacknplay(t, "run", "--project", projectDir, "cat", "/custom.txt")
	if err != nil {
		t.Fatalf("Failed to verify custom build: %v\nOutput: %s", err, output1)
	}
	if !strings.Contains(output1, "custom image") {
		t.Errorf("Custom build failed, got: %s", output1)
	}

	// Test 2: Verify environment variables
	output2, err := runPacknplay(t, "run", "--project", projectDir, "sh", "-c", "echo $BASE_VAR $LOCAL_VAR $DERIVED_VAR")
	if err != nil {
		t.Fatalf("Failed to verify env vars: %v\nOutput: %s", err, output2)
	}
	if !strings.Contains(output2, "base_value") {
		t.Errorf("BASE_VAR not set correctly, got: %s", output2)
	}
	if !strings.Contains(output2, "from_local_env") {
		t.Errorf("LOCAL_VAR substitution failed, got: %s", output2)
	}
	if !strings.Contains(output2, "base_value_derived") {
		t.Errorf("DERIVED_VAR not set correctly, got: %s", output2)
	}

	// Test 3: Verify onCreate ran
	output3, err := runPacknplay(t, "run", "--project", projectDir, "cat", "/tmp/setup.txt")
	if err != nil {
		t.Fatalf("Failed to verify onCreate: %v\nOutput: %s", err, output3)
	}
	if !strings.Contains(output3, "setup complete") {
		t.Errorf("onCreate command failed, got: %s", output3)
	}

	// Test 4: Verify user
	output4, err := runPacknplay(t, "run", "--project", projectDir, "whoami")
	if err != nil {
		t.Fatalf("Failed to verify user: %v\nOutput: %s", err, output4)
	}
	if !strings.Contains(output4, "nobody") {
		t.Errorf("remoteUser not set correctly, got: %s", output4)
	}

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
  "forwardPorts": [3000],
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
	output1, err := runPacknplay(t, "run", "--project", projectDir, "node", "--version")
	if err != nil {
		t.Fatalf("Failed to run node: %v\nOutput: %s", err, output1)
	}
	if !strings.Contains(output1, "v18") {
		t.Errorf("Expected Node v18, got: %s", output1)
	}

	// Test 2: Verify environment variable
	output2, err := runPacknplay(t, "run", "--project", projectDir, "sh", "-c", "echo $NODE_ENV")
	if err != nil {
		t.Fatalf("Failed to check NODE_ENV: %v\nOutput: %s", err, output2)
	}
	if !strings.Contains(output2, "development") {
		t.Errorf("NODE_ENV not set correctly, got: %s", output2)
	}

	// Test 3: Verify onCreate ran (npm version check)
	output3, err := runPacknplay(t, "run", "--project", projectDir, "cat", "/tmp/npm-version.txt")
	if err != nil {
		t.Fatalf("Failed to verify onCreate: %v\nOutput: %s", err, output3)
	}
	// Should contain npm version number
	if len(strings.TrimSpace(output3)) == 0 {
		t.Errorf("onCreate command (npm --version) failed, got empty output")
	}

	// Test 4: Verify postCreate ran
	output4, err := runPacknplay(t, "run", "--project", projectDir, "cat", "/tmp/deps.txt")
	if err != nil {
		t.Fatalf("Failed to verify postCreate: %v\nOutput: %s", err, output4)
	}
	if !strings.Contains(output4, "dependencies installed") {
		t.Errorf("postCreate command failed, got: %s", output4)
	}

	// Test 5: Verify package.json is accessible
	output5, err := runPacknplay(t, "run", "--project", projectDir, "cat", "package.json")
	if err != nil {
		t.Fatalf("Failed to read package.json: %v\nOutput: %s", err, output5)
	}
	if !strings.Contains(output5, "test-app") {
		t.Errorf("package.json not accessible, got: %s", output5)
	}

	t.Log("Real-world Node.js test passed!")
}
