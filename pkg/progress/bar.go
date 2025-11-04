package progress

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// ProgressBar handles terminal progress bar rendering
type ProgressBar struct {
	writer     io.Writer
	width      int
	lastLength int
	startTime  time.Time
	visible    bool
}

// NewProgressBar creates a new progress bar
func NewProgressBar(writer io.Writer, width int) *ProgressBar {
	if writer == nil {
		writer = os.Stderr
	}
	return &ProgressBar{
		writer:    writer,
		width:     width,
		startTime: time.Now(),
	}
}

// Update renders the progress bar with current progress
func (pb *ProgressBar) Update(percentage float64, statusText string) {
	// Clear previous line if this isn't the first update
	if pb.visible {
		pb.clearLine()
	}

	// Render new progress bar
	rendered := pb.renderBar(percentage, statusText)
	fmt.Fprint(pb.writer, rendered)
	pb.lastLength = len(rendered)
	pb.visible = true
}

// Complete clears the progress bar and shows completion message
func (pb *ProgressBar) Complete(statusText string) {
	if pb.visible {
		pb.clearLine()
	}

	// Show completion message
	duration := time.Since(pb.startTime)
	completionMsg := fmt.Sprintf("✅ %s (%v)\n", statusText, duration.Round(time.Millisecond))
	fmt.Fprint(pb.writer, completionMsg)
	pb.visible = false
}

// Error clears the progress bar and shows error message
func (pb *ProgressBar) Error(err error) {
	if pb.visible {
		pb.clearLine()
	}

	// Show error message
	errorMsg := fmt.Sprintf("❌ Error: %v\n", err)
	fmt.Fprint(pb.writer, errorMsg)
	pb.visible = false
}

// renderBar creates the visual progress bar string
func (pb *ProgressBar) renderBar(percentage float64, statusText string) string {
	// Ensure percentage is in valid range
	if percentage > 1.0 {
		percentage = 1.0
	}
	if percentage < 0.0 {
		percentage = 0.0
	}

	// Calculate bar dimensions
	barWidth := 20
	filledWidth := int(percentage * float64(barWidth))

	// Ensure filledWidth is in valid range
	if filledWidth > barWidth {
		filledWidth = barWidth
	}
	if filledWidth < 0 {
		filledWidth = 0
	}

	// Create progress bar visual
	filled := strings.Repeat("█", filledWidth)
	empty := strings.Repeat("░", barWidth-filledWidth)

	// Format percentage
	percentText := fmt.Sprintf("%3.0f%%", percentage*100)

	// Style components using lipgloss
	barStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("39")) // Blue
	percentStyle := lipgloss.NewStyle().Bold(true)
	statusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240")) // Gray

	// Build the complete progress line
	progressBar := barStyle.Render(filled + empty)
	percentDisplay := percentStyle.Render(percentText)
	statusDisplay := statusStyle.Render(statusText)

	// Combine components
	result := fmt.Sprintf("\r%s %s %s", progressBar, percentDisplay, statusDisplay)

	// Truncate if too long for terminal
	if len(result) > pb.width {
		maxStatusLen := pb.width - barWidth - 10 // Reserve space for bar and percentage
		if maxStatusLen > 0 && len(statusText) > maxStatusLen {
			truncatedStatus := statusText[:maxStatusLen-3] + "..."
			statusDisplay = statusStyle.Render(truncatedStatus)
			result = fmt.Sprintf("\r%s %s %s", progressBar, percentDisplay, statusDisplay)
		}
	}

	return result
}

// clearLine clears the current line in the terminal
func (pb *ProgressBar) clearLine() {
	// Move cursor to beginning of line and clear it
	fmt.Fprint(pb.writer, "\r"+strings.Repeat(" ", pb.lastLength)+"\r")
}

// IsTerminal checks if the output is a terminal (supports progress bars)
func (pb *ProgressBar) IsTerminal() bool {
	if f, ok := pb.writer.(*os.File); ok {
		return isatty(f.Fd())
	}
	return false
}

// isatty checks if a file descriptor is a terminal
func isatty(fd uintptr) bool {
	// Simple check - in real implementation we might use a proper isatty library
	// For now, assume stderr is always a terminal when it's a file
	return fd == os.Stderr.Fd()
}

// Hide temporarily hides the progress bar (useful for showing other output)
func (pb *ProgressBar) Hide() {
	if pb.visible {
		pb.clearLine()
		pb.visible = false
	}
}

// Show restores the progress bar after hiding
func (pb *ProgressBar) Show(percentage float64, statusText string) {
	pb.Update(percentage, statusText)
}

// SetWidth updates the terminal width for the progress bar
func (pb *ProgressBar) SetWidth(width int) {
	pb.width = width
}