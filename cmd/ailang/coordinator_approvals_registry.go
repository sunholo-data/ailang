package main

import (
	"context"
	"fmt"

	"github.com/sunholo-data/ailang/internal/coordinator"
)

// checkRegistryCanDispatch refuses an approval whose handoffs this process could
// not fire.
//
// The registry is the ONLY source of an agent's TriggerOnComplete — handoff
// topology never comes from the task, the message, or model output. So a
// registry that does not know the task's agent cannot dispatch anything, and an
// approval made against it is recorded, irreversible, and inert.
//
// This is deliberately a refusal rather than a warning. The approval write and
// the handoff are not one atomic unit; once the record says "approved" the task
// cannot be re-approved (ProcessApprovalRequest rejects a non-pending approval),
// so a warning would leave the operator holding a task that can never hand off.
func checkRegistryCanDispatch(
	ctx context.Context,
	bundle *coordinatorStoreBundle,
	registry *coordinator.AgentRegistry,
	regErr error,
	taskID string,
) error {
	task, err := bundle.Store.GetTask(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to read task %s before approving: %w", taskID, err)
	}
	if task == nil {
		return fmt.Errorf("task %s not found on this plane (%s)", taskID, bundle.Mode)
	}
	// A task with no agent has no handoff topology to lose.
	if task.AgentID == "" {
		return nil
	}

	if registry != nil && registry.GetAgentByID(task.AgentID) != nil {
		return nil // the registry knows it; handoffs can fire
	}

	detail := "the loaded agent registry has no entry for it"
	if regErr != nil {
		detail = fmt.Sprintf("the agent registry failed to load: %v", regErr)
	}

	return fmt.Errorf(
		"refusing to approve %s: its agent %q cannot be resolved — %s.\n"+
			"  Handoff topology (trigger_on_complete) comes ONLY from the registry, so this\n"+
			"  approval would be recorded and dispatch nothing, and the task could not be\n"+
			"  approved again. Point AILANG_CONFIG at the config that defines the agents on\n"+
			"  this plane, e.g. for %s:\n"+
			"    export AILANG_CONFIG=/path/to/ailang-multivac/config/config.cloud.yaml",
		taskID, task.AgentID, detail, bundle.Mode)
}
