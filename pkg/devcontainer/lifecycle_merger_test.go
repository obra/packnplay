package devcontainer

import (
	"encoding/json"
	"testing"
)

func TestMergeLifecycleCommands(t *testing.T) {
	// Create features with lifecycle commands
	feature1Metadata := &FeatureMetadata{
		ID: "feature1",
		OnCreateCommand: &LifecycleCommand{
			raw: "echo 'feature1 onCreate'",
		},
		PostCreateCommand: &LifecycleCommand{
			raw: "echo 'feature1 postCreate'",
		},
	}

	feature2Metadata := &FeatureMetadata{
		ID: "feature2",
		PostCreateCommand: &LifecycleCommand{
			raw: "echo 'feature2 postCreate'",
		},
		PostStartCommand: &LifecycleCommand{
			raw: "echo 'feature2 postStart'",
		},
	}

	feature1 := &ResolvedFeature{
		ID:       "feature1",
		Metadata: feature1Metadata,
	}

	feature2 := &ResolvedFeature{
		ID:       "feature2",
		Metadata: feature2Metadata,
	}

	// User commands
	userOnCreate := &LifecycleCommand{
		raw: "echo 'user onCreate'",
	}
	userPostCreate := &LifecycleCommand{
		raw: "echo 'user postCreate'",
	}

	merger := NewLifecycleMerger()
	merged := merger.MergeCommands([]*ResolvedFeature{feature1, feature2}, map[string]*LifecycleCommand{
		"onCreateCommand":   userOnCreate,
		"postCreateCommand": userPostCreate,
	})

	// Verify feature commands come before user commands
	onCreate := merged["onCreateCommand"]
	if onCreate == nil {
		t.Fatal("Expected onCreateCommand to be set")
	}

	// Should have: feature1.onCreateCommand, userOnCreate
	// For this test, we'll verify by converting to string slice
	onCreateCommands := onCreate.ToStringSlice()
	if len(onCreateCommands) != 2 {
		t.Errorf("Expected 2 onCreate commands, got %d", len(onCreateCommands))
	}
	if len(onCreateCommands) >= 1 && onCreateCommands[0] != "echo 'feature1 onCreate'" {
		t.Errorf("Expected first onCreate to be feature1, got %s", onCreateCommands[0])
	}
	if len(onCreateCommands) >= 2 && onCreateCommands[1] != "echo 'user onCreate'" {
		t.Errorf("Expected second onCreate to be user, got %s", onCreateCommands[1])
	}

	postCreate := merged["postCreateCommand"]
	if postCreate == nil {
		t.Fatal("Expected postCreateCommand to be set")
	}

	// Should have: feature1.postCreateCommand, feature2.postCreateCommand, userPostCreate
	postCreateCommands := postCreate.ToStringSlice()
	if len(postCreateCommands) != 3 {
		t.Errorf("Expected 3 postCreate commands, got %d", len(postCreateCommands))
	}
	if len(postCreateCommands) >= 1 && postCreateCommands[0] != "echo 'feature1 postCreate'" {
		t.Errorf("Expected first postCreate to be feature1, got %s", postCreateCommands[0])
	}
	if len(postCreateCommands) >= 2 && postCreateCommands[1] != "echo 'feature2 postCreate'" {
		t.Errorf("Expected second postCreate to be feature2, got %s", postCreateCommands[1])
	}
	if len(postCreateCommands) >= 3 && postCreateCommands[2] != "echo 'user postCreate'" {
		t.Errorf("Expected third postCreate to be user, got %s", postCreateCommands[2])
	}

	// Verify postStart only has feature command (no user command)
	postStart := merged["postStartCommand"]
	if postStart == nil {
		t.Fatal("Expected postStartCommand to be set")
	}
	postStartCommands := postStart.ToStringSlice()
	if len(postStartCommands) != 1 {
		t.Errorf("Expected 1 postStart command, got %d", len(postStartCommands))
	}
	if len(postStartCommands) >= 1 && postStartCommands[0] != "echo 'feature2 postStart'" {
		t.Errorf("Expected postStart to be feature2, got %s", postStartCommands[0])
	}
}

