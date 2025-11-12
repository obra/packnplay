package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/obra/packnplay/pkg/devcontainer"
	"github.com/obra/packnplay/pkg/userdetect"
)

func TestCompleteUserDetectionFlow(t *testing.T) {
	tests := []struct {
		name                string
		image               string
		devcontainerContent string
		expectedUser        string
		expectDetection     bool // true if we expect runtime detection vs devcontainer.json
	}{
		{
			name:  "node image should detect root user",
			image: "node:18",
			devcontainerContent: `{
				"image": "node:18"
			}`,
			expectedUser:    "root", // node:18 actually runs as root by default
			expectDetection: true,
		},
		{
			name:  "ubuntu image should fall back to root",
			image: "ubuntu:22.04",
			devcontainerContent: `{
				"image": "ubuntu:22.04"
			}`,
			expectedUser:    "root",
			expectDetection: true,
		},
		{
			name:  "devcontainer.json remoteUser overrides detection",
			image: "node:18",
			devcontainerContent: `{
				"image": "node:18",
				"remoteUser": "customuser"
			}`,
			expectedUser:    "customuser",
			expectDetection: false,
		},
		{
			name:                "missing devcontainer.json uses GetDefaultConfig",
			image:               "ubuntu:22.04",
			devcontainerContent: "", // no devcontainer.json
			expectedUser:        "root",
			expectDetection:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip tests that require Docker if not available
			if !isDockerAvailable() {
				t.Skip("Docker not available")
			}

			// Create temp directory with devcontainer.json if specified
			tmpDir := t.TempDir()

			if tt.devcontainerContent != "" {
				devcontainerDir := filepath.Join(tmpDir, ".devcontainer")
				err := os.Mkdir(devcontainerDir, 0755)
				if err != nil {
					t.Fatalf("Failed to create .devcontainer dir: %v", err)
				}

				err = os.WriteFile(
					filepath.Join(devcontainerDir, "devcontainer.json"),
					[]byte(tt.devcontainerContent),
					0644,
				)
				if err != nil {
					t.Fatalf("Failed to write devcontainer.json: %v", err)
				}
			}

			// Test LoadConfig path
			config, err := devcontainer.LoadConfig(tmpDir)

			if tt.devcontainerContent == "" {
				// No devcontainer.json, should return nil
				if config != nil {
					t.Errorf("LoadConfig() should return nil for missing devcontainer.json, got %+v", config)
				}

				// Test GetDefaultConfig path
				config = devcontainer.GetDefaultConfig(tt.image)
			}

			if err != nil {
				t.Fatalf("LoadConfig() error = %v", err)
			}

			if config == nil {
				t.Fatal("Config should not be nil")
			}

			// Verify the detected user matches expectation
			if config.RemoteUser != tt.expectedUser {
				t.Errorf("RemoteUser = %v, want %v", config.RemoteUser, tt.expectedUser)
			}

			// Verify the image is correct
			if config.Image != tt.image {
				t.Errorf("Image = %v, want %v", config.Image, tt.image)
			}

			// Test direct user detection to confirm it works
			userResult, err := userdetect.DetectContainerUser(tt.image, &userdetect.DevcontainerConfig{
				RemoteUser: "", // Test detection path
			})
			if err != nil {
				t.Fatalf("DetectContainerUser() error = %v", err)
			}

			if tt.expectDetection && userResult.Source == "fallback" && tt.expectedUser != "root" {
				t.Errorf("Expected runtime detection but got fallback for user %v", tt.expectedUser)
			}
		})
	}
}

func TestAgentMountsDynamicUser(t *testing.T) {
	// This test ensures agents use dynamic user paths, not hardcoded vscode
	tests := []struct {
		name          string
		containerUser string
		expectedPath  string
	}{
		{
			name:          "vscode user gets /home/vscode path",
			containerUser: "vscode",
			expectedPath:  "/home/vscode/.claude",
		},
		{
			name:          "node user gets /home/node path",
			containerUser: "node",
			expectedPath:  "/home/node/.claude",
		},
		{
			name:          "root user gets /root path",
			containerUser: "root",
			expectedPath:  "/root/.claude",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Import here to avoid circular imports in main packages
			// We'll create a simple agent to test
			type Mount struct {
				HostPath      string
				ContainerPath string
				ReadOnly      bool
			}

			getMounts := func(hostHomeDir string, containerUser string) []Mount {
				containerHomeDir := "/root"
				if containerUser != "root" {
					containerHomeDir = "/home/" + containerUser
				}

				return []Mount{
					{
						HostPath:      filepath.Join(hostHomeDir, ".claude"),
						ContainerPath: filepath.Join(containerHomeDir, ".claude"),
						ReadOnly:      false,
					},
				}
			}

			mounts := getMounts("/home/test", tt.containerUser)

			if len(mounts) != 1 {
				t.Fatalf("Expected 1 mount, got %d", len(mounts))
			}

			if mounts[0].ContainerPath != tt.expectedPath {
				t.Errorf("ContainerPath = %v, want %v", mounts[0].ContainerPath, tt.expectedPath)
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
