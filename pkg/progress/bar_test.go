package progress

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestProgressBar_Update(t *testing.T) {
	var buf bytes.Buffer
	bar := NewProgressBar(&buf, 80)

	// Test basic progress update
	bar.Update(0.5, "downloading test-image")

	output := buf.String()
	if !strings.Contains(output, "50%") {
		t.Errorf("expected 50%% in output, got: %s", output)
	}
	if !strings.Contains(output, "test-image") {
		t.Errorf("expected image name in output, got: %s", output)
	}
}

func TestProgressBar_Complete(t *testing.T) {
	var buf bytes.Buffer
	bar := NewProgressBar(&buf, 80)
	bar.startTime = time.Now().Add(-time.Second) // Fake 1 second duration

	bar.Complete("pull complete test-image")

	output := buf.String()
	if !strings.Contains(output, "✅") {
		t.Errorf("expected success emoji in completion, got: %s", output)
	}
	if !strings.Contains(output, "test-image") {
		t.Errorf("expected image name in completion, got: %s", output)
	}
}

func TestProgressBar_Error(t *testing.T) {
	var buf bytes.Buffer
	bar := NewProgressBar(&buf, 80)

	bar.Error(fmt.Errorf("network timeout"))

	output := buf.String()
	if !strings.Contains(output, "❌") {
		t.Errorf("expected error emoji, got: %s", output)
	}
	if !strings.Contains(output, "network timeout") {
		t.Errorf("expected error message, got: %s", output)
	}
}

func TestProgressBar_renderBar(t *testing.T) {
	bar := NewProgressBar(nil, 80)

	tests := []struct {
		percentage  float64
		statusText  string
		expectFill  bool
		expectEmpty bool
	}{
		{0.0, "starting", false, true},
		{0.5, "downloading", true, true},
		{1.0, "complete", true, false},
		{1.5, "over 100%", true, false}, // Should cap at 100%
		{-0.1, "negative", false, true}, // Should floor at 0%
	}

	for _, tt := range tests {
		result := bar.renderBar(tt.percentage, tt.statusText)

		if tt.expectFill && !strings.Contains(result, "█") {
			t.Errorf("expected filled chars for %f%%, got: %s", tt.percentage*100, result)
		}
		if tt.expectEmpty && !strings.Contains(result, "░") {
			t.Errorf("expected empty chars for %f%%, got: %s", tt.percentage*100, result)
		}
		if !strings.Contains(result, tt.statusText) {
			t.Errorf("expected status text %q in result: %s", tt.statusText, result)
		}
	}
}

func TestProgressBar_formatBytes(t *testing.T) {
	// Test various byte formatting scenarios
	tests := []struct {
		bytes    int64
		contains string // What the result should contain
	}{
		{0, "0B"},
		{512, "512B"},
		{1024, "1.0KB"},
		{1536, "1.5KB"},
		{1048576, "1.0MB"},
		{10485760, "10MB"}, // Should show whole number for >= 10
	}

	for _, tt := range tests {
		result := formatBytes(tt.bytes)
		if !strings.Contains(result, tt.contains) {
			t.Errorf("formatBytes(%d) = %s, expected to contain %s", tt.bytes, result, tt.contains)
		}
	}
}

func TestProgressBar_SetWidth(t *testing.T) {
	bar := NewProgressBar(nil, 80)
	bar.SetWidth(120)

	if bar.width != 120 {
		t.Errorf("expected width 120, got %d", bar.width)
	}
}