func TestMergeLifecycleCommands_NoFeatureCommands(t *testing.T) {
	// Feature with no lifecycle commands
	feature := &ResolvedFeature{
		ID:       "feature1",
		Metadata: &FeatureMetadata{ID: "feature1"},
	}

	// User command only
	userPostCreate := &LifecycleCommand{
		raw: "echo 'user postCreate'",
	}

	merger := NewLifecycleMerger()
	merged := merger.MergeCommands([]*ResolvedFeature{feature}, map[string]*LifecycleCommand{
		"postCreateCommand": userPostCreate,
	})

	// Should only have user command
	postCreate := merged["postCreateCommand"]
	if postCreate == nil {
		t.Fatal("Expected postCreateCommand to be set")
	}

	commands := postCreate.ToStringSlice()
	if len(commands) != 1 {
		t.Errorf("Expected 1 command, got %d", len(commands))
	}
	if len(commands) >= 1 && commands[0] != "echo 'user postCreate'" {
		t.Errorf("Expected user command, got %s", commands[0])
	}
}

func TestMergeLifecycleCommands_NoUserCommands(t *testing.T) {
	// Feature with lifecycle commands
	featureMetadata := &FeatureMetadata{
		ID: "feature1",
		PostCreateCommand: &LifecycleCommand{
			raw: "echo 'feature postCreate'",
		},
	}

	feature := &ResolvedFeature{
		ID:       "feature1",
		Metadata: featureMetadata,
	}

	merger := NewLifecycleMerger()
	merged := merger.MergeCommands([]*ResolvedFeature{feature}, map[string]*LifecycleCommand{})

	// Should only have feature command
	postCreate := merged["postCreateCommand"]
	if postCreate == nil {
		t.Fatal("Expected postCreateCommand to be set")
	}

	commands := postCreate.ToStringSlice()
	if len(commands) != 1 {
		t.Errorf("Expected 1 command, got %d", len(commands))
	}
	if len(commands) >= 1 && commands[0] != "echo 'feature postCreate'" {
		t.Errorf("Expected feature command, got %s", commands[0])
	}
}

func TestMergeLifecycleCommands_ArrayCommands(t *testing.T) {
	// Feature with array command
	featureMetadata := &FeatureMetadata{
		ID: "feature1",
		PostCreateCommand: &LifecycleCommand{
			raw: []interface{}{"npm", "install"},
		},
	}

	feature := &ResolvedFeature{
		ID:       "feature1",
		Metadata: featureMetadata,
	}

	// User with array command
	userPostCreate := &LifecycleCommand{
		raw: []interface{}{"npm", "run", "build"},
	}

	merger := NewLifecycleMerger()
	merged := merger.MergeCommands([]*ResolvedFeature{feature}, map[string]*LifecycleCommand{
		"postCreateCommand": userPostCreate,
	})

	postCreate := merged["postCreateCommand"]
	if postCreate == nil {
		t.Fatal("Expected postCreateCommand to be set")
	}

	commands := postCreate.ToStringSlice()
	// Should have 2 commands (one array command per feature/user)
	if len(commands) != 2 {
		t.Errorf("Expected 2 commands, got %d", len(commands))
	}
}

func TestMergeLifecycleCommands_AllHookTypes(t *testing.T) {
	// Test all five lifecycle hook types
	featureMetadata := &FeatureMetadata{
		ID: "feature1",
		OnCreateCommand: &LifecycleCommand{
			raw: "echo 'onCreate'",
		},
		UpdateContentCommand: &LifecycleCommand{
			raw: "echo 'updateContent'",
		},
		PostCreateCommand: &LifecycleCommand{
			raw: "echo 'postCreate'",
		},
		PostStartCommand: &LifecycleCommand{
			raw: "echo 'postStart'",
		},
		PostAttachCommand: &LifecycleCommand{
			raw: "echo 'postAttach'",
		},
	}

	feature := &ResolvedFeature{
		ID:       "feature1",
		Metadata: featureMetadata,
	}

	merger := NewLifecycleMerger()
	merged := merger.MergeCommands([]*ResolvedFeature{feature}, map[string]*LifecycleCommand{})

	// Verify all hook types are present
	hookTypes := []string{"onCreateCommand", "updateContentCommand", "postCreateCommand", "postStartCommand", "postAttachCommand"}
	for _, hookType := range hookTypes {
		if merged[hookType] == nil {
			t.Errorf("Expected %s to be set", hookType)
		}
	}
}

