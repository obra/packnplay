package devcontainer

import (
	"encoding/json"
	"reflect"
	"testing"
)

// TestLifecycleCommand_UnmarshalString tests parsing string lifecycle command
func TestLifecycleCommand_UnmarshalString(t *testing.T) {
	jsonData := `{
		"postCreateCommand": "npm install"
	}`

	var config struct {
		PostCreateCommand *LifecycleCommand `json:"postCreateCommand,omitempty"`
	}

	if err := json.Unmarshal([]byte(jsonData), &config); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if config.PostCreateCommand == nil {
		t.Fatal("Expected postCreateCommand to be set")
	}

	expected := "npm install"
	if str, ok := config.PostCreateCommand.AsString(); !ok || str != expected {
		t.Errorf("Expected string '%s', got ok=%v, str='%s'", expected, ok, str)
	}
}

// TestLifecycleCommand_UnmarshalArray tests parsing array lifecycle command
func TestLifecycleCommand_UnmarshalArray(t *testing.T) {
	jsonData := `{
		"postCreateCommand": ["npm", "install"]
	}`

	var config struct {
		PostCreateCommand *LifecycleCommand `json:"postCreateCommand,omitempty"`
	}

	if err := json.Unmarshal([]byte(jsonData), &config); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if config.PostCreateCommand == nil {
		t.Fatal("Expected postCreateCommand to be set")
	}

	expected := []string{"npm", "install"}
	if arr, ok := config.PostCreateCommand.AsArray(); !ok || !reflect.DeepEqual(arr, expected) {
		t.Errorf("Expected array %v, got ok=%v, arr=%v", expected, ok, arr)
	}
}

// TestLifecycleCommand_UnmarshalObject tests parsing object lifecycle command (parallel)
func TestLifecycleCommand_UnmarshalObject(t *testing.T) {
	jsonData := `{
		"postCreateCommand": {
			"task1": "echo 'task 1'",
			"task2": "echo 'task 2'"
		}
	}`

	var config struct {
		PostCreateCommand *LifecycleCommand `json:"postCreateCommand,omitempty"`
	}

	if err := json.Unmarshal([]byte(jsonData), &config); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if config.PostCreateCommand == nil {
		t.Fatal("Expected postCreateCommand to be set")
	}

	obj, ok := config.PostCreateCommand.AsObject()
	if !ok {
		t.Fatal("Expected object type")
	}

	if len(obj) != 2 {
		t.Errorf("Expected 2 tasks, got %d", len(obj))
	}

	if task1, exists := obj["task1"]; !exists || task1 != "echo 'task 1'" {
		t.Errorf("Expected task1='echo 'task 1'', got exists=%v, task1='%v'", exists, task1)
	}
}

// TestLifecycleCommand_UnmarshalNestedArray tests object with array values
func TestLifecycleCommand_UnmarshalNestedArray(t *testing.T) {
	jsonData := `{
		"postCreateCommand": {
			"task1": ["npm", "install"],
			"task2": "echo 'done'"
		}
	}`

	var config struct {
		PostCreateCommand *LifecycleCommand `json:"postCreateCommand,omitempty"`
	}

	if err := json.Unmarshal([]byte(jsonData), &config); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	obj, ok := config.PostCreateCommand.AsObject()
	if !ok {
		t.Fatal("Expected object type")
	}

	// Verify task1 is an array
	if task1, exists := obj["task1"]; !exists {
		t.Error("Expected task1 to exist")
	} else {
		// task1 should be a []interface{}
		arr, isArray := task1.([]interface{})
		if !isArray {
			t.Errorf("Expected task1 to be array, got %T", task1)
		} else if len(arr) != 2 {
			t.Errorf("Expected task1 array length 2, got %d", len(arr))
		}
	}
}

