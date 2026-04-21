package testing

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/core"
)

// BuildInlineTestHarness creates a synthetic Core expression for evaluating inline tests.
//
// Given a function binding and its test cases, builds a Core expression of the form:
//
//	LetRec("f", λ_f,
//	  Let("_test_1", App(f, arg_1),
//	    Let("_test_2", App(f, arg_2),
//	      Tuple([_test_1, _test_2])
//	    )
//	  )
//	)
//
// The function `f` is in scope for all test calls via the LetRec body.
// Returns a tuple of actual results for the test runner to compare against expected values.
//
// Input:
//   - binding: Core LetRec binding for the function being tested
//   - tests: List of test cases (each contains input/expected tuple expressions)
//
// Output: Core expression that evaluates all tests and returns tuple of actuals
func BuildInlineTestHarness(binding core.RecBinding, tests []TestCase) core.CoreExpr {
	if len(tests) == 0 {
		// No tests - return unit
		return &core.Lit{
			CoreNode: core.CoreNode{
				NodeID: nextNodeID(),
			},
			Kind:  core.UnitLit,
			Value: nil,
		}
	}

	// Build nested Lets for test evaluation
	// Start from the innermost (tuple of results) and work outward
	testVarNames := make([]string, len(tests))
	for i := range tests {
		testVarNames[i] = fmt.Sprintf("_test_%d", i+1)
	}

	// Innermost: Tuple of test result variables
	tupleElements := make([]core.CoreExpr, len(tests))
	for i, varName := range testVarNames {
		tupleElements[i] = &core.Var{
			CoreNode: core.CoreNode{
				NodeID: nextNodeID(),
			},
			Name: varName,
		}
	}

	var body core.CoreExpr = &core.Tuple{
		CoreNode: core.CoreNode{
			NodeID: nextNodeID(),
		},
		Elements: tupleElements,
	}

	// Build nested Lets from innermost outward (reverse order)
	for i := len(tests) - 1; i >= 0; i-- {
		testCase := tests[i]
		varName := testVarNames[i]

		// Extract (input, expected) tuple from test body
		// testCase.Body is []ast.Expr with one element: the tuple
		if len(testCase.Body) == 0 {
			panic(fmt.Sprintf("test case %d has empty body", i))
		}

		tupleExpr, ok := testCase.Body[0].(*ast.Tuple)
		if !ok {
			panic(fmt.Sprintf("test case %d body is not a tuple", i))
		}

		// tuple.Elements[0] is input (or tuple of inputs for multi-arg)
		// tuple.Elements[1] is expected (we don't use it here - Go compares later)
		inputExpr := tupleExpr.Elements[0]

		// Build function call: functionName(input)
		// For multi-arg functions, input will be a tuple we need to pass as multiple args
		functionCall := buildFunctionCall(binding.Name, inputExpr)

		// Wrap in Let binding
		body = &core.Let{
			CoreNode: core.CoreNode{
				NodeID: nextNodeID(),
			},
			Name:  varName,
			Value: functionCall,
			Body:  body,
		}
	}

	// Outermost: LetRec binding for the function
	harness := &core.LetRec{
		CoreNode: core.CoreNode{
			NodeID: nextNodeID(),
		},
		Bindings: []core.RecBinding{binding},
		Body:     body,
	}

	return harness
}

// buildFunctionCall builds a Core App expression for calling a function with given argument(s).
// Handles both single-arg and multi-arg function calls.
// IMPORTANT: Core Lambda can have multiple params, so App should pass all args at once,
// not as curried applications.
func buildFunctionCall(functionName string, inputExpr ast.Expr) core.CoreExpr {
	funcVar := &core.Var{
		CoreNode: core.CoreNode{
			NodeID: nextNodeID(),
		},
		Name: functionName,
	}

	// Check if input is a tuple (multi-arg function)
	if tuple, ok := inputExpr.(*ast.Tuple); ok && len(tuple.Elements) > 1 {
		// Multi-arg function: f(a, b, c) becomes App(f, [a, b, c])
		// Convert all tuple elements to Core
		args := make([]core.CoreExpr, len(tuple.Elements))
		for i, elem := range tuple.Elements {
			args[i] = astExprToCore(elem)
		}
		return &core.App{
			CoreNode: core.CoreNode{
				NodeID: nextNodeID(),
			},
			Func: funcVar,
			Args: args,
		}
	}

	// Single-arg function: f(a)
	inputCore := astExprToCore(inputExpr)
	return &core.App{
		CoreNode: core.CoreNode{
			NodeID: nextNodeID(),
		},
		Func: funcVar,
		Args: []core.CoreExpr{inputCore},
	}
}

