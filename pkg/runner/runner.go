package runner

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/mattn/go-isatty"
	"github.com/obra/packnplay/pkg/aws"
	"github.com/obra/packnplay/pkg/config"
	"github.com/obra/packnplay/pkg/container"
	"github.com/obra/packnplay/pkg/devcontainer"
	"github.com/obra/packnplay/pkg/docker"
	"github.com/obra/packnplay/pkg/git"
	"github.com/obra/packnplay/pkg/userdetect"
)

type RunConfig struct {
	Path           string
	Worktree       string
	NoWorktree     bool
	Env            []string
	Verbose        bool
	Runtime        string // docker, podman, or container
	Reconnect      bool   // Allow reconnecting to existing containers
	DefaultImage   string // default container image to use
	Command        []string
	Credentials    config.Credentials
	DefaultEnvVars []string // API keys to proxy from host
	PublishPorts   []string // Port mappings to publish to host
	HostPath       string   // Host directory path for the container
	LaunchCommand  string   // Original command line used to launch
}

// ContainerDetails holds detailed information about a running container
type ContainerDetails struct {
	Names         string
	Status        string
	Project       string
	Worktree      string
	HostPath      string
	LaunchCommand string
}

// FeaturePropertiesApplier applies feature metadata to container configuration
type FeaturePropertiesApplier struct{}

// NewFeaturePropertiesApplier creates a new properties applicator
func NewFeaturePropertiesApplier() *FeaturePropertiesApplier {
	return &FeaturePropertiesApplier{}
}

// ApplyFeatureProperties applies feature container properties to Docker args and environment
func (a *FeaturePropertiesApplier) ApplyFeatureProperties(baseArgs []string, features []*devcontainer.ResolvedFeature, baseEnv map[string]string) ([]string, map[string]string) {
	enhancedArgs := make([]string, len(baseArgs))
	copy(enhancedArgs, baseArgs)

	enhancedEnv := make(map[string]string)
	for k, v := range baseEnv {
		enhancedEnv[k] = v
	}

	for _, feature := range features {
		if feature.Metadata == nil {
			continue
		}

		metadata := feature.Metadata

		// Apply security properties
		if metadata.Privileged != nil && *metadata.Privileged {
			enhancedArgs = append(enhancedArgs, "--privileged")
		}

		for _, cap := range metadata.CapAdd {
			enhancedArgs = append(enhancedArgs, "--cap-add="+cap)
		}

		for _, secOpt := range metadata.SecurityOpt {
			enhancedArgs = append(enhancedArgs, "--security-opt="+secOpt)
		}

		// Apply init flag
		if metadata.Init != nil && *metadata.Init {
			enhancedArgs = append(enhancedArgs, "--init")
		}

		// Apply entrypoint
		if len(metadata.Entrypoint) > 0 {
			enhancedArgs = append(enhancedArgs, "--entrypoint="+strings.Join(metadata.Entrypoint, " "))
		}

		// NOTE: ContainerEnv from features is set in the Dockerfile as ENV statements,
		// not as runtime environment variables. This allows variable references like
		// ${PATH} to be properly resolved inside the container.

		// Apply feature-contributed mounts
		for _, mount := range metadata.Mounts {
			mountArg := "--mount=type=" + mount.Type + ",source=" + mount.Source + ",target=" + mount.Target
			enhancedArgs = append(enhancedArgs, mountArg)
		}
	}

	return enhancedArgs, enhancedEnv
}

// getTTYFlags returns appropriate TTY flags for docker commands
// Returns either ["-it"] if we have a TTY, or ["-i"] if we don't
func getTTYFlags() []string {
	if isatty.IsTerminal(os.Stdin.Fd()) || isatty.IsCygwinTerminal(os.Stdin.Fd()) {
		return []string{"-it"} // Interactive + TTY
	}
	return []string{"-i"} // Interactive only (no TTY)
}

