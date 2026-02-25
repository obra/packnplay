package agents

import (
	"strings"
	"testing"
)

func TestGetSupportedAgents(t *testing.T) {
	agents := GetSupportedAgents()

	if len(agents) < 3 {
		t.Errorf("GetSupportedAgents() returned %d agents, expected at least 3", len(agents))
	}

	// Check expected agents are present
	agentNames := make(map[string]bool)
	for _, agent := range agents {
		agentNames[agent.Name()] = true
	}

	expectedAgents := []string{"claude", "codex", "gemini", "opencode"}
	for _, expected := range expectedAgents {
		if !agentNames[expected] {
			t.Errorf("Expected agent '%s' not found in supported agents", expected)
		}
	}
}

func TestClaudeAgent(t *testing.T) {
	agent := &ClaudeAgent{}

	if agent.Name() != "claude" {
		t.Errorf("Name() = %v, want claude", agent.Name())
	}

	if agent.ConfigDir() != ".claude" {
		t.Errorf("ConfigDir() = %v, want .claude", agent.ConfigDir())
	}

	if agent.DefaultAPIKeyEnv() != "ANTHROPIC_API_KEY" {
		t.Errorf("DefaultAPIKeyEnv() = %v, want ANTHROPIC_API_KEY", agent.DefaultAPIKeyEnv())
	}

	if !agent.RequiresSpecialHandling() {
		t.Error("RequiresSpecialHandling() = false, want true for Claude")
	}
}

func TestClaudeAgentMount(t *testing.T) {
	agent := &ClaudeAgent{}

	// Always a single mount — symlink handles absolute path resolution instead
	t.Run("macOS-style different paths produces single mount", func(t *testing.T) {
		mounts := agent.GetMounts("/Users/testuser", "vscode")
		if len(mounts) != 1 {
			t.Fatalf("GetMounts() returned %d mounts, want 1", len(mounts))
		}
		if mounts[0].HostPath != "/Users/testuser/.claude" {
			t.Errorf("Mount HostPath = %v, want /Users/testuser/.claude", mounts[0].HostPath)
		}
		if mounts[0].ContainerPath != "/home/vscode/.claude" {
			t.Errorf("Mount ContainerPath = %v, want /home/vscode/.claude", mounts[0].ContainerPath)
		}
		if mounts[0].ReadOnly {
			t.Error("Mount should be read-write")
		}
	})

	t.Run("Linux same-user produces single mount", func(t *testing.T) {
		mounts := agent.GetMounts("/home/vscode", "vscode")
		if len(mounts) != 1 {
			t.Fatalf("GetMounts() returned %d mounts, want 1", len(mounts))
		}
		if mounts[0].ContainerPath != "/home/vscode/.claude" {
			t.Errorf("Mount ContainerPath = %v, want /home/vscode/.claude", mounts[0].ContainerPath)
		}
	})

	t.Run("root user produces single mount at /root/.claude", func(t *testing.T) {
		mounts := agent.GetMounts("/Users/testuser", "root")
		if len(mounts) != 1 {
			t.Fatalf("GetMounts() returned %d mounts, want 1", len(mounts))
		}
		if mounts[0].ContainerPath != "/root/.claude" {
			t.Errorf("Mount ContainerPath = %v, want /root/.claude", mounts[0].ContainerPath)
		}
	})

	t.Run("root-as-root produces single mount", func(t *testing.T) {
		mounts := agent.GetMounts("/root", "root")
		if len(mounts) != 1 {
			t.Fatalf("GetMounts() returned %d mounts, want 1", len(mounts))
		}
		if mounts[0].ContainerPath != "/root/.claude" {
			t.Errorf("Mount ContainerPath = %v, want /root/.claude", mounts[0].ContainerPath)
		}
	})
}

func TestClaudeAgentSetupCommands(t *testing.T) {
	agent := &ClaudeAgent{}

	t.Run("macOS-style: different paths produces symlink command", func(t *testing.T) {
		cmds := agent.GetSetupCommands("/Users/jesse", "vscode")
		if len(cmds) != 1 {
			t.Fatalf("GetSetupCommands() returned %d commands, want 1", len(cmds))
		}
		cmd := cmds[0]
		// Must create parent directory and symlink host home → container home
		if !strings.Contains(cmd, "mkdir -p /Users") {
			t.Errorf("command missing 'mkdir -p /Users': %q", cmd)
		}
		if !strings.Contains(cmd, "ln -sfn /home/vscode /Users/jesse") {
			t.Errorf("command missing symlink creation: %q", cmd)
		}
	})

	t.Run("Linux same-user: identical paths produces no commands", func(t *testing.T) {
		cmds := agent.GetSetupCommands("/home/vscode", "vscode")
		if len(cmds) != 0 {
			t.Errorf("GetSetupCommands() returned %d commands, want 0 for identical paths", len(cmds))
		}
	})

	t.Run("root-as-root: no commands needed", func(t *testing.T) {
		cmds := agent.GetSetupCommands("/root", "root")
		if len(cmds) != 0 {
			t.Errorf("GetSetupCommands() returned %d commands, want 0", len(cmds))
		}
	})

	t.Run("macOS-style root container user: symlinks to /root", func(t *testing.T) {
		cmds := agent.GetSetupCommands("/Users/jesse", "root")
		if len(cmds) != 1 {
			t.Fatalf("GetSetupCommands() returned %d commands, want 1", len(cmds))
		}
		if !strings.Contains(cmds[0], "ln -sfn /root /Users/jesse") {
			t.Errorf("command missing symlink to /root: %q", cmds[0])
		}
	})

	t.Run("Linux different users: symlink needed", func(t *testing.T) {
		cmds := agent.GetSetupCommands("/home/alice", "vscode")
		if len(cmds) != 1 {
			t.Fatalf("GetSetupCommands() returned %d commands, want 1", len(cmds))
		}
		if !strings.Contains(cmds[0], "ln -sfn /home/vscode /home/alice") {
			t.Errorf("command missing symlink: %q", cmds[0])
		}
	})

	t.Run("other agents return no setup commands", func(t *testing.T) {
		for _, agent := range GetSupportedAgents() {
			if agent.Name() == "claude" {
				continue
			}
			cmds := agent.GetSetupCommands("/Users/jesse", "vscode")
			if len(cmds) != 0 {
				t.Errorf("agent %q: GetSetupCommands() returned %d commands, want 0", agent.Name(), len(cmds))
			}
		}
	})
}

