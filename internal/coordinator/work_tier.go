package coordinator

// M-COORDINATOR-EXECUTION-TRUST M1a — the permission tier, and where it may
// come from.
//
// The session-protocol gate unlocks mutating tools for tier-1 work once the
// prerequisite floor is satisfied, with no explicit ack. That is only safe if
// the tier itself cannot be chosen by whoever asked for the work.
//
// It very nearly could be. `Task.Type` is a closed enum and is on the record —
// which made "read the tier off the task" look correct — but it is assigned by
// classifyTaskType(task.Content), a substring match over message text whose
// bug-fix branch is tested FIRST against {bug, fix, error, crash, broken, issue,
// problem, fail, wrong}. A sender writing "please fix this" would have selected
// their own permission tier, and "feature work has no auto-path" would have been
// vacuous. Measured as V18 of the design doc; caught by quorum round 0.
//
// So the tier is read from the AGENT CONFIG — the coordinator's own registry,
// which a sender cannot write. The sender chooses an inbox (V25: `to_inbox` is
// a plain field on `ailang messages send`); the coordinator chooses which agent
// serves that inbox, and the agent's config carries the tier. The sender
// controls the first arrow only.
//
// classifyTaskType keeps its existing job — shaping the system prompt
// (meta_prompt.go) — and gains no authority it did not already have.

// WorkTier is a closed permission tier. Any value that is not exactly WorkTier1
// is tier 2; there is no third state and no "unset means permissive".
type WorkTier string

const (
	// WorkTier1 is routine work — bug fixes, triage, replies, docs. Mutating
	// tools unlock once the prerequisite floor is met; an explicit ack is still
	// recorded when the model makes one.
	WorkTier1 WorkTier = "tier1"

	// WorkTier2 is feature/semantics work, and the fail-closed default. No
	// auto-path: it requires an explicit ack plus verified design evidence.
	WorkTier2 WorkTier = "tier2"
)

// ResolveWorkTier returns the tier a dispatch runs under.
//
// Fail-closed at every boundary: a nil agent, an unset tier, and any
// unrecognised value all resolve to WorkTier2. Only an exact WorkTier1 in the
// trusted config grants the tier-1 floor.
//
// pushBranch is AILANG_PUSH_BRANCH for this dispatch. When it is set the agent
// commits straight to that branch with no coordinator branch and no PR
// (coordinator_cloud.go), so the always-PR containment this design leaned on
// does not exist for that path — and it does not get tier 1 regardless of what
// the config says. Design doc V24; caught by quorum round 2.
func ResolveWorkTier(agent *AgentConfig, pushBranch string) WorkTier {
	if agent == nil {
		return WorkTier2
	}
	if pushBranch != "" {
		return WorkTier2
	}
	if agent.WorkTier == WorkTier1 {
		return WorkTier1
	}
	return WorkTier2
}
