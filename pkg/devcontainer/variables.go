// Variable substitution implementation adapted from devcontainers/cli (MIT License)
// Original: https://github.com/devcontainers/cli/blob/main/src/spec-common/variableSubstitution.ts
// Copyright (c) Microsoft Corporation. All rights reserved.

package devcontainer

import (
	"crypto/sha256"
	"encoding/base32"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Substitute performs variable substitution on any JSON-compatible value
// Recursively processes strings, arrays, and objects
// Preserves non-string types (numbers, booleans, null)
func Substitute(ctx *SubstituteContext, value interface{}) interface{} {
	switch v := value.(type) {
	case string:
		return substituteString(ctx, v)

	case []interface{}:
		// Recurse into arrays
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = Substitute(ctx, item)
		}
		return result

	case map[string]interface{}:
		// Recurse into objects
		result := make(map[string]interface{}, len(v))
		for k, val := range v {
			result[k] = Substitute(ctx, val)
		}
		return result

	default:
		// Preserve numbers, booleans, null
		return v
	}
}

// substituteString performs variable substitution on a string
// Supports patterns: ${varType:varName:default:more:colons}
func substituteString(ctx *SubstituteContext, s string) string {
	// Match ${...} patterns
	re := regexp.MustCompile(`\$\{([^}]+)\}`)

	return re.ReplaceAllStringFunc(s, func(match string) string {
		// Remove ${ and }
		inner := match[2 : len(match)-1]

		// Split on colons: varType:varName:default:more
		// Use SplitN with 3 to preserve colons in default value
		parts := strings.SplitN(inner, ":", 3)

		varType := parts[0]
		var varName, defaultVal string
		if len(parts) > 1 {
			varName = parts[1]
		}
		if len(parts) > 2 {
			defaultVal = parts[2] // Can contain more colons
		}

		// Lookup based on type
		switch varType {
		case "env", "localEnv":
			if val, ok := ctx.LocalEnv[varName]; ok {
				return val
			}
			return defaultVal

		case "containerEnv":
			if val, ok := ctx.ContainerEnv[varName]; ok {
				return val
			}
			return defaultVal

		case "localWorkspaceFolder":
			return ctx.LocalWorkspaceFolder

		case "localWorkspaceFolderBasename":
			return filepath.Base(ctx.LocalWorkspaceFolder)

		case "containerWorkspaceFolder":
			// May contain variables - recurse
			return substituteString(ctx, ctx.ContainerWorkspaceFolder)

		case "containerWorkspaceFolderBasename":
			// First resolve containerWorkspaceFolder, then get basename
			resolved := substituteString(ctx, ctx.ContainerWorkspaceFolder)
			return filepath.Base(resolved)

		case "devcontainerId":
			return generateDevContainerID(ctx.Labels)
		}

		// Preserve unknown variables
		return match
	})
}

// generateDevContainerID creates a deterministic ID from labels
// Uses SHA-256 hash + base32 encoding to create a 52-character ID
// Label order doesn't matter (sorted for determinism)
func generateDevContainerID(labels map[string]string) string {
	// Sort keys for determinism
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Build sorted string
	var builder strings.Builder
	for _, k := range keys {
		builder.WriteString(k)
		builder.WriteString("=")
		builder.WriteString(labels[k])
		builder.WriteString("\n")
	}

	// SHA-256 hash
	hash := sha256.Sum256([]byte(builder.String()))

	// Base32 encode to 52 chars (base32 is safe for container names)
	encoded := base32.StdEncoding.EncodeToString(hash[:])
	encoded = strings.ToLower(encoded)

	// Return first 52 characters
	if len(encoded) > 52 {
		return encoded[:52]
	}
	return encoded
}
