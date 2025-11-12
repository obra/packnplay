package cmd

import (
	"os"
	"strings"
	"testing"

	"github.com/obra/packnplay/pkg/config"
)

func TestExpandEnvVars(t *testing.T) {
	// Set test environment variables
	if err := os.Setenv("TEST_API_KEY", "sk-test-123"); err != nil {
		t.Fatalf("Failed to set TEST_API_KEY: %v", err)
	}
	if err := os.Setenv("TEST_URL", "https://api.example.com"); err != nil {
		t.Fatalf("Failed to set TEST_URL: %v", err)
	}
	defer func() {
		_ = os.Unsetenv("TEST_API_KEY")
		_ = os.Unsetenv("TEST_URL")
	}()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple variable substitution",
			input:    "${TEST_API_KEY}",
			expected: "sk-test-123",
		},
		{
			name:     "variable in string",
			input:    "Bearer ${TEST_API_KEY}",
			expected: "Bearer sk-test-123",
		},
		{
			name:     "multiple variables",
			input:    "${TEST_URL}/key/${TEST_API_KEY}",
			expected: "https://api.example.com/key/sk-test-123",
		},
		{
			name:     "no variables",
			input:    "plain string",
			expected: "plain string",
		},
		{
			name:     "undefined variable",
			input:    "${UNDEFINED_VAR}",
			expected: "",
		},
		{
			name:     "malformed variable (no closing brace)",
			input:    "${TEST_API_KEY",
			expected: "${TEST_API_KEY",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := expandEnvVars(tt.input)
			if result != tt.expected {
				t.Errorf("expandEnvVars(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestApplyEnvConfig(t *testing.T) {
	// Set test environment variables
	if err := os.Setenv("Z_AI_API_KEY", "zai-123"); err != nil {
		t.Fatalf("Failed to set Z_AI_API_KEY: %v", err)
	}
	if err := os.Setenv("ANTHROPIC_API_KEY", "sk-ant-456"); err != nil {
		t.Fatalf("Failed to set ANTHROPIC_API_KEY: %v", err)
	}
	defer func() {
		_ = os.Unsetenv("Z_AI_API_KEY")
		_ = os.Unsetenv("ANTHROPIC_API_KEY")
	}()

	tests := []struct {
		name     string
		config   config.EnvConfig
		expected map[string]string
	}{
		{
			name: "z.ai config with substitution",
			config: config.EnvConfig{
				Name: "Z.AI Claude",
				EnvVars: map[string]string{
					"ANTHROPIC_AUTH_TOKEN": "${Z_AI_API_KEY}",
					"ANTHROPIC_BASE_URL":   "https://api.z.ai/api/anthropic",
					"API_TIMEOUT_MS":       "3000000",
				},
			},
			expected: map[string]string{
				"ANTHROPIC_AUTH_TOKEN": "zai-123",
				"ANTHROPIC_BASE_URL":   "https://api.z.ai/api/anthropic",
				"API_TIMEOUT_MS":       "3000000",
			},
		},
		{
			name: "anthropic config",
			config: config.EnvConfig{
				Name: "Anthropic API",
				EnvVars: map[string]string{
					"ANTHROPIC_API_KEY":            "${ANTHROPIC_API_KEY}",
					"ANTHROPIC_DEFAULT_OPUS_MODEL": "claude-3-5-opus-20241022",
				},
			},
			expected: map[string]string{
				"ANTHROPIC_API_KEY":            "sk-ant-456",
				"ANTHROPIC_DEFAULT_OPUS_MODEL": "claude-3-5-opus-20241022",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := applyEnvConfig(tt.config)

			// Convert result slice to map for easier comparison
			resultMap := make(map[string]string)
			for _, env := range result {
				parts := strings.SplitN(env, "=", 2)
				if len(parts) == 2 {
					resultMap[parts[0]] = parts[1]
				}
			}

			// Check all expected values
			for key, expectedValue := range tt.expected {
				if actualValue, exists := resultMap[key]; !exists {
					t.Errorf("Expected env var %s not found in result", key)
				} else if actualValue != expectedValue {
					t.Errorf("Env var %s = %q, want %q", key, actualValue, expectedValue)
				}
			}

			// Check no extra values
			if len(resultMap) != len(tt.expected) {
				t.Errorf("Expected %d env vars, got %d", len(tt.expected), len(resultMap))
			}
		})
	}
}
