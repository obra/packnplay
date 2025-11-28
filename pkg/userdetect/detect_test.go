package userdetect

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestDetectContainerUser(t *testing.T) {
	tests := []struct {
		name         string
		image        string
		devcontainer *DevcontainerConfig
		expectedUser string
		shouldError  bool
	}{
		{
			name:         "devcontainer.json remoteUser takes precedence",
			image:        "ubuntu:22.04",
			devcontainer: &DevcontainerConfig{RemoteUser: "customuser"},
			expectedUser: "customuser",
		},
		{
			name:         "detect from node image (runs as root by default)",
			image:        "node:18",
			devcontainer: nil,
			expectedUser: "root", // node:18 actually runs as root by default
		},
		{
			name:         "detect from ubuntu image",
			image:        "ubuntu:22.04",
			devcontainer: nil,
			expectedUser: "root", // ubuntu runs as root
		},
		{
			name:         "detect from vscode devcontainer",
			image:        "mcr.microsoft.com/devcontainers/base:ubuntu",
			devcontainer: nil,
			expectedUser: "root", // this image runs as root by default (vscode user exists but isn't default)
		},
		{
			name:         "invalid image should error",
			image:        "nonexistent:invalid",
			devcontainer: nil,
			shouldError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip tests that require Docker if not available
			if !isDockerAvailable() {
				t.Skip("Docker not available")
			}

			result, err := DetectContainerUser(tt.image, tt.devcontainer)

			if tt.shouldError {
				if err == nil {
					t.Errorf("DetectContainerUser() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("DetectContainerUser() error = %v", err)
			}

			if result.User != tt.expectedUser {
				t.Errorf("DetectContainerUser() user = %v, want %v", result.User, tt.expectedUser)
			}

			// Verify user detection details are populated
			if result.Source == "" {
				t.Errorf("DetectContainerUser() source should not be empty")
			}

			if result.HomeDir == "" {
				t.Errorf("DetectContainerUser() homeDir should not be empty")
			}
		})
	}
}

func TestDetectUsersInImage(t *testing.T) {
	tests := []struct {
		name          string
		image         string
		expectedUsers []string
	}{
		{
			name:          "ubuntu should have root and potentially ubuntu user",
			image:         "ubuntu:22.04",
			expectedUsers: []string{"root"}, // at minimum root should exist
		},
		{
			name:          "node image should have node user",
			image:         "node:18",
			expectedUsers: []string{"root", "node"}, // both root and node should exist
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !isDockerAvailable() {
				t.Skip("Docker not available")
			}

			users, err := DetectUsersInImage(tt.image)
			if err != nil {
				t.Fatalf("DetectUsersInImage() error = %v", err)
			}

			// Verify expected users exist
			for _, expectedUser := range tt.expectedUsers {
				found := false
				for _, user := range users {
					if user.Username == expectedUser {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("DetectUsersInImage() missing expected user %v, got users: %v", expectedUser, users)
				}
			}
		})
	}
}

func TestGetImageDefaultUser(t *testing.T) {
	tests := []struct {
		name         string
		image        string
		expectedUser string
	}{
		{
			name:         "ubuntu image default",
			image:        "ubuntu:22.04",
			expectedUser: "root", // ubuntu runs as root by default
		},
		{
			name:         "node image default",
			image:        "node:18",
			expectedUser: "root", // node:18 doesn't set USER directive, defaults to root
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !isDockerAvailable() {
				t.Skip("Docker not available")
			}

			user, err := GetImageDefaultUser(tt.image)
			if err != nil {
				t.Fatalf("GetImageDefaultUser() error = %v", err)
			}

			if user != tt.expectedUser {
				t.Errorf("GetImageDefaultUser() = %v, want %v", user, tt.expectedUser)
			}
		})
	}
}

