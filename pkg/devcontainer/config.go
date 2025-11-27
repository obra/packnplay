package devcontainer

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/obra/packnplay/pkg/userdetect"
)

// LockedFeature represents a pinned feature version in devcontainer-lock.json
type LockedFeature struct {
	Version  string `json:"version"`  // Semantic version of the feature
	Resolved string `json:"resolved"` // Full OCI ref with digest or version
}

// LockFile represents devcontainer-lock.json which pins feature versions
type LockFile struct {
	Features map[string]LockedFeature `json:"features"`
}

// Config represents a parsed devcontainer.json
type Config struct {
	// Basic container configuration
	Image        string                 `json:"image"`
	DockerFile   string                 `json:"dockerFile"`
	Build        *BuildConfig           `json:"build,omitempty"`
	Name         string                 `json:"name,omitempty"`          // Display name for the dev container
	RemoteUser   string                 `json:"remoteUser"`
	ContainerEnv map[string]string      `json:"containerEnv,omitempty"`
	RemoteEnv    map[string]string      `json:"remoteEnv,omitempty"`
	ForwardPorts []interface{}          `json:"forwardPorts,omitempty"` // int or string
	Mounts       []string               `json:"mounts,omitempty"`       // Docker mount syntax
	RunArgs      []string               `json:"runArgs,omitempty"`      // Additional docker run arguments
	Features     map[string]interface{} `json:"features,omitempty"`

	// Workspace configuration - CRITICAL for proper workspace setup
	WorkspaceFolder string `json:"workspaceFolder,omitempty"` // Path inside container where workspace should be
	WorkspaceMount  string `json:"workspaceMount,omitempty"`  // Custom mount string for workspace

	// Lifecycle commands - complete Microsoft specification
	InitializeCommand    *LifecycleCommand `json:"initializeCommand,omitempty"`    // Runs on host before container creation
	OnCreateCommand      *LifecycleCommand `json:"onCreateCommand,omitempty"`
	UpdateContentCommand *LifecycleCommand `json:"updateContentCommand,omitempty"`
	PostCreateCommand    *LifecycleCommand `json:"postCreateCommand,omitempty"`
	PostStartCommand     *LifecycleCommand `json:"postStartCommand,omitempty"`
	PostAttachCommand    *LifecycleCommand `json:"postAttachCommand,omitempty"`   // Runs every time IDE attaches

	// Lifecycle control
	WaitFor string `json:"waitFor,omitempty"` // Which lifecycle command to wait for before setup is complete
}

// LoadConfig loads and parses .devcontainer/devcontainer.json if it exists
func LoadConfig(projectPath string) (*Config, error) {
	configPath := filepath.Join(projectPath, ".devcontainer", "devcontainer.json")

	// Check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	// If RemoteUser is not specified, detect the best user for the image
	if config.RemoteUser == "" && config.Image != "" {
		userResult, err := userdetect.DetectContainerUser(config.Image, nil)
		if err != nil {
			// If detection fails, fall back to a safe default
			config.RemoteUser = "root"
		} else {
			config.RemoteUser = userResult.User
		}
	}

	return &config, nil
}

// GetDefaultConfig returns the default devcontainer config
// If defaultImage is empty, uses "ghcr.io/obra/packnplay/devcontainer:latest"
func GetDefaultConfig(defaultImage string) *Config {
	if defaultImage == "" {
		defaultImage = "ghcr.io/obra/packnplay/devcontainer:latest"
	}

	// Detect the best user for this image
	userResult, err := userdetect.DetectContainerUser(defaultImage, nil)
	remoteUser := "root" // safe fallback
	if err == nil {
		remoteUser = userResult.User
	}

	return &Config{
		Image:      defaultImage,
		RemoteUser: remoteUser,
	}
}

// GetDockerfile returns the dockerfile path from either DockerFile field or Build.Dockerfile
func (c *Config) GetDockerfile() string {
	if c.Build != nil && c.Build.Dockerfile != "" {
		return c.Build.Dockerfile
	}
	return c.DockerFile
}

// HasDockerfile returns true if a dockerfile is specified
func (c *Config) HasDockerfile() bool {
	return c.GetDockerfile() != ""
}

// GetResolvedEnvironment applies variable substitution and returns resolved environment variables
// First applies containerEnv, then remoteEnv (which can reference containerEnv)
func (c *Config) GetResolvedEnvironment(ctx *SubstituteContext) map[string]string {
	result := make(map[string]string)

	// First pass: containerEnv
	for k, v := range c.ContainerEnv {
		resolved := substituteString(ctx, v)
		result[k] = resolved
		// Add to context for containerEnv: references
		ctx.ContainerEnv[k] = resolved
	}

	// Second pass: remoteEnv (can reference containerEnv)
	for k, v := range c.RemoteEnv {
		if v == "" {
			// Empty string/null removes variable
			delete(result, k)
		} else {
			result[k] = substituteString(ctx, v)
		}
	}

	return result
}

// LoadLockFile loads and parses .devcontainer/devcontainer-lock.json if it exists
// Returns nil if the lockfile doesn't exist (not an error)
func LoadLockFile(projectPath string) (*LockFile, error) {
	lockPath := filepath.Join(projectPath, ".devcontainer", "devcontainer-lock.json")

	// Check if file exists
	if _, err := os.Stat(lockPath); os.IsNotExist(err) {
		return nil, nil // No lockfile is not an error
	}

	data, err := os.ReadFile(lockPath)
	if err != nil {
		return nil, err
	}

	var lockfile LockFile
	if err := json.Unmarshal(data, &lockfile); err != nil {
		return nil, err
	}

	return &lockfile, nil
}