func TestMergeLifecycleCommands_WithObjectCommands(t *testing.T) {
	// Feature with object command (parallel tasks)
	objectCmd := make(map[string]interface{})
	objectCmd["task1"] = "echo 'task1'"
	objectCmd["task2"] = "echo 'task2'"

	featureMetadata := &FeatureMetadata{
		ID: "feature1",
		PostCreateCommand: &LifecycleCommand{
			raw: objectCmd,
		},
	}

	feature := &ResolvedFeature{
		ID:       "feature1",
		Metadata: featureMetadata,
	}

	merger := NewLifecycleMerger()
	merged := merger.MergeCommands([]*ResolvedFeature{feature}, map[string]*LifecycleCommand{})

	postCreate := merged["postCreateCommand"]
	if postCreate == nil {
		t.Fatal("Expected postCreateCommand to be set")
	}

	// Object commands should be converted to string slice (task names or serialized)
	commands := postCreate.ToStringSlice()
	if len(commands) == 0 {
		t.Error("Expected at least one command from object")
	}
}

func TestLifecycleCommand_ToStringSlice(t *testing.T) {
	tests := []struct {
		name     string
		raw      interface{}
		expected []string
	}{
		{
			name:     "string command",
			raw:      "echo 'hello'",
			expected: []string{"echo 'hello'"},
		},
		{
			name:     "array command",
			raw:      []interface{}{"npm", "install"},
			expected: []string{"npm install"},
		},
		{
			name: "object command",
			raw: map[string]interface{}{
				"task1": "echo 'task1'",
				"task2": "echo 'task2'",
			},
			expected: nil, // Will check length > 0 instead of exact match due to map iteration order
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lc := &LifecycleCommand{raw: tt.raw}
			result := lc.ToStringSlice()

			if tt.expected != nil {
				// Exact match expected
				if len(result) != len(tt.expected) {
					t.Errorf("Expected %d commands, got %d", len(tt.expected), len(result))
					return
				}
				for i, expected := range tt.expected {
					if result[i] != expected {
						t.Errorf("Expected command[%d]=%s, got %s", i, expected, result[i])
					}
				}
			} else {
				// Just check we got something (for object commands)
				if len(result) == 0 {
					t.Error("Expected at least one command")
				}
			}
		})
	}
}

func TestLifecycleCommand_ToStringSlice_ParsesJSON(t *testing.T) {
	// Test that ToStringSlice correctly handles commands parsed from JSON
	jsonData := `{
		"cmd1": "npm install",
		"cmd2": ["npm", "run", "build"],
		"cmd3": {
			"task1": "echo 'task1'",
			"task2": ["echo", "task2"]
		}
	}`

	var commands map[string]*LifecycleCommand
	if err := json.Unmarshal([]byte(jsonData), &commands); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// Test string command
	if cmd1 := commands["cmd1"]; cmd1 != nil {
		slice := cmd1.ToStringSlice()
		if len(slice) != 1 || slice[0] != "npm install" {
			t.Errorf("Expected ['npm install'], got %v", slice)
		}
	}

	// Test array command
	if cmd2 := commands["cmd2"]; cmd2 != nil {
		slice := cmd2.ToStringSlice()
		if len(slice) != 1 || slice[0] != "npm run build" {
			t.Errorf("Expected ['npm run build'], got %v", slice)
		}
	}

	// Test object command
	if cmd3 := commands["cmd3"]; cmd3 != nil {
		slice := cmd3.ToStringSlice()
		if len(slice) != 2 {
			t.Errorf("Expected 2 commands, got %d", len(slice))
		}
	}
}
