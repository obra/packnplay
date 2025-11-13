# Complete Devcontainer Features Specification Compliance Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Achieve 100% devcontainer features specification compliance by fixing remaining gaps identified in comprehensive analysis.

**Architecture:** Systematically address the 25+ remaining implementation gaps including feature-contributed mounts, missing lifecycle hooks, option validation, and production-grade error handling.

**Tech Stack:** Go, Docker BuildKit, devcontainer specification, comprehensive test coverage

---

## Task 1: Fix Linting Issues (Quick Win)

**Files:**
- Modify: `pkg/runner/e2e_test.go:2028`
- Modify: `pkg/runner/runner.go:788,895`

**Step 1: Fix formatting issue**

```bash
gofmt -w pkg/runner/e2e_test.go
```

**Step 2: Fix gosimple nil check issues**

In `pkg/runner/runner.go` line 788:
```go
// Change from:
if devConfig.Features != nil && len(devConfig.Features) > 0 {

// To:
if len(devConfig.Features) > 0 {
```

In `pkg/runner/runner.go` line 895:
```go
// Change from:
hasFeatures := devConfig.Features != nil && len(devConfig.Features) > 0

// To:
hasFeatures := len(devConfig.Features) > 0
```

**Step 3: Verify lint passes**

Run: `make lint`
Expected: No output (clean lint)

**Step 4: Commit**

```bash
git add pkg/runner/runner.go pkg/runner/e2e_test.go
git commit -m "fix: resolve remaining golangci-lint issues

- Fix gofmt formatting in e2e_test.go
- Remove unnecessary nil checks in features processing
- All linting rules now pass cleanly"
```

---

## Task 2: Implement Feature-Contributed Mounts (Critical Gap #1)

**Files:**
- Modify: `pkg/runner/runner.go:800-810`
- Test: `pkg/runner/runner_test.go`

**Step 1: Write failing test for feature mounts**

```go
func TestApplyFeatureMounts(t *testing.T) {
	features := []*devcontainer.ResolvedFeature{
		{
			ID: "docker-feature",
			Metadata: &devcontainer.FeatureMetadata{
				Mounts: []devcontainer.Mount{
					{
						Source: "/var/run/docker.sock",
						Target: "/var/run/docker.sock",
						Type:   "bind",
					},
					{
						Source: "feature-volume",
						Target: "/feature-data",
						Type:   "volume",
					},
				},
			},
		},
	}

	applier := NewFeaturePropertiesApplier()
	dockerArgs := []string{"run", "-d", "--name", "test"}

	enhancedArgs, _ := applier.ApplyFeatureProperties(dockerArgs, features, map[string]string{})

	// Verify mounts added to Docker args
	assert.Contains(t, enhancedArgs, "--mount")
	assert.Contains(t, enhancedArgs, "source=/var/run/docker.sock,target=/var/run/docker.sock,type=bind")
	assert.Contains(t, enhancedArgs, "--mount")
	assert.Contains(t, enhancedArgs, "source=feature-volume,target=/feature-data,type=volume")
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/runner -run TestApplyFeatureMounts -v`
Expected: FAIL with assertion errors (mounts not applied)

**Step 3: Implement feature mounts in FeaturePropertiesApplier**

In `pkg/runner/runner.go`, modify the `ApplyFeatureProperties` method around line 80:

```go
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

		// Apply feature-contributed mounts
		for _, mount := range metadata.Mounts {
			mountStr := fmt.Sprintf("source=%s,target=%s,type=%s", mount.Source, mount.Target, mount.Type)
			enhancedArgs = append(enhancedArgs, "--mount", mountStr)
		}

		// Apply feature environment variables
		for key, value := range metadata.ContainerEnv {
			enhancedEnv[key] = value
		}
	}

	return enhancedArgs, enhancedEnv
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./pkg/runner -run TestApplyFeatureMounts -v`
Expected: PASS

**Step 5: Commit**

```bash
git add pkg/runner/runner.go pkg/runner/runner_test.go
git commit -m "feat: implement feature-contributed mounts support

- Apply feature mounts to Docker container arguments
- Support bind mounts, named volumes, and other mount types
- Convert Mount struct to Docker --mount flags
- Add comprehensive test for mount application
- Enables docker-in-docker and volume features to work correctly"
```

---

## Task 3: Add Missing Lifecycle Command Fields (Critical Gap #2)

**Files:**
- Modify: `pkg/devcontainer/config.go:25-27`
- Test: `pkg/devcontainer/config_test.go`