func TestCodexAgent(t *testing.T) {
	agent := &CodexAgent{}

	if agent.Name() != "codex" {
		t.Errorf("Name() = %v, want codex", agent.Name())
	}

	if agent.ConfigDir() != ".codex" {
		t.Errorf("ConfigDir() = %v, want .codex", agent.ConfigDir())
	}

	if agent.DefaultAPIKeyEnv() != "OPENAI_API_KEY" {
		t.Errorf("DefaultAPIKeyEnv() = %v, want OPENAI_API_KEY", agent.DefaultAPIKeyEnv())
	}

	if agent.RequiresSpecialHandling() {
		t.Error("RequiresSpecialHandling() = true, want false for Codex")
	}

	// Test mounts with vscode user
	mounts := agent.GetMounts("/home/test", "vscode")
	if len(mounts) != 1 {
		t.Errorf("GetMounts() returned %d mounts, want 1", len(mounts))
	}

	expected := Mount{
		HostPath:      "/home/test/.codex",
		ContainerPath: "/home/vscode/.codex",
		ReadOnly:      false,
	}

	if mounts[0] != expected {
		t.Errorf("GetMounts() = %+v, want %+v", mounts[0], expected)
	}

	// Test with different user
	nodeMounts := agent.GetMounts("/home/test", "node")
	expectedNode := Mount{
		HostPath:      "/home/test/.codex",
		ContainerPath: "/home/node/.codex",
		ReadOnly:      false,
	}

	if nodeMounts[0] != expectedNode {
		t.Errorf("GetMounts() with node user = %+v, want %+v", nodeMounts[0], expectedNode)
	}
}

func TestGeminiAgent(t *testing.T) {
	agent := &GeminiAgent{}

	if agent.Name() != "gemini" {
		t.Errorf("Name() = %v, want gemini", agent.Name())
	}

	if agent.ConfigDir() != ".gemini" {
		t.Errorf("ConfigDir() = %v, want .gemini", agent.ConfigDir())
	}

	if agent.DefaultAPIKeyEnv() != "GEMINI_API_KEY" {
		t.Errorf("DefaultAPIKeyEnv() = %v, want GEMINI_API_KEY", agent.DefaultAPIKeyEnv())
	}

	if agent.RequiresSpecialHandling() {
		t.Error("RequiresSpecialHandling() = true, want false for Gemini")
	}
}

func TestGetDefaultEnvVars(t *testing.T) {
	envVars := GetDefaultEnvVars()

	// Should include key API variables for major AI coding agents
	requiredVars := []string{
		"ANTHROPIC_API_KEY",
		"OPENAI_API_KEY",
		"GEMINI_API_KEY",
		"GH_TOKEN",
		"QWEN_API_KEY",
		"DEEPSEEK_API_KEY",
	}

	envVarMap := make(map[string]bool)
	for _, v := range envVars {
		envVarMap[v] = true
	}

	for _, required := range requiredVars {
		if !envVarMap[required] {
			t.Errorf("Required env var %s not found in result", required)
		}
	}

	// Should have a reasonable number of env vars (not too few, not too many)
	if len(envVars) < 6 {
		t.Errorf("GetDefaultEnvVars() returned only %d vars, expected at least 6", len(envVars))
	}
}

func TestOpenCodeAgent(t *testing.T) {
	agent := &OpenCodeAgent{}

	if agent.Name() != "opencode" {
		t.Errorf("Name() = %v, want opencode", agent.Name())
	}

	if agent.ConfigDir() != ".config/opencode" {
		t.Errorf("ConfigDir() = %v, want .config/opencode", agent.ConfigDir())
	}

	if agent.DefaultAPIKeyEnv() != "OPENCODE_API_KEY" {
		t.Errorf("DefaultAPIKeyEnv() = %v, want OPENCODE_API_KEY", agent.DefaultAPIKeyEnv())
	}

	if agent.RequiresSpecialHandling() {
		t.Error("RequiresSpecialHandling() = true, want false for OpenCode")
	}

	// Test mounts with vscode user
	mounts := agent.GetMounts("/home/test", "vscode")
	if len(mounts) != 1 {
		t.Errorf("GetMounts() returned %d mounts, want 1", len(mounts))
	}

	expected := Mount{
		HostPath:      "/home/test/.config/opencode",
		ContainerPath: "/home/vscode/.config/opencode", 
		ReadOnly:      false,
	}

	if mounts[0] != expected {
		t.Errorf("GetMounts() = %+v, want %+v", mounts[0], expected)
	}
}
