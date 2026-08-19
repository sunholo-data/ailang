package eval

import (
	"regexp"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/core"
)

// RT_REC_003 is a string the product hands a human to act on, so it is tested the
// way rule 3k requires: the remedy is taken OUT of the evaluator's own output and
// EXECUTED, rather than reconstructed from what the message ought to say. A test
// that rebuilt the advice independently would pass on advice that cannot be
// followed — which is exactly how the message spent its life telling users to
// "enable tail recursion", an option that has never existed in this evaluator
// (tail-call machinery lives in internal/vm and internal/bytecode, which
// `ailang run` does not use).

// recursionRemedies maps a CLI flag the RT_REC_003 message may advertise onto the
// evaluator knob that flag drives. A flag with no entry here is advice the runtime
// cannot honour, and the test fails loudly rather than accepting it.
var recursionRemedies = map[string]func(*CoreEvaluator, int){
	"--max-recursion-depth": (*CoreEvaluator).SetMaxRecursionDepth,
}

// sumProgram builds `letrec sum = λn. if n <= 0 then 0 else n + sum(n-1) in sum(n)`,
// whose depth is proportional to n, so it overflows at a low ceiling and succeeds at
// a raised one. That difference is what makes the remedy checkable.
func sumProgram(n int) *core.LetRec {
	body := &core.If{
		Cond: &core.BinOp{
			Op:    "<=",
			Left:  &core.Var{Name: "n"},
			Right: &core.Lit{Kind: core.IntLit, Value: 0},
		},
		Then: &core.Lit{Kind: core.IntLit, Value: 0},
		Else: &core.BinOp{
			Op:   "+",
			Left: &core.Var{Name: "n"},
			Right: &core.App{
				Func: &core.Var{Name: "sum"},
				Args: []core.CoreExpr{
					&core.BinOp{
						Op:    "-",
						Left:  &core.Var{Name: "n"},
						Right: &core.Lit{Kind: core.IntLit, Value: 1},
					},
				},
			},
		},
	}
	return &core.LetRec{
		Bindings: []core.RecBinding{
			{Name: "sum", Value: &core.Lambda{Params: []string{"n"}, Body: body}},
		},
		Body: &core.App{
			Func: &core.Var{Name: "sum"},
			Args: []core.CoreExpr{&core.Lit{Kind: core.IntLit, Value: n}},
		},
	}
}

func newSumEvaluator(depth int) *CoreEvaluator {
	ev := NewCoreEvaluator()
	ev.SetExperimentalBinopShim(true)
	ev.SetMaxRecursionDepth(depth)
	return ev
}

// TestRTREC003AdvertisesOnlyRemediesThatExist reads every flag RT_REC_003 names out of
// the emitted error and requires each one to be a knob this evaluator actually has —
// then applies it and requires the previously-failing program to succeed.
func TestRTREC003AdvertisesOnlyRemediesThatExist(t *testing.T) {
	const n = 300

	_, err := newSumEvaluator(100).evalCore(sumProgram(n))
	if err == nil {
		t.Fatal("instrument failure: sum(300) at depth 100 did not exceed the recursion guard, so no message was produced")
	}
	msg := err.Error()
	if !strings.Contains(msg, "RT_REC_003") {
		t.Fatalf("instrument failure: expected an RT_REC_003 error, got: %v", err)
	}

	flags := regexp.MustCompile(`--[a-z][a-z0-9-]*`).FindAllString(msg, -1)
	// Anti-vacuity floor: a message naming no flag at all would satisfy every
	// assertion below by having nothing to check.
	if len(flags) == 0 {
		t.Fatalf("instrument failure: RT_REC_003 names no actionable flag at all: %q", msg)
	}

	for _, f := range flags {
		apply, ok := recursionRemedies[f]
		if !ok {
			t.Fatalf("RT_REC_003 advertises %q, which this evaluator has no knob for; message: %q", f, msg)
		}

		ev := NewCoreEvaluator()
		ev.SetExperimentalBinopShim(true)
		apply(ev, 10000)

		result, rerr := ev.evalCore(sumProgram(n))
		if rerr != nil {
			t.Fatalf("following the advertised remedy %q did not resolve the failure: %v", f, rerr)
		}
		intVal, ok := result.(*IntValue)
		if !ok {
			t.Fatalf("after remedy %q: expected IntValue, got %T", f, result)
		}
		if want := n * (n + 1) / 2; intVal.Value != want {
			t.Errorf("after remedy %q: expected sum(%d) = %d, got %d", f, n, want, intVal.Value)
		}
	}
}

// TestRTREC003DoesNotAdvertiseTailRecursion pins the specific defect: the evaluator has
// no tail-call elimination, so naming it sends the reader hunting for a flag that was
// never built. Kept separate from the flag check above because "enable tail recursion"
// is prose, not a flag, and so is invisible to a flag-shaped matcher.
func TestRTREC003DoesNotAdvertiseTailRecursion(t *testing.T) {
	_, err := newSumEvaluator(100).evalCore(sumProgram(300))
	if err == nil {
		t.Fatal("instrument failure: expected the recursion guard to fire")
	}
	lower := strings.ToLower(err.Error())
	if !strings.Contains(lower, "rt_rec_003") {
		t.Fatalf("instrument failure: expected an RT_REC_003 error, got: %v", err)
	}
	for _, banned := range []string{"tail recursion", "tail call", "tail-call"} {
		if strings.Contains(lower, banned) {
			t.Errorf("RT_REC_003 advertises %q, but this evaluator has no tail-call elimination: %q", banned, err.Error())
		}
	}
}
