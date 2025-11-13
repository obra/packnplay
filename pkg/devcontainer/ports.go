package devcontainer

import "fmt"

// ParseForwardPorts converts forwardPorts array to Docker -p format
// Input: [3000, "8080:8080", "127.0.0.1:9000:9000"]
// Output: ["3000:3000", "8080:8080", "127.0.0.1:9000:9000"]
func ParseForwardPorts(ports []interface{}) ([]string, error) {
	if ports == nil {
		return []string{}, nil
	}

	result := make([]string, 0, len(ports))

	for _, port := range ports {
		switch v := port.(type) {
		case float64:
			// JSON numbers are float64
			// Single port: 3000 → "3000:3000"
			// Validate port range
			portNum := int(v)
			if portNum < 1 || portNum > 65535 {
				return nil, fmt.Errorf("invalid port number: %d (must be 1-65535)", portNum)
			}
			// Default to localhost binding for security (matches Microsoft behavior)
			portStr := fmt.Sprintf("127.0.0.1:%d:%d", portNum, portNum)
			result = append(result, portStr)

		case string:
			// Already formatted: "8080:8080" or "127.0.0.1:8080:8080"
			result = append(result, v)

		default:
			return nil, fmt.Errorf("invalid port type: %T (expected number or string)", port)
		}
	}

	return result, nil
}