func Run(config *RunConfig) error {
	// Step 1: Determine working directory
	workDir := config.Path
	if workDir == "" {
		var err error
		workDir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}
	}

	// Make absolute
	workDir, err := filepath.Abs(workDir)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	// Step 2: Handle worktree logic
	var mountPath string
	var worktreeName string
	var mainRepoGitDir string // Path to main repo's .git directory for mounting

	if config.NoWorktree {
		// Use directory directly
		mountPath = workDir
		worktreeName = "no-worktree"
	} else {
		// Check if git repo
		if !git.IsGitRepo(workDir) {
			if config.Worktree != "" {
				return fmt.Errorf("--worktree specified but %s is not a git repository", workDir)
			}
			// Not a git repo and no worktree flag: use directly
			mountPath = workDir
			worktreeName = "no-worktree"
		} else {
			// Is a git repo
			explicitWorktree := config.Worktree != ""
			if explicitWorktree {
				worktreeName = config.Worktree
			} else {
				// Auto-detect from current branch
				branch, err := git.GetCurrentBranch(workDir)
				if err != nil {
					return fmt.Errorf("failed to get current branch: %w", err)
				}
				worktreeName = branch
			}

			// Check if worktree exists
			exists, err := git.WorktreeExists(worktreeName)
			if err != nil {
				return fmt.Errorf("failed to check worktree: %w", err)
			}

			if exists {
				// Worktree already exists - just use it
				actualPath, err := git.GetWorktreePath(worktreeName)
				if err != nil {
					return fmt.Errorf("failed to get worktree path: %w", err)
				}
				mountPath = actualPath
				if config.Verbose {
					fmt.Fprintf(os.Stderr, "Using existing worktree at %s\n", mountPath)
				}
			} else {
				// Create worktree
				mountPath = git.DetermineWorktreePath(workDir, worktreeName)
				if config.Verbose {
					fmt.Fprintf(os.Stderr, "Creating worktree at %s\n", mountPath)
				}

				if err := git.CreateWorktree(mountPath, worktreeName, config.Verbose); err != nil {
					return fmt.Errorf("failed to create worktree: %w", err)
				}
			}

			// Get main repo's .git directory for mounting
			// Resolve the real path (follow symlinks) to ensure .git paths match
			realWorkDir, err := filepath.EvalSymlinks(workDir)
			if err != nil {
				realWorkDir = workDir // Fallback if can't resolve
			}
			mainRepoGitDir = filepath.Join(realWorkDir, ".git")
		}
	}

	// Step 3: Load devcontainer config
	devConfig, err := devcontainer.LoadConfig(mountPath)
	if err != nil {
		return fmt.Errorf("failed to load devcontainer config: %w", err)
	}
	if devConfig == nil {
		// Use configured default image (supports custom default containers)
		defaultImage := getConfiguredDefaultImage(config)
		devConfig = devcontainer.GetDefaultConfig(defaultImage)
	}

	// Step 4: Initialize container client
	dockerClient, err := docker.NewClientWithRuntime(config.Runtime, config.Verbose)
	if err != nil {
		return fmt.Errorf("failed to initialize container runtime: %w", err)
	}

	// Step 5: Ensure image available using ImageManager service
	imageManager := NewImageManager(dockerClient, config.Verbose)
	if err := imageManager.EnsureAvailable(devConfig, mountPath); err != nil {
		return fmt.Errorf("failed to ensure image: %w", err)
	}

	// Step 5.5: Detect RemoteUser if not specified and we built from Dockerfile or features
	// For built images, the image name is derived from project path
	if devConfig.RemoteUser == "" && (devConfig.HasDockerfile() || len(devConfig.Features) > 0) {
		builtImageName := container.GenerateImageName(workDir)
		userResult, err := userdetect.DetectContainerUser(builtImageName, nil)
		if err != nil {
			// If detection fails, fall back to root
			devConfig.RemoteUser = "root"
			if config.Verbose {
				fmt.Fprintf(os.Stderr, "Warning: failed to detect user from built image, using root: %v\n", err)
			}
		} else {
			devConfig.RemoteUser = userResult.User
			if config.Verbose {
				fmt.Fprintf(os.Stderr, "Detected user %s from built image\n", devConfig.RemoteUser)
			}
		}
	}

	// Step 6: Generate container name and labels
	projectName := filepath.Base(workDir)
	containerName := container.GenerateContainerName(workDir, worktreeName)

	// Use enhanced labels if launch info is available
	var labels map[string]string
	if config.HostPath != "" && config.LaunchCommand != "" {
		labels = container.GenerateLabelsWithLaunchInfo(projectName, worktreeName, config.HostPath, config.LaunchCommand)
	} else {
		labels = container.GenerateLabels(projectName, worktreeName)
	}

	// Step 7: Check if container already running
	if isRunning, err := containerIsRunning(dockerClient, containerName); err != nil {
		return fmt.Errorf("failed to check container status: %w", err)
	} else if isRunning {
		// Container is running - check if user wants to reconnect
		if !config.Reconnect {
			// Get detailed container information
			details, err := getContainerDetails(dockerClient, containerName)
			if err != nil {
				// Fallback to basic error if we can't get details
				return fmt.Errorf("container already running for this worktree (unable to get details: %v)", err)
			}

			// Build command string
			var cmdStr strings.Builder
			for i, arg := range config.Command {
				if i > 0 {
					cmdStr.WriteString(" ")
				}
				if strings.Contains(arg, " ") {
					cmdStr.WriteString(fmt.Sprintf("'%s'", arg))
				} else {
					cmdStr.WriteString(arg)
				}
			}

			// Determine current working directory
			currentDir, err := os.Getwd()
			if err != nil {
				currentDir = ""
			} else {
				// Make absolute for comparison
				currentDir, _ = filepath.Abs(currentDir)
			}

			// Determine if we need worktree flag (if current dir doesn't match container's host path)
			needWorktreeFlag := true
			if currentDir != "" && details.HostPath != "" {
				// If current directory matches container's host path, we don't need --worktree
				needWorktreeFlag = currentDir != details.HostPath
			}

			worktreeFlag := ""
			if needWorktreeFlag && worktreeName != "no-worktree" {
				worktreeFlag = fmt.Sprintf(" --worktree=%s", worktreeName)
			}

			// Build detailed error message
			errorMsg := "container already running for this worktree\n\n"
			errorMsg += "Container Details:\n"
			errorMsg += fmt.Sprintf("  Name: %s\n", details.Names)
			errorMsg += fmt.Sprintf("  Status: %s\n", details.Status)
			errorMsg += fmt.Sprintf("  Project: %s\n", details.Project)
			errorMsg += fmt.Sprintf("  Worktree: %s\n", details.Worktree)
			if details.HostPath != "" {
				errorMsg += fmt.Sprintf("  Host Path: %s\n", details.HostPath)
			}
			if details.LaunchCommand != "" {
				errorMsg += fmt.Sprintf("  Original Command: %s\n", details.LaunchCommand)
			}

			errorMsg += "\nTo run your command in the existing container:\n"
			errorMsg += fmt.Sprintf("  packnplay run%s --reconnect %s\n", worktreeFlag, cmdStr.String())
			errorMsg += "\nTo stop the existing container:\n"
			errorMsg += fmt.Sprintf("  packnplay stop %s", details.Names)

			return fmt.Errorf("%s", errorMsg)
		}

		// User explicitly wants to reconnect
		if config.Verbose {
			fmt.Fprintf(os.Stderr, "Reconnecting to existing container %s\n", containerName)
		}

		// Get container ID
		containerID, err := getContainerID(dockerClient, containerName)
		if err != nil {
			return fmt.Errorf("failed to get container ID: %w", err)
		}

		// Run postStart command if defined (postStart runs every time container is accessed)
		if devConfig.PostStartCommand != nil {
			// Load metadata for lifecycle tracking
			metadata, err := LoadMetadata(containerID)
			if err != nil {
				if config.Verbose {
					fmt.Fprintf(os.Stderr, "Warning: failed to load metadata: %v\n", err)
				}
				metadata = nil
			}

			executor := NewLifecycleExecutor(dockerClient, containerID, devConfig.RemoteUser, config.Verbose, metadata)

			if config.Verbose {
				fmt.Fprintf(os.Stderr, "Running postStartCommand...\n")
			}
			if err := executor.Execute("postStart", devConfig.PostStartCommand); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: postStartCommand failed: %v\n", err)
			}

			// Save metadata after lifecycle execution
			if metadata != nil {
				if err := SaveMetadata(metadata); err != nil {
					if config.Verbose {
						fmt.Fprintf(os.Stderr, "Warning: failed to save metadata: %v\n", err)
					}
				}
			}
		}

		// Exec into existing container
		cmdPath, err := exec.LookPath(dockerClient.Command())
		if err != nil {
			return fmt.Errorf("failed to find docker command: %w", err)
		}

		// Use host path as working directory
		execArgs := []string{filepath.Base(cmdPath), "exec"}
		execArgs = append(execArgs, getTTYFlags()...)

		// Add user flag to exec if remoteUser is specified
		if devConfig.RemoteUser != "" {
			execArgs = append(execArgs, "--user", devConfig.RemoteUser)
		}

		execArgs = append(execArgs, "-w", workDir, containerID)
		execArgs = append(execArgs, config.Command...)

		return syscall.Exec(cmdPath, execArgs, os.Environ())
	}

	// Remove any stopped containers with same name (required for clean start)
	if config.Verbose {
		fmt.Fprintf(os.Stderr, "Checking for stopped container with same name...\n")
	}
	// Try to remove - ignore errors if container doesn't exist
	_, _ = dockerClient.Run("rm", containerName)

	// Step 8: Get current user and detect OS
	currentUser, err := user.Current()
	if err != nil {
		return fmt.Errorf("failed to get current user: %w", err)
	}

	// Check if we're on Linux (idmap only supported on Linux)
	isLinux := os.Getenv("OSTYPE") == "linux-gnu" || fileExists("/proc/version")

	// Note: Credentials are now managed by separate per-container files and watcher daemon
	// No need for Keychain extraction during container startup

	// Build docker run command for background container
	// Apple Container doesn't support -it with -d (detached mode)
	// For detached containers, we don't need TTY flags since they run in background
	isApple := currentUser.HomeDir != "" && !isLinux && dockerClient.Command() == "container"
	var args []string
	if isApple {
		args = []string{"run", "-d", "--sig-proxy=false"}
	} else {
		// For standard Docker, detached mode with signal handling (Microsoft pattern)
		args = []string{"run", "-d", "--sig-proxy=false"}
	}

	// Add labels
	args = append(args, container.LabelsToArgs(labels)...)

	// Add name
	args = append(args, "--name", containerName)

	// Add mounts with or without idmap based on OS
	homeDir := currentUser.HomeDir

	// Mount .claude directory, workspace, and git directory (if worktree)
	// Note: idmap support is kernel/Docker version dependent, so we don't use it for now
	// Just use simple volume mounts and run as container's default user

	// Check if we need container-managed credentials
	hostCredFile := filepath.Join(homeDir, ".claude", ".credentials.json")
	var needsCredentialOverlay bool
	var credentialFile string

	// Check if host has meaningful credentials (not just empty file)
	hostHasCredentials := false
	if fileExists(hostCredFile) {
		if stat, err := os.Stat(hostCredFile); err == nil && stat.Size() >= 20 {
			hostHasCredentials = true
		}
	}

	if !hostHasCredentials {
		needsCredentialOverlay = true
		if config.Verbose {
			if !fileExists(hostCredFile) {
				fmt.Fprintf(os.Stderr, "Host has no .credentials.json, using container-managed credentials\n")
			} else {
				fmt.Fprintf(os.Stderr, "Host .credentials.json is too small (%d bytes), using container-managed credentials\n", getFileSize(hostCredFile))
			}
		}

		var err error
		credentialFile, err = getOrCreateContainerCredentialFile(containerName)
		if err != nil {
			return fmt.Errorf("failed to get credential file: %w", err)
		}
	} else {
		if config.Verbose {
			fmt.Fprintf(os.Stderr, "Using host .credentials.json (%d bytes)\n", getFileSize(hostCredFile))
		}
	}

	// Mount .claude directory
	args = append(args, "-v", fmt.Sprintf("%s/.claude:/home/%s/.claude", homeDir, devConfig.RemoteUser))

	// Overlay mount credential file after .claude directory mount
	if needsCredentialOverlay {
		args = append(args, "-v", fmt.Sprintf("%s:/home/%s/.claude/.credentials.json", credentialFile, devConfig.RemoteUser))
	}

	// Ensure parent directory exists in container by creating it on first run
	// We'll create it after container starts but before exec

	// Mount workspace at host path (preserving absolute paths)
	args = append(args, "-v", fmt.Sprintf("%s:%s", mountPath, mountPath))

	// Mount AI agent config directories using MountBuilder (replaces hardcoded list)
	mountBuilder := NewMountBuilder(homeDir, devConfig.RemoteUser)
	agentMounts := mountBuilder.BuildAgentMounts()
	args = append(args, agentMounts...)

	// If using a worktree, also mount the main repo's .git directory at its real path
	// This allows the worktree's .git file (which contains gitdir: <path>) to resolve correctly
	if mainRepoGitDir != "" {
		args = append(args, "-v", fmt.Sprintf("%s:%s", mainRepoGitDir, mainRepoGitDir))
	}

	// Mount git config
	if config.Credentials.Git {
		gitconfigPath := filepath.Join(homeDir, ".gitconfig")
		if fileExists(gitconfigPath) {
			// Resolve symlinks to get the actual file path
			resolvedPath, err := resolveMountPath(gitconfigPath)
			if err != nil {
				if config.Verbose {
					fmt.Fprintf(os.Stderr, "Warning: failed to resolve .gitconfig symlink: %v\n", err)
				}
				// Fall back to original path if symlink resolution fails
				resolvedPath = gitconfigPath
			}
			args = append(args, "-v", fmt.Sprintf("%s:/home/%s/.gitconfig:ro", resolvedPath, devConfig.RemoteUser))
		}
	}

	// Mount SSH keys
	if config.Credentials.SSH {
		sshPath := filepath.Join(homeDir, ".ssh")
		if fileExists(sshPath) {
			args = append(args, "-v", fmt.Sprintf("%s:/home/%s/.ssh:ro", sshPath, devConfig.RemoteUser))
		}
	}

	// Note: On macOS, gh credentials from Keychain are copied in after container starts
	// On Linux, mount the gh config directory if it exists
	if config.Credentials.GH && isLinux {
		ghConfigPath := filepath.Join(homeDir, ".config", "gh")
		if fileExists(ghConfigPath) {
			args = append(args, "-v", fmt.Sprintf("%s:/home/%s/.config/gh", ghConfigPath, devConfig.RemoteUser))
		}
	}

	if config.Credentials.GPG {
		// Mount .gnupg directory (read-only for security)
		gnupgPath := filepath.Join(homeDir, ".gnupg")
		if fileExists(gnupgPath) {
			args = append(args, "-v", fmt.Sprintf("%s:/home/%s/.gnupg:ro", gnupgPath, devConfig.RemoteUser))
		}
	}

	if config.Credentials.NPM {
		// Mount .npmrc file
		npmrcPath := filepath.Join(homeDir, ".npmrc")
		if fileExists(npmrcPath) {
			// Resolve symlinks to get the actual file path
			resolvedPath, err := resolveMountPath(npmrcPath)
			if err != nil {
				if config.Verbose {
					fmt.Fprintf(os.Stderr, "Warning: failed to resolve .npmrc symlink: %v\n", err)
				}
				// Fall back to original path if symlink resolution fails
				resolvedPath = npmrcPath
			}
			args = append(args, "-v", fmt.Sprintf("%s:/home/%s/.npmrc:ro", resolvedPath, devConfig.RemoteUser))
		}
	}

	// AWS credentials handling
	// Track which credentials we obtained and from where to enforce priority order
	var awsCredentials map[string]string
	var awsCredSource string

	if config.Credentials.AWS {
		awsCredentials = make(map[string]string)

		// Priority 1: Check if static credentials are already set in environment
		if aws.HasStaticCredentials() {
			if config.Verbose {
				fmt.Fprintf(os.Stderr, "Using existing AWS credentials from environment variables\n")
			}
			// Get all AWS_* env vars from host, these will be added later
			for key, value := range aws.GetAWSEnvVars() {
				awsCredentials[key] = value
			}
		} else {
			// Priority 2: Try credential_process if AWS_PROFILE is set
			awsProfile := os.Getenv("AWS_PROFILE")
			if awsProfile != "" {
				credentialProcess, err := aws.ParseAWSConfig(awsProfile)
				if err != nil {
					// Always warn, not just in verbose mode
					fmt.Fprintf(os.Stderr, "Warning: failed to get credential_process for profile '%s': %v\n", awsProfile, err)
				} else {
					if config.Verbose {
						fmt.Fprintf(os.Stderr, "Executing credential_process for profile '%s'\n", awsProfile)
					}
					creds, err := aws.GetCredentialsFromProcess(credentialProcess)
					if err != nil {
						// Always warn, not just in verbose mode
						fmt.Fprintf(os.Stderr, "Warning: credential_process failed: %v\n", err)
					} else {
						awsCredSource = "credential_process"
						if config.Verbose {
							fmt.Fprintf(os.Stderr, "Successfully obtained AWS credentials from credential_process\n")
						}
						// Add credentials from credential_process
						awsCredentials["AWS_ACCESS_KEY_ID"] = creds.AccessKeyID
						awsCredentials["AWS_SECRET_ACCESS_KEY"] = creds.SecretAccessKey
						if creds.SessionToken != "" {
							awsCredentials["AWS_SESSION_TOKEN"] = creds.SessionToken
						}
						// Also include other AWS_* env vars (region, profile, etc.) but not credentials
						for key, value := range aws.GetAWSEnvVars() {
							if key != "AWS_ACCESS_KEY_ID" && key != "AWS_SECRET_ACCESS_KEY" && key != "AWS_SESSION_TOKEN" {
								awsCredentials[key] = value
							}
						}
					}
				}
			} else if config.Verbose {
				fmt.Fprintf(os.Stderr, "No AWS_PROFILE set, skipping credential_process lookup\n")
			}

			// If credential_process didn't work, try getting from environment anyway
			if awsCredSource == "" {
				for key, value := range aws.GetAWSEnvVars() {
					awsCredentials[key] = value
				}
				if len(awsCredentials) > 0 {
					if config.Verbose {
						fmt.Fprintf(os.Stderr, "Using AWS environment variables from host\n")
					}
				}
			}
		}

		// Mount ~/.aws directory if it exists (read-write for SSO token refresh)
		awsPath := filepath.Join(homeDir, ".aws")
		if fileExists(awsPath) {
			// Use read-write mount to allow SSO token refresh and CLI caching
			args = append(args, "-v", fmt.Sprintf("%s:/home/%s/.aws", awsPath, devConfig.RemoteUser))
			if config.Verbose {
				fmt.Fprintf(os.Stderr, "Mounting AWS config directory (read-write for token refresh)\n")
			}
		} else {
			// Always warn if ~/.aws is missing, not just in verbose
			fmt.Fprintf(os.Stderr, "Warning: ~/.aws directory not found, AWS CLI config and SSO cache unavailable\n")
		}
	}

	workingDir := mountPath

	// Set working directory to host path
	args = append(args, "-w", workingDir)

	// Add environment variables
	// Only pass safe terminal/locale variables - nothing else from host
	safeEnvVars := []string{"TERM", "LANG", "LC_ALL", "LC_CTYPE", "LC_MESSAGES", "COLORTERM"}
	for _, key := range safeEnvVars {
		if value := os.Getenv(key); value != "" {
			args = append(args, "-e", fmt.Sprintf("%s=%s", key, value))
		}
	}

	// Set HOME to container user's home directory (don't use host HOME)
	args = append(args, "-e", fmt.Sprintf("HOME=/home/%s", devConfig.RemoteUser))

	// Add IS_SANDBOX marker so tools know they're in a sandbox
	args = append(args, "-e", "IS_SANDBOX=1")

	// Don't set PATH - use container's default PATH to avoid host pollution

	// Add default environment variables (API keys for AI agents)
	for _, envVar := range config.DefaultEnvVars {
		if value := os.Getenv(envVar); value != "" {
			args = append(args, "-e", fmt.Sprintf("%s=%s", envVar, value))
		}
	}

	// Add AWS environment variables BEFORE user-specified env vars
	// This allows users to override AWS credentials if needed with --env flags
	if config.Credentials.AWS && len(awsCredentials) > 0 {
		// Add in deterministic order to avoid randomness from map iteration
		// Priority order: credentials first, then config vars
		credentialKeys := []string{"AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY", "AWS_SESSION_TOKEN"}
		for _, key := range credentialKeys {
			if value, exists := awsCredentials[key]; exists {
				args = append(args, "-e", fmt.Sprintf("%s=%s", key, value))
			}
		}
		// Then add other AWS vars (region, profile, etc.) in sorted order
		var otherKeys []string
		for key := range awsCredentials {
			isCredKey := false
			for _, credKey := range credentialKeys {
				if key == credKey {
					isCredKey = true
					break
				}
			}
			if !isCredKey {
				otherKeys = append(otherKeys, key)
			}
		}
		// Sort for deterministic output
		for _, key := range otherKeys {
			args = append(args, "-e", fmt.Sprintf("%s=%s", key, awsCredentials[key]))
		}
	}

	// Apply environment variables from devcontainer.json with variable substitution
	// This happens AFTER AWS credentials but BEFORE user --env flags
	// so that user flags can override devcontainer vars
	if devConfig.ContainerEnv != nil || devConfig.RemoteEnv != nil {
		// Create substitution context for variable resolution
		ctx := &devcontainer.SubstituteContext{
			LocalWorkspaceFolder:     mountPath,
			ContainerWorkspaceFolder: mountPath,
			LocalEnv:                 getLocalEnvMap(),
			ContainerEnv:             make(map[string]string),
			Labels:                   labels,
		}

		// Get resolved environment variables with substitution applied
		devEnvVars := devConfig.GetResolvedEnvironment(ctx)

		// Add each resolved env var to docker args in deterministic order
		var envKeys []string
		for k := range devEnvVars {
			envKeys = append(envKeys, k)
		}
		sort.Strings(envKeys)
		for _, k := range envKeys {
			args = append(args, "-e", fmt.Sprintf("%s=%s", k, devEnvVars[k]))
		}
	}

	// Add user-specified env vars from --env flags (these can override defaults, AWS, and devcontainer)
	for _, env := range config.Env {
		// Support both --env KEY=value and --env KEY (pass through from host)
		if strings.Contains(env, "=") {
			// KEY=value format - set specific value
			args = append(args, "-e", env)
		} else {
			// KEY format - pass through current value from host
			if value := os.Getenv(env); value != "" {
				args = append(args, "-e", fmt.Sprintf("%s=%s", env, value))
			}
		}
	}

	// Parse and apply port forwarding from devcontainer.json
	// Devcontainer ports are prepended so CLI -p flags take priority
	var publishPorts []string
	if len(devConfig.ForwardPorts) > 0 {
		devPorts, err := devcontainer.ParseForwardPorts(devConfig.ForwardPorts)
		if err != nil {
			return fmt.Errorf("failed to parse forwardPorts from devcontainer.json: %w", err)
		}
		// Prepend devcontainer ports so CLI -p flags (in config.PublishPorts) override
		publishPorts = append(devPorts, config.PublishPorts...)
	} else {
		publishPorts = config.PublishPorts
	}

	// Add port mappings (devcontainer ports + CLI -p flags)
	for _, port := range publishPorts {
		args = append(args, "-p", port)
	}

	// Add custom mounts from devcontainer.json
	for _, mount := range devConfig.Mounts {
		// Create substitution context for variable resolution
		ctx := &devcontainer.SubstituteContext{
			LocalWorkspaceFolder:     mountPath,
			ContainerWorkspaceFolder: mountPath,
			LocalEnv:                 getLocalEnvMap(),
			ContainerEnv:             make(map[string]string),
			Labels:                   labels,
		}

		// Apply variable substitution to mount string
		substitutedMount := devcontainer.Substitute(ctx, mount).(string)

		// Add as Docker mount flag
		args = append(args, "--mount", substitutedMount)
	}

	// Add user from remoteUser setting
	if devConfig.RemoteUser != "" {
		args = append(args, "--user", devConfig.RemoteUser)
	}

	// Add custom Docker run arguments from devcontainer.json
	for _, runArg := range devConfig.RunArgs {
		// Create substitution context for variable resolution
		ctx := &devcontainer.SubstituteContext{
			LocalWorkspaceFolder:     mountPath,
			ContainerWorkspaceFolder: mountPath,
			LocalEnv:                 getLocalEnvMap(),
			ContainerEnv:             make(map[string]string),
			Labels:                   labels,
		}

		// Apply variable substitution to run argument
		substitutedArg := devcontainer.Substitute(ctx, runArg).(string)

		// Add to Docker run command
		args = append(args, substitutedArg)
	}

	// Apply feature-contributed container properties (security options, capabilities, etc.)
	if len(devConfig.Features) > 0 {
		// Resolve features for properties application
		resolver := devcontainer.NewFeatureResolver(filepath.Join(os.TempDir(), "packnplay-features-cache"))

		var resolvedFeatures []*devcontainer.ResolvedFeature
		for reference, options := range devConfig.Features {
			// Convert options from map[string]interface{} if needed
			optionsMap, ok := options.(map[string]interface{})
			if !ok {
				if config.Verbose {
					fmt.Fprintf(os.Stderr, "Warning: invalid options format for feature %s\n", reference)
				}
				continue
			}

			feature, err := resolver.ResolveFeature(reference, optionsMap)
			if err != nil {
				if config.Verbose {
					fmt.Fprintf(os.Stderr, "Warning: failed to resolve feature %s for properties: %v\n", reference, err)
				}
				continue
			}
			resolvedFeatures = append(resolvedFeatures, feature)
		}

		// Apply feature container properties if we successfully resolved features
		if len(resolvedFeatures) > 0 {
			applier := NewFeaturePropertiesApplier()

			// Collect current environment variables that have been added to args
			currentEnv := make(map[string]string)

			// Apply feature properties
			var enhancedEnv map[string]string
			args, enhancedEnv = applier.ApplyFeatureProperties(args, resolvedFeatures, currentEnv)

			// Add feature-contributed environment variables to docker args
			// These go after devcontainer env but can still be overridden by user --env flags
			for k, v := range enhancedEnv {
				args = append(args, "-e", fmt.Sprintf("%s=%s", k, v))
			}
		}
	}

	// Add image
	imageName := devConfig.Image
	if devConfig.HasDockerfile() || len(devConfig.Features) > 0 {
		imageName = container.GenerateImageName(workDir)
	}
	args = append(args, imageName)

	// Add signal-aware command that keeps container alive (Microsoft pattern)
	// This provides graceful shutdown handling for SIGTERM/SIGINT
	args = append(args, "/bin/sh", "-c", `
echo "Container started"
trap "exit 0" 15
while sleep 1 & wait $!; do :; done
`)

	// Step 9: Start container in background
	if config.Verbose {
		fmt.Fprintf(os.Stderr, "Starting container %s\n", containerName)
		fmt.Fprintf(os.Stderr, "Full command: docker %v\n", args)
	}

	containerID, err := dockerClient.Run(args...)
	if err != nil {
		return fmt.Errorf("failed to start container: %w\nDocker output:\n%s", err, containerID)
	}
	containerID = strings.TrimSpace(containerID)

	// Step 10: Ensure host directory structure exists in container
	dirCommands := generateDirectoryCreationCommands(mountPath)
	for _, dirCmd := range dirCommands {
		if config.Verbose {
			fmt.Fprintf(os.Stderr, "Creating directory structure: %v\n", dirCmd)
		}
		_, err := dockerClient.Run(append([]string{"exec", containerID}, dirCmd...)...)
		if err != nil {
			_, _ = dockerClient.Run("rm", "-f", containerID)
			return fmt.Errorf("failed to create directory structure: %w", err)
		}
	}

	// Step 11: Copy config files into container

	// Copy ~/.claude.json
	claudeConfigSrc := filepath.Join(homeDir, ".claude.json")
	if _, err := os.Stat(claudeConfigSrc); err == nil {
		if err := copyFileToContainer(dockerClient, containerID, claudeConfigSrc, fmt.Sprintf("/home/%s/.claude.json", devConfig.RemoteUser), devConfig.RemoteUser, config.Verbose); err != nil {
			_, _ = dockerClient.Run("rm", "-f", containerID)
			return fmt.Errorf("failed to copy .claude.json: %w", err)
		}
	}

	// Copy container-managed credentials into place if needed (host has no .credentials.json)
	hostCredFile2 := filepath.Join(homeDir, ".claude", ".credentials.json")
	if !fileExists(hostCredFile2) {
		if config.Verbose {
			fmt.Fprintf(os.Stderr, "Copying container credentials into .claude directory...\n")
		}
		// Copy from mounted temp location to .claude directory
		_, err = dockerClient.Run("exec", containerID, "cp", "/tmp/packnplay-credentials.json", fmt.Sprintf("/home/%s/.claude/.credentials.json", devConfig.RemoteUser))
		if err != nil && config.Verbose {
			fmt.Fprintf(os.Stderr, "Warning: failed to copy credentials: %v\n", err)
		}
	}

	// Step 11: Execute lifecycle commands from devcontainer.json
	// Commands are tracked: onCreate/postCreate run once, postStart always runs
	// Feature lifecycle commands execute before user commands per specification
	hasLifecycleCommands := devConfig.OnCreateCommand != nil || devConfig.PostCreateCommand != nil || devConfig.PostStartCommand != nil
	hasFeatures := len(devConfig.Features) > 0

	if hasLifecycleCommands || hasFeatures {
		// Load metadata for tracking lifecycle execution
		metadata, err := LoadMetadata(containerID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to load metadata, commands will run: %v\n", err)
			// Continue with nil metadata - commands will run but not be tracked
			metadata = nil
		}

		executor := NewLifecycleExecutor(dockerClient, containerID, devConfig.RemoteUser, config.Verbose, metadata)

		// Resolve features and merge lifecycle commands if features exist
		var mergedCommands map[string]*devcontainer.LifecycleCommand
		if hasFeatures {
			// Resolve features for lifecycle merging
			resolver := devcontainer.NewFeatureResolver(filepath.Join(os.TempDir(), "packnplay-features-cache"))

			var resolvedFeatures []*devcontainer.ResolvedFeature
			for reference, options := range devConfig.Features {
				// Convert options from map[string]interface{} if needed
				optionsMap, ok := options.(map[string]interface{})
				if !ok {
					if config.Verbose {
						fmt.Fprintf(os.Stderr, "Warning: skipping feature %s with invalid options type\n", reference)
					}
					continue
				}

				// Use absolute path if provided, otherwise resolve relative to .devcontainer
				// Don't modify OCI registry references (they contain registry domains)
				fullPath := reference
				if !filepath.IsAbs(reference) && !strings.Contains(reference, "ghcr.io/") && !strings.Contains(reference, "mcr.microsoft.com/") {
					fullPath = filepath.Join(workDir, ".devcontainer", reference)
				}

				feature, err := resolver.ResolveFeature(fullPath, optionsMap)
				if err != nil {
					if config.Verbose {
						fmt.Fprintf(os.Stderr, "Warning: failed to resolve feature %s for lifecycle: %v\n", reference, err)
					}
					continue
				}
				resolvedFeatures = append(resolvedFeatures, feature)
			}

			// Merge feature and user lifecycle commands
			if len(resolvedFeatures) > 0 {
				merger := devcontainer.NewLifecycleMerger()
				userCommands := map[string]*devcontainer.LifecycleCommand{
					"onCreateCommand":   devConfig.OnCreateCommand,
					"postCreateCommand": devConfig.PostCreateCommand,
					"postStartCommand":  devConfig.PostStartCommand,
				}
				mergedCommands = merger.MergeCommands(resolvedFeatures, userCommands)
			}
		}

		// Use merged commands if available, otherwise use user commands directly
		onCreateCmd := devConfig.OnCreateCommand
		postCreateCmd := devConfig.PostCreateCommand
		postStartCmd := devConfig.PostStartCommand

		if mergedCommands != nil {
			if cmd, exists := mergedCommands["onCreateCommand"]; exists {
				onCreateCmd = cmd
			}
			if cmd, exists := mergedCommands["postCreateCommand"]; exists {
				postCreateCmd = cmd
			}
			if cmd, exists := mergedCommands["postStartCommand"]; exists {
				postStartCmd = cmd
			}
		}

		// onCreateCommand - runs once on creation, re-runs if command changes
		if onCreateCmd != nil {
			if config.Verbose {
				fmt.Fprintf(os.Stderr, "Running onCreateCommand...\n")
			}
			if err := executor.Execute("onCreate", onCreateCmd); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: onCreateCommand failed: %v\n", err)
			}
		}

		// postCreateCommand - runs once after creation, re-runs if command changes
		if postCreateCmd != nil {
			if config.Verbose {
				fmt.Fprintf(os.Stderr, "Running postCreateCommand...\n")
			}
			if err := executor.Execute("postCreate", postCreateCmd); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: postCreateCommand failed: %v\n", err)
			}
		}

		// postStartCommand - runs every time container starts
		if postStartCmd != nil {
			if config.Verbose {
				fmt.Fprintf(os.Stderr, "Running postStartCommand...\n")
			}
			if err := executor.Execute("postStart", postStartCmd); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: postStartCommand failed: %v\n", err)
			}
		}

		// Save metadata after lifecycle execution
		if metadata != nil {
			if err := SaveMetadata(metadata); err != nil {
				// Warn but don't fail container startup
				if config.Verbose {
					fmt.Fprintf(os.Stderr, "Warning: failed to save metadata: %v\n", err)
				}
			}
		}
	}

	// Step 12: Exec into container with user's command
	cmdPath, err := exec.LookPath(dockerClient.Command())
	if err != nil {
		return fmt.Errorf("failed to find docker command: %w", err)
	}

	execArgs := []string{filepath.Base(cmdPath), "exec"}
	execArgs = append(execArgs, getTTYFlags()...)

	// Add user flag to exec if remoteUser is specified
	if devConfig.RemoteUser != "" {
		execArgs = append(execArgs, "--user", devConfig.RemoteUser)
	}

	execArgs = append(execArgs, "-w", workingDir, containerID)
	execArgs = append(execArgs, config.Command...)

	// Use syscall.Exec to replace current process
	return syscall.Exec(cmdPath, execArgs, os.Environ())
}

