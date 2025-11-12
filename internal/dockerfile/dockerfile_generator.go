package dockerfile

import (
	"fmt"
	"os"
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
	for _, feature := range features {
		installScript := filepath.Join(feature.InstallPath, "install.sh")
		content, err := os.ReadFile(installScript)
		if err != nil {
			return "", fmt.Errorf("failed to read install script: %w", err)
		}

		// Add RUN command with the install script content
		sb.WriteString(fmt.Sprintf("RUN %s\n\n", string(content)))
	}

	// Switch back to remote user
	sb.WriteString(fmt.Sprintf("USER %s\n", remoteUser))

	return sb.String(), nil
}