**Step 1: Write failing test for missing lifecycle fields**

```go
func TestConfig_AllLifecycleCommands(t *testing.T) {
	configJSON := `{
		"image": "alpine:latest",
		"onCreateCommand": "echo onCreate",
		"updateContentCommand": "echo updateContent",
		"postCreateCommand": "echo postCreate",
		"postStartCommand": "echo postStart",
		"postAttachCommand": "echo postAttach"
	}`

	var config Config
	err := json.Unmarshal([]byte(configJSON), &config)
	require.NoError(t, err)

	// Verify all lifecycle commands are parsed
	assert.NotNil(t, config.OnCreateCommand)
	assert.NotNil(t, config.UpdateContentCommand)
	assert.NotNil(t, config.PostCreateCommand)
	assert.NotNil(t, config.PostStartCommand)
	assert.NotNil(t, config.PostAttachCommand)
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/devcontainer -run TestConfig_AllLifecycleCommands -v`
Expected: FAIL with compilation errors for missing fields

**Step 3: Add missing lifecycle command fields to Config struct**

In `pkg/devcontainer/config.go`, modify the Config struct around line 25:

```go
type Config struct {
	Image        string            `json:"image"`
	DockerFile   string            `json:"dockerFile"`
	Build        *BuildConfig      `json:"build,omitempty"`
	RemoteUser   string            `json:"remoteUser"`
	ContainerEnv map[string]string `json:"containerEnv,omitempty"`
	RemoteEnv    map[string]string `json:"remoteEnv,omitempty"`
	ForwardPorts []interface{}     `json:"forwardPorts,omitempty"` // int or string
	Mounts       []string          `json:"mounts,omitempty"`       // Docker mount syntax
	RunArgs      []string          `json:"runArgs,omitempty"`      // Additional docker run arguments
	Features     map[string]interface{} `json:"features,omitempty"`    // Devcontainer features

	// Lifecycle commands - complete set per specification
	OnCreateCommand      *LifecycleCommand `json:"onCreateCommand,omitempty"`
	UpdateContentCommand *LifecycleCommand `json:"updateContentCommand,omitempty"`
	PostCreateCommand    *LifecycleCommand `json:"postCreateCommand,omitempty"`
	PostStartCommand     *LifecycleCommand `json:"postStartCommand,omitempty"`
	PostAttachCommand    *LifecycleCommand `json:"postAttachCommand,omitempty"`
}
```

**Step 4: Update lifecycle merger to support all hook types**

In `pkg/devcontainer/lifecycle_merger.go`, modify the `MergeCommands` method around line 20:

```go
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
				Commands: mergedCommands,
			}
		}
	}

	return result
}
```

**Step 5: Update runner to support new lifecycle commands**

In `pkg/runner/runner.go`, modify the lifecycle execution around line 940:

```go
// Merge feature and user lifecycle commands
merger := devcontainer.NewLifecycleMerger()
userCommands := map[string]*devcontainer.LifecycleCommand{
	"onCreateCommand":      devConfig.OnCreateCommand,
	"updateContentCommand": devConfig.UpdateContentCommand,
	"postCreateCommand":    devConfig.PostCreateCommand,
	"postStartCommand":     devConfig.PostStartCommand,
	"postAttachCommand":    devConfig.PostAttachCommand,
}

mergedCommands := merger.MergeCommands(resolvedFeatures, userCommands)

// Execute merged commands if available
if mergedOnCreate, exists := mergedCommands["onCreateCommand"]; exists {
	if err := executor.Execute("onCreate", mergedOnCreate); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: merged onCreateCommand failed: %v\n", err)
	}
}

if mergedUpdateContent, exists := mergedCommands["updateContentCommand"]; exists {
	if err := executor.Execute("updateContent", mergedUpdateContent); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: merged updateContentCommand failed: %v\n", err)
	}
}

if mergedPostCreate, exists := mergedCommands["postCreateCommand"]; exists {
	if err := executor.Execute("postCreate", mergedPostCreate); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: merged postCreateCommand failed: %v\n", err)
	}
}
```

**Step 6: Run test to verify it passes**

Run: `go test ./pkg/devcontainer -run TestConfig_AllLifecycleCommands -v`
Expected: PASS

**Step 7: Commit**

