package quorum

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeRunner builds a deterministic per-model runner from a table so quorum
// tests never touch a real provider.
func fakeRunner(table map[string]*ReviewerOutcome) func(model, docPath, docBody string, maxCostUSD float64) *ReviewerOutcome {
	return func(model, _, _ string, _ float64) *ReviewerOutcome {
		if o, ok := table[model]; ok {
			return o
		}
		return &ReviewerOutcome{Model: model, AbsentReason: ReasonUnreachable, Err: "not in table"}
	}
}

func present(model string, v Verdict, obj string) *ReviewerOutcome {
	return &ReviewerOutcome{
		Model:   model,
		Present: true,
		CostUSD: 0.01,
		Result:  &ReviewResult{Verdict: v, StrongestObjection: obj, Catch: "c"},
	}
}

func TestRunQuorum_UnanimousPassProceeds(t *testing.T) {
	table := map[string]*ReviewerOutcome{
		"m1": present("m1", VerdictPass, "minor"),
		"m2": present("m2", VerdictPass, "minor"),
	}
	q := RunQuorum("doc.md", "body", "2026-07-14T00:00:00Z", []string{"m1", "m2"}, 0.10, nil, fakeRunner(table))
	if q.Synthesis.Verdict != SynthProceed {
		t.Fatalf("verdict = %q, want proceed", q.Synthesis.Verdict)
	}
	if len(q.Synthesis.BlockingObjections) != 0 {
		t.Errorf("unexpected blocking objections: %v", q.Synthesis.BlockingObjections)
	}
	if q.Synthesis.TotalCostUSD < 0.019 || q.Synthesis.TotalCostUSD > 0.021 {
		t.Errorf("total cost = %f, want ~0.02", q.Synthesis.TotalCostUSD)
	}
}

func TestRunQuorum_AnyRejectBlocks(t *testing.T) {
	table := map[string]*ReviewerOutcome{
		"m1": present("m1", VerdictPass, "minor"),
		"m2": present("m2", VerdictReject, "premise X unverified"),
	}
	q := RunQuorum("doc.md", "body", "t", []string{"m1", "m2"}, 0.10, nil, fakeRunner(table))
	if q.Synthesis.Verdict != SynthBlocked {
		t.Fatalf("verdict = %q, want blocked", q.Synthesis.Verdict)
	}
	if len(q.Synthesis.BlockingObjections) != 1 || !strings.Contains(q.Synthesis.BlockingObjections[0], "premise X unverified") {
		t.Errorf("blocking objections = %v, want the m2 objection", q.Synthesis.BlockingObjections)
	}
}

func TestRunQuorum_ControllerRejectBlocks(t *testing.T) {
	table := map[string]*ReviewerOutcome{
		"m1": present("m1", VerdictPass, "minor"),
	}
	ctrl := &ControllerReview{Verdict: VerdictReject, Note: "controller sees an axiom conflict"}
	q := RunQuorum("doc.md", "body", "t", []string{"m1"}, 0.10, ctrl, fakeRunner(table))
	if q.Synthesis.Verdict != SynthBlocked {
		t.Fatalf("verdict = %q, want blocked (controller rejected)", q.Synthesis.Verdict)
	}
	if q.ControllerInSession == nil || q.ControllerInSession.Verdict != VerdictReject {
		t.Errorf("controller review not recorded as a distinct entry")
	}
}

func TestRunQuorum_NMinus1DegradeNamesAbsentee(t *testing.T) {
	table := map[string]*ReviewerOutcome{
		"m1": present("m1", VerdictPass, "minor"),
		"m2": {Model: "m2", Present: false, AbsentReason: ReasonUnreachable, Err: "provider down"},
	}
	q := RunQuorum("doc.md", "body", "t", []string{"m1", "m2"}, 0.10, nil, fakeRunner(table))

	// Proceeds with N-1 (only m1 present, and it passed) ...
	if q.Synthesis.Verdict != SynthProceed {
		t.Fatalf("verdict = %q, want proceed on N-1", q.Synthesis.Verdict)
	}
	// ... but the absence is NAMED, never silent.
	if len(q.Synthesis.AbsentReviewers) != 1 {
		t.Fatalf("absent reviewers = %v, want 1", q.Synthesis.AbsentReviewers)
	}
	if q.Synthesis.AbsentReviewers[0].Model != "m2" || q.Synthesis.AbsentReviewers[0].Reason != ReasonUnreachable {
		t.Errorf("absentee not named correctly: %+v", q.Synthesis.AbsentReviewers[0])
	}
}

func TestRunQuorum_AllAbsentRefusesToProceed(t *testing.T) {
	table := map[string]*ReviewerOutcome{
		"m1": {Model: "m1", Present: false, AbsentReason: ReasonAuth},
		"m2": {Model: "m2", Present: false, AbsentReason: ReasonBudget},
	}
	q := RunQuorum("doc.md", "body", "t", []string{"m1", "m2"}, 0.10, nil, fakeRunner(table))
	// Zero signal must NOT be a silent proceed.
	if q.Synthesis.Verdict != SynthBlocked {
		t.Fatalf("verdict = %q, want blocked (all absent = zero signal)", q.Synthesis.Verdict)
	}
	if len(q.Synthesis.AbsentReviewers) != 2 {
		t.Errorf("want 2 named absentees, got %d", len(q.Synthesis.AbsentReviewers))
	}
}

func TestWriteJSONArtifactAndMarkdown(t *testing.T) {
	table := map[string]*ReviewerOutcome{
		"m1": present("m1", VerdictReject, "objection Y"),
		"m2": {Model: "m2", Present: false, AbsentReason: ReasonBudget},
	}
	q := RunQuorum("design_docs/foo-bar.md", "body", "2026-07-14T12:00:00Z", []string{"m1", "m2"}, 0.10, nil, fakeRunner(table))

	dir := t.TempDir()
	path, err := WriteJSONArtifact(dir, q)
	if err != nil {
		t.Fatalf("write artifact: %v", err)
	}
	if !strings.HasPrefix(filepath.Base(path), "foo-bar-") || !strings.HasSuffix(path, ".json") {
		t.Errorf("artifact name unexpected: %s", path)
	}
	// Round-trips to valid JSON with the expected shape.
	b, _ := os.ReadFile(path)
	var back QuorumResult
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatalf("artifact not valid JSON: %v", err)
	}
	if back.Synthesis.Verdict != SynthBlocked {
		t.Errorf("round-tripped verdict = %q, want blocked", back.Synthesis.Verdict)
	}

	md := MarkdownBlock(q)
	for _, want := range []string{"BLOCKED", "m1", "reject", "objection Y", "ABSENT", "budget"} {
		if !strings.Contains(md, want) {
			t.Errorf("markdown block missing %q:\n%s", want, md)
		}
	}
}

func TestAppendMarkdownToLog(t *testing.T) {
	q := RunQuorum("doc.md", "body", "t", []string{"m1"}, 0.10, nil,
		fakeRunner(map[string]*ReviewerOutcome{"m1": present("m1", VerdictPass, "minor")}))
	logFile := filepath.Join(t.TempDir(), "mission-log.md")
	if err := os.WriteFile(logFile, []byte("# existing\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := AppendMarkdownToLog(logFile, q); err != nil {
		t.Fatalf("append: %v", err)
	}
	b, _ := os.ReadFile(logFile)
	if !strings.Contains(string(b), "# existing") || !strings.Contains(string(b), "Design-quorum review") {
		t.Errorf("append did not preserve+add content:\n%s", b)
	}
}
