package testing

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/core"
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

// Tests for BuildClusterTestHarness

func TestBuildClusterTestHarness_EmptyTests(t *testing.T) {
	cluster := &PureCluster{
		FuncName: "test",
		Bindings: []core.RecBinding{
			{Name: "test", Value: &core.Var{CoreNode: core.CoreNode{NodeID: 1}, Name: "lambda"}},
		},
		Names: map[string]bool{"test": true},
	}

	result := BuildClusterTestHarness(cluster, []TestCase{})

	if _, ok := result.(*core.Lit); !ok {
		t.Errorf("Expected Lit (unit) for empty tests, got %T", result)
	}
}

func TestBuildClusterTestHarness_SingleBinding(t *testing.T) {
	// Single function (like factorial) - should work same as BuildInlineTestHarness
	cluster := &PureCluster{
		FuncName: "factorial",
		Bindings: []core.RecBinding{
			{Name: "factorial", Value: &core.Var{CoreNode: core.CoreNode{NodeID: 1}, Name: "lambda"}},
		},
		Names: map[string]bool{"factorial": true},
	}

	testCase := TestCase{
		Name:        "factorial_test_1",
		Body:        []ast.Expr{makeTuple(makeIntLit(5), makeIntLit(120))},
		IsInline:    true,
		FunctionCtx: "factorial",
	}

	result := BuildClusterTestHarness(cluster, []TestCase{testCase})

	letRec, ok := result.(*core.LetRec)
	if !ok {
		t.Fatalf("Expected LetRec, got %T", result)
	}

	// Should have single binding
	if len(letRec.Bindings) != 1 {
		t.Errorf("Expected 1 binding, got %d", len(letRec.Bindings))
	}
	if letRec.Bindings[0].Name != "factorial" {
		t.Errorf("Expected binding 'factorial', got '%s'", letRec.Bindings[0].Name)
	}

	// Body should call factorial
	let, ok := letRec.Body.(*core.Let)
	if !ok {
		t.Fatalf("Expected Let, got %T", letRec.Body)
	}

	app, ok := let.Value.(*core.App)
	if !ok {
		t.Fatalf("Expected App, got %T", let.Value)
	}

	funcVar, ok := app.Func.(*core.Var)
	if !ok || funcVar.Name != "factorial" {
		t.Errorf("Expected call to 'factorial', got %T", app.Func)
	}
}

func TestBuildClusterTestHarness_MultipleBindings(t *testing.T) {
	// lcm depends on gcd - cluster should have both
	cluster := &PureCluster{
		FuncName: "lcm",
		Bindings: []core.RecBinding{
			{Name: "gcd", Value: &core.Var{CoreNode: core.CoreNode{NodeID: 1}, Name: "gcd_lambda"}},
			{Name: "lcm", Value: &core.Var{CoreNode: core.CoreNode{NodeID: 2}, Name: "lcm_lambda"}},
		},
		Names: map[string]bool{"gcd": true, "lcm": true},
	}

	// Input for lcm is tuple (a, b)
	inputTuple := &ast.Tuple{Elements: []ast.Expr{makeIntLit(12), makeIntLit(8)}}
	testCase := TestCase{
		Name:        "lcm_test_1",
		Body:        []ast.Expr{makeTuple(inputTuple, makeIntLit(24))},
		IsInline:    true,
		FunctionCtx: "lcm",
	}

	result := BuildClusterTestHarness(cluster, []TestCase{testCase})

	letRec, ok := result.(*core.LetRec)
	if !ok {
		t.Fatalf("Expected LetRec, got %T", result)
	}

	// Should have both bindings
	if len(letRec.Bindings) != 2 {
		t.Errorf("Expected 2 bindings, got %d", len(letRec.Bindings))
	}

	// Collect binding names
	bindingNames := make(map[string]bool)
	for _, b := range letRec.Bindings {
		bindingNames[b.Name] = true
	}
	if !bindingNames["gcd"] || !bindingNames["lcm"] {
		t.Errorf("Expected bindings 'gcd' and 'lcm', got %v", bindingNames)
	}

	// Body should call lcm (the function under test), not gcd
	let, ok := letRec.Body.(*core.Let)
	if !ok {
		t.Fatalf("Expected Let, got %T", letRec.Body)
	}

	app, ok := let.Value.(*core.App)
	if !ok {
		t.Fatalf("Expected App, got %T", let.Value)
	}

	funcVar, ok := app.Func.(*core.Var)
	if !ok || funcVar.Name != "lcm" {
		t.Errorf("Expected call to 'lcm', got %T with name '%s'", app.Func, funcVar.Name)
	}
}

