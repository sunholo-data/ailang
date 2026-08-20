package matrix_test

import (
	"context"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/tools/internal/spike-listrep/matrix"
	"github.com/sunholo-data/ailang/tools/internal/spike-listrep/protocol"
)

func trials(v float64) []protocol.Trial {
	r := make([]protocol.Trial, 5)
	for i := range r {
		r[i] = protocol.Trial{Ordinal: i + 1, Value: v, Valid: true, Command: []string{"measured"}, RawOutput: "printed"}
	}
	return r
}
func fixture() ([]matrix.Cell, []matrix.Cell) {
	ac, bl := matrix.Sets()
	for i := range ac {
		v := 1.0
		if strings.Contains(ac[i].Name, "B1_Branching/arm=C0") && strings.HasSuffix(ac[i].Name, "L=16384") {
			v = 10
		}
		ac[i].Trials = trials(v)
	}
	for i := range bl {
		bl[i].Trials = trials(1)
	}
	return ac, bl
}
func enc() map[string]matrix.Encapsulation {
	return map[string]matrix.Encapsulation{"C1": matrix.Pass, "C2K8": matrix.Pass, "C2K32": matrix.Pass}
}
func find(c []matrix.Cell, s string) *matrix.Cell {
	for i := range c {
		if c[i].Name == s {
			return &c[i]
		}
	}
	panic(s)
}

func TestDistinctSetArithmetic(t *testing.T) { // Kills conflating benchmarkCatalog with AC-1.
	ac, bl := matrix.Sets()
	cat := protocol.BenchmarkCatalog()
	if len(cat) != 76 || len(ac) != 76 || len(bl) != 8 {
		t.Fatalf("catalog/ac1/blen=%d/%d/%d", len(cat), len(ac), len(bl))
	}
	cm := map[string]bool{}
	for _, x := range cat {
		cm[x] = true
	}
	am := map[string]bool{}
	for _, x := range ac {
		am[x.Name] = true
	}
	over := 0
	for x := range am {
		if cm[x] {
			over++
		}
	}
	if over != 68 || len(am)-over != 8 || len(cm)-over != 8 {
		t.Fatalf("overlap=%d differences=%d/%d", over, len(am)-over, len(cm)-over)
	}
}
func TestRefusalInsufficientTrials(t *testing.T) { // Kills silently shrinking a deadline-killed sample.
	a, b := fixture()
	a[0].Trials[0].Valid = false
	r, e := matrix.Verdict(a, b, enc(), false)
	if e == nil || !strings.Contains(r.Refusal, "insufficient_valid_trials") {
		t.Fatalf("%q %v", r.Refusal, e)
	}
}
func TestRefusalControlLeg(t *testing.T) { // Kills emitting a verdict after AC-2's firing control fails.
	a, b := fixture()
	for _, m := range []string{"1024", "4096"} {
		find(a, "BenchmarkListRep_B1_Branching/arm=C0/m="+m+"/L=16384").Trials = trials(7)
	}
	r, e := matrix.Verdict(a, b, enc(), false)
	if e == nil || r.Refusal != "control_leg_failed" || r.Overall != nil {
		t.Fatalf("%+v %v", r, e)
	}
}
func TestRefusalZeroControl(t *testing.T) { // Kills division by zero at an ordinal.
	a, b := fixture()
	find(a, "BenchmarkListRep_B1_Branching/arm=C0/m=1024/L=1024").Trials[2].Value = 0
	r, e := matrix.Verdict(a, b, enc(), false)
	if e == nil || r.Refusal != "zero_control_operand" {
		t.Fatalf("%+v %v", r, e)
	}
}
func TestRefusalUnknownEncapsulation(t *testing.T) { // Kills treating unknown clause (e) as pass.
	a, b := fixture()
	x := enc()
	x["C1"] = matrix.Unknown
	r, e := matrix.Verdict(a, b, x, false)
	if e == nil || r.Refusal != "encapsulation_unknown:C1" {
		t.Fatalf("%+v %v", r, e)
	}
}
func TestRefusalRegexNonSingleton(t *testing.T) { // Kills accepting regexes matching zero/many benchmarks.
	for _, p := range []string{"^none$", "^BenchmarkListRep_B3_Iteration/.*$"} {
		if _, e := protocol.ValidateCell(p); e == nil {
			t.Fatalf("accepted %s", p)
		}
	}
}
func TestRestrictedRefusesVerdict(t *testing.T) { // Kills allowing a partial run to produce GO/STOP.
	a, b := fixture()
	r, e := matrix.Verdict(a[:1], b[:1], enc(), true)
	if e == nil || r.Refusal != "partial_run" || r.Overall != nil {
		t.Fatalf("%+v %v", r, e)
	}
}
func TestGoAndStop(t *testing.T) { // Kills both final-verdict branches and any partial-go third state.
	a, b := fixture()
	r, e := matrix.Verdict(a, b, enc(), false)
	if e != nil || r.Overall.Kind != matrix.Go || r.Overall.Chosen == "" {
		t.Fatalf("%+v %v", r, e)
	}
	for _, arm := range matrix.Candidates {
		find(a, "B6/arm="+arm).Trials = trials(4)
	}
	r, e = matrix.Verdict(a, b, enc(), false)
	if e != nil || r.Overall.Kind != matrix.Stop || r.Overall.Chosen != "" {
		t.Fatalf("%+v %v", r, e)
	}
}
func TestTieBreaksByCThenB(t *testing.T) { // Kills reversing the ratified (c), then (b), tie-break order.
	a, b := fixture()
	find(a, "B6/arm=C1").Trials = trials(2)
	find(a, "B6/arm=C2K8").Trials = trials(1.5)
	find(a, "B6/arm=C2K32").Trials = trials(2.4)
	r, e := matrix.Verdict(a, b, enc(), false)
	if e != nil || r.Overall.Chosen != "C2K8" {
		t.Fatalf("%+v %v", r, e)
	}
	find(a, "B6/arm=C1").Trials = trials(1.5)
	for _, n := range []string{"4096", "65536"} {
		find(a, "BenchmarkListRep_B3_Iteration/arm=C1/n="+n).Trials = trials(1.2)
		find(a, "BenchmarkListRep_B3_Iteration/arm=C2K8/n="+n).Trials = trials(1.4)
	}
	r, e = matrix.Verdict(a, b, enc(), false)
	if e != nil || r.Overall.Chosen != "C1" {
		t.Fatalf("%+v %v", r, e)
	}
}
func TestWithinAndCrossArmOperands(t *testing.T) { // Kills swapping within-arm (a,d) with cross-arm (b,c).
	a, b := fixture()
	for _, n := range []string{"4096", "65536"} {
		find(a, "BenchmarkListRep_B3_Iteration/arm=C0/n="+n).Trials = trials(2)
	}
	find(a, "B6/arm=C0").Trials = trials(2)
	r, e := matrix.Verdict(a, b, enc(), false)
	if e != nil {
		t.Fatal(e)
	}
	c := r.Candidates[0].Clauses
	if !strings.Contains(c["a"].Analyses[0].Candidate[0].Command[0], "measured") || c["a"].Analyses[0].Control[0].Value != 1 {
		t.Fatal("a operands")
	}
	if c["a"].Analyses[0].Median != 1 || c["b"].Analyses[0].Median != .5 || c["c"].Analyses[0].Median != .5 || c["d"].Analyses[0].Median != 1 {
		t.Fatal("operand orientation")
	}
	find(b, "BenchmarkListRep_BLEN_Length/arm=C1/n=4096").Trials = trials(10)
	r, _ = matrix.Verdict(a, b, enc(), false)
	if r.Candidates[0].Clauses["d"].Pass != true {
		t.Fatal("d must divide 65536 by same-arm 4096")
	}
}

