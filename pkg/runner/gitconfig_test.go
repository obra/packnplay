package runner

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestIsSSHURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want bool
	}{
		// SCP-style SSH URLs
		{name: "scp style github", url: "git@github.com:", want: true},
		{name: "scp style gitlab", url: "git@gitlab.com:", want: true},
		{name: "scp style with path", url: "git@github.com:user/repo", want: true},
		{name: "scp style custom user", url: "deploy@myhost.com:repos/", want: true},

		// URL-style SSH URLs
		{name: "ssh scheme", url: "ssh://git@github.com/", want: true},
		{name: "ssh scheme with port", url: "ssh://git@github.com:22/user/repo", want: true},
		{name: "ssh scheme no user", url: "ssh://github.com/user/repo", want: true},

		// Non-SSH URLs
		{name: "https github", url: "https://github.com/", want: false},
		{name: "http github", url: "http://github.com/", want: false},
		{name: "https with user", url: "https://user@github.com/repo", want: false},
		{name: "file protocol", url: "file:///path/to/repo", want: false},
		{name: "bare path", url: "/path/to/repo", want: false},
		{name: "empty string", url: "", want: false},

		// Edge cases
		{name: "git protocol", url: "git://github.com/user/repo", want: false},
		{name: "ftp protocol", url: "ftp://example.com/repo", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isSSHURL(tt.url)
			if got != tt.want {
				t.Errorf("isSSHURL(%q) = %v, want %v", tt.url, got, tt.want)
			}
		})
	}
}

// writeGitconfig creates a temporary gitconfig file with the given content and
// sets GIT_CONFIG_GLOBAL to point at it for the duration of the test.
func writeGitconfig(t *testing.T, content string) {
	t.Helper()
	f, err := os.CreateTemp("", "gitconfig-test-*")
	if err != nil {
		t.Fatalf("failed to create temp gitconfig: %v", err)
	}
	t.Cleanup(func() { os.Remove(f.Name()) })
	if _, err := f.WriteString(content); err != nil {
		t.Fatalf("failed to write temp gitconfig: %v", err)
	}
	f.Close()
	t.Setenv("GIT_CONFIG_GLOBAL", f.Name())
}

func TestHasSSHInsteadOfRules(t *testing.T) {
	tests := []struct {
		name          string
		gitconfig     string
		wantSubstring []string // each must appear in at least one rewrite string
		wantCount     int
	}{
		{
			name: "SCP-style insteadOf",
			gitconfig: `[url "git@github.com:"]
	insteadOf = https://github.com/
`,
			wantSubstring: []string{"https://github.com/", "git@github.com:"},
			wantCount:     1,
		},
		{
			name: "ssh:// URL-style insteadOf",
			gitconfig: `[url "ssh://git@github.com/"]
	insteadOf = https://github.com/
`,
			wantSubstring: []string{"https://github.com/", "ssh://git@github.com/"},
			wantCount:     1,
		},
		{
			name: "pushinsteadOf with SSH target",
			gitconfig: `[url "git@github.com:"]
	pushInsteadOf = https://github.com/
`,
			wantSubstring: []string{"https://github.com/", "git@github.com:"},
			wantCount:     1,
		},
		{
			name: "multiple SSH insteadOf rules",
			gitconfig: `[url "git@github.com:"]
	insteadOf = https://github.com/
[url "git@gitlab.com:"]
	insteadOf = https://gitlab.com/
`,
			wantSubstring: []string{"github.com", "gitlab.com"},
			wantCount:     2,
		},
		{
			name: "HTTPS insteadOf not SSH — should not trigger",
			gitconfig: `[url "https://github.com/"]
	insteadOf = git://github.com/
`,
			wantCount: 0,
		},
		{
			name:      "empty gitconfig",
			gitconfig: "",
			wantCount: 0,
		},
		{
			name: "unrelated git config — no URL rewrites",
			gitconfig: `[core]
	autocrlf = false
[user]
	email = test@example.com
`,
			wantCount: 0,
		},
		{
			name: "mixed SSH and non-SSH insteadOf rules",
			gitconfig: `[url "git@github.com:"]
	insteadOf = https://github.com/
[url "https://internal.example.com/"]
	insteadOf = https://github.com/internal/
`,
			wantSubstring: []string{"git@github.com:"},
			wantCount:     1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writeGitconfig(t, tt.gitconfig)

			got := hasSSHInsteadOfRules()

			if len(got) != tt.wantCount {
				t.Errorf("hasSSHInsteadOfRules() returned %d rewrites, want %d\ngot: %v", len(got), tt.wantCount, got)
			}

			combined := strings.Join(got, "\n")
			for _, sub := range tt.wantSubstring {
				if !strings.Contains(combined, sub) {
					t.Errorf("hasSSHInsteadOfRules() output missing %q\ngot: %v", sub, got)
				}
			}
		})
	}
}

func TestWarnSSHInsteadOfRulesTo(t *testing.T) {
	tests := []struct {
		name          string
		gitconfig     string
		wantOutput    bool
		wantSubstring []string
	}{
		{
			name: "no SSH insteadOf rules — no output",
			gitconfig: `[core]
	autocrlf = false
`,
			wantOutput: false,
		},
		{
			name: "SSH insteadOf rule — emits warning",
			gitconfig: `[url "git@github.com:"]
	insteadOf = https://github.com/
`,
			wantOutput:    true,
			wantSubstring: []string{"WARNING", "git@github.com:", "https://github.com/", "packnplay --credentials ssh"},
		},
		{
			name: "multiple rules — all listed in warning",
			gitconfig: `[url "git@github.com:"]
	insteadOf = https://github.com/
[url "git@gitlab.com:"]
	insteadOf = https://gitlab.com/
`,
			wantOutput:    true,
			wantSubstring: []string{"git@github.com:", "git@gitlab.com:"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writeGitconfig(t, tt.gitconfig)

			var buf bytes.Buffer
			warnSSHInsteadOfRulesTo(&buf)

			output := buf.String()
			if tt.wantOutput && output == "" {
				t.Error("warnSSHInsteadOfRulesTo() produced no output, expected a warning")
			}
			if !tt.wantOutput && output != "" {
				t.Errorf("warnSSHInsteadOfRulesTo() produced unexpected output: %q", output)
			}
			for _, sub := range tt.wantSubstring {
				if !strings.Contains(output, sub) {
					t.Errorf("warnSSHInsteadOfRulesTo() output missing %q\ngot: %q", sub, output)
				}
			}
		})
	}
}
