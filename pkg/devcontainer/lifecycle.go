package devcontainer

import (
	"encoding/json"
	"fmt"
)

// LifecycleCommand represents a lifecycle command that can be a string, array, or object.
// - String: single shell command (e.g., "npm install")
// - Array: command with arguments (e.g., ["npm", "install"])
// - Object: multiple commands to run in parallel (e.g., {"server": "npm start", "watch": "npm run watch"})
//
// Note: Empty commands are valid:
//   - Empty string "": No operation (shell returns immediately)
//   - Empty array []: No operation (no command to execute)
//   - Empty object {}: No operation (no tasks to run)
type LifecycleCommand struct {
	raw interface{} // string | []interface{} | map[string]interface{}
}

// UnmarshalJSON implements custom JSON unmarshaling to handle multiple formats
func (lc *LifecycleCommand) UnmarshalJSON(data []byte) error {
	// Try string
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		lc.raw = s
		return nil
	}

	// Try array
	var arr []interface{}
	if err := json.Unmarshal(data, &arr); err == nil {
		lc.raw = arr
		return nil
	}

	// Try object
	var obj map[string]interface{}
	if err := json.Unmarshal(data, &obj); err == nil {
		lc.raw = obj
		return nil
	}

	return fmt.Errorf("lifecycle command must be string, array, or object")
}

// AsString returns the command as a string if it is one
func (lc *LifecycleCommand) AsString() (string, bool) {
	if s, ok := lc.raw.(string); ok {
		return s, true
	}
	return "", false
}

// AsArray returns the command as a string array if it is one
func (lc *LifecycleCommand) AsArray() ([]string, bool) {
	if arr, ok := lc.raw.([]interface{}); ok {
		result := make([]string, len(arr))
		for i, v := range arr {
			if s, ok := v.(string); ok {
				result[i] = s
			} else {
				return nil, false
			}
		}
		return result, true
	}
	return nil, false
}

// AsObject returns the command as an object (map) if it is one.
// The map values can be:
//   - string: shell command
//   - []interface{}: array command (each element should be string)
//
// Callers must perform type assertions on the values.
func (lc *LifecycleCommand) AsObject() (map[string]interface{}, bool) {
	if obj, ok := lc.raw.(map[string]interface{}); ok {
		return obj, true
	}
	return nil, false
}

// IsString returns true if the command is a string
func (lc *LifecycleCommand) IsString() bool {
	_, ok := lc.raw.(string)
	return ok
}

// IsArray returns true if the command is an array
func (lc *LifecycleCommand) IsArray() bool {
	_, ok := lc.raw.([]interface{})
	return ok
}

// IsObject returns true if the command is an object (parallel commands)
func (lc *LifecycleCommand) IsObject() bool {
	_, ok := lc.raw.(map[string]interface{})
	return ok
}

// IsMerged returns true if the command is a merged command (internal type from lifecycle merger)
func (lc *LifecycleCommand) IsMerged() bool {
	_, ok := lc.raw.(*MergedCommands)
	return ok
}

// AsMerged returns the merged commands if this is a merged command
func (lc *LifecycleCommand) AsMerged() ([]string, bool) {
	if merged, ok := lc.raw.(*MergedCommands); ok {
		return merged.commands, true
	}
	return nil, false
}

// ToStringSlice converts the lifecycle command to a slice of string commands
// This is useful for merging feature and user lifecycle commands
// - String: returns slice with single command
// - Array: joins array elements into a single command string
// - Object: returns slice of all task commands (order may vary)
// - MergedCommands: returns the commands as-is (used by lifecycle merger)
func (lc *LifecycleCommand) ToStringSlice() []string {
	if lc == nil {
		return nil
	}

	// Handle MergedCommands (internal type from lifecycle merger)
	// This must be checked before other types since it's stored in raw
	if merged, ok := lc.raw.(*MergedCommands); ok {
		return merged.commands
	}

	// Handle string command
	if s, ok := lc.AsString(); ok {
		return []string{s}
	}

	// Handle array command - join elements into single command
	if arr, ok := lc.AsArray(); ok {
		if len(arr) == 0 {
			return nil
		}
		// Join array elements with space (e.g., ["npm", "install"] -> "npm install")
		result := ""
		for i, elem := range arr {
			if i > 0 {
				result += " "
			}
			result += elem
		}
		return []string{result}
	}

	// Handle object command (parallel tasks)
	if obj, ok := lc.AsObject(); ok {
		var result []string
		for taskName, taskCmd := range obj {
			// Each task can be a string or array
			switch cmd := taskCmd.(type) {
			case string:
				result = append(result, cmd)
			case []interface{}:
				// Convert array to command string
				cmdStr := ""
				for i, elem := range cmd {
					if i > 0 {
						cmdStr += " "
					}
					if s, ok := elem.(string); ok {
						cmdStr += s
					}
				}
				if cmdStr != "" {
					result = append(result, cmdStr)
				}
			default:
				// Unknown type, use task name as fallback
				result = append(result, fmt.Sprintf("# Task: %s", taskName))
			}
		}
		return result
	}

	return nil
}