// ---------------------------------------------------------------------------
// Tie/spread rerun orchestration at MATRIX level (controller repair, iter 239).
//
// protocol.Paired with allowRerun=true sets Rerun and forces Pass=false. A driver
// that sets that flag and never collects the extra five trials therefore banks a
// straddling clause as a permanent FAIL, and a straddling C0 CONTROL leg aborts the
// whole run with control_leg_failed — a STOP produced by a rerun nobody ran, on the
// gate that commits or cancels ~16 person-days. These four arms pin the repair.
// ---------------------------------------------------------------------------

// straddle returns five trials whose paired ratios against a constant-1 control
// touch BOTH sides of threshold t, which is exactly protocol's rerun condition.
func straddle(t float64) []protocol.Trial {
	vs := []float64{t * 0.5, t * 0.5, t, t * 1.5, t * 1.5}
	r := make([]protocol.Trial, len(vs))
	for i, v := range vs {
		r[i] = protocol.Trial{Ordinal: i + 1, Value: v, Valid: true, Command: []string{"measured"}, RawOutput: "printed"}
	}
	return r
}

// countingCollector supplies the extra trials for a rerun and records how many
// times it was asked, so a test can prove the top-up is cached rather than repeated.
//
// It MUST mirror fixture()'s own value rule (C0 at L=16384 reads 10, everything
// else reads 1). A collector that returns one constant for both arms tops up the
// CONTROL with the candidate's value and collapses the ratio — which reds the arm
// for the fixture rather than for the code. Measured: a constant collector drove
// the C0 control leg's ten-trial median to 2.5 and failed it against >= 8.
// `override` replaces the value for cells the caller is deliberately steering.
func countingCollector(override float64, calls *int) matrix.Collector {
	return func(_ context.Context, in protocol.Invocation, start, count int) []protocol.Trial {
		*calls++
		v := 1.0
		if strings.Contains(in.Selector, "B1_Branching/arm=C0") && strings.Contains(in.Selector, "L=16384") {
			v = override
		} else if strings.Contains(in.Selector, "B3_Iteration") && !strings.Contains(in.Selector, "arm=C0") {
			v = override
		}
		out := make([]protocol.Trial, count)
		for i := range out {
			out[i] = protocol.Trial{Ordinal: start + i, Value: v, Valid: true, Command: []string{"rerun"}, RawOutput: "printed"}
		}
		return out
	}
}

