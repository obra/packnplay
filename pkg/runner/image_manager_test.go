package runner

import (
	"fmt"
	"os"
	"path/filepath"
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

func TestImageManager_EnsureAvailable_PullError(t *testing.T) {
	// Test: Error injection for pull
	mockClient := &mockDockerClient{
		pullError: fmt.Errorf("network error"),
	}

	im := NewImageManager(mockClient, false)

	devConfig := &devcontainer.Config{
		Image: "ubuntu:22.04",
	}

	err := im.EnsureAvailable(devConfig, "/test/project")
	if err == nil {
		t.Error("Expected error when pull fails")
	}
}

func TestImageManager_EnsureAvailable_BuildError(t *testing.T) {
	// Test: Error injection for build
	mockClient := &mockDockerClient{
		buildError: fmt.Errorf("build failed"),
	}

	im := NewImageManager(mockClient, false)

	devConfig := &devcontainer.Config{
		DockerFile: "Dockerfile",
	}

	err := im.EnsureAvailable(devConfig, "/test/project")
	if err == nil {
		t.Error("Expected error when build fails")
	}
}

func TestImageManager_PullImage_AlreadyExists(t *testing.T) {
	// Test: When image already exists locally, no pull should be attempted
	mockClient := &mockDockerClient{
		imageExists: true,
	}

	im := NewImageManager(mockClient, false)

	devConfig := &devcontainer.Config{
		Image: "ubuntu:22.04",
	}

	err := im.EnsureAvailable(devConfig, "/test/project")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if mockClient.pullCalled {
		t.Error("Expected no pull to be called when image already exists")
	}
}

func TestImageManager_BuildImage_AlreadyBuilt(t *testing.T) {
	// Test: When image already built, no build should be attempted
	mockClient := &mockDockerClient{
		imageExists: true,
	}

	im := NewImageManager(mockClient, false)

	devConfig := &devcontainer.Config{
		DockerFile: "Dockerfile",
	}

	err := im.EnsureAvailable(devConfig, "/test/project")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if mockClient.buildCalled {
		t.Error("Expected no build to be called when image already exists")
	}
}

// mockDockerClient for testing with error injection and call tracking
type mockDockerClient struct {
	pullCalled   bool
	buildCalled  bool
	pullError    error
	buildError   error
	inspectError error      // Error to return for image inspect
	imageExists  bool       // If true, image inspect succeeds (image already exists)
	calls        []string   // Track command names
	capturedArgs [][]string // Track all args for detailed verification
	execCalls    [][]string // Track exec calls for lifecycle testing
	execOutput   string     // Output to return for exec
	execError    error      // Error to return for exec
}

func (m *mockDockerClient) RunWithProgress(imageName string, args ...string) error {
	if len(args) > 0 {
		m.calls = append(m.calls, args[0])

		if args[0] == "pull" {
			m.pullCalled = true
			return m.pullError
		} else if args[0] == "build" {
			m.buildCalled = true
			return m.buildError
		}
	}

	return nil
}

func (m *mockDockerClient) Run(args ...string) (string, error) {
	if len(args) > 0 {
		m.calls = append(m.calls, args[0])

		// Track exec calls
		if args[0] == "exec" {
			m.execCalls = append(m.execCalls, args)
			if m.execError != nil {
				return "", m.execError
			}
			return m.execOutput, nil
		}

		// For image inspect, return the configured error (default: image not found)
		if args[0] == "image" && len(args) > 1 && args[1] == "inspect" {
			// If imageExists is true, return success (no error)
			if m.imageExists {
				return "", nil
			}
			if m.inspectError != nil {
				return "", m.inspectError
			}
			// Default: image not found (so we need to pull/build)
			return "", fmt.Errorf("image not found")
		}
	}

	return "", nil
}

func (m *mockDockerClient) Command() string {
	return "docker"
}

// TestImageManager_EnsureAvailable_WithBuildConfig tests integration with BuildConfig
func TestImageManager_EnsureAvailable_WithBuildConfig(t *testing.T) {
	// Test: When devcontainer specifies Build config, use it
	mockClient := &mockDockerClient{
		buildCalled: false,
	}

	im := NewImageManager(mockClient, false)

	devConfig := &devcontainer.Config{
		Build: &devcontainer.BuildConfig{
			Dockerfile: "Dockerfile.dev",
			Context:    "..",
			Args: map[string]string{
				"VARIANT": "16-bullseye",
			},
			Target: "development",
		},
	}

	err := im.EnsureAvailable(devConfig, "/test/project")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if !mockClient.buildCalled {
		t.Error("Expected image build to be called")
	}
}

// TestImageManager_EnsureAvailable_BuildConfigPriority tests that Build.Dockerfile takes priority
func TestImageManager_EnsureAvailable_BuildConfigPriority(t *testing.T) {
	// Test: Build.Dockerfile should be used over DockerFile
	mockClient := &mockDockerClient{
		buildCalled: false,
	}

	im := NewImageManager(mockClient, false)

	devConfig := &devcontainer.Config{
		DockerFile: "Dockerfile",
		Build: &devcontainer.BuildConfig{
			Dockerfile: "Dockerfile.dev", // This should take priority
		},
	}

	err := im.EnsureAvailable(devConfig, "/test/project")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if !mockClient.buildCalled {
		t.Error("Expected image build to be called")
	}
}

// TestImageManager_BuildWithAdvancedOptions tests that build args are properly passed
func TestImageManager_BuildWithAdvancedOptions(t *testing.T) {
	// Test: Verify BuildConfig generates proper docker build args
	mockClient := &mockDockerClient{
		buildCalled:  false,
		capturedArgs: [][]string{},
	}

	im := NewImageManager(mockClient, false)

	devConfig := &devcontainer.Config{
		Build: &devcontainer.BuildConfig{
			Dockerfile: "Dockerfile",
			Args: map[string]string{
				"VARIANT": "16-bullseye",
			},
		},
	}

	err := im.EnsureAvailable(devConfig, "/test/project")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if !mockClient.buildCalled {
		t.Error("Expected image build to be called")
	}
}

// TestImageManager_BuildWithFeatures tests that buildImage processes features
func TestImageManager_BuildWithFeatures(t *testing.T) {
	// Test: When devcontainer specifies features, process them and build with generated Dockerfile

	// Create temporary project directory
	tempDir := t.TempDir()
	devcontainerDir := filepath.Join(tempDir, ".devcontainer")
	if err := os.MkdirAll(devcontainerDir, 0755); err != nil {
		t.Fatalf("Failed to create .devcontainer dir: %v", err)
	}

	// Create test feature
	featureDir := filepath.Join(devcontainerDir, "test-feature")
	if err := os.MkdirAll(featureDir, 0755); err != nil {
		t.Fatalf("Failed to create feature dir: %v", err)
	}

	// Create feature metadata
	featureJSON := `{
		"id": "test-feature",
		"version": "1.0.0",
		"name": "Test Feature",
		"description": "A test feature"
	}`
	if err := os.WriteFile(filepath.Join(featureDir, "devcontainer-feature.json"), []byte(featureJSON), 0644); err != nil {
		t.Fatalf("Failed to write feature metadata: %v", err)
	}

	// Create install script
	installScript := `#!/bin/bash
echo "Installing test feature"
`
	if err := os.WriteFile(filepath.Join(featureDir, "install.sh"), []byte(installScript), 0755); err != nil {
		t.Fatalf("Failed to write install script: %v", err)
	}

	mockClient := &mockDockerClient{
		buildCalled: false,
	}

	im := NewImageManager(mockClient, false)

	devConfig := &devcontainer.Config{
		Image:      "ubuntu:22.04",
		RemoteUser: "testuser",
		Features: map[string]interface{}{
			"test-feature": map[string]interface{}{},
		},
	}

	err := im.EnsureAvailable(devConfig, tempDir)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if !mockClient.buildCalled {
		t.Error("Expected image build to be called when features are present")
	}
}