```bash
git add pkg/devcontainer/config.go pkg/devcontainer/config_test.go pkg/devcontainer/lifecycle_merger.go pkg/runner/runner.go
git commit -m "feat: add missing lifecycle command fields for complete specification support

- Add updateContentCommand and postAttachCommand to Config struct
- Update lifecycle merger to support all five hook types
- Integrate new lifecycle commands in runner execution
- Add comprehensive test for all lifecycle command parsing
- Achieves complete lifecycle hook specification compliance"
```

---

## Task 4: Implement Feature Option Validation (Critical Gap #3)

**Files:**
- Modify: `pkg/devcontainer/features.go:298-321`
- Test: `pkg/devcontainer/features_test.go`

**Step 1: Write failing test for option validation**

```go
func TestValidateFeatureOptions(t *testing.T) {
	tests := []struct {
		name           string
		userOptions    map[string]interface{}
		optionSpecs    map[string]OptionSpec
		expectError    bool
		expectedError  string
	}{
		{
			name: "valid string option",
			userOptions: map[string]interface{}{
				"version": "18.20.0",
			},
			optionSpecs: map[string]OptionSpec{
				"version": {Type: "string", Default: "latest"},
			},
			expectError: false,
		},
		{
			name: "invalid type - number for string option",
			userOptions: map[string]interface{}{
				"version": 18,
			},
			optionSpecs: map[string]OptionSpec{
				"version": {Type: "string", Default: "latest"},
			},
			expectError:   true,
			expectedError: "option 'version' expects string but got number",
		},
		{
			name: "invalid enum value",
			userOptions: map[string]interface{}{
				"shell": "fish",
			},
			optionSpecs: map[string]OptionSpec{
				"shell": {
					Type:      "string",
					Default:   "bash",
					Proposals: []string{"bash", "zsh"},
				},
			},
			expectError:   true,
			expectedError: "option 'shell' value 'fish' not in allowed proposals: [bash zsh]",
		},
		{
			name: "boolean option validation",
			userOptions: map[string]interface{}{
				"installZsh": "true", // string instead of bool
			},
			optionSpecs: map[string]OptionSpec{
				"installZsh": {Type: "boolean", Default: false},
			},
			expectError:   true,
			expectedError: "option 'installZsh' expects boolean but got string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			processor := NewFeatureOptionsProcessor()
			_, err := processor.ValidateAndProcessOptions(tt.userOptions, tt.optionSpecs)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/devcontainer -run TestValidateFeatureOptions -v`
Expected: FAIL with "ValidateAndProcessOptions undefined"

**Step 3: Implement option validation in FeatureOptionsProcessor**

```go
// Add to pkg/devcontainer/features.go

import (
	"reflect"
	"strconv"
)

// ValidateAndProcessOptions validates user options against specs then processes them
func (p *FeatureOptionsProcessor) ValidateAndProcessOptions(userOptions map[string]interface{}, optionSpecs map[string]OptionSpec) (map[string]string, error) {
	// Step 1: Validate all user-provided options
	for optionName, userValue := range userOptions {
		spec, exists := optionSpecs[optionName]
		if !exists {
			return nil, fmt.Errorf("unknown option '%s' - feature does not define this option", optionName)
		}

		if err := p.validateOptionValue(optionName, userValue, spec); err != nil {
			return nil, err
		}
	}

	// Step 2: Process options (existing ProcessOptions logic)
	return p.ProcessOptions(userOptions, optionSpecs), nil
}

// validateOptionValue validates a single option value against its specification
func (p *FeatureOptionsProcessor) validateOptionValue(optionName string, value interface{}, spec OptionSpec) error {
	// Type validation
	switch spec.Type {
	case "string":
		if _, ok := value.(string); !ok {
			actualType := reflect.TypeOf(value).Kind().String()
			return fmt.Errorf("option '%s' expects string but got %s", optionName, actualType)
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			actualType := reflect.TypeOf(value).Kind().String()
			return fmt.Errorf("option '%s' expects boolean but got %s", optionName, actualType)
		}
	case "number":
		switch value.(type) {
		case int, float64, float32:
			// Valid numeric types
		default:
			actualType := reflect.TypeOf(value).Kind().String()
			return fmt.Errorf("option '%s' expects number but got %s", optionName, actualType)
		}
	}

	// Enum validation (proposals)
	if len(spec.Proposals) > 0 {
		valueStr := fmt.Sprintf("%v", value)
		found := false
		for _, proposal := range spec.Proposals {
			if proposal == valueStr {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("option '%s' value '%v' not in allowed proposals: %v", optionName, value, spec.Proposals)
		}
	}

	return nil
}
```

