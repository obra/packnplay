package devcontainer

// SubstituteContext holds all variables available for substitution
// Used by the variable substitution engine to resolve ${...} patterns
type SubstituteContext struct {
	// Platform is typically "linux", "darwin", "windows"
	Platform string

	// LocalWorkspaceFolder is the absolute path on the host machine
	LocalWorkspaceFolder string

	// ContainerWorkspaceFolder is the absolute path inside the container
	// Can contain variables that need recursive resolution
	ContainerWorkspaceFolder string

	// LocalEnv contains host environment variables
	LocalEnv map[string]string

	// ContainerEnv contains container environment variables
	// Built from containerEnv and remoteEnv in devcontainer.json
	ContainerEnv map[string]string

	// Labels are Docker labels used to generate devcontainerId
	Labels map[string]string
}
