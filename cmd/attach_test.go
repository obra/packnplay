package cmd

import (
	"encoding/json"
	"testing"

	"github.com/obra/packnplay/pkg/devcontainer"
)

// TestPostAttachCommand_StringFormat tests that postAttachCommand in string format
// is correctly converted to a slice of commands
func TestPostAttachCommand_StringFormat(t *testing.T) {
	jsonData := []byte(`{
		"postAttachCommand": "echo 'hello world'"
	}`)

	var config devcontainer.Config
	err := json.Unmarshal(jsonData, &config)
	if err != nil {
		t.Fatalf("Failed to unmarshal config: %v", err)
	}

	if config.PostAttachCommand == nil {
		t.Fatal("Expected postAttachCommand to be set")
	}

	commands := config.PostAttachCommand.ToStringSlice()
	if len(commands) != 1 {
		t.Errorf("Expected 1 command, got %d", len(commands))
	}

	if len(commands) > 0 && commands[0] != "echo 'hello world'" {
		t.Errorf("Expected command to be 'echo 'hello world'', got %q", commands[0])
	}
}

// TestPostAttachCommand_ArrayFormat tests that postAttachCommand in array format
// is correctly converted to a slice of commands
func TestPostAttachCommand_ArrayFormat(t *testing.T) {
	jsonData := []byte(`{
		"postAttachCommand": ["npm", "start"]
	}`)

	var config devcontainer.Config
	err := json.Unmarshal(jsonData, &config)
	if err != nil {
		t.Fatalf("Failed to unmarshal config: %v", err)
	}

	if config.PostAttachCommand == nil {
		t.Fatal("Expected postAttachCommand to be set")
	}

	commands := config.PostAttachCommand.ToStringSlice()
	if len(commands) != 1 {
		t.Errorf("Expected 1 command, got %d", len(commands))
	}

	// Array format gets joined into a single command string
	if len(commands) > 0 && commands[0] != "npm start" {
		t.Errorf("Expected command to be 'npm start', got %q", commands[0])
	}
}

// TestPostAttachCommand_ObjectFormat tests that postAttachCommand in object format
// is correctly converted to a slice of commands
func TestPostAttachCommand_ObjectFormat(t *testing.T) {
	jsonData := []byte(`{
		"postAttachCommand": {
			"task1": "touch /tmp/task1",
			"task2": "touch /tmp/task2"
		}
	}`)

	var config devcontainer.Config
	err := json.Unmarshal(jsonData, &config)
	if err != nil {
		t.Fatalf("Failed to unmarshal config: %v", err)
	}

	if config.PostAttachCommand == nil {
		t.Fatal("Expected postAttachCommand to be set")
	}

	commands := config.PostAttachCommand.ToStringSlice()
	if len(commands) != 2 {
		t.Errorf("Expected 2 commands, got %d", len(commands))
	}

	// Object format returns all task commands (order may vary due to map iteration)
	// So we check that both commands are present
	commandSet := make(map[string]bool)
	for _, cmd := range commands {
		commandSet[cmd] = true
	}

	if !commandSet["touch /tmp/task1"] {
		t.Error("Expected to find command 'touch /tmp/task1'")
	}
	if !commandSet["touch /tmp/task2"] {
		t.Error("Expected to find command 'touch /tmp/task2'")
	}
}

// TestPostAttachCommand_ObjectWithArrayValues tests that postAttachCommand object format
// with array values is correctly handled
func TestPostAttachCommand_ObjectWithArrayValues(t *testing.T) {
	jsonData := []byte(`{
		"postAttachCommand": {
			"server": ["npm", "run", "server"],
			"watch": "npm run watch"
		}
	}`)

	var config devcontainer.Config
	err := json.Unmarshal(jsonData, &config)
	if err != nil {
		t.Fatalf("Failed to unmarshal config: %v", err)
	}

	if config.PostAttachCommand == nil {
		t.Fatal("Expected postAttachCommand to be set")
	}

	commands := config.PostAttachCommand.ToStringSlice()
	if len(commands) != 2 {
		t.Errorf("Expected 2 commands, got %d", len(commands))
	}

	// Object format returns all task commands (order may vary)
	commandSet := make(map[string]bool)
	for _, cmd := range commands {
		commandSet[cmd] = true
	}

	// Array values should be joined into a single command string
	if !commandSet["npm run server"] {
		t.Error("Expected to find command 'npm run server'")
	}
	if !commandSet["npm run watch"] {
		t.Error("Expected to find command 'npm run watch'")
	}
}

// TestPostAttachCommand_EmptyString tests that empty string commands are filtered out
func TestPostAttachCommand_EmptyString(t *testing.T) {
	jsonData := []byte(`{
		"postAttachCommand": ""
	}`)

	var config devcontainer.Config
	err := json.Unmarshal(jsonData, &config)
	if err != nil {
		t.Fatalf("Failed to unmarshal config: %v", err)
	}

	if config.PostAttachCommand == nil {
		t.Fatal("Expected postAttachCommand to be set")
	}

	commands := config.PostAttachCommand.ToStringSlice()
	// Empty string should return a slice with one empty string
	// The attach command should skip it in the loop
	if len(commands) != 1 {
		t.Errorf("Expected 1 command, got %d", len(commands))
	}
	if len(commands) > 0 && commands[0] != "" {
		t.Errorf("Expected empty string, got %q", commands[0])
	}
}

// TestPostAttachCommand_EmptyArray tests that empty array returns nil
func TestPostAttachCommand_EmptyArray(t *testing.T) {
	jsonData := []byte(`{
		"postAttachCommand": []
	}`)

	var config devcontainer.Config
	err := json.Unmarshal(jsonData, &config)
	if err != nil {
		t.Fatalf("Failed to unmarshal config: %v", err)
	}

	if config.PostAttachCommand == nil {
		t.Fatal("Expected postAttachCommand to be set")
	}

	commands := config.PostAttachCommand.ToStringSlice()
	// Empty array should return nil according to lifecycle.go
	if commands != nil {
		t.Errorf("Expected nil for empty array, got %v", commands)
	}
}

// TestPostAttachCommand_Nil tests that nil postAttachCommand returns nil
func TestPostAttachCommand_Nil(t *testing.T) {
	jsonData := []byte(`{
		"name": "test"
	}`)

	var config devcontainer.Config
	err := json.Unmarshal(jsonData, &config)
	if err != nil {
		t.Fatalf("Failed to unmarshal config: %v", err)
	}

	if config.PostAttachCommand != nil {
		t.Fatal("Expected postAttachCommand to be nil")
	}
}
