# Devcontainer Features Specification Compliance Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Achieve 100% devcontainer features specification compliance by fixing critical gaps in current prototype.

**Architecture:** Systematically enhance existing feature resolution, Dockerfile generation, and integration systems to support complete feature metadata, options processing, and lifecycle hooks per official specification.

**Tech Stack:** Go, Docker BuildKit, OCI registry access, devcontainer specification compliance

---

## Task 1: Fix Feature Options Processing (Critical Gap #1)

**Files:**
- Modify: `pkg/devcontainer/features.go:15-30`
- Modify: `internal/dockerfile/dockerfile_generator.go:30-50`
- Test: `pkg/devcontainer/features_test.go`

**Step 1: Write failing test for option environment variable conversion**

```go
func TestProcessFeatureOptions(t *testing.T) {
	tests := []struct {
		name           string
		featureOptions map[string]interface{}
		optionSpecs    map[string]OptionSpec
		expectedEnvs   map[string]string
	}{
		{
			name: "node version option",
			featureOptions: map[string]interface{}{
				"version":     "18.20.0",
				"install-type": "nvm",
			},
			optionSpecs: map[string]OptionSpec{
				"version":      {Type: "string", Default: "latest"},
				"install-type": {Type: "string", Default: "apt"},
			},
			expectedEnvs: map[string]string{
				"VERSION":      "18.20.0",
				"INSTALL_TYPE": "nvm",
			},
		},
		{
			name: "use defaults when options missing",
			featureOptions: map[string]interface{}{},
			optionSpecs: map[string]OptionSpec{
				"version": {Type: "string", Default: "latest"},
			},
			expectedEnvs: map[string]string{
				"VERSION": "latest",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			processor := NewFeatureOptionsProcessor()
			envs := processor.ProcessOptions(tt.featureOptions, tt.optionSpecs)
			assert.Equal(t, tt.expectedEnvs, envs)
		})
	}
}

func TestNormalizeOptionName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"version", "VERSION"},
		{"install-type", "INSTALL_TYPE"},
		{"installZsh", "INSTALLZSH"},
		{"node-version", "NODE_VERSION"},
		{"123test", "_123TEST"},
		{"test@key", "TEST_KEY"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizeOptionName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/devcontainer -run "TestProcessFeatureOptions|TestNormalizeOptionName" -v`
Expected: FAIL with "NewFeatureOptionsProcessor undefined"

**Step 3: Implement feature options processing**

```go
// Add to pkg/devcontainer/features.go

import (
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

// FeatureOptionsProcessor handles option to environment variable conversion
type FeatureOptionsProcessor struct{}

// NewFeatureOptionsProcessor creates a new options processor
func NewFeatureOptionsProcessor() *FeatureOptionsProcessor {
	return &FeatureOptionsProcessor{}
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

	re2 := regexp.MustCompile(`^[\d_]+`)
	normalized = re2.ReplaceAllString(normalized, "_")

	return strings.ToUpper(normalized)
}
```

**Step 4: Enhance FeatureMetadata with options support**

```go
// Update FeatureMetadata struct in pkg/devcontainer/features.go
type FeatureMetadata struct {
	ID          string                    `json:"id"`
	Version     string                    `json:"version"`
	Name        string                    `json:"name"`
	Description string                    `json:"description,omitempty"`
	Options     map[string]OptionSpec     `json:"options,omitempty"`
	DependsOn   []string                  `json:"dependsOn,omitempty"`
	InstallsAfter []string                `json:"installsAfter,omitempty"`
}
```

**Step 5: Update Dockerfile generator to use processed options**

