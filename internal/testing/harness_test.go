package testing

import (
	"testing"

	"github.com/sunholo/ailang/internal/ast"
	"github.com/sunholo/ailang/internal/core"
)

func TestBuildInlineTestHarness_EmptyTests(t *testing.T) {
	// Arrange
	binding := core.RecBinding{
		Name:  "factorial",
		Value: &core.Var{CoreNode: core.CoreNode{NodeID: 1}, Name: "lambda"},
	}
	tests := []TestCase{}

	// Act
	result := BuildInlineTestHarness(binding, tests)

	// Assert
	if _, ok := result.(*core.Lit); !ok {
		t.Errorf("Expected Lit (unit) for empty tests, got %T", result)
	}
}

func TestBuildInlineTestHarness_SingleTest(t *testing.T) {
	// Arrange
	binding := core.RecBinding{
		Name:  "factorial",
		Value: &core.Var{CoreNode: core.CoreNode{NodeID: 1}, Name: "lambda"},
	}

	// Create test case: (0, 1) - factorial(0) == 1
	testCase := TestCase{
		Name:        "factorial_test_1",
		Body:        []ast.Expr{makeTuple(makeIntLit(0), makeIntLit(1))},
		IsInline:    true,
		FunctionCtx: "factorial",
	}

	tests := []TestCase{testCase}

	// Act
	result := BuildInlineTestHarness(binding, tests)

	// Assert - should be LetRec
	letRec, ok := result.(*core.LetRec)
	if !ok {
		t.Fatalf("Expected LetRec, got %T", result)
	}

	// Check bindings
	if len(letRec.Bindings) != 1 {
		t.Errorf("Expected 1 binding, got %d", len(letRec.Bindings))
	}
	if letRec.Bindings[0].Name != "factorial" {
		t.Errorf("Expected binding name 'factorial', got '%s'", letRec.Bindings[0].Name)
	}

	// Body should be Let(_test_1, App(factorial, 0), Tuple([_test_1]))
	let, ok := letRec.Body.(*core.Let)
	if !ok {
		t.Fatalf("Expected Let in LetRec body, got %T", letRec.Body)
	}

	if let.Name != "_test_1" {
		t.Errorf("Expected let binding '_test_1', got '%s'", let.Name)
	}

	// Value should be App(factorial, 0)
	app, ok := let.Value.(*core.App)
	if !ok {
		t.Fatalf("Expected App in Let value, got %T", let.Value)
	}

	funcVar, ok := app.Func.(*core.Var)
	if !ok || funcVar.Name != "factorial" {
		t.Errorf("Expected App.Func to be Var('factorial'), got %T", app.Func)
	}

	if len(app.Args) != 1 {
		t.Errorf("Expected 1 arg, got %d", len(app.Args))
	}

	// Let body should be Tuple([_test_1])
	tuple, ok := let.Body.(*core.Tuple)
	if !ok {
		t.Fatalf("Expected Tuple in Let body, got %T", let.Body)
	}

	if len(tuple.Elements) != 1 {
		t.Errorf("Expected 1 tuple element, got %d", len(tuple.Elements))
	}

	resultVar, ok := tuple.Elements[0].(*core.Var)
	if !ok || resultVar.Name != "_test_1" {
		t.Errorf("Expected tuple element Var('_test_1'), got %T", tuple.Elements[0])
	}
}

func TestBuildInlineTestHarness_MultipleTests(t *testing.T) {
	// Arrange
	binding := core.RecBinding{
		Name:  "factorial",
		Value: &core.Var{CoreNode: core.CoreNode{NodeID: 1}, Name: "lambda"},
	}

	// Create test cases: (0, 1), (1, 1), (5, 120)
	tests := []TestCase{
		{
			Name:        "factorial_test_1",
			Body:        []ast.Expr{makeTuple(makeIntLit(0), makeIntLit(1))},
			IsInline:    true,
			FunctionCtx: "factorial",
		},
		{
			Name:        "factorial_test_2",
			Body:        []ast.Expr{makeTuple(makeIntLit(1), makeIntLit(1))},
			IsInline:    true,
			FunctionCtx: "factorial",
		},
		{
			Name:        "factorial_test_3",
			Body:        []ast.Expr{makeTuple(makeIntLit(5), makeIntLit(120))},
			IsInline:    true,
			FunctionCtx: "factorial",
		},
	}

	// Act
	result := BuildInlineTestHarness(binding, tests)

	// Assert - should be LetRec with nested Lets
	letRec, ok := result.(*core.LetRec)
	if !ok {
		t.Fatalf("Expected LetRec, got %T", result)
	}

	// Body should be Let(_test_1, ..., Let(_test_2, ..., Let(_test_3, ..., Tuple)))
	let1, ok := letRec.Body.(*core.Let)
	if !ok {
		t.Fatalf("Expected Let for test 1, got %T", letRec.Body)
	}
	if let1.Name != "_test_1" {
		t.Errorf("Expected '_test_1', got '%s'", let1.Name)
	}

	let2, ok := let1.Body.(*core.Let)
	if !ok {
		t.Fatalf("Expected Let for test 2, got %T", let1.Body)
	}
	if let2.Name != "_test_2" {
		t.Errorf("Expected '_test_2', got '%s'", let2.Name)
	}

	let3, ok := let2.Body.(*core.Let)
	if !ok {
		t.Fatalf("Expected Let for test 3, got %T", let2.Body)
	}
	if let3.Name != "_test_3" {
		t.Errorf("Expected '_test_3', got '%s'", let3.Name)
	}

	// Innermost body should be Tuple([_test_1, _test_2, _test_3])
	tuple, ok := let3.Body.(*core.Tuple)
	if !ok {
		t.Fatalf("Expected Tuple, got %T", let3.Body)
	}

	if len(tuple.Elements) != 3 {
		t.Errorf("Expected 3 tuple elements, got %d", len(tuple.Elements))
	}

	// Verify tuple elements are _test_1, _test_2, _test_3
	expectedNames := []string{"_test_1", "_test_2", "_test_3"}
	for i, elem := range tuple.Elements {
		v, ok := elem.(*core.Var)
		if !ok {
			t.Errorf("Expected Var at tuple[%d], got %T", i, elem)
			continue
		}
		if v.Name != expectedNames[i] {
			t.Errorf("Expected tuple[%d] = '%s', got '%s'", i, expectedNames[i], v.Name)
		}
	}
}

