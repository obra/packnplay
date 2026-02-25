package compose

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/obra/packnplay/pkg/docker"
)

// Runner handles Docker Compose orchestration
type Runner struct {
	workDir      string
	composeFiles []string
	service      string
	runServices  []string
	dockerClient *docker.Client
	verbose      bool
}

// NewRunner creates a new Docker Compose runner
func NewRunner(workDir string, composeFiles []string, service string, runServices []string, dockerClient *docker.Client, verbose bool) *Runner {
	return &Runner{
		workDir:      workDir,
		composeFiles: composeFiles,
		service:      service,
		runServices:  runServices,
		dockerClient: dockerClient,
		verbose:      verbose,
	}
}

// Up starts the Docker Compose services
// Returns the container ID of the specified service
func (r *Runner) Up() (string, error) {
	// Build docker compose up command
	args := []string{"compose"}

	// Add compose file(s)
	for _, f := range r.composeFiles {
		args = append(args, "-f", f)
	}

	// Add up command with detached mode
	args = append(args, "up", "-d")

	// Add specific services if runServices is specified
	if len(r.runServices) > 0 {
		args = append(args, r.runServices...)
	}

	// Execute compose up
	cmd := exec.Command(r.dockerClient.Command(), args...)
	cmd.Dir = r.workDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if r.verbose {
		fmt.Fprintf(os.Stderr, "+ %s %v\n", r.dockerClient.Command(), args)
	}

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("docker compose up failed: %w", err)
	}

	// Get container ID for the service
	return r.GetServiceContainerID()
}

// GetServiceContainerID returns the container ID for the specified service
func (r *Runner) GetServiceContainerID() (string, error) {
	// Use docker compose ps to get container ID for service
	args := []string{"compose"}
	for _, f := range r.composeFiles {
		args = append(args, "-f", f)
	}
	args = append(args, "ps", "-q", r.service)

	cmd := exec.Command(r.dockerClient.Command(), args...)
	cmd.Dir = r.workDir

	if r.verbose {
		fmt.Fprintf(os.Stderr, "+ %s %v\n", r.dockerClient.Command(), args)
		cmd.Stderr = os.Stderr
	}

	// Use Output() instead of CombinedOutput() to avoid capturing stderr warnings
	// (docker compose prints deprecation warnings to stderr that would pollute the container ID)
	output, err := cmd.Output()
	if err != nil {
		// If there's an error, try to get stderr for debugging
		// Note: exitErr.Stderr is only populated when cmd.Stderr is nil.
		// In verbose mode, stderr streams to terminal, so exitErr.Stderr will be empty.
		if exitErr, ok := err.(*exec.ExitError); ok && len(exitErr.Stderr) > 0 {
			return "", fmt.Errorf("failed to get service container ID: %w\nStderr: %s", err, exitErr.Stderr)
		}
		return "", fmt.Errorf("failed to get service container ID: %w", err)
	}

	containerID := strings.TrimSpace(string(output))
	if containerID == "" {
		return "", fmt.Errorf("service %s not found in compose setup (may not be running)", r.service)
	}

	return containerID, nil
}

// Down stops and removes the Docker Compose services
func (r *Runner) Down() error {
	args := []string{"compose"}
	for _, f := range r.composeFiles {
		args = append(args, "-f", f)
	}
	args = append(args, "down", "-v") // -v removes volumes

	cmd := exec.Command(r.dockerClient.Command(), args...)
	cmd.Dir = r.workDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if r.verbose {
		fmt.Fprintf(os.Stderr, "+ %s %v\n", r.dockerClient.Command(), args)
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker compose down failed: %w", err)
	}

	return nil
}

// ValidateComposeFiles checks that all compose files exist
func ValidateComposeFiles(workDir string, composeFiles []string) error {
	for _, f := range composeFiles {
		absPath := f
		if !filepath.IsAbs(f) {
			absPath = filepath.Join(workDir, f)
		}
		if _, err := os.Stat(absPath); err != nil {
			return fmt.Errorf("compose file not found: %s", f)
		}
	}
	return nil
}
