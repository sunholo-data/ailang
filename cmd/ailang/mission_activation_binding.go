package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/sunholo-data/ailang/internal/mission/activation"
	"github.com/sunholo-data/ailang/internal/mission/iteration"
)

func readMissionActivation(operationID string) (activation.Record, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return activation.Record{}, err
	}
	manager := activation.Manager{Dir: filepath.Join(home, ".ailang", "state", "mission-activations")}
	return manager.Inspect(operationID)
}

// Only inspection and prepare-only operations may call this helper. Mutating
// runtime commands continue to require the active machine binding.
func resolveMissionReadBinding(bindingPath, activationID, missionID, workItemID string) (iteration.Binding, error) {
	if activationID != "" {
		if bindingPath != "" {
			return iteration.Binding{}, fmt.Errorf("activation and explicit binding selection cannot be combined")
		}
		record, err := readMissionActivation(activationID)
		if err != nil {
			return iteration.Binding{}, err
		}
		if record.MissionID != missionID || record.WorkItemID != workItemID {
			return iteration.Binding{}, fmt.Errorf("activation %s belongs to %s/%s, not requested work item", activationID, record.MissionID, record.WorkItemID)
		}
		var binding iteration.Binding
		// Inspect validated the exact retained bytes and their hash. Never reload the
		// installation target: cleanup may already have restored a different binding.
		_, err = toml.Decode(string(record.Binding.Installed), &binding)
		return binding, err
	}
	if bindingPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return iteration.Binding{}, err
		}
		bindingPath = filepath.Join(home, ".config", "ailang", "mission-runtime.toml")
	}
	return iteration.LoadBinding(bindingPath)
}
