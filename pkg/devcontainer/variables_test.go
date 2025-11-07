package devcontainer

// Test cases adapted from devcontainers/cli (MIT License)
// Original: https://github.com/devcontainers/cli/blob/main/src/test/variableSubstitution.test.ts
// Copyright (c) Microsoft Corporation. All rights reserved.

import (
	"os"
	"testing"
)

func TestSubstituteEnvironmentVariable(t *testing.T) {
	// Test: ${env:VAR} and ${localEnv:VAR} substitution
	os.Setenv("TEST_ENV_VAR", "test-value")
	defer os.Unsetenv("TEST_ENV_VAR")

	ctx := &SubstituteContext{
		LocalEnv: map[string]string{
			"TEST_ENV_VAR": "test-value",
		},
		ContainerEnv: make(map[string]string),
	}

	// Test ${env:VAR}
	result := Substitute(ctx, "${env:TEST_ENV_VAR}")
	if result != "test-value" {
		t.Errorf("Expected 'test-value', got '%s'", result)
	}

	// Test ${localEnv:VAR}
	result = Substitute(ctx, "${localEnv:TEST_ENV_VAR}")
	if result != "test-value" {
		t.Errorf("Expected 'test-value', got '%s'", result)
	}
}

func TestSubstituteLocalEnvWithDefault(t *testing.T) {
	// Test: ${localEnv:MISSING:default} returns default when variable missing
	ctx := &SubstituteContext{
		LocalEnv:     make(map[string]string),
		ContainerEnv: make(map[string]string),
	}

	result := Substitute(ctx, "${localEnv:MISSING_VAR:fallback-value}")
	if result != "fallback-value" {
		t.Errorf("Expected 'fallback-value', got '%s'", result)
	}
}

func TestSubstituteMissingVariable(t *testing.T) {
	// Test: Missing variable without default returns empty string
	ctx := &SubstituteContext{
		LocalEnv:     make(map[string]string),
		ContainerEnv: make(map[string]string),
	}

	result := Substitute(ctx, "${localEnv:MISSING_VAR}")
	if result != "" {
		t.Errorf("Expected empty string, got '%s'", result)
	}
}

func TestSubstituteWorkspaceFolder(t *testing.T) {
	// Test: ${localWorkspaceFolder} substitution
	ctx := &SubstituteContext{
		LocalWorkspaceFolder: "/Users/jesse/projects/myapp",
		LocalEnv:             make(map[string]string),
		ContainerEnv:         make(map[string]string),
	}

	result := Substitute(ctx, "${localWorkspaceFolder}")
	if result != "/Users/jesse/projects/myapp" {
		t.Errorf("Expected '/Users/jesse/projects/myapp', got '%s'", result)
	}
}

func TestSubstituteWorkspaceFolderBasename(t *testing.T) {
	// Test: ${localWorkspaceFolderBasename} extracts basename
	ctx := &SubstituteContext{
		LocalWorkspaceFolder: "/Users/jesse/projects/myapp",
		LocalEnv:             make(map[string]string),
		ContainerEnv:         make(map[string]string),
	}

	result := Substitute(ctx, "${localWorkspaceFolderBasename}")
	if result != "myapp" {
		t.Errorf("Expected 'myapp', got '%s'", result)
	}
}

func TestSubstituteRecursive(t *testing.T) {
	// Test: ${containerWorkspaceFolder} can contain ${localWorkspaceFolderBasename}
	ctx := &SubstituteContext{
		LocalWorkspaceFolder:     "/Users/jesse/projects/myapp",
		ContainerWorkspaceFolder: "/workspace/${localWorkspaceFolderBasename}",
		LocalEnv:                 make(map[string]string),
		ContainerEnv:             make(map[string]string),
	}

	result := Substitute(ctx, "${containerWorkspaceFolder}")
	if result != "/workspace/myapp" {
		t.Errorf("Expected '/workspace/myapp', got '%s'", result)
	}
}

