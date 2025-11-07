package runner

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/obra/packnplay/pkg/devcontainer"
)

// TestLifecycleExecutor_ExecuteString tests executing a string command
func TestLifecycleExecutor_ExecuteString(t *testing.T) {
	mockClient := &mockDockerClient{
		execCalls: [][]string{},
	}

	executor := NewLifecycleExecutor(mockClient, "test-container", "testuser", false)

	// Create a string command
	jsonData := `"npm install"`
	var cmd devcontainer.LifecycleCommand
	if err := cmd.UnmarshalJSON([]byte(jsonData)); err != nil {
		t.Fatalf("Failed to unmarshal command: %v", err)
	}

	err := executor.Execute(&cmd)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify docker exec was called with shell command
	if len(mockClient.execCalls) != 1 {
		t.Fatalf("Expected 1 exec call, got %d", len(mockClient.execCalls))
	}

	execArgs := mockClient.execCalls[0]
	// Should be: exec -u testuser test-container sh -c "npm install"
	if !contains(execArgs, "exec") || !contains(execArgs, "test-container") ||
		!contains(execArgs, "sh") || !contains(execArgs, "-c") {
		t.Errorf("Expected docker exec with shell, got: %v", execArgs)
	}
}

// TestLifecycleExecutor_ExecuteArray tests executing an array command
func TestLifecycleExecutor_ExecuteArray(t *testing.T) {
	mockClient := &mockDockerClient{
		execCalls: [][]string{},
	}

	executor := NewLifecycleExecutor(mockClient, "test-container", "testuser", false)

	// Create an array command
	jsonData := `["npm", "install"]`
	var cmd devcontainer.LifecycleCommand
	if err := cmd.UnmarshalJSON([]byte(jsonData)); err != nil {
		t.Fatalf("Failed to unmarshal command: %v", err)
	}

	err := executor.Execute(&cmd)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify docker exec was called directly with command
	if len(mockClient.execCalls) != 1 {
		t.Fatalf("Expected 1 exec call, got %d", len(mockClient.execCalls))
	}

	execArgs := mockClient.execCalls[0]
	// Should be: exec -u testuser test-container npm install
	if !contains(execArgs, "exec") || !contains(execArgs, "test-container") ||
		!contains(execArgs, "npm") || !contains(execArgs, "install") {
		t.Errorf("Expected docker exec with direct command, got: %v", execArgs)
	}
}

// TestLifecycleExecutor_ExecuteObject tests executing parallel commands
func TestLifecycleExecutor_ExecuteObject(t *testing.T) {
	mockClient := &mockDockerClient{
		execCalls: [][]string{},
	}

	executor := NewLifecycleExecutor(mockClient, "test-container", "testuser", false)

	// Create an object command with 2 parallel tasks
	jsonData := `{
		"task1": "echo 'task 1'",
		"task2": "echo 'task 2'"
	}`
	var cmd devcontainer.LifecycleCommand
	if err := cmd.UnmarshalJSON([]byte(jsonData)); err != nil {
		t.Fatalf("Failed to unmarshal command: %v", err)
	}

	err := executor.Execute(&cmd)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify both commands were executed
	if len(mockClient.execCalls) != 2 {
		t.Fatalf("Expected 2 exec calls for parallel execution, got %d", len(mockClient.execCalls))
	}
}

// TestLifecycleExecutor_ExecuteError tests error handling
func TestLifecycleExecutor_ExecuteError(t *testing.T) {
	mockClient := &mockDockerClient{
		execError: fmt.Errorf("command failed"),
		execCalls: [][]string{},
	}

	executor := NewLifecycleExecutor(mockClient, "test-container", "testuser", false)

	// Create a command that will fail
	jsonData := `"npm install"`
	var cmd devcontainer.LifecycleCommand
	if err := cmd.UnmarshalJSON([]byte(jsonData)); err != nil {
		t.Fatalf("Failed to unmarshal command: %v", err)
	}

	err := executor.Execute(&cmd)
	if err == nil {
		t.Error("Expected error, got nil")
	}
}

