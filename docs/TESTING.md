# Testing Guide

This document describes how to run and write tests for packnplay.

## Test Suite Overview

packnplay has comprehensive test coverage with two types of tests:

1. **Unit Tests** (86+ tests): Fast, no external dependencies, use mocks
2. **End-to-End Tests** (28+ tests): Slower, require Docker daemon, use real containers

## Running Tests

### Quick Start

```bash
# Run all unit tests (fast, no Docker required)
go test -short ./...

# Run all tests including E2E tests (requires Docker daemon)
go test ./...

# Run tests in a specific package
go test ./pkg/runner

# Run with verbose output
go test -v ./pkg/runner
```

### Running E2E Tests

End-to-end tests require a running Docker daemon.

```bash
# Run all E2E tests
go test ./pkg/runner -run TestE2E

# Run E2E tests with verbose output
go test -v ./pkg/runner -run TestE2E

# Run specific E2E test
go test -v ./pkg/runner -run TestE2E_OnCreateCommand_RunsOnce

# Run specific test section (all lifecycle tests)
go test -v ./pkg/runner -run TestE2E.*Lifecycle
```

### Test Requirements

**Unit Tests:**
- No external dependencies
- Run with `-short` flag to skip E2E tests
- Complete in < 5 seconds
- Use mocks for Docker operations

**E2E Tests:**
- Require Docker daemon running (`docker info` must succeed)
- Skip gracefully if Docker unavailable
- Clean up all containers and metadata automatically
- Test real Docker operations (no mocks)
- Complete in < 60 seconds (with Docker)

## E2E Test Infrastructure

### Docker Detection

E2E tests automatically skip when Docker is unavailable:

```bash
# Tests skip in short mode
go test -short ./...

# Tests skip if Docker daemon unavailable
go test ./pkg/runner -run TestE2E
# Output: SKIP: Docker daemon not available
```

### Test Cleanup

E2E tests clean up automatically:

- **Containers**: Removed with `defer cleanupContainer()`
- **Metadata**: Removed from `~/.local/share/packnplay/metadata/`
- **Temporary files**: Project directories cleaned up with `defer os.RemoveAll()`

All cleanup happens even if tests fail (using defer).

### Manual Cleanup

If tests are interrupted, you can manually clean up:

```bash
# Remove all test containers
docker ps -aq --filter "label=managed-by=packnplay-e2e" | xargs -r docker rm -f

# Remove test metadata
rm -rf ~/.local/share/packnplay/metadata/packnplay-e2e-*
```

## Test Coverage

### Unit Test Coverage

Run with coverage report:

```bash
# Generate coverage report
go test -short -coverprofile=coverage.out ./...

# View coverage in browser
go tool cover -html=coverage.out

# Show coverage summary
go test -short -cover ./...
```

### E2E Test Coverage

E2E tests cover all devcontainer features:

- ✅ Image pull and caching
- ✅ Dockerfile builds (all configurations)
- ✅ Build args, targets, and contexts
- ✅ Environment variables (containerEnv, remoteEnv)
- ✅ Variable substitution (all types)
- ✅ Port forwarding (all formats)
- ✅ Lifecycle commands (onCreate, postCreate, postStart)
- ✅ Metadata tracking (run-once behavior)
- ✅ User detection and remoteUser
- ✅ Full integration scenarios

## Writing Tests

### Writing Unit Tests

Unit tests use mocks and follow standard Go testing patterns:

```go
func TestSomeFeature(t *testing.T) {
    // Arrange
    mockClient := &mockDockerClient{
        pullCalled: false,
    }

    // Act
    err := doSomething(mockClient)

    // Assert
    require.NoError(t, err)
    assert.True(t, mockClient.pullCalled)
}
```

### Writing E2E Tests

E2E tests use real Docker and follow this pattern:

```go
func TestE2E_NewFeature(t *testing.T) {
    // Skip if Docker unavailable
    skipIfNoDocker(t)

    // Create test project with devcontainer.json
    projectDir := createTestProject(t, map[string]string{
        ".devcontainer/devcontainer.json": `{
            "image": "alpine:latest",
            "containerEnv": {"TEST_VAR": "value"}
        }`,
    })
    defer os.RemoveAll(projectDir)

    // Setup cleanup
    containerName := getContainerNameForProject(projectDir)
    defer cleanupContainer(t, containerName)
    defer func() {
        containerID := getContainerIDByName(t, containerName)
        if containerID != "" {
            cleanupMetadata(t, containerID)
        }
    }()

    // Run packnplay
    output, err := runPacknplay(t, "run", "--project", projectDir, "sh", "-c", "echo $TEST_VAR")
    require.NoError(t, err)

    // Verify behavior
    assert.Contains(t, output, "value")
}
```

### Test Helper Functions

Available helpers in `pkg/runner/e2e_test.go`:

- `skipIfNoDocker(t)` - Skip test if Docker unavailable
- `createTestProject(t, files)` - Create temporary test directory
- `cleanupContainer(t, name)` - Remove container
- `cleanupMetadata(t, id)` - Remove metadata file
- `getContainerNameForProject(dir)` - Calculate container name
- `getContainerIDByName(t, name)` - Get container ID
- `runPacknplay(t, args...)` - Execute packnplay command
- `readMetadata(t, id)` - Read metadata JSON
- `parseLineCount(output)` - Parse wc -l output

