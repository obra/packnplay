package progress

import (
	"encoding/json"
	"fmt"
	"strings"
)

// LayerProgress tracks progress for a single Docker layer
type LayerProgress struct {
	ID       string `json:"id"`
	Status   string `json:"status"`
	Current  int64  `json:"current"`
	Total    int64  `json:"total"`
	Complete bool   `json:"complete"`
}

// ProgressTracker aggregates progress across all layers
type ProgressTracker struct {
	layers      map[string]*LayerProgress
	totalBytes  int64
	currentBytes int64
	status      string
	imageName   string
}

// NewProgressTracker creates a new progress tracker for an image
func NewProgressTracker(imageName string) *ProgressTracker {
	return &ProgressTracker{
		layers:    make(map[string]*LayerProgress),
		imageName: imageName,
		status:    "starting",
	}
}

// DockerProgressMessage represents Docker's JSON progress output
type DockerProgressMessage struct {
	Status         string `json:"status"`
	ID             string `json:"id,omitempty"`
	Progress       string `json:"progress,omitempty"`
	ProgressDetail struct {
		Current int64 `json:"current"`
		Total   int64 `json:"total"`
	} `json:"progressDetail,omitempty"`
	Stream string `json:"stream,omitempty"`
}

// ParseLine processes a single line of Docker JSON or plain text output
func (t *ProgressTracker) ParseLine(line string) (percentage float64, statusText string, err error) {
	line = strings.TrimSpace(line)
	if line == "" {
		return t.getProgress()
	}

	// Try to parse as JSON first (for pull operations)
	var msg DockerProgressMessage
	if err := json.Unmarshal([]byte(line), &msg); err == nil {
		// Valid JSON - handle as before
		t.updateStatus(msg.Status)
		if msg.ID != "" {
			t.updateLayer(msg)
		}
		return t.getProgress()
	}

	// Not JSON - handle as plain text (for build operations)
	t.parsePlainTextLine(line)
	return t.getProgress()
}

// parsePlainTextLine handles Docker build plain text output
func (t *ProgressTracker) parsePlainTextLine(line string) {
	line = strings.ToLower(line)

	// Update status based on build output patterns
	if strings.Contains(line, "building") || strings.Contains(line, "step") {
		t.status = "building"
	} else if strings.Contains(line, "sending build context") {
		t.status = "preparing"
	} else if strings.Contains(line, "successfully built") {
		t.status = "complete"
	} else if strings.Contains(line, "error") || strings.Contains(line, "failed") {
		t.status = "error"
	}

	// For build operations, we can't track precise progress
	// but we can provide status updates
}

// updateStatus updates the overall operation status
func (t *ProgressTracker) updateStatus(status string) {
	switch {
	case strings.Contains(status, "Pulling from"):
		t.status = "pulling"
	case strings.Contains(status, "Downloading"):
		t.status = "downloading"
	case strings.Contains(status, "Extracting"):
		t.status = "extracting"
	case strings.Contains(status, "Pull complete"):
		t.status = "complete"
	case strings.Contains(status, "Already exists"):
		t.status = "cached"
	case strings.Contains(status, "Building"):
		t.status = "building"
	case strings.Contains(status, "Successfully built"):
		t.status = "complete"
	}
}

// updateLayer updates progress for a specific layer
func (t *ProgressTracker) updateLayer(msg DockerProgressMessage) {
	layerID := msg.ID

	// Get or create layer progress
	layer, exists := t.layers[layerID]
	if !exists {
		layer = &LayerProgress{ID: layerID}
		t.layers[layerID] = layer
	}

	// Update layer status
	layer.Status = msg.Status

	// Update progress if available
	if msg.ProgressDetail.Total > 0 {
		// Remove old contribution from totals
		if layer.Total > 0 {
			t.totalBytes -= layer.Total
			t.currentBytes -= layer.Current
		}

		// Update layer progress
		layer.Current = msg.ProgressDetail.Current
		layer.Total = msg.ProgressDetail.Total

		// Add new contribution to totals
		t.totalBytes += layer.Total
		t.currentBytes += layer.Current
	}

	// Mark as complete if status indicates completion
	if strings.Contains(msg.Status, "complete") ||
	   strings.Contains(msg.Status, "Already exists") {
		layer.Complete = true
		// Ensure completed layers are counted as 100%
		if layer.Total > 0 && layer.Current < layer.Total {
			t.currentBytes += (layer.Total - layer.Current)
			layer.Current = layer.Total
		}
	}
}

// getProgress calculates overall progress percentage and status text
func (t *ProgressTracker) getProgress() (percentage float64, statusText string, err error) {
	if t.totalBytes == 0 {
		// No byte-level progress data (e.g., for build operations)
		switch t.status {
		case "complete":
			return 1.0, fmt.Sprintf("%s %s", t.status, t.imageName), nil
		case "cached":
			return 1.0, fmt.Sprintf("using cached %s", t.imageName), nil
		case "building":
			// For builds, show indeterminate progress
			return 0.5, fmt.Sprintf("%s %s", t.status, t.imageName), nil
		case "preparing":
			return 0.1, fmt.Sprintf("%s %s", t.status, t.imageName), nil
		case "error":
			return 0.0, fmt.Sprintf("%s %s", t.status, t.imageName), nil
		default:
			return 0.0, fmt.Sprintf("%s %s", t.status, t.imageName), nil
		}
	}

	// Calculate percentage from byte progress
	percentage = float64(t.currentBytes) / float64(t.totalBytes)
	if percentage > 1.0 {
		percentage = 1.0
	}

	// Format status text with details
	statusText = fmt.Sprintf("%s %s (%s/%s)",
		t.status,
		t.imageName,
		formatBytes(t.currentBytes),
		formatBytes(t.totalBytes))

	return percentage, statusText, nil
}

// formatBytes formats byte counts in human-readable format
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%dB", bytes)
	}

	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	format := "%.1f%s"
	if bytes/div >= 10 {
		format = "%.0f%s"
	}

	return fmt.Sprintf(format, float64(bytes)/float64(div), "KMGTPE"[exp:exp+1]+"B")
}

// GetLayerCount returns the number of layers being tracked
func (t *ProgressTracker) GetLayerCount() int {
	return len(t.layers)
}

// IsComplete returns true if the operation is complete
func (t *ProgressTracker) IsComplete() bool {
	return t.status == "complete" || t.status == "cached"
}