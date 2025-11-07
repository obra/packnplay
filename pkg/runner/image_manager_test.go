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

// mockDockerClient for testing with error injection and call tracking
type mockDockerClient struct {
	pullCalled   bool
	buildCalled  bool
	pullError    error
	buildError   error
	inspectError error // Error to return for image inspect
	calls        []string
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

		// For image inspect, return the configured error (default: image not found)
		if args[0] == "image" && len(args) > 1 && args[1] == "inspect" {
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
