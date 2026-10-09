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
func BuildInlineTestHarness(binding core.RecBinding, tests []TestCase) (core.CoreExpr, error) {
	if len(tests) == 0 {
		// No tests - return unit
		return &core.Lit{
			CoreNode: core.CoreNode{
				NodeID: nextNodeID(),
			},
			Kind:  core.UnitLit,
			Value: nil,
		}, nil
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
			return nil, fmt.Errorf("test case %d has empty body", i)
		}

		tupleExpr, ok := testCase.Body[0].(*ast.Tuple)
		if !ok || len(tupleExpr.Elements) != 2 {
			return nil, fmt.Errorf("test case %d body is not an (input, expected) tuple", i)
		}

		// tuple.Elements[0] is input (or tuple of inputs for multi-arg)
		// tuple.Elements[1] is expected (we don't use it here - Go compares later)
		inputExpr := tupleExpr.Elements[0]

		// Build function call: functionName(input)
		// For multi-arg functions, input will be a tuple we need to pass as multiple args
		functionCall, err := buildFunctionCall(binding.Name, inputExpr)
		if err != nil {
			return nil, err
		}

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

	return harness, nil
}

// buildFunctionCall builds a Core App expression for calling a function with given argument(s).
// Handles both single-arg and multi-arg function calls.
// IMPORTANT: Core Lambda can have multiple params, so App should pass all args at once,
// not as curried applications.
func buildFunctionCall(functionName string, inputExpr ast.Expr) (core.CoreExpr, error) {
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
			var err error
			args[i], err = astExprToCore(elem)
			if err != nil {
				return nil, err
			}
		}
		return &core.App{
			CoreNode: core.CoreNode{
				NodeID: nextNodeID(),
			},
			Func: funcVar,
			Args: args,
		}, nil
	}

	// Single-arg function: f(a)
	inputCore, err := astExprToCore(inputExpr)
	if err != nil {
		return nil, err
	}
	return &core.App{
		CoreNode: core.CoreNode{
			NodeID: nextNodeID(),
		},
		Func: funcVar,
		Args: []core.CoreExpr{inputCore},
	}, nil
}

// EnsuresParam pairs a function parameter name with its generated Core value
// for one iteration of an ensures property test.
type EnsuresParam struct {
	Name  string
	Value core.CoreExpr
}

// BuildEnsuresPropertyHarness creates a synthetic Core expression for evaluating
// a single ensures-clause predicate against the function's actual return value.
//
// Given the function binding, the parameter name/value pairs for one test iteration,
// and the predicate AST, builds:
//
//	LetRec(f, λ...,
//	  Let("p1", val_1,
//	    Let("p2", val_2,
//	      ...
//	      Let("result", App(f, [p1, p2, ...]),
//	        <predicate-as-Core>    -- may reference "result", "p1", "p2", ...
//	      )))
//
// Both the special name `result` and each function parameter are in scope for
// the predicate, since ensures clauses reference both (e.g. `result > x`).
//
// Evaluates to a BoolValue: true = ensures holds for this input, false = ensures violated.
//
// Inputs:
//   - binding: Core LetRec binding for the function being verified.
//   - params: Per-parameter (name, generated value) pairs for this iteration.
//   - predicate: AST expression of the ensures predicate. Goes through astExprToCore,
//     so arithmetic ops will produce raw `*core.BinOp` and trip the evaluator on
//     `+`/`*`/etc. Use BuildEnsuresPropertyHarnessFromCore for already-lowered predicates.
//
// Output: Core expression that, when evaluated, returns true if ensures holds for these inputs.
func BuildEnsuresPropertyHarness(binding core.RecBinding, params []EnsuresParam, predicate ast.Expr) (core.CoreExpr, error) {
	pred, err := astExprToCore(predicate)
	if err != nil {
		return nil, err
	}
	return BuildEnsuresPropertyHarnessFromCore(binding, params, pred), nil
}

