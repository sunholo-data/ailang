package quorum

import (
	"encoding/json"
	"math"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/ai"
	"github.com/sunholo-data/ailang/internal/coordinator"
)

// The token values below are deliberately ARBITRARY and DISTINCT per arm. An
// assertion against a status enum, a boolean, or a zero passes for any number
// of unrelated reasons; a specific pair of counts can only have come from the
// stub these arms inject.
const (
	textTierIn   = 1234
	textTierOut  = 567
	agenticIn    = 4321
	agenticOut   = 765
	failedRunIn  = 999
	failedRunOut = 111
)

const passVerdictJSON = `{"verdict":"pass","strongest_objection":"none — premises check out","catch":"ran the cited command"}`

// TestTextTierRecordsProviderTokens is the branch killer for the text tier's
// recording step. Before #708 the provider's counts were consumed by
// estimateCost and then discarded, so this arm reds if that assignment is
// removed — while the cost assertion beside it stays green, which is exactly
// why cost alone was never evidence that usage had been captured.
func TestTextTierRecordsProviderTokens(t *testing.T) {
	stub := &stubCaller{
		raw:  passVerdictJSON,
		resp: &ai.Response{InputTokens: textTierIn, OutputTokens: textTierOut},
	}
	out := runReviewerWith(stub, cheapModel(), &ReviewerOutcome{Model: "test-model"}, "doc.md", "body", DefaultMaxCostUSD)

	if !out.Present {
		t.Fatalf("expected Present, got absent: %s", out.Err)
	}
	if out.TokensIn != textTierIn || out.TokensOut != textTierOut {
		t.Errorf("tokens = %d/%d, want %d/%d — the provider's counts were read for cost and discarded (#708)",
			out.TokensIn, out.TokensOut, textTierIn, textTierOut)
	}
	// The cost arithmetic must be UNCHANGED by the recording step. Compared
	// with a tolerance, not for bit equality: measured on darwin/arm64 the two
	// call sites differ by 1 ULP (0x…aab vs 0x…aac) because the compiler is
	// free to contract `a*b + c*d` into an FMA in one inlining context and not
	// the other. A bit-exact assertion here would red for the FPU, not for the
	// change under test.
	want := estimateCost(cheapModel(), textTierIn, textTierOut)
	if math.Abs(out.CostUSD-want) > 1e-12 {
		t.Errorf("cost = %v, want %v — recording tokens must not alter pricing", out.CostUSD, want)
	}
}

// TestAgenticTierRecordsExecutorTokens is the branch killer for the agentic
// tier. It is a SEPARATE arm on purpose: the two tiers reach ReviewerOutcome by
// different code paths, and the agentic one additionally has to carry the
// counts through AgenticRun and the agenticCaller adapter — three places a
// count can be dropped, none of which the text-tier arm touches.
func TestAgenticTierRecordsExecutorTokens(t *testing.T) {
	stub := &stubAgenticRunner{
		run: &AgenticRun{
			Success:      true,
			CostUSD:      0.05,
			Output:       passVerdictJSON,
			InputTokens:  agenticIn,
			OutputTokens: agenticOut,
		},
	}
	out := RunAgenticReviewer("gpt5-6-sol@codex", "doc.md", "body", "", DefaultMaxCostUSD, time.Minute, stub.Run)

	if !out.Present {
		t.Fatalf("expected Present, got absent: %s", out.Err)
	}
	if out.TokensIn != agenticIn || out.TokensOut != agenticOut {
		t.Errorf("tier2 tokens = %d/%d, want %d/%d", out.TokensIn, out.TokensOut, agenticIn, agenticOut)
	}
	// Cost stays OBSERVED, never re-derived from the counts now available.
	if out.CostUSD != 0.05 {
		t.Errorf("observed cost = %f, want 0.05 — the agentic tier must not price from tokens", out.CostUSD)
	}
}

// TestAgenticTierRecordsTokensOnFailedRun pins the audit PARITY between cost and
// tokens. RunAgenticReviewer already records cost "regardless of outcome"; a
// failed-but-billed run that recorded cost and dropped tokens would reproduce
// #708 on the one path where the spend is least explicable.
func TestAgenticTierRecordsTokensOnFailedRun(t *testing.T) {
	stub := &stubAgenticRunner{
		run: &AgenticRun{
			Success:      false,
			Err:          "agent hit an internal error",
			CostUSD:      0.02,
			InputTokens:  failedRunIn,
			OutputTokens: failedRunOut,
		},
	}
	out := RunAgenticReviewer("gpt5-6-sol@codex", "doc.md", "body", "", DefaultMaxCostUSD, time.Minute, stub.Run)

	if out.Present {
		t.Fatal("a failed executor run must be an absence, not a pass")
	}
	if out.CostUSD != 0.02 {
		t.Errorf("cost = %f, want 0.02 (audit record on a failed run)", out.CostUSD)
	}
	if out.TokensIn != failedRunIn || out.TokensOut != failedRunOut {
		t.Errorf("tokens on a failed run = %d/%d, want %d/%d — the audit record must not be narrower than the cost record",
			out.TokensIn, out.TokensOut, failedRunIn, failedRunOut)
	}
}

