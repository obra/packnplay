package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfig_SaveAndLoad(t *testing.T) {
	// Use temp directory for test config
	tempDir := t.TempDir()
	if err := os.Setenv("XDG_CONFIG_HOME", tempDir); err != nil {
		t.Fatalf("Failed to set XDG_CONFIG_HOME: %v", err)
	}
	defer func() {
		if err := os.Unsetenv("XDG_CONFIG_HOME"); err != nil {
			t.Errorf("Failed to unset XDG_CONFIG_HOME: %v", err)
		}
	}()

	// Test config
	cfg := &Config{
		ContainerRuntime: "docker",
		DefaultCredentials: Credentials{
			Git: true,
			SSH: false,
			GH:  true,
			GPG: false,
			NPM: false,
		},
		DefaultEnvVars: []string{"ANTHROPIC_API_KEY", "OPENAI_API_KEY"},
		EnvConfigs: map[string]EnvConfig{
			"z.ai": {
				Name:        "Z.AI Claude",
				Description: "Test config",
				EnvVars: map[string]string{
					"ANTHROPIC_AUTH_TOKEN": "${Z_AI_API_KEY}",
					"ANTHROPIC_BASE_URL":   "https://api.z.ai/api/anthropic",
				},
			},
		},
	}

	// Save config
	if err := Save(cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Verify file exists
	configPath := GetConfigPath()
	if !fileExists(configPath) {
		t.Fatalf("Config file not created at %s", configPath)
	}

	// Load config back
	loaded, err := LoadWithoutRuntimeCheck()
	if err != nil {
		t.Fatalf("LoadWithoutRuntimeCheck() error = %v", err)
	}

	// Verify values
	if loaded.ContainerRuntime != cfg.ContainerRuntime {
		t.Errorf("ContainerRuntime = %v, want %v", loaded.ContainerRuntime, cfg.ContainerRuntime)
	}

	if loaded.DefaultCredentials.Git != cfg.DefaultCredentials.Git {
		t.Errorf("Git credentials = %v, want %v", loaded.DefaultCredentials.Git, cfg.DefaultCredentials.Git)
	}

	if loaded.DefaultCredentials.GH != cfg.DefaultCredentials.GH {
		t.Errorf("GH credentials = %v, want %v", loaded.DefaultCredentials.GH, cfg.DefaultCredentials.GH)
	}

	if len(loaded.DefaultEnvVars) != len(cfg.DefaultEnvVars) {
		t.Errorf("DefaultEnvVars length = %v, want %v", len(loaded.DefaultEnvVars), len(cfg.DefaultEnvVars))
	}

	if _, exists := loaded.EnvConfigs["z.ai"]; !exists {
		t.Errorf("z.ai config not found in loaded config")
	}
}

func TestGetConfigPath(t *testing.T) {
	tests := []struct {
		name           string
		xdgConfigHome  string
		expectedSuffix string
	}{
		{
			name:           "default XDG path",
			xdgConfigHome:  "",
			expectedSuffix: ".config/packnplay/config.json",
		},
		{
			name:           "custom XDG_CONFIG_HOME",
			xdgConfigHome:  "/custom/config",
			expectedSuffix: "/custom/config/packnplay/config.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.xdgConfigHome != "" {
				if err := os.Setenv("XDG_CONFIG_HOME", tt.xdgConfigHome); err != nil {
					t.Fatalf("Failed to set XDG_CONFIG_HOME: %v", err)
				}
				defer func() {
					if err := os.Unsetenv("XDG_CONFIG_HOME"); err != nil {
						t.Errorf("Failed to unset XDG_CONFIG_HOME: %v", err)
					}
				}()
			}

			path := GetConfigPath()
			if !filepath.IsAbs(path) {
				t.Errorf("GetConfigPath() returned relative path: %s", path)
			}

			if tt.xdgConfigHome != "" && path != tt.expectedSuffix {
				t.Errorf("GetConfigPath() = %v, want %v", path, tt.expectedSuffix)
			}
		})
	}
}

func TestDetectAvailableRuntimes(t *testing.T) {
	// This test depends on what's actually installed on the system
	runtimes := detectAvailableRuntimes()

	// Should find at least one runtime (docker is available in CI)
	if len(runtimes) == 0 {
		t.Error("detectAvailableRuntimes() returned empty list, expected at least one runtime")
	}

	// All returned runtimes should be valid
	validRuntimes := map[string]bool{
		"docker":   true,
		"podman":   true,
		"orbstack": true,
	}

	for _, runtime := range runtimes {
		if !validRuntimes[runtime] {
			t.Errorf("detectAvailableRuntimes() returned invalid runtime: %s", runtime)
		}
	}
}

func TestConfig_AWSCredentials(t *testing.T) {
	// Use temp directory for test config
	tempDir := t.TempDir()
	if err := os.Setenv("XDG_CONFIG_HOME", tempDir); err != nil {
		t.Fatalf("Failed to set XDG_CONFIG_HOME: %v", err)
	}
	defer func() {
		if err := os.Unsetenv("XDG_CONFIG_HOME"); err != nil {
			t.Errorf("Failed to unset XDG_CONFIG_HOME: %v", err)
		}
	}()

	// Test config with AWS enabled
	cfg := &Config{
		ContainerRuntime: "docker",
		DefaultCredentials: Credentials{
			Git: true,
			SSH: true,
			GH:  true,
			GPG: true,
			NPM: true,
			AWS: true, // Enable AWS credentials
		},
	}

	// Save config
	if err := Save(cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Load config back
	loaded, err := LoadWithoutRuntimeCheck()
	if err != nil {
		t.Fatalf("LoadWithoutRuntimeCheck() error = %v", err)
	}

	// Verify AWS credentials are preserved
	if loaded.DefaultCredentials.AWS != cfg.DefaultCredentials.AWS {
		t.Errorf("AWS credentials = %v, want %v", loaded.DefaultCredentials.AWS, cfg.DefaultCredentials.AWS)
	}

	// Test with AWS disabled
	cfg.DefaultCredentials.AWS = false
	if err := Save(cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err = LoadWithoutRuntimeCheck()
	if err != nil {
		t.Fatalf("LoadWithoutRuntimeCheck() error = %v", err)
	}

	if loaded.DefaultCredentials.AWS != false {
		t.Errorf("AWS credentials = %v, want false", loaded.DefaultCredentials.AWS)
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
