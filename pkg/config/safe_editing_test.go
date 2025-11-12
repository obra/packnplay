package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestConfigPreservationDuringEditing(t *testing.T) {
	// Test that interactive editing preserves settings not shown in UI

	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.json")

	// Create initial config with custom settings that won't be in UI
	originalConfig := &Config{
		ContainerRuntime: "docker",
		DefaultImage:     "ghcr.io/obra/packnplay-default:latest",
		DefaultCredentials: Credentials{
			Git: true,
			SSH: false,
			GH:  true,
		},
		DefaultEnvVars: []string{"CUSTOM_API_KEY", "SECRET_TOKEN"},
		EnvConfigs: map[string]EnvConfig{
			"custom-env": {
				Name:        "Custom Environment",
				Description: "Hand-added config",
				EnvVars: map[string]string{
					"CUSTOM_VAR": "custom_value",
				},
			},
		},
		DefaultContainer: DefaultContainerConfig{
			Image:               "my-custom/image:latest",
			CheckForUpdates:     true,
			AutoPullUpdates:     false,
			CheckFrequencyHours: 12, // Custom frequency
		},
	}

	// Save original config
	data, err := json.MarshalIndent(originalConfig, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}
	err = os.WriteFile(configFile, data, 0644)
	if err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	// Simulate interactive editing (updates only shown fields)
	runtime := "podman"
	creds := Credentials{
		Git: true,
		SSH: true,  // Changed from false
		GH:  false, // Changed from true
		GPG: true,  // New setting
	}
	updates := ConfigUpdates{
		ContainerRuntime:   &runtime,
		DefaultCredentials: &creds,
	}

	err = UpdateConfigSafely(configFile, updates)
	if err != nil {
		t.Errorf("UpdateConfigSafely() error = %v", err)
	}

	// Load the updated config
	updated, err := LoadConfigFromFile(configFile)
	if err != nil {
		t.Errorf("Failed to load updated config: %v", err)
	}

	// Verify shown settings were updated
	if updated.ContainerRuntime != "podman" {
		t.Errorf("Runtime = %v, want podman", updated.ContainerRuntime)
	}

	if !updated.DefaultCredentials.SSH {
		t.Error("SSH should be updated to true")
	}

	if updated.DefaultCredentials.GH {
		t.Error("GH should be updated to false")
	}

	if !updated.DefaultCredentials.GPG {
		t.Error("GPG should be updated to true")
	}

	// Verify hidden settings were preserved
	if len(updated.DefaultEnvVars) != 2 {
		t.Errorf("DefaultEnvVars count = %v, want 2 (should be preserved)", len(updated.DefaultEnvVars))
	}

	if updated.DefaultEnvVars[0] != "CUSTOM_API_KEY" {
		t.Errorf("Custom env var not preserved: %v", updated.DefaultEnvVars)
	}

	if len(updated.EnvConfigs) != 1 {
		t.Errorf("EnvConfigs count = %v, want 1 (should be preserved)", len(updated.EnvConfigs))
	}

	customEnv, exists := updated.EnvConfigs["custom-env"]
	if !exists {
		t.Error("Custom env config should be preserved")
	} else {
		if customEnv.EnvVars["CUSTOM_VAR"] != "custom_value" {
			t.Error("Custom env var value should be preserved")
		}
	}

	// Verify DefaultContainer settings were preserved
	if updated.DefaultContainer.Image != "my-custom/image:latest" {
		t.Errorf("Custom default image not preserved: %v", updated.DefaultContainer.Image)
	}

	if updated.DefaultContainer.CheckFrequencyHours != 12 {
		t.Errorf("Custom check frequency not preserved: %v", updated.DefaultContainer.CheckFrequencyHours)
	}
}

func TestLoadExistingOrEmpty(t *testing.T) {
	// Test loading existing config or returning empty template

	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.json")

	// Case 1: No file exists - should return empty config
	cfg, err := LoadExistingOrEmpty(configFile)
	if err != nil {
		t.Errorf("LoadExistingOrEmpty() error = %v for missing file", err)
	}

	if cfg == nil {
		t.Error("Should return empty config, not nil")
	}

	// Case 2: File exists - should load it
	existingConfig := &Config{
		ContainerRuntime: "podman",
		DefaultEnvVars:   []string{"TEST_VAR"},
	}

	data, _ := json.MarshalIndent(existingConfig, "", "  ")
	_ = os.WriteFile(configFile, data, 0644)

	cfg, err = LoadExistingOrEmpty(configFile)
	if err != nil {
		t.Errorf("LoadExistingOrEmpty() error = %v for existing file", err)
	}

	if cfg.ContainerRuntime != "podman" {
		t.Errorf("Should load existing runtime: %v", cfg.ContainerRuntime)
	}

	if len(cfg.DefaultEnvVars) != 1 || cfg.DefaultEnvVars[0] != "TEST_VAR" {
		t.Error("Should preserve existing env vars")
	}
}

func TestPartialConfigUpdate(t *testing.T) {
	// Test updating only specific sections of config

	original := &Config{
		ContainerRuntime: "docker",
		DefaultCredentials: Credentials{
			Git: true,
			SSH: false,
		},
		DefaultEnvVars: []string{"KEEP_THIS"},
		EnvConfigs: map[string]EnvConfig{
			"preserve-me": {Name: "Should be kept"},
		},
	}

	// Update only credentials
	credUpdates := Credentials{
		Git: true, // Keep
		SSH: true, // Change
		GH:  true, // Add
	}

	updated := applyCredentialUpdates(original, credUpdates)

	// Should preserve non-credential fields
	if updated.ContainerRuntime != "docker" {
		t.Error("Runtime should be preserved")
	}

	if len(updated.DefaultEnvVars) != 1 || updated.DefaultEnvVars[0] != "KEEP_THIS" {
		t.Error("Env vars should be preserved")
	}

	if len(updated.EnvConfigs) != 1 {
		t.Error("Env configs should be preserved")
	}

	// Should update credentials
	if !updated.DefaultCredentials.SSH {
		t.Error("SSH should be updated to true")
	}

	if !updated.DefaultCredentials.GH {
		t.Error("GH should be updated to true")
	}
}
