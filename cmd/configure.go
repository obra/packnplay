package cmd

import (
	"fmt"

	"github.com/obra/packnplay/pkg/config"
	"github.com/spf13/cobra"
)

var configureVerbose bool

var configureCmd = &cobra.Command{
	Use:   "configure",
	Short: "Edit packnplay configuration",
	Long: `Interactive configuration editor for packnplay settings.

Safely edits configuration while preserving any existing settings
not shown in the user interface (custom env configs, advanced settings, etc.).

Shows all configuration options in a logical flow:
  1. Container runtime selection
  2. Default credential mounting preferences
  3. Default container image and update settings

This command preserves all existing configuration values not displayed
in the interactive forms, ensuring manual edits and advanced settings
are never lost during configuration updates.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runInteractiveConfigure(configureVerbose)
	},
}

func runInteractiveConfigure(verbose bool) error {
	configPath := config.GetConfigPath()

	if verbose {
		fmt.Printf("Editing config: %s\n", configPath)
	}

	// Load existing config to show current values
	existingConfig, err := config.LoadExistingOrEmpty(configPath)
	if err != nil {
		return fmt.Errorf("failed to load existing config: %w", err)
	}

	// Run complete configuration flow
	return configureAll(existingConfig, configPath, verbose)
}

// configureAll implements the complete configuration flow
func configureAll(existing *config.Config, configPath string, verbose bool) error {
	return config.RunInteractiveConfiguration(existing, configPath, verbose)
}

func init() {
	rootCmd.AddCommand(configureCmd)
	configureCmd.Flags().BoolVarP(&configureVerbose, "verbose", "v", false, "Show detailed output")
}
