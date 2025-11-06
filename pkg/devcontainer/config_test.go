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
	_ = os.Mkdir(devcontainerDir, 0755)

	configContent := `{
		"image": "mcr.microsoft.com/devcontainers/base:ubuntu",
		"remoteUser": "vscode"
	}`

	_ = os.WriteFile(
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

func TestGetDefaultConfig(t *testing.T) {
	// Test with empty string - should use default image and detect user
	config := GetDefaultConfig("")
	if config.Image != "ghcr.io/obra/packnplay/devcontainer:latest" {
		t.Errorf("GetDefaultConfig(\"\") Image = %v, want ghcr.io/obra/packnplay/devcontainer:latest", config.Image)
	}
	// RemoteUser should be detected, not hardcoded. For non-existent images, should fall back to "root"
	if config.RemoteUser == "" {
		t.Errorf("GetDefaultConfig(\"\") RemoteUser should not be empty")
	}

	// Test with existing image (ubuntu should work)
	ubuntuImage := "ubuntu:22.04"
	config = GetDefaultConfig(ubuntuImage)
	if config.Image != ubuntuImage {
		t.Errorf("GetDefaultConfig(%v) Image = %v, want %v", ubuntuImage, config.Image, ubuntuImage)
	}
	// For ubuntu, should detect and use "root" as fallback since no better user found
	if config.RemoteUser == "" {
		t.Errorf("GetDefaultConfig(%v) RemoteUser should not be empty", ubuntuImage)
	}
}
