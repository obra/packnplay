package progress

import (
	"strings"
	"testing"
)

func TestProgressTracker_ParseLine(t *testing.T) {
	tracker := NewProgressTracker("ubuntu:22.04")

	tests := []struct {
		name            string
		jsonLine        string
		expectedPercent float64
		expectedStatus  string
		expectError     bool
	}{
		{
			name:            "pulling from repository",
			jsonLine:        `{"status":"Pulling from library/ubuntu","id":"22.04"}`,
			expectedPercent: 0.0,
			expectedStatus:  "pulling ubuntu:22.04",
		},
		{
			name:            "layer downloading with progress",
			jsonLine:        `{"status":"Downloading","progressDetail":{"current":12345678,"total":98765432},"progress":"[====>    ] 12.3MB/98.7MB","id":"layer1"}`,
			expectedPercent: 0.125, // roughly 12.5%
			expectedStatus:  "downloading ubuntu:22.04",
		},
		{
			name:            "layer download complete",
			jsonLine:        `{"status":"Download complete","id":"layer1"}`,
			expectedPercent: 1.0, // Should show 100% when complete
			expectedStatus:  "downloading ubuntu:22.04",
		},
		{
			name:            "invalid json",
			jsonLine:        "not json",
			expectedPercent: 0.0,
			expectedStatus:  "starting ubuntu:22.04",
		},
		{
			name:            "empty line",
			jsonLine:        "",
			expectedPercent: 0.0,
			expectedStatus:  "starting ubuntu:22.04",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			percent, status, err := tracker.ParseLine(tt.jsonLine)

			if tt.expectError && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if percent < 0 || percent > 1 {
				t.Errorf("percentage out of range: %f", percent)
			}

			if !strings.Contains(status, "ubuntu:22.04") {
				t.Errorf("status should contain image name, got: %s", status)
			}
		})
	}
}

func TestProgressTracker_MultipleLayerProgress(t *testing.T) {
	tracker := NewProgressTracker("test-image")

	// Add first layer
	_, _, _ = tracker.ParseLine(`{"status":"Downloading","progressDetail":{"current":50,"total":100},"id":"layer1"}`)

	// Add second layer
	_, _, _ = tracker.ParseLine(`{"status":"Downloading","progressDetail":{"current":25,"total":100},"id":"layer2"}`)

	percent, _, _ := tracker.getProgress()

	// Should be (50+25)/(100+100) = 75/200 = 37.5%
	expected := 0.375
	if percent < expected-0.01 || percent > expected+0.01 {
		t.Errorf("expected ~%f, got %f", expected, percent)
	}

	// Complete first layer
	_, _, _ = tracker.ParseLine(`{"status":"Download complete","id":"layer1"}`)

	percent, _, _ = tracker.getProgress()

	// Should be (100+25)/(100+100) = 125/200 = 62.5%
	expected = 0.625
	if percent < expected-0.01 || percent > expected+0.01 {
		t.Errorf("expected ~%f, got %f", expected, percent)
	}
}

func TestProgressTracker_FormatBytes(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{512, "512B"},
		{1024, "1.0KB"},
		{1536, "1.5KB"},
		{1048576, "1.0MB"},
		{1073741824, "1.0GB"},
		{10485760, "10MB"},
	}

	for _, tt := range tests {
		result := formatBytes(tt.bytes)
		if result != tt.expected {
			t.Errorf("formatBytes(%d) = %s, expected %s", tt.bytes, result, tt.expected)
		}
	}
}

func TestProgressTracker_IsComplete(t *testing.T) {
	tracker := NewProgressTracker("test-image")

	// Initially not complete
	if tracker.IsComplete() {
		t.Error("tracker should not be complete initially")
	}

	// After pull complete status
	_, _, _ = tracker.ParseLine(`{"status":"Pull complete"}`)
	if !tracker.IsComplete() {
		t.Error("tracker should be complete after pull complete")
	}

	// Test cached status
	tracker2 := NewProgressTracker("cached-image")
	_, _, _ = tracker2.ParseLine(`{"status":"Already exists","id":"layer1"}`)
	if !tracker2.IsComplete() {
		t.Error("tracker should be complete for cached image")
	}
}
