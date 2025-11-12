package config

import (
	"testing"
)

func TestDefaultContainerConfig(t *testing.T) {
	// Test that config supports configurable default container settings

	config := &Config{
		DefaultContainer: DefaultContainerConfig{
			Image:               "my-custom/image:latest",
			CheckForUpdates:     true,
			AutoPullUpdates:     false,
			CheckFrequencyHours: 24,
		},
	}

	if config.DefaultContainer.Image != "my-custom/image:latest" {
		t.Errorf("DefaultContainer.Image = %v, want my-custom/image:latest", config.DefaultContainer.Image)
	}

	if !config.DefaultContainer.CheckForUpdates {
		t.Error("CheckForUpdates should be true")
	}

	if config.DefaultContainer.AutoPullUpdates {
		t.Error("AutoPullUpdates should be false")
	}

	if config.DefaultContainer.CheckFrequencyHours != 24 {
		t.Errorf("CheckFrequencyHours = %v, want 24", config.DefaultContainer.CheckFrequencyHours)
	}
}

func TestGetDefaultImage(t *testing.T) {
	// Test that GetDefaultImage returns the configured default or fallback

	// Case 1: Custom default image configured
	config := &Config{
		DefaultContainer: DefaultContainerConfig{
			Image: "my-org/dev-container:v2",
		},
	}

	image := config.GetDefaultImage()
	if image != "my-org/dev-container:v2" {
		t.Errorf("GetDefaultImage() = %v, want my-org/dev-container:v2", image)
	}

	// Case 2: No custom image, should use packnplay default
	config = &Config{}
	image = config.GetDefaultImage()
	if image != "ghcr.io/obra/packnplay/devcontainer:latest" {
		t.Errorf("GetDefaultImage() = %v, want ghcr.io/obra/packnplay/devcontainer:latest", image)
	}

	// Case 3: Empty string image, should use default
	config = &Config{
		DefaultContainer: DefaultContainerConfig{
			Image: "",
		},
	}
	image = config.GetDefaultImage()
	if image != "ghcr.io/obra/packnplay/devcontainer:latest" {
		t.Errorf("GetDefaultImage() = %v, want ghcr.io/obra/packnplay/devcontainer:latest", image)
	}
}

func TestDefaultContainerDefaults(t *testing.T) {
	// Test that default container config has sensible defaults

	defaults := GetDefaultContainerConfig()

	if defaults.Image != "ghcr.io/obra/packnplay/devcontainer:latest" {
		t.Errorf("Default image = %v, want ghcr.io/obra/packnplay/devcontainer:latest", defaults.Image)
	}

	if !defaults.CheckForUpdates {
		t.Error("Default CheckForUpdates should be true")
	}

	if defaults.AutoPullUpdates {
		t.Error("Default AutoPullUpdates should be false")
	}

	if defaults.CheckFrequencyHours != 24 {
		t.Errorf("Default CheckFrequencyHours = %v, want 24", defaults.CheckFrequencyHours)
	}
}
