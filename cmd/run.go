package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/obra/packnplay/pkg/config"
	"github.com/obra/packnplay/pkg/runner"
	"github.com/spf13/cobra"
)

var (
	runPath         string
	runWorktree     string
	runNoWorktree   bool
	runEnv          []string
	runVerbose      bool
	runRuntime      string
	runConfig       string
	runReconnect    bool
	runPublishPorts []string
	// Credential flags
	runGitCreds *bool
	runSSHCreds *bool
	runGHCreds  *bool
	runGPGCreds *bool
	runNPMCreds *bool
	runAWSCreds *bool
	runAllCreds bool
)

var runCmd = &cobra.Command{
	Use:           "run [flags] [command...]",
	Short:         "Run command in container",
	Long:          `Start a container and execute the specified command inside it.`,
	Args:          cobra.MinimumNArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Ensure credential watcher is running (auto-managed daemon)
		if err := ensureCredentialWatcher(); err != nil {
			return fmt.Errorf("failed to start credential watcher: %w", err)
		}

		// If --runtime specified, we can skip config loading for runtime selection
		// But still need config for credentials
		var cfg *config.Config
		var err error

		if runRuntime != "" {
			// Runtime specified on command line - load config but don't fail if missing runtime
			cfg, err = config.LoadWithoutRuntimeCheck()
			if err != nil {
				// Config doesn't exist - use defaults
				cfg = &config.Config{
					ContainerRuntime: runRuntime,
					DefaultImage:     "ghcr.io/obra/packnplay/devcontainer:latest",
					DefaultCredentials: config.Credentials{
						Git: true,  // Always copy .gitconfig
						SSH: false, // SSH keys are credentials - user choice
						GH:  false, // GitHub auth - user choice
					},
				}
			}
		} else {
			// No runtime flag - load config (will prompt if runtime not set)
			cfg, err = config.Load()
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}
		}

		// Determine which credentials to use (flags override config)
		creds := cfg.DefaultCredentials

		// Check if flags were explicitly set
		if cmd.Flags().Changed("git-creds") {
			creds.Git = *runGitCreds
		}
		if cmd.Flags().Changed("ssh-creds") {
			creds.SSH = *runSSHCreds
		}
		if cmd.Flags().Changed("gh-creds") {
			creds.GH = *runGHCreds
		}
		if cmd.Flags().Changed("gpg-creds") {
			creds.GPG = *runGPGCreds
		}
		if cmd.Flags().Changed("npm-creds") {
			creds.NPM = *runNPMCreds
		}
		if cmd.Flags().Changed("aws-creds") {
			creds.AWS = *runAWSCreds
		}
		if runAllCreds {
			creds.Git = true
			creds.SSH = true
			creds.GH = true
			creds.GPG = true
			creds.NPM = true
			creds.AWS = true
		}

		// Determine which runtime to use (flag > config > detect)
		runtime := runRuntime
		if runtime == "" {
			runtime = cfg.ContainerRuntime
		}

		// Apply environment configuration if specified
		var configEnv []string
		if runConfig != "" {
			if envConfig, exists := cfg.EnvConfigs[runConfig]; exists {
				configEnv = applyEnvConfig(envConfig)
			} else {
				return fmt.Errorf("environment config '%s' not found in config file", runConfig)
			}
		}

		// Determine host path for labels
		hostPath := runPath
		if hostPath == "" {
			var err error
			hostPath, err = os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get working directory: %w", err)
			}
		}
		// Make absolute
		hostPath, err = filepath.Abs(hostPath)
		if err != nil {
			return fmt.Errorf("failed to resolve path: %w", err)
		}

		// Capture original command line for debugging
		launchCommand := strings.Join(os.Args, " ")

		runConfig := &runner.RunConfig{
			Path:           runPath,
			Worktree:       runWorktree,
			NoWorktree:     runNoWorktree,
			Env:            append(runEnv, configEnv...), // Merge user env vars with config env vars
			Verbose:        runVerbose,
			Runtime:        runtime,
			Reconnect:      runReconnect,
			DefaultImage:   cfg.DefaultImage,
			Command:        args,
			Credentials:    creds,
			DefaultEnvVars: cfg.DefaultEnvVars,
			PublishPorts:   runPublishPorts,
			HostPath:       hostPath,
			LaunchCommand:  launchCommand,
		}

		if err := runner.Run(runConfig); err != nil {
			// Print error without extra formatting since our error messages are already well-formatted
			fmt.Fprintln(os.Stderr, err.Error())
			// Return non-nil error to set exit code, but silence Cobra error handling
			os.Exit(1)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(runCmd)

	// Disable flag parsing after first positional arg (the command to run)
	// This allows the command and its args to be passed through without interpretation
	runCmd.Flags().SetInterspersed(false)

	runCmd.Flags().StringVar(&runPath, "path", "", "Project path (default: pwd)")
	runCmd.Flags().StringVar(&runWorktree, "worktree", "", "Worktree name (creates if needed)")
	runCmd.Flags().BoolVar(&runNoWorktree, "no-worktree", false, "Skip worktree, use directory directly")
	runCmd.Flags().StringSliceVar(&runEnv, "env", []string{}, "Additional env vars (KEY=value)")
	runCmd.Flags().StringArrayVarP(&runPublishPorts, "publish", "p", []string{}, "Publish container port(s) to host (format: [hostIP:]hostPort:containerPort[/protocol])")
	runCmd.Flags().StringVar(&runRuntime, "runtime", "", "Container runtime to use (docker/podman/container)")
	runCmd.Flags().StringVar(&runConfig, "config", "", "API config profile (anthropic, z.ai, anthropic-work, claude-personal)")
	runCmd.Flags().BoolVar(&runReconnect, "reconnect", false, "Reconnect to existing container instead of failing")
	runCmd.Flags().BoolVar(&runVerbose, "verbose", false, "Show all docker/git commands")

	// Credential flags (use pointers so we can detect if they were explicitly set)
	runGitCreds = runCmd.Flags().Bool("git-creds", false, "Mount git config (~/.gitconfig)")
	runSSHCreds = runCmd.Flags().Bool("ssh-creds", false, "Mount SSH keys (~/.ssh)")
	runGHCreds = runCmd.Flags().Bool("gh-creds", false, "Mount GitHub CLI credentials")
	runGPGCreds = runCmd.Flags().Bool("gpg-creds", false, "Mount GPG credentials for commit signing")
	runNPMCreds = runCmd.Flags().Bool("npm-creds", false, "Mount npm credentials")
	runAWSCreds = runCmd.Flags().Bool("aws-creds", false, "Mount AWS credentials")
	runCmd.Flags().BoolVar(&runAllCreds, "all-creds", false, "Mount all available credentials")
}

