package runner

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/obra/packnplay/pkg/devcontainer"
)

func TestMetadata_SaveAndLoad(t *testing.T) {
	// Create temp directory for test
	tempDir := t.TempDir()

	// Override XDG_DATA_HOME for test
	originalXDG := os.Getenv("XDG_DATA_HOME")
	os.Setenv("XDG_DATA_HOME", tempDir)
	defer os.Setenv("XDG_DATA_HOME", originalXDG)

	// Create metadata
	metadata := &ContainerMetadata{
		ContainerID: "test-container-123",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		LifecycleRan: map[string]LifecycleState{
			"onCreate": {
				Executed:    true,
				Timestamp:   time.Now(),
				CommandHash: "abc123",
			},
		},
	}

	// Save metadata
	err := SaveMetadata(metadata)
	if err != nil {
		t.Fatalf("SaveMetadata failed: %v", err)
	}

	// Load metadata
	loaded, err := LoadMetadata("test-container-123")
	if err != nil {
		t.Fatalf("LoadMetadata failed: %v", err)
	}

	// Verify loaded data
	if loaded.ContainerID != metadata.ContainerID {
		t.Errorf("ContainerID mismatch: got %s, want %s", loaded.ContainerID, metadata.ContainerID)
	}

	if len(loaded.LifecycleRan) != 1 {
		t.Errorf("LifecycleRan length: got %d, want 1", len(loaded.LifecycleRan))
	}

	state, exists := loaded.LifecycleRan["onCreate"]
	if !exists {
		t.Error("onCreate state not found")
	}

	if !state.Executed {
		t.Error("onCreate should be executed")
	}

	if state.CommandHash != "abc123" {
		t.Errorf("CommandHash mismatch: got %s, want abc123", state.CommandHash)
	}
}

func TestMetadata_LoadNonExistent(t *testing.T) {
	// Create temp directory for test
	tempDir := t.TempDir()

	// Override XDG_DATA_HOME for test
	originalXDG := os.Getenv("XDG_DATA_HOME")
	os.Setenv("XDG_DATA_HOME", tempDir)
	defer os.Setenv("XDG_DATA_HOME", originalXDG)

	// Try to load non-existent metadata
	metadata, err := LoadMetadata("non-existent-container")
	if err != nil {
		t.Fatalf("LoadMetadata should not error for non-existent: %v", err)
	}

	// Should return new metadata with initialized map
	if metadata == nil {
		t.Fatal("metadata should not be nil")
	}

	if metadata.ContainerID != "non-existent-container" {
		t.Errorf("ContainerID mismatch: got %s, want non-existent-container", metadata.ContainerID)
	}

	if metadata.LifecycleRan == nil {
		t.Error("LifecycleRan should be initialized")
	}

	if len(metadata.LifecycleRan) != 0 {
		t.Errorf("LifecycleRan should be empty, got %d items", len(metadata.LifecycleRan))
	}
}

func TestMetadata_ShouldRun_FirstTime(t *testing.T) {
	metadata := &ContainerMetadata{
		ContainerID:  "test-container",
		LifecycleRan: make(map[string]LifecycleState),
	}

	// Create a simple command
	cmdJSON := `"npm install"`
	var cmd devcontainer.LifecycleCommand
	if err := json.Unmarshal([]byte(cmdJSON), &cmd); err != nil {
		t.Fatalf("Failed to create command: %v", err)
	}

	// Should run on first time
	if !metadata.ShouldRun("onCreate", &cmd) {
		t.Error("onCreate should run on first time")
	}

	if !metadata.ShouldRun("postCreate", &cmd) {
		t.Error("postCreate should run on first time")
	}
}

func TestMetadata_ShouldRun_AlreadyRan(t *testing.T) {
	// Create a simple command
	cmdJSON := `"npm install"`
	var cmd devcontainer.LifecycleCommand
	if err := json.Unmarshal([]byte(cmdJSON), &cmd); err != nil {
		t.Fatalf("Failed to create command: %v", err)
	}

	cmdHash := HashCommand(&cmd)

	metadata := &ContainerMetadata{
		ContainerID: "test-container",
		LifecycleRan: map[string]LifecycleState{
			"onCreate": {
				Executed:    true,
				Timestamp:   time.Now(),
				CommandHash: cmdHash,
			},
		},
	}

	// Should NOT run if already executed with same hash
	if metadata.ShouldRun("onCreate", &cmd) {
		t.Error("onCreate should NOT run when already executed with same hash")
	}
}

