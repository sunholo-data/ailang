package coordinator

import "testing"

// A PR carries the label of what it is waiting for: the agent's own needs-*
// label, needs-merge-approval for agents without one, nothing when nothing waits.
// MU: return "" unconditionally, or drop the SkipApproval guard, and this fails.
func TestPRApprovalLabel(t *testing.T) {
	cases := []struct {
		agent *AgentConfig
		want  string
	}{
		{&AgentConfig{ID: "design-doc-creator"}, "needs-design-approval"},
		{&AgentConfig{ID: "sprint-planner"}, "needs-sprint-approval"},
		{&AgentConfig{ID: "sprint-executor"}, "needs-implementation-approval"},
		{&AgentConfig{ID: "ailang-core-triage"}, "needs-merge-approval"},
		{&AgentConfig{ID: "pkg-sunholo-email", SkipApproval: true}, ""},
		{&AgentConfig{ID: "custom", Approval: &ApprovalConfig{NeedsLabel: "needs-review"}}, "needs-review"},
	}
	for _, c := range cases {
		if got := PRApprovalLabel(c.agent); got != c.want {
			t.Errorf("%s: label %q, want %q", c.agent.ID, got, c.want)
		}
	}
}

// The PR's "merging starts X" must be exactly what approving the card
// dispatches: the non-auto edges. An auto edge already ran at completion.
func TestMergeStartsMatchesApprovalHandoffs(t *testing.T) {
	a := &AgentConfig{ID: "sprint-planner", TriggerOnComplete: []string{"sprint-executor", "sprint-evaluator"},
		AutoApproveHandoffTo: []string{"sprint-evaluator"}}
	got := approvalHandoffTargets(a)
	if len(got) != 1 || got[0] != "sprint-executor" {
		t.Errorf("approval handoffs = %v, want [sprint-executor]", got)
	}
}
