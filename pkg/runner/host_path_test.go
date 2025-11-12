package runner

import (
	"strings"
	"testing"

	"github.com/obra/packnplay/pkg/config"
)

func TestHostPathPreservation(t *testing.T) {
	// Test that containers mount the project at the same host path, not /workspace
	hostPath := "/Users/jesse/Documents/GitHub/myproject"

	runConfig := &RunConfig{
		Path:          hostPath,
		Worktree:      "main",
		Command:       []string{"bash"},
		Runtime:       "docker",
		DefaultImage:  "ubuntu:22.04",
		Credentials:   config.Credentials{},
		HostPath:      hostPath,
		LaunchCommand: "packnplay run bash",
	}

	// Get the mount arguments that would be generated
	mountArgs := generateMountArguments(runConfig, "myproject", "main")

	// Should mount at host path, not /workspace
	expectedMount := hostPath + ":" + hostPath
	found := false
	for i, arg := range mountArgs {
		if arg == "-v" && i+1 < len(mountArgs) {
			if strings.Contains(mountArgs[i+1], expectedMount) {
				found = true
				break
			}
		}
	}

	if !found {
		t.Errorf("Expected mount %s not found in args: %v", expectedMount, mountArgs)
	}

	// Should not mount at /workspace
	workspaceMount := hostPath + ":/workspace"
	for i, arg := range mountArgs {
		if arg == "-v" && i+1 < len(mountArgs) {
			if strings.Contains(mountArgs[i+1], workspaceMount) {
				t.Errorf("Found old /workspace mount %s in args: %v", workspaceMount, mountArgs)
			}
		}
	}
}

func TestWorkingDirectoryPreservation(t *testing.T) {
	// Test that working directory is set to host path, not /workspace
	hostPath := "/Users/jesse/Documents/GitHub/myproject"

	runConfig := &RunConfig{
		Path:          hostPath,
		Worktree:      "feature-branch",
		Command:       []string{"ls", "-la"},
		Runtime:       "docker",
		DefaultImage:  "ubuntu:22.04",
		Credentials:   config.Credentials{},
		HostPath:      hostPath,
		LaunchCommand: "packnplay run ls -la",
	}

	// Get the working directory that would be set
	workingDir := getWorkingDirectory(runConfig)

	if workingDir != hostPath {
		t.Errorf("Working directory = %v, want %v", workingDir, hostPath)
	}

	if workingDir == "/workspace" {
		t.Error("Working directory should not be /workspace anymore")
	}
}

func TestExecArguments(t *testing.T) {
	// Test that exec commands use host path for working directory
	hostPath := "/Users/jesse/Documents/GitHub/myproject"

	execArgs := generateExecArguments("container123", []string{"git", "status"}, hostPath)

	// Should include working directory flag with host path
	found := false
	for i, arg := range execArgs {
		if arg == "-w" && i+1 < len(execArgs) {
			if execArgs[i+1] == hostPath {
				found = true
				break
			}
		}
	}

	if !found {
		t.Errorf("Expected -w %s not found in exec args: %v", hostPath, execArgs)
	}

	// Should not use /workspace
	for i, arg := range execArgs {
		if arg == "-w" && i+1 < len(execArgs) {
			if execArgs[i+1] == "/workspace" {
				t.Errorf("Found old /workspace working dir in exec args: %v", execArgs)
			}
		}
	}
}

func TestDirectoryCreation(t *testing.T) {
	// Test that deep host paths get proper directory creation in container
	hostPath := "/Users/jesse/Documents/GitHub/very/deep/project/path"

	// Should generate commands to create the parent directory structure
	dirCommands := generateDirectoryCreationCommands(hostPath)

	if len(dirCommands) == 0 {
		t.Error("Should generate directory creation commands for deep paths")
	}

	// Should create parent directories
	expectedCommand := []string{"mkdir", "-p", "/Users/jesse/Documents/GitHub/very/deep/project"}
	if !commandsContain(dirCommands, expectedCommand) {
		t.Errorf("Expected mkdir command %v not found in: %v", expectedCommand, dirCommands)
	}
}

// Helper function to check if commands contain expected command
func commandsContain(commands [][]string, expected []string) bool {
	for _, cmd := range commands {
		if len(cmd) == len(expected) {
			match := true
			for i, arg := range expected {
				if cmd[i] != arg {
					match = false
					break
				}
			}
			if match {
				return true
			}
		}
	}
	return false
}
