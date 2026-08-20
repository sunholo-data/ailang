package protocol

import (
	"context"
	"strings"
	"testing"
)

func valid(v ...float64) []Trial {
	r := make([]Trial, len(v))
	for i, x := range v {
		r[i] = Trial{Ordinal: i + 1, Value: x, Valid: true}
	}
	return r
}
func TestAdjudicatorSyntheticVectors(t *testing.T) {
	o := valid(1, 1, 1, 1, 1)
	for _, x := range []struct {
		name        string
		c           []Trial
		t           float64
		rerun, pass bool
		med         float64
	}{{"odd", valid(5, 1, 4, 2, 3), 10, false, true, 3}, {"tie", valid(1, 2, 3, 4, 5), 3, true, false, 3}, {"spread", valid(1, 2, 2, 4, 5), 3, true, false, 2}} {
		t.Run(x.name, func(t *testing.T) {
			g := Paired(x.c, o, x.t, "<=", true)
			if g.Rerun != x.rerun || g.Pass != x.pass || g.Median != x.med {
				t.Fatalf("%+v", g)
			}
		})
	}
}
func TestInvalidTrialDoesNotShrinkSample(t *testing.T) {
	c := valid(1, 2, 3, 4, 5)
	c[2] = Trial{Ordinal: 3, Error: "deadline"}
	g := Paired(c, valid(1, 1, 1, 1, 1), 3, "<=", true)
	if g.Valid || len(g.Candidate) != 5 || !strings.Contains(g.Error, "invalid trial") {
		t.Fatalf("%+v", g)
	}
}
func TestValidateCellRejectsNonSingletonRegex(t *testing.T) {
	for _, p := range []string{"BenchmarkListRep_B3", "^BenchmarkListRep_B3_Iteration/.*$", "^does-not-exist$"} {
		if _, e := ValidateCell(p); e == nil {
			t.Errorf("accepted %q", p)
		}
	}
	if _, e := ValidateCell("^BenchmarkListRep_B3_Iteration/arm=C0/n=4096$"); e != nil {
		t.Fatal(e)
	}
}
func TestAdjudicateRerunProtocol(t *testing.T) {
	calls := 0
	g := func(_ context.Context, in Invocation, start, count int) []Trial {
		calls++
		r := make([]Trial, count)
		for i := range count {
			v := 1.0
			if start == 1 && i == 0 && in.Selector == "c" {
				v = 2
			}
			r[i] = Trial{Ordinal: start + i, Value: v, Valid: true}
		}
		return r
	}
	a := Adjudicate(context.Background(), g, Invocation{Selector: "c"}, Invocation{Selector: "d"}, 1, "<=")
	if calls != 4 || len(a.Candidate) != 10 || a.Rerun {
		t.Fatalf("calls=%d %+v", calls, a)
	}
}

// ParseMetric on a gcshape document must tolerate NON-NUMERIC sibling fields.
// The gcshape report carries a string `arm` beside its numeric counters, and the
// original map[string]float64 unmarshal failed on the whole document, so every B8
// trial came back invalid with `cannot unmarshal string into Go value of type
// float64`. No kill clause reads B8, so the only thing that ever asked for it was
// AC-1's completeness floor — which is exactly where it surfaced, ten minutes into
// a full matrix pass. Kills reverting to map[string]float64.
func TestParseMetricGCShapeToleratesStringFields(t *testing.T) {
	doc := []byte(`{"arm":"C0","n":12800,"num_gc_delta":414,"pause_total_ns_delta":26438517}`)
	in := Invocation{Kind: "gcshape", Selector: "C0", Metric: "num_gc_delta"}
	v, _, err := parseMetric(in, doc)
	if err != nil {
		t.Fatalf("gcshape parse failed on a document with a string field: %v", err)
	}
	if v != 414 {
		t.Fatalf("num_gc_delta: want 414, got %v", v)
	}
	// An ABSENT metric must still fail loudly — tolerating strings must not
	// degrade into tolerating a missing number.
	if _, _, err := parseMetric(Invocation{Kind: "gcshape", Selector: "C0", Metric: "nope"}, doc); err == nil {
		t.Fatal("an absent metric must be an error, not a silent zero")
	}
	// A metric whose value is non-numeric must fail too, not read as 0.
	if _, _, err := parseMetric(Invocation{Kind: "gcshape", Selector: "C0", Metric: "arm"}, doc); err == nil {
		t.Fatal("a non-numeric metric must be an error, not a silent zero")
	}
}

// Adjudicate must collect EXACTLY five trials per arm when no rerun is warranted.
// The M3-M5 -> M6 consolidation moved cmd/runner's tests into this package and
// dropped the arm that pinned this, so a mutant making Adjudicate always collect a
// rerun batch survived the ENTIRE suite (verified: landed by sha256, built rc=0,
// go test ./tools/internal/spike-listrep/... rc=0). That is the same unpinned-rerun
// class the matrix driver was just repaired for, in the opposite direction, on the
// still-live single-cell CLI path. Kills "always rerun" and "rerun batch sized wrong".
func TestAdjudicateCollectsFiveTrialsWhenNoRerunIsWarranted(t *testing.T) {
	type call struct{ start, count int }
	var calls []call
	gather := func(_ context.Context, _ Invocation, start, count int) []Trial {
		calls = append(calls, call{start, count})
		out := make([]Trial, count)
		for i := range out {
			// Constant 1.0 against a constant control: every paired ratio is
			// exactly 1.0, nowhere near the threshold, so no rerun is warranted.
			out[i] = Trial{Ordinal: start + i, Value: 1, Valid: true, Command: []string{"m"}, RawOutput: "r"}
		}
		return out
	}
	r := Adjudicate(context.Background(), gather, Invocation{}, Invocation{}, 2.0, "<=")
	if !r.Valid || !r.Pass {
		t.Fatalf("a flat 1.0 ratio against a <= 2.0 threshold must pass: valid=%v pass=%v err=%q", r.Valid, r.Pass, r.Error)
	}
	if r.Rerun {
		t.Error("no rerun should have been warranted")
	}
	if len(calls) != 2 {
		t.Fatalf("want exactly 2 collector calls (one per arm, no rerun batch), got %d: %+v", len(calls), calls)
	}
	for i, c := range calls {
		if c.start != 1 || c.count != InitialTrials {
			t.Errorf("call %d: want start=1 count=%d, got start=%d count=%d", i, InitialTrials, c.start, c.count)
		}
	}
	if len(r.Candidate) != InitialTrials || len(r.Control) != InitialTrials {
		t.Fatalf("want %d trials per arm, got cand=%d ctrl=%d", InitialTrials, len(r.Candidate), len(r.Control))
	}
}