// BuildEnsuresPropertyHarnessFromCore is the lowered-Core variant of
// BuildEnsuresPropertyHarness. The runner passes the predicate from
// `result.Artifacts.Core.Meta[funcName].Contracts[i].Expr`, which has already been
// through OpLowering — so arithmetic ops (`+`, `*`, ...) resolve to typed dictionary
// calls and the evaluator can run them.
//
// M-DX26 Phase 5.1: This is the path the runner uses in production. The AST-based
// BuildEnsuresPropertyHarness wrapper is kept for unit tests and for any caller
// that has only an AST predicate.
func BuildEnsuresPropertyHarnessFromCore(binding core.RecBinding, params []EnsuresParam, predicateCore core.CoreExpr) core.CoreExpr {

	funcVar := &core.Var{
		CoreNode: core.CoreNode{
			NodeID: nextNodeID(),
		},
		Name: binding.Name,
	}

	// Reference each bound parameter by name in the function call.
	callArgs := make([]core.CoreExpr, len(params))
	for i, p := range params {
		callArgs[i] = &core.Var{
			CoreNode: core.CoreNode{
				NodeID: nextNodeID(),
			},
			Name: p.Name,
		}
	}

	functionCall := &core.App{
		CoreNode: core.CoreNode{
			NodeID: nextNodeID(),
		},
		Func: funcVar,
		Args: callArgs,
	}

	// Innermost: Let("result", App(f, [p1, ...]), <predicate>).
	body := core.CoreExpr(&core.Let{
		CoreNode: core.CoreNode{
			NodeID: nextNodeID(),
		},
		Name:  "result",
		Value: functionCall,
		Body:  predicateCore,
	})

	// Wrap each parameter binding from innermost outward, so generated literals
	// are bound under their AILANG-source names and visible to the predicate.
	for i := len(params) - 1; i >= 0; i-- {
		body = &core.Let{
			CoreNode: core.CoreNode{
				NodeID: nextNodeID(),
			},
			Name:  params[i].Name,
			Value: params[i].Value,
			Body:  body,
		}
	}

	return &core.LetRec{
		CoreNode: core.CoreNode{
			NodeID: nextNodeID(),
		},
		Bindings: []core.RecBinding{binding},
		Body:     body,
	}
}

// BuildRequiresPropertyHarnessFromCore creates a synthetic Core expression for
// evaluating a single requires-clause predicate with generated parameter values.
//
// `requires` runs *before* the function call, so unlike ensures we:
//   - do not bind `result` (it doesn't exist yet),
//   - do not call the function at all (the predicate is over the inputs).
//
// Builds:
//
//	Let("p1", val_1,
//	  Let("p2", val_2,
//	    ...
//	    <predicate-as-already-lowered-Core>    -- references "p1", "p2", ...
//	    ))
//
// The predicate must already be through OpLowering (typically pulled from
// `Core.Meta[funcName].Contracts[i].Expr` where Kind == RequiresKind), so any
// arithmetic in the predicate already resolves to dictionary calls.
//
// Output evaluates to a BoolValue: true = requires holds for these inputs,
// false = requires violated (so the test should be discarded — calling the
// function with these inputs is "out of contract", not a function bug).
//
// M-DX26 Phase 5.2.
func BuildRequiresPropertyHarnessFromCore(params []EnsuresParam, predicateCore core.CoreExpr) core.CoreExpr {
	body := predicateCore
	for i := len(params) - 1; i >= 0; i-- {
		body = &core.Let{
			CoreNode: core.CoreNode{
				NodeID: nextNodeID(),
			},
			Name:  params[i].Name,
			Value: params[i].Value,
			Body:  body,
		}
	}
	return body
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
func BuildClusterTestHarness(cluster *PureCluster, tests []TestCase) (core.CoreExpr, error) {
	if len(tests) == 0 {
		// No tests - return unit
		return &core.Lit{
			CoreNode: core.CoreNode{
				NodeID: nextNodeID(),
			},
			Kind:  core.UnitLit,
			Value: nil,
		}, nil
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
			return nil, fmt.Errorf("test case %d has empty body", i)
		}

		tupleExpr, ok := testCase.Body[0].(*ast.Tuple)
		if !ok || len(tupleExpr.Elements) != 2 {
			return nil, fmt.Errorf("test case %d body is not an (input, expected) tuple", i)
		}

		inputExpr := tupleExpr.Elements[0]

		// Call the function under test (not dependencies)
		functionCall, err := buildFunctionCall(cluster.FuncName, inputExpr)
		if err != nil {
			return nil, err
		}

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

	return harness, nil
}

// Global node ID counter for test harness (simple approach for now).
var nodeIDCounter uint64 = 100000 // Start high to avoid conflicts

func nextNodeID() uint64 {
	nodeIDCounter++
	return nodeIDCounter
}
