package testing

import (
	"testing"

	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/types"
)

// Helper to create a mock CoreTypeInfo for testing
func makeMockCoreTypeInfo() types.CoreTypeInfo {
	return make(types.CoreTypeInfo)
}

// Helper to set type info for a binding (pure function)
func setTypePure(cti types.CoreTypeInfo, binding *core.RecBinding) {
	if binding.Value != nil {
		nodeID := binding.Value.ID()
		cti[nodeID] = &types.TFunc2{
			Params:    []types.Type{types.TInt},
			EffectRow: nil, // Pure
			Return:    types.TInt,
		}
	}
}

// Helper to set type info for an effectful function
func setTypeEffectful(cti types.CoreTypeInfo, binding *core.RecBinding, effects []string) {
	if binding.Value != nil {
		nodeID := binding.Value.ID()

		// Build effect row
		labels := make(map[string]types.Type)
		for _, eff := range effects {
			labels[eff] = types.Unit()
		}

		cti[nodeID] = &types.TFunc2{
			Params: []types.Type{types.TInt},
			EffectRow: &types.Row{
				Kind:   types.EffectRow,
				Labels: labels,
			},
			Return: types.TInt,
		}
	}
}

func TestIsPure_NilBinding(t *testing.T) {
	cti := makeMockCoreTypeInfo()
	binding := &core.RecBinding{Name: "test", Value: nil}

	if !IsPure(binding, cti) {
		t.Error("Expected nil binding to be pure")
	}
}

func TestIsPure_NilEffectRow(t *testing.T) {
	cti := makeMockCoreTypeInfo()
	body := makeLambda([]string{"n"}, &core.Lit{Value: int64(0)})
	binding := &core.RecBinding{Name: "f", Value: body}
	setTypePure(cti, binding)

	if !IsPure(binding, cti) {
		t.Error("Expected function with nil EffectRow to be pure")
	}
}

func TestIsPure_EmptyEffectRow(t *testing.T) {
	cti := makeMockCoreTypeInfo()
	body := makeLambda([]string{"n"}, &core.Lit{Value: int64(0)})
	binding := &core.RecBinding{Name: "f", Value: body}

	// Set with empty effect row
	nodeID := body.ID()
	cti[nodeID] = &types.TFunc2{
		Params:    []types.Type{types.TInt},
		EffectRow: &types.Row{Kind: types.EffectRow, Labels: map[string]types.Type{}},
		Return:    types.TInt,
	}

	if !IsPure(binding, cti) {
		t.Error("Expected function with empty EffectRow to be pure")
	}
}

func TestIsPure_WithEffects(t *testing.T) {
	cti := makeMockCoreTypeInfo()
	body := makeLambda([]string{"n"}, &core.Lit{Value: int64(0)})
	binding := &core.RecBinding{Name: "f", Value: body}
	setTypeEffectful(cti, binding, []string{"IO"})

	if IsPure(binding, cti) {
		t.Error("Expected function with IO effect to NOT be pure")
	}
}

func TestGetEffectNames_Pure(t *testing.T) {
	cti := makeMockCoreTypeInfo()
	body := makeLambda([]string{"n"}, &core.Lit{Value: int64(0)})
	binding := &core.RecBinding{Name: "f", Value: body}
	setTypePure(cti, binding)

	effects := GetEffectNames(binding, cti)
	if effects != nil {
		t.Errorf("Expected nil effects for pure function, got %v", effects)
	}
}

func TestGetEffectNames_Effectful(t *testing.T) {
	cti := makeMockCoreTypeInfo()
	body := makeLambda([]string{"n"}, &core.Lit{Value: int64(0)})
	binding := &core.RecBinding{Name: "f", Value: body}
	setTypeEffectful(cti, binding, []string{"IO", "FS"})

	effects := GetEffectNames(binding, cti)
	if len(effects) != 2 {
		t.Errorf("Expected 2 effects, got %v", effects)
	}
	// Effects should be sorted
	if effects[0] != "FS" || effects[1] != "IO" {
		t.Errorf("Expected [FS, IO], got %v", effects)
	}
}

func TestExtractPureCluster_SinglePureFunction(t *testing.T) {
	// Single function: factorial (self-recursive, no deps)
	factorialBody := makeLambda([]string{"n"},
		makeIf(
			makeBinOp("<=", makeVar("n"), &core.Lit{Value: int64(1)}),
			&core.Lit{Value: int64(1)},
			makeBinOp("*", makeVar("n"),
				makeApp(makeVar("factorial"),
					makeBinOp("-", makeVar("n"), &core.Lit{Value: int64(1)}))),
		),
	)

	prog := makeTestProgram(makeBinding("factorial", factorialBody))
	g := BuildCallGraph(prog)
	sccs := ComputeSCCs(g)

	cti := makeMockCoreTypeInfo()
	setTypePure(cti, g.Bindings["factorial"])

	cluster, err := ExtractPureCluster("factorial", g, sccs, cti)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if cluster.FuncName != "factorial" {
		t.Errorf("Expected FuncName 'factorial', got '%s'", cluster.FuncName)
	}
	if len(cluster.Bindings) != 1 {
		t.Errorf("Expected 1 binding, got %d", len(cluster.Bindings))
	}
	if !cluster.Names["factorial"] {
		t.Error("Expected 'factorial' in names")
	}
	if cluster.HasDependencies() {
		t.Error("Expected no dependencies for self-recursive function")
	}
}

