package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/obra/packnplay/pkg/container"
	"github.com/obra/packnplay/pkg/docker"
	"github.com/spf13/cobra"
)

var listVerbose bool

type ContainerInfo struct {
	Names  string `json:"Names"`
	Status string `json:"Status"`
	Labels string `json:"Labels"`
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all packnplay-managed containers",
	Long:  `Display all running containers managed by packnplay.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Initialize Docker client
		dockerClient, err := docker.NewClient(false)
		if err != nil {
			return fmt.Errorf("failed to initialize docker: %w", err)
		}

		// Get all packnplay-managed containers
		output, err := dockerClient.Run(
			"ps",
			"--filter", "label=managed-by=packnplay",
			"--format", "{{json .}}",
		)
		if err != nil {
			return fmt.Errorf("failed to list containers: %w", err)
		}

		if output == "" {
			fmt.Println("No packnplay-managed containers running")
			return nil
		}

		// Docker outputs one JSON object per line
		lines := splitLines(output)

		if listVerbose {
			// Verbose mode: use block format for better readability
			for i, line := range lines {
				if line == "" {
					continue
				}

				var info ContainerInfo
				if err := json.Unmarshal([]byte(line), &info); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to parse container info: %v\n", err)
					continue
				}

				// Parse labels with launch info support
				labels := container.ParseLabels(info.Labels)
				project := container.GetProjectFromLabels(labels)
				worktree := container.GetWorktreeFromLabels(labels)
				hostPath := container.GetHostPathFromLabels(labels)
				launchCommand := container.GetLaunchCommandFromLabels(labels)

				// Handle backward compatibility
				if hostPath == "" {
					hostPath = "N/A"
				}

				// Add spacing between containers
				if i > 0 {
					fmt.Println()
				}

				fmt.Printf("Container: %s\n", info.Names)
				fmt.Printf("  Status: %s\n", info.Status)
				fmt.Printf("  Project: %s\n", project)
				fmt.Printf("  Worktree: %s\n", worktree)
				fmt.Printf("  Host Path: %s\n", hostPath)
				if launchCommand != "" {
					fmt.Printf("  Commandline: %s\n", launchCommand)
				}
			}
		} else {
			// Normal mode: use tabular format
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
			_, _ = fmt.Fprintln(w, "CONTAINER\tSTATUS\tPROJECT\tWORKTREE\tHOST PATH")

			for _, line := range lines {
				if line == "" {
					continue
				}

				var info ContainerInfo
				if err := json.Unmarshal([]byte(line), &info); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to parse container info: %v\n", err)
					continue
				}

				// Parse labels with launch info support
				labels := container.ParseLabels(info.Labels)
				project := container.GetProjectFromLabels(labels)
				worktree := container.GetWorktreeFromLabels(labels)
				hostPath := container.GetHostPathFromLabels(labels)

				// Handle backward compatibility
				if hostPath == "" {
					hostPath = "N/A"
				}

				_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
					info.Names,
					info.Status,
					project,
					worktree,
					hostPath,
				)
			}

			return w.Flush()
		}

		return nil
	},
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().BoolVarP(&listVerbose, "verbose", "v", false, "Show detailed launch information")
}
