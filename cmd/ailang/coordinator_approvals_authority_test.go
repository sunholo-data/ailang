package main

import "testing"

// Every branch that could turn "this is Mark's decision" into "you may decide".
// The two directions are not symmetric: a false negative costs a wait, a false
// positive merges a change nobody reviewed and dispatches the next agent.

// clearAuthorityEnv isolates a case from the ambient environment. Without it a
// test inherits whatever the developer's shell (or a mission driver) exported,
// and a grant test would pass for the wrong reason.
func clearAuthorityEnv(t *testing.T) {
	t.Helper()
	for _, v := range []string{
		"AILANG_APPROVAL_POLICY",
		"AILANG_APPROVAL_AUTHORITY_MODELS",
		"AILANG_APPROVAL_CONTROLLER",
		"MISSION_CONTROL_ACTIVE",
		"MISSION_ROLE",
		"CONTROLLER_ID",
	} {
		t.Setenv(v, "")
	}
}

func TestAuthority_DefaultIsNone(t *testing.T) {
	clearAuthorityEnv(t)
	got := resolveApprovalAuthority()
	if got.Granted {
		t.Fatalf("a bare session must have no authority; got grant %q", got.Reason)
	}
	if got.Reason == "" {
		t.Error("a refusal must say why, and how to obtain the grant")
	}
}

func TestAuthority_MissionControllerOnATrustedRung(t *testing.T) {
	tests := []struct {
		controllerID string
		want         bool
		why          string
	}{
		{"claude:claude-fable-5-1", true, "fable is a named trusted rung"},
		{"claude:claude-astra-5", true, "astra is a named trusted rung"},
		{"claude:claude-opus-5", true, "opus is a named trusted rung"},
		// A version bump must not silently revoke a grant.
		{"claude:claude-fable-5-2", true, "matched as a substring, not an exact id"},
		// The dry-out fallbacks keep the loop breathing; they were never chosen
		// for judgment, and a degraded controller must not inherit the grant.
		{"codex:gpt-5.6-sol", false, "codex is a fallback rung"},
		{"pi:ollama/glm-5.3:cloud", false, "glm is a last-resort rung"},
		{"pi:openrouter/z-ai/glm-5.3", false, "glm is a last-resort rung"},
	}
	for _, tt := range tests {
		t.Run(tt.controllerID, func(t *testing.T) {
			clearAuthorityEnv(t)
			t.Setenv("MISSION_CONTROL_ACTIVE", "1")
			t.Setenv("CONTROLLER_ID", tt.controllerID)

			got := resolveApprovalAuthority()
			if got.Granted != tt.want {
				t.Errorf("granted = %v, want %v (%s) — reason: %s", got.Granted, tt.want, tt.why, got.Reason)
			}
			// Granted or not, the decider must be named: an approval with no
			// author cannot be audited afterwards.
			if got.Identity == "" {
				t.Error("a mission controller must be identified either way")
			}
		})
	}
}

func TestAuthority_UnnamedControllerIsNotTrusted(t *testing.T) {
	clearAuthorityEnv(t)
	// MISSION_CONTROL_ACTIVE without CONTROLLER_ID means the driver's ladder
	// walk did not publish which rung it got. Granting here would trust an
	// unknown model.
	t.Setenv("MISSION_CONTROL_ACTIVE", "1")
	if got := resolveApprovalAuthority(); got.Granted {
		t.Errorf("an unnamed controller must not be trusted; got %q", got.Reason)
	}
}

func TestAuthority_ASpawnedRoleNeverDecides(t *testing.T) {
	// The load-bearing rule: generator ≠ judge. A role running on the STRONGEST
	// model, with the controller flags also set, still must not approve — it is
	// the agent that produced the work.
	for _, role := range []string{"designer", "planner", "executor", "evaluator"} {
		t.Run(role, func(t *testing.T) {
			clearAuthorityEnv(t)
			t.Setenv("MISSION_ROLE", role)
			t.Setenv("MISSION_CONTROL_ACTIVE", "1")
			t.Setenv("CONTROLLER_ID", "claude:claude-fable-5-1")

			got := resolveApprovalAuthority()
			if got.Granted {
				t.Errorf("a spawned %s must never approve, even on fable; got %q", role, got.Reason)
			}
		})
	}
}

func TestAuthority_ControllerRoleIsTheController(t *testing.T) {
	clearAuthorityEnv(t)
	// MISSION_ROLE=controller is the controller itself, not a spawned role, so
	// the role check must not strip its grant.
	t.Setenv("MISSION_ROLE", "controller")
	t.Setenv("MISSION_CONTROL_ACTIVE", "1")
	t.Setenv("CONTROLLER_ID", "claude:claude-fable-5-1")

	if got := resolveApprovalAuthority(); !got.Granted {
		t.Errorf("the controller must keep its grant; got %q", got.Reason)
	}
}

func TestAuthority_AttendedGrantIsExplicit(t *testing.T) {
	clearAuthorityEnv(t)
	if got := resolveApprovalAuthority(); got.Granted {
		t.Fatal("an attended session must not be granted by default")
	}
	t.Setenv("AILANG_APPROVAL_CONTROLLER", "1")
	got := resolveApprovalAuthority()
	if !got.Granted {
		t.Errorf("the standing operator grant must be honoured; got %q", got.Reason)
	}
	if !got.RequireReviewable {
		t.Error("an attended grant still must not cover a row with no visible diff")
	}
}