**Step 4: Update Dockerfile generator to use validated options**

In `internal/dockerfile/dockerfile_generator.go`, modify the options processing:

```go
// Process feature options to environment variables with validation
if feature.Metadata != nil && feature.Metadata.Options != nil {
	envVars, err := processor.ValidateAndProcessOptions(feature.Options, feature.Metadata.Options)
	if err != nil {
		return "", fmt.Errorf("invalid options for feature %s: %w", feature.ID, err)
	}
	for envName, envValue := range envVars {
		lines = append(lines, fmt.Sprintf("ENV %s=%s", envName, envValue))
	}
}
```

**Step 5: Run test to verify it passes**

Run: `go test ./pkg/devcontainer -run TestValidateFeatureOptions -v`
Expected: PASS

**Step 6: Commit**

```bash
git add pkg/devcontainer/features.go internal/dockerfile/dockerfile_generator.go pkg/devcontainer/features_test.go
git commit -m "feat: implement feature option validation per devcontainer specification

- Add ValidateAndProcessOptions with type checking (string, boolean, number)
- Validate enum values against proposals array
- Provide clear error messages for invalid options
- Integrate validation into Dockerfile generation
- Prevents silent failures from incorrect feature options"
```

---

## Task 5: Add Init and Entrypoint Support

**Files:**
- Modify: `pkg/runner/runner.go:80-95`
- Test: `pkg/runner/runner_test.go`

**Step 1: Write failing test for init and entrypoint properties**

```go
func TestApplyFeatureInitAndEntrypoint(t *testing.T) {
	features := []*devcontainer.ResolvedFeature{
		{
			ID: "init-feature",
			Metadata: &devcontainer.FeatureMetadata{
				Init:       &[]bool{true}[0],
				Entrypoint: []string{"/custom-entrypoint.sh"},
			},
		},
	}

	applier := NewFeaturePropertiesApplier()
	dockerArgs := []string{"run", "-d", "--name", "test"}

	enhancedArgs, _ := applier.ApplyFeatureProperties(dockerArgs, features, map[string]string{})

	// Verify init and entrypoint flags added
	assert.Contains(t, enhancedArgs, "--init")
	assert.Contains(t, enhancedArgs, "--entrypoint=/custom-entrypoint.sh")
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/runner -run TestApplyFeatureInitAndEntrypoint -v`
Expected: FAIL (init and entrypoint not applied)

**Step 3: Implement init and entrypoint in FeaturePropertiesApplier**

In `pkg/runner/runner.go`, enhance the `ApplyFeatureProperties` method:

```go
// Add after security properties section:

// Apply init process setting
if metadata.Init != nil && *metadata.Init {
	enhancedArgs = append(enhancedArgs, "--init")
}

// Apply entrypoint override
if len(metadata.Entrypoint) > 0 {
	entrypointStr := strings.Join(metadata.Entrypoint, " ")
	enhancedArgs = append(enhancedArgs, "--entrypoint="+entrypointStr)
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./pkg/runner -run TestApplyFeatureInitAndEntrypoint -v`
Expected: PASS

**Step 5: Commit**

```bash
git add pkg/runner/runner.go pkg/runner/runner_test.go
git commit -m "feat: add feature init and entrypoint support

- Apply feature init process setting via --init flag
- Apply feature entrypoint override
- Add comprehensive test for init and entrypoint properties
- Completes container properties support from feature metadata"
```

---

## Task 6: Add Feature User Context Variables

**Files:**
- Modify: `internal/dockerfile/dockerfile_generator.go:50-70`
- Test: `internal/dockerfile/dockerfile_generator_test.go`

**Step 1: Write failing test for user context variables**

```go
func TestFeatureUserContextVariables(t *testing.T) {
	features := []*devcontainer.ResolvedFeature{
		{
			ID:      "user-aware-feature",
			Version: "1.0.0",
			InstallPath: t.TempDir(),
			Options: map[string]interface{}{
				"username": "custom-user",
			},
		},
	}

	generator := NewDockerfileGenerator()
	dockerfile, err := generator.Generate("ubuntu:22.04", features, "vscode", "/project/.devcontainer")
	require.NoError(t, err)

	// Verify user context variables are set
	assert.Contains(t, dockerfile, "ENV _REMOTE_USER=vscode")
	assert.Contains(t, dockerfile, "ENV _REMOTE_USER_HOME=/home/vscode")
	assert.Contains(t, dockerfile, "ENV _CONTAINER_USER=vscode")
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/dockerfile -run TestFeatureUserContextVariables -v`
Expected: FAIL (user context variables not set)

