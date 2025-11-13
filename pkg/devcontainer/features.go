package devcontainer

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// OptionSpec represents a feature option specification
type OptionSpec struct {
	Type        string      `json:"type"`
	Default     interface{} `json:"default,omitempty"`
	Description string      `json:"description,omitempty"`
	Proposals   []string    `json:"proposals,omitempty"`
}

// Mount represents a mount specification from feature metadata
type Mount struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Type   string `json:"type"`
}

// FeatureMetadata represents the metadata from devcontainer-feature.json
// Enhanced to support complete devcontainer-feature.json specification
type FeatureMetadata struct {
	// Required fields per specification
	ID      string `json:"id"`
	Version string `json:"version"`
	Name    string `json:"name"`

	// Optional description
	Description string `json:"description,omitempty"`

	// Options specification
	Options map[string]OptionSpec `json:"options,omitempty"`

	// Container properties that features can contribute
	ContainerEnv map[string]string `json:"containerEnv,omitempty"`
	Privileged   *bool             `json:"privileged,omitempty"`
	Init         *bool             `json:"init,omitempty"`
	CapAdd       []string          `json:"capAdd,omitempty"`
	SecurityOpt  []string          `json:"securityOpt,omitempty"`
	Entrypoint   []string          `json:"entrypoint,omitempty"`
	Mounts       []Mount           `json:"mounts,omitempty"`

	// Lifecycle hooks that features can contribute
	OnCreateCommand      *LifecycleCommand `json:"onCreateCommand,omitempty"`
	UpdateContentCommand *LifecycleCommand `json:"updateContentCommand,omitempty"`
	PostCreateCommand    *LifecycleCommand `json:"postCreateCommand,omitempty"`
	PostStartCommand     *LifecycleCommand `json:"postStartCommand,omitempty"`
	PostAttachCommand    *LifecycleCommand `json:"postAttachCommand,omitempty"`

	// Dependencies
	DependsOn     []string `json:"dependsOn,omitempty"`
	InstallsAfter []string `json:"installsAfter,omitempty"`
}

// ResolvedFeature represents a feature that has been resolved and is ready for installation
type ResolvedFeature struct {
	ID            string
	Version       string
	InstallPath   string
	Options       map[string]interface{}
	Metadata      *FeatureMetadata
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

// isOCIReference checks if a feature reference is an OCI registry reference
func isOCIReference(ref string) bool {
	// OCI references contain : (for version) or start with registry domains
	return strings.Contains(ref, "ghcr.io/") || strings.Contains(ref, "mcr.microsoft.com/")
}

// pullOCIFeature pulls an OCI feature to the cache directory
func (r *FeatureResolver) pullOCIFeature(ociRef string) (string, error) {
	// Create cache directory if it doesn't exist
	if err := os.MkdirAll(r.cacheDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Extract feature name for cache directory
	// e.g., ghcr.io/devcontainers/features/common-utils:2 -> common-utils-2
	parts := strings.Split(ociRef, "/")
	lastPart := parts[len(parts)-1]
	nameVersion := strings.ReplaceAll(lastPart, ":", "-")
	featureCacheDir := filepath.Join(r.cacheDir, "oci-cache", nameVersion)

	// Check if already cached
	if _, err := os.Stat(filepath.Join(featureCacheDir, "install.sh")); err == nil {
		return featureCacheDir, nil
	}

	// Create temporary directory for extraction
	if err := os.MkdirAll(featureCacheDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create feature cache directory: %w", err)
	}

	// Use oras to pull the OCI artifact
	cmd := exec.Command("oras", "pull", "--output", featureCacheDir, ociRef)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to pull OCI feature %s (is 'oras' installed?): %w\nOutput: %s", ociRef, err, string(output))
	}

	// Extract the tarball that oras downloaded
	// Find the .tgz file in the cache directory
	entries, err := os.ReadDir(featureCacheDir)
	if err != nil {
		return "", fmt.Errorf("failed to read cache directory: %w", err)
	}

	var tarballPath string
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".tgz") || strings.HasSuffix(entry.Name(), ".tar.gz") {
			tarballPath = filepath.Join(featureCacheDir, entry.Name())
			break
		}
	}

	if tarballPath == "" {
		return "", fmt.Errorf("no tarball found in cache directory after OCI pull")
	}

	// Extract tarball to the cache directory
	cmd = exec.Command("tar", "-xf", tarballPath, "-C", featureCacheDir)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to extract tarball: %w", err)
	}

	// Remove the tarball after extraction
	_ = os.Remove(tarballPath)

	return featureCacheDir, nil
}

