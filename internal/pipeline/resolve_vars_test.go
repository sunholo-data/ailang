package pipeline

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/types"
)

// TestVarResolverMonomorphicFloat tests that Var bound to monomorphic float gets resolved
func TestVarResolverMonomorphicFloat(t *testing.T) {
	// Setup: let x = 3.14 in x
	// CoreTI before: x → α1, 3.14 → float
	// CoreTI after: x → float (propagated from literal)

	coreTI := types.NewCoreTypeInfo()
	floatType := &types.TCon{Name: "float"}
	tvar := &types.TVar{Name: "α1"}

	// Create AST: Let(x, Lit(3.14), Var(x))
	lit := &core.Lit{
		CoreNode: core.CoreNode{NodeID: 1},
		Kind:     core.FloatLit,
		Value:    3.14,
	}
	varX := &core.Var{
		CoreNode: core.CoreNode{NodeID: 2},
		Name:     "x",
	}
	letExpr := &core.Let{
		CoreNode: core.CoreNode{NodeID: 3},
		Name:     "x",
		Value:    lit,
		Body:     varX,
	}
	prog := &core.Program{Decls: []core.CoreExpr{letExpr}}

	// Set initial CoreTI
	coreTI.Set(lit.ID(), floatType)      // Literal has concrete type
	coreTI.Set(varX.ID(), tvar)          // Var has TVar (unresolved)
	coreTI.Set(letExpr.ID(), floatType)  // Let result type

	// Run resolver
	resolver := NewVarResolver(coreTI)
	resolver.Resolve(prog)

	// Verify: Var should now have float type (propagated from literal)
	resolvedType, ok := coreTI.Get(varX.ID())
	assert.True(t, ok, "Var should still have CoreTI entry")
	assert.Equal(t, floatType, resolvedType, "Var should have float type from literal")
}

// TestVarResolverLetChain tests propagation through ANF temporaries
func TestVarResolverLetChain(t *testing.T) {
	// Setup: let x = 3.14 in let y = x in y
	// CoreTI before: x → α1, y → α2, 3.14 → float
	// CoreTI after: x → float, y → float (chained propagation)

	coreTI := types.NewCoreTypeInfo()
	floatType := &types.TCon{Name: "float"}
	tvar1 := &types.TVar{Name: "α1"}
	tvar2 := &types.TVar{Name: "α2"}

	// Create AST: Let(x, Lit(3.14), Let(y, Var(x), Var(y)))
	lit := &core.Lit{
		CoreNode: core.CoreNode{NodeID: 1},
		Kind:     core.FloatLit,
		Value:    3.14,
	}
	varX := &core.Var{
		CoreNode: core.CoreNode{NodeID: 2},
		Name:     "x",
	}
	varY := &core.Var{
		CoreNode: core.CoreNode{NodeID: 3},
		Name:     "y",
	}
	innerLet := &core.Let{
		CoreNode: core.CoreNode{NodeID: 4},
		Name:     "y",
		Value:    varX,
		Body:     varY,
	}
	outerLet := &core.Let{
		CoreNode: core.CoreNode{NodeID: 5},
		Name:     "x",
		Value:    lit,
		Body:     innerLet,
	}
	prog := &core.Program{Decls: []core.CoreExpr{outerLet}}

	// Set initial CoreTI
	coreTI.Set(lit.ID(), floatType)       // Literal has concrete type
	coreTI.Set(varX.ID(), tvar1)          // First Var has TVar
	coreTI.Set(varY.ID(), tvar2)          // Second Var has TVar
	coreTI.Set(innerLet.ID(), floatType)
	coreTI.Set(outerLet.ID(), floatType)

	// Run resolver
	resolver := NewVarResolver(coreTI)
	resolver.Resolve(prog)

	// Verify: Both Vars should have float type
	xType, okX := coreTI.Get(varX.ID())
	yType, okY := coreTI.Get(varY.ID())

	assert.True(t, okX, "Var x should have CoreTI entry")
	assert.True(t, okY, "Var y should have CoreTI entry")
	assert.Equal(t, floatType, xType, "Var x should have float type from literal")
	assert.Equal(t, floatType, yType, "Var y should have float type from Var x")
}

// TestVarResolverPolymorphicParam tests that lambda params stay polymorphic
func TestVarResolverPolymorphicParam(t *testing.T) {
	// Setup: \x. x (identity function)
	// CoreTI before: x → α1, lambda → α1 -> α1
	// CoreTI after: x → α1 (unchanged - preserve polymorphism)

	coreTI := types.NewCoreTypeInfo()
	tvar := &types.TVar{Name: "α1"}
	funcType := &types.TFunc2{
		Params:    []types.Type{tvar},
		Return:    tvar,
		EffectRow: &types.Row{Kind: types.KEffect{}, Labels: map[string]types.Type{}, Tail: nil},
	}

	// Create AST: Lambda(x, Var(x))
	varX := &core.Var{
		CoreNode: core.CoreNode{NodeID: 1},
		Name:     "x",
	}
	lambda := &core.Lambda{
		CoreNode: core.CoreNode{NodeID: 2},
		Params:   []string{"x"},
		Body:     varX,
	}
	prog := &core.Program{Decls: []core.CoreExpr{lambda}}

	// Set initial CoreTI
	coreTI.Set(varX.ID(), tvar)       // Lambda param has TVar
	coreTI.Set(lambda.ID(), funcType) // Lambda has polymorphic type

	// Run resolver
	resolver := NewVarResolver(coreTI)
	resolver.Resolve(prog)

	// Verify: Var should STILL have TVar (no propagation from polymorphic context)
	resolvedType, ok := coreTI.Get(varX.ID())
	assert.True(t, ok, "Var should still have CoreTI entry")
	assert.IsType(t, &types.TVar{}, resolvedType, "Var should remain polymorphic (TVar)")
}