// TestTokenAccountingGaps_FlagsBilledReviewerWithZeroTokens is the loud-failure
// arm the issue asks for. The healthy reviewer in the SAME result is the
// control: it proves the checker is discriminating rather than flagging every
// reviewer it is handed.
func TestTokenAccountingGaps_FlagsBilledReviewerWithZeroTokens(t *testing.T) {
	q := &QuorumResult{Reviewers: []*ReviewerOutcome{
		{Model: "healthy", CostUSD: 0.03, TokensIn: 100, TokensOut: 20},
		{Model: "billed-blind", CostUSD: 0.07},
	}}
	gaps := q.TokenAccountingGaps()

	if len(gaps) != 1 {
		t.Fatalf("gaps = %d (%v), want exactly 1 — the healthy reviewer is the control", len(gaps), gaps)
	}
	if !strings.Contains(gaps[0], "billed-blind") {
		t.Errorf("gap must NAME the offending reviewer, got %q", gaps[0])
	}
	if !strings.Contains(gaps[0], "0.0700") {
		t.Errorf("gap must quote the unreconcilable spend, got %q", gaps[0])
	}
}

// TestTokenAccountingGaps_CoversTier2Reviewers pins the checker's ENUMERATOR,
// one level below its condition. A checker that walked only q.Reviewers would
// pass every arm above and still be blind by construction to the agentic tier —
// the tier whose cost is OBSERVED rather than derived, and therefore the one
// most likely to bill without reporting usage.
func TestTokenAccountingGaps_CoversTier2Reviewers(t *testing.T) {
	q := &QuorumResult{
		Reviewers: []*ReviewerOutcome{
			{Model: "healthy-tier1", CostUSD: 0.03, TokensIn: 100, TokensOut: 20},
		},
		Tier2: &Tier2Result{Reviewers: []*ReviewerOutcome{
			nil, // a nil outcome must be skipped, not panic
			{Model: "escalated-blind", CostUSD: 0.42, Tier: TierAgentic},
		}},
	}
	gaps := q.TokenAccountingGaps()

	if len(gaps) != 1 {
		t.Fatalf("gaps = %d (%v), want exactly 1 from the tier-2 reviewer", len(gaps), gaps)
	}
	if !strings.Contains(gaps[0], "escalated-blind") || !strings.Contains(gaps[0], "tier2") {
		t.Errorf("gap must name the tier-2 reviewer AND its tier, got %q", gaps[0])
	}
}

// TestTokenAccountingGaps_SilentOnUnbilledReviewer: an absent or free reviewer
// records zero cost AND zero tokens. That is not a gap — flagging it would make
// the notice fire on every degraded quorum and train the reader to ignore it.
func TestTokenAccountingGaps_SilentOnUnbilledReviewer(t *testing.T) {
	q := &QuorumResult{Reviewers: []*ReviewerOutcome{
		{Model: "absent", AbsentReason: ReasonUnreachable},
		{Model: "free-local", CostUSD: 0},
	}}
	if gaps := q.TokenAccountingGaps(); len(gaps) != 0 {
		t.Errorf("gaps = %v, want none — zero cost with zero tokens is a free call, not a gap", gaps)
	}
}

// TestSynthesisTotalsTokens pins the totals at the SAME scope TotalCostUSD has
// always had: tier-1 reviewers only. Quoting a tier-1 cost beside a
// tier-1+tier-2 token total would make the two incomparable.
func TestSynthesisTotalsTokens(t *testing.T) {
	outcomes := []*ReviewerOutcome{
		{Model: "a", CostUSD: 0.01, TokensIn: 100, TokensOut: 10, Present: true, Result: &ReviewResult{Verdict: VerdictPass}},
		{Model: "b", CostUSD: 0.02, TokensIn: 250, TokensOut: 35, Present: true, Result: &ReviewResult{Verdict: VerdictPass}},
	}
	s := synthesize(outcomes, nil)

	if s.TotalTokensIn != 350 || s.TotalTokensOut != 45 {
		t.Errorf("totals = %d/%d, want 350/45", s.TotalTokensIn, s.TotalTokensOut)
	}
	if s.TotalCostUSD == 0 {
		t.Fatal("control: TotalCostUSD is 0 — the synthesis loop did not run, so the token totals prove nothing")
	}
}

