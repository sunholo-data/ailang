package main

import (
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/coordinator"
)

// The policy decides whether a session is told it may resolve a row, so every
// branch that could turn a "Mark's decision" into a "you may" is pinned here.
// The dangerous direction is one-way: a false negative costs a wait, a false
// positive merges a change nobody reviewed.

func TestAgentMayApprove_NeverCoversNothing(t *testing.T) {
	// A row that would pass every other test still fails under the default.
	row := pendingApprovalJSON{DiffAvailable: true, Evaluation: "PASS"}
	ok, reason := agentMayApprove(row, approvalPolicyNever)
	if ok {
		t.Fatalf("policy=never must cover no row; got actionable with reason %q", reason)
	}
	if reason == "" {
		t.Error("a refusal must say why — a bare false is unactionable in a banner")
	}
}

func TestAgentMayApprove_EvaluatedRequiresBothVerdictAndDiff(t *testing.T) {
	tests := []struct {
		name string
		row  pendingApprovalJSON
		want bool
	}{
		{"verdict PASS with a visible diff", pendingApprovalJSON{DiffAvailable: true, Evaluation: "PASS"}, true},
		{"lowercase pass still counts", pendingApprovalJSON{DiffAvailable: true, Evaluation: "pass"}, true},
		// The row that motivated the rule: an executor that recorded no diff.
		// An agent cannot compensate by "looking at the PR", so PASS alone is
		// not enough.
		{"PASS but no diff to review", pendingApprovalJSON{DiffAvailable: false, Evaluation: "PASS"}, false},
		{"diff but the evaluator never reported", pendingApprovalJSON{DiffAvailable: true, Evaluation: ""}, false},
		{"diff but the evaluator FAILED it", pendingApprovalJSON{DiffAvailable: true, Evaluation: "FAIL"}, false},
		{"diff but the evaluator was unavailable", pendingApprovalJSON{DiffAvailable: true, Evaluation: "UNAVAILABLE"}, false},
		{"neither", pendingApprovalJSON{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, reason := agentMayApprove(tt.row, approvalPolicyEvaluated)
			if got != tt.want {
				t.Errorf("actionable = %v, want %v (reason: %s)", got, tt.want, reason)
			}
		})
	}
}

func TestAgentMayApprove_AlwaysCoversEvenTheUnreviewable(t *testing.T) {
	// Deliberate: `always` is an explicit operator override, and it should not
	// quietly behave like `evaluated`. If it silently withheld the no-diff rows
	// the operator would have no policy that covers them at all.
	ok, _ := agentMayApprove(pendingApprovalJSON{DiffAvailable: false}, approvalPolicyAlways)
	if !ok {
		t.Error("policy=always must cover every pending row, including one with no diff")
	}
}

func TestResolveApprovalPolicy_DefaultsToNever(t *testing.T) {
	tests := map[string]approvalPolicy{
		"":            approvalPolicyNever,
		"never":       approvalPolicyNever,
		"evaluated":   approvalPolicyEvaluated,
		"  EVALUATED": approvalPolicyEvaluated,
		"always":      approvalPolicyAlways,
		// An unrecognised value must fall to the SAFE end, not the permissive
		// one — a typo'd policy should never widen what a session may merge.
		"evaluted": approvalPolicyNever,
		"yes":      approvalPolicyNever,
	}
	for value, want := range tests {
		t.Setenv("AILANG_APPROVAL_POLICY", value)
		if got := resolveApprovalPolicy(); got != want {
			t.Errorf("AILANG_APPROVAL_POLICY=%q → %q, want %q", value, got, want)
		}
	}
}

func TestParseApprovalContext_UnparseableIsNotEmpty(t *testing.T) {
	// A context that will not decode is exactly as unreviewable as one that is
	// missing. Rendering it as a clean zero is how "Files (0)" got approved
	// blind (#921).
	ctxData, ok := parseApprovalContext(`{"changed_files": [ this is not json`)
	if ok {
		t.Fatal("malformed context must not report as parsed")
	}
	if ctxData.DiffUnavailable == "" {
		t.Error("malformed context must state that the diff is unavailable, not stay silent")
	}
}

func TestBuildPendingApprovalJSON_ProjectsTheReviewableFacts(t *testing.T) {
	now := time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC)
	req := &coordinator.ApprovalRequestRecord{
		ID:          "apr-abc",
		TaskID:      "task-abc",
		Type:        "merge_handoff",
		Description: "Design: something",
		CreatedAt:   now.Add(-36 * time.Hour),
		Evaluation:  "PASS",
		ContextJSON: `{"handoff_targets":["sprint-planner"],"source_agent":"design-doc-creator",
		               "changed_files":["a.md","b.md"],"diff_stat":"2 files changed"}`,
	}

	row := buildPendingApprovalJSON(req, approvalPolicyEvaluated, now)

	if row.AgeHours != 36 {
		t.Errorf("age = %v hours, want 36", row.AgeHours)
	}
	if row.ChangedFiles != 2 || !row.DiffAvailable {
		t.Errorf("expected a visible 2-file diff, got files=%d available=%v", row.ChangedFiles, row.DiffAvailable)
	}
	// What approving STARTS is the part that makes this more than a merge.
	if len(row.HandoffTargets) != 1 || row.HandoffTargets[0] != "sprint-planner" {
		t.Errorf("handoff targets = %v, want [sprint-planner]", row.HandoffTargets)
	}
	if row.SourceAgent != "design-doc-creator" {
		t.Errorf("source agent = %q, want design-doc-creator", row.SourceAgent)
	}
	if !row.AgentActionable {
		t.Errorf("PASS + visible diff under policy=evaluated should be actionable; reason: %s", row.PolicyReason)
	}
}

func TestBuildPendingApprovalJSON_DiffUnavailableIsNeverActionable(t *testing.T) {
	now := time.Now()
	// The real prod shape (apr-10b5e305, 2026-09-08): an executor that recorded
	// no diff source. changed_files is absent AND diff_unavailable is set, and
	// either alone must be enough to withhold the row.
	req := &coordinator.ApprovalRequestRecord{
		ID:          "apr-nodiff",
		TaskID:      "task-nodiff",
		Type:        "merge",
		CreatedAt:   now,
		Evaluation:  "PASS",
		ContextJSON: `{"diff_unavailable":"this executor produced no diff source for the task"}`,
	}

	row := buildPendingApprovalJSON(req, approvalPolicyEvaluated, now)

	if row.DiffAvailable {
		t.Error("a row whose context declares the diff unavailable must not report diff_available")
	}
	if row.AgentActionable {
		t.Errorf("an unreviewable row must stay the operator's; reason: %s", row.PolicyReason)
	}
	if row.DiffUnavailable == "" {
		t.Error("the reason the diff is missing must survive into the row")
	}
}

func TestBuildPendingApprovalJSON_EmptyChangedFilesIsNotAVisibleDiff(t *testing.T) {
	now := time.Now()
	// No diff_unavailable marker, but nothing changed either. "Files (0)"
	// rendered confidently is the exact card that got approved blind.
	req := &coordinator.ApprovalRequestRecord{
		ID:          "apr-empty",
		TaskID:      "task-empty",
		CreatedAt:   now,
		Evaluation:  "PASS",
		ContextJSON: `{"changed_files":[]}`,
	}

	row := buildPendingApprovalJSON(req, approvalPolicyEvaluated, now)

	if row.DiffAvailable {
		t.Error("zero changed files is not a reviewable diff")
	}
	if row.AgentActionable {
		t.Errorf("a zero-file row must stay the operator's; reason: %s", row.PolicyReason)
	}
}
