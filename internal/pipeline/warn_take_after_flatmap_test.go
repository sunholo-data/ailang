package pipeline

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/core"
)

// listGlobal builds a resolved std/list reference.
func listGlobal(name string) *core.VarGlobal {
	return &core.VarGlobal{Ref: core.GlobalRef{Module: "std/list", Name: name}}
}

// progOf wraps expressions as a Core program.
func progOf(exprs ...core.CoreExpr) *core.Program {
	return &core.Program{Decls: exprs}
}

// TestDetectTakeAfterFlatMap_NestedAppArm pins the source-shaped
// App(take, [n, App(flatMap, [f, xs])]) arm of directTakeFlatMap.
//
// This arm is NOT reachable through the normal pipeline: elaboration always
// emits the ANF form (measured V1 iteration 186 — neutering this arm leaves
// both ./internal/diag and ./internal/pipeline green). It is retained as a
// guard should elaboration stop ANF-ing applications, so it is pinned here
// directly against a hand-built Core program rather than left as unexercised
// code that no test would notice rotting.
func TestDetectTakeAfterFlatMap_NestedAppArm(t *testing.T) {
	nested := &core.App{
		Func: listGlobal("take"),
		Args: []core.CoreExpr{
			&core.Var{Name: "n"},
			&core.App{
				Func: listGlobal("flatMap"),
				Args: []core.CoreExpr{&core.Var{Name: "f"}, &core.Var{Name: "xs"}},
			},
		},
	}

	warnings := DetectTakeAfterFlatMap(progOf(nested))
	if len(warnings) != 1 {
		t.Fatalf("nested App form: expected 1 warning, got %d", len(warnings))
	}
	w, ok := warnings[0].(*TakeAfterFlatMapWarning)
	if !ok {
		t.Fatalf("expected *TakeAfterFlatMapWarning, got %T", warnings[0])
	}
	if got := w.Code(); got != "LIST_TAKE_AFTER_FLATMAP" {
		t.Errorf("Code() = %q, want LIST_TAKE_AFTER_FLATMAP", got)
	}
	if s := w.String(); !strings.Contains(s, "takeFlatMap") {
		t.Errorf("String() must carry the fix identifier takeFlatMap, got:\n%s", s)
	}
	// Position() is part of the elaborate.Warning contract's usable surface;
	// exercise it so it cannot silently break.
	if line, col := w.Position(); line < 0 || col < 0 {
		t.Errorf("Position() returned negative coordinates: %d:%d", line, col)
	}
}

// TestDetectTakeAfterFlatMap_NegativeShapes pins the scope rule at the Core
// level: only a DIRECT take-of-flatMap matches. Each of these would be a
// false positive if the match were widened.
func TestDetectTakeAfterFlatMap_NegativeShapes(t *testing.T) {
	cases := []struct {
		name string
		expr core.CoreExpr
	}{
		{
			name: "take_of_map",
			expr: &core.App{
				Func: listGlobal("take"),
				Args: []core.CoreExpr{
					&core.Var{Name: "n"},
					&core.App{Func: listGlobal("map"), Args: []core.CoreExpr{&core.Var{Name: "f"}, &core.Var{Name: "xs"}}},
				},
			},
		},
		{
			name: "flatMap_from_another_module",
			expr: &core.App{
				Func: listGlobal("take"),
				Args: []core.CoreExpr{
					&core.Var{Name: "n"},
					&core.App{
						Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "user/mylist", Name: "flatMap"}},
						Args: []core.CoreExpr{&core.Var{Name: "f"}, &core.Var{Name: "xs"}},
					},
				},
			},
		},
		{
			name: "local_take_not_a_global",
			expr: &core.App{
				Func: &core.Var{Name: "take"},
				Args: []core.CoreExpr{
					&core.Var{Name: "n"},
					&core.App{Func: listGlobal("flatMap"), Args: []core.CoreExpr{&core.Var{Name: "f"}, &core.Var{Name: "xs"}}},
				},
			},
		},
		{
			name: "anf_let_bound_to_something_else",
			expr: &core.Let{
				Name:  "tmp",
				Value: &core.App{Func: listGlobal("map"), Args: []core.CoreExpr{&core.Var{Name: "f"}, &core.Var{Name: "xs"}}},
				Body: &core.App{
					Func: listGlobal("take"),
					Args: []core.CoreExpr{&core.Var{Name: "n"}, &core.Var{Name: "tmp"}},
				},
			},
		},
		{
			name: "anf_take_reads_a_different_binding",
			expr: &core.Let{
				Name:  "tmp",
				Value: &core.App{Func: listGlobal("flatMap"), Args: []core.CoreExpr{&core.Var{Name: "f"}, &core.Var{Name: "xs"}}},
				Body: &core.App{
					Func: listGlobal("take"),
					Args: []core.CoreExpr{&core.Var{Name: "n"}, &core.Var{Name: "unrelated"}},
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := DetectTakeAfterFlatMap(progOf(tc.expr)); len(got) != 0 {
				t.Errorf("expected no warning for %s, got %d: %v", tc.name, len(got), got)
			}
		})
	}
}

// TestDetectTakeAfterFlatMap_NilProgram pins the nil guard.
func TestDetectTakeAfterFlatMap_NilProgram(t *testing.T) {
	if got := DetectTakeAfterFlatMap(nil); got != nil {
		t.Errorf("nil program must yield nil warnings, got %v", got)
	}
}
