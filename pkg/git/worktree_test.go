package git

import (
	"testing"
)

func TestDetermineWorktreePath(t *testing.T) {
	tests := []struct {
		name         string
		projectPath  string
		worktreeName string
		wantContains []string
	}{
		{
			name:         "basic worktree path",
			projectPath:  "/home/user/myproject",
			worktreeName: "feature-auth",
			wantContains: []string{"packnplay/worktrees", "myproject", "feature-auth"},
		},
		{
			name:         "sanitize slashes in branch name",
			projectPath:  "/home/user/myproject",
			worktreeName: "feature/auth",
			wantContains: []string{"packnplay/worktrees", "myproject", "feature-auth"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetermineWorktreePath(tt.projectPath, tt.worktreeName)

			for _, want := range tt.wantContains {
				if !contains(got, want) {
					t.Errorf("DetermineWorktreePath() = %v, want to contain %v", got, want)
				}
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr || len(s) > len(substr) &&
			(s[0:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
				findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
