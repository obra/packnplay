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
	if len(features) == 0 {
		return fmt.Sprintf("FROM %s\nUSER %s\nWORKDIR /workspace", baseImage, remoteUser), nil
	}

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

// generateMultiStage generates a multi-stage Dockerfile for features outside the build context
func (g *DockerfileGenerator) generateMultiStage(baseImage string, features []*devcontainer.ResolvedFeature, remoteUser string, buildContextPath string) (string, error) {
	var sb strings.Builder

	// Stage 1: Feature preparation
	sb.WriteString("FROM alpine:latest as feature-prep\n\n")

	// Copy all OCI features to staging area
	for i, feature := range features {
		if !strings.HasPrefix(feature.InstallPath, buildContextPath) {
			// OCI feature - needs to be copied from absolute path
			// We'll use a build arg to pass the feature path at build time
			featureDestPath := fmt.Sprintf("/tmp/features/%d-%s", i, feature.ID)
			sb.WriteString(fmt.Sprintf("# Copy OCI feature: %s\n", feature.ID))

			// For OCI features in cache, we need to use COPY with the relative path from build context
			// But since they're outside the build context, we use a workaround:
			// The builder will need to copy these into the build context first
			relPath := filepath.Base(feature.InstallPath)
			if strings.Contains(feature.InstallPath, "oci-cache") {
				relPath = filepath.Join("oci-cache", filepath.Base(feature.InstallPath))
			}
			sb.WriteString(fmt.Sprintf("COPY %s %s\n", relPath, featureDestPath))
		}
	}
	sb.WriteString("\n")

	// Stage 2: Base image with features
	sb.WriteString(fmt.Sprintf("FROM %s as base\n", baseImage))
	sb.WriteString("USER root\n\n")

	// Copy features from prep stage
	sb.WriteString("# Copy features from prep stage\n")
	sb.WriteString("COPY --from=feature-prep /tmp/features /tmp/devcontainer-features\n\n")

	// Install features with options processing
	processor := devcontainer.NewFeatureOptionsProcessor()
	for i, feature := range features {
		sb.WriteString(fmt.Sprintf("# Install feature: %s\n", feature.ID))

		// Add environment variables from options
		if feature.Metadata != nil && feature.Metadata.Options != nil {
			envVars := processor.ProcessOptions(feature.Options, feature.Metadata.Options)
			for envName, envValue := range envVars {
				sb.WriteString(fmt.Sprintf("ENV %s=%s\n", envName, envValue))
			}
		}

		// Add feature-contributed container environment variables
		if feature.Metadata != nil && feature.Metadata.ContainerEnv != nil {
			for envName, envValue := range feature.Metadata.ContainerEnv {
				sb.WriteString(fmt.Sprintf("ENV %s=%s\n", envName, envValue))
			}
		}

		featureDestPath := fmt.Sprintf("/tmp/devcontainer-features/%d-%s", i, feature.ID)
		sb.WriteString(fmt.Sprintf("RUN cd %s && chmod +x install.sh && ./install.sh\n\n", featureDestPath))
	}

	// Switch to user
	if remoteUser != "" {
		sb.WriteString(fmt.Sprintf("USER %s\n", remoteUser))
	}
	sb.WriteString("WORKDIR /workspace\n")

	return sb.String(), nil
}

// generateSingleStage generates a single-stage Dockerfile for features within the build context
func (g *DockerfileGenerator) generateSingleStage(baseImage string, features []*devcontainer.ResolvedFeature, remoteUser string, buildContextPath string) (string, error) {
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

		// Add feature-contributed container environment variables
		if feature.Metadata != nil && feature.Metadata.ContainerEnv != nil {
			for envName, envValue := range feature.Metadata.ContainerEnv {
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
	sb.WriteString("WORKDIR /workspace\n")

	return sb.String(), nil
}
