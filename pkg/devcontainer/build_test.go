package devcontainer

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

// TestBuildConfig_BasicDockerfile tests parsing basic dockerfile configuration
func TestBuildConfig_BasicDockerfile(t *testing.T) {
	jsonData := `{
		"dockerfile": "Dockerfile"
	}`

	var config struct {
		Build *BuildConfig `json:"build"`
	}
	config.Build = &BuildConfig{}

	if err := json.Unmarshal([]byte(jsonData), &config.Build); err != nil {
		t.Fatalf("Failed to parse build config: %v", err)
	}

	if config.Build.Dockerfile != "Dockerfile" {
		t.Errorf("Expected dockerfile='Dockerfile', got '%s'", config.Build.Dockerfile)
	}
}

// TestBuildConfig_WithContext tests parsing dockerfile with context
func TestBuildConfig_WithContext(t *testing.T) {
	jsonData := `{
		"dockerfile": "Dockerfile.dev",
		"context": ".."
	}`

	var build BuildConfig
	if err := json.Unmarshal([]byte(jsonData), &build); err != nil {
		t.Fatalf("Failed to parse build config: %v", err)
	}

	if build.Dockerfile != "Dockerfile.dev" {
		t.Errorf("Expected dockerfile='Dockerfile.dev', got '%s'", build.Dockerfile)
	}

	if build.Context != ".." {
		t.Errorf("Expected context='..', got '%s'", build.Context)
	}
}

// TestBuildConfig_WithArgs tests parsing build args
func TestBuildConfig_WithArgs(t *testing.T) {
	jsonData := `{
		"dockerfile": "Dockerfile",
		"args": {
			"VARIANT": "16-bullseye",
			"NODE_VERSION": "16.14.0"
		}
	}`

	var build BuildConfig
	if err := json.Unmarshal([]byte(jsonData), &build); err != nil {
		t.Fatalf("Failed to parse build config: %v", err)
	}

	expectedArgs := map[string]string{
		"VARIANT":      "16-bullseye",
		"NODE_VERSION": "16.14.0",
	}

	if !reflect.DeepEqual(build.Args, expectedArgs) {
		t.Errorf("Expected args=%v, got %v", expectedArgs, build.Args)
	}
}

// TestBuildConfig_WithTarget tests parsing multi-stage build target
func TestBuildConfig_WithTarget(t *testing.T) {
	jsonData := `{
		"dockerfile": "Dockerfile",
		"target": "development"
	}`

	var build BuildConfig
	if err := json.Unmarshal([]byte(jsonData), &build); err != nil {
		t.Fatalf("Failed to parse build config: %v", err)
	}

	if build.Target != "development" {
		t.Errorf("Expected target='development', got '%s'", build.Target)
	}
}

// TestBuildConfig_WithCacheFromString tests parsing cacheFrom as string
func TestBuildConfig_WithCacheFromString(t *testing.T) {
	jsonData := `{
		"dockerfile": "Dockerfile",
		"cacheFrom": "ghcr.io/myorg/cache:latest"
	}`

	var build BuildConfig
	if err := json.Unmarshal([]byte(jsonData), &build); err != nil {
		t.Fatalf("Failed to parse build config: %v", err)
	}

	expected := []string{"ghcr.io/myorg/cache:latest"}
	if !reflect.DeepEqual(build.CacheFrom, expected) {
		t.Errorf("Expected cacheFrom=%v, got %v", expected, build.CacheFrom)
	}
}

// TestBuildConfig_WithCacheFromArray tests parsing cacheFrom as array
func TestBuildConfig_WithCacheFromArray(t *testing.T) {
	jsonData := `{
		"dockerfile": "Dockerfile",
		"cacheFrom": [
			"ghcr.io/myorg/cache:latest",
			"ghcr.io/myorg/cache:develop"
		]
	}`

	var build BuildConfig
	if err := json.Unmarshal([]byte(jsonData), &build); err != nil {
		t.Fatalf("Failed to parse build config: %v", err)
	}

	expected := []string{
		"ghcr.io/myorg/cache:latest",
		"ghcr.io/myorg/cache:develop",
	}
	if !reflect.DeepEqual(build.CacheFrom, expected) {
		t.Errorf("Expected cacheFrom=%v, got %v", expected, build.CacheFrom)
	}
}

// TestBuildConfig_WithOptions tests parsing additional build options
func TestBuildConfig_WithOptions(t *testing.T) {
	jsonData := `{
		"dockerfile": "Dockerfile",
		"options": ["--no-cache", "--pull"]
	}`

	var build BuildConfig
	if err := json.Unmarshal([]byte(jsonData), &build); err != nil {
		t.Fatalf("Failed to parse build config: %v", err)
	}

	expected := []string{"--no-cache", "--pull"}
	if !reflect.DeepEqual(build.Options, expected) {
		t.Errorf("Expected options=%v, got %v", expected, build.Options)
	}
}