// Kills "analyse passes allowRerun=true and never collects the extra trials"
// on the CONTROL leg — the mutation whose survival aborts the entire verdict.
func TestControlLegStraddleIsRerunNotAborted(t *testing.T) {
	a, b := fixture()
	// Make the C0 control leg at m=1024 straddle its >=8 threshold.
	find(a, "BenchmarkListRep_B1_Branching/arm=C0/m=1024/L=16384").Trials = straddle(8)

	// Without a collector the driver must REFUSE loudly, never silently fail the leg.
	r, e := matrix.Verdict(a, b, enc(), false)
	if e == nil || !strings.HasPrefix(r.Refusal, "rerun_required_but_no_collector") {
		t.Fatalf("no-collector straddle: want a loud rerun refusal, got err=%v refusal=%q", e, r.Refusal)
	}

	// With a collector the rerun is PERFORMED and the median of all ten decides.
	a, b = fixture()
	cell := find(a, "BenchmarkListRep_B1_Branching/arm=C0/m=1024/L=16384")
	cell.Trials = straddle(8)
	calls := 0
	r, e = matrix.VerdictWith(context.Background(), a, b, enc(), false, countingCollector(10, &calls))
	if e != nil {
		t.Fatalf("rerun should have resolved the straddle, got refusal=%q err=%v", r.Refusal, e)
	}
	if calls == 0 {
		t.Fatal("collector was never called: the rerun was not performed")
	}
	if got := len(cell.Trials); got != protocol.InitialTrials+protocol.RerunTrials {
		t.Fatalf("extended cell should hold all %d trials (AC-1 records them), got %d",
			protocol.InitialTrials+protocol.RerunTrials, got)
	}
}

// Kills "the rerun cascades" — the second analysis must run with allowRerun=false,
// so the median of all ten is final and can never demand a third batch.
func TestRerunDoesNotCascade(t *testing.T) {
	a, b := fixture()
	find(a, "BenchmarkListRep_B1_Branching/arm=C0/m=1024/L=16384").Trials = straddle(8)
	calls := 0
	// The top-up value keeps the ten ratios straddling; only allowRerun=false saves us.
	r, e := matrix.VerdictWith(context.Background(), a, b, enc(), false, countingCollector(8, &calls))
	if e != nil {
		t.Fatalf("a cascading rerun leaked into the verdict: refusal=%q err=%v", r.Refusal, e)
	}
	if calls > 2 {
		t.Fatalf("rerun cascaded: collector called %d times for one comparison's two arms", calls)
	}
}

// Kills "extend re-measures a cell that two clauses both extend" — V2 requires a
// cell to be collected exactly once and reused.
func TestRerunTopUpIsCachedPerCell(t *testing.T) {
	a, b := fixture()
	// B3 at both n feeds clause (b) for all three candidates against the SAME C0 cells.
	find(a, "BenchmarkListRep_B3_Iteration/arm=C1/n=4096").Trials = straddle(2)
	find(a, "BenchmarkListRep_B3_Iteration/arm=C2K8/n=4096").Trials = straddle(2)
	calls := 0
	_, _ = matrix.VerdictWith(context.Background(), a, b, enc(), false, countingCollector(1, &calls))
	// Two candidate cells + the ONE shared C0 control cell = 3 top-ups, not 4.
	if calls != 3 {
		t.Fatalf("want 3 cached top-ups (2 candidates + 1 shared control), got %d", calls)
	}
	if got := len(find(a, "BenchmarkListRep_B3_Iteration/arm=C0/n=4096").Trials); got != protocol.InitialTrials+protocol.RerunTrials {
		t.Fatalf("shared control cell should be topped up once to 10, got %d", got)
	}
}

// Kills "firstN is dropped and a 10-trial cell is paired against a 5-trial one" —
// protocol.Paired requires equal-length operand vectors paired BY ORDINAL.
func TestUnextendedClauseStillReadsFiveTrials(t *testing.T) {
	a, b := fixture()
	find(a, "BenchmarkListRep_B3_Iteration/arm=C1/n=4096").Trials = straddle(2)
	calls := 0
	r, e := matrix.VerdictWith(context.Background(), a, b, enc(), false, countingCollector(1, &calls))
	if e != nil {
		t.Fatalf("unexpected refusal=%q err=%v", r.Refusal, e)
	}
	// Clause (a) never straddled, so every one of its analyses must still be over 5.
	for _, cv := range r.Candidates {
		for _, an := range cv.Clauses["a"].Analyses {
			if len(an.Ratios) != protocol.InitialTrials {
				t.Fatalf("%s clause (a): want %d paired ratios, got %d",
					cv.Arm, protocol.InitialTrials, len(an.Ratios))
			}
		}
	}
}