func TestSubstituteDevContainerID(t *testing.T) {
	// Test: ${devcontainerId} generates SHA-256 based ID
	ctx := &SubstituteContext{
		Labels: map[string]string{
			"project": "myproject",
			"env":     "development",
		},
		LocalEnv:     make(map[string]string),
		ContainerEnv: make(map[string]string),
	}

	result := Substitute(ctx, "${devcontainerId}")
	resultStr, ok := result.(string)
	if !ok {
		t.Fatalf("Expected string result, got %T", result)
	}

	// Should be a 52-character lowercase string
	if len(resultStr) != 52 {
		t.Errorf("Expected 52 characters, got %d", len(resultStr))
	}

	// Should be lowercase
	for _, c := range resultStr {
		if c >= 'A' && c <= 'Z' {
			t.Errorf("Expected lowercase string, got '%s'", resultStr)
			break
		}
	}
}

func TestDevContainerIDDeterministic(t *testing.T) {
	// Test: Same labels → same ID, label order doesn't matter
	ctx1 := &SubstituteContext{
		Labels: map[string]string{
			"project": "myproject",
			"env":     "development",
			"version": "1.0",
		},
		LocalEnv:     make(map[string]string),
		ContainerEnv: make(map[string]string),
	}

	ctx2 := &SubstituteContext{
		Labels: map[string]string{
			"version": "1.0",
			"project": "myproject",
			"env":     "development",
		},
		LocalEnv:     make(map[string]string),
		ContainerEnv: make(map[string]string),
	}

	result1 := Substitute(ctx1, "${devcontainerId}")
	result2 := Substitute(ctx2, "${devcontainerId}")

	if result1 != result2 {
		t.Errorf("Expected same ID for same labels in different order, got '%s' and '%s'", result1, result2)
	}
}

func TestSubstituteMultipleColonsInDefault(t *testing.T) {
	// Test: ${localEnv:MISSING:default:a:b:c} → "default:a:b:c"
	ctx := &SubstituteContext{
		LocalEnv:     make(map[string]string),
		ContainerEnv: make(map[string]string),
	}

	result := Substitute(ctx, "${localEnv:MISSING:default:a:b:c}")
	if result != "default:a:b:c" {
		t.Errorf("Expected 'default:a:b:c', got '%s'", result)
	}
}

func TestSubstituteContainerEnv(t *testing.T) {
	// Test: ${containerEnv:PATH}
	ctx := &SubstituteContext{
		LocalEnv: make(map[string]string),
		ContainerEnv: map[string]string{
			"PATH":     "/usr/bin:/bin",
			"NODE_ENV": "production",
		},
	}

	result := Substitute(ctx, "${containerEnv:PATH}:/custom")
	if result != "/usr/bin:/bin:/custom" {
		t.Errorf("Expected '/usr/bin:/bin:/custom', got '%s'", result)
	}
}

func TestSubstituteContainerEnvWithDefault(t *testing.T) {
	// Test: ${containerEnv:MISSING:default}
	ctx := &SubstituteContext{
		LocalEnv:     make(map[string]string),
		ContainerEnv: make(map[string]string),
	}

	result := Substitute(ctx, "${containerEnv:MISSING:development}")
	if result != "development" {
		t.Errorf("Expected 'development', got '%s'", result)
	}
}

func TestSubstituteArray(t *testing.T) {
	// Test: Substitution in arrays
	ctx := &SubstituteContext{
		LocalEnv: map[string]string{
			"API_KEY": "secret-key",
		},
		ContainerEnv: make(map[string]string),
	}

	input := []interface{}{
		"echo ${localEnv:API_KEY}",
		"npm install",
		"PORT=${localEnv:PORT:3000}",
	}

	result := Substitute(ctx, input)
	resultArray, ok := result.([]interface{})
	if !ok {
		t.Fatalf("Expected array result, got %T", result)
	}

	if len(resultArray) != 3 {
		t.Fatalf("Expected 3 elements, got %d", len(resultArray))
	}

	if resultArray[0] != "echo secret-key" {
		t.Errorf("Expected 'echo secret-key', got '%s'", resultArray[0])
	}

	if resultArray[1] != "npm install" {
		t.Errorf("Expected 'npm install', got '%s'", resultArray[1])
	}

	if resultArray[2] != "PORT=3000" {
		t.Errorf("Expected 'PORT=3000', got '%s'", resultArray[2])
	}
}