// TestBuildConfig_Complete tests parsing all fields together
func TestBuildConfig_Complete(t *testing.T) {
	jsonData := `{
		"dockerfile": "Dockerfile.dev",
		"context": "..",
		"args": {
			"VARIANT": "16-bullseye"
		},
		"target": "development",
		"cacheFrom": ["ghcr.io/myorg/cache:latest"],
		"options": ["--pull"]
	}`

	var build BuildConfig
	if err := json.Unmarshal([]byte(jsonData), &build); err != nil {
		t.Fatalf("Failed to parse build config: %v", err)
	}

	if build.Dockerfile != "Dockerfile.dev" {
		t.Errorf("Expected dockerfile='Dockerfile.dev', got '%s'", build.Dockerfile)
	}

	if build.Context != ".." {
		t.Errorf("Expected context='..', got '%s'", build.Context)
	}

	expectedArgs := map[string]string{"VARIANT": "16-bullseye"}
	if !reflect.DeepEqual(build.Args, expectedArgs) {
		t.Errorf("Expected args=%v, got %v", expectedArgs, build.Args)
	}

	if build.Target != "development" {
		t.Errorf("Expected target='development', got '%s'", build.Target)
	}

	expectedCache := []string{"ghcr.io/myorg/cache:latest"}
	if !reflect.DeepEqual(build.CacheFrom, expectedCache) {
		t.Errorf("Expected cacheFrom=%v, got %v", expectedCache, build.CacheFrom)
	}

	expectedOptions := []string{"--pull"}
	if !reflect.DeepEqual(build.Options, expectedOptions) {
		t.Errorf("Expected options=%v, got %v", expectedOptions, build.Options)
	}
}

// TestBuildConfig_ToDockerArgs tests conversion to docker build arguments
func TestBuildConfig_ToDockerArgs(t *testing.T) {
	build := BuildConfig{
		Dockerfile: "Dockerfile.dev",
		Context:    "..",
		Args: map[string]string{
			"VARIANT":      "16-bullseye",
			"NODE_VERSION": "16.14.0",
		},
		Target:    "development",
		CacheFrom: []string{"ghcr.io/myorg/cache:latest"},
		Options:   []string{"--pull"},
	}

	args := build.ToDockerArgs("myapp:latest")

	// Verify key components are present
	hasTag := false
	hasDockerfile := false
	hasTarget := false
	hasCacheFrom := false
	hasContext := false

	for i, arg := range args {
		if arg == "-t" && i+1 < len(args) && args[i+1] == "myapp:latest" {
			hasTag = true
		}
		if arg == "-f" && i+1 < len(args) && args[i+1] == "Dockerfile.dev" {
			hasDockerfile = true
		}
		if arg == "--target" && i+1 < len(args) && args[i+1] == "development" {
			hasTarget = true
		}
		if arg == "--cache-from" && i+1 < len(args) && args[i+1] == "ghcr.io/myorg/cache:latest" {
			hasCacheFrom = true
		}
		if arg == ".." {
			hasContext = true
		}
	}

	if !hasTag {
		t.Error("Expected -t myapp:latest in docker args")
	}
	if !hasDockerfile {
		t.Error("Expected -f Dockerfile.dev in docker args")
	}
	if !hasTarget {
		t.Error("Expected --target development in docker args")
	}
	if !hasCacheFrom {
		t.Error("Expected --cache-from in docker args")
	}
	if !hasContext {
		t.Error("Expected context '..' at end of docker args")
	}
}

// TestBuildConfig_ToDockerArgs_MinimalConfig tests minimal configuration
func TestBuildConfig_ToDockerArgs_MinimalConfig(t *testing.T) {
	build := BuildConfig{
		Dockerfile: "Dockerfile",
	}

	args := build.ToDockerArgs("myapp:latest")

	// Should have: build, -t, tag, -f, dockerfile, context
	if len(args) < 6 {
		t.Errorf("Expected at least 6 args, got %d: %v", len(args), args)
	}

	// First arg should be "build"
	if args[0] != "build" {
		t.Errorf("Expected first arg to be 'build', got '%s'", args[0])
	}

	// Last arg should be context (defaults to ".")
	if args[len(args)-1] != "." {
		t.Errorf("Expected last arg to be '.', got '%s'", args[len(args)-1])
	}
}