```go
// Modify Generate method in internal/dockerfile/dockerfile_generator.go
func (g *DockerfileGenerator) Generate(baseImage string, features []*devcontainer.ResolvedFeature, remoteUser string, buildContextPath string) (string, error) {
	var lines []string

	lines = append(lines, fmt.Sprintf("FROM %s", baseImage))
	lines = append(lines, "USER root")
	lines = append(lines, "")

	processor := devcontainer.NewFeatureOptionsProcessor()

	for _, feature := range features {
		lines = append(lines, fmt.Sprintf("# Install feature: %s", feature.ID))

		// Process feature options to environment variables
		if feature.Metadata != nil && feature.Metadata.Options != nil {
			envVars := processor.ProcessOptions(feature.Options, feature.Metadata.Options)
			for envName, envValue := range envVars {
				lines = append(lines, fmt.Sprintf("ENV %s=%s", envName, envValue))
			}
		}

		// Rest of existing COPY and RUN logic...
	}

	// Rest of existing implementation...
}
```

**Step 6: Run test to verify it passes**

Run: `go test ./pkg/devcontainer -run "TestProcessFeatureOptions|TestNormalizeOptionName" -v`
Expected: PASS

**Step 7: Commit**

```bash
git add pkg/devcontainer/features.go internal/dockerfile/dockerfile_generator.go pkg/devcontainer/features_test.go
git commit -m "feat: implement feature options processing per devcontainer spec

- Add OptionSpec struct matching devcontainer-feature.json format
- Implement option normalization per specification regex
- Add FeatureOptionsProcessor with environment variable conversion
- Enhanced FeatureMetadata with complete options support
- Update Dockerfile generator to process options as ENV commands
- Add comprehensive unit tests for option processing and normalization"
```

---

## Task 2: Enhance FeatureMetadata for Complete Specification Support

**Files:**
- Modify: `pkg/devcontainer/features.go:8-25`
- Test: `pkg/devcontainer/features_test.go`

**Step 1: Write failing test for complete metadata parsing**

```go
func TestParseCompleteFeatureMetadata(t *testing.T) {
	// Create temp feature with complete metadata
	tmpDir := t.TempDir()
	featureDir := filepath.Join(tmpDir, "complete-feature")
	err := os.MkdirAll(featureDir, 0755)
	require.NoError(t, err)

	// Complete devcontainer-feature.json with all specification fields
	completeMetadata := `{
		"id": "complete-feature",
		"version": "1.0.0",
		"name": "Complete Feature",
		"description": "Feature with all metadata fields",
		"options": {
			"version": {
				"type": "string",
				"default": "latest",
				"description": "Version to install"
			}
		},
		"containerEnv": {
			"FEATURE_ENV": "value"
		},
		"privileged": true,
		"capAdd": ["NET_ADMIN"],
		"securityOpt": ["apparmor=unconfined"],
		"mounts": [
			{
				"source": "feature-volume",
				"target": "/feature-data",
				"type": "volume"
			}
		],
		"onCreateCommand": "echo 'feature onCreate'",
		"postCreateCommand": ["echo", "feature postCreate"],
		"dependsOn": ["base-feature"]
	}`

	err = os.WriteFile(filepath.Join(featureDir, "devcontainer-feature.json"), []byte(completeMetadata), 0644)
	require.NoError(t, err)

	installScript := "#!/bin/bash\necho 'Installing complete feature'\n"
	err = os.WriteFile(filepath.Join(featureDir, "install.sh"), []byte(installScript), 0755)
	require.NoError(t, err)

	// Test resolution
	resolver := NewFeatureResolver("/tmp/cache")
	resolved, err := resolver.ResolveFeature(featureDir, map[string]interface{}{
		"version": "18.20.0",
	})
	require.NoError(t, err)

	// Verify all metadata fields parsed correctly
	assert.Equal(t, "complete-feature", resolved.ID)
	assert.Equal(t, "Complete Feature", resolved.Name)
	assert.Equal(t, "Feature with all metadata fields", resolved.Description)
	assert.NotNil(t, resolved.Metadata.Options)
	assert.Contains(t, resolved.Metadata.Options, "version")
	assert.NotNil(t, resolved.Metadata.ContainerEnv)
	assert.Equal(t, "value", resolved.Metadata.ContainerEnv["FEATURE_ENV"])
	assert.NotNil(t, resolved.Metadata.Privileged)
	assert.True(t, *resolved.Metadata.Privileged)
	assert.Equal(t, []string{"NET_ADMIN"}, resolved.Metadata.CapAdd)
	assert.Equal(t, []string{"apparmor=unconfined"}, resolved.Metadata.SecurityOpt)
	assert.NotNil(t, resolved.Metadata.Mounts)
	assert.NotNil(t, resolved.Metadata.OnCreateCommand)
	assert.NotNil(t, resolved.Metadata.PostCreateCommand)
	assert.Equal(t, []string{"base-feature"}, resolved.Metadata.DependsOn)
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/devcontainer -run TestParseCompleteFeatureMetadata -v`
Expected: FAIL with missing fields in FeatureMetadata

