package runner

import (
	"strings"
	"testing"
	"time"

	"github.com/obra/packnplay/pkg/docker"
)

func TestGetRemoteImageInfo(t *testing.T) {
	// Test getting version info from remote registry

	// Skip if no network/docker available
	dockerClient, err := NewTestDockerClient()
	if err != nil {
		t.Skip("Docker not available for registry testing")
	}

	imageName := "ubuntu:22.04" // Use a known stable image for testing

	info, err := getRemoteImageInfo(dockerClient, imageName)
	if err != nil {
		t.Errorf("getRemoteImageInfo() error = %v", err)
	}

	if info.Digest == "" {
		t.Error("Remote image info should have digest")
	}

	if info.ShortDigest() == "" {
		t.Error("ShortDigest should not be empty")
	}
}

func TestCheckForNewVersion(t *testing.T) {
	// Test complete version checking workflow

	// Mock version tracker
	tracker := NewVersionTracker()

	// Test scenario: new version available, never notified
	localInfo := &ImageVersionInfo{
		Digest: "sha256:old123",
	}

	remoteInfo := &ImageVersionInfo{
		Digest:  "sha256:new456",
		Created: timeNow().Add(-2 * time.Hour), // 2 hours old
		Size:    "1.2GB",
	}

	result := checkForNewVersion("test:latest", localInfo, remoteInfo, tracker)

	if !result.shouldNotify {
		t.Error("Should notify about new version")
	}

	if result.localInfo.ShortDigest() != "old123" {
		t.Errorf("Local digest = %v, want old123", result.localInfo.ShortDigest())
	}

	if result.remoteInfo.ShortDigest() != "new456" {
		t.Errorf("Remote digest = %v, want new456", result.remoteInfo.ShortDigest())
	}
}

func TestNotificationMessage(t *testing.T) {
	// Test the notification message formatting

	localInfo := &ImageVersionInfo{
		Digest:  "sha256:abc123def",
		Created: timeNow().Add(-48 * time.Hour), // 2 days old
	}

	remoteInfo := &ImageVersionInfo{
		Digest:  "sha256:xyz789ghi",
		Created: timeNow().Add(-1 * time.Hour), // 1 hour old
		Size:    "1.5GB",
	}

	message := formatVersionNotification("my-org/image:latest", localInfo, remoteInfo)

	// Should contain image name
	if !containsString(message, "my-org/image:latest") {
		t.Errorf("Message should contain image name: %s", message)
	}

	// Should contain version info
	if !containsString(message, "abc123de") { // short digest
		t.Errorf("Message should contain local short digest: %s", message)
	}

	if !containsString(message, "xyz789gh") { // short digest
		t.Errorf("Message should contain remote short digest: %s", message)
	}

	// Should contain refresh command
	if !containsString(message, "packnplay refresh-container") {
		t.Errorf("Message should contain refresh command: %s", message)
	}
}

// Helper functions for testing
func NewTestDockerClient() (*docker.Client, error) {
	return docker.NewClient(false)
}

func timeNow() time.Time {
	return time.Now()
}

func timeHour() time.Duration {
	return time.Hour
}

func containsString(s, substr string) bool {
	return strings.Contains(s, substr)
}

// Types are implemented in runner.go
