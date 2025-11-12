package cmd

import (
	"strings"
	"testing"
)

func TestConfigureCommand(t *testing.T) {
	// Test that configure command exists and has proper structure

	if configureCmd == nil {
		t.Error("configureCmd should be defined")
	}

	if configureCmd.Use != "configure" {
		t.Errorf("configure command Use = %v, want configure", configureCmd.Use)
	}

	if configureCmd.Short == "" {
		t.Error("configure command should have Short description")
	}

	// Should mention that it preserves existing settings
	if !containsSubstring(configureCmd.Long, "preserve") {
		t.Error("configure command should mention preserving existing settings")
	}

	// Should mention the logical flow
	if !containsSubstring(configureCmd.Long, "logical flow") {
		t.Error("configure command should mention logical flow")
	}
}

func TestConfigureCommandFlags(t *testing.T) {
	// Test configure command flags (simplified)

	// Should NOT have section flag (simplified)
	flag := configureCmd.Flags().Lookup("section")
	if flag != nil {
		t.Error("configure command should not have --section flag (simplified design)")
	}

	// Should have verbose flag
	flag = configureCmd.Flags().Lookup("verbose")
	if flag == nil {
		t.Error("configure command should have --verbose flag")
	}
}

// Helper function
func containsSubstring(s, substr string) bool {
	return strings.Contains(s, substr)
}