## CI/CD Integration

### GitHub Actions

Example workflow for running tests:

```yaml
name: Tests
on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'

      - name: Run unit tests
        run: go test -short -cover ./...

      - name: Run E2E tests
        run: go test -v ./pkg/runner -run TestE2E

      - name: Upload coverage
        uses: codecov/codecov-action@v3
```

### Running in Docker

You can run the test suite inside Docker:

```bash
# Build test container
docker build -t packnplay-tests -f Dockerfile.test .

# Run tests (requires Docker-in-Docker)
docker run --rm \
  -v /var/run/docker.sock:/var/run/docker.sock \
  packnplay-tests go test ./...
```

## Troubleshooting

### E2E Tests Fail to Find Docker

**Problem**: Tests fail with "Docker daemon not available"

**Solutions**:
1. Verify Docker is running: `docker info`
2. Check Docker socket: `ls -la /var/run/docker.sock`
3. Run unit tests only: `go test -short ./...`

### Tests Leave Containers Behind

**Problem**: Containers remain after test failures

**Solutions**:
1. Tests should clean up automatically (check for panics)
2. Manual cleanup: `docker ps -aq --filter "label=managed-by=packnplay-e2e" | xargs -r docker rm -f`
3. Check test logs for cleanup errors

### E2E Tests Are Slow

**Problem**: E2E tests take > 60 seconds

**Solutions**:
1. Run specific tests: `go test ./pkg/runner -run TestE2E_OnCreate`
2. Run unit tests only: `go test -short ./...`
3. Ensure Docker has sufficient resources (RAM, CPU)

### Port Already in Use

**Problem**: Port forwarding tests fail with "address already in use"

**Solutions**:
1. Tests use unique container names to avoid conflicts
2. Check for containers with same name: `docker ps -a`
3. Stop conflicting container: `docker stop <container>`

### Metadata Tests Fail

**Problem**: Lifecycle metadata tests fail

**Solutions**:
1. Clean metadata: `rm -rf ~/.local/share/packnplay/metadata/packnplay-e2e-*`
2. Check metadata directory permissions: `ls -la ~/.local/share/packnplay/metadata/`
3. Run tests individually to isolate issue

## Test Development Workflow

### TDD Workflow

1. **RED**: Write failing test
```bash
go test -v ./pkg/runner -run TestE2E_NewFeature
# Output: FAIL
```

2. **GREEN**: Implement minimal code to pass
```bash
go test -v ./pkg/runner -run TestE2E_NewFeature
# Output: PASS
```

3. **REFACTOR**: Improve code while keeping tests green
```bash
go test -v ./pkg/runner -run TestE2E_NewFeature
# Output: PASS
```

### Running Tests During Development

```bash
# Watch mode (requires entr or similar)
find . -name '*.go' | entr -c go test -short ./...

# Run specific test repeatedly
while true; do clear; go test -v ./pkg/runner -run TestE2E_OnCreate; sleep 2; done
```

## Test Standards

### Code Quality

- Follow [Effective Go](https://golang.org/doc/effective_go.html)
- Use `require` for critical assertions (test fails immediately)
- Use `assert` for non-critical assertions (test continues)
- Clear test names describing what is tested
- One assertion per test (when possible)
- Proper cleanup with defer

### Test Naming

```go
// Unit tests
TestFunctionName_Scenario           // TestCreateContainer_WithImage
TestFunctionName_Scenario_Expected  // TestCreateContainer_NoImage_Error

// E2E tests
TestE2E_Feature                     // TestE2E_ImagePull
TestE2E_Feature_Scenario            // TestE2E_OnCreateCommand_RunsOnce
```

### Test Organization

- Unit tests in same package as code (`package runner`)
- E2E tests in `pkg/runner/e2e_test.go`
- Test helpers in test files
- Mock implementations in test files

## Additional Resources

- [DevContainer E2E Specification](DEVCONTAINER_E2E_SPEC.md) - Detailed test requirements
- [DevContainer Guide](DEVCONTAINER_GUIDE.md) - Feature documentation
- [Go Testing Package](https://pkg.go.dev/testing) - Official documentation
- [Testify Package](https://github.com/stretchr/testify) - Assertion library

## Contributing Tests

When contributing tests:

1. Run all tests before submitting: `go test ./...`
2. Ensure tests pass in short mode: `go test -short ./...`
3. Add E2E tests for new features requiring Docker
4. Update this guide if adding new test patterns
5. Follow existing test patterns and naming conventions

## Test Metrics

Current test coverage (as of 2025-11-07):

- **Unit Tests**: 86+ tests across all packages
- **E2E Tests**: 28+ tests covering all devcontainer features
- **Total Test Time**: < 5 seconds (unit), < 60 seconds (E2E)
- **Coverage**: > 80% for core packages

---

For questions or issues with tests, please file an issue on GitHub.
