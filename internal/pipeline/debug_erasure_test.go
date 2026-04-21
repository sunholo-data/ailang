package pipeline

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/types"
)

func TestEraseDebugFromEffectRow_NilStaysNil(t *testing.T) {
	result := EraseDebugFromEffectRow(nil)
	if result != nil {
		t.Fatalf("expected nil for nil input, got %v", result)
	}
}

func TestEraseDebugFromEffectRow_NoDebug(t *testing.T) {
	row := &types.Row{
		Kind:   types.EffectRow,
		Labels: map[string]types.Type{"IO": types.Unit()},
	}
	result := EraseDebugFromEffectRow(row)
	if result != row {
		t.Fatalf("expected same row when no Debug present")
	}
}

func TestEraseDebugFromEffectRow_DebugOnly(t *testing.T) {
	row := &types.Row{
		Kind:   types.EffectRow,
		Labels: map[string]types.Type{"Debug": types.Unit()},
	}
	result := EraseDebugFromEffectRow(row)
	if result != nil {
		t.Fatalf("expected nil (pure) when only Debug, got %v", result)
	}
}

func TestEraseDebugFromEffectRow_MixedEffects(t *testing.T) {
	row := &types.Row{
		Kind: types.EffectRow,
		Labels: map[string]types.Type{
			"IO":    types.Unit(),
			"Debug": types.Unit(),
			"Net":   types.Unit(),
		},
	}
	result := EraseDebugFromEffectRow(row)
	if result == nil {
		t.Fatal("expected non-nil row with IO and Net remaining")
	}
	if _, ok := result.Labels["Debug"]; ok {
		t.Fatal("Debug should be removed from labels")
	}
	if _, ok := result.Labels["IO"]; !ok {
		t.Fatal("IO should remain in labels")
	}
	if _, ok := result.Labels["Net"]; !ok {
		t.Fatal("Net should remain in labels")
	}
	if len(result.Labels) != 2 {
		t.Fatalf("expected 2 labels, got %d", len(result.Labels))
	}
}

func TestEraseDebugFromEffectRow_WithRowTail(t *testing.T) {
	row := &types.Row{
		Kind:   types.EffectRow,
		Labels: map[string]types.Type{"Debug": types.Unit()},
		Tail:   &types.RowVar{Name: "e", Kind: types.EffectRow},
	}
	result := EraseDebugFromEffectRow(row)
	// Debug removed but tail remains — not nil
	if result == nil {
		t.Fatal("expected non-nil row because tail variable exists")
	}
	if _, ok := result.Labels["Debug"]; ok {
		t.Fatal("Debug should be removed")
	}
	if result.Tail == nil || result.Tail.Name != "e" {
		t.Fatal("tail should be preserved")
	}
}

func TestEraseDebugFromEffectRow_WithBudgets(t *testing.T) {
	debugBudget := 10
	ioBudget := 5
	row := &types.Row{
		Kind: types.EffectRow,
		Labels: map[string]types.Type{
			"IO":    types.Unit(),
			"Debug": types.Unit(),
		},
		Budgets: map[string]*int{
			"IO":    &ioBudget,
			"Debug": &debugBudget,
		},
	}
	result := EraseDebugFromEffectRow(row)
	if result == nil {
		t.Fatal("expected non-nil row with IO remaining")
	}
	if _, ok := result.Budgets["Debug"]; ok {
		t.Fatal("Debug budget should be removed")
	}
	if result.Budgets["IO"] == nil || *result.Budgets["IO"] != 5 {
		t.Fatal("IO budget should be preserved")
	}
}

// --- Expression erasure tests ---

func TestEraseExpr_DebugLogCallBecomesUnit(t *testing.T) {
	eraser := &DebugEraser{}
	app := &core.App{
		Func: &core.Var{Name: "_debug_log"},
		Args: []core.CoreExpr{
			&core.Lit{Kind: core.StringLit, Value: "hello"},
			&core.Lit{Kind: core.StringLit, Value: "test.ail:1"},
		},
	}
	result := eraser.eraseExpr(app)
	lit, ok := result.(*core.Lit)
	if !ok {
		t.Fatalf("expected Lit, got %T", result)
	}
	if lit.Kind != core.UnitLit {
		t.Fatalf("expected UnitLit, got %v", lit.Kind)
	}
}

func TestEraseExpr_DebugCheckCallBecomesUnit(t *testing.T) {
	eraser := &DebugEraser{}
	app := &core.App{
		Func: &core.Var{Name: "_debug_check"},
		Args: []core.CoreExpr{
			&core.Lit{Kind: core.BoolLit, Value: true},
			&core.Lit{Kind: core.StringLit, Value: "check msg"},
			&core.Lit{Kind: core.StringLit, Value: "test.ail:2"},
		},
	}
	result := eraser.eraseExpr(app)
	lit, ok := result.(*core.Lit)
	if !ok {
		t.Fatalf("expected Lit, got %T", result)
	}
	if lit.Kind != core.UnitLit {
		t.Fatalf("expected UnitLit, got %v", lit.Kind)
	}
}

