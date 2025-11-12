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
func (g *DockerfileGenerator) Generate(baseImage string, remoteUser string, features []*devcontainer.ResolvedFeature) (string, error) {
	var sb strings.Builder

	// FROM statement
	sb.WriteString(fmt.Sprintf("FROM %s\n\n", baseImage))

	// Switch to root for installation
	sb.WriteString("USER root\n\n")

	// Install features
	for i, feature := range features {
		// COPY the feature directory into the image so install.sh can reference other files
		// Use basename for the COPY source (relative to build context which is .devcontainer)
		featureBasename := filepath.Base(feature.InstallPath)
		// If it's in oci-cache, we need the relative path oci-cache/basename
		featureSource := featureBasename
		if strings.Contains(feature.InstallPath, "oci-cache") {
			featureSource = filepath.Join("oci-cache", featureBasename)
		}

		featureDestPath := fmt.Sprintf("/tmp/devcontainer-features/%d-%s", i, feature.ID)
		sb.WriteString(fmt.Sprintf("COPY %s %s\n", featureSource, featureDestPath))

		// Run the install script from its directory so relative paths work
		sb.WriteString(fmt.Sprintf("RUN cd %s && chmod +x install.sh && ./install.sh\n\n", featureDestPath))
	}

	// Switch back to remote user if specified
	if remoteUser != "" {
		sb.WriteString(fmt.Sprintf("USER %s\n", remoteUser))
	}

	return sb.String(), nil
}
