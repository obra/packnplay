package runner

import (
	"fmt"
	"sync"

	"github.com/obra/packnplay/pkg/devcontainer"
)

// LifecycleExecutor executes lifecycle commands in a container.
type LifecycleExecutor struct {
	client        DockerClient
	containerName string
	containerUser string
	verbose       bool
}

// NewLifecycleExecutor creates a new lifecycle executor.
func NewLifecycleExecutor(client DockerClient, containerName, containerUser string, verbose bool) *LifecycleExecutor {
	return &LifecycleExecutor{
		client:        client,
		containerName: containerName,
		containerUser: containerUser,
		verbose:       verbose,
	}
}

// Execute executes a lifecycle command in the container.
func (le *LifecycleExecutor) Execute(cmd *devcontainer.LifecycleCommand) error {
	if cmd == nil {
		return nil
	}

	// Handle different command types
	if cmd.IsString() {
		str, _ := cmd.AsString()
		return le.executeShellCommand(str)
	}

	if cmd.IsArray() {
		arr, _ := cmd.AsArray()
		return le.executeDirectCommand(arr)
	}

	if cmd.IsObject() {
		obj, _ := cmd.AsObject()
		return le.executeParallelCommands(obj)
	}

	return fmt.Errorf("unknown lifecycle command type")
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

	// Check for errors
	for err := range errChan {
		return err // Return first error
	}

	return nil
}