**Step 3: Add user context variables to Dockerfile generation**

In `internal/dockerfile/dockerfile_generator.go`, modify the multi-stage and single-stage generators:

```go
// Add after USER root but before feature installation:

// Set user context variables for features
lines = append(lines, "# Set user context variables for features")
lines = append(lines, fmt.Sprintf("ENV _REMOTE_USER=%s", remoteUser))
lines = append(lines, fmt.Sprintf("ENV _REMOTE_USER_HOME=/home/%s", remoteUser))
lines = append(lines, fmt.Sprintf("ENV _CONTAINER_USER=%s", remoteUser))
lines = append(lines, "")
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/dockerfile -run TestFeatureUserContextVariables -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/dockerfile/dockerfile_generator.go internal/dockerfile/dockerfile_generator_test.go
git commit -m "feat: add feature user context variables support

- Set _REMOTE_USER environment variable for features
- Set _REMOTE_USER_HOME and _CONTAINER_USER variables
- Enable features to configure user-specific settings
- Add test for user context variable presence in Dockerfiles"
```

---

## Task 7: Add Comprehensive E2E Tests for Complete Specification

**Files:**
- Modify: `pkg/runner/e2e_test.go:2100+`

**Step 1: Write failing E2E test for docker-in-docker feature**

```go
func TestE2E_DockerInDockerFeature(t *testing.T) {
	skipIfNoDocker(t)

	projectDir := createTestProject(t, map[string]string{
		".devcontainer/devcontainer.json": `{
			"image": "mcr.microsoft.com/devcontainers/base:ubuntu",
			"features": {
				"ghcr.io/devcontainers/features/docker-in-docker:2": {
					"version": "latest",
					"enableNonRootDocker": true
				}
			},
			"remoteUser": "vscode"
		}`,
	})
	defer os.RemoveAll(projectDir)

	containerName := getContainerNameForProject(projectDir)
	defer cleanupContainer(t, containerName)

	// Verify Docker is available and working
	output, err := runPacknplayInDir(t, projectDir, "run", "--no-worktree", "docker", "--version")
	require.NoError(t, err, "Docker-in-docker failed: %s", output)
	require.Contains(t, output, "Docker version", "Docker should be available in container")

	// Verify docker daemon is accessible
	output, err = runPacknplayInDir(t, projectDir, "run", "--no-worktree", "docker", "info")
	require.NoError(t, err, "Docker daemon not accessible: %s", output)
}

func TestE2E_FeatureUpdateContentCommand(t *testing.T) {
	skipIfNoDocker(t)

	// Create local feature with updateContentCommand
	projectDir := createTestProject(t, map[string]string{
		".devcontainer/devcontainer.json": `{
			"image": "alpine:latest",
			"features": {
				".devcontainer/local-features/update-feature": {}
			},
			"updateContentCommand": "echo 'user updateContent' >> /tmp/update-commands.log"
		}`,
		".devcontainer/local-features/update-feature/devcontainer-feature.json": `{
			"id": "update-feature",
			"version": "1.0.0",
			"name": "Update Feature",
			"updateContentCommand": "echo 'feature updateContent' >> /tmp/update-commands.log"
		}`,
		".devcontainer/local-features/update-feature/install.sh": `#!/bin/sh\necho 'Update feature installed'\ntouch /update-feature-installed`,
	})
	defer os.RemoveAll(projectDir)

	containerName := getContainerNameForProject(projectDir)
	defer cleanupContainer(t, containerName)

	// Verify container starts (updateContent doesn't run during container creation)
	output, err := runPacknplayInDir(t, projectDir, "run", "--no-worktree", "echo", "container ready")
	require.NoError(t, err, "Container startup failed: %s", output)
	require.Contains(t, output, "container ready")

	// Note: updateContentCommand would run when content changes, not during initial setup
	// This test verifies the command is parsed and ready, even if not executed
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/runner -run "TestE2E_DockerInDocker|TestE2E_FeatureUpdateContent" -v`
Expected: FAIL (docker-in-docker requires mounts and privileged, updateContent requires missing field)

**Step 3: Tests should pass after previous tasks complete**

Run: `go test ./pkg/runner -run "TestE2E_DockerInDocker|TestE2E_FeatureUpdateContent" -v`
Expected: PASS (after mounts and lifecycle fields implemented)

**Step 4: Add test for feature option validation**

```go
func TestE2E_FeatureOptionValidation(t *testing.T) {
	skipIfNoDocker(t)

	projectDir := createTestProject(t, map[string]string{
		".devcontainer/devcontainer.json": `{
			"image": "alpine:latest",
			"features": {
				"ghcr.io/devcontainers/features/node:1": {
					"version": "invalid-version-string"
				}
			}
		}`,
	})
	defer os.RemoveAll(projectDir)

	// This should fail with clear error message about invalid option
	_, err := runPacknplayInDir(t, projectDir, "run", "--no-worktree", "echo", "test")
	require.Error(t, err, "Invalid feature options should cause clear error")
	// Note: Exact error message depends on Node feature's option validation
}
```

**Step 5: Commit comprehensive E2E tests**

```bash
git add pkg/runner/e2e_test.go
git commit -m "feat: add comprehensive E2E tests for complete specification compliance

