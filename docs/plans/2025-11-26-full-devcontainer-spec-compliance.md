# Plan: Full Devcontainer Specification Compliance

**Date:** 2025-11-26
**Goal:** Implement remaining 7 features to achieve 100% Microsoft devcontainer spec compliance
**Current:** 97% compliance (7 gaps remaining)

---

## Overview

This plan implements the remaining 3% of the devcontainer specification. Features are ordered by:
1. Impact (how often used)
2. Complexity (easier first builds momentum)
3. Dependencies (some features depend on others)

**Estimated total:** ~500 LOC implementation + ~300 LOC tests = ~800 LOC

---

## Part 1: Core Lifecycle Features (High Impact)

### Task 1: Implement `initializeCommand`

**Priority:** HIGH
**Complexity:** MEDIUM
**Estimated LOC:** ~100

**What it is:**
- Runs on the **HOST** before container creation
- Used for pre-container setup (e.g., `git submodule update`, `npm ci` on host)
- Security consideration: executes arbitrary code from devcontainer.json

**Implementation:**

**Files to modify:**
- `pkg/devcontainer/config.go` - Add `InitializeCommand` field (may already exist)
- `pkg/runner/runner.go` - Execute before container creation

**Step 1:** Add to Config struct if missing

```go
type Config struct {
    // ... existing fields ...
    InitializeCommand *LifecycleCommand `json:"initializeCommand,omitempty"`
}
```

**Step 2:** Execute in runner.go before container creation (before line 920)

```go
// Before creating container, run initializeCommand on HOST
if devConfig.InitializeCommand != nil {
    fmt.Fprintf(os.Stderr, "Running initializeCommand on host...\n")

    commands := devConfig.InitializeCommand.ToStringSlice()
    for _, cmdStr := range commands {
        if cmdStr == "" {
            continue
        }

        // Execute on HOST (not in container)
        cmd := exec.Command("/bin/sh", "-c", cmdStr)
        cmd.Dir = workDir // Run in project directory
        cmd.Stdout = os.Stdout
        cmd.Stderr = os.Stderr

        if err := cmd.Run(); err != nil {
            return fmt.Errorf("initializeCommand failed: %w", err)
        }
    }
}
```

**Step 3:** Add E2E test

```go
func TestE2E_InitializeCommand(t *testing.T) {
    // Test that initializeCommand runs on HOST before container
    // Create a file on host, verify container can see it via mount
}
```

**Security notes:**
- Add warning if initializeCommand is present: "Running host command from devcontainer.json"
- Consider adding `--allow-initialize` flag requirement (opt-in)
- Document security implications in README

---

### Task 2: Implement `remoteEnv`

**Priority:** MEDIUM
**Complexity:** MEDIUM
**Estimated LOC:** ~80

**What it is:**
- Environment variables computed **inside the container**
- Values can reference other env vars or run commands
- Example: `"remoteEnv": {"PATH": "${containerEnv:PATH}:/custom/bin"}`

**Implementation:**

**Files to modify:**
- `pkg/devcontainer/config.go` - Add `RemoteEnv` field
- `pkg/runner/runner.go` - Process and inject env vars after container starts

**Step 1:** Add to Config struct

```go
type Config struct {
    // ... existing fields ...
    RemoteEnv map[string]string `json:"remoteEnv,omitempty"`
}
```

**Step 2:** Process remoteEnv after container creation (after line 924)

```go
// Apply remoteEnv after container is running
if len(devConfig.RemoteEnv) > 0 {
    for key, value := range devConfig.RemoteEnv {
        // Variable substitution
        resolved := substituteVariables(value, containerID, workDir, mountPath)

        // Write to container's environment
        // Options:
        // 1. Write to /etc/environment
        // 2. Append to .bashrc/.zshrc
        // 3. Docker exec to set for current shell

        cmd := fmt.Sprintf("echo 'export %s=%q' >> /home/%s/.bashrc",
            key, resolved, devConfig.RemoteUser)
        _, err := dockerClient.Run("exec", containerID, "/bin/sh", "-c", cmd)
        if err != nil {
            fmt.Fprintf(os.Stderr, "Warning: failed to set remoteEnv %s: %v\n", key, err)
        }
    }
}
```