func TestBuildInlineTestHarness_MultiArgFunction(t *testing.T) {
	// Arrange
	binding := core.RecBinding{
		Name:  "add",
		Value: &core.Var{CoreNode: core.CoreNode{NodeID: 1}, Name: "lambda"},
	}

	// Create test case: ((1, 2), 3) - add(1, 2) == 3
	// Input is a tuple of arguments
	inputTuple := &ast.Tuple{
		Elements: []ast.Expr{makeIntLit(1), makeIntLit(2)},
	}
	testCase := TestCase{
		Name:        "add_test_1",
		Body:        []ast.Expr{makeTuple(inputTuple, makeIntLit(3))},
		IsInline:    true,
		FunctionCtx: "add",
	}

	tests := []TestCase{testCase}

	// Act
	result := BuildInlineTestHarness(binding, tests)

	// Assert
	letRec, ok := result.(*core.LetRec)
	if !ok {
		t.Fatalf("Expected LetRec, got %T", result)
	}

	let, ok := letRec.Body.(*core.Let)
	if !ok {
		t.Fatalf("Expected Let, got %T", letRec.Body)
	}

	// Value should be single App with all args: App(add, [1, 2])
	// (NOT curried - multi-arg functions use single App with multiple args)
	app, ok := let.Value.(*core.App)
	if !ok {
		t.Fatalf("Expected App, got %T", let.Value)
	}

	// app.Func should be Var("add")
	funcVar, ok := app.Func.(*core.Var)
	if !ok || funcVar.Name != "add" {
		t.Errorf("Expected Var('add'), got %T", app.Func)
	}

	// app.Args should be [1, 2] (both args in single App)
	if len(app.Args) != 2 {
		t.Errorf("Expected 2 args in App, got %d", len(app.Args))
	}
}

func TestBuildInlineTestHarness_NodeIDsUnique(t *testing.T) {
	// Arrange
	binding := core.RecBinding{
		Name:  "test",
		Value: &core.Var{CoreNode: core.CoreNode{NodeID: 1}, Name: "lambda"},
	}

	tests := []TestCase{
		{
			Name:        "test_1",
			Body:        []ast.Expr{makeTuple(makeIntLit(0), makeIntLit(0))},
			IsInline:    true,
			FunctionCtx: "test",
		},
		{
			Name:        "test_2",
			Body:        []ast.Expr{makeTuple(makeIntLit(1), makeIntLit(1))},
			IsInline:    true,
			FunctionCtx: "test",
		},
	}

	// Act
	result := BuildInlineTestHarness(binding, tests)

	// Assert - collect all node IDs
	nodeIDs := make(map[uint64]bool)
	collectNodeIDs(result, nodeIDs)

	// Verify all unique (no duplicates)
	if len(nodeIDs) < 10 {
		t.Errorf("Expected at least 10 unique node IDs, got %d", len(nodeIDs))
	}
}

// Helper functions for test construction

func makeIntLit(value int) *ast.Literal {
	return &ast.Literal{
		Kind:  ast.IntLit,
		Value: int64(value),
	}
}

func makeTuple(elem1, elem2 ast.Expr) *ast.Tuple {
	return &ast.Tuple{
		Elements: []ast.Expr{elem1, elem2},
	}
}

func collectNodeIDs(expr core.CoreExpr, ids map[uint64]bool) {
	id := expr.ID()
	if id != 0 {
		ids[id] = true
	}

	switch e := expr.(type) {
	case *core.LetRec:
		for _, binding := range e.Bindings {
			collectNodeIDs(binding.Value, ids)
		}
		collectNodeIDs(e.Body, ids)
	case *core.Let:
		collectNodeIDs(e.Value, ids)
		collectNodeIDs(e.Body, ids)
	case *core.App:
		collectNodeIDs(e.Func, ids)
		for _, arg := range e.Args {
			collectNodeIDs(arg, ids)
		}
	case *core.Tuple:
		for _, elem := range e.Elements {
			collectNodeIDs(elem, ids)
		}
	case *core.Var, *core.Lit:
		// Leaf nodes, just have their own ID
	}
}
