package runner

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/obra/packnplay/pkg/devcontainer"
	"github.com/stretchr/testify/assert"
)

// TestFeatureMountVariableSubstitution tests that variables in feature mounts are properly substituted
func TestFeatureMountVariableSubstitution(t *testing.T) {
	features := []*devcontainer.ResolvedFeature{
		{
			ID: "test-feature",
			Metadata: &devcontainer.FeatureMetadata{
				Mounts: []devcontainer.Mount{
					{
						Source: "test-volume-${devcontainerId}",
						Target: "/data",
						Type:   "volume",
					},
				},
			},
		},
	}

	applier := NewFeaturePropertiesApplier()
	dockerArgs := []string{"run", "-d", "--name", "test"}

	// Create substitution context with labels that will generate a devcontainerId
	ctx := &devcontainer.SubstituteContext{
		LocalWorkspaceFolder:     "/test/workspace",
		ContainerWorkspaceFolder: "/workspace",
		LocalEnv:                 map[string]string{},
		ContainerEnv:             map[string]string{},
		Labels: map[string]string{
			"project": "test-project",
			"env":     "test",
		},
	}

	enhancedArgs, _, _ := applier.ApplyFeatureProperties(dockerArgs, features, map[string]string{}, ctx)

	// Find the mount argument
	var mountArg string
	for _, arg := range enhancedArgs {
		if strings.HasPrefix(arg, "--mount=") {
			mountArg = arg
			break
		}
	}

	assert.NotEmpty(t, mountArg, "Mount argument should be present")
	assert.NotContains(t, mountArg, "${devcontainerId}", "Variable should be substituted")
	assert.Contains(t, mountArg, "test-volume-", "Mount should have substituted prefix")
	assert.Contains(t, mountArg, "target=/data", "Target should be preserved")
	assert.Contains(t, mountArg, "type=volume", "Type should be preserved")

	t.Logf("Generated mount argument: %s", mountArg)
}

// TestMultipleEntrypoints_Warning tests that a warning is printed when multiple features override entrypoint
func TestMultipleEntrypoints_Warning(t *testing.T) {
	// Create two features that both set entrypoint
	features := []*devcontainer.ResolvedFeature{
		{
			ID: "feature-a",
			Metadata: &devcontainer.FeatureMetadata{
				Entrypoint: []string{"/bin/bash", "-c"},
			},
		},
		{
			ID: "feature-b",
			Metadata: &devcontainer.FeatureMetadata{
				Entrypoint: []string{"/bin/sh", "-c"},
			},
		},
	}

	applier := NewFeaturePropertiesApplier()
	dockerArgs := []string{"run", "-d", "--name", "test"}

	// Create substitution context
	ctx := &devcontainer.SubstituteContext{
		LocalWorkspaceFolder:     "/test/workspace",
		ContainerWorkspaceFolder: "/workspace",
		LocalEnv:                 map[string]string{},
		ContainerEnv:             map[string]string{},
		Labels:                   map[string]string{},
	}

	// Capture stderr to verify warning is printed
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	enhancedArgs, _, _ := applier.ApplyFeatureProperties(dockerArgs, features, map[string]string{}, ctx)

	// Restore stderr and read captured output
	w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	buf.ReadFrom(r)
	stderrOutput := buf.String()

	// Verify warning was printed
	assert.Contains(t, stderrOutput, "Warning: feature 'feature-b' overrides entrypoint from 'feature-a'",
		"Warning should be printed when second feature overrides entrypoint")

	// Verify the last entrypoint is used (Docker uses the last --entrypoint flag)
	var entrypointArg string
	for _, arg := range enhancedArgs {
		if strings.HasPrefix(arg, "--entrypoint=") {
			entrypointArg = arg
		}
	}

	// Docker will use the last --entrypoint flag
	assert.Contains(t, entrypointArg, "/bin/sh", "Last feature's entrypoint should be in the args")

	t.Logf("Captured warning: %s", stderrOutput)
	t.Logf("Final entrypoint arg: %s", entrypointArg)
}

// TestSingleEntrypoint_NoWarning tests that no warning is printed for a single entrypoint
func TestSingleEntrypoint_NoWarning(t *testing.T) {
	features := []*devcontainer.ResolvedFeature{
		{
			ID: "feature-a",
			Metadata: &devcontainer.FeatureMetadata{
				Entrypoint: []string{"/bin/bash", "-c"},
			},
		},
	}

	applier := NewFeaturePropertiesApplier()
	dockerArgs := []string{"run", "-d", "--name", "test"}

	ctx := &devcontainer.SubstituteContext{
		LocalWorkspaceFolder:     "/test/workspace",
		ContainerWorkspaceFolder: "/workspace",
		LocalEnv:                 map[string]string{},
		ContainerEnv:             map[string]string{},
		Labels:                   map[string]string{},
	}

	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	enhancedArgs, _, _ := applier.ApplyFeatureProperties(dockerArgs, features, map[string]string{}, ctx)

	// Restore stderr and read captured output
	w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	buf.ReadFrom(r)
	stderrOutput := buf.String()

	// Verify no warning was printed
	assert.NotContains(t, stderrOutput, "Warning:", "No warning should be printed for single entrypoint")

	// Verify entrypoint is set
	var entrypointArg string
	for _, arg := range enhancedArgs {
		if strings.HasPrefix(arg, "--entrypoint=") {
			entrypointArg = arg
			break
		}
	}

	assert.Contains(t, entrypointArg, "/bin/bash", "Entrypoint should be set")
	t.Logf("No warning output (as expected)")
	t.Logf("Entrypoint arg: %s", entrypointArg)
}