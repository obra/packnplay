package cmd

import (
	"fmt"

	"github.com/obra/packnplay/pkg/config"
	"github.com/obra/packnplay/pkg/docker"
	"github.com/spf13/cobra"
)

var refreshVerbose bool

var refreshCmd = &cobra.Command{
	Use:   "refresh-container",
	Short: "Pull latest version of default container image",
	Long:  `Force pull the latest version of the configured default container image to get updated tools and dependencies.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load config to get the configured default image
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		defaultImage := cfg.GetDefaultImage()

		dockerClient, err := docker.NewClient(refreshVerbose)
		if err != nil {
			return fmt.Errorf("failed to initialize docker: %w", err)
		}

		if refreshVerbose {
			fmt.Printf("Pulling latest version of %s...\n", defaultImage)
		}

		output, err := dockerClient.Run("pull", defaultImage)
		if err != nil {
			return fmt.Errorf("failed to pull image %s: %w\nDocker output:\n%s", defaultImage, err, output)
		}

		if refreshVerbose {
			fmt.Printf("Successfully updated %s\n", defaultImage)
		} else {
			fmt.Printf("Default container updated to latest version\n")
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(refreshCmd)
	refreshCmd.Flags().BoolVarP(&refreshVerbose, "verbose", "v", false, "Show detailed output")
}
