package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/mattn/go-isatty"
	"github.com/obra/packnplay/pkg/container"
	"github.com/obra/packnplay/pkg/devcontainer"
	"github.com/obra/packnplay/pkg/docker"
	"github.com/spf13/cobra"
)

var (
	attachPath     string
	attachWorktree string
)

// getTTYFlags returns appropriate TTY flags for docker commands
// Returns either ["-it"] if we have a TTY, or ["-i"] if we don't
func getTTYFlags() []string {
	if isatty.IsTerminal(os.Stdin.Fd()) || isatty.IsCygwinTerminal(os.Stdin.Fd()) {
		return []string{"-it"} // Interactive + TTY
	}
	return []string{"-i"} // Interactive only (no TTY)
}

var attachCmd = &cobra.Command{
	Use:   "attach [flags]",
	Short: "Attach to running container",
	Long:  `Attach to an existing running container with an interactive shell.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Determine working directory
		workDir := attachPath
		if workDir == "" {
			var err error
			workDir, err = os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get working directory: %w", err)
			}
		}

		workDir, err := filepath.Abs(workDir)
		if err != nil {
			return fmt.Errorf("failed to resolve path: %w", err)
		}

		// Determine worktree name
		worktreeName := attachWorktree
		if worktreeName == "" {
			return fmt.Errorf("--worktree flag is required for attach")
		}

		// Generate container name
		containerName := container.GenerateContainerName(workDir, worktreeName)

		// Initialize Docker client
		dockerClient, err := docker.NewClient(false)
		if err != nil {
			return fmt.Errorf("failed to initialize docker: %w", err)
		}

		// Check if container is running
		output, err := dockerClient.Run("ps", "--filter", fmt.Sprintf("name=%s", containerName), "--format", "{{.Names}}")
		if err != nil {
			return fmt.Errorf("failed to check container status: %w", err)
		}

		if strings.TrimSpace(output) != containerName {
			return fmt.Errorf("no running container found for worktree '%s'", worktreeName)
		}

		// Run postAttachCommand if configured
		devConfig, err := devcontainer.LoadConfig(workDir)
		if err == nil && devConfig != nil && devConfig.PostAttachCommand != nil {
			fmt.Fprintf(os.Stderr, "Running postAttachCommand...\n")

			// Get all commands (handles string, array, and object formats)
			commands := devConfig.PostAttachCommand.ToStringSlice()

			for _, cmdStr := range commands {
				if cmdStr == "" {
					continue
				}
				_, err := dockerClient.Run("exec", containerName, "/bin/sh", "-c", cmdStr)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Warning: postAttachCommand failed: %v\n", err)
				}
			}
		}

		// Execute docker exec with interactive shell
		cmdPath, err := exec.LookPath(dockerClient.Command())
		if err != nil {
			return fmt.Errorf("failed to find docker command: %w", err)
		}

		argv := []string{filepath.Base(cmdPath), "exec"}
		argv = append(argv, getTTYFlags()...)
		argv = append(argv, containerName, "/bin/bash")

		return syscall.Exec(cmdPath, argv, os.Environ())
	},
}

func init() {
	rootCmd.AddCommand(attachCmd)

	attachCmd.Flags().StringVar(&attachPath, "path", "", "Project path (default: pwd)")
	attachCmd.Flags().StringVar(&attachWorktree, "worktree", "", "Worktree name")
}