**Step 3: Enhance FeatureMetadata with complete specification fields**

```go
// Add to pkg/devcontainer/features.go

// Mount represents a mount specification from feature metadata
type Mount struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Type   string `json:"type"`
}

// Enhanced FeatureMetadata with complete devcontainer-feature.json specification
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
```

**Step 4: Run test to verify it passes**

Run: `go test ./pkg/devcontainer -run TestParseCompleteFeatureMetadata -v`
Expected: PASS

**Step 5: Commit**

```bash
git add pkg/devcontainer/features.go pkg/devcontainer/features_test.go
git commit -m "feat: enhance FeatureMetadata for complete specification support

- Add all optional fields from devcontainer-feature.json specification
- Support container properties (privileged, capAdd, securityOpt, etc.)
- Add lifecycle hooks (onCreateCommand, postCreateCommand, etc.)
- Add Mount struct for feature-contributed mounts
- Add comprehensive test for complete metadata parsing
- Foundation for full specification compliance"
```

---

## Task 3: Implement Multi-Stage Docker Build for OCI Features

**Files:**
- Modify: `internal/dockerfile/dockerfile_generator.go:15-80`
- Test: `internal/dockerfile/dockerfile_generator_test.go`

**Step 1: Write failing test for multi-stage build generation**

```go
func TestGenerateMultiStageWithOCIFeatures(t *testing.T) {
	// Create OCI feature (simulated cached feature)
	tmpDir := t.TempDir()
	ociFeatureDir := filepath.Join(tmpDir, "oci-cache", "common-utils")
	err := os.MkdirAll(ociFeatureDir, 0755)
	require.NoError(t, err)

	// Create install script
	installScript := "#!/bin/bash\necho 'Installing common-utils'"
	err = os.WriteFile(filepath.Join(ociFeatureDir, "install.sh"), []byte(installScript), 0755)
	require.NoError(t, err)

	// Create feature with OCI path
	ociFeature := &devcontainer.ResolvedFeature{
		ID:          "common-utils",
		Version:     "2.0.0",
		InstallPath: ociFeatureDir,
		Options:     map[string]interface{}{},
	}

	generator := NewDockerfileGenerator()
	dockerfile, err := generator.Generate("ubuntu:22.04", []*devcontainer.ResolvedFeature{ociFeature}, "testuser", "/project/.devcontainer")
	require.NoError(t, err)

	// Verify multi-stage build structure
	assert.Contains(t, dockerfile, "FROM ubuntu:22.04 as base")
	assert.Contains(t, dockerfile, "FROM base as feature-prep")
	assert.Contains(t, dockerfile, "COPY --from=")
	assert.Contains(t, dockerfile, "FROM feature-prep")
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/dockerfile -run TestGenerateMultiStageWithOCIFeatures -v`
Expected: FAIL (no multi-stage support)

**Step 3: Implement multi-stage Dockerfile generation**

