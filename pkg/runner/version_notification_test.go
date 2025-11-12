package runner

import (
	"testing"
	"time"
)

func TestVersionNotificationLogic(t *testing.T) {
	// Test when we should notify about new versions

	tests := []struct {
		name           string
		currentDigest  string
		remoteDigest   string
		lastNotified   time.Time
		shouldNotify   bool
		expectedReason string
	}{
		{
			name:           "new version should notify",
			currentDigest:  "sha256:old123",
			remoteDigest:   "sha256:new456",
			lastNotified:   time.Time{}, // never notified
			shouldNotify:   true,
			expectedReason: "new version available",
		},
		{
			name:           "same version should not notify",
			currentDigest:  "sha256:same123",
			remoteDigest:   "sha256:same123",
			lastNotified:   time.Time{},
			shouldNotify:   false,
			expectedReason: "same version",
		},
		{
			name:           "recently notified should not notify again",
			currentDigest:  "sha256:old123",
			remoteDigest:   "sha256:new456",
			lastNotified:   time.Now().Add(-1 * time.Hour), // notified 1 hour ago
			shouldNotify:   false,
			expectedReason: "recently notified",
		},
		{
			name:           "old notification should notify again",
			currentDigest:  "sha256:old123",
			remoteDigest:   "sha256:new456",
			lastNotified:   time.Now().Add(-25 * time.Hour), // notified 25 hours ago
			shouldNotify:   true,
			expectedReason: "new version available",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldNotifyAboutVersion(tt.currentDigest, tt.remoteDigest, tt.lastNotified, 24*time.Hour)

			if result.shouldNotify != tt.shouldNotify {
				t.Errorf("shouldNotify = %v, want %v", result.shouldNotify, tt.shouldNotify)
			}

			if result.reason != tt.expectedReason {
				t.Errorf("reason = %v, want %v", result.reason, tt.expectedReason)
			}
		})
	}
}

func TestImageVersionInfo(t *testing.T) {
	// Test getting version information from images

	info := &ImageVersionInfo{
		Digest:  "sha256:abc123def456",
		Created: time.Now().Add(-2 * time.Hour),
		Size:    "1.2GB",
		Tags:    []string{"latest", "v1.0"},
	}

	if info.Digest != "sha256:abc123def456" {
		t.Errorf("Digest = %v, want sha256:abc123def456", info.Digest)
	}

	if info.AgeString() == "" {
		t.Error("AgeString() should not be empty")
	}

	if info.ShortDigest() != "abc123de" {
		t.Errorf("ShortDigest() = %v, want abc123de", info.ShortDigest())
	}
}

func TestVersionTrackingStorage(t *testing.T) {
	// Test storing and retrieving version notification history

	tracker := NewVersionTracker()

	// Should start empty
	if tracker.HasNotified("test-image:latest", "sha256:abc123") {
		t.Error("New tracker should not have notifications")
	}

	// Mark as notified
	tracker.MarkNotified("test-image:latest", "sha256:abc123")

	// Should now show as notified
	if !tracker.HasNotified("test-image:latest", "sha256:abc123") {
		t.Error("Should show as notified after MarkNotified")
	}

	// Different digest should not show as notified
	if tracker.HasNotified("test-image:latest", "sha256:different") {
		t.Error("Different digest should not show as notified")
	}
}

// Types are implemented in runner.go
