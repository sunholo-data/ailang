package coordinator

import "testing"

// M-COORDINATOR-EXECUTION-TRUST M1a.
//
// These arms exist because the first draft of M1a said "the task type is
// already on the record, so the classifier need not guess." The field IS on the
// record — and it is computed by classifyTaskType(), a substring match over
// SENDER-CONTROLLED message content (design doc V18). Quorum round 0 caught the
// authority hole; these tests are what stop it coming back.

func TestResolveWorkTier_NoAgentFailsClosed(t *testing.T) {
	if got := ResolveWorkTier(nil, ""); got != WorkTier2 {
		t.Fatalf("nil agent must fail closed to %q, got %q", WorkTier2, got)
	}
}

func TestResolveWorkTier_UnsetFailsClosed(t *testing.T) {
	if got := ResolveWorkTier(&AgentConfig{ID: "a"}, ""); got != WorkTier2 {
		t.Fatalf("unset tier must fail closed to %q, got %q", WorkTier2, got)
	}
}

// MU-5e: an unknown or model-supplied value is not a tier-1 grant.
func TestUnknownTierFailsClosedToTier2(t *testing.T) {
	for _, v := range []WorkTier{"tier0", "TIER1", "1", "routine", " tier1", "admin"} {
		if got := ResolveWorkTier(&AgentConfig{ID: "a", WorkTier: v}, ""); got != WorkTier2 {
			t.Errorf("unknown tier %q must fail closed to %q, got %q", v, WorkTier2, got)
		}
	}
}

func TestResolveWorkTier_ExplicitTier1IsHonored(t *testing.T) {
	if got := ResolveWorkTier(&AgentConfig{ID: "a", WorkTier: WorkTier1}, ""); got != WorkTier1 {
		t.Fatalf("explicit tier1 in trusted config must resolve to %q, got %q", WorkTier1, got)
	}
}

// V24 / quorum round 2: the direct-push path has no PR containment, so it never
// receives the tier-1 auto-disarm floor no matter what the agent config says.
func TestPushBranchRefusesTier1(t *testing.T) {
	agent := &AgentConfig{ID: "a", WorkTier: WorkTier1}
	if got := ResolveWorkTier(agent, "main"); got != WorkTier2 {
		t.Fatalf("a PushBranch dispatch must never get tier1 (V24), got %q", got)
	}
	if got := ResolveWorkTier(agent, ""); got != WorkTier1 {
		t.Fatalf("control: without PushBranch the same agent must be tier1, got %q", got)
	}
}

// MU-5d — THE load-bearing arm. Tier must not be derivable from message
// content. Every string below classifies as TaskTypeBugFix via classifyTaskType
// (that is asserted, so the arm cannot pass vacuously by picking inputs the
// classifier ignores), yet none of them may buy tier 1.
func TestTierIsNotDerivableFromMessageContent(t *testing.T) {
	senderControlled := []string{
		"please fix this bug",
		"error: everything is broken",
		"this is wrong and it failed",
		"issue with the parser, crash on startup",
	}
	untrusted := &AgentConfig{ID: "untrusted"} // no tier in trusted config

	for _, content := range senderControlled {
		if ct := classifyTaskType(content); ct != TaskTypeBugFix {
			t.Fatalf("precondition: %q must classify as %q for this arm to mean anything, got %q",
				content, TaskTypeBugFix, ct)
		}
		if got := ResolveWorkTier(untrusted, ""); got != WorkTier2 {
			t.Errorf("content %q must not buy tier1 — tier comes from trusted config only, got %q",
				content, got)
		}
	}
}