// TestAgenticRunFromExecuteResult_CarriesExecutorTokens pins the PRODUCTION
// mapping, which every other arm in this file bypasses: they stub AgenticRunner
// directly, so the adapter from coordinator.ExecuteResult — the only place the
// executor's real counts enter the quorum — was reachable by no test at all. A
// mutation zeroing the token mapping there left the whole package green, which
// is what this arm exists to stop.
func TestAgenticRunFromExecuteResult_CarriesExecutorTokens(t *testing.T) {
	run := agenticRunFromExecuteResult(&coordinator.ExecuteResult{
		Output:       "verdict",
		Success:      true,
		Cost:         0.31,
		InputTokens:  8642,
		OutputTokens: 1357,
	})

	if run.InputTokens != 8642 || run.OutputTokens != 1357 {
		t.Errorf("mapped tokens = %d/%d, want 8642/1357 — the executor's counts are dropped on the production path",
			run.InputTokens, run.OutputTokens)
	}
	// Controls: the fields that already worked must still map, so a failure
	// above is about tokens rather than about the mapping never running.
	if run.CostUSD != 0.31 || run.Output != "verdict" || !run.Success {
		t.Fatalf("control: cost/output/success mis-mapped (%v, %q, %v) — the mapping itself is broken",
			run.CostUSD, run.Output, run.Success)
	}
}

// TestAgenticRunFromExecuteResult_NilIsFailureNotSilentPass keeps the nil guard
// pinned through the extraction: a nil executor result must be a named failure,
// never a zero-value run that reads as a successful empty review.
func TestAgenticRunFromExecuteResult_NilIsFailureNotSilentPass(t *testing.T) {
	run := agenticRunFromExecuteResult(nil)
	if run == nil {
		t.Fatal("mapping must never return nil")
	}
	if run.Success {
		t.Error("a nil executor result must not be Success")
	}
	if !strings.Contains(run.Err, "nil result") {
		t.Errorf("failure must name the cause, got %q", run.Err)
	}
}

// TestWrittenArtifactCarriesTokenKeys asserts on the ARTIFACT, not on the
// struct. The consumer of this work is a controller running `jq` over the
// written JSON to fill the Gate-3 chain ledger, so the load-bearing claim is
// about the emitted keys. Every other arm here would stay green under a field
// rename or a `json:"-"` tag while the artifact remained exactly as tokenless
// as it was before (#708).
func TestWrittenArtifactCarriesTokenKeys(t *testing.T) {
	dir := t.TempDir()
	q := &QuorumResult{
		Doc:          "doc.md",
		ISOTimestamp: "2026-08-18T00:00:00Z",
		Reviewers: []*ReviewerOutcome{
			{Model: "m", CostUSD: 0.04, TokensIn: 3141, TokensOut: 592, Present: true,
				Result: &ReviewResult{Verdict: VerdictPass}},
		},
	}
	q.Synthesis = synthesize(q.Reviewers, nil)

	path, err := WriteJSONArtifact(dir, q)
	if err != nil {
		t.Fatalf("write artifact: %v", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read artifact: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("artifact is not valid JSON: %v", err)
	}
	rev := got["reviewers"].([]any)[0].(map[string]any)
	for key, want := range map[string]float64{"tokens_in": 3141, "tokens_out": 592} {
		v, ok := rev[key]
		if !ok {
			t.Errorf("artifact reviewer has no %q key — the Gate-3 ledger cannot be filled from it", key)
			continue
		}
		if v.(float64) != want {
			t.Errorf("artifact %s = %v, want %v", key, v, want)
		}
	}
	// Control: the key that has always been written must still be there, so a
	// missing token key above is about tokens rather than a broken writer.
	if _, ok := rev["cost_usd"]; !ok {
		t.Fatal("control: cost_usd missing too — the artifact writer itself is broken")
	}
	syn := got["synthesis"].(map[string]any)
	if syn["total_tokens_in"].(float64) != 3141 || syn["total_tokens_out"].(float64) != 592 {
		t.Errorf("synthesis totals not in the artifact: %v / %v", syn["total_tokens_in"], syn["total_tokens_out"])
	}
}
