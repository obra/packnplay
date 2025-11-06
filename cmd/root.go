package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "packnplay",
	Short: "Launch commands in isolated Docker containers",
	Long: `packnplay runs commands (like Claude Code) inside isolated Docker containers
with automated worktree and dev container management.

Configuration:
  Config file: ~/.config/packnplay/config.json
  Credentials: ~/.local/share/packnplay/credentials/
  Worktrees:   ~/.local/share/packnplay/worktrees/

Default container: ghcr.io/obra/packnplay/devcontainer:latest
  Includes: Node.js, Claude Code, OpenAI Codex, Google Gemini, GitHub CLI,
            GitHub Copilot, Qwen Code, Cursor CLI, Sourcegraph Amp

Supported AI agents: claude, codex, gemini, copilot, qwen, cursor, amp, deepseek`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
