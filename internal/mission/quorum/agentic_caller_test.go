package quorum

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

// stubAgenticRunner is an AgenticRunner test double. It records the prompts it
// was handed and returns a canned run, without touching any real executor.
type stubAgenticRunner struct {
	run *AgenticRun
	err error

	gotSystem string
	gotUser   string
	called    bool
}

func (s *stubAgenticRunner) Run(_ context.Context, sysPrompt, userPrompt string) (*AgenticRun, error) {
	s.called = true
	s.gotSystem = sysPrompt
	s.gotUser = userPrompt
	return s.run, s.err
}

func TestRunAgenticReviewer_ValidVerdict(t *testing.T) {
	stub := &stubAgenticRunner{
		run: &AgenticRun{
			Success: true,
			CostUSD: 0.05,
			Output:  `{"verdict":"reject","strongest_objection":"premise X is FALSE — ran ailang check on the cited repro, it passed","catch":"grepped internal/foo.go for BarFunc; absent"}`,
		},
	}
	out := RunAgenticReviewer("gpt5-6-sol@codex", "doc.md", "body", "", DefaultMaxCostUSD, time.Minute, stub.Run)

	if !out.Present {
		t.Fatalf("expected Present, got absent: %s", out.Err)
	}
	if out.Result.Verdict != VerdictReject {
		t.Errorf("verdict = %q, want reject", out.Result.Verdict)
	}
	if out.CostUSD != 0.05 {
		t.Errorf("observed cost = %f, want 0.05", out.CostUSD)
	}
	// The agentic system prompt must ride the SHIPPED reviewer systemPrompt +
	// the verification instruction.
	if !strings.Contains(stub.gotSystem, "REJECT by default") {
		t.Errorf("shipped reviewer prompt not carried into the agentic call")
	}
	if !strings.Contains(stub.gotSystem, "READ-ONLY repository tools") {
		t.Errorf("agentic verification instruction missing from system prompt")
	}
}

func TestRunAgenticReviewer_OverCapIsBudgetAbsence(t *testing.T) {
	stub := &stubAgenticRunner{
		run: &AgenticRun{
			Success: true,
			CostUSD: 0.50, // >> cap
			Output:  `{"verdict":"pass","strongest_objection":"none","catch":"looks verified"}`,
		},
	}
	out := RunAgenticReviewer("m@codex", "doc.md", "body", "", DefaultMaxCostUSD, time.Minute, stub.Run)

	if out.Present {
		t.Fatalf("expected absent for over-cap run")
	}
	if out.AbsentReason != ReasonBudget {
		t.Errorf("absent reason = %q, want %q", out.AbsentReason, ReasonBudget)
	}
	if out.Result != nil {
		t.Errorf("over-cap outcome must not carry a verdict")
	}
	// Cost is still recorded for audit.
	if out.CostUSD != 0.50 {
		t.Errorf("observed cost not recorded on budget absence: %f", out.CostUSD)
	}
}

func TestRunAgenticReviewer_MalformedOutputIsInvalidAbsence(t *testing.T) {
	stub := &stubAgenticRunner{
		run: &AgenticRun{
			Success: true,
			CostUSD: 0.01,
			Output:  `{"verdict":"reject","strongest_objection":"","catch":""}`, // gate violation
		},
	}
	out := RunAgenticReviewer("m@codex", "doc.md", "body", "", DefaultMaxCostUSD, time.Minute, stub.Run)

	if out.Present {
		t.Fatalf("expected absent for gate-violating output")
	}
	if out.AbsentReason != ReasonInvalid {
		t.Errorf("absent reason = %q, want %q", out.AbsentReason, ReasonInvalid)
	}
}

func TestRunAgenticReviewer_ExecutorErrorIsUnreachable(t *testing.T) {
	stub := &stubAgenticRunner{err: errors.New("executor killed: context deadline exceeded")}
	out := RunAgenticReviewer("m@codex", "doc.md", "body", "", DefaultMaxCostUSD, time.Minute, stub.Run)

	if out.Present {
		t.Fatalf("expected absent for executor error")
	}
	if out.AbsentReason != ReasonUnreachable {
		t.Errorf("absent reason = %q, want %q", out.AbsentReason, ReasonUnreachable)
	}
}

func TestRunAgenticReviewer_RunFailureIsUnreachable(t *testing.T) {
	// Success=false (executor ran but the agent failed) is also a named absence,
	// never a silent pass.
	stub := &stubAgenticRunner{
		run: &AgenticRun{Success: false, Err: "agent hit an internal error", CostUSD: 0.02},
	}
	out := RunAgenticReviewer("m@codex", "doc.md", "body", "", DefaultMaxCostUSD, time.Minute, stub.Run)

	if out.Present {
		t.Fatalf("expected absent for failed run")
	}
	if out.AbsentReason != ReasonUnreachable {
		t.Errorf("absent reason = %q, want %q", out.AbsentReason, ReasonUnreachable)
	}
}

// TestAgenticCaller_SatisfiesJSONCaller is a compile-time assertion that the
// agentic caller rides the EXISTING JSONCaller seam (call.go:35) with NO
// interface change.
func TestAgenticCaller_SatisfiesJSONCaller(t *testing.T) {
	var _ JSONCaller = &agenticCaller{}
}

// TestBuildAgenticPrompt_FocusesContestedPremise proves the escalated prompt
// carries the specific contested premise (Tier-2 focus) while the un-escalated
// prompt is just the base doc review.
func TestBuildAgenticPrompt_FocusesContestedPremise(t *testing.T) {
	base := BuildAgenticPrompt("doc.md", "body", "")
	if strings.Contains(base, "ESCALATED VERIFICATION") {
		t.Errorf("un-escalated prompt should not contain escalation framing")
	}
	focused := BuildAgenticPrompt("doc.md", "body", "premise: internal/foo.go exports BarFunc")
	if !strings.Contains(focused, "ESCALATED VERIFICATION") {
		t.Errorf("escalated prompt missing escalation framing")
	}
	if !strings.Contains(focused, "internal/foo.go exports BarFunc") {
		t.Errorf("escalated prompt missing the contested premise")
	}
}
