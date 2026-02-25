package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/obra/packnplay/pkg/container"
	"github.com/obra/packnplay/pkg/docker"
	"github.com/spf13/cobra"
)

var (
	stopPath     string
	stopWorktree string
	stopAll      bool
)

var stopCmd = &cobra.Command{
	Use:   "stop [container_name] [flags]",
	Short: "Stop container",
	Long:  `Stop the container by name, or for the specified project/worktree.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Initialize Docker client
		dockerClient, err := docker.NewClient(false)
		if err != nil {
			return fmt.Errorf("failed to initialize docker: %w", err)
		}

		// If --all flag, stop all packnplay-managed containers
		if stopAll {
			return stopAllContainers(dockerClient)
		}

		// If container name provided as argument, use that
		if len(args) > 0 {
			containerName := args[0]
			return stopContainer(dockerClient, containerName)
		}

		// Otherwise, use worktree-based approach
		// Determine working directory
		workDir := stopPath
		if workDir == "" {
			var err error
			workDir, err = os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get working directory: %w", err)
			}
		}

		workDir, err = filepath.Abs(workDir)
		if err != nil {
			return fmt.Errorf("failed to resolve path: %w", err)
		}

		// Determine worktree name
		worktreeName := stopWorktree
		if worktreeName == "" {
			return fmt.Errorf("container name or --worktree flag is required for stop (or use --all)")
		}

		// Generate container name
		containerName := container.GenerateContainerName(workDir, worktreeName)

		// Stop and remove container
		return stopContainer(dockerClient, containerName)
	},
}

func stopContainer(dockerClient *docker.Client, containerName string) error {
	fmt.Printf("Stopping container %s...\n", containerName)
	_, err := dockerClient.Run("stop", containerName)
	if err != nil {
		return fmt.Errorf("failed to stop container: %w", err)
	}

	_, err = dockerClient.Run("rm", containerName)
	if err != nil {
		return fmt.Errorf("failed to remove container: %w", err)
	}

	fmt.Printf("Container %s stopped and removed\n", containerName)
	return nil
}

func stopAllContainers(dockerClient *docker.Client) error {
	// Get all packnplay-managed containers
	output, err := dockerClient.Run("ps", "--filter", "label=managed-by=packnplay", "--format", "{{json .}}")
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	if strings.TrimSpace(output) == "" {
		fmt.Println("No packnplay-managed containers running")
		return nil
	}

	// Parse container names
	lines := strings.Split(strings.TrimSpace(output), "\n")
	var containerNames []string
	for _, line := range lines {
		var info struct {
			Names string `json:"Names"`
		}
		if err := json.Unmarshal([]byte(line), &info); err != nil {
			continue
		}
		containerNames = append(containerNames, info.Names)
	}

	// Stop each container
	for _, name := range containerNames {
		if err := stopContainer(dockerClient, name); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
		}
	}

	fmt.Printf("\nStopped %d container(s)\n", len(containerNames))
	return nil
}

func init() {
	rootCmd.AddCommand(stopCmd)

	stopCmd.Flags().StringVar(&stopPath, "path", "", "Project path (default: pwd)")
	stopCmd.Flags().StringVar(&stopWorktree, "worktree", "", "Worktree name")
	stopCmd.Flags().BoolVarP(&stopAll, "all", "a", false, "Stop all packnplay-managed containers")
}