func containerIsRunning(dockerClient *docker.Client, name string) (bool, error) {
	// Apple Container doesn't support --filter, so get all and filter client-side
	isApple := dockerClient.Command() == "container"

	var output string
	var err error

	if isApple {
		output, err = dockerClient.Run("ps", "--format", "json")
	} else {
		output, err = dockerClient.Run("ps", "--filter", fmt.Sprintf("name=%s", name), "--format", "{{.Names}}")
	}

	if err != nil {
		return false, err
	}

	// For Apple Container, output is JSON array
	if isApple {
		// Check if container exists AND is running
		// Look for: "id":"<name>" followed by "status":"running"
		idMatch := fmt.Sprintf(`"id":"%s"`, name)
		if !strings.Contains(output, idMatch) {
			return false, nil
		}

		// Find the container object and check if status is running
		// Simple check: find the id, then check if "status":"running" appears before next "id"
		idIdx := strings.Index(output, idMatch)
		nextIdIdx := strings.Index(output[idIdx+len(idMatch):], `"id":"`)
		var searchRegion string
		if nextIdIdx == -1 {
			searchRegion = output[idIdx:]
		} else {
			searchRegion = output[idIdx : idIdx+len(idMatch)+nextIdIdx]
		}

		return strings.Contains(searchRegion, `"status":"running"`), nil
	}

	// Docker/Podman - simple name matching
	return strings.TrimSpace(output) == name, nil
}

