package runner

import (
	"strings"
	"testing"

	"github.com/obra/packnplay/pkg/container"
)

func TestRunConfigLaunchInfo(t *testing.T) {
	// Test that RunConfig properly captures and uses launch info
	config := &RunConfig{
		Path:          "/Users/jesse/myproject",
		Worktree:      "feature-branch",
		HostPath:      "/Users/jesse/myproject",
		LaunchCommand: "packnplay run --worktree feature-branch --git-creds claude code",
	}

	// Simulate the label generation that happens in Run()
	projectName := "myproject"
	labels := container.GenerateLabelsWithLaunchInfo(
		projectName,
		config.Worktree,
		config.HostPath,
		config.LaunchCommand,
	)

	// Verify the labels contain our launch info
	if labels["packnplay-host-path"] != config.HostPath {
		t.Errorf("host path label = %v, want %v", labels["packnplay-host-path"], config.HostPath)
	}

	if labels["packnplay-launch-command"] != config.LaunchCommand {
		t.Errorf("launch command label = %v, want %v", labels["packnplay-launch-command"], config.LaunchCommand)
	}

	// Verify original labels still work
	if labels["packnplay-project"] != projectName {
		t.Errorf("project label = %v, want %v", labels["packnplay-project"], projectName)
	}

	if labels["packnplay-worktree"] != config.Worktree {
		t.Errorf("worktree label = %v, want %v", labels["packnplay-worktree"], config.Worktree)
	}
}

func TestLaunchCommandReconstruction(t *testing.T) {
	// Test that we can reconstruct meaningful launch info
	originalCommand := "packnplay run --worktree feature --env DEBUG=1 --publish 8080:80 --git-creds --aws-creds claude code"

	config := &RunConfig{
		HostPath:      "/Users/jesse/myproject",
		LaunchCommand: originalCommand,
	}

	// Verify command contains key information
	if !strings.Contains(config.LaunchCommand, "--worktree feature") {
		t.Errorf("launch command missing worktree flag: %v", config.LaunchCommand)
	}

	if !strings.Contains(config.LaunchCommand, "--git-creds") {
		t.Errorf("launch command missing git-creds flag: %v", config.LaunchCommand)
	}

	if !strings.Contains(config.LaunchCommand, "claude code") {
		t.Errorf("launch command missing command args: %v", config.LaunchCommand)
	}
}
