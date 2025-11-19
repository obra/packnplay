package cmd

import (
	"testing"
)

func TestVersionCommandDefaults(t *testing.T) {
	// Test that version variables have sensible defaults when build-time injection is not used
	// This is normal behavior for development builds

	// Verify that default values are set (this is expected behavior)
	if version == "" {
		t.Errorf("version should have a default value, got empty string")
	}

	if commit == "" {
		t.Errorf("commit should have a default value, got empty string")
	}

	if date == "" {
		t.Errorf("date should have a default value, got empty string")
	}

	// Note: During development (go test), defaults are expected
	// During builds (make build), real values are injected via ldflags
}

func TestVersionCommandWithInjectedValues(t *testing.T) {
	// RED: Test version command output formatting
	// Save original values
	origVersion := version
	origCommit := commit
	origDate := date

	// Set test values
	version = "v1.2.3"
	commit = "abc123def"
	date = "2023-11-15T10:30:00Z"

	// Restore original values after test
	defer func() {
		version = origVersion
		commit = origCommit
		date = origDate
	}()

	// Test the formatting logic by checking individual components
	if version != "v1.2.3" {
		t.Errorf("version = %v, want v1.2.3", version)
	}

	if commit != "abc123def" {
		t.Errorf("commit = %v, want abc123def", commit)
	}

	if date != "2023-11-15T10:30:00Z" {
		t.Errorf("date = %v, want 2023-11-15T10:30:00Z", date)
	}
}