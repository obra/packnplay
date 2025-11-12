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

func TestResolveDependencies(t *testing.T) {
	// Create temp directory for test features
	tmpDir := t.TempDir()
	cacheDir := filepath.Join(tmpDir, "cache")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		t.Fatalf("Failed to create cache directory: %v", err)
	}

	// Create feature B (no dependencies)
	featureBPath := filepath.Join(tmpDir, "feature-b")
	if err := os.MkdirAll(featureBPath, 0755); err != nil {
		t.Fatalf("Failed to create feature B directory: %v", err)
	}
	metadataB := map[string]interface{}{
		"id":      "feature-b",
		"version": "1.0.0",
		"name":    "Feature B",
	}
	metadataBJSON, _ := json.Marshal(metadataB)
	_ = os.WriteFile(filepath.Join(featureBPath, "devcontainer-feature.json"), metadataBJSON, 0644)
	_ = os.WriteFile(filepath.Join(featureBPath, "install.sh"), []byte("#!/bin/bash\necho 'B'\n"), 0755)

	// Create feature A (depends on feature-b)
	featureAPath := filepath.Join(tmpDir, "feature-a")
	if err := os.MkdirAll(featureAPath, 0755); err != nil {
		t.Fatalf("Failed to create feature A directory: %v", err)
	}
	metadataA := map[string]interface{}{
		"id":        "feature-a",
		"version":   "1.0.0",
		"name":      "Feature A",
		"dependsOn": []string{"feature-b"},
	}
	metadataAJSON, _ := json.Marshal(metadataA)
	_ = os.WriteFile(filepath.Join(featureAPath, "devcontainer-feature.json"), metadataAJSON, 0644)
	_ = os.WriteFile(filepath.Join(featureAPath, "install.sh"), []byte("#!/bin/bash\necho 'A'\n"), 0755)

	// Create feature C (installs after feature-a)
	featureCPath := filepath.Join(tmpDir, "feature-c")
	if err := os.MkdirAll(featureCPath, 0755); err != nil {
		t.Fatalf("Failed to create feature C directory: %v", err)
	}
	metadataC := map[string]interface{}{
		"id":            "feature-c",
		"version":       "1.0.0",
		"name":          "Feature C",
		"installsAfter": []string{"feature-a"},
	}
	metadataCJSON, _ := json.Marshal(metadataC)
	_ = os.WriteFile(filepath.Join(featureCPath, "devcontainer-feature.json"), metadataCJSON, 0644)
	_ = os.WriteFile(filepath.Join(featureCPath, "install.sh"), []byte("#!/bin/bash\necho 'C'\n"), 0755)

	// Create resolver and resolve features
	resolver := NewFeatureResolver(cacheDir)
	features := map[string]*ResolvedFeature{
		"feature-a": {ID: "feature-a", InstallPath: featureAPath},
		"feature-b": {ID: "feature-b", InstallPath: featureBPath},
		"feature-c": {ID: "feature-c", InstallPath: featureCPath},
	}

	// Call ResolveFeatures
	ordered, err := resolver.ResolveFeatures(features)
	if err != nil {
		t.Fatalf("Failed to resolve features: %v", err)
	}

	// Assert order is [feature-b, feature-a, feature-c]
	expectedOrder := []string{"feature-b", "feature-a", "feature-c"}
	if len(ordered) != len(expectedOrder) {
		t.Fatalf("Expected %d features, got %d", len(expectedOrder), len(ordered))
	}

	for i, expected := range expectedOrder {
		if ordered[i].ID != expected {
			t.Errorf("Expected feature %d to be '%s', got '%s'", i, expected, ordered[i].ID)
		}
	}
}
