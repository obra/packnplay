package runner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/obra/packnplay/pkg/devcontainer"
)

// ImageManager handles container image availability (pull/build).
// Extracted from runner.Run() lines 153-156 and 685-737.
type ImageManager struct {
	client  DockerClient
	verbose bool
}

// DockerClient interface provides the necessary Docker operations for image management.
// The imageName parameter in RunWithProgress is used for progress tracking display.
type DockerClient interface {
	// RunWithProgress executes a Docker command with progress tracking
	RunWithProgress(imageName string, args ...string) error
	// Run executes a Docker command and returns the output
	Run(args ...string) (string, error)
	// Command returns the Docker command being used
	Command() string
}

// NewImageManager creates a new ImageManager with the given Docker client and verbosity setting.
func NewImageManager(client DockerClient, verbose bool) *ImageManager {
	return &ImageManager{
		client:  client,
		verbose: verbose,
	}
}

// EnsureAvailable ensures the container image is available locally.
// If a Dockerfile is specified in devConfig, it builds the image.
// If an image name is specified, it pulls the image if not already present.
// Returns an error if neither image nor Dockerfile is specified.
func (im *ImageManager) EnsureAvailable(devConfig *devcontainer.Config, projectPath string) error {
	// If Dockerfile specified (either DockerFile or Build.Dockerfile), build it
	if devConfig.HasDockerfile() {
		return im.buildImage(devConfig, projectPath)
	}

	// Otherwise pull the image
	if devConfig.Image != "" {
		return im.pullImage(devConfig.Image)
	}

	return fmt.Errorf("no image or dockerfile specified")
}

// pullImage pulls a container image
func (im *ImageManager) pullImage(image string) error {
	// Check if exists locally
	_, err := im.client.Run("image", "inspect", image)
	if err == nil {
		// Image exists locally - nothing to do
		if im.verbose {
			fmt.Fprintf(os.Stderr, "Image %s already exists locally\n", image)
		}
		return nil
	}

	// Need to pull
	if im.verbose {
		fmt.Fprintf(os.Stderr, "Pulling image %s\n", image)
	}

	// CORRECT: Pass imageName as first parameter for progress tracking
	if err := im.client.RunWithProgress(image, "pull", image); err != nil {
		return fmt.Errorf("failed to pull image %s: %w", image, err)
	}
	return nil
}

// buildImage builds a container image from Dockerfile
// Extracted from runner.Run() lines 685-737
//
// SECURITY WARNING: Build args are persisted in image metadata and can be
// inspected with `docker history`. Users should not put secrets in build args.
// For secrets, use containerEnv with ${localEnv:SECRET} variable substitution
// which injects secrets at runtime without persisting them in the image.
func (im *ImageManager) buildImage(devConfig *devcontainer.Config, projectPath string) error {
	projectName := filepath.Base(projectPath)
	// Docker image names must be lowercase
	imageName := fmt.Sprintf("packnplay-%s-devcontainer:latest", strings.ToLower(projectName))

	// Check if already built
	_, err := im.client.Run("image", "inspect", imageName)
	if err == nil {
		// Image already exists
		if im.verbose {
			fmt.Fprintf(os.Stderr, "Image %s already exists\n", imageName)
		}
		return nil
	}

	// Use GetDockerfile() helper which checks both DockerFile and Build.Dockerfile
	dockerfile := devConfig.GetDockerfile()
	if dockerfile == "" {
		return fmt.Errorf("no dockerfile specified")
	}

	// Need to build
	if im.verbose {
		fmt.Fprintf(os.Stderr, "Building image from %s\n", dockerfile)
	}

	var buildArgs []string

	// If Build configuration exists, use it for advanced options
	if devConfig.Build != nil {
		// Make a copy of Build config to modify paths
		buildConfig := *devConfig.Build

		// Adjust paths to be relative to .devcontainer directory
		buildConfig.Dockerfile = filepath.Join(projectPath, ".devcontainer", buildConfig.Dockerfile)
		if buildConfig.Context != "" {
			buildConfig.Context = filepath.Join(projectPath, ".devcontainer", buildConfig.Context)
		} else {
			buildConfig.Context = filepath.Join(projectPath, ".devcontainer")
		}

		// Use BuildConfig to generate docker args
		buildArgs = buildConfig.ToDockerArgs(imageName)
	} else {
		// Simple build without advanced options
		dockerfilePath := filepath.Join(projectPath, ".devcontainer", dockerfile)
		contextPath := filepath.Join(projectPath, ".devcontainer")

		buildArgs = []string{
			"build",
			"-f", dockerfilePath,
			"-t", imageName,
			contextPath,
		}
	}

	// CORRECT: Pass imageName as first parameter for progress tracking
	if err := im.client.RunWithProgress(imageName, buildArgs...); err != nil {
		return fmt.Errorf("failed to build image from %s: %w", dockerfile, err)
	}
	return nil
}