// ResolveFeature resolves a local feature from the given path with the specified options
func (r *FeatureResolver) ResolveFeature(featurePath string, options map[string]interface{}) (*ResolvedFeature, error) {
	// Check if this is an OCI reference
	if isOCIReference(featurePath) {
		cachedPath, err := r.pullOCIFeature(featurePath)
		if err != nil {
			return nil, err
		}
		featurePath = cachedPath
	}
	// Read metadata from devcontainer-feature.json if it exists
	metadataPath := filepath.Join(featurePath, "devcontainer-feature.json")
	metadataBytes, err := os.ReadFile(metadataPath)

	var metadata FeatureMetadata
	if err == nil {
		// Metadata file exists - parse it
		if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
			return nil, fmt.Errorf("failed to parse feature metadata: %w", err)
		}
	} else if os.IsNotExist(err) {
		// Metadata file doesn't exist - use minimal defaults for local features
		// Use the directory name as the feature ID
		metadata = FeatureMetadata{
			ID:      filepath.Base(featurePath),
			Version: "1.0.0",
		}
	} else {
		// Some other error reading the file
		return nil, fmt.Errorf("failed to read feature metadata: %w", err)
	}

	// Create resolved feature
	resolved := &ResolvedFeature{
		ID:            metadata.ID,
		Version:       metadata.Version,
		InstallPath:   featurePath,
		Options:       options,
		Metadata:      &metadata,
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

		var metadata FeatureMetadata
		if err == nil {
			// Metadata file exists - parse it
			if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
				return nil, fmt.Errorf("failed to parse metadata for feature %s: %w", id, err)
			}
		} else if os.IsNotExist(err) {
			// Metadata file doesn't exist - use minimal defaults
			metadata = FeatureMetadata{
				ID:      id,
				Version: "1.0.0",
			}
		} else {
			// Some other error reading the file
			return nil, fmt.Errorf("failed to read metadata for feature %s: %w", id, err)
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

// FeatureOptionsProcessor handles option to environment variable conversion
type FeatureOptionsProcessor struct{}

// NewFeatureOptionsProcessor creates a new options processor
func NewFeatureOptionsProcessor() *FeatureOptionsProcessor {
	return &FeatureOptionsProcessor{}
}

// ValidateAndProcessOptions validates feature options and converts to environment variables
func (p *FeatureOptionsProcessor) ValidateAndProcessOptions(userOptions map[string]interface{}, optionSpecs map[string]OptionSpec) (map[string]string, error) {
	// First validate all user-provided options
	for optionName, userValue := range userOptions {
		spec, exists := optionSpecs[optionName]
		if !exists {
			// Option not in spec - skip validation
			continue
		}

		if err := p.validateOptionValue(optionName, userValue, spec); err != nil {
			return nil, err
		}
	}

	// Then process options (apply defaults, convert to env vars)
	return p.ProcessOptions(userOptions, optionSpecs), nil
}

// validateOptionValue validates a single option value against its spec
func (p *FeatureOptionsProcessor) validateOptionValue(optionName string, value interface{}, spec OptionSpec) error {
	// Validate type
	switch spec.Type {
	case "string":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("option '%s' must be of type string", optionName)
		}
		// Validate enum (proposals)
		if len(spec.Proposals) > 0 {
			strValue := value.(string)
			valid := false
			for _, proposal := range spec.Proposals {
				if strValue == proposal {
					valid = true
					break
				}
			}
			if !valid {
				return fmt.Errorf("option '%s' value '%s' must be one of: %v", optionName, strValue, spec.Proposals)
			}
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("option '%s' must be of type boolean", optionName)
		}
	case "number":
		// Accept int, int64, float64
		switch value.(type) {
		case int, int64, float64:
			// Valid number types
		default:
			return fmt.Errorf("option '%s' must be of type number", optionName)
		}
	}

	return nil
}

// ProcessOptions converts feature options to environment variables per specification
func (p *FeatureOptionsProcessor) ProcessOptions(userOptions map[string]interface{}, optionSpecs map[string]OptionSpec) map[string]string {
	result := make(map[string]string)

	// Process all option specs (apply defaults, then user overrides)
	for optionName, spec := range optionSpecs {
		envName := normalizeOptionName(optionName)

		// Start with default value
		value := spec.Default

		// Override with user value if provided
		if userValue, exists := userOptions[optionName]; exists {
			value = userValue
		}

		// Convert to string
		if value != nil {
			result[envName] = fmt.Sprintf("%v", value)
		}
	}

	return result
}

// normalizeOptionName converts option name to environment variable per specification
func normalizeOptionName(name string) string {
	// Per spec: replace non-word chars with underscore, prefix digits with underscore, uppercase
	re := regexp.MustCompile(`[^\w_]`)
	normalized := re.ReplaceAllString(name, "_")

	re2 := regexp.MustCompile(`^[\d]+`)
	if re2.MatchString(normalized) {
		normalized = "_" + normalized
	}

	return strings.ToUpper(normalized)
}