**Step 3:** Implement variable substitution

```go
func substituteVariables(value, containerID, workDir, mountPath string) string {
    // ${containerEnv:VAR} - read from container environment
    // ${localEnv:VAR} - read from host environment
    // ${containerWorkspaceFolder} - mount path
    // ${localWorkspaceFolder} - host path

    // Use regex to find and replace variables
}
```

**Step 4:** Add E2E test

```go
func TestE2E_RemoteEnv(t *testing.T) {
    // Test remoteEnv sets environment variables in container
    // Verify variable substitution works
}
```

---

## Part 2: Container Management (Medium Impact)

### Task 3: Implement Container Restart Behavior

**Priority:** MEDIUM
**Complexity:** LOW
**Estimated LOC:** ~50

**What it is:**
- Currently: packnplay recreates stopped containers
- Spec: should restart stopped containers if they exist
- Benefit: faster reconnect, preserves container state

**Implementation:**

**Files to modify:**
- `pkg/runner/runner.go` - Check for stopped containers before recreating

**Step 1:** Detect stopped containers (around line 350)

```go
// Check for stopped container with same name
stoppedID, err := dockerClient.Run("ps", "-aq", "--filter", "name="+containerName)
if err == nil && strings.TrimSpace(stoppedID) != "" {
    // Container exists but is stopped
    fmt.Fprintf(os.Stderr, "Restarting stopped container %s...\n", containerName)

    _, err := dockerClient.Run("start", stoppedID)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Failed to restart container, recreating: %v\n", err)
        // Fall through to recreation
    } else {
        // Successfully restarted, continue with existing container
        containerID = stoppedID
        // Skip container creation, jump to lifecycle commands
    }
}
```

**Step 2:** Add E2E test

```go
func TestE2E_ContainerRestart(t *testing.T) {
    // Create container, stop it, run again - should restart not recreate
    // Verify container ID is same
    // Verify state is preserved (file created in first run still exists)
}
```

---

## Part 3: Advanced Features (Lower Priority)

### Task 4: Implement HTTPS Tarball Features

**Priority:** LOW
**Complexity:** MEDIUM
**Estimated LOC:** ~120

**What it is:**
- Download features from HTTPS URLs
- Example: `"features": {"https://example.com/feature.tgz": {}}`
- Used for custom/private features not in OCI registry

**Implementation:**

**Files to modify:**
- `pkg/devcontainer/features.go` - Add HTTPS detection and download

**Step 1:** Detect HTTPS feature references in ResolveFeature (line 229)

```go
func (r *FeatureResolver) ResolveFeature(featurePath string, options map[string]interface{}) (*ResolvedFeature, error) {
    // Existing: checks for OCI registry (ghcr.io, mcr.microsoft.com)
    // Existing: checks for local paths

    // Add: check for HTTPS tarball
    if strings.HasPrefix(featurePath, "https://") {
        return r.resolveHTTPSFeature(featurePath, options)
    }

    // ... existing code ...
}
```

**Step 2:** Implement HTTPS download

```go
func (r *FeatureResolver) resolveHTTPSFeature(url string, options map[string]interface{}) (*ResolvedFeature, error) {
    // Download tarball
    resp, err := http.Get(url)
    if err != nil {
        return nil, fmt.Errorf("failed to download feature from %s: %w", url, err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != 200 {
        return nil, fmt.Errorf("failed to download feature: HTTP %d", resp.StatusCode)
    }

    // Extract to cache directory
    cacheDir := filepath.Join(r.cacheDir, hashURL(url))
    if err := os.MkdirAll(cacheDir, 0755); err != nil {
        return nil, err
    }

    // Untar
    if err := extractTarball(resp.Body, cacheDir); err != nil {
        return nil, err
    }

    // Process like local feature
    return r.processLocalFeature(cacheDir, options)
}
```

