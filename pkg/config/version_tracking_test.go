package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestVersionTrackingPersistence(t *testing.T) {
	// Test saving and loading version tracking data

	tmpDir := t.TempDir()
	trackingFile := filepath.Join(tmpDir, "version-tracking.json")

	// Create tracking data
	tracking := &VersionTrackingData{
		LastCheck: time.Now(),
		Notifications: map[string]VersionNotification{
			"test-image:latest": {
				Digest:     "sha256:abc123",
				NotifiedAt: time.Now().Add(-1 * time.Hour),
				ImageName:  "test-image:latest",
			},
		},
	}

	// Save to file
	err := SaveVersionTracking(tracking, trackingFile)
	if err != nil {
		t.Errorf("SaveVersionTracking() error = %v", err)
	}

	// Load from file
	loaded, err := LoadVersionTracking(trackingFile)
	if err != nil {
		t.Errorf("LoadVersionTracking() error = %v", err)
	}

	// Verify data matches
	if len(loaded.Notifications) != 1 {
		t.Errorf("Loaded notifications count = %v, want 1", len(loaded.Notifications))
	}

	if notification, exists := loaded.Notifications["test-image:latest"]; !exists {
		t.Error("Should have loaded test-image notification")
	} else {
		if notification.Digest != "sha256:abc123" {
			t.Errorf("Loaded digest = %v, want sha256:abc123", notification.Digest)
		}
	}
}

func TestVersionTrackingFileLocation(t *testing.T) {
	// Test that version tracking file is stored in correct location

	path := GetVersionTrackingPath()

	// Should be in XDG config directory
	if !containsSubstring(path, "packnplay") {
		t.Errorf("Tracking path should contain 'packnplay': %s", path)
	}

	if !containsSubstring(path, "version-tracking.json") {
		t.Errorf("Tracking path should end with version-tracking.json: %s", path)
	}

	// Directory should be creatable
	dir := filepath.Dir(path)
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		t.Errorf("Should be able to create tracking directory: %v", err)
	}
}

func TestShouldCheckForUpdates(t *testing.T) {
	// Test when we should check for updates based on config and last check time

	tests := []struct {
		name        string
		config      DefaultContainerConfig
		lastCheck   time.Time
		shouldCheck bool
	}{
		{
			name: "checking disabled should not check",
			config: DefaultContainerConfig{
				CheckForUpdates: false,
			},
			shouldCheck: false,
		},
		{
			name: "first time should check",
			config: DefaultContainerConfig{
				CheckForUpdates:     true,
				CheckFrequencyHours: 24,
			},
			lastCheck:   time.Time{}, // never checked
			shouldCheck: true,
		},
		{
			name: "recent check should not check again",
			config: DefaultContainerConfig{
				CheckForUpdates:     true,
				CheckFrequencyHours: 24,
			},
			lastCheck:   time.Now().Add(-1 * time.Hour), // checked 1 hour ago
			shouldCheck: false,
		},
		{
			name: "old check should check again",
			config: DefaultContainerConfig{
				CheckForUpdates:     true,
				CheckFrequencyHours: 24,
			},
			lastCheck:   time.Now().Add(-25 * time.Hour), // checked 25 hours ago
			shouldCheck: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldCheckForUpdates(tt.config, tt.lastCheck)
			if result != tt.shouldCheck {
				t.Errorf("shouldCheckForUpdates() = %v, want %v", result, tt.shouldCheck)
			}
		})
	}
}

// Helper function
func containsSubstring(s, substr string) bool {
	return strings.Contains(s, substr)
}