// getContainerDetails gets detailed information about a container
func getContainerDetails(dockerClient *docker.Client, name string) (*ContainerDetails, error) {
	// Get container information using docker ps with JSON format
	output, err := dockerClient.Run(
		"ps",
		"--filter", fmt.Sprintf("name=%s", name),
		"--format", "{{json .}}",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get container details: %w", err)
	}

	if strings.TrimSpace(output) == "" {
		return nil, fmt.Errorf("container not found")
	}

	// Parse the JSON output (should be one line)
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) == 0 {
		return nil, fmt.Errorf("no container information found")
	}

	// Parse the first (and should be only) line
	var containerInfo struct {
		Names  string `json:"Names"`
		Status string `json:"Status"`
		Labels string `json:"Labels"`
	}

	if err := json.Unmarshal([]byte(lines[0]), &containerInfo); err != nil {
		return nil, fmt.Errorf("failed to parse container info: %w", err)
	}

	// Parse labels to get detailed information
	labels := container.ParseLabels(containerInfo.Labels)
	project := container.GetProjectFromLabels(labels)
	worktree := container.GetWorktreeFromLabels(labels)
	hostPath := container.GetHostPathFromLabels(labels)
	launchCommand := container.GetLaunchCommandFromLabels(labels)

	return &ContainerDetails{
		Names:         containerInfo.Names,
		Status:        containerInfo.Status,
		Project:       project,
		Worktree:      worktree,
		HostPath:      hostPath,
		LaunchCommand: launchCommand,
	}, nil
}

