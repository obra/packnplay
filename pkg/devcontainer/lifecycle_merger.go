package devcontainer

// LifecycleMerger handles merging feature and user lifecycle commands
type LifecycleMerger struct{}

// NewLifecycleMerger creates a new lifecycle merger
func NewLifecycleMerger() *LifecycleMerger {
	return &LifecycleMerger{}
}

// MergeCommands merges feature lifecycle commands with user commands
// Feature commands execute before user commands per specification
//
// The returned LifecycleCommand uses a special internal format to preserve
// multiple commands as separate entities that should be executed in sequence.
func (m *LifecycleMerger) MergeCommands(features []*ResolvedFeature, userCommands map[string]*LifecycleCommand) map[string]*LifecycleCommand {
	result := make(map[string]*LifecycleCommand)

	hookTypes := []string{"onCreateCommand", "updateContentCommand", "postCreateCommand", "postStartCommand", "postAttachCommand"}

	for _, hookType := range hookTypes {
		var mergedCommands []string

		// First, add feature commands in installation order
		for _, feature := range features {
			if feature.Metadata == nil {
				continue
			}

			var featureCommand *LifecycleCommand
			switch hookType {
			case "onCreateCommand":
				featureCommand = feature.Metadata.OnCreateCommand
			case "updateContentCommand":
				featureCommand = feature.Metadata.UpdateContentCommand
			case "postCreateCommand":
				featureCommand = feature.Metadata.PostCreateCommand
			case "postStartCommand":
				featureCommand = feature.Metadata.PostStartCommand
			case "postAttachCommand":
				featureCommand = feature.Metadata.PostAttachCommand
			}

			if featureCommand != nil {
				commands := featureCommand.ToStringSlice()
				mergedCommands = append(mergedCommands, commands...)
			}
		}

		// Then, add user commands
		if userCommand, exists := userCommands[hookType]; exists && userCommand != nil {
			commands := userCommand.ToStringSlice()
			mergedCommands = append(mergedCommands, commands...)
		}

		// Create merged lifecycle command if we have any commands
		if len(mergedCommands) > 0 {
			// Store as a MergedLifecycleCommand that preserves individual commands
			result[hookType] = &LifecycleCommand{
				raw: &MergedCommands{commands: mergedCommands},
			}
		}
	}

	return result
}

// MergedCommands represents multiple commands that should be executed in sequence
// This is an internal type used by the lifecycle merger
type MergedCommands struct {
	commands []string
}
