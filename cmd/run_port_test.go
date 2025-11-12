package cmd

import (
	"testing"

	"github.com/obra/packnplay/pkg/config"
	"github.com/obra/packnplay/pkg/runner"
)

func TestRunPortFlag(t *testing.T) {
	// Clear the global variable before test
	runPublishPorts = []string{}

	// Test that -p flag accepts port mapping
	err := runCmd.ParseFlags([]string{"-p", "8080:3000"})
	if err != nil {
		t.Fatalf("ParseFlags() error = %v", err)
	}

	// Test that we can access the parsed port value
	if len(runPublishPorts) != 1 {
		t.Errorf("Expected 1 port mapping, got %d", len(runPublishPorts))
	}

	if runPublishPorts[0] != "8080:3000" {
		t.Errorf("Expected port mapping '8080:3000', got '%s'", runPublishPorts[0])
	}
}

func TestRunMultiplePortFlags(t *testing.T) {
	// Clear the global variable before test
	runPublishPorts = []string{}

	// Test that multiple -p flags work
	err := runCmd.ParseFlags([]string{"-p", "8080:3000", "-p", "9000:9001", "-p", "5432:5432"})
	if err != nil {
		t.Fatalf("ParseFlags() error = %v", err)
	}

	// Test that we have all three port mappings
	if len(runPublishPorts) != 3 {
		t.Errorf("Expected 3 port mappings, got %d", len(runPublishPorts))
	}

	expectedPorts := []string{"8080:3000", "9000:9001", "5432:5432"}
	for i, expected := range expectedPorts {
		if i >= len(runPublishPorts) {
			t.Errorf("Missing port mapping at index %d", i)
			continue
		}
		if runPublishPorts[i] != expected {
			t.Errorf("Expected port mapping '%s' at index %d, got '%s'", expected, i, runPublishPorts[i])
		}
	}
}

func TestDockerCompatiblePortFormats(t *testing.T) {
	tests := []struct {
		name     string
		flags    []string
		expected []string
	}{
		{
			name:     "basic port mapping",
			flags:    []string{"-p", "8080:3000"},
			expected: []string{"8080:3000"},
		},
		{
			name:     "with host IP",
			flags:    []string{"-p", "127.0.0.1:8080:3000"},
			expected: []string{"127.0.0.1:8080:3000"},
		},
		{
			name:     "with protocol",
			flags:    []string{"-p", "8080:3000/tcp"},
			expected: []string{"8080:3000/tcp"},
		},
		{
			name:     "UDP protocol",
			flags:    []string{"-p", "5353:53/udp"},
			expected: []string{"5353:53/udp"},
		},
		{
			name:     "host IP with protocol",
			flags:    []string{"-p", "127.0.0.1:8080:3000/tcp"},
			expected: []string{"127.0.0.1:8080:3000/tcp"},
		},
		{
			name:     "same port both sides",
			flags:    []string{"-p", "3000:3000"},
			expected: []string{"3000:3000"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear the global variable before each test
			runPublishPorts = []string{}

			err := runCmd.ParseFlags(tt.flags)
			if err != nil {
				t.Fatalf("ParseFlags() error = %v", err)
			}

			if len(runPublishPorts) != len(tt.expected) {
				t.Errorf("Expected %d port mappings, got %d", len(tt.expected), len(runPublishPorts))
			}

			for i, expected := range tt.expected {
				if i >= len(runPublishPorts) {
					t.Errorf("Missing port mapping at index %d", i)
					continue
				}
				if runPublishPorts[i] != expected {
					t.Errorf("Expected port mapping '%s' at index %d, got '%s'", expected, i, runPublishPorts[i])
				}
			}
		})
	}
}

func TestRunConfigIncludesPortMappings(t *testing.T) {
	// This tests that port mappings from flags are passed to RunConfig
	// This test should fail until we implement the integration

	// We'll need to create a mock runner.Run function that captures the RunConfig
	// For now, let's test the creation of RunConfig structure

	// Clear the global variable before test
	runPublishPorts = []string{}

	// Parse some port flags
	err := runCmd.ParseFlags([]string{"-p", "8080:3000", "-p", "9000:9001", "echo", "hello"})
	if err != nil {
		t.Fatalf("ParseFlags() error = %v", err)
	}

	// Create a basic config like the run command does
	cfg := &config.Config{
		ContainerRuntime: "docker",
		DefaultImage:     "ubuntu:22.04",
	}

	// This is where we'll create the RunConfig - this should include port mappings
	runConfig := &runner.RunConfig{
		Runtime:      cfg.ContainerRuntime,
		DefaultImage: cfg.DefaultImage,
		Command:      []string{"echo", "hello"},
		PublishPorts: runPublishPorts, // This field doesn't exist yet - test should fail
	}

	// Verify the port mappings are included
	if len(runConfig.PublishPorts) != 2 {
		t.Errorf("Expected 2 port mappings in RunConfig, got %d", len(runConfig.PublishPorts))
	}

	expectedPorts := []string{"8080:3000", "9000:9001"}
	for i, expected := range expectedPorts {
		if i >= len(runConfig.PublishPorts) {
			t.Errorf("Missing port mapping at index %d", i)
			continue
		}
		if runConfig.PublishPorts[i] != expected {
			t.Errorf("Expected port mapping '%s' at index %d in RunConfig, got '%s'", expected, i, runConfig.PublishPorts[i])
		}
	}
}
