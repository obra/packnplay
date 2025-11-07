package devcontainer

import "testing"

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

	if result[0] != "3000:3000" {
		t.Errorf("Expected '3000:3000', got '%s'", result[0])
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

	expected := []string{"3000:3000", "8080:80", "127.0.0.1:9000:9000"}
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
