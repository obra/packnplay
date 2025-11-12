package runner

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/obra/packnplay/pkg/devcontainer"
)

// TestMetadataIntegration_FirstRun tests that commands run on first execution
func TestMetadataIntegration_FirstRun(t *testing.T) {
	// Create temp directory for test
	tempDir := t.TempDir()

	// Override XDG_DATA_HOME for test
	originalXDG := os.Getenv("XDG_DATA_HOME")
	os.Setenv("XDG_DATA_HOME", tempDir)
	defer os.Setenv("XDG_DATA_HOME", originalXDG)

	// Create mock client and executor
	mockClient := &mockDockerClient{
		execCalls: [][]string{},
	}

	metadata, err := LoadMetadata("test-container-123")
	if err != nil {
		t.Fatalf("LoadMetadata failed: %v", err)
	}

	executor := NewLifecycleExecutor(mockClient, "test-container-123", "testuser", false, metadata)

	// Create onCreate command
	cmdJSON := `"npm install"`
	var cmd devcontainer.LifecycleCommand
	if err := json.Unmarshal([]byte(cmdJSON), &cmd); err != nil {
		t.Fatalf("Failed to create command: %v", err)
	}

	// Execute onCreate command - should run
	err = executor.Execute("onCreate", &cmd)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify command was executed
	if len(mockClient.execCalls) != 1 {
		t.Errorf("Expected 1 exec call, got %d", len(mockClient.execCalls))
	}

	// Save metadata
	if err := SaveMetadata(metadata); err != nil {
		t.Fatalf("SaveMetadata failed: %v", err)
	}
}

// TestMetadataIntegration_SecondRun tests that commands don't run again
func TestMetadataIntegration_SecondRun(t *testing.T) {
	// Create temp directory for test
	tempDir := t.TempDir()

	// Override XDG_DATA_HOME for test
	originalXDG := os.Getenv("XDG_DATA_HOME")
	os.Setenv("XDG_DATA_HOME", tempDir)
	defer os.Setenv("XDG_DATA_HOME", originalXDG)

	// Create onCreate command
	cmdJSON := `"npm install"`
	var cmd devcontainer.LifecycleCommand
	if err := json.Unmarshal([]byte(cmdJSON), &cmd); err != nil {
		t.Fatalf("Failed to create command: %v", err)
	}

	// First run - should execute
	{
		mockClient := &mockDockerClient{
			execCalls: [][]string{},
		}

		metadata, err := LoadMetadata("test-container-456")
		if err != nil {
			t.Fatalf("LoadMetadata failed: %v", err)
		}

		executor := NewLifecycleExecutor(mockClient, "test-container-456", "testuser", false, metadata)

		err = executor.Execute("onCreate", &cmd)
		if err != nil {
			t.Fatalf("Execute failed: %v", err)
		}

		// Should have executed
		if len(mockClient.execCalls) != 1 {
			t.Errorf("First run: Expected 1 exec call, got %d", len(mockClient.execCalls))
		}

		// Save metadata
		if err := SaveMetadata(metadata); err != nil {
			t.Fatalf("SaveMetadata failed: %v", err)
		}
	}

	// Second run - should NOT execute (already ran)
	{
		mockClient := &mockDockerClient{
			execCalls: [][]string{},
		}

		metadata, err := LoadMetadata("test-container-456")
		if err != nil {
			t.Fatalf("LoadMetadata failed: %v", err)
		}

		executor := NewLifecycleExecutor(mockClient, "test-container-456", "testuser", false, metadata)

		err = executor.Execute("onCreate", &cmd)
		if err != nil {
			t.Fatalf("Execute failed: %v", err)
		}

		// Should NOT have executed
		if len(mockClient.execCalls) != 0 {
			t.Errorf("Second run: Expected 0 exec calls (already ran), got %d", len(mockClient.execCalls))
		}
	}
}

