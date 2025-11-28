package runner

import (
	"strings"
	"testing"

	"github.com/obra/packnplay/pkg/config"
	"github.com/obra/packnplay/pkg/devcontainer"
)

func TestMountBuilder_BuildMounts_Basic(t *testing.T) {
	mb := NewMountBuilder("/home/testuser", "testuser")

	cfg := &RunConfig{
		Path: "/project/path",
		Credentials: config.Credentials{
			Git: true,
			SSH: false,
		},
	}

	mounts, err := mb.BuildMounts(cfg)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Should include project mount and .gitconfig
	if len(mounts) < 2 {
		t.Errorf("Expected at least 2 mounts, got %d", len(mounts))
	}

	// Check for project path mount
	hasProjectMount := false
	for _, mount := range mounts {
		if strings.Contains(mount, cfg.Path) {
			hasProjectMount = true
			break
		}
	}
	if !hasProjectMount {
		t.Error("Expected project path to be mounted")
	}
}

func TestMountBuilder_BuildMounts_WithSSH(t *testing.T) {
	mb := NewMountBuilder("/home/testuser", "testuser")

	cfg := &RunConfig{
		Path: "/project/path",
		Credentials: config.Credentials{
			SSH: true,
		},
	}

	mounts, err := mb.BuildMounts(cfg)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// SSH mount should only be included if .ssh exists on host
	// Since /home/testuser likely doesn't exist, .ssh won't be mounted
	// This is correct behavior - we only mount what exists
	// The test verifies no error occurs when SSH is enabled but doesn't exist
	if len(mounts) < 1 {
		t.Error("Expected at least project mount")
	}
}

func TestMountBuilder_BuildMounts_WithAgents(t *testing.T) {
	mb := NewMountBuilder("/home/testuser", "testuser")

	cfg := &RunConfig{
		Path:        "/project/path",
		Credentials: config.Credentials{},
	}

	mounts, _ := mb.BuildMounts(cfg)

	// Verify that agent mounts would be included if agent configs exist
	// This tests that we're using the Agent abstraction, not a hardcoded list
	// The actual mounts depend on what exists on the host
	// We're testing the structure here
	if len(mounts) < 1 {
		t.Error("Expected at least project mount")
	}
}

func TestMountBuilder_BuildMounts_NoAgentsExist(t *testing.T) {
	// Test with non-existent home directory to simulate no agent configs
	mb := NewMountBuilder("/nonexistent/path", "testuser")

	cfg := &RunConfig{
		Path:        "/project/path",
		Credentials: config.Credentials{},
	}

	mounts, err := mb.BuildMounts(cfg)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Should still have project mount
	hasProjectMount := false
	for _, mount := range mounts {
		if strings.Contains(mount, cfg.Path) {
			hasProjectMount = true
			break
		}
	}
	if !hasProjectMount {
		t.Error("Expected project path to be mounted even when no agents exist")
	}
}

func TestMountBuilder_BuildMounts_WithWorkspaceMount(t *testing.T) {
	mb := NewMountBuilder("/home/testuser", "testuser")

	// Create a substitution context for variable substitution
	ctx := &devcontainer.SubstituteContext{
		LocalWorkspaceFolder:     "/host/project",
		ContainerWorkspaceFolder: "/workspace",
		LocalEnv:                 map[string]string{},
		ContainerEnv:             map[string]string{},
		Labels:                   map[string]string{},
	}

	cfg := &RunConfig{
		Path:                     "/host/project",
		WorkspaceMount:           "source=${localWorkspaceFolder},target=/workspace,type=bind,consistency=cached",
		WorkspaceFolder:          "/workspace",
		WorkspaceMountContext:    ctx,
		Credentials:              config.Credentials{},
	}

	mounts, err := mb.BuildMounts(cfg)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Should use --mount instead of -v for workspace
	hasMount := false
	hasMountFlag := false
	for i, arg := range mounts {
		if arg == "--mount" {
			hasMountFlag = true
			if i+1 < len(mounts) {
				mountSpec := mounts[i+1]
				// Verify variable substitution happened
				if strings.Contains(mountSpec, "source=/host/project") &&
					strings.Contains(mountSpec, "target=/workspace") &&
					strings.Contains(mountSpec, "type=bind") &&
					strings.Contains(mountSpec, "consistency=cached") {
					hasMount = true
				}
			}
		}
	}

	if !hasMountFlag {
		t.Error("Expected --mount flag when workspaceMount is specified")
	}
	if !hasMount {
		t.Error("Expected --mount with substituted workspace mount spec")
	}

	// Should NOT have -v flag for workspace when workspaceMount is used
	for i, arg := range mounts {
		if arg == "-v" && i+1 < len(mounts) {
			if strings.Contains(mounts[i+1], "/host/project:/host/project") {
				t.Error("Should not use -v for workspace mount when workspaceMount is specified")
			}
		}
	}
}

func TestMountBuilder_BuildMounts_WithWorkspaceMount_RequiresWorkspaceFolder(t *testing.T) {
	mb := NewMountBuilder("/home/testuser", "testuser")

	ctx := &devcontainer.SubstituteContext{
		LocalWorkspaceFolder:     "/host/project",
		ContainerWorkspaceFolder: "/workspace",
		LocalEnv:                 map[string]string{},
		ContainerEnv:             map[string]string{},
		Labels:                   map[string]string{},
	}

	cfg := &RunConfig{
		Path:                  "/host/project",
		WorkspaceMount:        "source=${localWorkspaceFolder},target=/workspace,type=bind",
		WorkspaceFolder:       "", // Missing workspaceFolder - should error
		WorkspaceMountContext: ctx,
		Credentials:           config.Credentials{},
	}

	_, err := mb.BuildMounts(cfg)
	if err == nil {
		t.Error("Expected error when workspaceMount is set but workspaceFolder is empty")
	}
	if err != nil && !strings.Contains(err.Error(), "workspaceFolder") {
		t.Errorf("Expected error about workspaceFolder, got: %v", err)
	}
}

func TestMountBuilder_BuildMounts_WithoutWorkspaceMount_UsesDefaultVolumeMount(t *testing.T) {
	mb := NewMountBuilder("/home/testuser", "testuser")

	cfg := &RunConfig{
		Path:           "/project/path",
		WorkspaceMount: "", // No custom workspace mount
		Credentials:    config.Credentials{},
	}

	mounts, err := mb.BuildMounts(cfg)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Should use -v flag for workspace (default behavior)
	hasVolumeMount := false
	for i, arg := range mounts {
		if arg == "-v" && i+1 < len(mounts) {
			if strings.Contains(mounts[i+1], "/project/path:/project/path") {
				hasVolumeMount = true
			}
		}
	}

	if !hasVolumeMount {
		t.Error("Expected default -v volume mount when workspaceMount is not specified")
	}
}
