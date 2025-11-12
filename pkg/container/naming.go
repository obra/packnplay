package container

import (
	"fmt"
	"path/filepath"
	"strings"
)

// GenerateContainerName creates a container name from project and worktree
func GenerateContainerName(projectPath, worktreeName string) string {
	projectName := filepath.Base(projectPath)
	sanitizedWorktree := sanitizeName(worktreeName)
	return fmt.Sprintf("packnplay-%s-%s", projectName, sanitizedWorktree)
}

// GenerateImageName creates an image name for a built devcontainer
// Docker image names must be lowercase
func GenerateImageName(projectPath string) string {
	projectName := strings.ToLower(filepath.Base(projectPath))
	return fmt.Sprintf("packnplay-%s-devcontainer:latest", projectName)
}

// sanitizeName converts a name to docker-compatible format
func sanitizeName(name string) string {
	// Docker container names: [a-zA-Z0-9][a-zA-Z0-9_.-]*
	name = strings.ReplaceAll(name, "/", "-")
	name = strings.ReplaceAll(name, " ", "-")
	name = strings.ReplaceAll(name, ":", "-")
	return name
}

// GenerateLabels creates Docker labels for packnplay-managed containers
func GenerateLabels(projectName, worktreeName string) map[string]string {
	return map[string]string{
		"managed-by":         "packnplay",
		"packnplay-project":  projectName,
		"packnplay-worktree": worktreeName,
	}
}

// GenerateLabelsWithLaunchInfo creates Docker labels including host path and launch command
func GenerateLabelsWithLaunchInfo(projectName, worktreeName, hostPath, launchCommand string) map[string]string {
	return map[string]string{
		"managed-by":               "packnplay",
		"packnplay-project":        projectName,
		"packnplay-worktree":       worktreeName,
		"packnplay-host-path":      hostPath,
		"packnplay-launch-command": launchCommand,
	}
}

// LabelsToArgs converts label map to docker --label args
func LabelsToArgs(labels map[string]string) []string {
	args := make([]string, 0, len(labels)*2)
	for k, v := range labels {
		args = append(args, "--label", fmt.Sprintf("%s=%s", k, v))
	}
	return args
}