**Step 3:** Add E2E test (requires test HTTP server)

```go
func TestE2E_HTTPSFeature(t *testing.T) {
    // Start local HTTP server serving feature tarball
    // Reference in devcontainer.json
    // Verify feature is downloaded and installed
}
```

---

### Task 5: Implement Private Feature Authentication

**Priority:** LOW
**Complexity:** HIGH
**Estimated LOC:** ~150

**What it is:**
- Authenticate to private OCI registries
- Support for Docker credentials, tokens, etc.
- Example: private features at `ghcr.io/private-org/private-feature`

**Implementation:**

**Files to modify:**
- `pkg/devcontainer/features.go` - Add auth to OCI pulls

**Step 1:** Use Docker credentials for OCI pulls

```go
func (r *FeatureResolver) pullOCIFeature(ociRef string) (string, error) {
    // Current: uses `docker pull` which inherits Docker auth
    // This should already work if user is logged in with `docker login`

    // Enhancement: explicit credential passing
    // Read from ~/.docker/config.json
    // Pass to oras pull with --username and --password flags

    // Actually, this may already work! Test with private registry first.
}
```

**Step 2:** Add explicit credential support if needed

```go
// Add environment variables or config options:
// DEVCONTAINER_FEATURE_REGISTRY_USERNAME
// DEVCONTAINER_FEATURE_REGISTRY_PASSWORD
// DEVCONTAINER_FEATURE_REGISTRY_TOKEN
```

**Step 3:** Test with private registry

```go
func TestE2E_PrivateFeature(t *testing.T) {
    // Requires access to private registry
    // Or mock with local registry
    t.Skip("Requires private registry setup")
}
```

---

### Task 6: Implement Lockfile Support

**Priority:** LOW
**Complexity:** MEDIUM
**Estimated LOC:** ~100

**What it is:**
- `devcontainer-lock.json` pins feature versions
- Ensures reproducible builds
- Generated by `devcontainer features pin` command

**Implementation:**

**Files to modify:**
- `pkg/devcontainer/config.go` - Add lockfile loading
- `pkg/devcontainer/features.go` - Use locked versions

**Step 1:** Load lockfile if exists

```go
type LockFile struct {
    Features map[string]LockedFeature `json:"features"`
}

type LockedFeature struct {
    Version string `json:"version"`
    Resolved string `json:"resolved"` // Full OCI ref with digest
}

func LoadLockFile(dir string) (*LockFile, error) {
    lockPath := filepath.Join(dir, ".devcontainer", "devcontainer-lock.json")
    // ... load and parse ...
}
```

**Step 2:** Use locked versions in feature resolution

```go
func (r *FeatureResolver) ResolveFeature(featurePath string, options map[string]interface{}) (*ResolvedFeature, error) {
    // Check if lockfile has entry for this feature
    if r.lockfile != nil {
        if locked, exists := r.lockfile.Features[featurePath]; exists {
            // Use locked version instead of latest
            featurePath = locked.Resolved
        }
    }

    // ... continue with resolution ...
}
```

**Step 3:** Add command to generate lockfile (future enhancement)

```bash
packnplay lock  # Generate devcontainer-lock.json
```

**Step 4:** Add test

```go
func TestE2E_Lockfile(t *testing.T) {
    // Create lockfile with pinned version
    // Verify feature is installed at pinned version, not latest
}
```

---

### Task 7: Implement `portsAttributes`

**Priority:** LOW
**Complexity:** LOW
**Estimated LOC:** ~60

