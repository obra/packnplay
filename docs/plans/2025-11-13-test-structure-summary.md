# Microsoft devcontainer CLI Test Structure - Executive Summary

## Quick Overview

The Microsoft devcontainer CLI repository (`vendor/devcontainer-cli/`) contains a comprehensive TypeScript test suite for container-features functionality. This analysis identifies all tests you should port to packnplay.

**Total Test Lines**: 4,400 lines across 11 test files
**Test Framework**: Mocha + Chai
**Language**: TypeScript
**Example Fixtures**: 15 feature sets + 27 test configurations

---

## The 11 Test Files You Need to Port

### Tier 1: Core Functionality (Must Have)

1. **featureHelpers.test.ts** (925 lines)
   - Tests feature identifier parsing from all sources
   - Feature ID sanitization (getSafeId)
   - Backward compatibility handling
   - **Location**: `/vendor/devcontainer-cli/src/test/container-features/featureHelpers.test.ts`

2. **containerFeaturesOrder.test.ts** (699 lines)
   - Feature dependency resolution (installsAfter, dependsOn)
   - Circular dependency detection
   - Deterministic ordering algorithms
   - **Location**: `/vendor/devcontainer-cli/src/test/container-features/containerFeaturesOrder.test.ts`

3. **lifecycleHooks.test.ts** (461 lines)
   - Lifecycle command execution (onCreate, postCreate, postStart, postAttach)
   - Hook ordering verification
   - Container resume behavior
   - **Location**: `/vendor/devcontainer-cli/src/test/container-features/lifecycleHooks.test.ts`

4. **featuresCLICommands.test.ts** (703 lines)
   - CLI command testing (features test, features package)
   - Test filtering and options
   - Output validation patterns
   - **Location**: `/vendor/devcontainer-cli/src/test/container-features/featuresCLICommands.test.ts`

### Tier 2: Integration & Configuration (Should Have)

5. **e2e.test.ts** (255 lines)
   - End-to-end container building with features
   - Invalid config detection
   - Command execution in containers
   - **Location**: `/vendor/devcontainer-cli/src/test/container-features/e2e.test.ts`

6. **generateFeaturesConfig.test.ts** (137 lines)
   - Feature configuration generation
   - Dockerfile layer building
   - VSCode customizations merging
   - **Location**: `/vendor/devcontainer-cli/src/test/container-features/generateFeaturesConfig.test.ts`

7. **containerFeaturesOCI.test.ts** (310 lines)
   - OCI registry reference parsing
   - Version/tag/digest handling
   - Reference structure validation
   - **Location**: `/vendor/devcontainer-cli/src/test/container-features/containerFeaturesOCI.test.ts`

### Tier 3: Advanced Features (Nice to Have)

8. **lockfile.test.ts** (260 lines)
   - Reproducible builds via lockfiles
   - Frozen lockfile enforcement
   - Outdated package detection
   - **Location**: `/vendor/devcontainer-cli/src/test/container-features/lockfile.test.ts`

9. **containerFeaturesOCIPush.test.ts** (374 lines)
   - Publishing features to registries
   - Authentication and tagging
   - Manifest generation
   - **Location**: `/vendor/devcontainer-cli/src/test/container-features/containerFeaturesOCIPush.test.ts`

10. **registryCompatibilityOCI.test.ts** (172 lines)
    - Multi-registry compatibility
    - Authentication strategies (GitHub, Azure, Docker config)
    - **Location**: `/vendor/devcontainer-cli/src/test/container-features/registryCompatibilityOCI.test.ts`

11. **featureAdvisories.test.ts** (104 lines)
    - Security advisory matching
    - Version range validation
    - **Location**: `/vendor/devcontainer-cli/src/test/container-features/featureAdvisories.test.ts`

---

## Test Utilities & Fixtures

### Shared Test Utilities (testUtils.ts)
**Location**: `/vendor/devcontainer-cli/src/test/testUtils.ts`

