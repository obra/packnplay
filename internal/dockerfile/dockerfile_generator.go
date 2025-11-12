package dockerfile

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/obra/packnplay/pkg/devcontainer"
)

// DockerfileGenerator generates Dockerfiles with devcontainer features
type DockerfileGenerator struct{}

// NewDockerfileGenerator creates a new DockerfileGenerator
func NewDockerfileGenerator() *DockerfileGenerator {
	return &DockerfileGenerator{}
}

// Generate creates a Dockerfile with the specified base image, remote user, and features
// The buildContextPath is the directory used as the Docker build context (typically .devcontainer)
func (g *DockerfileGenerator) Generate(baseImage string, remoteUser string, features []*devcontainer.ResolvedFeature, buildContextPath string) (string, error) {
	var sb strings.Builder

	// FROM statement
	sb.WriteString(fmt.Sprintf("FROM %s\n\n", baseImage))

	// Switch to root for installation
	sb.WriteString("USER root\n\n")

	// Install features
	processor := devcontainer.NewFeatureOptionsProcessor()
	for i, feature := range features {
		sb.WriteString(fmt.Sprintf("# Install feature: %s\n", feature.ID))

		// Process feature options to environment variables
		if feature.Metadata != nil && feature.Metadata.Options != nil {
			envVars := processor.ProcessOptions(feature.Options, feature.Metadata.Options)
			for envName, envValue := range envVars {
				sb.WriteString(fmt.Sprintf("ENV %s=%s\n", envName, envValue))
			}
		}

		// COPY the feature directory into the image so install.sh can reference other files
		// Calculate relative path from build context to feature directory
		relPath, err := filepath.Rel(buildContextPath, feature.InstallPath)
		if err != nil {
			// If we can't compute relative path, try to use the feature as-is
			// This might happen for OCI features in cache
			relPath = filepath.Base(feature.InstallPath)
			if strings.Contains(feature.InstallPath, "oci-cache") {
				relPath = filepath.Join("oci-cache", filepath.Base(feature.InstallPath))
			}
		}

		featureDestPath := fmt.Sprintf("/tmp/devcontainer-features/%d-%s", i, feature.ID)
		sb.WriteString(fmt.Sprintf("COPY %s %s\n", relPath, featureDestPath))

		// Run the install script from its directory so relative paths work
		sb.WriteString(fmt.Sprintf("RUN cd %s && chmod +x install.sh && ./install.sh\n\n", featureDestPath))
	}

	// Switch back to remote user if specified
	if remoteUser != "" {
		sb.WriteString(fmt.Sprintf("USER %s\n", remoteUser))
	}

	return sb.String(), nil
}