func TestBuildClusterTestHarness_MutualRecursion(t *testing.T) {
	// isEven and isOdd are mutually recursive
	cluster := &PureCluster{
		FuncName: "isEven",
		Bindings: []core.RecBinding{
			{Name: "isEven", Value: &core.Var{CoreNode: core.CoreNode{NodeID: 1}, Name: "isEven_lambda"}},
			{Name: "isOdd", Value: &core.Var{CoreNode: core.CoreNode{NodeID: 2}, Name: "isOdd_lambda"}},
		},
		Names: map[string]bool{"isEven": true, "isOdd": true},
	}

	// Test cases for isEven
	tests := []TestCase{
		{
			Name:        "isEven_test_1",
			Body:        []ast.Expr{makeTuple(makeIntLit(0), makeBoolLit(true))},
			IsInline:    true,
			FunctionCtx: "isEven",
		},
		{
			Name:        "isEven_test_2",
			Body:        []ast.Expr{makeTuple(makeIntLit(1), makeBoolLit(false))},
			IsInline:    true,
			FunctionCtx: "isEven",
		},
		{
			Name:        "isEven_test_3",
			Body:        []ast.Expr{makeTuple(makeIntLit(4), makeBoolLit(true))},
			IsInline:    true,
			FunctionCtx: "isEven",
		},
	}

	result := BuildClusterTestHarness(cluster, tests)

	letRec, ok := result.(*core.LetRec)
	if !ok {
		t.Fatalf("Expected LetRec, got %T", result)
	}

	// Both mutually recursive functions should be in the LetRec
	if len(letRec.Bindings) != 2 {
		t.Errorf("Expected 2 bindings for mutual recursion, got %d", len(letRec.Bindings))
	}

	// Verify 3 test calls
	let1, ok := letRec.Body.(*core.Let)
	if !ok {
		t.Fatalf("Expected Let for test 1, got %T", letRec.Body)
	}

	let2, ok := let1.Body.(*core.Let)
	if !ok {
		t.Fatalf("Expected Let for test 2, got %T", let1.Body)
	}

	let3, ok := let2.Body.(*core.Let)
	if !ok {
		t.Fatalf("Expected Let for test 3, got %T", let2.Body)
	}

	tuple, ok := let3.Body.(*core.Tuple)
	if !ok {
		t.Fatalf("Expected Tuple, got %T", let3.Body)
	}

	if len(tuple.Elements) != 3 {
		t.Errorf("Expected 3 tuple elements, got %d", len(tuple.Elements))
	}
}

func makeBoolLit(value bool) *ast.Literal {
	return &ast.Literal{
		Kind:  ast.BoolLit,
		Value: value,
	}
}

func makeStringLit(value string) *ast.Literal {
	return &ast.Literal{
		Kind:  ast.StringLit,
		Value: value,
	}
}

func makeIdent(name string) *ast.Identifier {
	return &ast.Identifier{Name: name}
}

// Tests for BuildEnsuresPropertyHarness (M-DX26 Phase 5)