// TestMetadataIntegration_CommandChange tests re-execution when command changes
func TestMetadataIntegration_CommandChange(t *testing.T) {
	// Create temp directory for test
	tempDir := t.TempDir()

	// Override XDG_DATA_HOME for test
	originalXDG := os.Getenv("XDG_DATA_HOME")
	os.Setenv("XDG_DATA_HOME", tempDir)
	defer os.Setenv("XDG_DATA_HOME", originalXDG)

	// First command
	cmd1JSON := `"npm install"`
	var cmd1 devcontainer.LifecycleCommand
	if err := json.Unmarshal([]byte(cmd1JSON), &cmd1); err != nil {
		t.Fatalf("Failed to create command: %v", err)
	}

	// Second command (different)
	cmd2JSON := `"npm ci"`
	var cmd2 devcontainer.LifecycleCommand
	if err := json.Unmarshal([]byte(cmd2JSON), &cmd2); err != nil {
		t.Fatalf("Failed to create command: %v", err)
	}

	// First run with cmd1 - should execute
	{
		mockClient := &mockDockerClient{
			execCalls: [][]string{},
		}

		metadata, err := LoadMetadata("test-container-789")
		if err != nil {
			t.Fatalf("LoadMetadata failed: %v", err)
		}

		executor := NewLifecycleExecutor(mockClient, "test-container-789", "testuser", false, metadata)

		err = executor.Execute("onCreate", &cmd1)
		if err != nil {
			t.Fatalf("Execute failed: %v", err)
		}

		// Should have executed
		if len(mockClient.execCalls) != 1 {
			t.Errorf("First run: Expected 1 exec call, got %d", len(mockClient.execCalls))
		}

		// Save metadata
		if err := SaveMetadata(metadata); err != nil {
			t.Fatalf("SaveMetadata failed: %v", err)
		}
	}

	// Second run with cmd2 (changed) - should execute again
	{
		mockClient := &mockDockerClient{
			execCalls: [][]string{},
		}

		metadata, err := LoadMetadata("test-container-789")
		if err != nil {
			t.Fatalf("LoadMetadata failed: %v", err)
		}

		executor := NewLifecycleExecutor(mockClient, "test-container-789", "testuser", false, metadata)

		err = executor.Execute("onCreate", &cmd2)
		if err != nil {
			t.Fatalf("Execute failed: %v", err)
		}

		// Should have executed (command changed)
		if len(mockClient.execCalls) != 1 {
			t.Errorf("Second run: Expected 1 exec call (command changed), got %d", len(mockClient.execCalls))
		}
	}
}

// TestMetadataIntegration_PostStartAlwaysRuns tests that postStart always runs
func TestMetadataIntegration_PostStartAlwaysRuns(t *testing.T) {
	// Create temp directory for test
	tempDir := t.TempDir()

	// Override XDG_DATA_HOME for test
	originalXDG := os.Getenv("XDG_DATA_HOME")
	os.Setenv("XDG_DATA_HOME", tempDir)
	defer os.Setenv("XDG_DATA_HOME", originalXDG)

	// Create postStart command
	cmdJSON := `"echo 'starting'"`
	var cmd devcontainer.LifecycleCommand
	if err := json.Unmarshal([]byte(cmdJSON), &cmd); err != nil {
		t.Fatalf("Failed to create command: %v", err)
	}

	// First run - should execute
	{
		mockClient := &mockDockerClient{
			execCalls: [][]string{},
		}

		metadata, err := LoadMetadata("test-container-poststart")
		if err != nil {
			t.Fatalf("LoadMetadata failed: %v", err)
		}

		executor := NewLifecycleExecutor(mockClient, "test-container-poststart", "testuser", false, metadata)

		err = executor.Execute("postStart", &cmd)
		if err != nil {
			t.Fatalf("Execute failed: %v", err)
		}

		// Should have executed
		if len(mockClient.execCalls) != 1 {
			t.Errorf("First run: Expected 1 exec call, got %d", len(mockClient.execCalls))
		}

		// Save metadata
		if err := SaveMetadata(metadata); err != nil {
			t.Fatalf("SaveMetadata failed: %v", err)
		}
	}

	// Second run - should STILL execute (postStart always runs)
	{
		mockClient := &mockDockerClient{
			execCalls: [][]string{},
		}

		metadata, err := LoadMetadata("test-container-poststart")
		if err != nil {
			t.Fatalf("LoadMetadata failed: %v", err)
		}

		executor := NewLifecycleExecutor(mockClient, "test-container-poststart", "testuser", false, metadata)

		err = executor.Execute("postStart", &cmd)
		if err != nil {
			t.Fatalf("Execute failed: %v", err)
		}

		// Should have executed (postStart always runs)
		if len(mockClient.execCalls) != 1 {
			t.Errorf("Second run: Expected 1 exec call (postStart always runs), got %d", len(mockClient.execCalls))
		}
	}
}

