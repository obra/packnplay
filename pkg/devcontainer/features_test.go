package devcontainer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestResolveLocalFeature(t *testing.T) {
	// Create temp directory for test feature
	tmpDir := t.TempDir()
	featurePath := filepath.Join(tmpDir, "test-feature")
	if err := os.MkdirAll(featurePath, 0755); err != nil {
		t.Fatalf("Failed to create feature directory: %v", err)
	}

	// Create devcontainer-feature.json
	metadata := map[string]interface{}{
		"id":          "test-feature",
		"version":     "1.0.0",
		"name":        "Test Feature",
		"description": "A test feature for unit testing",
	}
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		t.Fatalf("Failed to marshal metadata: %v", err)
	}
	metadataFile := filepath.Join(featurePath, "devcontainer-feature.json")
	if err := os.WriteFile(metadataFile, metadataJSON, 0644); err != nil {
		t.Fatalf("Failed to write metadata file: %v", err)
	}

	// Create install.sh
	installScript := "#!/bin/bash\necho 'Installing test feature'\n"
	installFile := filepath.Join(featurePath, "install.sh")
	if err := os.WriteFile(installFile, []byte(installScript), 0755); err != nil {
		t.Fatalf("Failed to write install script: %v", err)
	}

	// Create cache directory
	cacheDir := filepath.Join(tmpDir, "cache")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		t.Fatalf("Failed to create cache directory: %v", err)
	}

	// Create resolver and resolve feature
	resolver := NewFeatureResolver(cacheDir)
	options := map[string]interface{}{
		"someOption": "someValue",
	}
	resolved, err := resolver.ResolveFeature(featurePath, options)
	if err != nil {
		t.Fatalf("Failed to resolve feature: %v", err)
	}

	// Verify resolved feature
	if resolved.ID != "test-feature" {
		t.Errorf("Expected ID 'test-feature', got '%s'", resolved.ID)
	}
	if resolved.Version != "1.0.0" {
		t.Errorf("Expected Version '1.0.0', got '%s'", resolved.Version)
	}
	if resolved.InstallPath != featurePath {
		t.Errorf("Expected InstallPath '%s', got '%s'", featurePath, resolved.InstallPath)
	}
	if resolved.Options == nil {
		t.Error("Expected Options to be set, got nil")
	}
	if val, ok := resolved.Options["someOption"]; !ok || val != "someValue" {
		t.Errorf("Expected option 'someOption' with value 'someValue', got %v", resolved.Options)
	}
}