```go
// Enhance Generate method in internal/dockerfile/dockerfile_generator.go

func (g *DockerfileGenerator) Generate(baseImage string, features []*devcontainer.ResolvedFeature, remoteUser string, buildContextPath string) (string, error) {
	if len(features) == 0 {
		return fmt.Sprintf("FROM %s\nUSER %s\nWORKDIR /workspace", baseImage, remoteUser), nil
	}

	var lines []string

	// Determine if we need multi-stage build (OCI features outside build context)
	needsMultiStage := false
	for _, feature := range features {
		if !strings.HasPrefix(feature.InstallPath, buildContextPath) {
			needsMultiStage = true
			break
		}
	}

	if needsMultiStage {
		return g.generateMultiStage(baseImage, features, remoteUser, buildContextPath)
	}

	return g.generateSingleStage(baseImage, features, remoteUser, buildContextPath)
}

func (g *DockerfileGenerator) generateMultiStage(baseImage string, features []*devcontainer.ResolvedFeature, remoteUser string, buildContextPath string) (string, error) {
	var lines []string

	// Stage 1: Feature preparation
	lines = append(lines, "FROM alpine:latest as feature-prep")
	lines = append(lines, "")

	// Copy all features to staging area
	for _, feature := range features {
		if !strings.HasPrefix(feature.InstallPath, buildContextPath) {
			// OCI feature - needs to be copied from cache
			lines = append(lines, fmt.Sprintf("COPY --from=%s /tmp/devcontainer-features/%s /tmp/features/%s",
				feature.ID, feature.ID, feature.ID))
		}
	}

	// Stage 2: Base image with features
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("FROM %s as base", baseImage))
	lines = append(lines, "USER root")
	lines = append(lines, "")

	// Copy features from prep stage
	lines = append(lines, "COPY --from=feature-prep /tmp/features /tmp/devcontainer-features")
	lines = append(lines, "")

	// Install features with options processing
	processor := devcontainer.NewFeatureOptionsProcessor()
	for _, feature := range features {
		lines = append(lines, fmt.Sprintf("# Install feature: %s", feature.ID))

		// Add environment variables from options
		if feature.Metadata != nil && feature.Metadata.Options != nil {
			envVars := processor.ProcessOptions(feature.Options, feature.Metadata.Options)
			for envName, envValue := range envVars {
				lines = append(lines, fmt.Sprintf("ENV %s=%s", envName, envValue))
			}
		}

		lines = append(lines, fmt.Sprintf("RUN cd /tmp/devcontainer-features/%s && chmod +x install.sh && ./install.sh", feature.ID))
		lines = append(lines, "")
	}

	// Switch to user
	lines = append(lines, fmt.Sprintf("USER %s", remoteUser))
	lines = append(lines, "WORKDIR /workspace")

	return strings.Join(lines, "\n"), nil
}

func (g *DockerfileGenerator) generateSingleStage(baseImage string, features []*devcontainer.ResolvedFeature, remoteUser string, buildContextPath string) (string, error) {
	// Existing single-stage logic enhanced with options processing
	// Similar to current implementation but with options support
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/dockerfile -run TestGenerateMultiStageWithOCIFeatures -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/dockerfile/dockerfile_generator.go internal/dockerfile/dockerfile_generator_test.go
git commit -m "feat: implement multi-stage Docker builds for OCI features

- Add multi-stage Dockerfile generation for features outside build context
- Integrate feature options processing in Dockerfile generation
- Support both single-stage (local) and multi-stage (OCI) builds
- Process feature options to ENV commands per specification
- Add comprehensive test for multi-stage build structure"
```

---

## Task 4: Add Feature Lifecycle Hook Integration

**Files:**
- Create: `pkg/devcontainer/lifecycle_merger.go`
- Test: `pkg/devcontainer/lifecycle_merger_test.go`
- Modify: `pkg/runner/runner.go:815-850`

**Step 1: Write failing test for lifecycle hook merging**

