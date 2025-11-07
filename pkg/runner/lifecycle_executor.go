package runner

import (
	"fmt"
	"sync"

	"github.com/obra/packnplay/pkg/devcontainer"
)

// LifecycleExecutor executes lifecycle commands in a container.
// It supports three command formats:
//   - String: Shell command executed via sh -c
//   - Array: Direct command execution without shell
//   - Object: Multiple commands executed in parallel
type LifecycleExecutor struct {
	client        DockerClient
	containerName string
	containerUser string
	verbose       bool
	metadata      *ContainerMetadata
}

// NewLifecycleExecutor creates a new lifecycle executor.
func NewLifecycleExecutor(client DockerClient, containerName, containerUser string, verbose bool, metadata *ContainerMetadata) *LifecycleExecutor {
	return &LifecycleExecutor{
		client:        client,
		containerName: containerName,
		containerUser: containerUser,
		verbose:       verbose,
		metadata:      metadata,
	}
}

// Execute executes a lifecycle command in the container.
// The commandType parameter is used for tracking (e.g., "onCreate", "postCreate", "postStart").
// Returns error if execution fails, nil if skipped or successful.
func (le *LifecycleExecutor) Execute(commandType string, cmd *devcontainer.LifecycleCommand) error {
	if cmd == nil {
		return nil
	}

	// Check if command should run (based on metadata tracking)
	if le.metadata != nil && !le.metadata.ShouldRun(commandType, cmd) {
		if le.verbose {
			fmt.Printf("Skipping %s (already executed)\n", commandType)
		}
		return nil
	}

	// Handle different command types
	var err error
	if cmd.IsString() {
		str, _ := cmd.AsString()
		err = le.executeShellCommand(str)
	} else if cmd.IsArray() {
		arr, _ := cmd.AsArray()
		err = le.executeDirectCommand(arr)
	} else if cmd.IsObject() {
		obj, _ := cmd.AsObject()
		err = le.executeParallelCommands(obj)
	} else {
		return fmt.Errorf("unknown lifecycle command type")
	}

	// Mark as executed if successful
	if err == nil && le.metadata != nil {
		le.metadata.MarkExecuted(commandType, cmd)
	}

	return err
}

// executeShellCommand executes a single shell command in the container.
//
// SECURITY NOTE: Command comes from devcontainer.json (user's own config file).
// This is executed in the user's own container with their own credentials.
// No privilege escalation occurs. The user is running their own commands
// in their own environment, so command injection is not a concern here.
func (le *LifecycleExecutor) executeShellCommand(cmd string) error {
	// Use docker exec to run command in container
	args := []string{
		"exec",
		"-u", le.containerUser,
		le.containerName,
		"sh", "-c", cmd,
	}

	output, err := le.client.Run(args...)
	if le.verbose || err != nil {
		fmt.Println(output)
	}

	return err
}

// executeDirectCommand executes a command with direct arguments (no shell).
func (le *LifecycleExecutor) executeDirectCommand(cmdArray []string) error {
	if len(cmdArray) == 0 {
		return nil
	}

	// Build docker exec args
	args := []string{
		"exec",
		"-u", le.containerUser,
		le.containerName,
	}
	args = append(args, cmdArray...)

	output, err := le.client.Run(args...)
	if le.verbose || err != nil {
		fmt.Println(output)
	}

	return err
}

// executeParallelCommands executes multiple commands in parallel.
func (le *LifecycleExecutor) executeParallelCommands(commands map[string]interface{}) error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(commands))

	for name, cmd := range commands {
		wg.Add(1)
		go func(taskName string, taskCmd interface{}) {
			defer wg.Done()

			var err error
			switch v := taskCmd.(type) {
			case string:
				err = le.executeShellCommand(v)
			case []interface{}:
				// Convert []interface{} to []string
				strArray := make([]string, len(v))
				for i, item := range v {
					if s, ok := item.(string); ok {
						strArray[i] = s
					} else {
						err = fmt.Errorf("task %s: invalid command array element type: %T", taskName, item)
						errChan <- err
						return
					}
				}
				err = le.executeDirectCommand(strArray)
			default:
				err = fmt.Errorf("task %s: invalid command type: %T", taskName, taskCmd)
			}

			if err != nil {
				errChan <- fmt.Errorf("task %s: %w", taskName, err)
			}
		}(name, cmd)
	}

	wg.Wait()
	close(errChan)

	// Collect all errors
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	if len(errors) == 0 {
		return nil
	}

	// Return single error or combined error message
	if len(errors) == 1 {
		return errors[0]
	}

	// Multiple errors - combine them
	errMsg := "multiple tasks failed:"
	for _, err := range errors {
		errMsg += fmt.Sprintf("\n  - %s", err.Error())
	}
	return fmt.Errorf("%s", errMsg)
}
