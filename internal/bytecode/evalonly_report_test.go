package bytecode

import (
	"reflect"
	"testing"
)

// The strict-VM coverage report walks NestedProtos from the roots: an
// evaluator-only prototype is reported only when something the roots can run
// reaches it, a root that is itself evaluator-only is reported, cycles
// terminate, and the result is sorted by name.
func TestEvalOnlyReachable(t *testing.T) {
	img := &BytecodeImage{Prototypes: []*FuncPrototype{
		{Name: "m.main", NestedProtos: []int{1, 2}},                                                // 0
		{Name: "m.zeta", EvalOnly: true, EvalReason: "too wide", File: "m.ail", Line: 9},           // 1
		{Name: "m.mid", NestedProtos: []int{0, 3}},                                                 // 2: cycles back to main
		{Name: "std/x.alpha", EvalOnly: true, EvalReason: "not wired", File: "std/x.ail", Line: 4}, // 3
		{Name: "std/x.unused", EvalOnly: true, EvalReason: "closure arg"},                          // 4: unreachable
		{Name: "m.selfEval", EvalOnly: true, EvalReason: "lower panic", Line: 2},                   // 5
	}}

	got := img.EvalOnlyReachable([]int{0})
	want := []EvalOnlyEntry{
		{Name: "m.zeta", File: "m.ail", Line: 9, Reason: "too wide"},
		{Name: "std/x.alpha", File: "std/x.ail", Line: 4, Reason: "not wired"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("from main:\n got %+v\nwant %+v", got, want)
	}

	if got := img.EvalOnlyReachable([]int{5, 5, -1, 99}); len(got) != 1 || got[0].Name != "m.selfEval" {
		t.Errorf("an evaluator-only root must be reported once; out-of-range roots ignored: %+v", got)
	}
	if got := img.EvalOnlyReachable(nil); len(got) != 0 {
		t.Errorf("no roots, no report: %+v", got)
	}
	if idx := img.ProtoIndex(img.Prototypes[3]); idx != 3 {
		t.Errorf("ProtoIndex = %d, want 3", idx)
	}
	if idx := img.ProtoIndex(&FuncPrototype{Name: "m.main"}); idx != -1 {
		t.Errorf("ProtoIndex of a foreign prototype = %d, want -1", idx)
	}
}
