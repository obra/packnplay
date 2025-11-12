package aws

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseAWSConfig(t *testing.T) {
	tests := []struct {
		name          string
		configContent string
		profile       string
		wantProcess   string
		wantErr       bool
		errContains   string
	}{
		{
			name: "simple profile with credential_process",
			configContent: `[profile test]
credential_process = granted credential-process --profile test`,
			profile:     "test",
			wantProcess: "granted credential-process --profile test",
			wantErr:     false,
		},
		{
			name: "profile with inline comment",
			configContent: `[profile prod]
credential_process = aws-vault exec prod --json  # Production credentials
region = us-west-2`,
			profile:     "prod",
			wantProcess: "aws-vault exec prod --json",
			wantErr:     false,
		},
		{
			name: "profile with extra whitespace",
			configContent: `[profile  spaced  ]
credential_process = some-command`,
			profile:     "spaced",
			wantProcess: "some-command",
			wantErr:     false,
		},
		{
			name: "default profile",
			configContent: `[default]
credential_process = aws-vault exec default --json`,
			profile:     "default",
			wantProcess: "aws-vault exec default --json",
			wantErr:     false,
		},
		{
			name: "profile not found",
			configContent: `[profile test]
credential_process = test-command`,
			profile:     "nonexistent",
			wantErr:     true,
			errContains: "not found",
		},
		{
			name: "profile exists but no credential_process",
			configContent: `[profile test]
region = us-east-1
output = json`,
			profile:     "test",
			wantErr:     true,
			errContains: "no credential_process configured",
		},
		{
			name: "multiple profiles",
			configContent: `[profile dev]
credential_process = dev-command

[profile prod]
credential_process = prod-command

[profile staging]
credential_process = staging-command`,
			profile:     "prod",
			wantProcess: "prod-command",
			wantErr:     false,
		},
		{
			name: "profile with semicolon comment",
			configContent: `[profile test]
credential_process = my-command ; this is a comment`,
			profile:     "test",
			wantProcess: "my-command",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary config file
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config")
			if err := os.WriteFile(configPath, []byte(tt.configContent), 0644); err != nil {
				t.Fatalf("Failed to write test config: %v", err)
			}

			// Set AWS_CONFIG_FILE to our temp file
			oldConfigFile := os.Getenv("AWS_CONFIG_FILE")
			if err := os.Setenv("AWS_CONFIG_FILE", configPath); err != nil {
				t.Fatalf("Failed to set AWS_CONFIG_FILE: %v", err)
			}
			defer func() {
				if err := os.Setenv("AWS_CONFIG_FILE", oldConfigFile); err != nil {
					t.Errorf("Failed to restore AWS_CONFIG_FILE: %v", err)
				}
			}()

			got, err := ParseAWSConfig(tt.profile)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseAWSConfig() expected error but got none")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("ParseAWSConfig() error = %v, want error containing %q", err, tt.errContains)
				}
				return
			}
			if err != nil {
				t.Errorf("ParseAWSConfig() unexpected error = %v", err)
				return
			}
			if got != tt.wantProcess {
				t.Errorf("ParseAWSConfig() = %q, want %q", got, tt.wantProcess)
			}
		})
	}
}