// getContainerID gets the container ID by name
func getContainerID(dockerClient *docker.Client, name string) (string, error) {
	isApple := dockerClient.Command() == "container"

	var output string
	var err error

	if isApple {
		output, err = dockerClient.Run("ps", "--format", "json")
	} else {
		output, err = dockerClient.Run("ps", "--filter", fmt.Sprintf("name=%s", name), "--format", "{{.ID}}")
	}

	if err != nil {
		return "", err
	}

	// For Apple Container, search for container with matching ID in JSON
	if isApple {
		idPrefix := fmt.Sprintf(`"id":"%s"`, name)
		if !strings.Contains(output, idPrefix) {
			return "", fmt.Errorf("container not found")
		}
		// Container name IS the ID in Apple Container
		return name, nil
	}

	// Docker/Podman - ID in output
	return strings.TrimSpace(output), nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// getLocalEnvMap returns the current environment as a map
func getLocalEnvMap() map[string]string {
	env := make(map[string]string)
	for _, e := range os.Environ() {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			env[parts[0]] = parts[1]
		}
	}
	return env
}

// resolveMountPath resolves symlinks to get the actual file path for mounting
func resolveMountPath(path string) (string, error) {
	// Use filepath.EvalSymlinks to resolve any symlinks
	resolvedPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", fmt.Errorf("failed to resolve symlinks for %s: %w", path, err)
	}
	return resolvedPath, nil
}