// TestVarResolverMixedBindings tests selective propagation (mono vs poly)
func TestVarResolverMixedBindings(t *testing.T) {
	// Setup: let x = 3.14 in (\y. x + y)
	// x is monomorphic (float), y is polymorphic (α1)
	// CoreTI before: x → α1, y → α2, 3.14 → float
	// CoreTI after: x → float (propagated), y → α2 (unchanged)

	coreTI := types.NewCoreTypeInfo()
	floatType := &types.TCon{Name: "float"}
	tvar1 := &types.TVar{Name: "α1"}
	tvar2 := &types.TVar{Name: "α2"}

	// Create AST: Let(x, Lit(3.14), Lambda(y, BinOp(Var(x), Add, Var(y))))
	lit := &core.Lit{
		CoreNode: core.CoreNode{NodeID: 1},
		Kind:     core.FloatLit,
		Value:    3.14,
	}
	varX := &core.Var{
		CoreNode: core.CoreNode{NodeID: 2},
		Name:     "x",
	}
	varY := &core.Var{
		CoreNode: core.CoreNode{NodeID: 3},
		Name:     "y",
	}
	binop := &core.BinOp{
		CoreNode: core.CoreNode{NodeID: 4},
		Op:       "+",
		Left:     varX,
		Right:    varY,
	}
	lambda := &core.Lambda{
		CoreNode: core.CoreNode{NodeID: 5},
		Params:   []string{"y"},
		Body:     binop,
	}
	letExpr := &core.Let{
		CoreNode: core.CoreNode{NodeID: 6},
		Name:     "x",
		Value:    lit,
		Body:     lambda,
	}
	prog := &core.Program{Decls: []core.CoreExpr{letExpr}}

	// Set initial CoreTI
	coreTI.Set(lit.ID(), floatType)
	coreTI.Set(varX.ID(), tvar1) // Var x has TVar (should be resolved)
	coreTI.Set(varY.ID(), tvar2) // Var y has TVar (should stay polymorphic)
	coreTI.Set(binop.ID(), floatType)
	coreTI.Set(lambda.ID(), &types.TFunc2{
		Params:    []types.Type{floatType},
		Return:    floatType,
		EffectRow: &types.Row{Kind: types.KEffect{}, Labels: map[string]types.Type{}, Tail: nil},
	})
	coreTI.Set(letExpr.ID(), floatType)

	// Run resolver
	resolver := NewVarResolver(coreTI)
	resolver.Resolve(prog)

	// Verify: x should be resolved, y should stay TVar
	xType, okX := coreTI.Get(varX.ID())
	yType, okY := coreTI.Get(varY.ID())

	assert.True(t, okX, "Var x should have CoreTI entry")
	assert.True(t, okY, "Var y should have CoreTI entry")
	assert.Equal(t, floatType, xType, "Var x should have float type (monomorphic)")
	assert.IsType(t, &types.TVar{}, yType, "Var y should remain TVar (polymorphic param)")
}

// TestVarResolverIdempotent tests that running resolver twice has no effect
func TestVarResolverIdempotent(t *testing.T) {
	// Setup: let x = 3.14 in x
	// Run resolver twice, verify type stays the same

	coreTI := types.NewCoreTypeInfo()
	floatType := &types.TCon{Name: "float"}
	tvar := &types.TVar{Name: "α1"}

	lit := &core.Lit{
		CoreNode: core.CoreNode{NodeID: 1},
		Kind:     core.FloatLit,
		Value:    3.14,
	}
	varX := &core.Var{
		CoreNode: core.CoreNode{NodeID: 2},
		Name:     "x",
	}
	letExpr := &core.Let{
		CoreNode: core.CoreNode{NodeID: 3},
		Name:     "x",
		Value:    lit,
		Body:     varX,
	}
	prog := &core.Program{Decls: []core.CoreExpr{letExpr}}

	coreTI.Set(lit.ID(), floatType)
	coreTI.Set(varX.ID(), tvar)
	coreTI.Set(letExpr.ID(), floatType)

	// First run
	resolver1 := NewVarResolver(coreTI)
	resolver1.Resolve(prog)
	typeAfterFirst, _ := coreTI.Get(varX.ID())

	// Second run
	resolver2 := NewVarResolver(coreTI)
	resolver2.Resolve(prog)
	typeAfterSecond, _ := coreTI.Get(varX.ID())

	// Verify: Type should be the same after both runs
	assert.Equal(t, typeAfterFirst, typeAfterSecond, "Resolver should be idempotent")
	assert.Equal(t, floatType, typeAfterSecond, "Type should be float after both runs")
}

