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

	expectedAgents := []string{"claude", "codex", "gemini"}
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

	// Test mounts with vscode user
	mounts := agent.GetMounts("/home/test", "vscode")
	if len(mounts) != 1 {
		t.Errorf("GetMounts() returned %d mounts, want 1", len(mounts))
	}

	if mounts[0].HostPath != "/home/test/.claude" {
		t.Errorf("Mount HostPath = %v, want /home/test/.claude", mounts[0].HostPath)
	}

	if mounts[0].ContainerPath != "/home/vscode/.claude" {
		t.Errorf("Mount ContainerPath = %v, want /home/vscode/.claude", mounts[0].ContainerPath)
	}

	// Test mounts with root user
	rootMounts := agent.GetMounts("/home/test", "root")
	if rootMounts[0].ContainerPath != "/root/.claude" {
		t.Errorf("Mount ContainerPath for root = %v, want /root/.claude", rootMounts[0].ContainerPath)
	}
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
