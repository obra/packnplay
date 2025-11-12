package runner

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/obra/packnplay/pkg/config"
	"github.com/obra/packnplay/pkg/devcontainer"
	"github.com/stretchr/testify/assert"
)

func TestGetOrCreateContainerCredentialFile(t *testing.T) {
	// Use temp directory for test
	tempDir := t.TempDir()
	if err := os.Setenv("XDG_DATA_HOME", tempDir); err != nil {
		t.Fatalf("Failed to set XDG_DATA_HOME: %v", err)
	}
	defer func() {
		if err := os.Unsetenv("XDG_DATA_HOME"); err != nil {
			t.Errorf("Failed to unset XDG_DATA_HOME: %v", err)
		}
	}()

	// Test file creation
	credFile, err := getOrCreateContainerCredentialFile("test-container")
	if err != nil {
		t.Fatalf("getOrCreateContainerCredentialFile() error = %v", err)
	}

	// Verify file exists
	if !fileExists(credFile) {
		t.Errorf("Credential file not created at %s", credFile)
	}

	// Verify file path format
	expectedDir := filepath.Join(tempDir, "packnplay", "credentials")
	expectedFile := filepath.Join(expectedDir, "claude-credentials.json")

	if credFile != expectedFile {
		t.Errorf("Credential file path = %v, want %v", credFile, expectedFile)
	}

	// Verify file permissions
	stat, err := os.Stat(credFile)
	if err != nil {
		t.Fatalf("Failed to stat credential file: %v", err)
	}

	if stat.Mode().Perm() != 0600 {
		t.Errorf("Credential file permissions = %v, want 0600", stat.Mode().Perm())
	}

	// Test second call returns same file
	credFile2, err := getOrCreateContainerCredentialFile("another-container")
	if err != nil {
		t.Fatalf("Second getOrCreateContainerCredentialFile() error = %v", err)
	}

	if credFile != credFile2 {
		t.Errorf("Second call returned different file: %v != %v", credFile, credFile2)
	}
}

func TestGetInitialContainerCredentials(t *testing.T) {
	// Test when no initial credentials available
	_, err := getInitialContainerCredentials()
	if err == nil {
		t.Skip("getInitialContainerCredentials() might find credentials on this system - skipping")
	}
}

func TestGetFileSize(t *testing.T) {
	// Create test file
	tempFile := filepath.Join(t.TempDir(), "test.txt")
	content := "test content"
	err := os.WriteFile(tempFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	size := getFileSize(tempFile)
	expectedSize := int64(len(content))

	if size != expectedSize {
		t.Errorf("getFileSize() = %v, want %v", size, expectedSize)
	}

	// Test non-existent file
	nonExistentSize := getFileSize("/non/existent/file")
	if nonExistentSize != 0 {
		t.Errorf("getFileSize() for non-existent file = %v, want 0", nonExistentSize)
	}
}

func TestRunConfig(t *testing.T) {
	// Test RunConfig struct fields
	cfg := &RunConfig{
		Path:           "/test/path",
		Worktree:       "feature-branch",
		NoWorktree:     false,
		Env:            []string{"TEST=value"},
		Verbose:        true,
		Runtime:        "docker",
		Command:        []string{"claude", "test"},
		DefaultEnvVars: []string{"ANTHROPIC_API_KEY"},
		Credentials: config.Credentials{
			Git: true,
			SSH: false,
		},
	}

	// Verify all fields are accessible
	if cfg.Path != "/test/path" {
		t.Errorf("RunConfig.Path = %v, want /test/path", cfg.Path)
	}

	if cfg.Worktree != "feature-branch" {
		t.Errorf("RunConfig.Worktree = %v, want feature-branch", cfg.Worktree)
	}

	if len(cfg.DefaultEnvVars) != 1 || cfg.DefaultEnvVars[0] != "ANTHROPIC_API_KEY" {
		t.Errorf("RunConfig.DefaultEnvVars = %v, want [ANTHROPIC_API_KEY]", cfg.DefaultEnvVars)
	}
}

func TestApplyFeatureContainerProperties(t *testing.T) {
	// Test that features can contribute security options, capabilities, etc.
	privilegedTrue := true
	features := []*devcontainer.ResolvedFeature{
		{
			ID: "docker-feature",
			Metadata: &devcontainer.FeatureMetadata{
				Privileged:  &privilegedTrue,
				CapAdd:      []string{"NET_ADMIN", "SYS_PTRACE"},
				SecurityOpt: []string{"apparmor=unconfined"},
				ContainerEnv: map[string]string{
					"FEATURE_VAR": "feature-value",
				},
			},
		},
	}

	applier := NewFeaturePropertiesApplier()
	dockerArgs := []string{"run", "-d", "--name", "test"}

	enhancedArgs, enhancedEnv := applier.ApplyFeatureProperties(dockerArgs, features, map[string]string{})

	// Verify security properties added
	assert.Contains(t, enhancedArgs, "--privileged")
	assert.Contains(t, enhancedArgs, "--cap-add=NET_ADMIN")
	assert.Contains(t, enhancedArgs, "--cap-add=SYS_PTRACE")
	assert.Contains(t, enhancedArgs, "--security-opt=apparmor=unconfined")

	// Verify environment variables added
	assert.Equal(t, "feature-value", enhancedEnv["FEATURE_VAR"])
}
