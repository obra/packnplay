package runner

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/obra/packnplay/pkg/devcontainer"
)

// ContainerMetadata tracks the lifecycle execution state for a container.
// This metadata is persisted to disk to ensure onCreate/postCreate commands
// run only once, while postStart commands always run.
type ContainerMetadata struct {
	ContainerID  string                    `json:"containerId"`
	CreatedAt    time.Time                 `json:"createdAt"`
	UpdatedAt    time.Time                 `json:"updatedAt"`
	LifecycleRan map[string]LifecycleState `json:"lifecycleRan"`
}

// LifecycleState tracks the execution state of a specific lifecycle command.
type LifecycleState struct {
	Executed    bool      `json:"executed"`
	Timestamp   time.Time `json:"timestamp"`
	CommandHash string    `json:"commandHash"`
}

// GetMetadataPath returns the path where metadata for a container should be stored.
// Creates the directory if it doesn't exist.
// Location: ${XDG_DATA_HOME}/packnplay/metadata/{container-id}.json
// or ~/.local/share/packnplay/metadata/{container-id}.json
func GetMetadataPath(containerID string) (string, error) {
	// Get data directory
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		dataHome = filepath.Join(homeDir, ".local", "share")
	}

	// Create metadata directory
	metadataDir := filepath.Join(dataHome, "packnplay", "metadata")
	if err := os.MkdirAll(metadataDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create metadata directory: %w", err)
	}

	return filepath.Join(metadataDir, containerID+".json"), nil
}

// LoadMetadata loads the metadata for a container from disk.
// If the metadata file doesn't exist, returns a new initialized metadata object.
// This function never errors on missing file - it treats it as first run.
func LoadMetadata(containerID string) (*ContainerMetadata, error) {
	path, err := GetMetadataPath(containerID)
	if err != nil {
		return nil, err
	}

	// If file doesn't exist, return new metadata (first run)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return &ContainerMetadata{
			ContainerID:  containerID,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
			LifecycleRan: make(map[string]LifecycleState),
		}, nil
	}

	// Read and parse existing metadata
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata file: %w", err)
	}

	var metadata ContainerMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("failed to parse metadata: %w", err)
	}

	// Ensure map is initialized
	if metadata.LifecycleRan == nil {
		metadata.LifecycleRan = make(map[string]LifecycleState)
	}

	return &metadata, nil
}

// SaveMetadata saves the metadata for a container to disk.
func SaveMetadata(metadata *ContainerMetadata) error {
	path, err := GetMetadataPath(metadata.ContainerID)
	if err != nil {
		return err
	}

	// Marshal to JSON with indentation for readability
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Write to file
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write metadata file: %w", err)
	}

	return nil
}

// HashCommand computes a deterministic hash of a lifecycle command.
// The hash is based on the JSON representation of the command.
// Returns empty string for nil command.
func HashCommand(cmd *devcontainer.LifecycleCommand) string {
	if cmd == nil {
		return ""
	}

	// Get the raw command content and marshal it
	// We need to access the different command types to get the actual content
	var data []byte
	var err error

	if cmd.IsString() {
		str, _ := cmd.AsString()
		data, err = json.Marshal(str)
	} else if cmd.IsArray() {
		arr, _ := cmd.AsArray()
		data, err = json.Marshal(arr)
	} else if cmd.IsObject() {
		obj, _ := cmd.AsObject()
		data, err = json.Marshal(obj)
	} else {
		// Unknown type
		return ""
	}

	if err != nil {
		// Should not happen in practice, but return empty string if it does
		return ""
	}

	// Compute SHA-256 hash
	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash)
}

// ShouldRun determines whether a lifecycle command should be executed.
// Returns true if:
//   - This is postStart (always runs)
//   - Command hasn't been executed before
//   - Command has changed (different hash)
//
// Returns false if:
//   - Command is nil
//   - Command has been executed before with same hash (for onCreate/postCreate)
func (m *ContainerMetadata) ShouldRun(commandType string, cmd *devcontainer.LifecycleCommand) bool {
	// Nil command should not run
	if cmd == nil {
		return false
	}

	// postStart always runs (no tracking)
	if commandType == "postStart" {
		return true
	}

	// Check if command has been executed before
	state, exists := m.LifecycleRan[commandType]
	if !exists {
		// First time running this command type
		return true
	}

	// Command has been executed before - check if it changed
	currentHash := HashCommand(cmd)
	if currentHash != state.CommandHash {
		// Command changed - run it again
		return true
	}

	// Command already executed with same hash - skip
	return false
}

// MarkExecuted marks a lifecycle command as executed with the current timestamp.
// This should be called after successfully executing a lifecycle command.
func (m *ContainerMetadata) MarkExecuted(commandType string, cmd *devcontainer.LifecycleCommand) {
	if cmd == nil {
		return
	}

	now := time.Now()
	m.LifecycleRan[commandType] = LifecycleState{
		Executed:    true,
		Timestamp:   now,
		CommandHash: HashCommand(cmd),
	}
	m.UpdatedAt = now
}
