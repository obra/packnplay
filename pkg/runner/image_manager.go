package runner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/obra/packnplay/internal/dockerfile"
	"github.com/obra/packnplay/pkg/container"
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
// If features are specified, it builds the image with features.
// If an image name is specified, it pulls the image if not already present.
// Returns an error if neither image nor Dockerfile is specified.
// Deprecated: Use EnsureAvailableWithLockfile for consistent feature versioning.
func (im *ImageManager) EnsureAvailable(devConfig *devcontainer.Config, projectPath string) error {
	return im.EnsureAvailableWithLockfile(devConfig, projectPath, nil)
}

// EnsureAvailableWithLockfile ensures the container image is available locally with lockfile support.
// If a Dockerfile is specified in devConfig, it builds the image.
// If features are specified, it builds the image with features using locked versions from lockfile.
// If an image name is specified, it pulls the image if not already present.
// Returns an error if neither image nor Dockerfile is specified.
func (im *ImageManager) EnsureAvailableWithLockfile(devConfig *devcontainer.Config, projectPath string, lockfile *devcontainer.LockFile) error {
	// If features are specified, build with features
	if len(devConfig.Features) > 0 {
		return im.buildImageWithLockfile(devConfig, projectPath, lockfile)
	}

	// If Dockerfile specified (either DockerFile or Build.Dockerfile), build it
	if devConfig.HasDockerfile() {
		return im.buildImageWithLockfile(devConfig, projectPath, lockfile)
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
// Deprecated: Use buildImageWithLockfile for consistent feature versioning.
func (im *ImageManager) buildImage(devConfig *devcontainer.Config, projectPath string) error {
	return im.buildImageWithLockfile(devConfig, projectPath, nil)
}

// buildImageWithLockfile builds a container image from Dockerfile with lockfile support
//
// SECURITY WARNING: Build args are persisted in image metadata and can be
// inspected with `docker history`. Users should not put secrets in build args.
// For secrets, use containerEnv with ${localEnv:SECRET} variable substitution
// which injects secrets at runtime without persisting them in the image.
func (im *ImageManager) buildImageWithLockfile(devConfig *devcontainer.Config, projectPath string, lockfile *devcontainer.LockFile) error {
	imageName := container.GenerateImageName(projectPath)

	// Check if already built
	_, err := im.client.Run("image", "inspect", imageName)
	if err == nil {
		// Image already exists
		if im.verbose {
			fmt.Fprintf(os.Stderr, "Image %s already exists\n", imageName)
		}
		return nil
	}

	// Process features if present
	if len(devConfig.Features) > 0 {
		return im.buildWithFeaturesAndLockfile(devConfig, projectPath, imageName, lockfile)
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

// buildWithFeatures builds a container image with devcontainer features
// Deprecated: Use buildWithFeaturesAndLockfile for consistent feature versioning.
func (im *ImageManager) buildWithFeatures(devConfig *devcontainer.Config, projectPath string, imageName string) error {
	return im.buildWithFeaturesAndLockfile(devConfig, projectPath, imageName, nil)
}

// buildWithFeaturesAndLockfile builds a container image with devcontainer features using provided lockfile
func (im *ImageManager) buildWithFeaturesAndLockfile(devConfig *devcontainer.Config, projectPath string, imageName string, lockfile *devcontainer.LockFile) error {
	// If lockfile not provided, try to load it
	// This maintains backward compatibility but the caller should ideally provide it
	if lockfile == nil {
		var err error
		lockfile, err = devcontainer.LoadLockFile(projectPath)
		if err != nil {
			return fmt.Errorf("failed to load lockfile: %w", err)
		}
	}

	// Resolve features
	resolver := devcontainer.NewFeatureResolver(filepath.Join(projectPath, ".devcontainer"), lockfile)
	resolvedFeatures := make(map[string]*devcontainer.ResolvedFeature)

	for featurePath, options := range devConfig.Features {
		optionsMap, ok := options.(map[string]interface{})
		if !ok {
			optionsMap = map[string]interface{}{}
		}

		// Use absolute path if provided, otherwise resolve relative to .devcontainer
		// Don't modify OCI registry references (they contain registry domains) or HTTP(S) URLs
		fullPath := featurePath
		if !filepath.IsAbs(featurePath) &&
			!strings.Contains(featurePath, "ghcr.io/") &&
			!strings.Contains(featurePath, "mcr.microsoft.com/") &&
			!strings.HasPrefix(featurePath, "http://") &&
			!strings.HasPrefix(featurePath, "https://") {
			fullPath = filepath.Join(projectPath, ".devcontainer", featurePath)
		}

		feature, err := resolver.ResolveFeature(fullPath, optionsMap)
		if err != nil {
			return fmt.Errorf("failed to resolve feature %s: %w", featurePath, err)
		}
		resolvedFeatures[feature.ID] = feature
	}

	// Resolve dependencies (using override order if specified)
	orderedFeatures, err := resolver.ResolveFeaturesWithOverride(resolvedFeatures, devConfig.OverrideFeatureInstallOrder)
	if err != nil {
		return fmt.Errorf("failed to resolve feature dependencies: %w", err)
	}

	// Copy remote features (OCI/HTTPS) into build context so Docker can access them
	buildContextPath := filepath.Join(projectPath, ".devcontainer")
	ociCacheDir := filepath.Join(buildContextPath, "oci-cache")

	for _, feature := range orderedFeatures {
		// Check if this is a remote feature (outside the build context)
		if !strings.HasPrefix(feature.InstallPath, buildContextPath) {
			// Copy remote feature into build context
			destDir := filepath.Join(ociCacheDir, filepath.Base(feature.InstallPath))

			// Remove existing cached copy in build context
			os.RemoveAll(destDir)

			// Copy feature directory into build context
			if err := copyDir(feature.InstallPath, destDir); err != nil {
				return fmt.Errorf("failed to copy remote feature %s into build context: %w", feature.ID, err)
			}

			// Update feature's InstallPath to point to the new location in build context
			feature.InstallPath = destDir
		}
	}

	// Generate Dockerfile with features
	generator := dockerfile.NewDockerfileGenerator()
	baseImage := devConfig.Image
	if baseImage == "" {
		baseImage = "ubuntu:22.04"
	}

	dockerfileContent, err := generator.Generate(baseImage, devConfig.RemoteUser, orderedFeatures, buildContextPath)
	if err != nil {
		return fmt.Errorf("failed to generate Dockerfile: %w", err)
	}

	// Write Dockerfile to temporary location
	tempDockerfile := filepath.Join(projectPath, ".devcontainer", "Dockerfile.generated")
	if err := os.WriteFile(tempDockerfile, []byte(dockerfileContent), 0644); err != nil {
		return fmt.Errorf("failed to write generated Dockerfile: %w", err)
	}

	// Build with generated Dockerfile
	contextPath := filepath.Join(projectPath, ".devcontainer")
	buildArgs := []string{
		"build",
		"-f", tempDockerfile,
		"-t", imageName,
		contextPath,
	}

	if err := im.client.RunWithProgress(imageName, buildArgs...); err != nil {
		return fmt.Errorf("failed to build image with features: %w", err)
	}

	// Clean up OCI cache in build context after successful build
	os.RemoveAll(ociCacheDir)

	return nil
}

// copyDir recursively copies a directory from src to dst
func copyDir(src, dst string) error {
	// Get properties of source dir
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	// Create destination directory
	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return err
	}

	// Read all entries in source directory
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

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

// copyFile copies a single file from src to dst
func copyFile(src, dst string) error {
	// Read source file
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	// Get source file permissions
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	// Write destination file with same permissions
	return os.WriteFile(dst, data, srcInfo.Mode())
}