func TestSubstituteObject(t *testing.T) {
	// Test: Substitution in nested objects
	ctx := &SubstituteContext{
		LocalWorkspaceFolder: "/workspace",
		LocalEnv: map[string]string{
			"NODE_ENV": "development",
		},
		ContainerEnv: make(map[string]string),
	}

	input := map[string]interface{}{
		"workDir": "${localWorkspaceFolder}",
		"nodeEnv": "${localEnv:NODE_ENV}",
		"nested": map[string]interface{}{
			"path": "${localWorkspaceFolder}/src",
		},
	}

	result := Substitute(ctx, input)
	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", result)
	}

	if resultMap["workDir"] != "/workspace" {
		t.Errorf("Expected '/workspace', got '%s'", resultMap["workDir"])
	}

	if resultMap["nodeEnv"] != "development" {
		t.Errorf("Expected 'development', got '%s'", resultMap["nodeEnv"])
	}

	nested, ok := resultMap["nested"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected nested map, got %T", resultMap["nested"])
	}

	if nested["path"] != "/workspace/src" {
		t.Errorf("Expected '/workspace/src', got '%s'", nested["path"])
	}
}

func TestSubstitutePreservesNonStrings(t *testing.T) {
	// Test: Numbers, booleans, and null are preserved
	ctx := &SubstituteContext{
		LocalEnv:     make(map[string]string),
		ContainerEnv: make(map[string]string),
	}

	input := map[string]interface{}{
		"port":     float64(3000),
		"enabled":  true,
		"disabled": false,
		"nothing":  nil,
		"name":     "test",
		"withEnv":  "${localEnv:TEST:fallback}",
	}

	result := Substitute(ctx, input)
	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", result)
	}

	if resultMap["port"] != float64(3000) {
		t.Errorf("Expected 3000, got %v", resultMap["port"])
	}

	if resultMap["enabled"] != true {
		t.Errorf("Expected true, got %v", resultMap["enabled"])
	}

	if resultMap["disabled"] != false {
		t.Errorf("Expected false, got %v", resultMap["disabled"])
	}

	if resultMap["nothing"] != nil {
		t.Errorf("Expected nil, got %v", resultMap["nothing"])
	}

	if resultMap["name"] != "test" {
		t.Errorf("Expected 'test', got '%s'", resultMap["name"])
	}

	if resultMap["withEnv"] != "fallback" {
		t.Errorf("Expected 'fallback', got '%s'", resultMap["withEnv"])
	}
}

func TestSubstituteContainerWorkspaceFolderBasename(t *testing.T) {
	// Test: ${containerWorkspaceFolderBasename} with recursive substitution
	ctx := &SubstituteContext{
		LocalWorkspaceFolder:     "/Users/jesse/myapp",
		ContainerWorkspaceFolder: "/workspace/${localWorkspaceFolderBasename}",
		LocalEnv:                 make(map[string]string),
		ContainerEnv:             make(map[string]string),
	}

	result := Substitute(ctx, "${containerWorkspaceFolderBasename}")
	if result != "myapp" {
		t.Errorf("Expected 'myapp', got '%s'", result)
	}
}

func TestSubstituteMultipleVariablesInString(t *testing.T) {
	// Test: Multiple variables in a single string
	ctx := &SubstituteContext{
		LocalWorkspaceFolder: "/workspace",
		LocalEnv: map[string]string{
			"USER": "testuser",
		},
		ContainerEnv: make(map[string]string),
	}

	result := Substitute(ctx, "Path: ${localWorkspaceFolder}, User: ${localEnv:USER}")
	if result != "Path: /workspace, User: testuser" {
		t.Errorf("Expected 'Path: /workspace, User: testuser', got '%s'", result)
	}
}