func TestEraseExpr_VarGlobalDebugCallBecomesUnit(t *testing.T) {
	eraser := &DebugEraser{}
	app := &core.App{
		Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "std/debug", Name: "_debug_log"}},
		Args: []core.CoreExpr{
			&core.Lit{Kind: core.StringLit, Value: "hello"},
			&core.Lit{Kind: core.StringLit, Value: "test.ail:1"},
		},
	}
	result := eraser.eraseExpr(app)
	lit, ok := result.(*core.Lit)
	if !ok {
		t.Fatalf("expected Lit, got %T", result)
	}
	if lit.Kind != core.UnitLit {
		t.Fatalf("expected UnitLit, got %v", lit.Kind)
	}
}

func TestEraseExpr_NonDebugCallPreserved(t *testing.T) {
	eraser := &DebugEraser{}
	app := &core.App{
		Func: &core.Var{Name: "println"},
		Args: []core.CoreExpr{
			&core.Lit{Kind: core.StringLit, Value: "hello"},
		},
	}
	result := eraser.eraseExpr(app)
	_, ok := result.(*core.App)
	if !ok {
		t.Fatalf("expected App preserved, got %T", result)
	}
}

func TestEraseExpr_NestedDebugInLet(t *testing.T) {
	eraser := &DebugEraser{}
	// let _ = _debug_log("msg", "loc") in 42
	expr := &core.Let{
		Name: "_",
		Value: &core.App{
			Func: &core.Var{Name: "_debug_log"},
			Args: []core.CoreExpr{
				&core.Lit{Kind: core.StringLit, Value: "msg"},
				&core.Lit{Kind: core.StringLit, Value: "loc"},
			},
		},
		Body: &core.Lit{Kind: core.IntLit, Value: 42},
	}
	result := eraser.eraseExpr(expr)
	letExpr, ok := result.(*core.Let)
	if !ok {
		t.Fatalf("expected Let, got %T", result)
	}
	// Value should be unit now
	lit, ok := letExpr.Value.(*core.Lit)
	if !ok {
		t.Fatalf("expected Let value to be Lit, got %T", letExpr.Value)
	}
	if lit.Kind != core.UnitLit {
		t.Fatal("expected Let value to be UnitLit after debug erasure")
	}
	// Body should be preserved
	bodyLit, ok := letExpr.Body.(*core.Lit)
	if !ok || bodyLit.Kind != core.IntLit {
		t.Fatal("expected Let body to be preserved as IntLit")
	}
}

func TestEraseExpr_DebugInMatchArm(t *testing.T) {
	eraser := &DebugEraser{}
	expr := &core.Match{
		Scrutinee: &core.Var{Name: "x"},
		Arms: []core.MatchArm{
			{
				Pattern: &core.VarPattern{Name: "y"},
				Body: &core.App{
					Func: &core.Var{Name: "_debug_log"},
					Args: []core.CoreExpr{
						&core.Lit{Kind: core.StringLit, Value: "matched"},
						&core.Lit{Kind: core.StringLit, Value: "loc"},
					},
				},
			},
		},
	}
	result := eraser.eraseExpr(expr)
	matchExpr, ok := result.(*core.Match)
	if !ok {
		t.Fatalf("expected Match, got %T", result)
	}
	lit, ok := matchExpr.Arms[0].Body.(*core.Lit)
	if !ok || lit.Kind != core.UnitLit {
		t.Fatal("expected match arm body to be UnitLit after debug erasure")
	}
}

func TestEraseExpr_NilReturnsNil(t *testing.T) {
	eraser := &DebugEraser{}
	result := eraser.eraseExpr(nil)
	if result != nil {
		t.Fatal("expected nil for nil input")
	}
}

func TestErase_FullProgram(t *testing.T) {
	eraser := &DebugEraser{}
	prog := &core.Program{
		Decls: []core.CoreExpr{
			// A function that calls debug
			&core.Lambda{
				Params: []string{"x"},
				Body: &core.Let{
					Name: "_",
					Value: &core.App{
						Func: &core.Var{Name: "_debug_log"},
						Args: []core.CoreExpr{
							&core.Lit{Kind: core.StringLit, Value: "log"},
							&core.Lit{Kind: core.StringLit, Value: "loc"},
						},
					},
					Body: &core.Var{Name: "x"},
				},
			},
			// A non-debug function
			&core.Lambda{
				Params: []string{"y"},
				Body:   &core.Var{Name: "y"},
			},
		},
	}

	result := eraser.Erase(prog)
	if len(result.Decls) != 2 {
		t.Fatalf("expected 2 decls, got %d", len(result.Decls))
	}

	// First decl: debug call should be erased
	lam0, ok := result.Decls[0].(*core.Lambda)
	if !ok {
		t.Fatalf("expected Lambda, got %T", result.Decls[0])
	}
	letExpr, ok := lam0.Body.(*core.Let)
	if !ok {
		t.Fatalf("expected Let, got %T", lam0.Body)
	}
	lit, ok := letExpr.Value.(*core.Lit)
	if !ok || lit.Kind != core.UnitLit {
		t.Fatal("expected debug call in first decl to be erased to unit")
	}

	// Second decl: should be unchanged
	lam1, ok := result.Decls[1].(*core.Lambda)
	if !ok {
		t.Fatalf("expected Lambda, got %T", result.Decls[1])
	}
	_, ok = lam1.Body.(*core.Var)
	if !ok {
		t.Fatal("expected second decl body to remain Var")
	}
}
