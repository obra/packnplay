package devcontainer

import (
	"encoding/json"
	"testing"
)

func TestParseForwardPorts_Integer(t *testing.T) {
	// JSON numbers are float64
	ports := []interface{}{float64(3000)}

	result, err := ParseForwardPorts(ports)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("Expected 1 port, got %d", len(result))
	}

	if result[0] != "127.0.0.1:3000:3000" {
		t.Errorf("Expected '127.0.0.1:3000:3000', got '%s'", result[0])
	}
}

func TestParseForwardPorts_String(t *testing.T) {
	ports := []interface{}{"8080:80"}

	result, err := ParseForwardPorts(ports)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result[0] != "8080:80" {
		t.Errorf("Expected '8080:80', got '%s'", result[0])
	}
}

func TestParseForwardPorts_Mixed(t *testing.T) {
	ports := []interface{}{
		float64(3000),
		"8080:80",
		"127.0.0.1:9000:9000",
	}

	result, err := ParseForwardPorts(ports)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(result) != 3 {
		t.Fatalf("Expected 3 ports, got %d", len(result))
	}

	expected := []string{"127.0.0.1:3000:3000", "8080:80", "127.0.0.1:9000:9000"}
	for i, exp := range expected {
		if result[i] != exp {
			t.Errorf("Port %d: expected '%s', got '%s'", i, exp, result[i])
		}
	}
}

func TestParseForwardPorts_InvalidType(t *testing.T) {
	ports := []interface{}{true} // boolean is invalid

	_, err := ParseForwardPorts(ports)
	if err == nil {
		t.Error("Expected error for invalid port type")
	}
}

func TestParseForwardPorts_Empty(t *testing.T) {
	ports := []interface{}{}

	result, err := ParseForwardPorts(ports)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(result) != 0 {
		t.Errorf("Expected empty result, got %d ports", len(result))
	}
}