// TestBuildConfig_ToDockerArgs_BuildArgsOrder tests that build args are properly formatted
func TestBuildConfig_ToDockerArgs_BuildArgsOrder(t *testing.T) {
	build := BuildConfig{
		Dockerfile: "Dockerfile",
		Args: map[string]string{
			"VAR1": "value1",
			"VAR2": "value2",
		},
	}

	args := build.ToDockerArgs("myapp:latest")

	// Count --build-arg occurrences
	buildArgCount := 0
	for i, arg := range args {
		if arg == "--build-arg" {
			buildArgCount++
			if i+1 >= len(args) {
				t.Error("--build-arg found but no value follows")
			}
		}
	}

	if buildArgCount != 2 {
		t.Errorf("Expected 2 --build-arg flags, got %d", buildArgCount)
	}
}

// TestBuildConfig_CacheFromInvalidType tests error handling for invalid cacheFrom type
func TestBuildConfig_CacheFromInvalidType(t *testing.T) {
	jsonData := `{
		"dockerfile": "Dockerfile",
		"cacheFrom": 123
	}`

	var build BuildConfig
	err := json.Unmarshal([]byte(jsonData), &build)
	if err == nil {
		t.Error("Expected error for cacheFrom as number")
	}

	expectedError := "cacheFrom must be string or array"
	if err != nil && !strings.Contains(err.Error(), expectedError) {
		t.Errorf("Expected error containing '%s', got: %v", expectedError, err)
	}
}

// TestBuildConfig_CacheFromArrayWithNonString tests error handling for non-string in cacheFrom array
func TestBuildConfig_CacheFromArrayWithNonString(t *testing.T) {
	jsonData := `{
		"dockerfile": "Dockerfile",
		"cacheFrom": ["valid", 123, "another"]
	}`

	var build BuildConfig
	err := json.Unmarshal([]byte(jsonData), &build)
	if err == nil {
		t.Error("Expected error for cacheFrom array with non-string element")
	}

	expectedError := "cacheFrom array must contain strings"
	if err != nil && !strings.Contains(err.Error(), expectedError) {
		t.Errorf("Expected error containing '%s', got: %v", expectedError, err)
	}
}

// TestBuildConfig_CacheFromObject tests error handling for cacheFrom as object
func TestBuildConfig_CacheFromObjectInvalid(t *testing.T) {
	jsonData := `{
		"dockerfile": "Dockerfile",
		"cacheFrom": {"invalid": "object"}
	}`

	var build BuildConfig
	err := json.Unmarshal([]byte(jsonData), &build)
	if err == nil {
		t.Error("Expected error for cacheFrom as object")
	}
}

// TestBuildConfig_CacheFromEmptyArray tests empty cacheFrom array is valid
func TestBuildConfig_CacheFromEmptyArray(t *testing.T) {
	jsonData := `{
		"dockerfile": "Dockerfile",
		"cacheFrom": []
	}`

	var build BuildConfig
	if err := json.Unmarshal([]byte(jsonData), &build); err != nil {
		t.Fatalf("Failed to parse build config with empty cacheFrom array: %v", err)
	}

	if len(build.CacheFrom) != 0 {
		t.Errorf("Expected empty cacheFrom array, got %d elements", len(build.CacheFrom))
	}
}

// TestBuildConfig_ArgsWithSpecialCharacters tests build args with special characters
func TestBuildConfig_ArgsWithSpecialCharacters(t *testing.T) {
	jsonData := `{
		"dockerfile": "Dockerfile",
		"args": {
			"URL": "https://example.com:8080/path?query=value",
			"EMPTY": "",
			"EQUALS": "key=value",
			"SPACES": "value with spaces"
		}
	}`

	var build BuildConfig
	if err := json.Unmarshal([]byte(jsonData), &build); err != nil {
		t.Fatalf("Failed to parse build config with special characters: %v", err)
	}

	expectedArgs := map[string]string{
		"URL":    "https://example.com:8080/path?query=value",
		"EMPTY":  "",
		"EQUALS": "key=value",
		"SPACES": "value with spaces",
	}

	if !reflect.DeepEqual(build.Args, expectedArgs) {
		t.Errorf("Expected args=%v, got %v", expectedArgs, build.Args)
	}
}

// TestBuildConfig_InvalidJSON tests error handling for malformed JSON
func TestBuildConfig_InvalidJSON(t *testing.T) {
	jsonData := `{
		"dockerfile": "Dockerfile",
		"args": {
			"KEY": incomplete
		}
	}`

	var build BuildConfig
	err := json.Unmarshal([]byte(jsonData), &build)
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}