func TestAuthority_AllowlistIsConfigurable(t *testing.T) {
	clearAuthorityEnv(t)
	t.Setenv("MISSION_CONTROL_ACTIVE", "1")
	t.Setenv("CONTROLLER_ID", "codex:gpt-5.6-sol")
	if resolveApprovalAuthority().Granted {
		t.Fatal("codex is not trusted by default")
	}
	t.Setenv("AILANG_APPROVAL_AUTHORITY_MODELS", "gpt-5.6-sol")
	if !resolveApprovalAuthority().Granted {
		t.Error("an explicitly allowlisted rung must be trusted")
	}
	// An empty or whitespace-only override must fall back to the defaults
	// rather than trusting nothing or, worse, everything.
	t.Setenv("AILANG_APPROVAL_AUTHORITY_MODELS", "  , ")
	t.Setenv("CONTROLLER_ID", "claude:claude-fable-5-1")
	if !resolveApprovalAuthority().Granted {
		t.Error("a blank allowlist must fall back to the defaults, not revoke fable")
	}
}

func TestAuthority_OnlyAlwaysCoversAnUnreviewableRow(t *testing.T) {
	noDiff := pendingApprovalJSON{DiffAvailable: false, Evaluation: "PASS"}

	clearAuthorityEnv(t)
	t.Setenv("MISSION_CONTROL_ACTIVE", "1")
	t.Setenv("CONTROLLER_ID", "claude:claude-fable-5-1")
	if ok, reason := resolveApprovalAuthority().coversRow(noDiff); ok {
		t.Errorf("even a trusted controller must not approve what it cannot see (%s)", reason)
	}

	// `always` is the deliberate operator override, and it must actually mean it
	// — a policy that silently behaved like the stricter one would leave the
	// no-diff rows covered by nothing at all.
	t.Setenv("AILANG_APPROVAL_POLICY", "always")
	if ok, reason := resolveApprovalAuthority().coversRow(noDiff); !ok {
		t.Errorf("policy=always must cover an unreviewable row; got %q", reason)
	}
}

func TestAuthority_EvaluatedDemandsAVerdict(t *testing.T) {
	clearAuthorityEnv(t)
	t.Setenv("AILANG_APPROVAL_POLICY", "evaluated")
	auth := resolveApprovalAuthority()

	if ok, _ := auth.coversRow(pendingApprovalJSON{DiffAvailable: true, Evaluation: "PASS"}); !ok {
		t.Error("PASS with a visible diff must be covered")
	}
	for _, verdict := range []string{"", "FAIL", "UNAVAILABLE"} {
		if ok, _ := auth.coversRow(pendingApprovalJSON{DiffAvailable: true, Evaluation: verdict}); ok {
			t.Errorf("verdict %q must not be covered under evaluated", verdict)
		}
	}
}

func TestAuthority_ControllerGrantDoesNotDemandAVerdict(t *testing.T) {
	clearAuthorityEnv(t)
	t.Setenv("MISSION_CONTROL_ACTIVE", "1")
	t.Setenv("CONTROLLER_ID", "claude:claude-fable-5-1")

	// A controller grant is trust in the DECIDER. Requiring a verdict on top
	// would cover nothing on the package lanes — measured 2026-09-08: all 6
	// pending prod rows came from pkg agents, which have no sprint-evaluator
	// edge and therefore never carry a verdict.
	ok, reason := resolveApprovalAuthority().coversRow(pendingApprovalJSON{DiffAvailable: true, Evaluation: ""})
	if !ok {
		t.Errorf("a trusted controller must cover a reviewable row with no evaluator edge; got %q", reason)
	}
}

func TestResolveApprovalPolicy_UnknownValuesFallSafe(t *testing.T) {
	for value, want := range map[string]approvalPolicy{
		"":            approvalPolicyNever,
		"never":       approvalPolicyNever,
		"evaluated":   approvalPolicyEvaluated,
		"  EVALUATED": approvalPolicyEvaluated,
		"always":      approvalPolicyAlways,
		// A typo must never widen what a session may merge.
		"evaluted": approvalPolicyNever,
		"yes":      approvalPolicyNever,
	} {
		t.Setenv("AILANG_APPROVAL_POLICY", value)
		if got := resolveApprovalPolicy(); got != want {
			t.Errorf("AILANG_APPROVAL_POLICY=%q → %q, want %q", value, got, want)
		}
	}
}

func TestIdentityLabel_NamesAnAttendedSession(t *testing.T) {
	clearAuthorityEnv(t)
	t.Setenv("CLAUDECODE", "1")
	t.Setenv("CLAUDE_CODE_SESSION_ID", "sess-123")
	// This is what lands in the approval's resolved_by, so it has to identify
	// the decider rather than the login the fleet happens to share.
	if got := identityLabel(); got != "attended claude-code session sess-123" {
		t.Errorf("identity = %q, want the session named", got)
	}

	t.Setenv("CONTROLLER_ID", "claude:claude-fable-5-1")
	if got := identityLabel(); got != "controller claude:claude-fable-5-1" {
		t.Errorf("identity = %q, want the controller named", got)
	}
}
