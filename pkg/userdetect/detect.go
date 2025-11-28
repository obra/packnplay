package userdetect

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// DevcontainerConfig represents the relevant parts of devcontainer.json for user detection
type DevcontainerConfig struct {
	RemoteUser   string `json:"remoteUser,omitempty"`
	UserEnvProbe string `json:"userEnvProbe,omitempty"`
}

// UserDetectionResult contains the detected user and metadata about how it was detected
type UserDetectionResult struct {
	User    string `json:"user"`
	Source  string `json:"source"` // "devcontainer", "image_default", "runtime_detection", "fallback"
	HomeDir string `json:"homeDir"`
}

// UserInfo represents a user found in the container
type UserInfo struct {
	Username string `json:"username"`
	UID      string `json:"uid"`
	GID      string `json:"gid"`
	HomeDir  string `json:"homeDir"`
}

// CachedUserResult stores cached user detection results
type CachedUserResult struct {
	ImageID   string `json:"imageId"`
	User      string `json:"user"`
	HomeDir   string `json:"homeDir"`
	Source    string `json:"source"`
	Timestamp int64  `json:"timestamp"`
}

// DetectContainerUser determines the best user to use for a container
// Priority: devcontainer.json > cached result > runtime detection > fallback
func DetectContainerUser(image string, devcontainer *DevcontainerConfig) (*UserDetectionResult, error) {
	// Extract userEnvProbe setting (defaults to empty string if not set)
	var userEnvProbe string
	if devcontainer != nil {
		userEnvProbe = devcontainer.UserEnvProbe
	}

	// 1. Check devcontainer.json first
	if devcontainer != nil && devcontainer.RemoteUser != "" {
		homeDir := "/root"
		if devcontainer.RemoteUser != "root" {
			homeDir = "/home/" + devcontainer.RemoteUser
		}
		return &UserDetectionResult{
			User:    devcontainer.RemoteUser,
			Source:  "devcontainer",
			HomeDir: homeDir,
		}, nil
	}

	// 2. Get image ID for caching
	imageID, err := getImageID(image)
	if err != nil {
		return nil, fmt.Errorf("failed to get image ID for %s: %w", image, err)
	}

	// 3. Check cache first
	if cached := getCachedUserResult(imageID); cached != nil {
		return &UserDetectionResult{
			User:    cached.User,
			Source:  cached.Source,
			HomeDir: cached.HomeDir,
		}, nil
	}

	// 4. Do direct runtime detection with userEnvProbe setting
	result, err := detectRuntimeUserDirectWithProbe(image, userEnvProbe)
	if err != nil {
		// Fallback to root if detection fails
		result = &UserDetectionResult{
			User:    "root",
			Source:  "fallback",
			HomeDir: "/root",
		}
	}

	// 5. Cache the result
	cacheUserResult(imageID, result)

	return result, nil
}

// DetectUsersInImage finds all users that exist in the given image
func DetectUsersInImage(image string) ([]UserInfo, error) {
	// Run container briefly to examine /etc/passwd
	cmd := exec.Command("docker", "run", "--rm", image, "cat", "/etc/passwd")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to examine users in image %s: %w", image, err)
	}

	var users []UserInfo
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		parts := strings.Split(line, ":")
		if len(parts) >= 6 {
			users = append(users, UserInfo{
				Username: parts[0],
				UID:      parts[2],
				GID:      parts[3],
				HomeDir:  parts[5],
			})
		}
	}

	return users, nil
}

// GetImageDefaultUser gets the default user from Docker image config
func GetImageDefaultUser(image string) (string, error) {
	cmd := exec.Command("docker", "image", "inspect", image, "--format", "{{.Config.User}}")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to inspect image %s: %w", image, err)
	}

	user := strings.TrimSpace(string(output))
	if user == "" {
		return "root", nil // Empty user means root
	}

	return user, nil
}

// getShellFlags returns the appropriate shell flags based on userEnvProbe setting
func getShellFlags(userEnvProbe string) []string {
	switch userEnvProbe {
	case "none":
		return []string{}
	case "loginShell":
		return []string{"-l"}
	case "interactiveShell":
		return []string{"-i"}
	case "loginInteractiveShell":
		return []string{"-l", "-i"}
	default:
		// Default to loginInteractiveShell for empty or unknown values
		return []string{"-l", "-i"}
	}
}