func getFileSize(path string) int64 {
	if stat, err := os.Stat(path); err == nil {
		return stat.Size()
	}
	return 0
}

// generateMountArguments creates Docker mount arguments for host path preservation
func generateMountArguments(config *RunConfig, projectName, worktreeName string) []string {
	var args []string

	// Mount at host path, not /workspace
	hostPath := config.HostPath
	if hostPath == "" {
		hostPath = config.Path
	}

	// Add mount argument: -v hostPath:hostPath
	args = append(args, "-v", fmt.Sprintf("%s:%s", hostPath, hostPath))

	return args
}

// getWorkingDirectory returns the working directory that should be used in the container
func getWorkingDirectory(config *RunConfig) string {
	// Use host path as working directory, not /workspace
	if config.HostPath != "" {
		return config.HostPath
	}
	if config.Path != "" {
		return config.Path
	}
	return "/workspace" // fallback
}

// generateExecArguments creates exec arguments with host path working directory
func generateExecArguments(containerID string, command []string, workingDir string) []string {
	args := []string{"exec"}
	args = append(args, getTTYFlags()...)
	args = append(args, "-w", workingDir, containerID)
	args = append(args, command...)
	return args
}

// generateDirectoryCreationCommands creates commands to set up directory structure in container
func generateDirectoryCreationCommands(hostPath string) [][]string {
	var commands [][]string

	// Create parent directories in container
	parentDir := filepath.Dir(hostPath)
	if parentDir != "/" && parentDir != "." {
		commands = append(commands, []string{"mkdir", "-p", parentDir})
	}

	return commands
}