func TestGetCredentialsFromProcess(t *testing.T) {
	tests := []struct {
		name        string
		command     string
		wantErr     bool
		errContains string
	}{
		{
			name:    "valid command returning credentials",
			command: "echo '{\"Version\": 1, \"AccessKeyId\": \"AKIATEST\", \"SecretAccessKey\": \"secret123\", \"SessionToken\": \"token456\"}'",
			wantErr: false,
		},
		{
			name:    "valid command without session token",
			command: "echo '{\"Version\": 1, \"AccessKeyId\": \"AKIATEST\", \"SecretAccessKey\": \"secret123\"}'",
			wantErr: false,
		},
		{
			name:    "command with quoted arguments",
			command: "echo '{\"Version\": 1, \"AccessKeyId\": \"AKIATEST\", \"SecretAccessKey\": \"secret123\"}'",
			wantErr: false,
		},
		{
			name:        "command that fails",
			command:     "sh -c 'exit 1'",
			wantErr:     true,
			errContains: "credential_process failed",
		},
		{
			name:        "command with invalid JSON",
			command:     "echo 'not json'",
			wantErr:     true,
			errContains: "failed to parse",
		},
		{
			name:        "missing AccessKeyId",
			command:     "echo '{\"Version\": 1, \"SecretAccessKey\": \"secret123\"}'",
			wantErr:     true,
			errContains: "missing required field 'AccessKeyId'",
		},
		{
			name:        "missing SecretAccessKey",
			command:     "echo '{\"Version\": 1, \"AccessKeyId\": \"AKIATEST\"}'",
			wantErr:     true,
			errContains: "missing required field 'SecretAccessKey'",
		},
		{
			name:        "empty string",
			command:     "",
			wantErr:     true,
			errContains: "empty credential_process",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetCredentialsFromProcess(tt.command)
			if tt.wantErr {
				if err == nil {
					t.Errorf("GetCredentialsFromProcess() expected error but got none")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("GetCredentialsFromProcess() error = %v, want error containing %q", err, tt.errContains)
				}
				return
			}
			if err != nil {
				t.Errorf("GetCredentialsFromProcess() unexpected error = %v", err)
				return
			}
			if got == nil {
				t.Errorf("GetCredentialsFromProcess() returned nil credentials")
				return
			}
			if got.AccessKeyID == "" {
				t.Errorf("GetCredentialsFromProcess() missing AccessKeyID")
			}
			if got.SecretAccessKey == "" {
				t.Errorf("GetCredentialsFromProcess() missing SecretAccessKey")
			}
		})
	}
}

func TestGetCredentialsFromProcessTimeout(t *testing.T) {
	// Test that long-running commands timeout
	command := "sleep 60"
	_, err := GetCredentialsFromProcess(command)
	if err == nil {
		t.Error("GetCredentialsFromProcess() expected timeout error but got none")
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Errorf("GetCredentialsFromProcess() error = %v, want timeout error", err)
	}
}

