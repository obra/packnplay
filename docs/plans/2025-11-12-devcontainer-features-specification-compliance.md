# Devcontainer Features Specification Compliance Design

**Date:** 2025-11-12
**Goal:** Achieve 100% compliance with devcontainer features specification
**Approach:** Systematically fix current implementation gaps for full parity

## Requirements

**Purpose:** Full parity with official implementation - specification compliance and VS Code compatibility
**Constraints:** Specification compliance, VS Code compatibility
**Success Criteria:** Pass official devcontainer features compliance tests, support complete feature metadata

## Critical Gaps Analysis

Fresh eyes review identified 20 critical gaps in current implementation. Priority fixes for specification compliance:

### P0 - Core Functionality Broken
1. **Feature options ignored** - Environment variable conversion missing
2. **OCI build context limitation** - Features outside context fail Docker COPY
3. **Incomplete metadata** - Missing required specification fields
4. **No lifecycle hook integration** - Feature commands not merged with user commands

### P1 - Specification Non-Compliance
5. **Missing security properties** - capAdd, privileged, securityOpt not supported
6. **No containerEnv from features** - Feature environment variables not applied
7. **User context variables missing** - _REMOTE_USER not available to features
8. **Feature-contributed mounts not supported** - Mounts from feature metadata ignored

## Technical Architecture Enhancements

### Enhanced Feature Metadata Structure
```go
type FeatureMetadata struct {
    // Required per specification
    ID      string `json:"id"`
    Version string `json:"version"`
    Name    string `json:"name"`

    // Options specification
    Options map[string]OptionSpec `json:"options,omitempty"`

    // Container properties from features
    ContainerEnv map[string]string `json:"containerEnv,omitempty"`
    Privileged   *bool              `json:"privileged,omitempty"`
    Init         *bool              `json:"init,omitempty"`
    CapAdd       []string           `json:"capAdd,omitempty"`
    SecurityOpt  []string           `json:"securityOpt,omitempty"`
    Mounts       []Mount           `json:"mounts,omitempty"`

    // Lifecycle hooks per specification
    OnCreateCommand      *LifecycleCommand `json:"onCreateCommand,omitempty"`
    UpdateContentCommand *LifecycleCommand `json:"updateContentCommand,omitempty"`
    PostCreateCommand    *LifecycleCommand `json:"postCreateCommand,omitempty"`
    PostStartCommand     *LifecycleCommand `json:"postStartCommand,omitempty"`
    PostAttachCommand    *LifecycleCommand `json:"postAttachCommand,omitempty"`

    // Dependencies
    DependsOn     []string `json:"dependsOn,omitempty"`
    InstallsAfter []string `json:"installsAfter,omitempty"`
}

type OptionSpec struct {
    Type        string      `json:"type"`
    Default     interface{} `json:"default,omitempty"`
    Description string      `json:"description,omitempty"`
    Proposals   []string    `json:"proposals,omitempty"`
}
```

### Feature Options Processing Engine
Implement environment variable conversion per specification:
- Option name normalization using official regex
- Default value application from feature metadata
- Environment file generation for each feature
- Dockerfile ENV commands for proper sourcing

### OCI Build Context Solution
Use Docker multi-stage builds to solve build context limitations:
- Stage 1: Feature preparation from cache
- Stage 2: Base image with features copied in
- Maintains Docker layer caching for performance
- Supports any OCI feature location

### Lifecycle Command Merger
Integrate feature lifecycle hooks with user commands:
- Parse lifecycle commands from all installed features
- Execute in installation order per specification
- Run feature commands before user commands
- Support all five lifecycle hook types

## Implementation Strategy

### Phase 1: Core Specification Compliance (Critical)
1. **Fix feature options processing** - Environment variable conversion
2. **Enhance FeatureMetadata** - Add all specification fields
3. **Solve OCI build context** - Multi-stage build approach
4. **Add lifecycle hook merger** - Feature commands integration

### Phase 2: Container Properties Integration
5. **Security properties support** - capAdd, privileged, securityOpt
6. **Feature-contributed environment** - containerEnv from metadata
7. **Feature-contributed mounts** - Mount specifications from features
8. **User context variables** - _REMOTE_USER and workspace variables

### Phase 3: Production Quality
9. **Robust OCI client** - Replace oras stub with proper implementation
10. **Comprehensive error handling** - Clear messages for all failure modes
11. **Performance optimization** - Feature caching and build parallelization
12. **Security validation** - Feature signature verification and sandboxing

## Success Metrics

### Specification Compliance Tests
- Microsoft universal devcontainer image works perfectly
- All 25+ features install with correct options and configurations
- Feature lifecycle hooks execute in proper order
- Security properties applied correctly

### VS Code Compatibility Tests
- Identical behavior to VS Code devcontainer extension
- Same build performance characteristics
- Same error messages and user experience
- Support for all VS Code devcontainer patterns

### Real-World Usage Validation
- Popular community features work without modification
- Complex multi-feature setups function correctly
- Feature dependency chains resolve properly
- Performance matches or exceeds VS Code implementation

## Technical Validation

### Feature Options Test
```json
{
  "features": {
    "ghcr.io/devcontainers/features/node:1": {
      "version": "18.20.0",
      "installType": "nvm"
    }
  }
}
```
Expected: Node.js 18.20.0 installed via nvm (not default version)

### Multi-Feature Dependency Test
```json
{
  "features": {
    "ghcr.io/devcontainers/features/docker-in-docker:2": {},
    "ghcr.io/devcontainers/features/node:1": {"version": "18"},
    "ghcr.io/devcontainers/features/common-utils:2": {"installZsh": true}
  }
}
```
Expected: All features install in dependency order with correct options

### Lifecycle Hook Test
```json
{
  "features": {
    "ghcr.io/devcontainers/features/python:1": {}
  },
  "postCreateCommand": "pip install -r requirements.txt"
}
```
Expected: Python feature's postCreateCommand runs before user's pip install

## Implementation Scope

**Estimated effort:** 15 hours for P0 (critical gaps) + 10 hours for P1 (specification compliance)
**Risk mitigation:** Comprehensive test suite with real community features
**Quality gate:** Must pass Microsoft universal image compatibility test