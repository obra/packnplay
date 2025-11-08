# E2E Test Fix Plan - Complete Redesign

## Problems Identified

### Critical Issues:
1. **Invalid flag**: Tests use `--project` (doesn't exist, should be `--path`)
2. **Container lifecycle misunderstanding**: Containers run `sleep infinity` and stay alive
3. **Missing --reconnect**: Second test runs error because container already running
4. **Port verification broken**: Checking ports on stopped containers
5. **Metadata leaks**: Multiple container IDs created, only last one cleaned
6. **Wrong documentation**: Cleanup commands don't match actual metadata/container names

### Tests Have Never Actually Run
- Tests compile and skip without Docker ✓
- Tests would fail immediately with "unknown flag: --project"
- All E2E testing has been theoretical

## The Real packnplay Workflow

From runner.go analysis:

```
First Run:
1. docker run -d <image> sleep infinity     # Container stays alive
2. docker exec <id> <onCreate command>      # Via metadata tracking
3. docker exec <id> <postCreate command>    # Via metadata tracking
4. docker exec <id> <postStart command>     # Always runs
5. docker exec <id> <user command>          # Test command
6. Container keeps running (sleep infinity)

Second Run (same directory):
1. Check: Container already running
2. WITHOUT --reconnect: ERROR (tells user to use --reconnect)
3. WITH --reconnect:
   - Skip onCreate (metadata shows executed)
   - Skip postCreate (metadata shows executed)
   - Run postStart (always runs)
   - docker exec <id> <user command>
```

## Solution: Three-Phase Fix

### Phase 1: Fix Test Infrastructure

**Change working directory instead of using --project flag**

```go
func runPacknplayInDir(t *testing.T, dir string, args ...string) (string, error) {
    t.Helper()

    oldwd, err := os.Getwd()
    if err != nil {
        t.Fatalf("Failed to get working directory: %v", err)
    }

    if err := os.Chdir(dir); err != nil {
        t.Fatalf("Failed to chdir to %s: %v", dir, err)
    }
    defer os.Chdir(oldwd)

    return runPacknplay(t, args...)
}
```

**Add --no-worktree flag to all tests** (ensures predictable container naming)

**Container cleanup must stop running containers**:
```go
func cleanupContainer(t *testing.T, containerName string) {
    exec.Command("docker", "stop", containerName).Run()  // Stop first
    exec.Command("docker", "rm", "-f", containerName).Run()
}
```

### Phase 2: Fix Lifecycle Tests

**New pattern for onCreate/postCreate tests**:

```go
func TestE2E_OnCreateCommand_RunsOnce(t *testing.T) {
    skipIfNoDocker(t)

    projectDir := createTestProject(t, map[string]string{
        ".devcontainer/devcontainer.json": `{
  "image": "alpine:latest",
  "onCreateCommand": "echo 'onCreate executed' > /tmp/onCreate-ran.txt"
}`,
    })
    defer os.RemoveAll(projectDir)

    containerName := getContainerNameForProject(projectDir)
    defer cleanupContainer(t, containerName)

    // First run - creates container, runs onCreate
    output1, err := runPacknplayInDir(t, projectDir, "run", "--no-worktree", "cat", "/tmp/onCreate-ran.txt")
    require.NoError(t, err)
    require.Contains(t, output1, "onCreate executed")

    // Get container ID (container is still running with sleep infinity)
    containerID := getContainerIDByName(t, containerName)
    require.NotEmpty(t, containerID, "Container should exist after first run")
    defer cleanupMetadata(t, containerID)

    // Verify metadata shows onCreate executed
    metadata := readMetadata(t, containerID)
    require.NotNil(t, metadata)

    lifecycleRan := metadata["lifecycleRan"].(map[string]interface{})
    onCreate := lifecycleRan["onCreate"].(map[string]interface{})
    require.True(t, onCreate["executed"].(bool))
    firstHash := onCreate["commandHash"].(string)
    require.NotEmpty(t, firstHash)

    // Second run - use --reconnect to exec into existing container
    output2, err := runPacknplayInDir(t, projectDir, "run", "--no-worktree", "--reconnect", "cat", "/tmp/onCreate-ran.txt")
    require.NoError(t, err)
    require.Contains(t, output2, "onCreate executed")

    // Verify metadata unchanged (onCreate didn't run again)
    metadata2 := readMetadata(t, containerID)
    onCreate2 := metadata2["lifecycleRan"].(map[string]interface{})["onCreate"].(map[string]interface{})
    secondHash := onCreate2["commandHash"].(string)
    require.Equal(t, firstHash, secondHash, "onCreate should not run again on reconnect")
}
```

**postStart test pattern**:

```go
func TestE2E_PostStartCommand_RunsEveryTime(t *testing.T) {
    // Creates file with timestamp on each run
    // Verify file updated between first and second run
    // Use --reconnect for second run
    // Parse timestamps to verify postStart ran twice
}
```

### Phase 3: Fix Port Tests

**Keep container running and verify ports**:

```go
func TestE2E_PortForwarding(t *testing.T) {
    skipIfNoDocker(t)

    projectDir := createTestProject(t, map[string]string{
        ".devcontainer/devcontainer.json": `{
  "image": "alpine:latest",
  "forwardPorts": [3000, 8080]
}`,
    })
    defer os.RemoveAll(projectDir)

    containerName := getContainerNameForProject(projectDir)
    defer cleanupContainer(t, containerName)

    // Start container (it runs sleep infinity)
    output, err := runPacknplayInDir(t, projectDir, "run", "--no-worktree", "echo", "container started")
    require.NoError(t, err)

    // Container is now running - verify ports
    portOutput, err := exec.Command("docker", "port", containerName, "3000").CombinedOutput()
    require.NoError(t, err, "docker port should work on running container")
    require.Contains(t, string(portOutput), ":3000", "Port 3000 should be mapped")

    // Verify second port
    portOutput2, err := exec.Command("docker", "port", containerName, "8080").CombinedOutput()
    require.NoError(t, err)
    require.Contains(t, string(portOutput2), ":8080", "Port 8080 should be mapped")
}
```

### Phase 4: Fix Cleanup

**Update documentation**:

```markdown
# Remove test containers (actual container names)
docker ps -aq --filter "name=packnplay-packnplay-e2e" | xargs -r docker rm -f

# Remove metadata files (all metadata - no safe filter)
# WARNING: This removes ALL packnplay metadata!
rm -rf ~/.local/share/packnplay/metadata/
```

**Add metadata cleanup for all container IDs created during test**:
- Not needed if we use --reconnect properly (only one container created)

### Phase 5: Add require/assert imports

Add to imports:
```go
import (
    "github.com/stretchr/testify/require"
    "github.com/stretchr/testify/assert"
)
```

Update go.mod if needed.

## Success Criteria

1. ✅ Tests use valid flags (--path or chdir)
2. ✅ Tests handle long-lived containers (--reconnect)
3. ✅ Lifecycle tests verify metadata tracking works
4. ✅ Port tests verify actual port mappings on running containers
5. ✅ No metadata leaks (proper cleanup)
6. ✅ Documentation matches reality
7. ✅ Tests actually run with Docker and pass

## Implementation Order

1. Add testify imports and update go.mod
2. Add runPacknplayInDir helper
3. Fix all tests to use runPacknplayInDir
4. Add --no-worktree to all test calls
5. Fix lifecycle tests with --reconnect pattern
6. Fix port tests to verify running containers
7. Update documentation
8. Run full test suite with Docker
9. Fix any remaining issues
10. Commit and push

---

**Status**: Ready for Implementation
**Estimated Time**: 3-4 hours
**Risk Level**: Medium (tests have never run, may find more issues)
