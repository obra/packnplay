package devcontainer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOverrideFeatureInstallOrderJSONParsing(t *testing.T) {
	// Test that overrideFeatureInstallOrder can be parsed from JSON
	jsonData := `{
		"image": "ubuntu:22.04",
		"features": {
			"feature-a": {},
			"feature-b": {},
			"feature-c": {}
		},
		"overrideFeatureInstallOrder": ["feature-c", "feature-a", "feature-b"]
	}`

	var config Config
	err := json.Unmarshal([]byte(jsonData), &config)
	if err != nil {
		t.Fatalf("Failed to unmarshal config: %v", err)
	}

	if config.OverrideFeatureInstallOrder == nil {
		t.Fatal("Expected OverrideFeatureInstallOrder to be set, got nil")
	}

	expectedOrder := []string{"feature-c", "feature-a", "feature-b"}
	if len(config.OverrideFeatureInstallOrder) != len(expectedOrder) {
		t.Fatalf("Expected %d features in override order, got %d", len(expectedOrder), len(config.OverrideFeatureInstallOrder))
	}

	for i, expected := range expectedOrder {
		if config.OverrideFeatureInstallOrder[i] != expected {
			t.Errorf("Expected feature %d to be '%s', got '%s'", i, expected, config.OverrideFeatureInstallOrder[i])
		}
	}
}

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

func TestPortAttributes_RequireLocalPortAndElevateIfNeeded(t *testing.T) {
	tests := []struct {
		name                    string
		json                    string
		wantRequireLocalPort    *bool
		wantElevateIfNeeded     *bool
		wantLabel               string
	}{
		{
			name: "requireLocalPort true, elevateIfNeeded false",
			json: `{
				"image": "alpine:latest",
				"portsAttributes": {
					"3000": {
						"label": "App",
						"requireLocalPort": true,
						"elevateIfNeeded": false
					}
				}
			}`,
			wantRequireLocalPort: func() *bool { v := true; return &v }(),
			wantElevateIfNeeded:  func() *bool { v := false; return &v }(),
			wantLabel:            "App",
		},
		{
			name: "requireLocalPort false, elevateIfNeeded true",
			json: `{
				"image": "alpine:latest",
				"portsAttributes": {
					"8080": {
						"label": "API",
						"requireLocalPort": false,
						"elevateIfNeeded": true
					}
				}
			}`,
			wantRequireLocalPort: func() *bool { v := false; return &v }(),
			wantElevateIfNeeded:  func() *bool { v := true; return &v }(),
			wantLabel:            "API",
		},
		{
			name: "both fields omitted",
			json: `{
				"image": "alpine:latest",
				"portsAttributes": {
					"9000": {
						"label": "DB"
					}
				}
			}`,
			wantRequireLocalPort: nil,
			wantElevateIfNeeded:  nil,
			wantLabel:            "DB",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var config Config
			err := json.Unmarshal([]byte(tt.json), &config)
			require.NoError(t, err)

			// Find the first port
			var attrs PortAttributes
			for _, v := range config.PortsAttributes {
				attrs = v
				break
			}

			assert.Equal(t, tt.wantLabel, attrs.Label)

			if tt.wantRequireLocalPort == nil {
				assert.Nil(t, attrs.RequireLocalPort)
			} else {
				require.NotNil(t, attrs.RequireLocalPort)
				assert.Equal(t, *tt.wantRequireLocalPort, *attrs.RequireLocalPort)
			}

			if tt.wantElevateIfNeeded == nil {
				assert.Nil(t, attrs.ElevateIfNeeded)
			} else {
				require.NotNil(t, attrs.ElevateIfNeeded)
				assert.Equal(t, *tt.wantElevateIfNeeded, *attrs.ElevateIfNeeded)
			}
		})
	}
}