// TestBuildEnsuresPropertyHarness_SingleArg verifies the harness shape for
// `pure func absolute(x: int) -> int ensures { result >= 0 }`
// with a generated input value of 5.
func TestBuildEnsuresPropertyHarness_SingleArg(t *testing.T) {
	binding := core.RecBinding{
		Name:  "absolute",
		Value: &core.Var{CoreNode: core.CoreNode{NodeID: 1}, Name: "lambda"},
	}

	params := []EnsuresParam{
		{Name: "x", Value: &core.Lit{CoreNode: core.CoreNode{NodeID: 2}, Kind: core.IntLit, Value: int64(5)}},
	}

	// Predicate: result >= 0
	predicate := &ast.BinaryOp{
		Left:  makeIdent("result"),
		Op:    ">=",
		Right: makeIntLit(0),
	}

	result := BuildEnsuresPropertyHarness(binding, params, predicate)

	letRec, ok := result.(*core.LetRec)
	if !ok {
		t.Fatalf("Expected LetRec, got %T", result)
	}
	if len(letRec.Bindings) != 1 || letRec.Bindings[0].Name != "absolute" {
		t.Fatalf("Expected single binding 'absolute', got %v", letRec.Bindings)
	}

	// Outer Let binds the parameter `x` to the generated value.
	xLet, ok := letRec.Body.(*core.Let)
	if !ok {
		t.Fatalf("Expected outer Let (param binding) in LetRec body, got %T", letRec.Body)
	}
	if xLet.Name != "x" {
		t.Errorf("Expected outer Let.Name = 'x' (function param name), got %q", xLet.Name)
	}

	// Inner Let binds `result` to the function call.
	resultLet, ok := xLet.Body.(*core.Let)
	if !ok {
		t.Fatalf("Expected inner Let (result binding), got %T", xLet.Body)
	}
	if resultLet.Name != "result" {
		t.Errorf("Expected inner Let.Name = 'result', got %q", resultLet.Name)
	}

	app, ok := resultLet.Value.(*core.App)
	if !ok {
		t.Fatalf("Expected App in result Let.Value, got %T", resultLet.Value)
	}
	funcVar, ok := app.Func.(*core.Var)
	if !ok || funcVar.Name != "absolute" {
		t.Errorf("Expected App.Func = Var('absolute'), got %T", app.Func)
	}
	if len(app.Args) != 1 {
		t.Errorf("Expected 1 arg, got %d", len(app.Args))
	}
	// The arg should be a Var reference to "x", not the raw literal.
	argVar, ok := app.Args[0].(*core.Var)
	if !ok || argVar.Name != "x" {
		t.Errorf("Expected App.Args[0] = Var('x'), got %T", app.Args[0])
	}

	binOp, ok := resultLet.Body.(*core.BinOp)
	if !ok {
		t.Fatalf("Expected BinOp in result Let.Body (predicate), got %T", resultLet.Body)
	}
	if binOp.Op != ">=" {
		t.Errorf("Expected predicate Op '>=', got %q", binOp.Op)
	}
	leftVar, ok := binOp.Left.(*core.Var)
	if !ok || leftVar.Name != "result" {
		t.Errorf("Expected predicate Left = Var('result'), got %T", binOp.Left)
	}
}

// TestBuildEnsuresPropertyHarness_MultiArg verifies multi-argument function call
// `pure func add(x: int, y: int) -> int ensures { result == x + y }`
// with generated values (3, 4).
func TestBuildEnsuresPropertyHarness_MultiArg(t *testing.T) {
	binding := core.RecBinding{
		Name:  "add",
		Value: &core.Var{CoreNode: core.CoreNode{NodeID: 1}, Name: "lambda"},
	}

	params := []EnsuresParam{
		{Name: "x", Value: &core.Lit{CoreNode: core.CoreNode{NodeID: 2}, Kind: core.IntLit, Value: int64(3)}},
		{Name: "y", Value: &core.Lit{CoreNode: core.CoreNode{NodeID: 3}, Kind: core.IntLit, Value: int64(4)}},
	}

	// Predicate: result == x + y
	predicate := &ast.BinaryOp{
		Left: makeIdent("result"),
		Op:   "==",
		Right: &ast.BinaryOp{
			Left:  makeIdent("x"),
			Op:    "+",
			Right: makeIdent("y"),
		},
	}

	result := BuildEnsuresPropertyHarness(binding, params, predicate)

	// Outer = Let(x, 3, ...), then Let(y, 4, ...), then Let(result, App, predicate).
	letRec := result.(*core.LetRec)
	xLet := letRec.Body.(*core.Let)
	if xLet.Name != "x" {
		t.Errorf("Expected outer Let name 'x', got %q", xLet.Name)
	}
	yLet := xLet.Body.(*core.Let)
	if yLet.Name != "y" {
		t.Errorf("Expected next Let name 'y', got %q", yLet.Name)
	}
	resultLet := yLet.Body.(*core.Let)
	app := resultLet.Value.(*core.App)

	if len(app.Args) != 2 {
		t.Fatalf("Expected 2 args, got %d", len(app.Args))
	}
	for i, name := range []string{"x", "y"} {
		argVar, ok := app.Args[i].(*core.Var)
		if !ok {
			t.Fatalf("Expected Var at arg %d, got %T", i, app.Args[i])
		}
		if argVar.Name != name {
			t.Errorf("Expected arg %d = Var(%s), got Var(%s)", i, name, argVar.Name)
		}
	}
}