func TestMetadata_ShouldRun_CommandChanged(t *testing.T) {
	// Create original command
	cmd1JSON := `"npm install"`
	var cmd1 devcontainer.LifecycleCommand
	if err := json.Unmarshal([]byte(cmd1JSON), &cmd1); err != nil {
		t.Fatalf("Failed to create command: %v", err)
	}

	cmd1Hash := HashCommand(&cmd1)

	metadata := &ContainerMetadata{
		ContainerID: "test-container",
		LifecycleRan: map[string]LifecycleState{
			"onCreate": {
				Executed:    true,
				Timestamp:   time.Now(),
				CommandHash: cmd1Hash,
			},
		},
	}

	// Create different command
	cmd2JSON := `"npm ci"`
	var cmd2 devcontainer.LifecycleCommand
	if err := json.Unmarshal([]byte(cmd2JSON), &cmd2); err != nil {
		t.Fatalf("Failed to create command: %v", err)
	}

	// Should run because command changed
	if !metadata.ShouldRun("onCreate", &cmd2) {
		t.Error("onCreate should run when command changed")
	}
}

func TestMetadata_ShouldRun_PostStartAlwaysRuns(t *testing.T) {
	// Create a command
	cmdJSON := `"echo hello"`
	var cmd devcontainer.LifecycleCommand
	if err := json.Unmarshal([]byte(cmdJSON), &cmd); err != nil {
		t.Fatalf("Failed to create command: %v", err)
	}

	cmdHash := HashCommand(&cmd)

	metadata := &ContainerMetadata{
		ContainerID: "test-container",
		LifecycleRan: map[string]LifecycleState{
			"postStart": {
				Executed:    true,
				Timestamp:   time.Now(),
				CommandHash: cmdHash,
			},
		},
	}

	// postStart should ALWAYS run even if already executed
	if !metadata.ShouldRun("postStart", &cmd) {
		t.Error("postStart should ALWAYS run")
	}
}

func TestMetadata_HashCommand_Deterministic(t *testing.T) {
	// Test string command
	cmdJSON := `"npm install"`
	var cmd1 devcontainer.LifecycleCommand
	if err := json.Unmarshal([]byte(cmdJSON), &cmd1); err != nil {
		t.Fatalf("Failed to create command: %v", err)
	}

	var cmd2 devcontainer.LifecycleCommand
	if err := json.Unmarshal([]byte(cmdJSON), &cmd2); err != nil {
		t.Fatalf("Failed to create command: %v", err)
	}

	hash1 := HashCommand(&cmd1)
	hash2 := HashCommand(&cmd2)

	if hash1 != hash2 {
		t.Errorf("Hash should be deterministic: got %s and %s", hash1, hash2)
	}

	// Test different command produces different hash
	diffJSON := `"npm ci"`
	var cmd3 devcontainer.LifecycleCommand
	if err := json.Unmarshal([]byte(diffJSON), &cmd3); err != nil {
		t.Fatalf("Failed to create command: %v", err)
	}

	hash3 := HashCommand(&cmd3)
	if hash1 == hash3 {
		t.Error("Different commands should produce different hashes")
	}
}

func TestMetadata_HashCommand_ArrayCommand(t *testing.T) {
	// Test array command
	cmdJSON := `["npm", "install"]`
	var cmd devcontainer.LifecycleCommand
	if err := json.Unmarshal([]byte(cmdJSON), &cmd); err != nil {
		t.Fatalf("Failed to create command: %v", err)
	}

	hash := HashCommand(&cmd)
	if hash == "" {
		t.Error("Hash should not be empty for array command")
	}

	// Hash should be consistent
	hash2 := HashCommand(&cmd)
	if hash != hash2 {
		t.Error("Array command hash should be consistent")
	}
}

