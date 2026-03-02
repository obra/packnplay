// ABOUTME: Tests for CLI volume handling: normalization and reconnect warnings.
// ABOUTME: Verifies shorthand expansion, relative path resolution, and ignored-flag warnings.

package runner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNormalizeVolume(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "bare path becomes same-path bind mount",
			input:    "/Users/clkao/git/noteplan-plugin",
			expected: "/Users/clkao/git/noteplan-plugin:/Users/clkao/git/noteplan-plugin",
		},
		{
			name:     "leading colon creates anonymous volume",
			input:    ":/data",
			expected: "/data",
		},
		{
			name:     "host:container passed through",
			input:    "/host/path:/container/path",
			expected: "/host/path:/container/path",
		},
		{
			name:     "host:container:options passed through",
			input:    "/host/path:/container/path:ro",
			expected: "/host/path:/container/path:ro",
		},
		{
			name:     "named volume passed through",
			input:    "myvolume:/data",
			expected: "myvolume:/data",
		},
		{
			name:     "named volume with options passed through",
			input:    "myvolume:/data:ro",
			expected: "myvolume:/data:ro",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeVolume(tt.input)
			if got != tt.expected {
				t.Errorf("normalizeVolume(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestIgnoredCreationFlags(t *testing.T) {
	tests := []struct {
		name         string
		config       *RunConfig
		wantWarnings []string
		wantNone     bool
	}{
		{
			name:     "no flags set",
			config:   &RunConfig{},
			wantNone: true,
		},
		{
			name:         "volumes ignored",
			config:       &RunConfig{Volumes: []string{"/tmp:/tmp"}},
			wantWarnings: []string{"-v/--volume"},
		},
		{
			name:         "ports ignored",
			config:       &RunConfig{PublishPorts: []string{"8080:80"}},
			wantWarnings: []string{"-p/--publish"},
		},
		{
			name:         "both ignored",
			config:       &RunConfig{Volumes: []string{"/a:/a"}, PublishPorts: []string{"3000:3000"}},
			wantWarnings: []string{"-v/--volume", "-p/--publish"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := ignoredCreationFlags(tt.config)
			if tt.wantNone {
				if msg != "" {
					t.Errorf("expected no warning, got %q", msg)
				}
				return
			}
			if msg == "" {
				t.Fatal("expected warning, got empty string")
			}
			for _, want := range tt.wantWarnings {
				if !strings.Contains(msg, want) {
					t.Errorf("warning %q missing expected substring %q", msg, want)
				}
			}
		})
	}
}

func TestNormalizeVolumeRelativePath(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "relative path resolved to absolute",
			input:    "data",
			expected: filepath.Join(cwd, "data") + ":" + filepath.Join(cwd, "data"),
		},
		{
			name:     "dot-relative path resolved to absolute",
			input:    "./mydir",
			expected: filepath.Join(cwd, "mydir") + ":" + filepath.Join(cwd, "mydir"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeVolume(tt.input)
			if got != tt.expected {
				t.Errorf("normalizeVolume(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}
