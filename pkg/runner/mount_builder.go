package runner

import (
	"fmt"
	"path/filepath"

	"github.com/obra/packnplay/pkg/agents"
	"github.com/obra/packnplay/pkg/config"
)

// MountBuilder constructs volume mount arguments for containers.
// It handles project mounts, credential mounts, and AI agent configuration mounts.
type MountBuilder struct {
	hostHomeDir   string
	containerUser string
}

// NewMountBuilder creates a MountBuilder with the specified host home directory
// and container user. The hostHomeDir is used to locate credential and agent
// configuration directories. The containerUser determines the target paths in
// the container.
func NewMountBuilder(hostHomeDir, containerUser string) *MountBuilder {
	return &MountBuilder{
		hostHomeDir:   hostHomeDir,
		containerUser: containerUser,
	}
}

// BuildMounts constructs all volume mount arguments for a container.
// It returns Docker -v flag arguments as a slice of strings.
// Extracted from runner.Run() lines 345-426 to improve testability and maintainability.
//
// Mount order:
//  1. Project directory (read-write)
//  2. .git directory if exists (read-write)
//  3. Credentials (git, ssh, gh, gpg, npm, aws) based on config (read-only)
//  4. AI agent configurations using Agent abstraction (read-write)
func (mb *MountBuilder) BuildMounts(cfg *RunConfig) ([]string, error) {
	var args []string

	// 1. Mount project directory
	projectMount := fmt.Sprintf("%s:%s", cfg.Path, cfg.Path)
	args = append(args, "-v", projectMount)

	// 2. Mount .git directory (if exists)
	gitDir := filepath.Join(cfg.Path, ".git")
	if fileExists(gitDir) {
		args = append(args, "-v", fmt.Sprintf("%s:%s", gitDir, gitDir))
	}

	// 3. Mount credentials based on config
	credMounts := mb.buildCredentialMounts(cfg.Credentials)
	args = append(args, credMounts...)

	// 4. Mount agent configs using Agent abstraction (NOT hardcoded list)
	agentMounts := mb.buildAgentMounts()
	args = append(args, agentMounts...)

	return args, nil
}

// buildCredentialMounts constructs credential volume mounts
func (mb *MountBuilder) buildCredentialMounts(creds config.Credentials) []string {
	var args []string

	if creds.Git {
		gitconfig := filepath.Join(mb.hostHomeDir, ".gitconfig")
		if fileExists(gitconfig) {
			target := fmt.Sprintf("/home/%s/.gitconfig", mb.containerUser)
			args = append(args, "-v", fmt.Sprintf("%s:%s:ro", gitconfig, target))
		}
	}

	if creds.SSH {
		sshDir := filepath.Join(mb.hostHomeDir, ".ssh")
		if fileExists(sshDir) {
			target := fmt.Sprintf("/home/%s/.ssh", mb.containerUser)
			args = append(args, "-v", fmt.Sprintf("%s:%s:ro", sshDir, target))
		}
	}

	if creds.GH {
		ghConfigPath := filepath.Join(mb.hostHomeDir, ".config", "gh")
		if fileExists(ghConfigPath) {
			target := fmt.Sprintf("/home/%s/.config/gh", mb.containerUser)
			args = append(args, "-v", fmt.Sprintf("%s:%s", ghConfigPath, target))
		}
	}

	if creds.GPG {
		gnupgPath := filepath.Join(mb.hostHomeDir, ".gnupg")
		if fileExists(gnupgPath) {
			target := fmt.Sprintf("/home/%s/.gnupg", mb.containerUser)
			args = append(args, "-v", fmt.Sprintf("%s:%s:ro", gnupgPath, target))
		}
	}

	if creds.NPM {
		npmrcPath := filepath.Join(mb.hostHomeDir, ".npmrc")
		if fileExists(npmrcPath) {
			target := fmt.Sprintf("/home/%s/.npmrc", mb.containerUser)
			args = append(args, "-v", fmt.Sprintf("%s:%s:ro", npmrcPath, target))
		}
	}

	if creds.AWS {
		awsDir := filepath.Join(mb.hostHomeDir, ".aws")
		if fileExists(awsDir) {
			target := fmt.Sprintf("/home/%s/.aws", mb.containerUser)
			args = append(args, "-v", fmt.Sprintf("%s:%s:ro", awsDir, target))
		}
	}

	return args
}

// buildAgentMounts constructs agent config directory mounts
// Uses the Agent abstraction instead of hardcoded list (fixes architectural smell)
func (mb *MountBuilder) buildAgentMounts() []string {
	var args []string

	for _, agent := range agents.GetSupportedAgents() {
		// Check if agent config exists on host
		agentPath := filepath.Join(mb.hostHomeDir, agent.ConfigDir())
		if !fileExists(agentPath) {
			continue // Agent config doesn't exist on host
		}

		// Get mounts from agent
		mounts := agent.GetMounts(mb.hostHomeDir, mb.containerUser)
		for _, mount := range mounts {
			// Convert Mount struct to Docker -v format
			// IMPORTANT: Mount struct has no String() method, convert manually
			mountStr := fmt.Sprintf("%s:%s", mount.HostPath, mount.ContainerPath)
			if mount.ReadOnly {
				mountStr += ":ro"
			}
			args = append(args, "-v", mountStr)
		}
	}

	return args
}