// NotificationDecision represents whether to notify about a version update
type NotificationDecision struct {
	shouldNotify bool
	reason       string
}

// shouldNotifyAboutVersion determines if user should be notified about version changes
func shouldNotifyAboutVersion(currentDigest, remoteDigest string, lastNotified time.Time, frequency time.Duration) NotificationDecision {
	if currentDigest == remoteDigest {
		return NotificationDecision{false, "same version"}
	}

	if time.Since(lastNotified) < frequency {
		return NotificationDecision{false, "recently notified"}
	}

	return NotificationDecision{true, "new version available"}
}

// ImageVersionInfo holds version information about an image
type ImageVersionInfo struct {
	Digest  string
	Created time.Time
	Size    string
	Tags    []string
}

// AgeString returns a human-readable age string
func (i *ImageVersionInfo) AgeString() string {
	age := time.Since(i.Created)
	if age < time.Hour {
		return "just released"
	}
	if age < 24*time.Hour {
		return fmt.Sprintf("%.0f hours old", age.Hours())
	}
	return fmt.Sprintf("%.0f days old", age.Hours()/24)
}

// ShortDigest returns first 8 characters of digest
func (i *ImageVersionInfo) ShortDigest() string {
	if len(i.Digest) < 8 {
		return i.Digest
	}
	// Skip sha256: prefix if present
	digest := strings.TrimPrefix(i.Digest, "sha256:")
	if len(digest) >= 8 {
		return digest[:8]
	}
	return digest
}

// VersionTracker tracks which image versions have been seen and notified
type VersionTracker struct {
	notifications map[string]time.Time // image:digest -> when notified
}

// NewVersionTracker creates a new version tracker
func NewVersionTracker() *VersionTracker {
	return &VersionTracker{
		notifications: make(map[string]time.Time),
	}
}

// HasNotified returns true if we've notified about this image:digest combination
func (vt *VersionTracker) HasNotified(image, digest string) bool {
	key := image + ":" + digest
	_, exists := vt.notifications[key]
	return exists
}

// MarkNotified marks that we've notified about this image:digest combination
func (vt *VersionTracker) MarkNotified(image, digest string) {
	key := image + ":" + digest
	vt.notifications[key] = time.Now()
}

// getConfiguredDefaultImage returns the user's configured default image or fallback
func getConfiguredDefaultImage(runConfig *RunConfig) string {
	// For now, use the existing DefaultImage field
	// TODO: This will be enhanced to use config.DefaultContainer.Image
	if runConfig.DefaultImage != "" {
		return runConfig.DefaultImage
	}
	return "ghcr.io/obra/packnplay/devcontainer:latest"
}

// getRemoteImageInfo gets version information about an image from the registry
func getRemoteImageInfo(dockerClient *docker.Client, imageName string) (*ImageVersionInfo, error) {
	// Use docker manifest inspect to get remote info without pulling
	_, err := dockerClient.Run("manifest", "inspect", imageName)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect remote image: %w", err)
	}

	// For now, return minimal info (digest would be parsed from manifest)
	return &ImageVersionInfo{
		Digest:  "sha256:remote123", // Simplified for test
		Created: time.Now(),
		Size:    "unknown",
		Tags:    []string{"latest"},
	}, nil
}

// VersionCheckResult holds the result of checking for new versions
type VersionCheckResult struct {
	shouldNotify bool
	localInfo    *ImageVersionInfo
	remoteInfo   *ImageVersionInfo
}

// checkForNewVersion compares local and remote versions and determines if notification needed
func checkForNewVersion(imageName string, localInfo, remoteInfo *ImageVersionInfo, tracker *VersionTracker) VersionCheckResult {
	decision := shouldNotifyAboutVersion(localInfo.Digest, remoteInfo.Digest, time.Time{}, 24*time.Hour)

	return VersionCheckResult{
		shouldNotify: decision.shouldNotify,
		localInfo:    localInfo,
		remoteInfo:   remoteInfo,
	}
}

// formatVersionNotification creates a user-friendly notification message
func formatVersionNotification(imageName string, localInfo, remoteInfo *ImageVersionInfo) string {
	return fmt.Sprintf(`ℹ️  New version available: %s
   Current: %s (%s)
   Latest:  %s (%s)

   To update: packnplay refresh-container`,
		imageName,
		localInfo.ShortDigest(), localInfo.AgeString(),
		remoteInfo.ShortDigest(), remoteInfo.AgeString())
}