**What it is:**
- Configure per-port settings
- Labels, descriptions, protocol (http/https)
- Example:
  ```json
  "portsAttributes": {
    "3000": {
      "label": "Application",
      "protocol": "https"
    }
  }
  ```

**Implementation:**

**Files to modify:**
- `pkg/devcontainer/config.go` - Add `PortsAttributes` field
- `pkg/runner/runner.go` - Apply port labels (low priority, mostly UI metadata)

**Step 1:** Add to Config struct

```go
type PortAttributes struct {
    Label    string `json:"label,omitempty"`
    Protocol string `json:"protocol,omitempty"` // http, https
    OnAutoForward string `json:"onAutoForward,omitempty"` // notify, openBrowser, silent
}

type Config struct {
    // ... existing fields ...
    PortsAttributes map[string]PortAttributes `json:"portsAttributes,omitempty"`
}
```

**Step 2:** Apply as Docker labels (informational only)

```go
// When creating container, add labels for port metadata
if len(devConfig.PortsAttributes) > 0 {
    for port, attrs := range devConfig.PortsAttributes {
        if attrs.Label != "" {
            args = append(args, "--label",
                fmt.Sprintf("devcontainer.port.%s.label=%s", port, attrs.Label))
        }
        if attrs.Protocol != "" {
            args = append(args, "--label",
                fmt.Sprintf("devcontainer.port.%s.protocol=%s", port, attrs.Protocol))
        }
    }
}
```

**Step 3:** Add test

```go
func TestE2E_PortsAttributes(t *testing.T) {
    // Verify port labels are applied to container
    // Check docker inspect output
}
```

**Note:** This is mostly metadata for IDE integration. packnplay is a CLI tool, so the main benefit is documentation/labels on containers.

---

## Summary

| Task | Priority | Complexity | LOC | Impact |
|------|----------|------------|-----|--------|
| 1. initializeCommand | HIGH | MEDIUM | ~100 | High - common use case |
| 2. remoteEnv | MEDIUM | MEDIUM | ~80 | Medium - PATH customization |
| 3. Container restart | MEDIUM | LOW | ~50 | Medium - faster reconnect |
| 4. HTTPS features | LOW | MEDIUM | ~120 | Low - rare use case |
| 5. Private auth | LOW | HIGH | ~150 | Low - may already work |
| 6. Lockfile support | LOW | MEDIUM | ~100 | Low - enterprise feature |
| 7. portsAttributes | LOW | LOW | ~60 | Low - UI metadata |
| **Total** | | | **~660** | |

## Execution Order

**Phase 1: Core Features (Tasks 1-3)**
- These provide the most user value
- Can be done in 2-3 days
- Gets us to ~99% compliance

**Phase 2: Advanced Features (Tasks 4-7)**
- Lower priority, edge cases
- Can be spread over time or wait for user requests
- Gets us to 100% compliance

## Testing Strategy

Each task includes:
1. Unit tests for core logic
2. E2E tests for real Docker integration
3. Update to README documenting new feature

## Documentation Updates

After completion:
- Update README to remove gaps section
- Change "97% compliance" to "100% compliance"
- Add examples for each new feature
- Update DEVCONTAINER_GUIDE.md with detailed docs

## Risks & Considerations

1. **initializeCommand security**: Runs arbitrary host code. Consider:
   - Warning message when present
   - Optional `--allow-initialize` flag
   - Document security implications

2. **Container restart state**: Must preserve:
   - Environment variables
   - File modifications
   - Running processes (may not be preserved)

3. **HTTPS features**: Need to handle:
   - Compression formats (tar.gz, tgz, tar)
   - Checksum validation (optional)
   - Caching strategy

4. **Private auth**: May already work via Docker login. Test first before implementing custom auth.

## Success Criteria

- [ ] All 7 features implemented
- [ ] E2E tests pass for each feature
- [ ] README updated to show 100% compliance
- [ ] No regressions in existing features
- [ ] Documentation complete with examples
