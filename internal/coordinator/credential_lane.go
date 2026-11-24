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

import (
	"context"
	"fmt"
	"time"
)

// laneCodexSubscription names the cloud codex subscription credential.
const laneCodexSubscription = "codex-subscription"

// laneAgySubscription names the cloud agy (Google AI subscription) credential: a
// token file agy rewrites on refresh, leased exactly like codex's auth.json
// (M-AGY-CLOUD D2: one cloud agy run at a time).
const laneAgySubscription = "agy-subscription"

// A failed task keeps its lane for its agent's timeout plus this grace (D-FLEET-39).
//
// The executor jobs run with maxRetries: 1, and a `failed` completion comes
// from an attempt that exits non-zero, so Cloud Run starts a retry of the SAME
// execution — and the retry takes the lease again. The coordinator cannot see
// the retry: it is not told the attempt number, and the retry's own
// completion is skipped because the task is already terminal. So the lane
// stays held for as long as the retry can run: the agent's timeout (the
// executor's per-attempt limit) plus this grace for the retry's start, clone
// and credential write-back. Measured 2026-10-16: attempt 0 of
// ailang-agent-executor-codex-go-dgcmd released the lease and published
// `failed` for task-d61964fa at 08:44:03Z, attempt 1 re-acquired it at
// 08:44:53Z, and the two codex tasks dispatched at 08:47:47Z and 08:52:49Z
// both died on "credential already owned".
const credentialLaneRetryGrace = 15 * time.Minute

// credentialLaneFailedScan bounds the read of recently failed tasks, newest
// first by created_at: on Firestore that is the existing (status ASC,
// created_at DESC) index, `tasks_status` in the multivac terraform, so the
// query needs no new index. A task failed within its retry window is near the
// top; one created long before 50 later failures is missed, which is today's
// behaviour, not a new failure mode.
const credentialLaneFailedScan = 50

// credentialLane returns the shared-credential lane a dispatch would occupy, or
// "" when its credential is not exclusive. Every cloud codex job is on the
// subscription: the metered codex key went with the *-apikey jobs (2026-11-10).
func credentialLane(provider string) string {
	switch provider {
	case "codex":
		return laneCodexSubscription
	case "agy":
		return laneAgySubscription
	}
	return ""
}

// credentialLaneHolder returns the ID of a task other than self that occupies
// lane, or "" when the lane is free. A task occupies its lane while it is in
// flight (queued or running), and after it fails for as long as its Cloud Run
// retry may still hold the credential (credentialLaneRetryGrace). claimed holds
// lanes already dispatched earlier in this tick, which the store may not show
// yet.
//
// Holding never takes over a lease: a lane that stays held is a task that
// waits, while a lane freed too early is a task that dies at acquire.
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
		if credentialLane(resolveDispatchProvider(d.agentRegistry, t.AgentID, d.coordConfig)) == lane {
			return t.ID, nil
		}
	}
	failed, err := d.taskStore.ListTasks(ctx, &TaskFilter{Status: []TaskStatus{TaskStatusFailed}, OrderDesc: true, Limit: credentialLaneFailedScan})
	if err != nil {
		return "", err
	}
	now := time.Now()
	for _, t := range failed {
		if t.ID == self || t.CompletedAt == nil {
			continue
		}
		if credentialLane(resolveDispatchProvider(d.agentRegistry, t.AgentID, d.coordConfig)) != lane {
			continue
		}
		var agent *AgentConfig
		if d.agentRegistry != nil {
			agent = d.agentRegistry.GetAgentByID(t.AgentID)
		}
		since := now.Sub(*t.CompletedAt)
		if window := agent.GetEffectiveTimeout() + credentialLaneRetryGrace; since < window {
			return fmt.Sprintf("%s (failed %s ago; its Cloud Run retry may hold the credential for up to %s)",
				t.ID, since.Round(time.Second), window), nil
		}
	}
	return "", nil
}
