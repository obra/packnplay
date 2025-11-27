# Fix Plan: Adversarial Review Findings

**Date:** 2025-11-26
**Source:** Adversarial code review of devcontainer implementation

---

## Issue 1: postAttachCommand Only Supports String Format (CRITICAL)

**Location:** `cmd/attach.go:77-88`

**Problem:** The attach command only handles string format via `AsString()`:
```go
if cmdStr, ok := devConfig.PostAttachCommand.AsString(); ok && cmdStr != "" {
    _, err := dockerClient.Run("exec", containerName, "/bin/sh", "-c", cmdStr)
}
```

Array format `["npm", "start"]` and object format `{"task1": "...", "task2": "..."}` are silently ignored.

**Impact:** Users who specify postAttachCommand in array or object format get no error, no warning - command just doesn't run.

### Fix

**Step 1:** Read `cmd/attach.go` to understand current implementation

**Step 2:** Replace string-only handling with full lifecycle command support

Use the same pattern as `LifecycleExecutor.Execute()` in runner.go, or extract to a shared helper:

```go
// Handle postAttachCommand if configured
if devConfig.PostAttachCommand != nil {
    fmt.Fprintf(os.Stderr, "Running postAttachCommand...\n")

    // Get all commands (handles string, array, and object formats)
    commands := devConfig.PostAttachCommand.ToStringSlice()

    for _, cmdStr := range commands {
        if cmdStr == "" {
            continue
        }
        _, err := dockerClient.Run("exec", containerName, "/bin/sh", "-c", cmdStr)
        if err != nil {
            fmt.Fprintf(os.Stderr, "Warning: postAttachCommand failed: %v\n", err)
        }
    }
}
```

**Step 3:** Add E2E test for postAttachCommand (all three formats)

Create test in `pkg/runner/e2e_test.go` or a new `cmd/attach_test.go`:

```go
func TestE2E_PostAttachCommand_StringFormat(t *testing.T) {
    // Test: postAttachCommand: "touch /tmp/attach-ran"
    // Verify file exists after attach
}

func TestE2E_PostAttachCommand_ArrayFormat(t *testing.T) {
    // Test: postAttachCommand: ["touch", "/tmp/attach-ran"]
    // Verify file exists after attach
}

func TestE2E_PostAttachCommand_ObjectFormat(t *testing.T) {
    // Test: postAttachCommand: {"task1": "touch /tmp/task1", "task2": "touch /tmp/task2"}
    // Verify both files exist after attach
}
```

**Estimated LOC:** ~80

---

## Issue 2: Multiple Features with Entrypoint - No Warning (HIGH)

**Location:** `pkg/runner/runner.go:100-107`

**Problem:** When multiple features specify `entrypoint`, the loop appends multiple `--entrypoint=` flags. Docker uses the last one, silently ignoring earlier ones.

```go
for _, resolvedFeature := range resolvedFeatures {
    // ...
    if len(metadata.Entrypoint) > 0 {
        enhancedArgs = append(enhancedArgs, "--entrypoint="+metadata.Entrypoint[0])
    }
}
```

**Impact:** Feature composition is broken - user doesn't know Feature A's entrypoint was overridden by Feature B.

### Fix

**Step 1:** Track if entrypoint has already been set

```go
var entrypointSet bool
var entrypointSource string

for _, resolvedFeature := range resolvedFeatures {
    metadata := resolvedFeature.Metadata
    if metadata == nil {
        continue
    }

    // ... other property handling ...

    if len(metadata.Entrypoint) > 0 {
        if entrypointSet {
            fmt.Fprintf(os.Stderr, "Warning: feature '%s' overrides entrypoint from '%s'\n",
                resolvedFeature.ID, entrypointSource)
        }
        enhancedArgs = append(enhancedArgs, "--entrypoint="+metadata.Entrypoint[0])
        if len(metadata.Entrypoint) > 1 {
            entrypointArgs = metadata.Entrypoint[1:]
        }
        entrypointSet = true
        entrypointSource = resolvedFeature.ID
    }
}
```

**Step 2:** Add test for warning behavior

```go
func TestE2E_MultipleEntrypoints_Warning(t *testing.T) {
    // Create two features that both set entrypoint
    // Verify warning is printed to stderr
    // Verify container uses the last feature's entrypoint
}
```

**Estimated LOC:** ~30

---

## Issue 3: Empty String in capAdd/securityOpt Arrays (MEDIUM)

**Location:** `pkg/runner/runner.go:87-93`

**Problem:** If a feature specifies `"capAdd": [""]` or `"securityOpt": [""]`, the code appends `--cap-add=` or `--security-opt=` with an empty value, which Docker rejects.

### Fix

**Step 1:** Filter empty strings before appending

```go
for _, cap := range metadata.CapAdd {
    if cap != "" {
        enhancedArgs = append(enhancedArgs, "--cap-add="+cap)
    }
}

for _, opt := range metadata.SecurityOpt {
    if opt != "" {
        enhancedArgs = append(enhancedArgs, "--security-opt="+opt)
    }
}
```

**Step 2:** Add test for empty string filtering

```go
func TestE2E_FeatureCapAdd_EmptyStringIgnored(t *testing.T) {
    // Feature with capAdd: ["", "SYS_ADMIN", ""]
    // Verify only --cap-add=SYS_ADMIN is passed (not --cap-add= or --cap-add=)
}
```

**Estimated LOC:** ~20

---

## Issue 4: No E2E Tests for postAttachCommand (MEDIUM)

**Location:** `pkg/runner/e2e_test.go`

**Problem:** Zero E2E tests for postAttachCommand. The attach command is only tested manually.

### Fix

This is addressed as part of Issue 1. The tests added there will cover:
- String format
- Array format
- Object format
- Failure handling

**Note:** Testing `attach` is harder than `run` because:
1. Container must already be running
2. Need to exec into container to verify command ran
3. Attach uses syscall.Exec which replaces the process

May need a different test approach - perhaps:
1. Start container with `run`
2. Call attach code directly (not syscall.Exec path)
3. Verify postAttachCommand executed

**Estimated LOC:** Included in Issue 1

---

## Summary

| Issue | Severity | Fix | LOC |
|-------|----------|-----|-----|
| 1. postAttachCommand formats | CRITICAL | Handle all formats + tests | ~80 |
| 2. Multiple entrypoint warning | HIGH | Track + warn | ~30 |
| 3. Empty string filtering | MEDIUM | Filter + test | ~20 |
| 4. postAttachCommand E2E tests | MEDIUM | Included in #1 | - |
| **Total** | | | **~130** |

## Execution Order

1. **Issue 1** - Fix postAttachCommand (CRITICAL, blocks Issue 4)
2. **Issue 2** - Add entrypoint warning (HIGH, independent)
3. **Issue 3** - Filter empty strings (MEDIUM, quick fix)

Issues 2 and 3 can be done in parallel after Issue 1.

## Verification

After all fixes:
```bash
go test -v ./... -timeout 10m
go install .
# Manual test of attach with array format
```