func TestParseForwardPorts_Nil(t *testing.T) {
	result, err := ParseForwardPorts(nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(result) != 0 {
		t.Errorf("Expected empty result for nil input, got %d ports", len(result))
	}
}

func TestParseForwardPorts_NegativePort(t *testing.T) {
	ports := []interface{}{float64(-1)}

	_, err := ParseForwardPorts(ports)
	if err == nil {
		t.Error("Expected error for negative port number")
	}
}

func TestParseForwardPorts_ZeroPort(t *testing.T) {
	ports := []interface{}{float64(0)}

	_, err := ParseForwardPorts(ports)
	if err == nil {
		t.Error("Expected error for zero port number")
	}
}

func TestParseForwardPorts_OutOfRangePort(t *testing.T) {
	ports := []interface{}{float64(65536)}

	_, err := ParseForwardPorts(ports)
	if err == nil {
		t.Error("Expected error for out-of-range port number")
	}
}

func TestOtherPortsAttributes_AppliedToUnspecifiedPorts(t *testing.T) {
	config := &Config{
		ForwardPorts: []interface{}{float64(3000), float64(8080), float64(9000)},
		PortsAttributes: map[string]PortAttributes{
			"3000": {
				Label: "Web",
			},
		},
		OtherPortsAttributes: PortAttributes{
			Label:         "Other Service",
			OnAutoForward: "ignore",
		},
	}

	// Get attributes for port 3000 (explicitly defined)
	attrs3000 := config.GetPortAttributes("3000")
	if attrs3000.Label != "Web" {
		t.Errorf("Expected port 3000 to have label 'Web', got '%s'", attrs3000.Label)
	}
	if attrs3000.OnAutoForward != "" {
		t.Errorf("Expected port 3000 to not inherit onAutoForward, got '%s'", attrs3000.OnAutoForward)
	}

	// Get attributes for port 8080 (should use otherPortsAttributes)
	attrs8080 := config.GetPortAttributes("8080")
	if attrs8080.Label != "Other Service" {
		t.Errorf("Expected port 8080 to have label 'Other Service', got '%s'", attrs8080.Label)
	}
	if attrs8080.OnAutoForward != "ignore" {
		t.Errorf("Expected port 8080 to have onAutoForward 'ignore', got '%s'", attrs8080.OnAutoForward)
	}

	// Get attributes for port 9000 (should use otherPortsAttributes)
	attrs9000 := config.GetPortAttributes("9000")
	if attrs9000.Label != "Other Service" {
		t.Errorf("Expected port 9000 to have label 'Other Service', got '%s'", attrs9000.Label)
	}
	if attrs9000.OnAutoForward != "ignore" {
		t.Errorf("Expected port 9000 to have onAutoForward 'ignore', got '%s'", attrs9000.OnAutoForward)
	}
}

func TestOtherPortsAttributes_NotAppliedWhenExplicitlyDefined(t *testing.T) {
	config := &Config{
		ForwardPorts: []interface{}{float64(3000), float64(8080)},
		PortsAttributes: map[string]PortAttributes{
			"3000": {
				Label:         "Web",
				OnAutoForward: "notify",
			},
			"8080": {
				Label: "API",
			},
		},
		OtherPortsAttributes: PortAttributes{
			Label:         "Other Service",
			OnAutoForward: "ignore",
		},
	}

	// Get attributes for port 3000 (explicitly defined, should not use otherPortsAttributes)
	attrs3000 := config.GetPortAttributes("3000")
	if attrs3000.Label != "Web" {
		t.Errorf("Expected port 3000 to have label 'Web', got '%s'", attrs3000.Label)
	}
	if attrs3000.OnAutoForward != "notify" {
		t.Errorf("Expected port 3000 to have onAutoForward 'notify', got '%s'", attrs3000.OnAutoForward)
	}

	// Get attributes for port 8080 (explicitly defined, should not use otherPortsAttributes)
	attrs8080 := config.GetPortAttributes("8080")
	if attrs8080.Label != "API" {
		t.Errorf("Expected port 8080 to have label 'API', got '%s'", attrs8080.Label)
	}
	// OnAutoForward not set explicitly, should remain empty (not inherit from otherPortsAttributes)
	if attrs8080.OnAutoForward != "" {
		t.Errorf("Expected port 8080 to not have onAutoForward set, got '%s'", attrs8080.OnAutoForward)
	}
}

func TestOtherPortsAttributes_EmptyWhenNotSet(t *testing.T) {
	config := &Config{
		ForwardPorts: []interface{}{float64(3000)},
		PortsAttributes: map[string]PortAttributes{
			"8080": {
				Label: "API",
			},
		},
	}

	// Get attributes for port 3000 (not in portsAttributes, no otherPortsAttributes)
	attrs3000 := config.GetPortAttributes("3000")
	if attrs3000.Label != "" {
		t.Errorf("Expected port 3000 to have empty label, got '%s'", attrs3000.Label)
	}
	if attrs3000.OnAutoForward != "" {
		t.Errorf("Expected port 3000 to have empty onAutoForward, got '%s'", attrs3000.OnAutoForward)
	}
}

func TestOtherPortsAttributes_JSONMarshaling(t *testing.T) {
	jsonStr := `{
		"image": "ubuntu:22.04",
		"forwardPorts": [3000, 8080, 9000],
		"portsAttributes": {
			"3000": {
				"label": "Web"
			}
		},
		"otherPortsAttributes": {
			"label": "Other Service",
			"onAutoForward": "ignore"
		}
	}`

	var config Config
	err := json.Unmarshal([]byte(jsonStr), &config)
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Verify otherPortsAttributes was parsed
	if config.OtherPortsAttributes.Label != "Other Service" {
		t.Errorf("Expected otherPortsAttributes.Label to be 'Other Service', got '%s'", config.OtherPortsAttributes.Label)
	}
	if config.OtherPortsAttributes.OnAutoForward != "ignore" {
		t.Errorf("Expected otherPortsAttributes.OnAutoForward to be 'ignore', got '%s'", config.OtherPortsAttributes.OnAutoForward)
	}

	// Verify GetPortAttributes works correctly
	attrs3000 := config.GetPortAttributes("3000")
	if attrs3000.Label != "Web" {
		t.Errorf("Expected port 3000 to have label 'Web', got '%s'", attrs3000.Label)
	}

	attrs8080 := config.GetPortAttributes("8080")
	if attrs8080.Label != "Other Service" {
		t.Errorf("Expected port 8080 to have label 'Other Service', got '%s'", attrs8080.Label)
	}
	if attrs8080.OnAutoForward != "ignore" {
		t.Errorf("Expected port 8080 to have onAutoForward 'ignore', got '%s'", attrs8080.OnAutoForward)
	}
}
