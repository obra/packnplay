package container

import (
	"testing"
)

func TestParseLabels_Basic(t *testing.T) {
	labelString := "packnplay-project=myproject,packnplay-worktree=feature-branch"

	labels := ParseLabels(labelString)

	if labels["packnplay-project"] != "myproject" {
		t.Errorf("Expected project=myproject, got %s", labels["packnplay-project"])
	}

	if labels["packnplay-worktree"] != "feature-branch" {
		t.Errorf("Expected worktree=feature-branch, got %s", labels["packnplay-worktree"])
	}
}

func TestParseLabels_WithAllFields(t *testing.T) {
	labelString := "packnplay-project=proj,packnplay-worktree=wt,packnplay-host-path=/path,packnplay-launch-command=bash"

	labels := ParseLabels(labelString)

	if len(labels) != 4 {
		t.Errorf("Expected 4 labels, got %d", len(labels))
	}

	if labels["packnplay-project"] != "proj" {
		t.Errorf("Expected project=proj, got %s", labels["packnplay-project"])
	}

	if labels["packnplay-worktree"] != "wt" {
		t.Errorf("Expected worktree=wt, got %s", labels["packnplay-worktree"])
	}

	if labels["packnplay-host-path"] != "/path" {
		t.Errorf("Expected host-path=/path, got %s", labels["packnplay-host-path"])
	}

	if labels["packnplay-launch-command"] != "bash" {
		t.Errorf("Expected launch-command=bash, got %s", labels["packnplay-launch-command"])
	}
}

func TestParseLabels_EmptyString(t *testing.T) {
	labelString := ""

	labels := ParseLabels(labelString)

	if len(labels) != 0 {
		t.Errorf("Expected 0 labels for empty string, got %d", len(labels))
	}
}

func TestParseLabels_MalformedPairs(t *testing.T) {
	labelString := "packnplay-project=myproject,invalid-no-equals,packnplay-worktree=branch"

	labels := ParseLabels(labelString)

	// Should skip malformed pair and only get valid ones
	if len(labels) != 2 {
		t.Errorf("Expected 2 labels (skipping malformed), got %d", len(labels))
	}

	if labels["packnplay-project"] != "myproject" {
		t.Errorf("Expected project=myproject, got %s", labels["packnplay-project"])
	}

	if labels["packnplay-worktree"] != "branch" {
		t.Errorf("Expected worktree=branch, got %s", labels["packnplay-worktree"])
	}
}

func TestGetProjectFromLabels(t *testing.T) {
	labels := map[string]string{
		"packnplay-project": "myproject",
		"other-label":       "value",
	}

	project := GetProjectFromLabels(labels)
	if project != "myproject" {
		t.Errorf("Expected myproject, got %s", project)
	}
}

func TestGetProjectFromLabels_Missing(t *testing.T) {
	labels := map[string]string{
		"other-label": "value",
	}

	project := GetProjectFromLabels(labels)
	if project != "" {
		t.Errorf("Expected empty string for missing project, got %s", project)
	}
}

func TestGetWorktreeFromLabels(t *testing.T) {
	labels := map[string]string{
		"packnplay-worktree": "feature-branch",
	}

	worktree := GetWorktreeFromLabels(labels)
	if worktree != "feature-branch" {
		t.Errorf("Expected feature-branch, got %s", worktree)
	}
}

func TestGetHostPathFromLabels(t *testing.T) {
	labels := map[string]string{
		"packnplay-host-path": "/home/user/project",
	}

	hostPath := GetHostPathFromLabels(labels)
	if hostPath != "/home/user/project" {
		t.Errorf("Expected /home/user/project, got %s", hostPath)
	}
}

func TestGetLaunchCommandFromLabels(t *testing.T) {
	labels := map[string]string{
		"packnplay-launch-command": "bash -c 'echo hello'",
	}

	launchCommand := GetLaunchCommandFromLabels(labels)
	if launchCommand != "bash -c 'echo hello'" {
		t.Errorf("Expected bash -c 'echo hello', got %s", launchCommand)
	}
}