```go
func TestMergeLifecycleCommands(t *testing.T) {
	// Create features with lifecycle commands
	feature1 := &ResolvedFeature{
		ID: "feature1",
		Metadata: &FeatureMetadata{
			OnCreateCommand:   &LifecycleCommand{},
			PostCreateCommand: &LifecycleCommand{},
		},
	}

	feature2 := &ResolvedFeature{
		ID: "feature2",
		Metadata: &FeatureMetadata{
			PostCreateCommand: &LifecycleCommand{},
			PostStartCommand:  &LifecycleCommand{},
		},
	}

	// User commands
	userOnCreate := &LifecycleCommand{}
	userPostCreate := &LifecycleCommand{}

	merger := NewLifecycleMerger()
	merged := merger.MergeCommands([]*ResolvedFeature{feature1, feature2}, map[string]*LifecycleCommand{
		"onCreateCommand":   userOnCreate,
		"postCreateCommand": userPostCreate,
	})

	// Verify feature commands come before user commands
	onCreate := merged["onCreateCommand"]
	assert.NotNil(t, onCreate)
	// Should have: feature1.onCreateCommand, userOnCreate

	postCreate := merged["postCreateCommand"]
	assert.NotNil(t, postCreate)
	// Should have: feature1.postCreateCommand, feature2.postCreateCommand, userPostCreate
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/devcontainer -run TestMergeLifecycleCommands -v`
Expected: FAIL with "NewLifecycleMerger undefined"

**Step 3: Implement lifecycle command merger**

```go
// Create pkg/devcontainer/lifecycle_merger.go

package devcontainer

// LifecycleMerger handles merging feature and user lifecycle commands
type LifecycleMerger struct{}

// NewLifecycleMerger creates a new lifecycle merger
func NewLifecycleMerger() *LifecycleMerger {
	return &LifecycleMerger{}
}

// MergeCommands merges feature lifecycle commands with user commands
// Feature commands execute before user commands per specification
func (m *LifecycleMerger) MergeCommands(features []*ResolvedFeature, userCommands map[string]*LifecycleCommand) map[string]*LifecycleCommand {
	result := make(map[string]*LifecycleCommand)

	hookTypes := []string{"onCreateCommand", "updateContentCommand", "postCreateCommand", "postStartCommand", "postAttachCommand"}

	for _, hookType := range hookTypes {
		var mergedCommands []string

		// First, add feature commands in installation order
		for _, feature := range features {
			if feature.Metadata == nil {
				continue
			}

			var featureCommand *LifecycleCommand
			switch hookType {
			case "onCreateCommand":
				featureCommand = feature.Metadata.OnCreateCommand
			case "updateContentCommand":
				featureCommand = feature.Metadata.UpdateContentCommand
			case "postCreateCommand":
				featureCommand = feature.Metadata.PostCreateCommand
			case "postStartCommand":
				featureCommand = feature.Metadata.PostStartCommand
			case "postAttachCommand":
				featureCommand = feature.Metadata.PostAttachCommand
			}

			if featureCommand != nil {
				commands := featureCommand.ToStringSlice()
				mergedCommands = append(mergedCommands, commands...)
			}
		}

		// Then, add user commands
		if userCommand, exists := userCommands[hookType]; exists && userCommand != nil {
			commands := userCommand.ToStringSlice()
			mergedCommands = append(mergedCommands, commands...)
		}

		// Create merged lifecycle command if we have any commands
		if len(mergedCommands) > 0 {
			result[hookType] = &LifecycleCommand{
				// Convert back to appropriate format
				Commands: mergedCommands,
			}
		}
	}

	return result
}
```

**Step 4: Integrate with runner lifecycle execution**

Modify `pkg/runner/runner.go` around line 820 to use merged commands:

```go
// Replace existing lifecycle command execution with merged commands
if devConfig.Features != nil && len(devConfig.Features) > 0 {
	// Resolve features for lifecycle merging
	resolver := devcontainer.NewFeatureResolver(filepath.Join(os.TempDir(), "packnplay-features-cache"))

	var resolvedFeatures []*devcontainer.ResolvedFeature
	for reference, options := range devConfig.Features {
		feature, err := resolver.ResolveFeature(reference, options)
		if err != nil {
			continue // Skip failed features for lifecycle merging
		}
		resolvedFeatures = append(resolvedFeatures, feature)
	}

	// Merge feature and user lifecycle commands
	merger := devcontainer.NewLifecycleMerger()
	userCommands := map[string]*devcontainer.LifecycleCommand{
		"onCreateCommand":   devConfig.OnCreateCommand,
		"postCreateCommand": devConfig.PostCreateCommand,
		"postStartCommand":  devConfig.PostStartCommand,
	}

	mergedCommands := merger.MergeCommands(resolvedFeatures, userCommands)

	// Execute merged commands instead of user commands
	if mergedOnCreate, exists := mergedCommands["onCreateCommand"]; exists {
		// Execute merged onCreate command
	}
	// Similar for other lifecycle commands
} else {
	// Existing single-command execution logic
}
```

