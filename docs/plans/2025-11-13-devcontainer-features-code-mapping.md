# Detailed Code Location Mapping - Devcontainer Features Gaps

## Critical Gap #1: Feature-Contributed Mounts

### Specification Requirement
Features can define Docker mounts that should be applied to running containers:
```json
{
  "id": "docker-in-docker",
  "mounts": [
    {
      "source": "docker-sock",
      "target": "/var/run/docker.sock",
      "type": "bind"
    }
  ]
}
```

### Data Flow Analysis

**1. Parsing (✅ WORKS)**
- File: `pkg/devcontainer/features.go` lines 21-26
- Mount struct defined correctly
- Parsed from JSON successfully
- Test: `TestParseCompleteFeatureMetadata` line 346-350 ✅

**2. Storage (✅ WORKS)**
- File: `pkg/devcontainer/features.go` lines 49
- Stored in `ResolvedFeature.Metadata.Mounts`
- Available when feature is resolved

**3. Application (❌ MISSING)**
- File: `pkg/runner/runner.go` lines 788-822
- Feature properties applied in `FeaturePropertiesApplier.ApplyFeatureProperties()`
- **Line 804: TODO: Apply feature-contributed mounts (Task 6)** ← EXPLICIT TODO
- Current code never reads Mounts field

### Required Fix

**Location:** `pkg/runner/runner.go` lines 800-808

**Current Code:**
```go
for _, feature := range features {
    if feature.Metadata == nil {
        continue
    }

    metadata := feature.Metadata

    // Apply security properties
    if metadata.Privileged != nil && *metadata.Privileged {
        enhancedArgs = append(enhancedArgs, "--privileged")
    }

    for _, cap := range metadata.CapAdd {
        enhancedArgs = append(enhancedArgs, "--cap-add="+cap)
    }

    for _, secOpt := range metadata.SecurityOpt {
        enhancedArgs = append(enhancedArgs, "--security-opt="+secOpt)
    }

    // Apply feature environment variables
    for key, value := range metadata.ContainerEnv {
        enhancedEnv[key] = value
    }

    // TODO: Apply feature-contributed mounts (Task 6)  ← HERE
}
```

**Required Addition:**
```go
// Apply feature-contributed mounts
for _, mount := range metadata.Mounts {
    // Convert Mount struct to Docker mount syntax
    // Docker formats: bind:/src:/dst, volume:name:/dst, tmpfs:/dst
    mountStr := fmt.Sprintf("%s:%s:%s", mount.Type, mount.Source, mount.Target)
    enhancedArgs = append(enhancedArgs, "--mount", mountStr)
}
```

### Test Impact

**Missing Test:** `TestE2E_FeatureMounts` (REQUIRED)
```go
// Should verify mounts are actually created
// Test with feature that mounts docker socket or volume
```

---

## Critical Gap #2: Missing Lifecycle Command Fields

### Specification Requirement
Devcontainer.json should support 5 lifecycle hooks. Current only supports 3.

### Data Flow Analysis

**1. Feature Metadata (✅ PARTIAL)**
- File: `pkg/devcontainer/features.go` lines 51-56
- All 5 hooks defined in FeatureMetadata ✅
- Stored when feature is parsed

**2. User Config Parsing (❌ PARTIAL)**
- File: `pkg/devcontainer/config.go` lines 25-27
- Only 3 of 5 hooks present:
  ```go
  OnCreateCommand   *LifecycleCommand `json:"onCreateCommand,omitempty"`
  PostCreateCommand *LifecycleCommand `json:"postCreateCommand,omitempty"`
  PostStartCommand  *LifecycleCommand `json:"postStartCommand,omitempty"`
  // MISSING: UpdateContentCommand
  // MISSING: PostAttachCommand
  ```

**3. Lifecycle Merger (✅ PARTIAL)**
- File: `pkg/devcontainer/lifecycle_merger.go` lines 19
- Merger handles all 5 hooks correctly
- But if they're not in Config, user hooks are null