func TestMetadata_HashCommand_ObjectCommand(t *testing.T) {
	// Test object command (parallel tasks)
	cmdJSON := `{"server": "npm start", "watch": "npm run watch"}`
	var cmd devcontainer.LifecycleCommand
	if err := json.Unmarshal([]byte(cmdJSON), &cmd); err != nil {
		t.Fatalf("Failed to create command: %v", err)
	}

	hash := HashCommand(&cmd)
	if hash == "" {
		t.Error("Hash should not be empty for object command")
	}

	// Hash should be consistent
	hash2 := HashCommand(&cmd)
	if hash != hash2 {
		t.Error("Object command hash should be consistent")
	}
}

func TestMetadata_GetMetadataPath(t *testing.T) {
	// Create temp directory for test
	tempDir := t.TempDir()

	// Override XDG_DATA_HOME for test
	originalXDG := os.Getenv("XDG_DATA_HOME")
	os.Setenv("XDG_DATA_HOME", tempDir)
	defer os.Setenv("XDG_DATA_HOME", originalXDG)

	path, err := GetMetadataPath("test-container-123")
	if err != nil {
		t.Fatalf("GetMetadataPath failed: %v", err)
	}

	expectedDir := filepath.Join(tempDir, "packnplay", "metadata")
	expectedPath := filepath.Join(expectedDir, "test-container-123.json")

	if path != expectedPath {
		t.Errorf("Path mismatch: got %s, want %s", path, expectedPath)
	}

	// Verify directory was created
	if _, err := os.Stat(expectedDir); os.IsNotExist(err) {
		t.Error("Metadata directory should be created")
	}
}

func TestMetadata_MarkExecuted(t *testing.T) {
	metadata := &ContainerMetadata{
		ContainerID:  "test-container",
		LifecycleRan: make(map[string]LifecycleState),
	}

	// Create a command
	cmdJSON := `"npm install"`
	var cmd devcontainer.LifecycleCommand
	if err := json.Unmarshal([]byte(cmdJSON), &cmd); err != nil {
		t.Fatalf("Failed to create command: %v", err)
	}

	// Mark as executed
	beforeTime := time.Now().Add(-time.Second)
	metadata.MarkExecuted("onCreate", &cmd)
	afterTime := time.Now().Add(time.Second)

	// Verify state
	state, exists := metadata.LifecycleRan["onCreate"]
	if !exists {
		t.Fatal("onCreate state should exist after marking executed")
	}

	if !state.Executed {
		t.Error("onCreate should be marked as executed")
	}

	if state.CommandHash == "" {
		t.Error("CommandHash should not be empty")
	}

	if state.Timestamp.Before(beforeTime) || state.Timestamp.After(afterTime) {
		t.Error("Timestamp should be set to current time")
	}

	// Verify UpdatedAt was updated
	if metadata.UpdatedAt.Before(beforeTime) || metadata.UpdatedAt.After(afterTime) {
		t.Error("UpdatedAt should be set to current time")
	}
}

func TestMetadata_NilCommand(t *testing.T) {
	metadata := &ContainerMetadata{
		ContainerID:  "test-container",
		LifecycleRan: make(map[string]LifecycleState),
	}

	// ShouldRun with nil command should return false
	if metadata.ShouldRun("onCreate", nil) {
		t.Error("ShouldRun should return false for nil command")
	}

	// HashCommand with nil should return empty string
	hash := HashCommand(nil)
	if hash != "" {
		t.Error("HashCommand should return empty string for nil")
	}
}

func TestMetadata_EmptyCommand(t *testing.T) {
	metadata := &ContainerMetadata{
		ContainerID:  "test-container",
		LifecycleRan: make(map[string]LifecycleState),
	}

	// Create empty string command
	cmdJSON := `""`
	var cmd devcontainer.LifecycleCommand
	if err := json.Unmarshal([]byte(cmdJSON), &cmd); err != nil {
		t.Fatalf("Failed to create command: %v", err)
	}

	// Empty command should still be trackable
	hash := HashCommand(&cmd)
	if hash == "" {
		t.Error("Empty command should still produce a hash")
	}

	// Should run first time
	if !metadata.ShouldRun("onCreate", &cmd) {
		t.Error("Empty command should run first time")
	}

	// Mark as executed
	metadata.MarkExecuted("onCreate", &cmd)

	// Should not run second time
	if metadata.ShouldRun("onCreate", &cmd) {
		t.Error("Empty command should not run second time")
	}
}
