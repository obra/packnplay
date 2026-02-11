package container

import (
	"testing"
)

func TestGenerateContainerName(t *testing.T) {
	tests := []struct {
		name         string
		projectPath  string
		worktreeName string
		want         string
	}{
		{
			name:         "basic naming",
			projectPath:  "/home/user/myproject",
			worktreeName: "main",
			want:         "packnplay-myproject-main",
		},
		{
			name:         "sanitized worktree name",
			projectPath:  "/home/user/myproject",
			worktreeName: "feature/auth",
			want:         "packnplay-myproject-feature-auth",
		},
		{
			name:         "worktree with @ symbol",
			projectPath:  "/home/user/myproject",
			worktreeName: "user@team-PROJ-147",
			want:         "packnplay-myproject-user-team-PROJ-147",
		},
		{
			name:         "project path with special chars",
			projectPath:  "/home/user/my@project",
			worktreeName: "main",
			want:         "packnplay-my-project-main",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateContainerName(tt.projectPath, tt.worktreeName)
			if got != tt.want {
				t.Errorf("GenerateContainerName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGenerateLabels(t *testing.T) {
	labels := GenerateLabels("myproject", "feature-auth")

	if labels["managed-by"] != "packnplay" {
		t.Errorf("managed-by label = %v, want packnplay", labels["managed-by"])
	}

	if labels["packnplay-project"] != "myproject" {
		t.Errorf("packnplay-project label = %v, want myproject", labels["packnplay-project"])
	}

	if labels["packnplay-worktree"] != "feature-auth" {
		t.Errorf("packnplay-worktree label = %v, want feature-auth", labels["packnplay-worktree"])
	}
}

func TestGenerateLabelsWithLaunchInfo(t *testing.T) {
	hostPath := "/Users/jesse/myproject"
	launchCommand := "packnplay run --worktree feature --env DEBUG=1 --git-creds claude code"

	labels := GenerateLabelsWithLaunchInfo("myproject", "feature-auth", hostPath, launchCommand)

	// Test existing labels still work
	if labels["managed-by"] != "packnplay" {
		t.Errorf("managed-by label = %v, want packnplay", labels["managed-by"])
	}

	if labels["packnplay-project"] != "myproject" {
		t.Errorf("packnplay-project label = %v, want myproject", labels["packnplay-project"])
	}

	if labels["packnplay-worktree"] != "feature-auth" {
		t.Errorf("packnplay-worktree label = %v, want feature-auth", labels["packnplay-worktree"])
	}

	// Test new labels
	if labels["packnplay-host-path"] != hostPath {
		t.Errorf("packnplay-host-path label = %v, want %v", labels["packnplay-host-path"], hostPath)
	}

	if labels["packnplay-launch-command"] != launchCommand {
		t.Errorf("packnplay-launch-command label = %v, want %v", labels["packnplay-launch-command"], launchCommand)
	}
}
