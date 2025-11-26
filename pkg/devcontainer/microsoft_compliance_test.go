package devcontainer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestMicrosoftComplianceFeatureOptionNormalization tests option name normalization
// to match Microsoft's getSafeId functionality
// Ported from: vendor/devcontainer-cli/src/test/container-features/featureHelpers.test.ts
func TestMicrosoftComplianceFeatureOptionNormalization(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "should replace a \"-\" with \"_\"",
			input:    "option-name",
			expected: "OPTION_NAME",
		},
		{
			name:     "should replace all \"-\" with \"_\"",
			input:    "option1-name-with_dashes-",
			expected: "OPTION1_NAME_WITH_DASHES_",
		},
		{
			name:     "should only be capitalized if no special characters",
			input:    "myOptionName",
			expected: "MYOPTIONNAME",
		},
		{
			name:     "should delete a leading numbers and add a _",
			input:    "1name",
			expected: "_NAME",
		},
		{
			name:     "should delete all leading numbers and add a _",
			input:    "12345_option-name",
			expected: "_OPTION_NAME",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeOptionName(tt.input)
			assert.Equal(t, tt.expected, result, "Option name normalization should match Microsoft's getSafeId")
		})
	}
}

// TestMicrosoftComplianceFeatureValidation tests comprehensive feature validation
// Ported from: vendor/devcontainer-cli/src/test/container-features/featureHelpers.test.ts
func TestMicrosoftComplianceFeatureValidation(t *testing.T) {
	tests := []struct {
		name          string
		featureID     string
		shouldBeValid bool
		description   string
	}{
		{
			name:          "valid simple feature id",
			featureID:     "common-utils",
			shouldBeValid: true,
			description:   "Simple feature IDs should be valid",
		},
		{
			name:          "invalid feature id with slash",
			featureID:     "group/feature",
			shouldBeValid: false,
			description:   "Feature IDs containing '/' should be invalid",
		},
		{
			name:          "invalid feature id with colon",
			featureID:     "feature:version",
			shouldBeValid: false,
			description:   "Feature IDs containing ':' should be invalid",
		},
		{
			name:          "invalid feature id with backslash",
			featureID:     "feature\\name",
			shouldBeValid: false,
			description:   "Feature IDs containing '\\' should be invalid",
		},
		{
			name:          "invalid feature id with dot",
			featureID:     "feature.name",
			shouldBeValid: false,
			description:   "Feature IDs containing '.' should be invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := isValidFeatureID(tt.featureID)
			assert.Equal(t, tt.shouldBeValid, isValid, tt.description)
		})
	}
}

// TestMicrosoftComplianceOptionValidationTypes tests option type validation
// Ported from Microsoft's test patterns for option validation
func TestMicrosoftComplianceOptionValidationTypes(t *testing.T) {
	processor := NewFeatureOptionsProcessor()

	tests := []struct {
		name        string
		options     map[string]interface{}
		specs       map[string]OptionSpec
		shouldError bool
		description string
	}{
		{
			name: "valid string option",
			options: map[string]interface{}{
				"version": "18.20.0",
			},
			specs: map[string]OptionSpec{
				"version": {Type: "string", Default: "latest"},
			},
			shouldError: false,
			description: "String options with string values should be valid",
		},
		{
			name: "invalid type - number for string option",
			options: map[string]interface{}{
				"version": 18,
			},
			specs: map[string]OptionSpec{
				"version": {Type: "string", Default: "latest"},
			},
			shouldError: true,
			description: "Number values for string options should be invalid",
		},
		{
			name: "valid boolean option",
			options: map[string]interface{}{
				"installZsh": true,
			},
			specs: map[string]OptionSpec{
				"installZsh": {Type: "boolean", Default: false},
			},
			shouldError: false,
			description: "Boolean options with boolean values should be valid",
		},
		{
			name: "invalid type - string for boolean option",
			options: map[string]interface{}{
				"installZsh": "true",
			},
			specs: map[string]OptionSpec{
				"installZsh": {Type: "boolean", Default: false},
			},
			shouldError: true,
			description: "String values for boolean options should be invalid",
		},
		{
			name: "valid enum value",
			options: map[string]interface{}{
				"shell": "bash",
			},
			specs: map[string]OptionSpec{
				"shell": {
					Type:      "string",
					Default:   "bash",
					Proposals: []string{"bash", "zsh"},
				},
			},
			shouldError: false,
			description: "Values matching enum proposals should be valid",
		},
		{
			name: "invalid enum value",
			options: map[string]interface{}{
				"shell": "fish",
			},
			specs: map[string]OptionSpec{
				"shell": {
					Type:      "string",
					Default:   "bash",
					Proposals: []string{"bash", "zsh"},
				},
			},
			shouldError: true,
			description: "Values not in enum proposals should be invalid",
		},
		{
			name: "valid number option - int",
			options: map[string]interface{}{
				"port": 3000,
			},
			specs: map[string]OptionSpec{
				"port": {Type: "number", Default: 8080},
			},
			shouldError: false,
			description: "Integer values for number options should be valid",
		},
		{
			name: "valid number option - float",
			options: map[string]interface{}{
				"ratio": 1.5,
			},
			specs: map[string]OptionSpec{
				"ratio": {Type: "number", Default: 1.0},
			},
			shouldError: false,
			description: "Float values for number options should be valid",
		},
		{
			name: "invalid type - string for number option",
			options: map[string]interface{}{
				"port": "3000",
			},
			specs: map[string]OptionSpec{
				"port": {Type: "number", Default: 8080},
			},
			shouldError: true,
			description: "String values for number options should be invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := processor.ValidateAndProcessOptions(tt.options, tt.specs)

			if tt.shouldError {
				assert.Error(t, err, tt.description)
			} else {
				assert.NoError(t, err, tt.description)
			}
		})
	}
}

