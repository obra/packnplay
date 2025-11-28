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

// PortAttributes represents attributes for a specific port
type PortAttributes struct {
	Label         string `json:"label,omitempty"`         // User-visible label for the port
	Protocol      string `json:"protocol,omitempty"`      // http or https
	OnAutoForward string `json:"onAutoForward,omitempty"` // notify, openBrowser, openBrowserOnce, openPreview, silent, ignore
}

// HostRequirements represents minimum host system requirements (advisory only)
type HostRequirements struct {
	Cpus    *int    `json:"cpus,omitempty"`    // Minimum CPU cores
	Memory  *string `json:"memory,omitempty"`  // Minimum RAM (e.g., "8gb")
	Storage *string `json:"storage,omitempty"` // Minimum disk space (e.g., "32gb")
	Gpu     *bool   `json:"gpu,omitempty"`     // Requires GPU
}

// Config represents a parsed devcontainer.json
type Config struct {
	// Basic container configuration
	Image        string                 `json:"image"`
	DockerFile   string                 `json:"dockerFile"`
	Build        *BuildConfig           `json:"build,omitempty"`
	Name         string                 `json:"name,omitempty"`          // Display name for the dev container
	ContainerUser   string                    `json:"containerUser,omitempty"` // User for container operations (docker run --user)
	RemoteUser      string                    `json:"remoteUser"`              // User for remote operations (docker exec --user)
	ContainerEnv    map[string]string         `json:"containerEnv,omitempty"`
	RemoteEnv       map[string]string         `json:"remoteEnv,omitempty"`
	ForwardPorts    []interface{}             `json:"forwardPorts,omitempty"`    // int or string
	PortsAttributes map[string]PortAttributes `json:"portsAttributes,omitempty"` // Port-specific metadata
	Mounts          []string                  `json:"mounts,omitempty"`          // Docker mount syntax
	RunArgs         []string                  `json:"runArgs,omitempty"`         // Additional docker run arguments
	Features        map[string]interface{}    `json:"features,omitempty"`

	// Docker Compose orchestration (alternative to image/dockerfile)
	DockerComposeFile interface{} `json:"dockerComposeFile,omitempty"` // string or []string - path(s) to compose file(s)
	Service           string      `json:"service,omitempty"`           // Service name to connect to
	RunServices       []string    `json:"runServices,omitempty"`       // Services to start (empty = all services)

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

	// Host requirements (advisory validation only)
	HostRequirements *HostRequirements `json:"hostRequirements,omitempty"`
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

// GetDockerComposeFiles returns dockerComposeFile as a string slice
// Handles both string and []string JSON values
func (c *Config) GetDockerComposeFiles() []string {
	if c.DockerComposeFile == nil {
		return nil
	}

	switch v := c.DockerComposeFile.(type) {
	case string:
		return []string{v}
	case []interface{}:
		result := make([]string, 0, len(v))
		for _, item := range v {
			if str, ok := item.(string); ok {
				result = append(result, str)
			}
		}
		return result
	case []string:
		return v
	default:
		return nil
	}
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
