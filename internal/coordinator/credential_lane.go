package coordinator

// Credential lanes: agents whose cloud jobs share ONE rotating credential.
//
// A cloud codex job on the ChatGPT subscription takes an exclusive lease on the
// single auth.json secret (internal/credentiallease) because codex rotates the
// refresh token, so two concurrent holders would revoke each other. The lease
// fails fast — it cannot tell a live owner from a dead one — so a second job
// dispatched while the first runs exits at once with "credential already
// owned", turns=0, and the task is banked as an ordinary failure with no
// requeue. Measured 2026-10-12/13: four sprint-planner handoffs approved in
// pairs, two lost per pair (task-e0230f3b, task-fbaaf2f8, task-892cad50 ×2).
//
// The lease stays the hard guarantee; this gate keeps the dispatcher from
// launching a job that is certain to lose it. A task held back here stays
// pending and is retried on the next tick.

import "context"

// laneCodexSubscription names the cloud codex subscription credential.
const laneCodexSubscription = "codex-subscription"

// credentialLane returns the shared-credential lane a dispatch would occupy, or
// "" when its credential is not exclusive.
func credentialLane(agent *AgentConfig, provider string) string {
	if provider != "codex" {
		return ""
	}
	if agent != nil && agent.AuthMode == "apikey" {
		return "" // the metered key is not leased
	}
	return laneCodexSubscription
}

// credentialLaneHolder returns the ID of an in-flight task (queued or running)
// other than self that occupies lane, or "" when the lane is free. claimed
// holds lanes already dispatched earlier in this tick, which the store may not
// show yet.
func (d *Daemon) credentialLaneHolder(ctx context.Context, lane, self string, claimed map[string]string) (string, error) {
	if id, ok := claimed[lane]; ok {
		return id, nil
	}
	inflight, err := d.taskStore.ListTasks(ctx, &TaskFilter{Status: []TaskStatus{TaskStatusQueued, TaskStatusRunning}})
	if err != nil {
		return "", err
	}
	for _, t := range inflight {
		if t.ID == self {
			continue
		}
		var agent *AgentConfig
		if d.agentRegistry != nil {
			agent = d.agentRegistry.GetAgentByID(t.AgentID)
		}
		if credentialLane(agent, resolveDispatchProvider(d.agentRegistry, t.AgentID, d.coordConfig)) == lane {
			return t.ID, nil
		}
	}
	return "", nil
}