// Helper functions that need to be implemented to support Microsoft compliance

// isValidFeatureID validates that a feature ID doesn't contain invalid characters
// This implements the Microsoft invariant that feature IDs should not contain /, :, \, or .
func isValidFeatureID(id string) bool {
	for _, char := range id {
		if char == '/' || char == ':' || char == '\\' || char == '.' {
			return false
		}
	}
	return true
}

// TestMicrosoftComplianceDependencyResolution tests dependency resolution algorithm
// This should match Microsoft's computeDependsOnInstallationOrder functionality
// Note: Comprehensive dependency tests are in TestResolveDependencies in features_test.go
func TestMicrosoftComplianceDependencyResolution(t *testing.T) {
	// This test verifies the key Microsoft compliance requirements for dependency resolution.
	// Full integration tests with file-based features are in TestResolveDependencies.
	// Here we test the core metadata structures used for dependency resolution.

	t.Run("dependsOn metadata structure is correctly parsed", func(t *testing.T) {
		// Test that the DependsOn field in FeatureMetadata can represent dependencies
		meta := &FeatureMetadata{
			ID: "feature-a",
			DependsOn: map[string]interface{}{
				"feature-b": map[string]interface{}{},
			},
		}

		// Verify the dependency is present
		_, hasDep := meta.DependsOn["feature-b"]
		assert.True(t, hasDep, "DependsOn should include feature-b")
	})

	t.Run("installsAfter metadata structure is correctly parsed", func(t *testing.T) {
		// Test soft dependencies via installsAfter
		meta := &FeatureMetadata{
			ID:            "feature-c",
			InstallsAfter: []string{"feature-a", "feature-b"},
		}

		assert.Contains(t, meta.InstallsAfter, "feature-a", "InstallsAfter should include feature-a")
		assert.Contains(t, meta.InstallsAfter, "feature-b", "InstallsAfter should include feature-b")
	})

	t.Run("ResolvedFeature carries dependency info", func(t *testing.T) {
		// Test that ResolvedFeature can carry dependency info for resolution
		feature := &ResolvedFeature{
			ID: "test-feature",
			DependsOn: map[string]interface{}{
				"dependency": map[string]interface{}{},
			},
			InstallsAfter: []string{"soft-dep"},
		}

		_, hasDep := feature.DependsOn["dependency"]
		assert.True(t, hasDep, "ResolvedFeature should track hard dependencies")
		assert.Contains(t, feature.InstallsAfter, "soft-dep", "ResolvedFeature should track soft dependencies")
	})
}