Critical functions to port:
- `shellExec(command, options?, suppressOutput?, doNotThrow?)` - Command execution
- `devContainerUp(cli, workspaceFolder, options?)` - Container lifecycle
- `devContainerDown(options)` - Cleanup
- `devContainerStop(options)` - Pause
- Build kit options for parameterized testing

### Example Feature Sets (15 directories)
**Location**: `/vendor/devcontainer-cli/src/test/container-features/example-v2-features-sets/`

Key fixtures:
- `simple/` - Basic features (color, hello)
- `a-installs-after-b/` - Dependency testing
- `lifecycle-hooks/` - Hook execution
- `failing-test/` - Error scenarios
- `dockerfile-scenario-test/` - Complex scenarios

### Test Configurations (27 directories)
**Location**: `/vendor/devcontainer-cli/src/test/container-features/configs/`

Key configs:
- `feature-dependencies/` - Dependency scenarios
- `lifecycle-hooks-*/` - Hook variations
- `dockerfile-with-v2-*` - Dockerfile combinations
- `registry-compatibility/` - Auth scenarios
- `invalid-configs/` - Error cases

---

## Feature Coverage Matrix

| Feature | Test File | Lines | Tests |
|---------|-----------|-------|-------|
| Feature Resolution | featureHelpers.test.ts | 925 | OCI, GitHub, Tarball, Local, Legacy |
| Dependency Order | containerFeaturesOrder.test.ts | 699 | installsAfter, dependsOn, circular |
| Lifecycle Hooks | lifecycleHooks.test.ts | 461 | onCreate, postCreate, postStart, postAttach |
| CLI Commands | featuresCLICommands.test.ts | 703 | test, package, filtering |
| E2E Integration | e2e.test.ts | 255 | Build, install, execute, validate |
| Config Generation | generateFeaturesConfig.test.ts | 137 | Dockerfile, customizations |
| OCI References | containerFeaturesOCI.test.ts | 310 | Parsing, validation |
| Registry Push | containerFeaturesOCIPush.test.ts | 374 | Publish, tag, auth |
| Lockfiles | lockfile.test.ts | 260 | Generate, freeze, outdated |
| Advisories | featureAdvisories.test.ts | 104 | Version matching |
| Registry Compat | registryCompatibilityOCI.test.ts | 172 | Multi-registry, auth |

---

## Test Execution Patterns

### Pattern 1: Feature Resolution
```
Input: Feature identifier (any source type)
Process: Parse → Identify source → Fetch metadata
Output: Resolved feature with all properties
Tests: Handle all 5 source types + backward compat
```

### Pattern 2: Dependency Ordering
```
Input: List of features with dependencies
Process: Build graph → Detect cycles → Sort topologically
Output: Ordered feature list
Tests: Valid/invalid orderings, circular deps
```

### Pattern 3: Lifecycle Execution
```
Input: Container + lifecycle commands + features
Process: Install features → Execute hooks in order
Output: Marker files showing execution order
Tests: 15 execution phases, resume behavior
```

### Pattern 4: E2E Testing
```
Input: devcontainer.json with features
Process: Build container → Install features → Test
Output: Container ready + test results
Tests: Success/failure, output validation
```

---

## Key Testing Concepts

### Feature Sources (5 types)
1. **OCI Registry**: `ghcr.io/devcontainers/features/docker-in-docker:1`
2. **GitHub Release**: `owner/repo/feature@version`
3. **Direct Tarball**: `https://example.com/path/feature.tgz`
4. **Local Path**: `./color`
5. **Legacy/Cached**: `docker-in-docker` (auto-mapped)

### Dependency Models (2 types)
1. **installsAfter**: Feature must install after specified feature(s)
2. **dependsOn**: Feature depends on functionality in specified feature(s)

### Lifecycle Stages (5 types)
1. **onCreateCommand**: During container creation
2. **updateContentCommand**: After source code mount
3. **postCreateCommand**: After container creation
4. **postStartCommand**: On container start/resume
5. **postAttachCommand**: On editor attach