func TestConfig_SecurityProperties(t *testing.T) {
	tests := []struct {
		name           string
		json           string
		wantPrivileged *bool
		wantInit       *bool
		wantCapAdd     []string
		wantSecurityOpt []string
		wantEntrypoint interface{} // can be string or []string
	}{
		{
			name: "all security properties present",
			json: `{
				"image": "alpine:latest",
				"privileged": true,
				"init": true,
				"capAdd": ["SYS_ADMIN", "NET_ADMIN"],
				"securityOpt": ["seccomp=unconfined", "apparmor=unconfined"],
				"entrypoint": "/bin/sh"
			}`,
			wantPrivileged: boolPtr(true),
			wantInit:       boolPtr(true),
			wantCapAdd:     []string{"SYS_ADMIN", "NET_ADMIN"},
			wantSecurityOpt: []string{"seccomp=unconfined", "apparmor=unconfined"},
			wantEntrypoint: "/bin/sh",
		},
		{
			name: "entrypoint as array",
			json: `{
				"image": "alpine:latest",
				"entrypoint": ["/bin/sh", "-c"]
			}`,
			wantPrivileged: nil,
			wantInit:       nil,
			wantCapAdd:     nil,
			wantSecurityOpt: nil,
			wantEntrypoint: []string{"/bin/sh", "-c"},
		},
		{
			name: "privileged false",
			json: `{
				"image": "alpine:latest",
				"privileged": false
			}`,
			wantPrivileged: boolPtr(false),
			wantInit:       nil,
			wantCapAdd:     nil,
			wantSecurityOpt: nil,
			wantEntrypoint: nil,
		},
		{
			name:           "no security properties",
			json:           `{"image": "alpine:latest"}`,
			wantPrivileged: nil,
			wantInit:       nil,
			wantCapAdd:     nil,
			wantSecurityOpt: nil,
			wantEntrypoint: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var config Config
			err := json.Unmarshal([]byte(tt.json), &config)
			require.NoError(t, err)

			assert.Equal(t, tt.wantPrivileged, config.Privileged)
			assert.Equal(t, tt.wantInit, config.Init)
			assert.Equal(t, tt.wantCapAdd, config.CapAdd)
			assert.Equal(t, tt.wantSecurityOpt, config.SecurityOpt)

			// Handle entrypoint which can be string or array
			if tt.wantEntrypoint == nil {
				assert.Nil(t, config.Entrypoint)
			} else {
				switch expected := tt.wantEntrypoint.(type) {
				case string:
					require.NotNil(t, config.Entrypoint)
					assert.Equal(t, []string{expected}, config.Entrypoint)
				case []string:
					assert.Equal(t, expected, config.Entrypoint)
				}
			}
		})
	}
}

func boolPtr(b bool) *bool {
	return &b
}

func TestConfig_WorkspaceMount(t *testing.T) {
	tests := []struct {
		name               string
		json               string
		wantWorkspaceMount string
		wantWorkspaceFolder string
		wantError          bool
	}{
		{
			name: "workspaceMount with workspaceFolder",
			json: `{
				"image": "alpine:latest",
				"workspaceMount": "source=${localWorkspaceFolder},target=/workspace,type=bind,consistency=cached",
				"workspaceFolder": "/workspace"
			}`,
			wantWorkspaceMount: "source=${localWorkspaceFolder},target=/workspace,type=bind,consistency=cached",
			wantWorkspaceFolder: "/workspace",
			wantError:          false,
		},
		{
			name: "workspaceMount without workspaceFolder",
			json: `{
				"image": "alpine:latest",
				"workspaceMount": "source=${localWorkspaceFolder},target=/app,type=bind"
			}`,
			wantWorkspaceMount: "source=${localWorkspaceFolder},target=/app,type=bind",
			wantWorkspaceFolder: "",
			wantError:          false,
		},
		{
			name: "no workspaceMount",
			json: `{
				"image": "alpine:latest"
			}`,
			wantWorkspaceMount: "",
			wantWorkspaceFolder: "",
			wantError:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var config Config
			err := json.Unmarshal([]byte(tt.json), &config)

			if tt.wantError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantWorkspaceMount, config.WorkspaceMount)
			assert.Equal(t, tt.wantWorkspaceFolder, config.WorkspaceFolder)
		})
	}
}

func TestConfig_WorkspaceMount_Integration(t *testing.T) {
	// Create temp dir with devcontainer.json containing workspaceMount
	tmpDir := t.TempDir()
	devcontainerDir := filepath.Join(tmpDir, ".devcontainer")
	require.NoError(t, os.Mkdir(devcontainerDir, 0755))

	configContent := `{
		"image": "mcr.microsoft.com/devcontainers/base:ubuntu",
		"workspaceMount": "source=${localWorkspaceFolder},target=/workspace,type=bind,consistency=cached",
		"workspaceFolder": "/workspace",
		"remoteUser": "vscode"
	}`

	require.NoError(t, os.WriteFile(
		filepath.Join(devcontainerDir, "devcontainer.json"),
		[]byte(configContent),
		0644,
	))

	config, err := LoadConfig(tmpDir)
	require.NoError(t, err)
	require.NotNil(t, config)

	assert.Equal(t, "mcr.microsoft.com/devcontainers/base:ubuntu", config.Image)
	assert.Equal(t, "source=${localWorkspaceFolder},target=/workspace,type=bind,consistency=cached", config.WorkspaceMount)
	assert.Equal(t, "/workspace", config.WorkspaceFolder)
	assert.Equal(t, "vscode", config.RemoteUser)
}