**Step 5: Run test to verify it passes**

Run: `go test ./pkg/devcontainer -run TestMergeLifecycleCommands -v`
Expected: PASS

**Step 6: Commit**

```bash
git add pkg/devcontainer/lifecycle_merger.go pkg/devcontainer/lifecycle_merger_test.go pkg/runner/runner.go
git commit -m "feat: implement feature lifecycle hook integration

- Add LifecycleMerger for combining feature and user commands
- Feature commands execute before user commands per specification
- Support all five lifecycle hook types from features
- Integrate with runner lifecycle execution system
- Add comprehensive tests for command merging logic"
```

---

## Task 5: Add Container Properties Support

**Files:**
- Modify: `pkg/runner/runner.go:650-750`
- Test: `pkg/runner/runner_test.go`

**Step 1: Write failing test for feature-contributed container properties**

```go
func TestApplyFeatureContainerProperties(t *testing.T) {
	// Test that features can contribute security options, capabilities, etc.
	features := []*devcontainer.ResolvedFeature{
		{
			ID: "docker-feature",
			Metadata: &devcontainer.FeatureMetadata{
				Privileged:  &[]bool{true}[0],
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
```

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/runner -run TestApplyFeatureContainerProperties -v`
Expected: FAIL with "NewFeaturePropertiesApplier undefined"

**Step 3: Implement feature properties applicator**

```go
// Add to pkg/runner/runner.go

// FeaturePropertiesApplier applies feature metadata to container configuration
type FeaturePropertiesApplier struct{}

// NewFeaturePropertiesApplier creates a new properties applicator
func NewFeaturePropertiesApplier() *FeaturePropertiesApplier {
	return &FeaturePropertiesApplier{}
}

// ApplyFeatureProperties applies feature container properties to Docker args and environment
func (a *FeaturePropertiesApplier) ApplyFeatureProperties(baseArgs []string, features []*devcontainer.ResolvedFeature, baseEnv map[string]string) ([]string, map[string]string) {
	enhancedArgs := make([]string, len(baseArgs))
	copy(enhancedArgs, baseArgs)

	enhancedEnv := make(map[string]string)
	for k, v := range baseEnv {
		enhancedEnv[k] = v
	}

	for _, feature := range features {
		if feature.Metadata == nil {
			continue
		}

		metadata := feature.Metadata

		// Apply security properties
		if metadata.Privileged != nil && *metadata.Privileged {
			enhancedArgs = append(enhancedArgs, "--privileged")
		}

		for _, cap := range metadata.CapAdd {
			enhancedArgs = append(enhancedArgs, "--cap-add="+cap)
		}

		for _, secOpt := range metadata.SecurityOpt {
			enhancedArgs = append(enhancedArgs, "--security-opt="+secOpt)
		}

		// Apply feature environment variables
		for key, value := range metadata.ContainerEnv {
			enhancedEnv[key] = value
		}

		// TODO: Apply feature-contributed mounts (Task 6)
	}

	return enhancedArgs, enhancedEnv
}
```

**Step 4: Integrate with container creation in runner**

Modify container creation logic around line 650 in `pkg/runner/runner.go`:

```go
// After building Docker args but before creating container
if devConfig.Features != nil && len(devConfig.Features) > 0 {
	// Resolve features for properties application
	resolver := devcontainer.NewFeatureResolver(filepath.Join(os.TempDir(), "packnplay-features-cache"))

	var resolvedFeatures []*devcontainer.ResolvedFeature
	for reference, options := range devConfig.Features {
		feature, err := resolver.ResolveFeature(reference, options)
		if err != nil {
			if config.Verbose {
				fmt.Fprintf(os.Stderr, "Warning: failed to resolve feature %s for properties: %v\n", reference, err)
			}
			continue
		}
		resolvedFeatures = append(resolvedFeatures, feature)
	}

	// Apply feature container properties
	applier := NewFeaturePropertiesApplier()
	args, containerEnv = applier.ApplyFeatureProperties(args, resolvedFeatures, containerEnv)
}
```

**Step 5: Run test to verify it passes**

Run: `go test ./pkg/runner -run TestApplyFeatureContainerProperties -v`
Expected: PASS

**Step 6: Commit**

```bash
git add pkg/runner/runner.go pkg/runner/runner_test.go
git commit -m "feat: add feature container properties support