**4. Lifecycle Executor (❌ MISSING)**
- File: `pkg/runner/runner.go` lines 895-958
- Only executes 3 hooks (onCreate, postCreate, postStart)
- Never executes mergedCommands for updateContent/postAttach

### Required Fixes

**Fix #1: Add missing fields to Config struct**

Location: `pkg/devcontainer/config.go` lines 25-27

Current:
```go
type Config struct {
    // ... other fields ...
    OnCreateCommand   *LifecycleCommand `json:"onCreateCommand,omitempty"`
    PostCreateCommand *LifecycleCommand `json:"postCreateCommand,omitempty"`
    PostStartCommand  *LifecycleCommand `json:"postStartCommand,omitempty"`
}
```

Required:
```go
type Config struct {
    // ... other fields ...
    OnCreateCommand       *LifecycleCommand `json:"onCreateCommand,omitempty"`
    UpdateContentCommand  *LifecycleCommand `json:"updateContentCommand,omitempty"`
    PostCreateCommand     *LifecycleCommand `json:"postCreateCommand,omitempty"`
    PostStartCommand      *LifecycleCommand `json:"postStartCommand,omitempty"`
    PostAttachCommand     *LifecycleCommand `json:"postAttachCommand,omitempty"`
}
```

**Fix #2: Update lifecycle executor to execute all hooks**

Location: `pkg/runner/runner.go` lines 895-958 (LifecycleExecutor.Execute method)

Current: Only checks/executes 3 commands
Required: Handle all 5 commands

Potential issue with postAttachCommand and updateContentCommand:
- postAttachCommand: Only applies to IDE extensions (VS Code), not CLI
  - Could be skipped or logged as "not applicable"
- updateContentCommand: Triggers when workspace syncs
  - packnplay doesn't have "sync" concept
  - Could execute once on container creation as approximation

---

## Medium Gap #1: Option Validation

### Specification Requirement
Feature options should be validated against their option specifications.

### Current Implementation

**Data Available:**
- File: `pkg/devcontainer/features.go` lines 14-19
- OptionSpec has: type, default, description, proposals
- These are parsed correctly ✅

**Processing:**
- File: `pkg/devcontainer/features.go` lines 298-321
- ProcessOptions converts to environment variables
- **No validation performed** ❌

### Required Validation

```go
func (p *FeatureOptionsProcessor) ValidateOption(
    optionName string,
    userValue interface{},
    spec OptionSpec,
) error {
    // 1. Type validation
    switch spec.Type {
    case "string":
        if _, ok := userValue.(string); !ok {
            return fmt.Errorf("option %s must be string, got %T", optionName, userValue)
        }
    case "number":
        // Handle int/float
    case "boolean":
        // Handle bool
    }
    
    // 2. Proposals validation
    if len(spec.Proposals) > 0 {
        userStr := fmt.Sprintf("%v", userValue)
        valid := false
        for _, proposal := range spec.Proposals {
            if userStr == proposal {
                valid = true
                break
            }
        }
        if !valid {
            return fmt.Errorf(
                "option %s=%s not in proposals: %v",
                optionName, userStr, spec.Proposals,
            )
        }
    }
    
    return nil
}
```

### Test Impact

**Missing Tests:**
- `TestValidateOptionType` - Type checking
- `TestValidateOptionProposals` - Enum validation
- `TestInvalidOptionHandling` - User feedback

---

## Medium Gap #2: Feature Init and Entrypoint

### Specification Requirement
Features can control container init process and entrypoint:
```json
{
  "init": true,
  "entrypoint": ["/usr/bin/dumb-init", "--"]
}
```

### Current Implementation

**Parsed:**
- File: `pkg/devcontainer/features.go` lines 45, 48
- Fields exist: `Init *bool`, `Entrypoint []string`
- Parsed from JSON ✅

**Not Applied:**
- File: `pkg/runner/runner.go` lines 790-808
- No code to apply Init or Entrypoint to Docker args
- Feature properties only apply: Privileged, CapAdd, SecurityOpt, ContainerEnv