func TestExtractPureCluster_WithPureDependency(t *testing.T) {
	// lcm -> gcd (both pure)
	gcdBody := makeLambda([]string{"a", "b"},
		makeIf(
			makeBinOp("==", makeVar("b"), &core.Lit{Value: int64(0)}),
			makeVar("a"),
			makeApp(makeVar("gcd"), makeVar("b"), makeBinOp("%", makeVar("a"), makeVar("b"))),
		),
	)
	lcmBody := makeLambda([]string{"a", "b"},
		makeBinOp("/",
			makeBinOp("*", makeVar("a"), makeVar("b")),
			makeApp(makeVar("gcd"), makeVar("a"), makeVar("b")),
		),
	)

	prog := makeTestProgram(
		makeBinding("gcd", gcdBody),
		makeBinding("lcm", lcmBody),
	)
	g := BuildCallGraph(prog)
	sccs := ComputeSCCs(g)

	cti := makeMockCoreTypeInfo()
	setTypePure(cti, g.Bindings["gcd"])
	setTypePure(cti, g.Bindings["lcm"])

	cluster, err := ExtractPureCluster("lcm", g, sccs, cti)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(cluster.Bindings) != 2 {
		t.Errorf("Expected 2 bindings, got %d", len(cluster.Bindings))
	}
	if !cluster.Names["lcm"] || !cluster.Names["gcd"] {
		t.Errorf("Expected both lcm and gcd in names, got %v", cluster.Names)
	}
	if !cluster.HasDependencies() {
		t.Error("Expected lcm to have dependencies")
	}

	deps := cluster.DependencyNames()
	if len(deps) != 1 || deps[0] != "gcd" {
		t.Errorf("Expected [gcd], got %v", deps)
	}
}

func TestExtractPureCluster_MutualRecursion(t *testing.T) {
	// isEven <-> isOdd (both pure)
	isEvenBody := makeLambda([]string{"n"},
		makeIf(
			makeBinOp("==", makeVar("n"), &core.Lit{Value: int64(0)}),
			&core.Lit{Value: true},
			makeApp(makeVar("isOdd"), makeBinOp("-", makeVar("n"), &core.Lit{Value: int64(1)})),
		),
	)
	isOddBody := makeLambda([]string{"n"},
		makeIf(
			makeBinOp("==", makeVar("n"), &core.Lit{Value: int64(0)}),
			&core.Lit{Value: false},
			makeApp(makeVar("isEven"), makeBinOp("-", makeVar("n"), &core.Lit{Value: int64(1)})),
		),
	)

	prog := makeTestProgram(
		makeBinding("isEven", isEvenBody),
		makeBinding("isOdd", isOddBody),
	)
	g := BuildCallGraph(prog)
	sccs := ComputeSCCs(g)

	cti := makeMockCoreTypeInfo()
	setTypePure(cti, g.Bindings["isEven"])
	setTypePure(cti, g.Bindings["isOdd"])

	// Test from isEven perspective
	cluster, err := ExtractPureCluster("isEven", g, sccs, cti)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(cluster.Bindings) != 2 {
		t.Errorf("Expected 2 bindings, got %d", len(cluster.Bindings))
	}
	if !cluster.Names["isEven"] || !cluster.Names["isOdd"] {
		t.Errorf("Expected both isEven and isOdd in names, got %v", cluster.Names)
	}
}

func TestExtractPureCluster_EffectfulDependency(t *testing.T) {
	// main -> effectful (main is pure, effectful has IO)
	effectfulBody := makeLambda([]string{"x"}, &core.Lit{Value: int64(0)})
	mainBody := makeLambda([]string{"x"}, makeApp(makeVar("effectful"), makeVar("x")))

	prog := makeTestProgram(
		makeBinding("effectful", effectfulBody),
		makeBinding("main", mainBody),
	)
	g := BuildCallGraph(prog)
	sccs := ComputeSCCs(g)

	cti := makeMockCoreTypeInfo()
	setTypePure(cti, g.Bindings["main"])
	setTypeEffectful(cti, g.Bindings["effectful"], []string{"IO"})

	_, err := ExtractPureCluster("main", g, sccs, cti)
	if err == nil {
		t.Fatal("Expected error for effectful dependency")
	}

	purityErr, ok := err.(*PurityError)
	if !ok {
		t.Fatalf("Expected PurityError, got %T: %v", err, err)
	}

	if purityErr.FuncUnderTest != "main" {
		t.Errorf("Expected FuncUnderTest 'main', got '%s'", purityErr.FuncUnderTest)
	}
	if purityErr.EffectfulFunc != "effectful" {
		t.Errorf("Expected EffectfulFunc 'effectful', got '%s'", purityErr.EffectfulFunc)
	}
	if len(purityErr.Effects) != 1 || purityErr.Effects[0] != "IO" {
		t.Errorf("Expected effects [IO], got %v", purityErr.Effects)
	}
}

func TestExtractPureCluster_NotFound(t *testing.T) {
	prog := makeTestProgram() // Empty program
	g := BuildCallGraph(prog)
	sccs := ComputeSCCs(g)
	cti := makeMockCoreTypeInfo()

	_, err := ExtractPureCluster("nonexistent", g, sccs, cti)
	if err == nil {
		t.Error("Expected error for non-existent function")
	}
}

func TestPurityError_String(t *testing.T) {
	err := &PurityError{
		FuncUnderTest: "myFunc",
		EffectfulFunc: "badDep",
		Effects:       []string{"IO", "FS"},
	}

	msg := err.Error()
	if msg != "cannot test 'myFunc': dependency 'badDep' has effects [IO FS]" {
		t.Errorf("Unexpected error message: %s", msg)
	}
}
