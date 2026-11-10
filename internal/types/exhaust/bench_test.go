package exhaust

import (
	"strconv"
	"testing"
)

// Performance budgets of design §7, as benchmarks. Wall time is measured by
// the benchmarks (go test -bench . ./internal/types/exhaust/); the tests
// below pin the deterministic part — the verdict and the step count — so a
// complexity regression fails `go test` without a flaky timer.

// stringLiteralArms is a 1,000-arm string-literal match ending in `_`
// (the census maximum is 97 arms).
func stringLiteralArms(n int) []Arm {
	out := make([]Arm, 0, n+1)
	for i := 0; i < n; i++ {
		out = append(out, Arm{Pattern: Lit{Value: "s" + strconv.Itoa(i)}})
	}
	return append(out, Arm{Pattern: Wild{}})
}

// boolTupleArms is a 12-column bool-tuple match with 2^10 rows: every
// combination of the first ten columns, the last two wildcards.
func boolTupleArms() []Arm {
	out := make([]Arm, 0, 1<<10)
	for m := 0; m < 1<<10; m++ {
		elems := make([]Pattern, 12)
		for i := 0; i < 12; i++ {
			if i < 10 {
				elems[i] = Lit{Value: m&(1<<i) != 0}
			} else {
				elems[i] = Wild{}
			}
		}
		out = append(out, Arm{Pattern: Tuple{Elems: elems}})
	}
	return out
}

func TestBudgetCases(t *testing.T) {
	cases := []struct {
		name     string
		arms     []Arm
		maxSteps int
	}{
		{"1000 string literals", stringLiteralArms(1000), 10_000},
		{"12-column bool tuple, 2^10 rows", boolTupleArms(), 100_000},
	}
	for _, tc := range cases {
		res, err := Check(tc.arms, nil, Options{})
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if !res.Exhaustive || len(res.Redundant) != 0 {
			t.Errorf("%s: exhaustive=%v redundant=%v, want exhaustive, none redundant", tc.name, res.Exhaustive, res.Redundant)
		}
		if res.Steps > tc.maxSteps {
			t.Errorf("%s: %d steps, want ≤ %d", tc.name, res.Steps, tc.maxSteps)
		}
		t.Logf("%s: %d steps", tc.name, res.Steps)
	}
}

func BenchmarkStringLiterals1000(b *testing.B) {
	arms := stringLiteralArms(1000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := Check(arms, nil, Options{}); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkBoolTuple12x1024(b *testing.B) {
	arms := boolTupleArms()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := Check(arms, nil, Options{}); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkTypicalMatch is the census median-to-p99 shape: a handful of
// constructor arms with one nested level.
func BenchmarkTypicalMatch(b *testing.B) {
	arms := arms(some(ok(w)), some(er(l("x"))), some(er(w)), none())
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := Check(arms, testUniverse, Options{}); err != nil {
			b.Fatal(err)
		}
	}
}
