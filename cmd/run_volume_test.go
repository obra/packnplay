// ABOUTME: Tests for the -v/--volume CLI flag for bind-mounting volumes into containers.
// ABOUTME: Mirrors run_port_test.go pattern for the -p/--publish flag.

package cmd

import (
	"testing"

	"github.com/obra/packnplay/pkg/config"
	"github.com/obra/packnplay/pkg/runner"
)

func TestRunVolumeFlag(t *testing.T) {
	runVolumes = []string{}

	err := runCmd.ParseFlags([]string{"-v", "/tmp/test:/test"})
	if err != nil {
		t.Fatalf("ParseFlags() error = %v", err)
	}

	if len(runVolumes) != 1 {
		t.Errorf("Expected 1 volume mount, got %d", len(runVolumes))
	}

	if runVolumes[0] != "/tmp/test:/test" {
		t.Errorf("Expected volume mount '/tmp/test:/test', got '%s'", runVolumes[0])
	}
}

func TestRunMultipleVolumeFlags(t *testing.T) {
	runVolumes = []string{}

	err := runCmd.ParseFlags([]string{"-v", "/tmp/a:/a", "-v", "/tmp/b:/b", "-v", "/tmp/c:/c"})
	if err != nil {
		t.Fatalf("ParseFlags() error = %v", err)
	}

	if len(runVolumes) != 3 {
		t.Errorf("Expected 3 volume mounts, got %d", len(runVolumes))
	}

	expected := []string{"/tmp/a:/a", "/tmp/b:/b", "/tmp/c:/c"}
	for i, exp := range expected {
		if i >= len(runVolumes) {
			t.Errorf("Missing volume mount at index %d", i)
			continue
		}
		if runVolumes[i] != exp {
			t.Errorf("Expected volume mount '%s' at index %d, got '%s'", exp, i, runVolumes[i])
		}
	}
}

func TestDockerCompatibleVolumeFormats(t *testing.T) {
	tests := []struct {
		name     string
		flags    []string
		expected []string
	}{
		{
			name:     "host path to container path",
			flags:    []string{"-v", "/host:/container"},
			expected: []string{"/host:/container"},
		},
		{
			name:     "read-only mount",
			flags:    []string{"-v", "/host:/container:ro"},
			expected: []string{"/host:/container:ro"},
		},
		{
			name:     "named volume",
			flags:    []string{"-v", "myvolume:/data"},
			expected: []string{"myvolume:/data"},
		},
		{
			name:     "bare absolute path",
			flags:    []string{"-v", "/data"},
			expected: []string{"/data"},
		},
		{
			name:     "read-write explicit",
			flags:    []string{"-v", "/host:/container:rw"},
			expected: []string{"/host:/container:rw"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runVolumes = []string{}

			err := runCmd.ParseFlags(tt.flags)
			if err != nil {
				t.Fatalf("ParseFlags() error = %v", err)
			}

			if len(runVolumes) != len(tt.expected) {
				t.Errorf("Expected %d volume mounts, got %d", len(tt.expected), len(runVolumes))
			}

			for i, exp := range tt.expected {
				if i >= len(runVolumes) {
					t.Errorf("Missing volume mount at index %d", i)
					continue
				}
				if runVolumes[i] != exp {
					t.Errorf("Expected volume mount '%s' at index %d, got '%s'", exp, i, runVolumes[i])
				}
			}
		})
	}
}

func TestRunConfigIncludesVolumes(t *testing.T) {
	runVolumes = []string{}

	err := runCmd.ParseFlags([]string{"-v", "/tmp/a:/a", "-v", "/tmp/b:/b", "echo", "hello"})
	if err != nil {
		t.Fatalf("ParseFlags() error = %v", err)
	}

	cfg := &config.Config{
		ContainerRuntime: "docker",
		DefaultImage:     "ubuntu:22.04",
	}

	runConfig := &runner.RunConfig{
		Runtime:      cfg.ContainerRuntime,
		DefaultImage: cfg.DefaultImage,
		Command:      []string{"echo", "hello"},
		Volumes:      runVolumes,
	}

	if len(runConfig.Volumes) != 2 {
		t.Errorf("Expected 2 volume mounts in RunConfig, got %d", len(runConfig.Volumes))
	}

	expected := []string{"/tmp/a:/a", "/tmp/b:/b"}
	for i, exp := range expected {
		if i >= len(runConfig.Volumes) {
			t.Errorf("Missing volume mount at index %d", i)
			continue
		}
		if runConfig.Volumes[i] != exp {
			t.Errorf("Expected volume mount '%s' at index %d in RunConfig, got '%s'", exp, i, runConfig.Volumes[i])
		}
	}
}
