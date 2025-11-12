package devcontainer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// FeatureMetadata represents the metadata from devcontainer-feature.json
type FeatureMetadata struct {
	ID            string   `json:"id"`
	Version       string   `json:"version"`
	Name          string   `json:"name"`
	Description   string   `json:"description"`
	DependsOn     []string `json:"dependsOn,omitempty"`
	InstallsAfter []string `json:"installsAfter,omitempty"`
}

// ResolvedFeature represents a feature that has been resolved and is ready for installation
type ResolvedFeature struct {
	ID            string
	Version       string
	InstallPath   string
	Options       map[string]interface{}
	DependsOn     []string
	InstallsAfter []string
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
		ID:            metadata.ID,
		Version:       metadata.Version,
		InstallPath:   featurePath,
		Options:       options,
		DependsOn:     metadata.DependsOn,
		InstallsAfter: metadata.InstallsAfter,
	}

	return resolved, nil
}

// ResolveFeatures resolves feature dependencies and returns features in installation order
func (r *FeatureResolver) ResolveFeatures(features map[string]*ResolvedFeature) ([]*ResolvedFeature, error) {
	// Load metadata for all features
	featureMetadata := make(map[string]*FeatureMetadata)
	for id, feature := range features {
		metadataPath := filepath.Join(feature.InstallPath, "devcontainer-feature.json")
		metadataBytes, err := os.ReadFile(metadataPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read metadata for feature %s: %w", id, err)
		}
		var metadata FeatureMetadata
		if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
			return nil, fmt.Errorf("failed to parse metadata for feature %s: %w", id, err)
		}
		featureMetadata[id] = &metadata
		// Update the feature with dependency info
		feature.DependsOn = metadata.DependsOn
		feature.InstallsAfter = metadata.InstallsAfter
	}

	// Round-based resolution algorithm
	var result []*ResolvedFeature
	installed := make(map[string]bool)
	remaining := make(map[string]*ResolvedFeature)
	for id, feature := range features {
		remaining[id] = feature
	}

	for len(remaining) > 0 {
		var roundInstalls []*ResolvedFeature

		// Try to find features that can be installed in this round
		for _, feature := range remaining {
			// Check if all hard dependencies (dependsOn) are satisfied
			canInstall := true
			for _, depID := range feature.DependsOn {
				if !installed[depID] {
					canInstall = false
					break
				}
			}

			// Check if all soft dependencies (installsAfter) are satisfied or not in the set
			if canInstall {
				for _, afterID := range feature.InstallsAfter {
					// Only block if the feature exists in our set and isn't installed yet
					if _, exists := features[afterID]; exists && !installed[afterID] {
						canInstall = false
						break
					}
				}
			}

			if canInstall {
				roundInstalls = append(roundInstalls, feature)
			}
		}

		// If no features can be installed, we have an error
		if len(roundInstalls) == 0 {
			// Build list of remaining features for error message
			var remainingIDs []string
			for id := range remaining {
				remainingIDs = append(remainingIDs, id)
			}
			return nil, fmt.Errorf("cannot resolve dependencies: features %v have unsatisfied dependencies", remainingIDs)
		}

		// Install this round's features
		for _, feature := range roundInstalls {
			result = append(result, feature)
			installed[feature.ID] = true
			delete(remaining, feature.ID)
		}
	}

	return result, nil
}