// TestLifecycleExecutor_NilCommand tests handling of nil command
func TestLifecycleExecutor_NilCommand(t *testing.T) {
	mockClient := &mockDockerClient{
		execCalls: [][]string{},
	}

	executor := NewLifecycleExecutor(mockClient, "test-container", "testuser", false)

	// Execute nil command (should be no-op)
	err := executor.Execute(nil)
	if err != nil {
		t.Fatalf("Expected no error for nil command, got: %v", err)
	}

	// Should not have called exec
	if len(mockClient.execCalls) != 0 {
		t.Errorf("Expected 0 exec calls for nil command, got %d", len(mockClient.execCalls))
	}
}

// TestLifecycleExecutor_ExecuteAllLifecycle tests executing all lifecycle commands
func TestLifecycleExecutor_ExecuteAllLifecycle(t *testing.T) {
	mockClient := &mockDockerClient{
		execCalls: [][]string{},
	}

	executor := NewLifecycleExecutor(mockClient, "test-container", "testuser", false)

	// Create a config with all lifecycle commands
	jsonData := `{
		"image": "ubuntu:22.04",
		"onCreateCommand": "echo 'onCreate'",
		"postCreateCommand": "echo 'postCreate'",
		"postStartCommand": "echo 'postStart'"
	}`

	var config devcontainer.Config
	if err := json.Unmarshal([]byte(jsonData), &config); err != nil {
		t.Fatalf("Failed to unmarshal config: %v", err)
	}

	// Execute all lifecycle commands
	if config.OnCreateCommand != nil {
		if err := executor.Execute(config.OnCreateCommand); err != nil {
			t.Errorf("onCreate failed: %v", err)
		}
	}

	if config.PostCreateCommand != nil {
		if err := executor.Execute(config.PostCreateCommand); err != nil {
			t.Errorf("postCreate failed: %v", err)
		}
	}

	if config.PostStartCommand != nil {
		if err := executor.Execute(config.PostStartCommand); err != nil {
			t.Errorf("postStart failed: %v", err)
		}
	}

	// Should have executed 3 commands
	if len(mockClient.execCalls) != 3 {
		t.Errorf("Expected 3 exec calls, got %d", len(mockClient.execCalls))
	}
}

// TestLifecycleExecutor_VerboseOutput tests verbose mode
func TestLifecycleExecutor_VerboseOutput(t *testing.T) {
	mockClient := &mockDockerClient{
		execCalls:  [][]string{},
		execOutput: "test output",
	}

	// Create executor in verbose mode
	executor := NewLifecycleExecutor(mockClient, "test-container", "testuser", true)

	jsonData := `"echo 'test'"`
	var cmd devcontainer.LifecycleCommand
	if err := cmd.UnmarshalJSON([]byte(jsonData)); err != nil {
		t.Fatalf("Failed to unmarshal command: %v", err)
	}

	err := executor.Execute(&cmd)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// In verbose mode, output should be captured
	// (actual output testing would require capturing stdout)
}

// Enhanced mockDockerClient with exec tracking
type mockDockerClientWithExec struct {
	mockDockerClient
	execCalls  [][]string
	execOutput string
	execError  error
}

// TestLifecycleExecutor_MultipleParallelErrors tests handling of multiple task failures
func TestLifecycleExecutor_MultipleParallelErrors(t *testing.T) {
	mockClient := &mockDockerClient{
		execError: fmt.Errorf("command failed"),
		execCalls: [][]string{},
	}

	executor := NewLifecycleExecutor(mockClient, "test-container", "testuser", false)

	// Create an object command with 3 tasks that will all fail
	jsonData := `{
		"task1": "echo 'task 1'",
		"task2": "echo 'task 2'",
		"task3": "echo 'task 3'"
	}`
	var cmd devcontainer.LifecycleCommand
	if err := cmd.UnmarshalJSON([]byte(jsonData)); err != nil {
		t.Fatalf("Failed to unmarshal command: %v", err)
	}

	err := executor.Execute(&cmd)
	if err == nil {
		t.Fatal("Expected error when all tasks fail")
	}

	// Error should mention multiple failures
	errMsg := err.Error()
	if !strings.Contains(errMsg, "multiple tasks failed") {
		t.Errorf("Expected error to mention multiple failures, got: %s", errMsg)
	}

	// Should contain task names
	if !strings.Contains(errMsg, "task") {
		t.Errorf("Expected error to include task names, got: %s", errMsg)
	}
}

// contains checks if a string slice contains all the given strings
func contains(slice []string, strs ...string) bool {
	sliceStr := strings.Join(slice, " ")
	for _, s := range strs {
		if !strings.Contains(sliceStr, s) {
			return false
		}
	}
	return true
}