func TestHasStaticCredentials(t *testing.T) {
	tests := []struct {
		name      string
		accessKey string
		secretKey string
		want      bool
	}{
		{
			name:      "both keys present",
			accessKey: "AKIATEST",
			secretKey: "secret123",
			want:      true,
		},
		{
			name:      "only access key",
			accessKey: "AKIATEST",
			secretKey: "",
			want:      false,
		},
		{
			name:      "only secret key",
			accessKey: "",
			secretKey: "secret123",
			want:      false,
		},
		{
			name:      "neither key",
			accessKey: "",
			secretKey: "",
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore original env
			oldAccessKey := os.Getenv("AWS_ACCESS_KEY_ID")
			oldSecretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
			defer func() {
				if err := os.Setenv("AWS_ACCESS_KEY_ID", oldAccessKey); err != nil {
					t.Errorf("Failed to restore AWS_ACCESS_KEY_ID: %v", err)
				}
				if err := os.Setenv("AWS_SECRET_ACCESS_KEY", oldSecretKey); err != nil {
					t.Errorf("Failed to restore AWS_SECRET_ACCESS_KEY: %v", err)
				}
			}()

			// Set test values
			if tt.accessKey != "" {
				if err := os.Setenv("AWS_ACCESS_KEY_ID", tt.accessKey); err != nil {
					t.Fatalf("Failed to set AWS_ACCESS_KEY_ID: %v", err)
				}
			} else {
				if err := os.Unsetenv("AWS_ACCESS_KEY_ID"); err != nil {
					t.Fatalf("Failed to unset AWS_ACCESS_KEY_ID: %v", err)
				}
			}
			if tt.secretKey != "" {
				if err := os.Setenv("AWS_SECRET_ACCESS_KEY", tt.secretKey); err != nil {
					t.Fatalf("Failed to set AWS_SECRET_ACCESS_KEY: %v", err)
				}
			} else {
				if err := os.Unsetenv("AWS_SECRET_ACCESS_KEY"); err != nil {
					t.Fatalf("Failed to unset AWS_SECRET_ACCESS_KEY: %v", err)
				}
			}

			got := HasStaticCredentials()
			if got != tt.want {
				t.Errorf("HasStaticCredentials() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetAWSEnvVars(t *testing.T) {
	// Save original env
	originalEnv := os.Environ()
	defer func() {
		// Restore original environment
		os.Clearenv()
		for _, e := range originalEnv {
			parts := strings.SplitN(e, "=", 2)
			if len(parts) == 2 {
				if err := os.Setenv(parts[0], parts[1]); err != nil {
					t.Errorf("Failed to restore env var %s: %v", parts[0], err)
				}
			}
		}
	}()

	// Set test AWS variables
	testVars := map[string]string{
		"AWS_REGION":         "us-west-2",
		"AWS_PROFILE":        "test",
		"AWS_ACCESS_KEY_ID":  "AKIATEST",
		"AWS_DEFAULT_REGION": "us-east-1",
		// These should be filtered out
		"AWS_CONTAINER_CREDENTIALS_RELATIVE_URI": "http://169.254.170.2/path",
		"AWS_CONTAINER_CREDENTIALS_FULL_URI":     "http://169.254.170.2/full",
		"AWS_CONTAINER_AUTHORIZATION_TOKEN":      "token123",
	}

	for k, v := range testVars {
		if err := os.Setenv(k, v); err != nil {
			t.Fatalf("Failed to set %s: %v", k, err)
		}
	}

	// Also set a non-AWS variable to ensure it's not included
	if err := os.Setenv("NOT_AWS", "value"); err != nil {
		t.Fatalf("Failed to set NOT_AWS: %v", err)
	}

	result := GetAWSEnvVars()

	// Check that normal AWS vars are present
	expectedVars := []string{"AWS_REGION", "AWS_PROFILE", "AWS_ACCESS_KEY_ID", "AWS_DEFAULT_REGION"}
	for _, key := range expectedVars {
		if _, exists := result[key]; !exists {
			t.Errorf("GetAWSEnvVars() missing expected key %q", key)
		}
	}

	// Check that container credential vars are filtered out
	filteredVars := []string{
		"AWS_CONTAINER_CREDENTIALS_RELATIVE_URI",
		"AWS_CONTAINER_CREDENTIALS_FULL_URI",
		"AWS_CONTAINER_AUTHORIZATION_TOKEN",
	}
	for _, key := range filteredVars {
		if _, exists := result[key]; exists {
			t.Errorf("GetAWSEnvVars() should have filtered out %q", key)
		}
	}

	// Check that non-AWS vars are not included
	if _, exists := result["NOT_AWS"]; exists {
		t.Errorf("GetAWSEnvVars() should not include non-AWS variable NOT_AWS")
	}
}

func TestCredentialsValidation(t *testing.T) {
	// Test that credentials are properly validated
	testCases := []struct {
		name        string
		jsonOutput  string
		wantErr     bool
		errContains string
	}{
		{
			name:       "valid minimal credentials",
			jsonOutput: `{"AccessKeyId": "AKIA123", "SecretAccessKey": "secret"}`,
			wantErr:    false,
		},
		{
			name:       "valid with session token",
			jsonOutput: `{"AccessKeyId": "AKIA123", "SecretAccessKey": "secret", "SessionToken": "token"}`,
			wantErr:    false,
		},
		{
			name:        "empty AccessKeyId",
			jsonOutput:  `{"AccessKeyId": "", "SecretAccessKey": "secret"}`,
			wantErr:     true,
			errContains: "AccessKeyId",
		},
		{
			name:        "empty SecretAccessKey",
			jsonOutput:  `{"AccessKeyId": "AKIA123", "SecretAccessKey": ""}`,
			wantErr:     true,
			errContains: "SecretAccessKey",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			command := fmt.Sprintf("echo '%s'", tc.jsonOutput)
			_, err := GetCredentialsFromProcess(command)
			if tc.wantErr {
				if err == nil {
					t.Error("expected error but got none")
				} else if !strings.Contains(err.Error(), tc.errContains) {
					t.Errorf("error = %v, want error containing %q", err, tc.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}