func TestDirectDetection(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker not available")
	}

	// Test direct detection with a simple image
	result, err := detectRuntimeUserDirect("ubuntu:22.04")
	if err != nil {
		t.Fatalf("detectRuntimeUserDirect() error = %v", err)
	}

	if result.User == "" {
		t.Error("detectRuntimeUserDirect() should return non-empty user")
	}

	if result.HomeDir == "" {
		t.Error("detectRuntimeUserDirect() should return non-empty home directory")
	}

	if result.Source != "runtime_detection" {
		t.Errorf("detectRuntimeUserDirect() source = %v, want runtime_detection", result.Source)
	}

	t.Logf("Detected user: %s, home: %s", result.User, result.HomeDir)
}

func TestCaching(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker not available")
	}

	image := "ubuntu:22.04"

	// Clear any existing cache for this test
	imageID, err := getImageID(image)
	if err != nil {
		t.Fatalf("getImageID() error = %v", err)
	}

	// Delete cache file if it exists
	cacheFilePath, err := getCacheFilePath(imageID)
	if err == nil {
		_ = os.Remove(cacheFilePath) // Ignore errors
	}

	// First detection should hit the container
	result1, err := DetectContainerUser(image, nil)
	if err != nil {
		t.Fatalf("DetectContainerUser() error = %v", err)
	}

	// Second detection should hit cache
	result2, err := DetectContainerUser(image, nil)
	if err != nil {
		t.Fatalf("DetectContainerUser() error = %v", err)
	}

	// Results should be identical
	if result1.User != result2.User {
		t.Errorf("Cached result user mismatch: %v vs %v", result1.User, result2.User)
	}

	if result1.HomeDir != result2.HomeDir {
		t.Errorf("Cached result homeDir mismatch: %v vs %v", result1.HomeDir, result2.HomeDir)
	}

	// Verify cache file was created
	if _, err := os.Stat(cacheFilePath); os.IsNotExist(err) {
		t.Error("Cache file should have been created")
	}
}

func TestGetImageID(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker not available")
	}

	imageID, err := getImageID("ubuntu:22.04")
	if err != nil {
		t.Fatalf("getImageID() error = %v", err)
	}

	if imageID == "" {
		t.Error("getImageID() should return non-empty image ID")
	}

	// Image ID should start with sha256:
	if !strings.HasPrefix(imageID, "sha256:") {
		t.Errorf("Image ID should start with 'sha256:', got: %s", imageID)
	}

	t.Logf("Image ID: %s", imageID)
}

func TestGetShellFlags(t *testing.T) {
	tests := []struct {
		name         string
		userEnvProbe string
		wantFlags    []string
	}{
		{
			name:         "none - no flags",
			userEnvProbe: "none",
			wantFlags:    []string{},
		},
		{
			name:         "loginShell - login flag",
			userEnvProbe: "loginShell",
			wantFlags:    []string{"-l"},
		},
		{
			name:         "interactiveShell - interactive flag",
			userEnvProbe: "interactiveShell",
			wantFlags:    []string{"-i"},
		},
		{
			name:         "loginInteractiveShell - both flags",
			userEnvProbe: "loginInteractiveShell",
			wantFlags:    []string{"-l", "-i"},
		},
		{
			name:         "empty defaults to loginInteractiveShell",
			userEnvProbe: "",
			wantFlags:    []string{"-l", "-i"},
		},
		{
			name:         "unknown value defaults to loginInteractiveShell",
			userEnvProbe: "invalidValue",
			wantFlags:    []string{"-l", "-i"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flags := getShellFlags(tt.userEnvProbe)
			if len(flags) != len(tt.wantFlags) {
				t.Errorf("getShellFlags(%q) = %v, want %v", tt.userEnvProbe, flags, tt.wantFlags)
				return
			}
			for i := range flags {
				if flags[i] != tt.wantFlags[i] {
					t.Errorf("getShellFlags(%q) = %v, want %v", tt.userEnvProbe, flags, tt.wantFlags)
					return
				}
			}
		})
	}
}

// Helper function to check if Docker is available for testing
func isDockerAvailable() bool {
	// Skip Docker tests in CI since CI itself runs in Docker
	if os.Getenv("CI") != "" {
		return false
	}
	cmd := exec.Command("docker", "info")
	return cmd.Run() == nil
}