// TestVarResolverNestedLet tests nested let bindings with shadowing
func TestVarResolverNestedLet(t *testing.T) {
	// Setup: let x = 3.14 in let x = 42 in x
	// Outer x is float, inner x is int, Var should resolve to inner (int)

	coreTI := types.NewCoreTypeInfo()
	floatType := &types.TCon{Name: "float"}
	intType := &types.TCon{Name: "int"}
	tvar := &types.TVar{Name: "α1"}

	// Create AST
	floatLit := &core.Lit{
		CoreNode: core.CoreNode{NodeID: 1},
		Kind:     core.FloatLit,
		Value:    3.14,
	}
	intLit := &core.Lit{
		CoreNode: core.CoreNode{NodeID: 2},
		Kind:     core.IntLit,
		Value:    42,
	}
	varX := &core.Var{
		CoreNode: core.CoreNode{NodeID: 3},
		Name:     "x",
	}
	innerLet := &core.Let{
		CoreNode: core.CoreNode{NodeID: 4},
		Name:     "x",
		Value:    intLit,
		Body:     varX,
	}
	outerLet := &core.Let{
		CoreNode: core.CoreNode{NodeID: 5},
		Name:     "x",
		Value:    floatLit,
		Body:     innerLet,
	}
	prog := &core.Program{Decls: []core.CoreExpr{outerLet}}

	// Set initial CoreTI
	coreTI.Set(floatLit.ID(), floatType)
	coreTI.Set(intLit.ID(), intType)
	coreTI.Set(varX.ID(), tvar)
	coreTI.Set(innerLet.ID(), intType)
	coreTI.Set(outerLet.ID(), intType)

	// Run resolver
	resolver := NewVarResolver(coreTI)
	resolver.Resolve(prog)

	// Verify: Var should have int type (from inner binding, not outer)
	resolvedType, ok := coreTI.Get(varX.ID())
	assert.True(t, ok, "Var should have CoreTI entry")
	assert.Equal(t, intType, resolvedType, "Var should have int type (from inner binding)")
}

// TestVarResolverNonMonomorphic tests that polymorphic let bindings aren't propagated
func TestVarResolverNonMonomorphic(t *testing.T) {
	// Setup: let id = \x. x in id(3.14)
	// id has polymorphic type (α → α), should NOT propagate to Var(id)

	coreTI := types.NewCoreTypeInfo()
	tvar := &types.TVar{Name: "α1"}
	polyType := &types.TFunc2{
		Params:    []types.Type{tvar},
		Return:    tvar,
		EffectRow: &types.Row{Kind: types.KEffect{}, Labels: map[string]types.Type{}, Tail: nil},
	}

	// Create AST: Let(id, Lambda(x, Var(x)), App(Var(id), Lit(3.14)))
	varXInLambda := &core.Var{
		CoreNode: core.CoreNode{NodeID: 1},
		Name:     "x",
	}
	lambda := &core.Lambda{
		CoreNode: core.CoreNode{NodeID: 2},
		Params:   []string{"x"},
		Body:     varXInLambda,
	}
	varID := &core.Var{
		CoreNode: core.CoreNode{NodeID: 3},
		Name:     "id",
	}
	lit := &core.Lit{
		CoreNode: core.CoreNode{NodeID: 4},
		Kind:     core.FloatLit,
		Value:    3.14,
	}
	app := &core.App{
		CoreNode: core.CoreNode{NodeID: 5},
		Func:     varID,
		Args:     []core.CoreExpr{lit},
	}
	letExpr := &core.Let{
		CoreNode: core.CoreNode{NodeID: 6},
		Name:     "id",
		Value:    lambda,
		Body:     app,
	}
	prog := &core.Program{Decls: []core.CoreExpr{letExpr}}

	// Set initial CoreTI
	coreTI.Set(varXInLambda.ID(), tvar)
	coreTI.Set(lambda.ID(), polyType)
	coreTI.Set(varID.ID(), &types.TVar{Name: "α2"}) // Var(id) has TVar
	coreTI.Set(lit.ID(), &types.TCon{Name: "float"})
	coreTI.Set(app.ID(), &types.TCon{Name: "float"})
	coreTI.Set(letExpr.ID(), &types.TCon{Name: "float"})

	// Run resolver
	resolver := NewVarResolver(coreTI)
	resolver.Resolve(prog)

	// Verify: Var(id) should STILL have TVar (polymorphic binding not propagated)
	idType, ok := coreTI.Get(varID.ID())
	assert.True(t, ok, "Var(id) should have CoreTI entry")
	assert.IsType(t, &types.TVar{}, idType, "Var(id) should remain TVar (polymorphic binding)")
}
