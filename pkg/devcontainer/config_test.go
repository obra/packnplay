package devcontainer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestConfig_MountsAndRunArgs(t *testing.T) {
	tests := []struct {
		name        string
		json        string
		wantMounts  []string
		wantRunArgs []string
	}{
		{
			name: "mounts and runArgs present",
			json: `{
				"image": "alpine:latest",
				"mounts": [
					"source=/var/run/docker.sock,target=/var/run/docker.sock,type=bind",
					"source=my-volume,target=/data,type=volume"
				],
				"runArgs": ["--memory=2g", "--cpus=2"]
			}`,
			wantMounts: []string{
				"source=/var/run/docker.sock,target=/var/run/docker.sock,type=bind",
				"source=my-volume,target=/data,type=volume",
			},
			wantRunArgs: []string{"--memory=2g", "--cpus=2"},
		},
		{
			name:        "mounts and runArgs absent",
			json:        `{"image": "alpine:latest"}`,
			wantMounts:  nil,
			wantRunArgs: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var config Config
			err := json.Unmarshal([]byte(tt.json), &config)
			require.NoError(t, err)

			assert.Equal(t, tt.wantMounts, config.Mounts)
			assert.Equal(t, tt.wantRunArgs, config.RunArgs)
		})
	}
}

func TestConfig_Features(t *testing.T) {
	tests := []struct {
		name         string
		json         string
		wantFeatures map[string]interface{}
	}{
		{
			name: "features present",
			json: `{
				"image": "alpine:latest",
				"features": {
					"ghcr.io/devcontainers/features/node:1": {},
					"ghcr.io/devcontainers/features/docker-in-docker:2": {
						"version": "latest"
					}
				}
			}`,
			wantFeatures: map[string]interface{}{
				"ghcr.io/devcontainers/features/node:1": map[string]interface{}{},
				"ghcr.io/devcontainers/features/docker-in-docker:2": map[string]interface{}{
					"version": "latest",
				},
			},
		},
		{
			name:         "features absent",
			json:         `{"image": "alpine:latest"}`,
			wantFeatures: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var config Config
			err := json.Unmarshal([]byte(tt.json), &config)
			require.NoError(t, err)

			assert.Equal(t, tt.wantFeatures, config.Features)
		})
	}
}

func TestConfig_AllLifecycleCommands(t *testing.T) {
	jsonStr := `{
		"image": "alpine:latest",
		"updateContentCommand": "apt-get update",
		"postAttachCommand": "echo attached"
	}`

	var config Config
	err := json.Unmarshal([]byte(jsonStr), &config)
	require.NoError(t, err)

	require.NotNil(t, config.UpdateContentCommand)
	cmd, ok := config.UpdateContentCommand.AsString()
	require.True(t, ok)
	assert.Equal(t, "apt-get update", cmd)

	require.NotNil(t, config.PostAttachCommand)
	cmd, ok = config.PostAttachCommand.AsString()
	require.True(t, ok)
	assert.Equal(t, "echo attached", cmd)
}

func TestConfig_UserEnvProbe(t *testing.T) {
	tests := []struct {
		name            string
		json            string
		wantUserEnvProbe string
	}{
		{
			name: "userEnvProbe set to none",
			json: `{
				"image": "alpine:latest",
				"userEnvProbe": "none"
			}`,
			wantUserEnvProbe: "none",
		},
		{
			name: "userEnvProbe set to loginShell",
			json: `{
				"image": "alpine:latest",
				"userEnvProbe": "loginShell"
			}`,
			wantUserEnvProbe: "loginShell",
		},
		{
			name: "userEnvProbe set to interactiveShell",
			json: `{
				"image": "alpine:latest",
				"userEnvProbe": "interactiveShell"
			}`,
			wantUserEnvProbe: "interactiveShell",
		},
		{
			name: "userEnvProbe set to loginInteractiveShell",
			json: `{
				"image": "alpine:latest",
				"userEnvProbe": "loginInteractiveShell"
			}`,
			wantUserEnvProbe: "loginInteractiveShell",
		},
		{
			name: "userEnvProbe not set (empty)",
			json: `{
				"image": "alpine:latest"
			}`,
			wantUserEnvProbe: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var config Config
			err := json.Unmarshal([]byte(tt.json), &config)
			require.NoError(t, err)

			assert.Equal(t, tt.wantUserEnvProbe, config.UserEnvProbe)
		})
	}
}

