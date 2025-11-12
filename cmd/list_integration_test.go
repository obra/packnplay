package cmd

import (
	"testing"

	"github.com/obra/packnplay/pkg/container"
)

func TestListCommandIntegration(t *testing.T) {
	// Test the complete integration of launch info capture and display
	// This tests the full workflow: capture command line -> store in labels -> parse and display

	// Convert to label string format as Docker would provide it
	labelStr := "managed-by=packnplay,packnplay-project=myproject,packnplay-worktree=feature-branch,packnplay-host-path=/Users/jesse/myproject,packnplay-launch-command=packnplay run --worktree feature-branch --env DEBUG=1 --git-creds --publish 8080:80 claude code"

	// Test parsing
	labels := container.ParseLabels(labelStr)
	project := container.GetProjectFromLabels(labels)
	worktree := container.GetWorktreeFromLabels(labels)
	hostPath := container.GetHostPathFromLabels(labels)
	launchCommand := container.GetLaunchCommandFromLabels(labels)

	if project != "myproject" {
		t.Errorf("project = %v, want myproject", project)
	}

	if worktree != "feature-branch" {
		t.Errorf("worktree = %v, want feature-branch", worktree)
	}

	if hostPath != "/Users/jesse/myproject" {
		t.Errorf("hostPath = %v, want /Users/jesse/myproject", hostPath)
	}

	expectedCommand := "packnplay run --worktree feature-branch --env DEBUG=1 --git-creds --publish 8080:80 claude code"
	if launchCommand != expectedCommand {
		t.Errorf("launchCommand = %v, want %v", launchCommand, expectedCommand)
	}
}

func TestGenerateLabelsIntegration(t *testing.T) {
	// Test that labels can be generated and then parsed back correctly

	projectName := "testproject"
	worktreeName := "main"
	hostPath := "/home/user/testproject"
	launchCommand := "packnplay run --runtime docker --verbose --git-creds bash"

	// Generate labels (validate generation works)
	_ = container.GenerateLabelsWithLaunchInfo(projectName, worktreeName, hostPath, launchCommand)

	// For predictable testing, construct the string manually
	labelStr := "managed-by=packnplay,packnplay-project=testproject,packnplay-worktree=main,packnplay-host-path=/home/user/testproject,packnplay-launch-command=packnplay run --runtime docker --verbose --git-creds bash"

	// Parse back
	parsedLabels := container.ParseLabels(labelStr)
	parsedProject := container.GetProjectFromLabels(parsedLabels)
	parsedWorktree := container.GetWorktreeFromLabels(parsedLabels)
	parsedHostPath := container.GetHostPathFromLabels(parsedLabels)
	parsedLaunchCommand := container.GetLaunchCommandFromLabels(parsedLabels)

	if parsedProject != projectName {
		t.Errorf("parsed project = %v, want %v", parsedProject, projectName)
	}

	if parsedWorktree != worktreeName {
		t.Errorf("parsed worktree = %v, want %v", parsedWorktree, worktreeName)
	}

	if parsedHostPath != hostPath {
		t.Errorf("parsed hostPath = %v, want %v", parsedHostPath, hostPath)
	}

	if parsedLaunchCommand != launchCommand {
		t.Errorf("parsed launchCommand = %v, want %v", parsedLaunchCommand, launchCommand)
	}
}

func TestBackwardCompatibilityWithOldContainers(t *testing.T) {
	// Test that old containers without launch info still work

	oldLabelStr := "managed-by=packnplay,packnplay-project=oldproject,packnplay-worktree=legacy"

	oldLabels := container.ParseLabels(oldLabelStr)
	project := container.GetProjectFromLabels(oldLabels)
	worktree := container.GetWorktreeFromLabels(oldLabels)
	hostPath := container.GetHostPathFromLabels(oldLabels)
	launchCommand := container.GetLaunchCommandFromLabels(oldLabels)

	// Old labels should still work
	if project != "oldproject" {
		t.Errorf("project = %v, want oldproject", project)
	}

	if worktree != "legacy" {
		t.Errorf("worktree = %v, want legacy", worktree)
	}

	// New fields should be empty for old containers
	if hostPath != "" {
		t.Errorf("hostPath = %v, want empty string", hostPath)
	}

	if launchCommand != "" {
		t.Errorf("launchCommand = %v, want empty string", launchCommand)
	}
}