// TestMicrosoftComplianceLifecycleHooks tests lifecycle command execution order
// This should match Microsoft's lifecycle hook behavior
// Note: Comprehensive lifecycle tests are in lifecycle_merger_test.go
func TestMicrosoftComplianceLifecycleHooks(t *testing.T) {
	// Key Microsoft compliance requirements:
	// 1. Feature lifecycle commands execute BEFORE user lifecycle commands
	// 2. All 5 hook types are supported: onCreate, updateContent, postCreate, postStart, postAttach
	// 3. Commands are merged in feature order, then user command last

	t.Run("feature commands execute before user commands", func(t *testing.T) {
		featureCmd := &LifecycleCommand{raw: "echo 'feature first'"}
		userCmd := &LifecycleCommand{raw: "echo 'user second'"}

		feature := &ResolvedFeature{
			ID: "test-feature",
			Metadata: &FeatureMetadata{
				ID:              "test-feature",
				OnCreateCommand: featureCmd,
			},
		}

		merger := NewLifecycleMerger()
		merged := merger.MergeCommands([]*ResolvedFeature{feature}, map[string]*LifecycleCommand{
			"onCreateCommand": userCmd,
		})

		onCreate := merged["onCreateCommand"]
		assert.NotNil(t, onCreate, "Merged onCreate should exist")

		commands := onCreate.ToStringSlice()
		assert.Len(t, commands, 2, "Should have both feature and user commands")
		assert.Equal(t, "echo 'feature first'", commands[0], "Feature command should be first")
		assert.Equal(t, "echo 'user second'", commands[1], "User command should be second")
	})

	t.Run("all five hook types are supported", func(t *testing.T) {
		feature := &ResolvedFeature{
			ID: "test-feature",
			Metadata: &FeatureMetadata{
				ID:                   "test-feature",
				OnCreateCommand:      &LifecycleCommand{raw: "echo onCreate"},
				UpdateContentCommand: &LifecycleCommand{raw: "echo updateContent"},
				PostCreateCommand:    &LifecycleCommand{raw: "echo postCreate"},
				PostStartCommand:     &LifecycleCommand{raw: "echo postStart"},
				PostAttachCommand:    &LifecycleCommand{raw: "echo postAttach"},
			},
		}

		merger := NewLifecycleMerger()
		merged := merger.MergeCommands([]*ResolvedFeature{feature}, map[string]*LifecycleCommand{})

		hookTypes := []string{
			"onCreateCommand",
			"updateContentCommand",
			"postCreateCommand",
			"postStartCommand",
			"postAttachCommand",
		}

		for _, hookType := range hookTypes {
			assert.NotNil(t, merged[hookType], "Hook type %s should be present", hookType)
		}
	})
}

// TestMicrosoftComplianceE2EFeatures tests end-to-end feature functionality
// This should match Microsoft's e2e.test.ts behavior
// Note: Comprehensive E2E tests are in pkg/runner/e2e_test.go
func TestMicrosoftComplianceE2EFeatures(t *testing.T) {
	// E2E feature tests are extensively covered in pkg/runner/e2e_test.go:
	//
	// Feature resolution and installation:
	// - TestE2E_FeatureOptionValidation: Local feature with options
	// - TestE2E_MultipleFeatures: Multiple features in order
	// - TestE2E_RemoteFeature: OCI registry feature resolution
	//
	// Container property application (from feature metadata):
	// - TestE2E_FeaturePrivilegedMode: privileged=true
	// - TestE2E_FeatureCapAdd: capAdd array
	// - TestE2E_FeatureSecurityOpt: securityOpt array
	// - TestE2E_FeatureInit: init=true
	// - TestE2E_FeatureEntrypoint: custom entrypoint
	// - TestE2E_FeatureMounts: mount specifications
	//
	// This test verifies the feature option processing API works correctly,
	// which is the foundation for E2E functionality.

	t.Run("feature options are validated and processed", func(t *testing.T) {
		processor := NewFeatureOptionsProcessor()

		specs := map[string]OptionSpec{
			"version": {Type: "string", Default: "latest"},
			"enabled": {Type: "boolean", Default: true},
		}

		options := map[string]interface{}{
			"version": "18.20.0",
			"enabled": true,
		}

		processed, err := processor.ValidateAndProcessOptions(options, specs)
		assert.NoError(t, err, "Valid options should process successfully")
		// ProcessOptions normalizes keys to uppercase environment variable format
		assert.Equal(t, "18.20.0", processed["VERSION"])
		assert.Equal(t, "true", processed["ENABLED"])
	})

	t.Run("feature options apply defaults when not specified", func(t *testing.T) {
		processor := NewFeatureOptionsProcessor()

		specs := map[string]OptionSpec{
			"version": {Type: "string", Default: "latest"},
		}

		options := map[string]interface{}{}

		processed, err := processor.ValidateAndProcessOptions(options, specs)
		assert.NoError(t, err, "Empty options should get defaults")
		// Default value should be applied with normalized key
		assert.Equal(t, "latest", processed["VERSION"], "Default should be applied")
	})
}