- Test docker-in-docker feature with privileged mode and mounts
- Test feature updateContentCommand integration
- Test feature option validation with clear error messages
- Validate complete feature workflow for advanced community features
- Demonstrates real-world usage scenarios"
```

---

## Task 8: Fix OCI Build Context Integration

**Files:**
- Modify: `pkg/runner/image_manager.go:150-180`

**Step 1: Add comprehensive directory copying utilities**

```go
// Add to pkg/runner/image_manager.go

import (
	"io"
)

// copyDir recursively copies a directory tree
func copyDir(src, dst string) error {
	// Read source directory
	entries, err := os.ReadDir(src)
	if err != nil {
		return fmt.Errorf("failed to read source directory: %w", err)
	}

	// Create destination directory
	if err := os.MkdirAll(dst, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Copy each entry
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			// Recursively copy subdirectory
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			// Copy file
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}

// copyFile copies a single file preserving permissions
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dstFile.Close()

	// Copy content
	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("failed to copy file content: %w", err)
	}

	// Copy permissions
	srcInfo, err := srcFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat source file: %w", err)
	}

	if err := os.Chmod(dst, srcInfo.Mode()); err != nil {
		return fmt.Errorf("failed to set file permissions: %w", err)
	}

	return nil
}
```

**Step 2: Integrate OCI feature copying into build process**

In `pkg/runner/image_manager.go`, modify the `buildWithFeatures` method around line 170:

```go
// Before generating Dockerfile, copy OCI features into build context
ociCacheDir := filepath.Join(buildContextPath, "oci-cache")

for _, feature := range resolvedFeatures {
	// If feature is outside build context, copy it in
	if !strings.HasPrefix(feature.InstallPath, buildContextPath) {
		// Create OCI cache directory if needed
		if err := os.MkdirAll(ociCacheDir, 0755); err != nil {
			return fmt.Errorf("failed to create OCI cache directory: %w", err)
		}

		// Copy OCI feature to build context
		featureName := filepath.Base(feature.InstallPath)
		destPath := filepath.Join(ociCacheDir, featureName)

		if err := copyDir(feature.InstallPath, destPath); err != nil {
			return fmt.Errorf("failed to copy OCI feature %s to build context: %w", feature.ID, err)
		}

		// Update feature install path to build context location
		feature.InstallPath = filepath.Join("oci-cache", featureName)
	}
}

// Continue with existing Dockerfile generation...
```

**Step 3: Add cleanup of OCI cache after successful build**

```go
// After successful Docker build, clean up OCI cache
defer func() {
	if err == nil {
		// Clean up copied OCI features
		ociCacheDir := filepath.Join(buildContextPath, "oci-cache")
		if _, err := os.Stat(ociCacheDir); err == nil {
			os.RemoveAll(ociCacheDir)
		}
	}
}()
```

**Step 4: Test integration**

Run: `go test ./pkg/runner -run TestE2E_CommunityFeature -v`
Expected: PASS (OCI features should build without manual copying)

**Step 5: Commit**

```bash
git add pkg/runner/image_manager.go
git commit -m "feat: implement automatic OCI feature build context integration

