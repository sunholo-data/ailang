package coordinator

import (
	"fmt"
	"github.com/sunholo-data/ailang/internal/messaging"
)

// TaskInput aliases the canonical wire type without making messaging depend on coordinator.
type TaskInput = messaging.TaskInput

// ValidateInputsForAgent uses only the trusted registry grant, never task content.
func ValidateInputsForAgent(inputs []TaskInput, agent *AgentConfig) error {
	if err := messaging.ValidateTaskInputs(inputs); err != nil {
		return fmt.Errorf("%w: %v", ErrDispatchPermanent, err)
	}
	if len(inputs) == 0 {
		return nil
	}
	name := "<unregistered>"
	allow := map[string]bool{}
	if agent != nil {
		name = agent.ID
		for _, repo := range agent.InputsAllow {
			if err := messaging.ValidateInputRepo(repo); err != nil {
				return fmt.Errorf("%w: inputs_allow for agent %q: %v", ErrDispatchPermanent, name, err)
			}
			allow[repo] = true
		}
	}
	for _, input := range inputs {
		if !allow[input.Repo] {
			return fmt.Errorf("%w: inputs: repo %q is not in inputs_allow for agent %q", ErrDispatchPermanent, input.Repo, name)
		}
	}
	return nil
}
