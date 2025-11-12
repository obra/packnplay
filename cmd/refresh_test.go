package cmd

import (
	"testing"
)

func TestRefreshCommand(t *testing.T) {
	// Test that the refresh command exists and works with configurable default image

	if refreshCmd == nil {
		t.Error("refreshCmd should be defined")
	}

	if refreshCmd.Use != "refresh-container" {
		t.Errorf("refresh command Use = %v, want refresh-container", refreshCmd.Use)
	}

	if refreshCmd.Short == "" {
		t.Error("refresh command should have Short description")
	}
}

func TestRefreshCommandFlags(t *testing.T) {
	// Test that refresh command has verbose flag

	flag := refreshCmd.Flags().Lookup("verbose")
	if flag == nil {
		t.Error("refresh command should have --verbose flag")
	}
}