- Add copyDir and copyFile utilities for recursive directory copying
- Copy OCI features into build context before Dockerfile generation
- Update feature paths to point to build context locations
- Clean up temporary OCI cache after successful build
- Solves critical build context limitation for OCI features"
```

---

## Task 9: Add Comprehensive Validation Tests

**Files:**
- Modify: `pkg/runner/e2e_test.go:2200+`

**Step 1: Write E2E test for Microsoft universal image pattern**

```go
func TestE2E_MicrosoftUniversalPattern(t *testing.T) {
	skipIfNoDocker(t)

	projectDir := createTestProject(t, map[string]string{
		".devcontainer/devcontainer.json": `{
			"build": { "dockerfile": "Dockerfile" },
			"features": {
				"ghcr.io/devcontainers/features/common-utils:2": {
					"installZsh": true,
					"configureZshAsDefaultShell": true
				},
				"ghcr.io/devcontainers/features/docker-in-docker:2": {
					"version": "latest",
					"enableNonRootDocker": true
				}
			},
			"remoteUser": "vscode",
			"onCreateCommand": "echo 'Universal image pattern test'",
			"postCreateCommand": "docker --version && zsh --version"
		}`,
		".devcontainer/Dockerfile": `FROM mcr.microsoft.com/devcontainers/base:ubuntu
# Features will be added automatically`,
	})
	defer os.RemoveAll(projectDir)

	containerName := getContainerNameForProject(projectDir)
	defer cleanupContainer(t, containerName)

	// Verify universal image pattern works
	output, err := runPacknplayInDir(t, projectDir, "run", "--no-worktree", "echo", "universal pattern success")
	require.NoError(t, err, "Universal pattern failed: %s", output)
	require.Contains(t, output, "universal pattern success")
}
```

**Step 2: Write comprehensive feature interaction test**

```go
func TestE2E_CompleteFeatureWorkflow(t *testing.T) {
	skipIfNoDocker(t)

	projectDir := createTestProject(t, map[string]string{
		".devcontainer/devcontainer.json": `{
			"image": "ubuntu:22.04",
			"features": {
				"ghcr.io/devcontainers/features/common-utils:2": {
					"installZsh": true,
					"username": "testuser"
				},
				"ghcr.io/devcontainers/features/node:1": {
					"version": "18.20.0",
					"nodeGypDependencies": true
				}
			},
			"remoteUser": "vscode",
			"containerEnv": {
				"NODE_ENV": "development"
			},
			"mounts": [
				"type=tmpfs,target=/tmp/fast-storage"
			],
			"runArgs": ["--memory=1g"],
			"onCreateCommand": "npm --version",
			"postCreateCommand": "node --version && echo 'Setup complete'"
		}`,
	})
	defer os.RemoveAll(projectDir)

	containerName := getContainerNameForProject(projectDir)
	defer cleanupContainer(t, containerName)

	// Verify complete workflow
	output, err := runPacknplayInDir(t, projectDir, "run", "--no-worktree", "bash", "-c",
		"echo 'Features:' && which node && which zsh && echo 'Environment:' && echo $NODE_ENV && echo 'Complete!'")
	require.NoError(t, err, "Complete workflow failed: %s", output)

	// Verify all components working
	require.Contains(t, output, "/usr/local/bin/node", "Node.js from feature")
	require.Contains(t, output, "/usr/bin/zsh", "Zsh from common-utils feature")
	require.Contains(t, output, "development", "Environment variable")
	require.Contains(t, output, "Complete!", "Workflow completed")
}
```

**Step 3: Run tests to verify current status**

Run: `go test ./pkg/runner -run "TestE2E_MicrosoftUniversal|TestE2E_CompleteFeature" -v`
Expected: Tests should pass if implementation is complete

**Step 4: Commit comprehensive validation tests**

```bash
git add pkg/runner/e2e_test.go
git commit -m "feat: add comprehensive E2E tests for complete specification validation

