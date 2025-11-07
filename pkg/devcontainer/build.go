package devcontainer

import (
	"encoding/json"
	"fmt"
)

// BuildConfig represents build configuration for devcontainer.
// Corresponds to the "build" property in devcontainer.json.
type BuildConfig struct {
	// Dockerfile path relative to .devcontainer directory
	Dockerfile string `json:"dockerfile"`

	// Context is the build context path (defaults to .devcontainer if empty)
	Context string `json:"context,omitempty"`

	// Args are build-time variables passed to docker build
	Args map[string]string `json:"args,omitempty"`

	// Target specifies which build stage to build in multi-stage Dockerfiles
	Target string `json:"target,omitempty"`

	// CacheFrom specifies images to use as cache sources (can be string or array in JSON)
	CacheFrom []string `json:"-"` // Handled by custom unmarshal

	// Options are additional docker build flags
	Options []string `json:"options,omitempty"`
}

// UnmarshalJSON implements custom JSON unmarshaling to handle cacheFrom as string or array
func (b *BuildConfig) UnmarshalJSON(data []byte) error {
	// First unmarshal into a temporary struct with all fields except cacheFrom
	type Alias BuildConfig
	aux := &struct {
		CacheFrom interface{} `json:"cacheFrom,omitempty"`
		*Alias
	}{
		Alias: (*Alias)(b),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// Handle cacheFrom as string or array
	if aux.CacheFrom != nil {
		switch v := aux.CacheFrom.(type) {
		case string:
			b.CacheFrom = []string{v}
		case []interface{}:
			b.CacheFrom = make([]string, len(v))
			for i, item := range v {
				if s, ok := item.(string); ok {
					b.CacheFrom[i] = s
				} else {
					return fmt.Errorf("cacheFrom array must contain strings, got %T", item)
				}
			}
		default:
			return fmt.Errorf("cacheFrom must be string or array, got %T", v)
		}
	}

	return nil
}

// ToDockerArgs converts BuildConfig to docker build command arguments
//
// SECURITY WARNING: Build args are persisted in image metadata and can be
// inspected with `docker history`. Users should not put secrets in build args.
// For secrets, use containerEnv with ${localEnv:SECRET} variable substitution
// which injects secrets at runtime without persisting them in the image.
func (b *BuildConfig) ToDockerArgs(tag string) []string {
	args := []string{"build"}

	// Tag
	args = append(args, "-t", tag)

	// Dockerfile
	args = append(args, "-f", b.Dockerfile)

	// Build args
	for k, v := range b.Args {
		args = append(args, "--build-arg", fmt.Sprintf("%s=%s", k, v))
	}

	// Target
	if b.Target != "" {
		args = append(args, "--target", b.Target)
	}

	// Cache from
	for _, cache := range b.CacheFrom {
		args = append(args, "--cache-from", cache)
	}

	// Additional options
	args = append(args, b.Options...)

	// Context (defaults to current directory)
	context := b.Context
	if context == "" {
		context = "."
	}
	args = append(args, context)

	return args
}
