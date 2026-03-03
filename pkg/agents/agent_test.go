package agents

import (
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

func TestClaudeAgentDualMount(t *testing.T) {
	agent := &ClaudeAgent{}

	// macOS-style: host home differs from container home → dual mount
	t.Run("different paths produces dual mount", func(t *testing.T) {
		mounts := agent.GetMounts("/Users/testuser", "vscode")
		if len(mounts) != 2 {
			t.Fatalf("GetMounts() returned %d mounts, want 2", len(mounts))
		}

		// First mount: same-path for absolute plugin path resolution
		if mounts[0].HostPath != "/Users/testuser/.claude" {
			t.Errorf("Mount[0] HostPath = %v, want /Users/testuser/.claude", mounts[0].HostPath)
		}
		if mounts[0].ContainerPath != "/Users/testuser/.claude" {
			t.Errorf("Mount[0] ContainerPath = %v, want /Users/testuser/.claude", mounts[0].ContainerPath)
		}

		// Second mount: container $HOME for Claude Code discovery
		if mounts[1].HostPath != "/Users/testuser/.claude" {
			t.Errorf("Mount[1] HostPath = %v, want /Users/testuser/.claude", mounts[1].HostPath)
		}
		if mounts[1].ContainerPath != "/home/vscode/.claude" {
			t.Errorf("Mount[1] ContainerPath = %v, want /home/vscode/.claude", mounts[1].ContainerPath)
		}

		// Both should be read-write
		if mounts[0].ReadOnly || mounts[1].ReadOnly {
			t.Error("Mounts should be read-write")
		}
	})

	// Linux same-user: identical paths → single mount optimization
	t.Run("identical paths produces single mount", func(t *testing.T) {
		mounts := agent.GetMounts("/home/vscode", "vscode")
		if len(mounts) != 1 {
			t.Fatalf("GetMounts() returned %d mounts, want 1 for identical paths", len(mounts))
		}

		if mounts[0].HostPath != "/home/vscode/.claude" {
			t.Errorf("Mount HostPath = %v, want /home/vscode/.claude", mounts[0].HostPath)
		}
		if mounts[0].ContainerPath != "/home/vscode/.claude" {
			t.Errorf("Mount ContainerPath = %v, want /home/vscode/.claude", mounts[0].ContainerPath)
		}
		if mounts[0].ReadOnly {
			t.Error("Mount should be read-write")
		}
	})

	// Root-as-root: identical paths → single mount optimization
	t.Run("root as root produces single mount", func(t *testing.T) {
		mounts := agent.GetMounts("/root", "root")
		if len(mounts) != 1 {
			t.Fatalf("GetMounts() returned %d mounts, want 1 for root-as-root", len(mounts))
		}

		if mounts[0].HostPath != "/root/.claude" {
			t.Errorf("Mount HostPath = %v, want /root/.claude", mounts[0].HostPath)
		}
		if mounts[0].ContainerPath != "/root/.claude" {
			t.Errorf("Mount ContainerPath = %v, want /root/.claude", mounts[0].ContainerPath)
		}
	})

	// Root user with different host path → dual mount
	t.Run("root user produces dual mount", func(t *testing.T) {
		mounts := agent.GetMounts("/Users/testuser", "root")
		if len(mounts) != 2 {
			t.Fatalf("GetMounts() returned %d mounts, want 2", len(mounts))
		}

		if mounts[0].ContainerPath != "/Users/testuser/.claude" {
			t.Errorf("Mount[0] ContainerPath = %v, want /Users/testuser/.claude", mounts[0].ContainerPath)
		}
		if mounts[1].ContainerPath != "/root/.claude" {
			t.Errorf("Mount[1] ContainerPath = %v, want /root/.claude", mounts[1].ContainerPath)
		}
	})

	// Linux different user → dual mount
	t.Run("linux different user produces dual mount", func(t *testing.T) {
		mounts := agent.GetMounts("/home/alice", "vscode")
		if len(mounts) != 2 {
			t.Fatalf("GetMounts() returned %d mounts, want 2", len(mounts))
		}

		if mounts[0].ContainerPath != "/home/alice/.claude" {
			t.Errorf("Mount[0] ContainerPath = %v, want /home/alice/.claude", mounts[0].ContainerPath)
		}
		if mounts[1].ContainerPath != "/home/vscode/.claude" {
			t.Errorf("Mount[1] ContainerPath = %v, want /home/vscode/.claude", mounts[1].ContainerPath)
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