// astExprToCore converts a simple AST expression to Core (literals and composite types).
// This is a minimal converter for test harness construction.
func astExprToCore(expr ast.Expr) core.CoreExpr {
	switch e := expr.(type) {
	case *ast.Literal:
		return &core.Lit{
			CoreNode: core.CoreNode{
				NodeID: nextNodeID(),
			},
			Kind:  astLitKindToCore(e.Kind),
			Value: e.Value,
		}
	case *ast.Tuple:
		// Convert tuple elements
		elements := make([]core.CoreExpr, len(e.Elements))
		for i, elem := range e.Elements {
			elements[i] = astExprToCore(elem)
		}
		return &core.Tuple{
			CoreNode: core.CoreNode{
				NodeID: nextNodeID(),
			},
			Elements: elements,
		}
	case *ast.List:
		// Convert list elements
		elements := make([]core.CoreExpr, len(e.Elements))
		for i, elem := range e.Elements {
			elements[i] = astExprToCore(elem)
		}
		return &core.List{
			CoreNode: core.CoreNode{
				NodeID: nextNodeID(),
			},
			Elements: elements,
		}
	case *ast.UnaryOp:
		// Handle unary operations like -3
		operand := astExprToCore(e.Expr)
		return &core.UnOp{
			CoreNode: core.CoreNode{
				NodeID: nextNodeID(),
			},
			Op:      e.Op,
			Operand: operand,
		}
	case *ast.Identifier:
		// Handle identifiers (including ADT constructors like None, True, False)
		return &core.Var{
			CoreNode: core.CoreNode{
				NodeID: nextNodeID(),
			},
			Name: e.Name,
		}
	case *ast.FuncCall:
		// Handle function/constructor application like Some(5), Pair(1, 2)
		fn := astExprToCore(e.Func)
		args := make([]core.CoreExpr, len(e.Args))
		for i, arg := range e.Args {
			args[i] = astExprToCore(arg)
		}
		return &core.App{
			CoreNode: core.CoreNode{
				NodeID: nextNodeID(),
			},
			Func: fn,
			Args: args,
		}
	case *ast.Record:
		// Handle record literals like {x: 1, y: 2}
		fields := make(map[string]core.CoreExpr)
		for _, field := range e.Fields {
			fields[field.Name] = astExprToCore(field.Value)
		}
		return &core.Record{
			CoreNode: core.CoreNode{
				NodeID: nextNodeID(),
			},
			Fields: fields,
		}
	default:
		panic(fmt.Sprintf("unsupported AST expression type in test harness: %T", expr))
	}
}

// astLitKindToCore converts AST literal kind to Core literal kind.
func astLitKindToCore(kind ast.LiteralKind) core.LitKind {
	switch kind {
	case ast.IntLit:
		return core.IntLit
	case ast.FloatLit:
		return core.FloatLit
	case ast.BoolLit:
		return core.BoolLit
	case ast.StringLit:
		return core.StringLit
	case ast.UnitLit:
		return core.UnitLit
	default:
		panic(fmt.Sprintf("unsupported literal kind: %v", kind))
	}
}

// BuildClusterTestHarness creates a synthetic Core expression for testing a function
// with cross-function dependencies.
//
// Given a pure cluster (function under test + all dependencies) and test cases, builds:
//
//	LetRec([f, g1, g2, ...],
//	  Let("_test_1", App(f, arg_1),
//	    Let("_test_2", App(f, arg_2),
//	      Tuple([_test_1, _test_2])
//	    )
//	  )
//	)
//
// All cluster bindings are in scope via the shared LetRec.
func BuildClusterTestHarness(cluster *PureCluster, tests []TestCase) core.CoreExpr {
	if len(tests) == 0 {
		// No tests - return unit
		return &core.Lit{
			CoreNode: core.CoreNode{
				NodeID: nextNodeID(),
			},
			Kind:  core.UnitLit,
			Value: nil,
		}
	}

	// Build nested Lets for test evaluation (same as single-binding version)
	testVarNames := make([]string, len(tests))
	for i := range tests {
		testVarNames[i] = fmt.Sprintf("_test_%d", i+1)
	}

	// Innermost: Tuple of test result variables
	tupleElements := make([]core.CoreExpr, len(tests))
	for i, varName := range testVarNames {
		tupleElements[i] = &core.Var{
			CoreNode: core.CoreNode{
				NodeID: nextNodeID(),
			},
			Name: varName,
		}
	}

	var body core.CoreExpr = &core.Tuple{
		CoreNode: core.CoreNode{
			NodeID: nextNodeID(),
		},
		Elements: tupleElements,
	}

	// Build nested Lets from innermost outward (reverse order)
	for i := len(tests) - 1; i >= 0; i-- {
		testCase := tests[i]
		varName := testVarNames[i]

		if len(testCase.Body) == 0 {
			panic(fmt.Sprintf("test case %d has empty body", i))
		}

		tupleExpr, ok := testCase.Body[0].(*ast.Tuple)
		if !ok {
			panic(fmt.Sprintf("test case %d body is not a tuple", i))
		}

		inputExpr := tupleExpr.Elements[0]

		// Call the function under test (not dependencies)
		functionCall := buildFunctionCall(cluster.FuncName, inputExpr)

		body = &core.Let{
			CoreNode: core.CoreNode{
				NodeID: nextNodeID(),
			},
			Name:  varName,
			Value: functionCall,
			Body:  body,
		}
	}

	// Outermost: LetRec with ALL cluster bindings
	harness := &core.LetRec{
		CoreNode: core.CoreNode{
			NodeID: nextNodeID(),
		},
		Bindings: cluster.Bindings,
		Body:     body,
	}

	return harness
}

// Global node ID counter for test harness (simple approach for now).
var nodeIDCounter uint64 = 100000 // Start high to avoid conflicts

func nextNodeID() uint64 {
	nodeIDCounter++
	return nodeIDCounter
}
