package devcontainer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// FeatureMetadata represents the metadata from devcontainer-feature.json
type FeatureMetadata struct {
	ID          string `json:"id"`
	Version     string `json:"version"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// ResolvedFeature represents a feature that has been resolved and is ready for installation
type ResolvedFeature struct {
	ID          string
	Version     string
	InstallPath string
	Options     map[string]interface{}
}

// FeatureResolver handles resolving features from various sources
type FeatureResolver struct {
	cacheDir string
}

// NewFeatureResolver creates a new FeatureResolver with the specified cache directory
func NewFeatureResolver(cacheDir string) *FeatureResolver {
	return &FeatureResolver{
		cacheDir: cacheDir,
	}
}

// ResolveFeature resolves a local feature from the given path with the specified options
func (r *FeatureResolver) ResolveFeature(featurePath string, options map[string]interface{}) (*ResolvedFeature, error) {
	// Read metadata from devcontainer-feature.json
	metadataPath := filepath.Join(featurePath, "devcontainer-feature.json")
	metadataBytes, err := os.ReadFile(metadataPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read feature metadata: %w", err)
	}

	var metadata FeatureMetadata
	if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
		return nil, fmt.Errorf("failed to parse feature metadata: %w", err)
	}

	// Create resolved feature
	resolved := &ResolvedFeature{
		ID:          metadata.ID,
		Version:     metadata.Version,
		InstallPath: featurePath,
		Options:     options,
	}

	return resolved, nil
}