### Required Fix

Location: `pkg/runner/runner.go` lines 800-808 (in FeaturePropertiesApplier)

```go
for _, feature := range features {
    if feature.Metadata == nil {
        continue
    }

    // ... existing security properties ...

    // Apply init process
    if feature.Metadata.Init != nil && *feature.Metadata.Init {
        enhancedArgs = append(enhancedArgs, "--init")
    }

    // Apply entrypoint
    if len(feature.Metadata.Entrypoint) > 0 {
        enhancedArgs = append(enhancedArgs, "--entrypoint")
        enhancedArgs = append(enhancedArgs, strings.Join(feature.Metadata.Entrypoint, " "))
    }
}
```

### Test Impact

**Missing Test:** `TestE2E_FeatureInitAndEntrypoint`

---

## Medium Gap #3: Feature-Requested User Context

### Specification Requirement
Features can request to run as specific user:
```json
{
  "remoteUser": "appuser"
}
```

### Current Status

**Missing Field in FeatureMetadata:**
- File: `pkg/devcontainer/features.go`
- No `RemoteUser` field
- Can't represent feature user requirement

### Required Fix

**Step 1: Add field to FeatureMetadata**
```go
type FeatureMetadata struct {
    // ... existing fields ...
    RemoteUser string `json:"remoteUser,omitempty"`
}
```

**Step 2: Merge user context**
- Need logic to resolve conflicts if multiple features request different users
- Specification doesn't define conflict resolution
- Recommendation: User config takes priority, feature can suggest

---

## Low Gap #1: Missing Spec Fields

### Additional Optional Fields Not Implemented

**1. customizations (optional)**
- IDE extension configurations (VS Code specific)
- Not relevant for CLI tool
- Safe to ignore for packnplay

**2. legacyIds (optional)**
- Backward compatibility for renamed features
- Not critical for first implementation

**3. remoteUser in FeatureMetadata (optional)**
- Feature can request user context
- Covered in Medium Gap #3

---

## Testing Coverage Summary

### Unit Tests (Passing)
- ✅ TestResolveLocalFeature - `features_test.go:27`
- ✅ TestResolveDependencies - `features_test.go:92`
- ✅ TestResolveOCIFeature - `features_test.go:171`
- ✅ TestProcessFeatureOptions - `features_test.go:243`
- ✅ TestNormalizeOptionName - `features_test.go:295`
- ✅ TestParseCompleteFeatureMetadata - `features_test.go:318`

### E2E Tests (Passing)
- ✅ TestE2E_BasicFeatureIntegration - `e2e_test.go:1897`
- ✅ TestE2E_CommunityFeature - `e2e_test.go:1940`
- ✅ TestE2E_NodeFeatureWithVersion - `e2e_test.go:1972`
- ✅ TestE2E_FeatureLifecycleCommands - `e2e_test.go:2003`

### Missing Critical E2E Tests
- ❌ TestE2E_FeatureMounts - Would verify mounts are created
- ❌ TestE2E_DockerInDocker - Would verify privileged + mounts
- ❌ TestE2E_OptionValidation - Would catch invalid option errors
- ❌ TestE2E_ComplexDependencies - Would test diamond dependencies
- ❌ TestE2E_InitAndEntrypoint - Would verify init process

---

## Code Change Estimate

| Change | Location | Effort | Lines |
|--------|----------|--------|-------|
| Add feature mounts | runner.go:804 | 2 hours | 10 |
| Add lifecycle fields | config.go:25 | 1 hour | 2 |
| Execute all lifecycle hooks | runner.go:895+ | 2 hours | 20 |
| Option validation | features.go | 3 hours | 40 |
| Init/entrypoint application | runner.go:804+ | 1 hour | 8 |
| Feature remoteUser | features.go | 2 hours | 30 |
| Unit tests | features_test.go | 3 hours | 60 |
| E2E tests | e2e_test.go | 5 hours | 100 |

**Total Estimated Effort: 19-20 hours**

