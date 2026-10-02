package lower

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/types"
)

// #1545: a function whose type declares a dynamic frame mode the VM cannot
// push (Rand[mode=seeded|crypto], @limit/@min budgets, Net[scope=public]) must
// lower as an EvalOnly stub so the call runs on the evaluator, which pushes
// the frame. Lowering it normally silently drops the mode: crypto draws come
// from math/rand and budgets are not enforced.

func intFn(row *types.Row) *types.TFunc2 {
	return &types.TFunc2{Params: []types.Type{types.TInt}, Return: types.TInt, EffectRow: row}
}

func randRow(mode string) *types.Row {
	row := &types.Row{Kind: types.EffectRow, Labels: map[string]types.Type{"Rand": types.Unit()}}
	if mode != "" {
		row.Params = map[string]map[string]string{"Rand": {"mode": mode}}
	}
	return row
}

// singleFnProgram is `export func f(x) = x` with lambda type fnType.
func singleFnProgram(fnType types.Type) (*core.Program, types.CoreTypeInfo) {
	cti := makeCTI(map[uint64]types.Type{100: fnType, 101: types.TInt})
	prog := &core.Program{
		Decls: []core.CoreExpr{&core.Let{
			CoreNode: core.CoreNode{NodeID: 1},
			Name:     "f",
			Value: &core.Lambda{
				CoreNode: core.CoreNode{NodeID: 100},
				Params:   []string{"x"},
				Body:     coreVar(101, "x"),
			},
			Body: &core.Lit{CoreNode: core.CoreNode{NodeID: 2}, Kind: core.UnitLit},
		}},
		Meta: map[string]*core.DeclMeta{"f": {Name: "f", IsExport: true}},
	}
	return prog, cti
}

func lowerOne(t *testing.T, prog *core.Program, cti types.CoreTypeInfo) string {
	t.Helper()
	out, err := LowerProgram(prog, cti, nil, "main")
	if err != nil {
		t.Fatal(err)
	}
	if len(out.FuncDecls) != 1 {
		t.Fatalf("want 1 func, got %d", len(out.FuncDecls))
	}
	if len(out.FuncDecls[0].Params) != 1 {
		t.Fatalf("stub/function must keep arity 1, got %d", len(out.FuncDecls[0].Params))
	}
	return out.FuncDecls[0].LowerError
}

func TestFrameMode_RandModesLowerEvalOnly(t *testing.T) {
	for _, mode := range []string{"seeded", "crypto"} {
		prog, cti := singleFnProgram(intFn(randRow(mode)))
		reason := lowerOne(t, prog, cti)
		if !strings.Contains(reason, "Rand[mode="+mode+"]") {
			t.Errorf("Rand[mode=%s] function must lower EvalOnly naming the mode, LowerError=%q", mode, reason)
		}
	}
}

func TestFrameMode_OsRandLowersNormally(t *testing.T) {
	for _, mode := range []string{"", "os"} {
		prog, cti := singleFnProgram(intFn(randRow(mode)))
		if reason := lowerOne(t, prog, cti); reason != "" {
			t.Errorf("Rand (mode %q) must lower normally, got LowerError=%q", mode, reason)
		}
	}
}

func TestFrameMode_BudgetsLowerEvalOnly(t *testing.T) {
	two, one := 2, 1
	cases := map[string]*types.Row{
		"IO @limit=2": {Kind: types.EffectRow, Labels: map[string]types.Type{"IO": types.Unit()}, Budgets: map[string]*int{"IO": &two}},
		"IO @min=1":   {Kind: types.EffectRow, Labels: map[string]types.Type{"IO": types.Unit()}, MinBudgets: map[string]*int{"IO": &one}},
	}
	for want, row := range cases {
		prog, cti := singleFnProgram(intFn(row))
		if reason := lowerOne(t, prog, cti); !strings.Contains(reason, want) {
			t.Errorf("%s function must lower EvalOnly naming the budget, LowerError=%q", want, reason)
		}
	}
}

// A nested lambda with a declared mode would become a VM closure (closures
// cannot cross the bridge), so the enclosing top-level function goes to the
// evaluator.
func TestFrameMode_NestedModedLambdaStubsEnclosing(t *testing.T) {
	inner := &types.TFunc2{Params: []types.Type{types.TInt}, Return: types.TInt, EffectRow: randRow("crypto")}
	outer := &types.TFunc2{Params: []types.Type{types.TInt}, Return: inner}
	cti := makeCTI(map[uint64]types.Type{100: outer, 200: inner, 201: types.TInt})
	prog := &core.Program{
		Decls: []core.CoreExpr{&core.Let{
			CoreNode: core.CoreNode{NodeID: 1},
			Name:     "mk",
			Value: &core.Lambda{
				CoreNode: core.CoreNode{NodeID: 100},
				Params:   []string{"x"},
				Body: &core.Lambda{
					CoreNode: core.CoreNode{NodeID: 200},
					Params:   []string{"y"},
					Body:     coreVar(201, "y"),
				},
			},
			Body: &core.Lit{CoreNode: core.CoreNode{NodeID: 2}, Kind: core.UnitLit},
		}},
		Meta: map[string]*core.DeclMeta{"mk": {Name: "mk", IsExport: true}},
	}
	if reason := lowerOne(t, prog, cti); !strings.Contains(reason, "Rand[mode=crypto]") {
		t.Fatalf("a function holding a Rand[mode=crypto] lambda must lower EvalOnly, LowerError=%q", reason)
	}
}
