package docker

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Client handles Docker CLI interactions
type Client struct {
	cmd     string
	verbose bool
}

// NewClient creates a new Docker client
func NewClient(verbose bool) (*Client, error) {
	return NewClientWithRuntime("", verbose)
}

// NewClientWithRuntime creates a client with a specific runtime preference
func NewClientWithRuntime(preferredRuntime string, verbose bool) (*Client, error) {
	client := &Client{verbose: verbose}

	var cmd string
	var err error

	if preferredRuntime != "" {
		cmd, err = client.UseSpecificRuntime(preferredRuntime)
	} else {
		cmd, err = client.DetectCLI()
	}

	if err != nil {
		return nil, err
	}
	client.cmd = cmd
	return client, nil
}

// UseSpecificRuntime uses a specific container runtime
func (c *Client) UseSpecificRuntime(runtime string) (string, error) {
	if runtime == "orbstack" {
		// OrbStack uses Docker CLI but with orbstack context
		if _, err := exec.LookPath("docker"); err != nil {
			return "", fmt.Errorf("OrbStack requires docker CLI to be available")
		}

		// Verify OrbStack context is available
		cmd := exec.Command("docker", "context", "ls", "--format", "{{.Name}}")
		output, err := cmd.Output()
		if err != nil {
			return "", fmt.Errorf("failed to check Docker contexts for OrbStack: %w", err)
		}

		contexts := strings.Split(string(output), "\n")
		orbstackFound := false
		for _, ctx := range contexts {
			if strings.TrimSpace(ctx) == "orbstack" {
				orbstackFound = true
				break
			}
		}

		if !orbstackFound {
			return "", fmt.Errorf("OrbStack context not found - is OrbStack running?")
		}

		// Set OrbStack as the active Docker context
		if err := exec.Command("docker", "context", "use", "orbstack").Run(); err != nil {
			return "", fmt.Errorf("failed to switch to OrbStack context: %w", err)
		}

		return "docker", nil
	}

	if _, err := exec.LookPath(runtime); err != nil {
		return "", fmt.Errorf("container runtime '%s' not found in PATH", runtime)
	}
	return runtime, nil
}

// DetectCLI finds the docker command to use
func (c *Client) DetectCLI() (string, error) {
	// Check for DOCKER_CMD environment variable (legacy support)
	if envCmd := os.Getenv("DOCKER_CMD"); envCmd != "" {
		if _, err := exec.LookPath(envCmd); err != nil {
			return "", fmt.Errorf("DOCKER_CMD=%s not found in PATH", envCmd)
		}
		return envCmd, nil
	}

	// Try in order: docker, podman
	// Note: Apple Container support disabled due to incompatibilities
	runtimes := []string{"docker", "podman"}
	for _, runtime := range runtimes {
		if _, err := exec.LookPath(runtime); err == nil {
			return runtime, nil
		}
	}

	return "", fmt.Errorf("no container runtime found (tried: docker, podman)")
}

// Run executes a docker command
func (c *Client) Run(args ...string) (string, error) {
	// Translate Docker commands to Apple Container CLI if needed
	if c.cmd == "container" {
		args = c.translateToAppleContainer(args)
	}

	cmd := exec.Command(c.cmd, args...)

	if c.verbose {
		fmt.Fprintf(os.Stderr, "+ %s %v\n", c.cmd, args)
	}

	output, err := cmd.CombinedOutput()

	if c.verbose && len(output) > 0 {
		fmt.Fprintf(os.Stderr, "%s\n", output)
	}

	return string(output), err
}

// translateToAppleContainer translates Docker CLI args to Apple Container CLI
func (c *Client) translateToAppleContainer(args []string) []string {
	if len(args) == 0 {
		return args
	}

	switch args[0] {
	case "ps":
		// Translate: ps -> ls (list)
		newArgs := []string{"ls"}

		// Apple Container doesn't support --filter or Go template format
		// Remove --filter and --format args, use --format json instead
		for i := 1; i < len(args); i++ {
			if args[i] == "--filter" && i+1 < len(args) {
				// Skip --filter and its value
				i++
				continue
			}
			if args[i] == "--format" && i+1 < len(args) {
				// Skip --format and its template value
				i++
				continue
			}
			newArgs = append(newArgs, args[i])
		}

		// Always use json format for Apple Container
		newArgs = append(newArgs, "--format", "json")
		return newArgs

	case "rm":
		// Translate: rm -> delete
		newArgs := []string{"delete"}
		newArgs = append(newArgs, args[1:]...)
		return newArgs

	case "pull":
		// Translate: pull -> image pull
		newArgs := []string{"image", "pull"}
		newArgs = append(newArgs, args[1:]...)
		return newArgs

	case "build":
		// build stays as build (no translation needed)
		return args

	case "image":
		// image commands need special handling
		if len(args) > 1 && args[1] == "inspect" {
			// image inspect -> images ls with filter by name
			// For now, keep as-is and handle in response parsing
			return args
		}
		return args
	}

	return args
}

// Command returns the docker command being used
func (c *Client) Command() string {
	return c.cmd
}
