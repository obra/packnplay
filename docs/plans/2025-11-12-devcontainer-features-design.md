# Devcontainer Features Implementation Design

**Date:** 2025-11-12
**Goal:** Add devcontainer features support for VS Code compatibility and community ecosystem access
**Approach:** Build-time feature processing with Dockerfile generation

## Requirements

**Purpose:** Community ecosystem access - tap into existing devcontainer features
**Constraints:** Minimal complexity while maintaining VS Code compatibility
**Success Criteria:** Support Microsoft universal image and popular community features

## Design

### Architecture Overview

Features install during Docker image build (not runtime) by generating enhanced Dockerfiles that include feature installation layers. Each feature becomes a cached Docker layer for optimal performance.

### Core Components

**1. Feature Resolver**
- Downloads features from OCI registries, HTTPS tarballs, and local paths
- Caches feature artifacts in `~/.packnplay/features-cache/`
- Validates `devcontainer-feature.json` metadata

**2. Dependency Graph Builder**
- Implements round-based dependency resolution algorithm per official spec
- Handles `dependsOn` (hard dependencies) and `installsAfter` (soft dependencies)
- Supports `overrideFeatureInstallOrder` user configuration
- Detects and fails on circular dependencies

**3. Dockerfile Generator**
- Generates enhanced Dockerfile with feature installation layers
- Each feature gets separate RUN layer for optimal caching
- Converts feature options to environment variables
- Preserves user Dockerfile patterns when possible

**4. Feature Executor**
- Executes `install.sh` scripts as root during build
- Sources feature options as environment variables
- Handles lifecycle command integration

### Integration Points

**Config Structure:**
```go
type Config struct {
    // ... existing fields ...
    Features map[string]interface{} `json:"features,omitempty"`
}
```

**Processing Flow:**
1. Parse features from devcontainer.json
2. Resolve and download feature artifacts
3. Build dependency graph and installation order
4. Generate Dockerfile with feature installation layers
5. Execute Docker build with enhanced Dockerfile
6. Merge feature-contributed configuration (env vars, capabilities, etc.)

### Feature Resolution

**OCI Registry Support:**
- Download from ghcr.io/devcontainers/features/* using Docker CLI
- Parse version tags and metadata
- Cache by feature ID and version hash

**HTTPS Tarball Support:**
- HTTP download and extract to cache
- Validate feature structure and metadata

**Local Feature Support:**
- Direct filesystem access for development/custom features
- No caching required

### Dockerfile Generation Strategy

**Enhanced Dockerfile Structure:**
```dockerfile
FROM baseimage as base
USER root

# Feature installation (one layer per feature)
RUN feature1_installation_commands
RUN feature2_installation_commands
RUN feature3_installation_commands

# User and workspace setup
USER remoteUser
WORKDIR /workspace
```

**Benefits:**
- Optimal Docker layer caching (features cache independently)
- Clear separation between system setup (features) and user config
- Compatible with existing Dockerfile patterns
- Matches official devcontainer implementation

### Error Handling and Security

**Error Handling:**
- Feature download failures: Clear error messages with retry suggestions
- Dependency resolution failures: Show dependency graph and conflicts
- Installation failures: Docker build errors with feature context

**Security Considerations:**
- Features run as root with full system access (per spec)
- Validate feature metadata before installation
- Sandbox feature downloads in cache directory
- Rely on Docker security boundaries for isolation

## Implementation Scope

**Phase 1: Core Feature System**
- Basic feature parsing and resolution
- OCI registry support (ghcr.io/devcontainers/features/*)
- Simple dependency resolution (no circular dependency detection)
- Dockerfile generation and build integration

**Phase 2: Advanced Features**
- Full dependency graph resolution with circular detection
- HTTPS tarball and local feature support
- Feature option processing and environment variable conversion
- Lifecycle command integration from features

**Phase 3: Performance and Polish**
- Comprehensive caching system
- Error message improvements
- E2E test coverage for popular features
- Documentation and examples

## Success Metrics

- Support Microsoft universal devcontainer image
- Install popular features (docker-in-docker, node, python, git)
- Maintain fast build times through effective caching
- Pass comprehensive E2E tests with real features
- Integrate cleanly with existing packnplay functionality