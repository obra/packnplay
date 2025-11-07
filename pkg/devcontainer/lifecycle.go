package devcontainer

import (
	"encoding/json"
	"fmt"
)

// LifecycleCommand represents a lifecycle command that can be a string, array, or object.
// - String: single shell command (e.g., "npm install")
// - Array: command with arguments (e.g., ["npm", "install"])
// - Object: multiple commands to run in parallel (e.g., {"server": "npm start", "watch": "npm run watch"})
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

// AsObject returns the command as an object (map) if it is one
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
