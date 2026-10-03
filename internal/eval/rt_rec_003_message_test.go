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

// TestRTREC003AdvisesTailCalls: since M-EVAL-TAIL-CALLS (#1486) a tail call runs
// in constant depth, so the message recommends one, and following the advice
// must actually work under the same limit that just failed. (Before #1486 this
// test banned the phrase, because the evaluator had no tail-call elimination.)
func TestRTREC003AdvisesTailCalls(t *testing.T) {
	_, err := newSumEvaluator(100).evalCore(sumProgram(300))
	if err == nil {
		t.Fatal("instrument failure: expected the recursion guard to fire")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "tail call") {
		t.Fatalf("RT_REC_003 should recommend a tail call: %q", err.Error())
	}

	// The advised rewrite: sumAcc(i, acc) = if i == 0 then acc else sumAcc(i - 1, acc + i)
	body := &core.If{
		Cond: &core.BinOp{Op: "==", Left: &core.Var{Name: "i"}, Right: &core.Lit{Kind: core.IntLit, Value: 0}},
		Then: &core.Var{Name: "acc"},
		Else: &core.App{Func: &core.Var{Name: "sumAcc"}, Args: []core.CoreExpr{
			&core.BinOp{Op: "-", Left: &core.Var{Name: "i"}, Right: &core.Lit{Kind: core.IntLit, Value: 1}},
			&core.BinOp{Op: "+", Left: &core.Var{Name: "acc"}, Right: &core.Var{Name: "i"}},
		}},
	}
	prog := &core.LetRec{
		Bindings: []core.RecBinding{{Name: "sumAcc", Value: &core.Lambda{Params: []string{"i", "acc"}, Body: body}}},
		Body: &core.App{Func: &core.Var{Name: "sumAcc"}, Args: []core.CoreExpr{
			&core.Lit{Kind: core.IntLit, Value: 300}, &core.Lit{Kind: core.IntLit, Value: 0},
		}},
	}
	res, err := newSumEvaluator(100).evalCore(prog)
	if err != nil {
		t.Fatalf("the advised tail-call rewrite still failed under the same limit: %v", err)
	}
	if iv, ok := res.(*IntValue); !ok || iv.Value != 300*301/2 {
		t.Fatalf("sumAcc(300) = %v, want %d", res, 300*301/2)
	}
}

// TestRTREC003WarnsListAccumulatorsAreQuadratic (#1501): the message once told users
// to carry partial results in an accumulator and use foldl, which for a LIST
// accumulator is the O(n^2) consing loop. It must point list builders at the
// one-pass helpers instead.
func TestRTREC003WarnsListAccumulatorsAreQuadratic(t *testing.T) {
	msg := (&RecursionLimitError{Limit: 10}).Error()
	for _, want := range []string{"mapAccumL", "O(n^2)"} {
		if !strings.Contains(msg, want) {
			t.Errorf("RT_REC_003 should mention %q: %q", want, msg)
		}
	}
}