// detectRuntimeUserDirect asks the container directly what user it runs as
func detectRuntimeUserDirect(image string) (*UserDetectionResult, error) {
	return detectRuntimeUserDirectWithProbe(image, "")
}

// detectRuntimeUserDirectWithProbe asks the container directly what user it runs as
// using the specified userEnvProbe setting
func detectRuntimeUserDirectWithProbe(image string, userEnvProbe string) (*UserDetectionResult, error) {
	shellFlags := getShellFlags(userEnvProbe)

	// Build command: docker run --rm <image> sh <flags> -c "whoami && echo $HOME"
	args := []string{"run", "--rm", image, "sh"}
	args = append(args, shellFlags...)
	args = append(args, "-c", "whoami && echo $HOME")

	cmd := exec.Command("docker", args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to detect user in image %s: %w", image, err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) != 2 {
		return nil, fmt.Errorf("unexpected output from user detection in image %s: %s", image, string(output))
	}

	user := strings.TrimSpace(lines[0])
	homeDir := strings.TrimSpace(lines[1])

	if user == "" {
		return nil, fmt.Errorf("empty user returned from image %s", image)
	}

	return &UserDetectionResult{
		User:    user,
		Source:  "runtime_detection",
		HomeDir: homeDir,
	}, nil
}

// getImageID gets the image ID for caching purposes
func getImageID(image string) (string, error) {
	cmd := exec.Command("docker", "image", "inspect", image, "--format", "{{.Id}}")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get image ID for %s: %w", image, err)
	}

	imageID := strings.TrimSpace(string(output))
	if imageID == "" {
		return "", fmt.Errorf("empty image ID returned for %s", image)
	}

	return imageID, nil
}

// getCacheDir returns the directory for user detection cache
func getCacheDir() (string, error) {
	cacheDir := os.Getenv("XDG_CACHE_HOME")
	if cacheDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		cacheDir = filepath.Join(homeDir, ".cache")
	}

	packnplayCacheDir := filepath.Join(cacheDir, "packnplay", "userdetect")
	err := os.MkdirAll(packnplayCacheDir, 0755)
	if err != nil {
		return "", fmt.Errorf("failed to create cache directory: %w", err)
	}

	return packnplayCacheDir, nil
}

// getCacheFilePath returns the cache file path for a given image ID
func getCacheFilePath(imageID string) (string, error) {
	cacheDir, err := getCacheDir()
	if err != nil {
		return "", err
	}

	// Use hash of image ID as filename to avoid filesystem issues
	hash := sha256.Sum256([]byte(imageID))
	filename := fmt.Sprintf("%x.json", hash)

	return filepath.Join(cacheDir, filename), nil
}

// getCachedUserResult retrieves cached user detection result
func getCachedUserResult(imageID string) *CachedUserResult {
	cacheFilePath, err := getCacheFilePath(imageID)
	if err != nil {
		return nil
	}

	data, err := os.ReadFile(cacheFilePath)
	if err != nil {
		return nil // Cache miss
	}

	var cached CachedUserResult
	if err := json.Unmarshal(data, &cached); err != nil {
		return nil // Invalid cache
	}

	// Verify imageID matches (sanity check)
	if cached.ImageID != imageID {
		return nil
	}

	return &cached
}

// cacheUserResult stores user detection result in cache
func cacheUserResult(imageID string, result *UserDetectionResult) {
	cacheFilePath, err := getCacheFilePath(imageID)
	if err != nil {
		return // Silently fail cache writes
	}

	cached := CachedUserResult{
		ImageID:   imageID,
		User:      result.User,
		HomeDir:   result.HomeDir,
		Source:    result.Source,
		Timestamp: 0, // Could add timestamp for cache expiry later
	}

	data, err := json.Marshal(cached)
	if err != nil {
		return
	}

	// Write atomically using temp file
	tempFile := cacheFilePath + ".tmp"
	if err := os.WriteFile(tempFile, data, 0644); err != nil {
		return
	}

	if err := os.Rename(tempFile, cacheFilePath); err != nil {
		_ = os.Remove(tempFile) // Cleanup on failure
	}
}
