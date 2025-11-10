# Devcontainer Mounts and RunArgs Implementation Design

**Date:** 2025-11-10
**Goal:** Add custom mounts and runArgs support for devcontainer spec completeness
**Approach:** Incremental addition to existing Config struct

## Requirements

**Purpose:** Feature completeness - match Microsoft devcontainer specification
**Constraints:** Backward compatibility, minimal complexity
**Success Criteria:** New fields work with existing infrastructure

## Design

### Config Structure Changes

Add two fields to `pkg/devcontainer/config.go`:

```go
type Config struct {
    // ... existing fields ...
    Mounts  []string `json:"mounts,omitempty"`   // Docker mount syntax
    RunArgs []string `json:"runArgs,omitempty"`  // Additional docker run arguments
}
```

### Mounts Implementation

**Supports Docker mount syntax:**
```json
{
  "mounts": [
    "source=/var/run/docker.sock,target=/var/run/docker.sock,type=bind",
    "source=my-volume,target=/data,type=volume",
    "type=tmpfs,target=/tmp"
  ]
}
```

**Processing:**
1. Parse strings from devcontainer.json
2. Apply variable substitution using existing `SubstituteVariables()`
3. Convert to `--mount` flags for Docker
4. Add after existing volume mounts in runner

### RunArgs Implementation

**Supports any Docker run arguments:**
```json
{
  "runArgs": ["--memory=2g", "--cpus=2", "--device=/dev/fuse"]
}
```

**Processing:**
1. Parse array from devcontainer.json
2. Apply variable substitution for path references
3. Append to Docker run command before image name
4. Docker validates arguments and provides clear error messages

### Integration Points

**File Changes:**
- `pkg/devcontainer/config.go` - Add fields to Config struct
- `pkg/runner/runner.go` - Process new fields in container creation
- `pkg/runner/e2e_test.go` - Add comprehensive E2E tests

**No changes needed:**
- Variable substitution (reuse existing system)
- Docker client (existing command building works)
- Mount builder (extend existing logic)

## Implementation Plan

### Phase 1: Add Missing E2E Tests (Priority 1)
- Test existing `cacheFrom` and `options` build features
- Test error handling and timeout scenarios for lifecycle commands

### Phase 2: Implement Custom Mounts (Priority 2)
- Add `mounts` field to Config struct
- Integrate mount processing into runner
- Add E2E tests for mount functionality

### Phase 3: Implement Custom RunArgs (Priority 2)
- Add `runArgs` field to Config struct
- Integrate args processing into runner
- Add E2E tests for runArgs functionality

### Phase 4: Verification
- Run full test suite to ensure no regressions
- Test with real-world scenarios
- Update documentation

## Security Considerations

**Mounts:** Docker validates mount syntax and permissions. Invalid mounts fail safely.
**RunArgs:** Docker validates all arguments. Dangerous combinations fail with clear errors.
**Variable Substitution:** Existing system handles malicious input safely.

No additional security validation needed - Docker provides robust validation.

## Backward Compatibility

All changes are additive:
- New fields are optional with `omitempty` tags
- Existing devcontainer.json files work unchanged
- No modification to existing API or behavior
- Tests verify existing functionality unaffected

## Testing Strategy

Comprehensive E2E tests using real Docker:
- Mount types: bind, volume, tmpfs
- Variable substitution in mounts and runArgs
- Error scenarios and validation
- Integration with existing features

Tests follow proven E2E pattern with proper cleanup and isolation.