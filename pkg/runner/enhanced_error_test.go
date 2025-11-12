package runner

import (
	"testing"

	"github.com/obra/packnplay/pkg/container"
)

func TestParseLabelsFromString(t *testing.T) {
	testCases := []struct {
		name              string
		labels            string
		expectedProject   string
		expectedWorktree  string
		expectedHostPath  string
		expectedLaunchCmd string
	}{
		{
			name:              "complete labels",
			labels:            "managed-by=packnplay,packnplay-project=myproject,packnplay-worktree=main,packnplay-host-path=/Users/jesse/myproject,packnplay-launch-command=packnplay run --git-creds bash",
			expectedProject:   "myproject",
			expectedWorktree:  "main",
			expectedHostPath:  "/Users/jesse/myproject",
			expectedLaunchCmd: "packnplay run --git-creds bash",
		},
		{
			name:              "minimal labels",
			labels:            "managed-by=packnplay,packnplay-project=simple,packnplay-worktree=feature",
			expectedProject:   "simple",
			expectedWorktree:  "feature",
			expectedHostPath:  "",
			expectedLaunchCmd: "",
		},
		{
			name:              "empty labels",
			labels:            "",
			expectedProject:   "",
			expectedWorktree:  "",
			expectedHostPath:  "",
			expectedLaunchCmd: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			labels := container.ParseLabels(tc.labels)
			project := container.GetProjectFromLabels(labels)
			worktree := container.GetWorktreeFromLabels(labels)
			hostPath := container.GetHostPathFromLabels(labels)
			launchCmd := container.GetLaunchCommandFromLabels(labels)

			if project != tc.expectedProject {
				t.Errorf("project = %v, want %v", project, tc.expectedProject)
			}
			if worktree != tc.expectedWorktree {
				t.Errorf("worktree = %v, want %v", worktree, tc.expectedWorktree)
			}
			if hostPath != tc.expectedHostPath {
				t.Errorf("hostPath = %v, want %v", hostPath, tc.expectedHostPath)
			}
			if launchCmd != tc.expectedLaunchCmd {
				t.Errorf("launchCmd = %v, want %v", launchCmd, tc.expectedLaunchCmd)
			}
		})
	}
}

func TestContainerDetails(t *testing.T) {
	// Test the ContainerDetails struct
	details := ContainerDetails{
		Names:         "packnplay-myproject-main",
		Status:        "Up 5 minutes",
		Project:       "myproject",
		Worktree:      "main",
		HostPath:      "/Users/jesse/myproject",
		LaunchCommand: "packnplay run --git-creds claude code",
	}

	if details.Names != "packnplay-myproject-main" {
		t.Errorf("Names = %v, want packnplay-myproject-main", details.Names)
	}

	if details.HostPath != "/Users/jesse/myproject" {
		t.Errorf("HostPath = %v, want /Users/jesse/myproject", details.HostPath)
	}

	if details.LaunchCommand != "packnplay run --git-creds claude code" {
		t.Errorf("LaunchCommand = %v, want packnplay run --git-creds claude code", details.LaunchCommand)
	}
}