// TestMetadataIntegration_PersistenceAcrossRestarts simulates container restarts
func TestMetadataIntegration_PersistenceAcrossRestarts(t *testing.T) {
	// Create temp directory for test
	tempDir := t.TempDir()

	// Override XDG_DATA_HOME for test
	originalXDG := os.Getenv("XDG_DATA_HOME")
	os.Setenv("XDG_DATA_HOME", tempDir)
	defer os.Setenv("XDG_DATA_HOME", originalXDG)

	containerID := "test-container-persistent"

	// Commands
	onCreateJSON := `"npm install"`
	var onCreate devcontainer.LifecycleCommand
	if err := json.Unmarshal([]byte(onCreateJSON), &onCreate); err != nil {
		t.Fatalf("Failed to create onCreate command: %v", err)
	}

	postCreateJSON := `"npm run build"`
	var postCreate devcontainer.LifecycleCommand
	if err := json.Unmarshal([]byte(postCreateJSON), &postCreate); err != nil {
		t.Fatalf("Failed to create postCreate command: %v", err)
	}

	postStartJSON := `"npm run dev"`
	var postStart devcontainer.LifecycleCommand
	if err := json.Unmarshal([]byte(postStartJSON), &postStart); err != nil {
		t.Fatalf("Failed to create postStart command: %v", err)
	}

	// First container start - all commands should run
	{
		mockClient := &mockDockerClient{execCalls: [][]string{}}
		metadata, _ := LoadMetadata(containerID)
		executor := NewLifecycleExecutor(mockClient, containerID, "testuser", false, metadata)

		_ = executor.Execute("onCreate", &onCreate)
		_ = executor.Execute("postCreate", &postCreate)
		_ = executor.Execute("postStart", &postStart)

		// All should execute
		if len(mockClient.execCalls) != 3 {
			t.Errorf("First start: Expected 3 exec calls, got %d", len(mockClient.execCalls))
		}

		_ = SaveMetadata(metadata)
	}

	// Wait a bit to simulate time passing
	time.Sleep(10 * time.Millisecond)

	// Second container start - only postStart should run
	{
		mockClient := &mockDockerClient{execCalls: [][]string{}}
		metadata, _ := LoadMetadata(containerID)
		executor := NewLifecycleExecutor(mockClient, containerID, "testuser", false, metadata)

		_ = executor.Execute("onCreate", &onCreate)
		_ = executor.Execute("postCreate", &postCreate)
		_ = executor.Execute("postStart", &postStart)

		// Only postStart should execute
		if len(mockClient.execCalls) != 1 {
			t.Errorf("Second start: Expected 1 exec call (postStart only), got %d", len(mockClient.execCalls))
		}

		_ = SaveMetadata(metadata)
	}

	// Third container start with changed onCreate - onCreate and postStart should run
	{
		mockClient := &mockDockerClient{execCalls: [][]string{}}
		metadata, _ := LoadMetadata(containerID)
		executor := NewLifecycleExecutor(mockClient, containerID, "testuser", false, metadata)

		// Change onCreate command
		onCreateChangedJSON := `"npm ci"` // Different command
		var onCreateChanged devcontainer.LifecycleCommand
		if err := json.Unmarshal([]byte(onCreateChangedJSON), &onCreateChanged); err != nil {
			t.Fatalf("Failed to create changed onCreate command: %v", err)
		}

		_ = executor.Execute("onCreate", &onCreateChanged) // Changed
		_ = executor.Execute("postCreate", &postCreate)    // Same
		_ = executor.Execute("postStart", &postStart)      // Always runs

		// onCreate (changed) and postStart should execute
		if len(mockClient.execCalls) != 2 {
			t.Errorf("Third start: Expected 2 exec calls (onCreate changed + postStart), got %d", len(mockClient.execCalls))
		}

		_ = SaveMetadata(metadata)
	}
}