// ensureCredentialWatcher starts the credential sync daemon if not already running
func ensureCredentialWatcher() error {
	// Check if watcher is already running
	if isWatcherRunning() {
		return nil
	}

	// Start watcher in background
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	cmd := exec.Command(executable, "watch-credentials")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true, // Detach from parent process group
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start watcher: %w", err)
	}

	// Let it start up
	time.Sleep(100 * time.Millisecond)
	return nil
}

// isWatcherRunning checks if credential watcher daemon is running
func isWatcherRunning() bool {
	cmd := exec.Command("pgrep", "-f", "packnplay.*watch-credentials")
	err := cmd.Run()
	return err == nil
}

// applyEnvConfig processes environment configuration and returns env var array
func applyEnvConfig(envConfig config.EnvConfig) []string {
	var envVars []string

	for key, value := range envConfig.EnvVars {
		// Substitute ${VAR_NAME} with actual environment variable values
		resolvedValue := expandEnvVars(value)
		envVars = append(envVars, fmt.Sprintf("%s=%s", key, resolvedValue))
	}

	return envVars
}

// expandEnvVars substitutes ${VAR_NAME} with environment variable values
func expandEnvVars(value string) string {
	// Simple variable substitution for ${VAR_NAME} pattern
	result := value

	// Find all ${VAR_NAME} patterns
	for {
		start := strings.Index(result, "${")
		if start == -1 {
			break
		}

		end := strings.Index(result[start:], "}")
		if end == -1 {
			break
		}
		end += start

		// Extract variable name
		varName := result[start+2 : end]

		// Get environment variable value
		envValue := os.Getenv(varName)

		// Replace ${VAR_NAME} with value
		result = result[:start] + envValue + result[end+1:]
	}

	return result
}