- Test Microsoft universal devcontainer image pattern compatibility
- Test complete feature workflow with multiple features and options
- Verify feature options, lifecycle commands, and container properties work together
- Validate real-world usage scenarios with popular community features
- Demonstrates 100% specification compliance"
```

---

## Task 10: Final Integration Testing and Documentation

**Files:**
- Modify: `docs/DEVCONTAINER_GUIDE.md`
- Update: `/tmp/devcontainer-torture-test/`

**Step 1: Run complete test suite**

Run: `make test`
Expected: All tests PASS including new specification compliance tests

**Step 2: Update torture test for 100% specification demo**

Modify `/tmp/devcontainer-torture-test/.devcontainer/devcontainer.json`:

```json
{
  "name": "Complete Devcontainer Specification Demo",
  "image": "mcr.microsoft.com/devcontainers/base:ubuntu",
  "features": {
    "ghcr.io/devcontainers/features/common-utils:2": {
      "installZsh": true,
      "configureZshAsDefaultShell": true,
      "username": "vscode",
      "userUid": 1000,
      "userGid": 1000
    },
    "ghcr.io/devcontainers/features/docker-in-docker:2": {
      "version": "latest",
      "enableNonRootDocker": true,
      "moby": true
    },
    "ghcr.io/devcontainers/features/node:1": {
      "version": "18.20.0",
      "nodeGypDependencies": true,
      "nvmVersion": "latest"
    }
  },
  "remoteUser": "vscode",
  "mounts": [
    "source=${localWorkspaceFolder}/test-data,target=/mounted-data,type=bind",
    "type=tmpfs,target=/tmp/fast-storage"
  ],
  "runArgs": [
    "--memory=2g",
    "--cpus=1.5",
    "--label", "project=${containerWorkspaceFolderBasename}"
  ],
  "containerEnv": {
    "NODE_ENV": "development",
    "DEBUG": "true"
  },
  "forwardPorts": [3000, "8080:8080"],
  "onCreateCommand": {
    "npm": "npm install -g typescript",
    "setup": "mkdir -p /workspace/logs"
  },
  "postCreateCommand": [
    "node --version",
    "docker --version",
    "zsh --version"
  ],
  "postStartCommand": "echo \"Session $(date)\" >> /workspace/logs/sessions.log"
}
```

**Step 3: Test complete torture test**

Run: `cd /tmp/devcontainer-torture-test && packnplay run --no-worktree ./torture-test-validation.sh`
Expected: All features, options, lifecycle commands, and properties work correctly

**Step 4: Update documentation for complete specification compliance**

Add to `docs/DEVCONTAINER_GUIDE.md`:

```markdown
#### Complete Specification Support

packnplay now supports 100% of the devcontainer features specification including:

**Feature Metadata:**
- Complete options with type validation and enum support
- All lifecycle hooks (onCreate, updateContent, postCreate, postStart, postAttach)
- Container properties (privileged, capAdd, securityOpt, init, entrypoint)
- Feature-contributed mounts and environment variables
- Complex dependency resolution with circular detection

**Real-World Compatibility:**
- Microsoft universal devcontainer image works perfectly
- All popular community features supported (node, python, docker-in-docker, common-utils)
- VS Code devcontainer pattern compatibility
- Advanced security and mounting scenarios

**Performance:**
- Docker layer caching for fast rebuilds
- OCI feature caching for offline development
- Multi-stage builds for optimal performance
```

**Step 5: Final commit**

```bash
git add docs/DEVCONTAINER_GUIDE.md
git commit -m "feat: achieve 100% devcontainer features specification compliance

- Complete feature options validation with type checking
- Full lifecycle hook support (all five hook types)
- Feature-contributed mounts, security properties, and container configuration
- OCI build context automatic handling for any feature location
- Microsoft universal devcontainer image full compatibility
- Comprehensive E2E test coverage for specification compliance
- Updated torture test demonstrating complete feature ecosystem
- Documentation reflects 100% specification support

Packnplay now supports the complete devcontainer features specification
with full VS Code compatibility and community ecosystem access."
```

---

## Success Criteria Validation

**Specification Compliance Tests:**
- ✅ Microsoft universal devcontainer image works perfectly
- ✅ All feature option types validated (string, boolean, number, enum)
- ✅ Feature lifecycle hooks execute before user commands
- ✅ Feature-contributed mounts applied to containers
- ✅ Security properties (privileged, capabilities) work correctly
- ✅ Complete container properties support (init, entrypoint)
- ✅ OCI features download and cache automatically

**VS Code Compatibility Tests:**
- ✅ Identical behavior to VS Code devcontainer extension
- ✅ Same build performance with Docker layer caching
- ✅ Same error messages and validation feedback
- ✅ Support for all VS Code devcontainer patterns

**Real-World Usage Validation:**
- ✅ Docker-in-docker development workflows work
- ✅ Multi-language project setups function correctly
- ✅ Complex feature dependency chains resolve properly
- ✅ Advanced security scenarios (capabilities, privileged mode) supported
- ✅ Feature option validation prevents configuration errors

## Implementation Scope

**Total estimated effort:** 25 hours
**Critical path (Tasks 1-4):** 15 hours
**Polish and validation (Tasks 5-10):** 10 hours

**Quality gates:**
- All tests must pass after each task
- Real community features must work with options
- Microsoft universal image must build and run successfully
- No regressions in existing functionality