func TestLoadLockFile(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func(tmpDir string) error
		wantLockfile *LockFile
		wantErr     bool
	}{
		{
			name: "valid lockfile with multiple features",
			setupFunc: func(tmpDir string) error {
				devcontainerDir := filepath.Join(tmpDir, ".devcontainer")
				if err := os.Mkdir(devcontainerDir, 0755); err != nil {
					return err
				}

				lockContent := `{
					"features": {
						"ghcr.io/devcontainers/features/node:1": {
							"version": "1.2.3",
							"resolved": "ghcr.io/devcontainers/features/node@sha256:abc123"
						},
						"ghcr.io/devcontainers/features/docker-in-docker:2": {
							"version": "2.0.0",
							"resolved": "ghcr.io/devcontainers/features/docker-in-docker@sha256:def456"
						}
					}
				}`

				return os.WriteFile(
					filepath.Join(devcontainerDir, "devcontainer-lock.json"),
					[]byte(lockContent),
					0644,
				)
			},
			wantLockfile: &LockFile{
				Features: map[string]LockedFeature{
					"ghcr.io/devcontainers/features/node:1": {
						Version:  "1.2.3",
						Resolved: "ghcr.io/devcontainers/features/node@sha256:abc123",
					},
					"ghcr.io/devcontainers/features/docker-in-docker:2": {
						Version:  "2.0.0",
						Resolved: "ghcr.io/devcontainers/features/docker-in-docker@sha256:def456",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "missing lockfile returns nil without error",
			setupFunc: func(tmpDir string) error {
				// Create .devcontainer dir but no lockfile
				devcontainerDir := filepath.Join(tmpDir, ".devcontainer")
				return os.Mkdir(devcontainerDir, 0755)
			},
			wantLockfile: nil,
			wantErr:      false,
		},
		{
			name: "invalid JSON returns error",
			setupFunc: func(tmpDir string) error {
				devcontainerDir := filepath.Join(tmpDir, ".devcontainer")
				if err := os.Mkdir(devcontainerDir, 0755); err != nil {
					return err
				}

				// Write invalid JSON
				return os.WriteFile(
					filepath.Join(devcontainerDir, "devcontainer-lock.json"),
					[]byte(`{invalid json`),
					0644,
				)
			},
			wantLockfile: nil,
			wantErr:      true,
		},
		{
			name: "empty features map",
			setupFunc: func(tmpDir string) error {
				devcontainerDir := filepath.Join(tmpDir, ".devcontainer")
				if err := os.Mkdir(devcontainerDir, 0755); err != nil {
					return err
				}

				lockContent := `{"features": {}}`

				return os.WriteFile(
					filepath.Join(devcontainerDir, "devcontainer-lock.json"),
					[]byte(lockContent),
					0644,
				)
			},
			wantLockfile: &LockFile{
				Features: map[string]LockedFeature{},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			if tt.setupFunc != nil {
				err := tt.setupFunc(tmpDir)
				require.NoError(t, err, "setup function should not fail")
			}

			lockfile, err := LoadLockFile(tmpDir)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, lockfile)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantLockfile, lockfile)
			}
		})
	}
}

func TestConfig_ShouldOverrideCommand(t *testing.T) {
	tests := []struct {
		name           string
		overrideCmd    *bool
		expectedResult bool
	}{
		{
			name:           "nil (unset) defaults to true",
			overrideCmd:    nil,
			expectedResult: true,
		},
		{
			name: "explicitly true",
			overrideCmd: func() *bool {
				v := true
				return &v
			}(),
			expectedResult: true,
		},
		{
			name: "explicitly false",
			overrideCmd: func() *bool {
				v := false
				return &v
			}(),
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				OverrideCommand: tt.overrideCmd,
			}
			result := config.ShouldOverrideCommand()
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}
