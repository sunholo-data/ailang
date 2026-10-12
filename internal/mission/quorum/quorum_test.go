package quorum

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
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

// zeroSignalObjection is the guard's synthetic objection. The fix keeps it verbatim.
const zeroSignalObjection = "no reviewer produced a verdict (all absent) — refusing to proceed on zero signal"

// costedAbsent is an absent outcome with non-zero cost and tokens, so the
// accounting assertions prove absent outcomes are summed rather than dropped.
func costedAbsent(model, reason string) *ReviewerOutcome {
	return &ReviewerOutcome{Model: model, Present: false, AbsentReason: reason, CostUSD: 0.02, TokensIn: 300, TokensOut: 40}
}

// costedPresent is a present outcome that also carries tokens.
func costedPresent(model string, v Verdict, obj string) *ReviewerOutcome {
	o := present(model, v, obj)
	o.TokensIn, o.TokensOut = 500, 70
	return o
}

// TestSynthesize_ExternalSignalMatrix pins the zero-signal guard to present
// EXTERNAL reviewers. The controller keeps its veto and its artifact entry, but
// a controller pass cannot supply the external-signal minimum on its own.
func TestSynthesize_ExternalSignalMatrix(t *testing.T) {
	pass := &ControllerReview{Verdict: VerdictPass, Note: "controller passes"}
	reject := &ControllerReview{Verdict: VerdictReject, Note: "controller veto"}
	controllerVeto := "controller (in-session): controller veto"
	twoAbsent := []*ReviewerOutcome{costedAbsent("m1", ReasonAuth), costedAbsent("m2", ReasonBudget)}
	absentPair := []AbsentNote{{Model: "m1", Reason: ReasonAuth}, {Model: "m2", Reason: ReasonBudget}}

	cases := []struct {
		name           string
		outcomes       []*ReviewerOutcome
		ctrl           *ControllerReview
		wantVerdict    SynthesisVerdict
		wantObjections []string
		wantAbsent     []AbsentNote
	}{
		{"two absent, controller nil", twoAbsent, nil, SynthBlocked, []string{zeroSignalObjection}, absentPair},
		{"two absent, controller pass", twoAbsent, pass, SynthBlocked, []string{zeroSignalObjection}, absentPair},
		{"two absent, controller reject", twoAbsent, reject, SynthBlocked, []string{controllerVeto, zeroSignalObjection}, absentPair},
		{"empty slice, controller nil", nil, nil, SynthBlocked, []string{zeroSignalObjection}, nil},
		{"empty slice, controller pass", nil, pass, SynthBlocked, []string{zeroSignalObjection}, nil},
		{"empty slice, controller reject", nil, reject, SynthBlocked, []string{controllerVeto, zeroSignalObjection}, nil},
		{"one pass, one absent, controller nil", []*ReviewerOutcome{costedPresent("m1", VerdictPass, "minor"), costedAbsent("m2", ReasonAuth)}, nil, SynthProceed, nil, []AbsentNote{{Model: "m2", Reason: ReasonAuth}}},
		{"one pass, one absent, controller pass", []*ReviewerOutcome{costedPresent("m1", VerdictPass, "minor"), costedAbsent("m2", ReasonAuth)}, pass, SynthProceed, nil, []AbsentNote{{Model: "m2", Reason: ReasonAuth}}},
		{"one pass, one absent, controller reject", []*ReviewerOutcome{costedPresent("m1", VerdictPass, "minor"), costedAbsent("m2", ReasonAuth)}, reject, SynthBlocked, []string{controllerVeto}, []AbsentNote{{Model: "m2", Reason: ReasonAuth}}},
		{"one reject, one absent, controller nil", []*ReviewerOutcome{costedPresent("m1", VerdictReject, "premise X unverified"), costedAbsent("m2", ReasonBudget)}, nil, SynthBlocked, []string{"m1: premise X unverified"}, []AbsentNote{{Model: "m2", Reason: ReasonBudget}}},
		{"one reject, one absent, controller pass", []*ReviewerOutcome{costedPresent("m1", VerdictReject, "premise X unverified"), costedAbsent("m2", ReasonBudget)}, pass, SynthBlocked, []string{"m1: premise X unverified"}, []AbsentNote{{Model: "m2", Reason: ReasonBudget}}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			table := map[string]*ReviewerOutcome{}
			models := []string{}
			var wantCost float64
			var wantIn, wantOut int
			for _, o := range c.outcomes {
				table[o.Model] = o
				models = append(models, o.Model)
				wantCost += o.CostUSD
				wantIn += o.TokensIn
				wantOut += o.TokensOut
			}
			q := RunQuorum("doc.md", "body", "t", models, 0.10, c.ctrl, fakeRunner(table))

			if q.Synthesis.Verdict != c.wantVerdict {
				t.Errorf("verdict = %q, want %q", q.Synthesis.Verdict, c.wantVerdict)
			}
			if !reflect.DeepEqual(q.Synthesis.BlockingObjections, c.wantObjections) {
				t.Errorf("blocking objections = %q, want %q", q.Synthesis.BlockingObjections, c.wantObjections)
			}
			if len(q.Synthesis.AbsentReviewers) != len(c.wantAbsent) {
				t.Fatalf("absent reviewers = %+v, want %+v", q.Synthesis.AbsentReviewers, c.wantAbsent)
			}
			for i, want := range c.wantAbsent {
				if q.Synthesis.AbsentReviewers[i] != want {
					t.Errorf("absent[%d] = %+v, want %+v (input order)", i, q.Synthesis.AbsentReviewers[i], want)
				}
			}
			if c.ctrl == nil {
				if q.ControllerInSession != nil {
					t.Errorf("controller recorded although none was supplied: %+v", q.ControllerInSession)
				}
			} else if q.ControllerInSession == nil || q.ControllerInSession.Verdict != c.ctrl.Verdict || q.ControllerInSession.Note != c.ctrl.Note {
				t.Errorf("controller entry = %+v, want %+v", q.ControllerInSession, c.ctrl)
			}
			// Totals include absent outcomes: accounting is not reduced by the gate.
			if d := q.Synthesis.TotalCostUSD - wantCost; d < -1e-9 || d > 1e-9 {
				t.Errorf("total cost = %f, want %f", q.Synthesis.TotalCostUSD, wantCost)
			}
			if q.Synthesis.TotalTokensIn != wantIn || q.Synthesis.TotalTokensOut != wantOut {
				t.Errorf("tokens in/out = %d/%d, want %d/%d", q.Synthesis.TotalTokensIn, q.Synthesis.TotalTokensOut, wantIn, wantOut)
			}
		})
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

// TestWriteJSONArtifact_NoClobberOnSameSecond proves two immediate successive
// writes for the SAME doc + SAME timestamp yield two distinct files rather than
// clobbering — the O_CREATE|O_EXCL + suffix-retry guard.
func TestWriteJSONArtifact_NoClobberOnSameSecond(t *testing.T) {
	q := RunQuorum("design_docs/foo-bar.md", "body", "2026-07-14T12:00:00Z", []string{"m1"}, 0.10, nil,
		fakeRunner(map[string]*ReviewerOutcome{"m1": present("m1", VerdictPass, "minor")}))

	dir := t.TempDir()
	p1, err := WriteJSONArtifact(dir, q)
	if err != nil {
		t.Fatalf("first write: %v", err)
	}
	p2, err := WriteJSONArtifact(dir, q)
	if err != nil {
		t.Fatalf("second write: %v", err)
	}
	if p1 == p2 {
		t.Fatalf("same-second writes clobbered: both wrote %s", p1)
	}
	// Both files must exist on disk (neither overwritten).
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Fatalf("want 2 artifact files, got %d: %v", len(entries), names)
	}
}

// TestSynthesis_AbsentReviewersMarshalsAsEmptySlice proves the absent_reviewers
// field marshals as [] (not null) when all reviewers were present.
func TestSynthesis_AbsentReviewersMarshalsAsEmptySlice(t *testing.T) {
	q := RunQuorum("doc.md", "body", "t", []string{"m1"}, 0.10, nil,
		fakeRunner(map[string]*ReviewerOutcome{"m1": present("m1", VerdictPass, "minor")}))
	if q.Synthesis.AbsentReviewers == nil {
		t.Fatal("AbsentReviewers is nil; want non-nil empty slice")
	}
	b, err := json.Marshal(q.Synthesis)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"absent_reviewers":[]`) {
		t.Errorf("absent_reviewers did not marshal as []:\n%s", b)
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
