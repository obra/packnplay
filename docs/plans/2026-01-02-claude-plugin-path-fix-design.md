# Claude Code Plugin Path Fix Design

## Problem

Claude Code plugins and skills don't work when running via `packnplay run claude` because `installed_plugins.json` contains hardcoded absolute host paths that don't exist inside the container.

**Example:**
```json
{
  "plugins": {
    "superpowers@superpowers-marketplace": [{
      "installPath": "/Users/myuser/.claude/plugins/cache/superpowers-marketplace/superpowers/4.0.3"
    }]
  }
}
```

Currently, packnplay mounts `~/.claude` to a different path:
- Host: `/Users/myuser/.claude`
- Container: `/home/{remoteUser}/.claude`

When Claude Code reads `installed_plugins.json`, it tries to access `/Users/myuser/.claude/plugins/cache/...` which doesn't exist in the container.

## Solution: Dual Mount

Mount `~/.claude` at **both** paths:
1. Same as host path (for absolute path resolution in plugin files)
2. At container `$HOME/.claude` (for Claude Code config discovery)

Docker fully supports mounting the same source to multiple targets.

## Implementation

**File:** `pkg/agents/agent.go`

**Change `ClaudeAgent.GetMounts()`:**

```go
func (c *ClaudeAgent) GetMounts(hostHomeDir string, containerUser string) []Mount {
    containerHomeDir := "/root"
    if containerUser != "root" {
        containerHomeDir = "/home/" + containerUser
    }

    hostClaudePath := filepath.Join(hostHomeDir, ".claude")
    containerClaudePath := filepath.Join(containerHomeDir, ".claude")

    // If paths are already the same, only need one mount
    if hostClaudePath == containerClaudePath {
        return []Mount{
            {
                HostPath:      hostClaudePath,
                ContainerPath: containerClaudePath,
                ReadOnly:      false,
            },
        }
    }

    // Dual mount: same-path for absolute references + $HOME path for Claude discovery
    return []Mount{
        {
            HostPath:      hostClaudePath,
            ContainerPath: hostClaudePath,  // Same as host (for plugin absolute paths)
            ReadOnly:      false,
        },
        {
            HostPath:      hostClaudePath,
            ContainerPath: containerClaudePath,  // At container $HOME (for Claude Code)
            ReadOnly:      false,
        },
    }
}
```

## Design Rationale

### Why Dual Mount Over Alternatives?

| Approach | Pros | Cons |
|----------|------|------|
| **Dual mount** | Simple, no lifecycle commands, works immediately | Two mount entries |
| Symlink at startup | Single mount | Requires postCreateCommand, adds complexity |
| Path rewriting | Could fix all paths | High complexity, fragile |
| CLAUDE_CONFIG_DIR | Simple env var | Doesn't fix paths inside installed_plugins.json |

Dual mount aligns with packnplay's philosophy of minimal complexity and transparency.

### Why Not Same-Path Only?

Claude Code discovers its config at `$HOME/.claude`. If we only mount at the host path (e.g., `/Users/aedgcomb/.claude`), Claude Code wouldn't find its config unless we also set `HOME=/Users/aedgcomb`, which would break other container expectations.

## Edge Cases

1. **Identical paths** (Linux, same user): Optimization skips dual mount
2. **Root container user**: Uses `/root/.claude` as second path
3. **Credential overlay**: Still targets `$HOME/.claude` path correctly
4. **Host path doesn't exist**: Docker creates directory automatically

## Testing

### Unit Tests (`pkg/agents/agent_test.go`)

```go
func TestClaudeAgentDualMount(t *testing.T) {
    agent := &ClaudeAgent{}

    // macOS-style: different paths -> dual mount
    t.Run("different paths produces dual mount", func(t *testing.T) {
        mounts := agent.GetMounts("/Users/testuser", "vscode")
        if len(mounts) != 2 {
            t.Fatalf("GetMounts() returned %d mounts, want 2", len(mounts))
        }
        if mounts[0].ContainerPath != "/Users/testuser/.claude" {
            t.Errorf("Mount[0] ContainerPath = %v, want /Users/testuser/.claude", mounts[0].ContainerPath)
        }
        if mounts[1].ContainerPath != "/home/vscode/.claude" {
            t.Errorf("Mount[1] ContainerPath = %v, want /home/vscode/.claude", mounts[1].ContainerPath)
        }
    })

    // Linux same-user: identical paths -> single mount
    t.Run("identical paths produces single mount", func(t *testing.T) {
        mounts := agent.GetMounts("/home/vscode", "vscode")
        if len(mounts) != 1 {
            t.Fatalf("GetMounts() returned %d mounts, want 1", len(mounts))
        }
    })

    // Root user with different host path -> dual mount
    t.Run("root user produces dual mount", func(t *testing.T) {
        mounts := agent.GetMounts("/Users/testuser", "root")
        if len(mounts) != 2 {
            t.Fatalf("GetMounts() returned %d mounts, want 2", len(mounts))
        }
        if mounts[1].ContainerPath != "/root/.claude" {
            t.Errorf("Mount[1] ContainerPath = %v, want /root/.claude", mounts[1].ContainerPath)
        }
    })
}
```

### Integration Tests (`pkg/runner/mount_builder_test.go`)

Verify `BuildAgentMounts()` produces correct Docker `-v` flags.

### Manual E2E Verification

1. Install Claude Code plugins on host
2. Run `packnplay run claude`
3. Verify `/superpowers:brainstorming` works

## Release Notes

```markdown
### Bug Fixes

- **Claude Code Plugin Support**: Fixed plugin/skill discovery when running
  via `packnplay run claude`. Plugins now work correctly by mounting
  `~/.claude` at both the host path (for absolute path resolution) and
  the container's `$HOME/.claude` (for Claude Code discovery).
```

## Files to Modify

1. `pkg/agents/agent.go` - Update `ClaudeAgent.GetMounts()`
2. `pkg/agents/agent_test.go` - Add dual mount tests
3. `pkg/runner/runner.go` - Remove redundant hardcoded `.claude` mount (now handled by agent abstraction)
4. `pkg/runner/mount_builder_test.go` - Update integration tests if needed
