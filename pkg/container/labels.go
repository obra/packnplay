package container

import (
	"strings"
)

// Label key constants for packnplay container labels
const (
	LabelProject       = "packnplay-project"
	LabelWorktree      = "packnplay-worktree"
	LabelHostPath      = "packnplay-host-path"
	LabelLaunchCommand = "packnplay-launch-command"
	LabelManagedBy     = "managed-by"
)

// ParseLabels parses a comma-separated label string into a map.
// This consolidates 3 duplicate implementations across the codebase:
// - runner.go:762-782 parseLabelsFromString
// - list.go:140-155 parseLabels
// - list.go:157-176 parseLabelsWithLaunchInfo
//
// Format: "key1=value1,key2=value2,key3=value3"
// Returns: map[string]string with parsed key-value pairs
func ParseLabels(labelString string) map[string]string {
	labels := make(map[string]string)

	if labelString == "" {
		return labels
	}

	pairs := strings.Split(labelString, ",")
	for _, pair := range pairs {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) == 2 {
			labels[parts[0]] = parts[1]
		}
	}

	return labels
}

// GetProjectFromLabels extracts the project name from label map
func GetProjectFromLabels(labels map[string]string) string {
	return labels[LabelProject]
}

// GetWorktreeFromLabels extracts the worktree name from label map
func GetWorktreeFromLabels(labels map[string]string) string {
	return labels[LabelWorktree]
}

// GetHostPathFromLabels extracts the host path from label map
func GetHostPathFromLabels(labels map[string]string) string {
	return labels[LabelHostPath]
}

// GetLaunchCommandFromLabels extracts the launch command from label map
func GetLaunchCommandFromLabels(labels map[string]string) string {
	return labels[LabelLaunchCommand]
}
