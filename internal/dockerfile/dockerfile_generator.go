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

		// Execute the install script using sh
		// Note: We use sh -c to execute the script content directly
		scriptStr := strings.ReplaceAll(string(content), `\`, `\\`)
		scriptStr = strings.ReplaceAll(scriptStr, `"`, `\"`)
		scriptStr = strings.ReplaceAll(scriptStr, "\n", " && ")

		// Remove shebang if present
		if strings.HasPrefix(scriptStr, "#!/") {
			parts := strings.SplitN(scriptStr, " && ", 2)
			if len(parts) > 1 {
				scriptStr = parts[1]
			}
		}

		// Remove trailing && if present
		scriptStr = strings.TrimSuffix(strings.TrimSpace(scriptStr), "&&")
		scriptStr = strings.TrimSpace(scriptStr)

		sb.WriteString(fmt.Sprintf("RUN sh -c \"%s\"\n\n", scriptStr))
	}

	// Switch back to remote user if specified
	if remoteUser != "" {
		sb.WriteString(fmt.Sprintf("USER %s\n", remoteUser))
	}

	return sb.String(), nil
}
