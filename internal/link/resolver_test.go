package link

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/eval"
)

// mockCompileUnit implements CompileUnit for testing.
type mockCompileUnit struct {
	core     *core.Program
	moduleID string
}

func (m *mockCompileUnit) GetCore() *core.Program { return m.core }
func (m *mockCompileUnit) GetModuleID() string    { return m.moduleID }

// TestResolver_SequentialLetChain verifies that sequential Let bindings
// accumulate in the evaluator env, so later functions can reference earlier ones.
// This is the root cause of the "undefined variable" bug when packages with
// helper functions are loaded as dependencies.
func TestResolver_SequentialLetChain(t *testing.T) {
	// Build a module with 3 sequential Lets:
	//   let f1 = \() -> 10
	//   let f2 = \() -> f1()     (references f1)
	//   let f3 = \() -> f2()     (references f2, transitively f1)
	prog := &core.Program{
		Decls: []core.CoreExpr{
			&core.Let{
				Name: "f1",
				Value: &core.Lambda{
					Params: []string{"_unit"},
					Body:   &core.Lit{Kind: core.IntLit, Value: int64(10)},
				},
				Body: &core.Lit{Kind: core.UnitLit},
			},
			&core.Let{
				Name: "f2",
				Value: &core.Lambda{
					Params: []string{"_unit"},
					Body: &core.App{
						Func: &core.Var{Name: "f1"},
						Args: []core.CoreExpr{&core.Lit{Kind: core.UnitLit}},
					},
				},
				Body: &core.Lit{Kind: core.UnitLit},
			},
			&core.Let{
				Name: "f3",
				Value: &core.Lambda{
					Params: []string{"_unit"},
					Body: &core.App{
						Func: &core.Var{Name: "f2"},
						Args: []core.CoreExpr{&core.Lit{Kind: core.UnitLit}},
					},
				},
				Body: &core.Lit{Kind: core.UnitLit},
			},
		},
		Meta: map[string]*core.DeclMeta{
			"f1": {Name: "f1", IsExport: true},
			"f2": {Name: "f2", IsExport: true},
			"f3": {Name: "f3", IsExport: true},
		},
	}

	linker := NewModuleLinker(nil)
	resolver := NewResolver(linker)
	resolver.RegisterCompiledModule("test/helpers", &mockCompileUnit{
		core:     prog,
		moduleID: "test/helpers",
	})

	// Resolve f3 — triggers evaluation of the entire module.
	// Before the fix, f2's lambda would fail with "undefined variable: f1"
	val, err := resolver.ResolveValue(core.GlobalRef{Module: "test/helpers", Name: "f3"})
	if err != nil {
		t.Fatalf("ResolveValue(f3) failed: %v", err)
	}
	if val == nil {
		t.Fatal("ResolveValue(f3) returned nil")
	}

	// Call f3 to verify the chain works at runtime
	evaluator := eval.NewCoreEvaluator()
	evaluator.SetGlobalResolver(resolver)
	result, err := evaluator.CallValueN(val, []eval.Value{&eval.UnitValue{}})
	if err != nil {
		t.Fatalf("Calling f3() failed: %v", err)
	}

	intVal, ok := result.(*eval.IntValue)
	if !ok {
		t.Fatalf("f3() returned %T, want *eval.IntValue", result)
	}
	if intVal.Value != 10 {
		t.Errorf("f3() = %d, want 10", intVal.Value)
	}
}

// TestResolver_LetRecFollowedByLet verifies that LetRec bindings are
// visible to subsequent Let declarations.
func TestResolver_LetRecFollowedByLet(t *testing.T) {
	// Build a module:
	//   letrec recA = \() -> 42
	//          recB = \() -> recA()   (mutual group, references recA)
	//   let wrapper = \() -> recB()   (Let that references LetRec binding)
	prog := &core.Program{
		Decls: []core.CoreExpr{
			&core.LetRec{
				Bindings: []core.RecBinding{
					{
						Name: "recA",
						Value: &core.Lambda{
							Params: []string{"_"},
							Body:   &core.Lit{Kind: core.IntLit, Value: int64(42)},
						},
					},
					{
						Name: "recB",
						Value: &core.Lambda{
							Params: []string{"_"},
							Body: &core.App{
								Func: &core.Var{Name: "recA"},
								Args: []core.CoreExpr{&core.Lit{Kind: core.UnitLit}},
							},
						},
					},
				},
				Body: &core.Lit{Kind: core.UnitLit},
			},
			// let wrapper = \() -> recB()
			&core.Let{
				Name: "wrapper",
				Value: &core.Lambda{
					Params: []string{"_"},
					Body: &core.App{
						Func: &core.Var{Name: "recB"},
						Args: []core.CoreExpr{&core.Lit{Kind: core.UnitLit}},
					},
				},
				Body: &core.Lit{Kind: core.UnitLit},
			},
		},
		Meta: map[string]*core.DeclMeta{
			"recA":    {Name: "recA", IsExport: false},
			"recB":    {Name: "recB", IsExport: false},
			"wrapper": {Name: "wrapper", IsExport: true},
		},
	}

	linker := NewModuleLinker(nil)
	resolver := NewResolver(linker)
	resolver.RegisterCompiledModule("test/letrec", &mockCompileUnit{
		core:     prog,
		moduleID: "test/letrec",
	})

	// Resolve wrapper — triggers module eval.
	// Before the fix, wrapper's lambda would fail with "undefined variable: recB"
	// because LetRec bindings weren't accumulated in the evaluator env.
	val, err := resolver.ResolveValue(core.GlobalRef{Module: "test/letrec", Name: "wrapper"})
	if err != nil {
		t.Fatalf("ResolveValue(wrapper) failed: %v", err)
	}

	evaluator := eval.NewCoreEvaluator()
	evaluator.SetGlobalResolver(resolver)

	result, err := evaluator.CallValueN(val, []eval.Value{&eval.UnitValue{}})
	if err != nil {
		t.Fatalf("Calling wrapper() failed: %v", err)
	}

	intVal, ok := result.(*eval.IntValue)
	if !ok {
		t.Fatalf("wrapper() returned %T, want *eval.IntValue", result)
	}
	if intVal.Value != 42 {
		t.Errorf("wrapper() = %d, want 42", intVal.Value)
	}
}