### Test Model
1. Parse configuration
2. Resolve feature identifiers
3. Build dependency graph
4. Order features for installation
5. Generate Dockerfile layers
6. Build/start container
7. Execute installation scripts
8. Run lifecycle hooks
9. Execute test scripts
10. Validate output

---

## Porting Strategy

### Phase 1: Core Utilities (Week 1)
- Port testUtils patterns to Go
- Create Docker/container execution wrappers
- Set up test fixtures (example-v2-features-sets)

### Phase 2: Core Features (Week 2-3)
- Port featureHelpers tests (feature resolution)
- Port containerFeaturesOrder tests (dependency ordering)
- Port lifecycleHooks tests (hook execution)

### Phase 3: Integration (Week 4)
- Port e2e tests
- Port generateFeaturesConfig tests
- Port CLI command tests

### Phase 4: Advanced (Week 5)
- Port remaining tests as needed
- Add registry compatibility tests
- Add security advisory tests

---

## Go Test Translation

The Microsoft tests use:
- **Mocha**: Translates to Go testing.T patterns
- **Chai assertions**: Translates to standard Go assertions
- **Shell execution**: Translates to exec.Command or similar
- **Fixtures**: Copy as test data directories

Example translation:
```typescript
describe('Feature resolution', () => {
  it('should parse OCI registry', () => {
    const result = processFeatureIdentifier(params, config);
    assert.equal(result.type, 'oci');
  });
});
```

Becomes:
```go
func TestFeatureResolution(t *testing.T) {
  t.Run("should parse OCI registry", func(t *testing.T) {
    result := ProcessFeatureIdentifier(params, config)
    if result.Type != "oci" {
      t.Fatalf("expected oci, got %s", result.Type)
    }
  })
}
```

---

## Files to Copy

### Test Files (Copy All 11)
- `featuresCLICommands.test.ts`
- `containerFeaturesOrder.test.ts`
- `featureHelpers.test.ts`
- `lifecycleHooks.test.ts`
- `e2e.test.ts`
- `generateFeaturesConfig.test.ts`
- `containerFeaturesOCI.test.ts`
- `containerFeaturesOCIPush.test.ts`
- `lockfile.test.ts`
- `featureAdvisories.test.ts`
- `registryCompatibilityOCI.test.ts`

### Utilities
- `testUtils.ts` (as reference)

### Fixtures (Copy Entire Directories)
- `example-v2-features-sets/`
- `configs/`

### Reference Files
- `package.json` (for test setup)
- `tsconfig.json` (for compilation settings)

---

## Documentation

Two detailed documents have been created:

1. **2025-11-13-microsoft-devcontainer-test-analysis.md** (21 KB)
   - Complete breakdown of all 11 test files
   - Detailed test groups and assertions
   - Test patterns and best practices
   - Configuration examples

2. **2025-11-13-microsoft-devcontainer-test-file-paths.txt** (8 KB)
   - Absolute file paths to all test files
   - Directory structure reference
   - Copy priorities and patterns
   - Feature coverage matrix

---

## Next Steps

1. **Read** the full analysis documents
2. **Review** the example test files to understand patterns
3. **Identify** which tests are critical for packnplay
4. **Start** with Tier 1 tests (core functionality)
5. **Create** Go equivalents using the same patterns
6. **Port** fixtures and configurations
7. **Iterate** through Tiers 2 and 3 as time permits

---

## Quick Links to Files

All files are in `/vendor/devcontainer-cli/src/test/container-features/`:

- **Test Files**: `*.test.ts`
- **Utilities**: `testUtils.ts`
- **Fixtures**: `example-v2-features-sets/`
- **Configs**: `configs/`

Start with reading:
1. `featureHelpers.test.ts` - Feature resolution
2. `containerFeaturesOrder.test.ts` - Dependency ordering
3. `lifecycleHooks.test.ts` - Hook execution
4. `testUtils.ts` - Test utilities

---

**Analysis Created**: 2025-11-13
**Total Test Lines**: 4,400
**Test Files**: 11
**Example Fixtures**: 15 directories
**Test Configs**: 27 directories
