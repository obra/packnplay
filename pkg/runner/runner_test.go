package runner

import (
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

	enhancedArgs, _ := applier.ApplyFeatureProperties(dockerArgs, features, map[string]string{}, ctx)

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