// TestBuildEnsuresPropertyHarness_StringPredicate verifies string-returning func
// with a string-equality predicate (the reporter's exact case).
func TestBuildEnsuresPropertyHarness_StringPredicate(t *testing.T) {
	binding := core.RecBinding{
		Name:  "tag",
		Value: &core.Var{CoreNode: core.CoreNode{NodeID: 1}, Name: "lambda"},
	}

	params := []EnsuresParam{
		{Name: "n", Value: &core.Lit{CoreNode: core.CoreNode{NodeID: 2}, Kind: core.IntLit, Value: int64(0)}},
	}

	// Predicate: result == "neg" || result == "pos"
	predicate := &ast.BinaryOp{
		Left: &ast.BinaryOp{
			Left:  makeIdent("result"),
			Op:    "==",
			Right: makeStringLit("neg"),
		},
		Op: "||",
		Right: &ast.BinaryOp{
			Left:  makeIdent("result"),
			Op:    "==",
			Right: makeStringLit("pos"),
		},
	}

	result := BuildEnsuresPropertyHarness(binding, params, predicate)

	letRec := result.(*core.LetRec)
	nLet := letRec.Body.(*core.Let)
	resultLet := nLet.Body.(*core.Let)
	binOp, ok := resultLet.Body.(*core.BinOp)
	if !ok {
		t.Fatalf("Expected BinOp predicate, got %T", resultLet.Body)
	}
	if binOp.Op != "||" {
		t.Errorf("Expected outer Op '||', got %q", binOp.Op)
	}
}

// TestBuildEnsuresPropertyHarness_PredicateIgnoresResult verifies a predicate
// that doesn't reference `result` still builds a valid harness — the function is
// still called (its return value goes into `result`, unused), and the predicate
// evaluates against parameter references only.
func TestBuildEnsuresPropertyHarness_PredicateIgnoresResult(t *testing.T) {
	binding := core.RecBinding{
		Name:  "id",
		Value: &core.Var{CoreNode: core.CoreNode{NodeID: 1}, Name: "lambda"},
	}

	params := []EnsuresParam{
		{Name: "x", Value: &core.Lit{CoreNode: core.CoreNode{NodeID: 2}, Kind: core.IntLit, Value: int64(7)}},
	}

	// Predicate: true (constant — pathological but legal)
	predicate := makeBoolLit(true)

	result := BuildEnsuresPropertyHarness(binding, params, predicate)

	letRec := result.(*core.LetRec)
	xLet := letRec.Body.(*core.Let)
	resultLet, ok := xLet.Body.(*core.Let)
	if !ok {
		t.Fatalf("Expected inner result Let, got %T", xLet.Body)
	}
	if resultLet.Name != "result" {
		t.Errorf("Expected inner Let.Name = 'result' even when unused, got %q", resultLet.Name)
	}

	lit, ok := resultLet.Body.(*core.Lit)
	if !ok {
		t.Fatalf("Expected Lit in result Let.Body, got %T", resultLet.Body)
	}
	if v, ok := lit.Value.(bool); !ok || v != true {
		t.Errorf("Expected predicate Lit(true), got %v", lit.Value)
	}
}