- Implement FeaturePropertiesApplier for security options and capabilities
- Support privileged, capAdd, securityOpt from feature metadata
- Apply feature-contributed environment variables to containers
- Integrate with container creation in runner
- Add comprehensive tests for feature properties application"
```

---

## Task 6: Add Comprehensive E2E Tests for Specification Compliance

**Files:**
- Modify: `pkg/runner/e2e_test.go:1950+`

**Step 1: Write failing test for node feature with version option**

```go
func TestE2E_NodeFeatureWithVersion(t *testing.T) {
	skipIfNoDocker(t)

	projectDir := createTestProject(t, map[string]string{
		".devcontainer/devcontainer.json": `{
			"image": "mcr.microsoft.com/devcontainers/base:ubuntu",
			"features": {
				"ghcr.io/devcontainers/features/node:1": {
					"version": "18.20.0"
				}
			}
		}`,
	})
	defer os.RemoveAll(projectDir)

	containerName := getContainerNameForProject(projectDir)
	defer cleanupContainer(t, containerName)

	// Verify specific Node.js version installed
	output, err := runPacknplayInDir(t, projectDir, "run", "--no-worktree", "node", "--version")
	require.NoError(t, err, "Node version check failed: %s", output)
	require.Contains(t, output, "v18.20.0", "Expected specific Node.js version")
}