// TestLifecycleCommand_InvalidType tests error handling for invalid types
func TestLifecycleCommand_InvalidType(t *testing.T) {
	jsonData := `{
		"postCreateCommand": 123
	}`

	var config struct {
		PostCreateCommand *LifecycleCommand `json:"postCreateCommand,omitempty"`
	}

	err := json.Unmarshal([]byte(jsonData), &config)
	if err == nil {
		t.Error("Expected error for invalid lifecycle command type")
	}
}

// TestLifecycleCommand_NullValue tests null lifecycle command
func TestLifecycleCommand_NullValue(t *testing.T) {
	jsonData := `{
		"postCreateCommand": null
	}`

	var config struct {
		PostCreateCommand *LifecycleCommand `json:"postCreateCommand,omitempty"`
	}

	if err := json.Unmarshal([]byte(jsonData), &config); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if config.PostCreateCommand != nil {
		t.Error("Expected postCreateCommand to be nil")
	}
}

// TestConfig_WithLifecycleCommands tests Config struct with lifecycle fields
func TestConfig_WithLifecycleCommands(t *testing.T) {
	jsonData := `{
		"image": "ubuntu:22.04",
		"onCreateCommand": "npm install",
		"postCreateCommand": ["npm", "run", "build"],
		"postStartCommand": {
			"server": "npm start",
			"watch": "npm run watch"
		}
	}`

	var config Config
	if err := json.Unmarshal([]byte(jsonData), &config); err != nil {
		t.Fatalf("Failed to unmarshal config: %v", err)
	}

	if config.Image != "ubuntu:22.04" {
		t.Errorf("Expected image='ubuntu:22.04', got '%s'", config.Image)
	}

	if config.OnCreateCommand == nil {
		t.Error("Expected onCreateCommand to be set")
	}

	if config.PostCreateCommand == nil {
		t.Error("Expected postCreateCommand to be set")
	}

	if config.PostStartCommand == nil {
		t.Error("Expected postStartCommand to be set")
	}
}

// TestLifecycleCommand_EmptyString tests empty string command
func TestLifecycleCommand_EmptyString(t *testing.T) {
	jsonData := `{
		"postCreateCommand": ""
	}`

	var config struct {
		PostCreateCommand *LifecycleCommand `json:"postCreateCommand,omitempty"`
	}

	if err := json.Unmarshal([]byte(jsonData), &config); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if config.PostCreateCommand == nil {
		t.Fatal("Expected postCreateCommand to be set")
	}

	if str, ok := config.PostCreateCommand.AsString(); !ok || str != "" {
		t.Errorf("Expected empty string, got ok=%v, str='%s'", ok, str)
	}
}

// TestLifecycleCommand_EmptyArray tests empty array command
func TestLifecycleCommand_EmptyArray(t *testing.T) {
	jsonData := `{
		"postCreateCommand": []
	}`

	var config struct {
		PostCreateCommand *LifecycleCommand `json:"postCreateCommand,omitempty"`
	}

	if err := json.Unmarshal([]byte(jsonData), &config); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if config.PostCreateCommand == nil {
		t.Fatal("Expected postCreateCommand to be set")
	}

	if arr, ok := config.PostCreateCommand.AsArray(); !ok || len(arr) != 0 {
		t.Errorf("Expected empty array, got ok=%v, len=%d", ok, len(arr))
	}
}

// TestLifecycleCommand_EmptyObject tests empty object command
func TestLifecycleCommand_EmptyObject(t *testing.T) {
	jsonData := `{
		"postCreateCommand": {}
	}`

	var config struct {
		PostCreateCommand *LifecycleCommand `json:"postCreateCommand,omitempty"`
	}

	if err := json.Unmarshal([]byte(jsonData), &config); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if config.PostCreateCommand == nil {
		t.Fatal("Expected postCreateCommand to be set")
	}

	if obj, ok := config.PostCreateCommand.AsObject(); !ok || len(obj) != 0 {
		t.Errorf("Expected empty object, got ok=%v, len=%d", ok, len(obj))
	}
}