// checkAndNotifyAboutUpdates checks for new versions and notifies user if appropriate
func checkAndNotifyAboutUpdates(dockerClient *docker.Client, imageName string, verbose bool) error {
	// Load configuration to check update preferences
	cfg, err := config.LoadOrDefault()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Check if update checking is enabled
	if !config.ShouldCheckForUpdates(cfg.DefaultContainer, time.Time{}) {
		return nil // Checking disabled or too recent
	}

	// Load version tracking data
	trackingPath := config.GetVersionTrackingPath()
	tracking, err := config.LoadVersionTracking(trackingPath)
	if err != nil {
		return fmt.Errorf("failed to load version tracking: %w", err)
	}

	// Only check for updates if it's time to do so
	if !config.ShouldCheckForUpdates(cfg.DefaultContainer, tracking.LastCheck) {
		return nil
	}

	// Check remote registry for new versions (only for default image)
	if imageName != cfg.GetDefaultImage() {
		return nil // Only check updates for default image
	}

	// Get local image info
	localInfo, err := getLocalImageInfo(dockerClient, imageName)
	if err != nil {
		return fmt.Errorf("failed to get local image info: %w", err)
	}

	// Get remote image info
	remoteInfo, err := getRemoteImageInfo(dockerClient, imageName)
	if err != nil {
		return fmt.Errorf("failed to get remote image info: %w", err)
	}

	// Check if we should notify
	result := checkForNewVersion(imageName, localInfo, remoteInfo, NewVersionTracker())

	if result.shouldNotify {
		// Show notification with specific version info
		message := formatVersionNotification(imageName, result.localInfo, result.remoteInfo)
		fmt.Println(message)

		// Mark as notified and update tracking
		tracking.Notifications[imageName] = config.VersionNotification{
			Digest:     remoteInfo.Digest,
			NotifiedAt: time.Now(),
			ImageName:  imageName,
		}
		tracking.LastCheck = time.Now()

		// Save tracking data
		if err := config.SaveVersionTracking(tracking, trackingPath); err != nil && verbose {
			fmt.Fprintf(os.Stderr, "Warning: failed to save tracking data: %v\n", err)
		}
	}

	return nil
}

// getLocalImageInfo gets version information about a local image
func getLocalImageInfo(dockerClient *docker.Client, imageName string) (*ImageVersionInfo, error) {
	// Get local image digest
	output, err := dockerClient.Run("image", "inspect", "--format", "{{.RepoDigests}}", imageName)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect local image: %w", err)
	}

	// Parse creation time
	createdOutput, err := dockerClient.Run("image", "inspect", "--format", "{{.Created}}", imageName)
	if err != nil {
		return nil, fmt.Errorf("failed to get image creation time: %w", err)
	}

	created, err := time.Parse(time.RFC3339, strings.TrimSpace(createdOutput))
	if err != nil {
		created = time.Now() // Fallback
	}

	// Extract digest from RepoDigests output (simplified)
	digest := "sha256:local123" // Simplified for now
	if strings.Contains(output, "sha256:") {
		// Extract actual digest from output
		parts := strings.Split(output, "@sha256:")
		if len(parts) > 1 {
			digest = "sha256:" + strings.Fields(parts[1])[0]
		}
	}

	return &ImageVersionInfo{
		Digest:  digest,
		Created: created,
		Size:    "unknown",
		Tags:    []string{},
	}, nil
}

// getOrCreateContainerCredentialFile manages shared credential file for all containers
func getOrCreateContainerCredentialFile(containerName string) (string, error) {
	// Get credentials directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	xdgDataHome := os.Getenv("XDG_DATA_HOME")
	if xdgDataHome == "" {
		xdgDataHome = filepath.Join(homeDir, ".local", "share")
	}

	// Use persistent shared credential file in XDG data directory
	credentialsDir := filepath.Join(xdgDataHome, "packnplay", "credentials")
	if err := os.MkdirAll(credentialsDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create credentials dir: %w", err)
	}
	credentialFile := filepath.Join(credentialsDir, "claude-credentials.json")

	// If file doesn't exist, initialize it
	if !fileExists(credentialFile) {
		// Try to get initial credentials from keychain (macOS) or copy from host (Linux)
		initialCreds, err := getInitialContainerCredentials()
		if err != nil {
			// Create empty file - user will need to authenticate in container
			if err := os.WriteFile(credentialFile, []byte("{}"), 0600); err != nil {
				return "", fmt.Errorf("failed to create credential file: %w", err)
			}
		} else {
			if err := os.WriteFile(credentialFile, []byte(initialCreds), 0600); err != nil {
				return "", fmt.Errorf("failed to write initial credentials: %w", err)
			}
		}
	}

	return credentialFile, nil
}

// getInitialContainerCredentials gets initial credentials for new containers
func getInitialContainerCredentials() (string, error) {
	// Check if we're on macOS and can get from keychain
	if !fileExists("/proc/version") { // macOS detection
		cmd := exec.Command("security", "find-generic-password",
			"-s", "packnplay-containers-credentials",
			"-a", "packnplay",
			"-w")

		output, err := cmd.Output()
		if err == nil {
			return strings.TrimSpace(string(output)), nil
		}
	} else {
		// Linux: Check if host has .credentials.json we can copy
		homeDir, _ := os.UserHomeDir()
		hostCredFile := filepath.Join(homeDir, ".claude", ".credentials.json")
		if fileExists(hostCredFile) {
			content, err := os.ReadFile(hostCredFile)
			if err == nil {
				return string(content), nil
			}
		}
	}

	return "", fmt.Errorf("no initial credentials available")
}

// copyFileToContainer copies a file into container and fixes ownership
func copyFileToContainer(dockerClient *docker.Client, containerID, srcPath, dstPath, user string, verbose bool) error {
	if verbose {
		fmt.Fprintf(os.Stderr, "Copying %s to container at %s\n", srcPath, dstPath)
	}

	// Check if this is Apple Container (no cp command)
	isApple := dockerClient.Command() == "container"

	if isApple {
		// Apple Container: use exec with base64 to write file
		return copyFileViaExec(dockerClient, containerID, srcPath, dstPath, user, verbose)
	}

	// Docker/Podman: use cp command
	// Ensure parent directory exists in container
	dstDir := filepath.Dir(dstPath)
	output, err := dockerClient.Run("exec", containerID, "mkdir", "-p", dstDir)
	if err != nil {
		return fmt.Errorf("failed to create parent directory %s: %w\nDocker output:\n%s", dstDir, err, output)
	}

	// Copy file
	containerDst := fmt.Sprintf("%s:%s", containerID, dstPath)
	output, err = dockerClient.Run("cp", srcPath, containerDst)
	if err != nil {
		return fmt.Errorf("failed to copy file %s to %s: %w\nDocker output:\n%s", srcPath, dstPath, err, output)
	}

	// Fix ownership (docker cp creates as root)
	// Only chown the specific file, not the entire directory (might contain read-only mounts)
	_, err = dockerClient.Run("exec", "-u", "root", containerID, "chown", fmt.Sprintf("%s:%s", user, user), dstPath)
	if err != nil && verbose {
		fmt.Fprintf(os.Stderr, "Warning: failed to fix ownership: %v\n", err)
	}

	return nil
}

// copyFileViaExec copies a file using a temp directory mount (for Apple Container)
func copyFileViaExec(dockerClient *docker.Client, containerID, srcPath, dstPath, user string, verbose bool) error {
	// Create temp directory for file transfer
	tempDir, err := os.MkdirTemp("", "packnplay-transfer-*")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Copy file to temp directory
	tempFileName := filepath.Base(srcPath)
	tempFilePath := filepath.Join(tempDir, tempFileName)

	content, err := os.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("failed to read source file: %w", err)
	}

	if err := os.WriteFile(tempFilePath, content, 0644); err != nil {
		return fmt.Errorf("failed to write to temp file: %w", err)
	}

	// This function is no longer used for Apple Container
	// Just return error for now
	return fmt.Errorf("file copying not supported for Apple Container")
}