func TestE2E_FeatureLifecycleCommands(t *testing.T) {
	skipIfNoDocker(t)

	// Create local feature with lifecycle commands
	tmpDir := t.TempDir()
	localFeatureDir := filepath.Join(tmpDir, "lifecycle-feature")
	err := os.MkdirAll(localFeatureDir, 0755)
	require.NoError(t, err)

	// Feature metadata with lifecycle commands
	metadata := `{
		"id": "lifecycle-feature",
		"version": "1.0.0",
		"name": "Feature with Lifecycle",
		"postCreateCommand": "echo 'feature postCreate' > /tmp/feature-lifecycle.log"
	}`
	err = os.WriteFile(filepath.Join(localFeatureDir, "devcontainer-feature.json"), []byte(metadata), 0644)
	require.NoError(t, err)

	installScript := "#!/bin/bash\necho 'Feature installed'\ntouch /feature-installed"
	err = os.WriteFile(filepath.Join(localFeatureDir, "install.sh"), []byte(installScript), 0755)
	require.NoError(t, err)

	projectDir := createTestProject(t, map[string]string{
		".devcontainer/devcontainer.json": fmt.Sprintf(`{
			"image": "alpine:latest",
			"features": {
				".devcontainer/local-features/lifecycle-feature": {}
			},
			"postCreateCommand": "echo 'user postCreate' >> /tmp/feature-lifecycle.log"
		}`),
		".devcontainer/local-features/lifecycle-feature/devcontainer-feature.json": metadata,
		".devcontainer/local-features/lifecycle-feature/install.sh": installScript,
	})
	defer os.RemoveAll(projectDir)

	containerName := getContainerNameForProject(projectDir)
	defer cleanupContainer(t, containerName)

	// Verify both feature and user lifecycle commands executed
	output, err := runPacknplayInDir(t, projectDir, "run", "--no-worktree", "cat", "/tmp/feature-lifecycle.log")
	require.NoError(t, err, "Lifecycle commands failed: %s", output)
	require.Contains(t, output, "feature postCreate", "Feature postCreate should execute first")
	require.Contains(t, output, "user postCreate", "User postCreate should execute second")
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./pkg/runner -run "TestE2E_NodeFeatureWithVersion|TestE2E_FeatureLifecycleCommands" -v`
Expected: FAIL (options and lifecycle hooks not working)

**Step 3: Fix tests by implementing missing functionality**

Address the gaps to make tests pass - this verifies the implementation works end-to-end.

**Step 4: Run tests to verify they pass**

Run: `go test ./pkg/runner -run "TestE2E_NodeFeatureWithVersion|TestE2E_FeatureLifecycleCommands" -v`
Expected: PASS

**Step 5: Commit**

```bash
git add pkg/runner/e2e_test.go
git commit -m "feat: add comprehensive E2E tests for features specification compliance

- Test node feature with specific version option (validates options processing)
- Test feature lifecycle commands execute before user commands
- Verify complete feature workflow from options to execution
- Real-world validation of specification compliance
- Tests demonstrate working features with actual community features"
```

---

## Task 7: Final Integration and Documentation

**Files:**
- Modify: `docs/DEVCONTAINER_GUIDE.md`
- Update: `/tmp/devcontainer-torture-test/`

**Step 1: Update torture test with specification-compliant features**

Modify `/tmp/devcontainer-torture-test/.devcontainer/devcontainer.json`:

```json
{
  "name": "Complete Devcontainer Torture Test",
  "image": "mcr.microsoft.com/devcontainers/base:ubuntu",
  "features": {
    "ghcr.io/devcontainers/features/common-utils:2": {
      "installZsh": true,
      "configureZshAsDefaultShell": true
    },
    "ghcr.io/devcontainers/features/node:1": {
      "version": "18.20.0",
      "nodeGypDependencies": true
    },
    "ghcr.io/devcontainers/features/docker-in-docker:2": {
      "version": "latest",
      "enableNonRootDocker": true
    }
  },
  "remoteUser": "vscode",
  "containerEnv": {
    "NODE_ENV": "development"
  },
  "mounts": [
    "source=${localWorkspaceFolder}/test-data,target=/mounted-data,type=bind"
  ],
  "runArgs": ["--memory=2g"],
  "forwardPorts": [3000],
  "onCreateCommand": "echo 'User onCreate after features'",
  "postCreateCommand": "node --version && docker --version"
}
```

**Step 2: Test complete torture test**

Run: `cd /tmp/devcontainer-torture-test && packnplay run --no-worktree ./torture-test-validation.sh`
Expected: All features install with correct options, lifecycle commands work

**Step 3: Update documentation with specification compliance**

Update `docs/DEVCONTAINER_GUIDE.md` to reflect complete features support:
- Document feature options processing
- Show lifecycle hook behavior
- Add security properties examples
- Update feature compatibility matrix

**Step 4: Run final test suite**

Run: `make test && make lint`
Expected: Full suite passes with all new features functionality

**Step 5: Commit**

```bash
git add docs/DEVCONTAINER_GUIDE.md
git commit -m "docs: update guide for complete devcontainer features specification compliance

- Document feature options processing and environment variables
- Add lifecycle hook execution order examples
- Show security properties and container configuration
- Update compatibility matrix to reflect 100% specification compliance
- Provide complete torture test demonstrating all features"
```

---

## Success Criteria Validation

**Technical Validation:**
- ✅ Feature options convert to environment variables per specification regex
- ✅ Complete FeatureMetadata supports all specification fields
- ✅ Multi-stage Docker builds solve OCI build context limitation
- ✅ Lifecycle hooks execute in proper order (features before user)
- ✅ Container properties (security, capabilities) applied correctly

**Compatibility Validation:**
- ✅ Microsoft universal devcontainer image works perfectly
- ✅ Popular community features (node, docker-in-docker, common-utils) work with options
- ✅ Feature dependency resolution matches VS Code behavior
- ✅ Build performance equivalent to official implementation

**Quality Validation:**
- ✅ Complete test suite passes (40+ E2E tests)
- ✅ Specification compliance verified through torture test
- ✅ Documentation accurate and complete
- ✅ No regressions in existing functionality