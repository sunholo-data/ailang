package eval

import (
	"testing"

	"github.com/sunholo/ailang/internal/core"
)

// TestSimpleRecursion_Factorial tests self-recursion with factorial
func TestSimpleRecursion_Factorial(t *testing.T) {
	// Build: letrec fac = λn. if n <= 1 then 1 else n * fac(n-1) in fac(5)
	// Expected: 120

	// Lambda body: if n <= 1 then 1 else n * fac(n-1)
	facBody := &core.If{
		Cond: &core.BinOp{
			Op:   "<=",
			Left: &core.Var{Name: "n"},
			Right: &core.Lit{
				Kind:  core.IntLit,
				Value: 1,
			},
		},
		Then: &core.Lit{Kind: core.IntLit, Value: 1},
		Else: &core.BinOp{
			Op:   "*",
			Left: &core.Var{Name: "n"},
			Right: &core.App{
				Func: &core.Var{Name: "fac"},
				Args: []core.CoreExpr{
					&core.BinOp{
						Op:    "-",
						Left:  &core.Var{Name: "n"},
						Right: &core.Lit{Kind: core.IntLit, Value: 1},
					},
				},
			},
		},
	}

	// fac = λn. <body>
	facLambda := &core.Lambda{
		Params: []string{"n"},
		Body:   facBody,
	}

	// letrec fac = <lambda> in fac(5)
	letrec := &core.LetRec{
		Bindings: []core.RecBinding{
			{Name: "fac", Value: facLambda},
		},
		Body: &core.App{
			Func: &core.Var{Name: "fac"},
			Args: []core.CoreExpr{
				&core.Lit{Kind: core.IntLit, Value: 5},
			},
		},
	}

	evaluator := NewCoreEvaluator()
	evaluator.SetExperimentalBinopShim(true) // Enable operator shim for tests
	result, err := evaluator.evalCore(letrec)

	if err != nil {
		t.Fatalf("Expected factorial to succeed, got error: %v", err)
	}

	intVal, ok := result.(*IntValue)
	if !ok {
		t.Fatalf("Expected IntValue, got %T: %v", result, result)
	}

	if intVal.Value != 120 {
		t.Errorf("Expected factorial(5) = 120, got %d", intVal.Value)
	}
}

// TestSimpleRecursion_Fibonacci tests non-tail recursion with fibonacci
func TestSimpleRecursion_Fibonacci(t *testing.T) {
	// Build: letrec fib = λn. if n <= 1 then n else fib(n-1) + fib(n-2) in fib(10)
	// Expected: 55

	// Lambda body: if n <= 1 then n else fib(n-1) + fib(n-2)
	fibBody := &core.If{
		Cond: &core.BinOp{
			Op:   "<=",
			Left: &core.Var{Name: "n"},
			Right: &core.Lit{
				Kind:  core.IntLit,
				Value: 1,
			},
		},
		Then: &core.Var{Name: "n"},
		Else: &core.BinOp{
			Op: "+",
			Left: &core.App{
				Func: &core.Var{Name: "fib"},
				Args: []core.CoreExpr{
					&core.BinOp{
						Op:    "-",
						Left:  &core.Var{Name: "n"},
						Right: &core.Lit{Kind: core.IntLit, Value: 1},
					},
				},
			},
			Right: &core.App{
				Func: &core.Var{Name: "fib"},
				Args: []core.CoreExpr{
					&core.BinOp{
						Op:    "-",
						Left:  &core.Var{Name: "n"},
						Right: &core.Lit{Kind: core.IntLit, Value: 2},
					},
				},
			},
		},
	}

	// fib = λn. <body>
	fibLambda := &core.Lambda{
		Params: []string{"n"},
		Body:   fibBody,
	}

	// letrec fib = <lambda> in fib(10)
	letrec := &core.LetRec{
		Bindings: []core.RecBinding{
			{Name: "fib", Value: fibLambda},
		},
		Body: &core.App{
			Func: &core.Var{Name: "fib"},
			Args: []core.CoreExpr{
				&core.Lit{Kind: core.IntLit, Value: 10},
			},
		},
	}

	evaluator := NewCoreEvaluator()
	evaluator.SetExperimentalBinopShim(true) // Enable operator shim for tests
	result, err := evaluator.evalCore(letrec)

	if err != nil {
		t.Fatalf("Expected fibonacci to succeed, got error: %v", err)
	}

	intVal, ok := result.(*IntValue)
	if !ok {
		t.Fatalf("Expected IntValue, got %T: %v", result, result)
	}

	if intVal.Value != 55 {
		t.Errorf("Expected fib(10) = 55, got %d", intVal.Value)
	}
}

// TestRecursiveValueError tests that non-function recursive values error correctly
func TestRecursiveValueError(t *testing.T) {
	// Build: letrec x = x in x
	// Expected: RT_REC_001 error (recursive value used before initialization)

	letrec := &core.LetRec{
		Bindings: []core.RecBinding{
			{Name: "x", Value: &core.Var{Name: "x"}},
		},
		Body: &core.Var{Name: "x"},
	}

	evaluator := NewCoreEvaluator()
	_, err := evaluator.evalCore(letrec)

	if err == nil {
		t.Fatal("Expected error for 'letrec x = x in x', got nil")
	}

	// Check for RT_REC_001 error code
	expectedErrSubstring := "RT_REC_001"
	if !contains(err.Error(), expectedErrSubstring) {
		t.Errorf("Expected error containing '%s', got: %v", expectedErrSubstring, err)
	}
}

// TestMutualRecursion_IsEvenOdd tests mutual recursion with isEven/isOdd
func TestMutualRecursion_IsEvenOdd(t *testing.T) {
	// Build: letrec
	//   isEven = λn. if n == 0 then true else isOdd(n-1),
	//   isOdd = λn. if n == 0 then false else isEven(n-1)
	// in isEven(42)
	// Expected: true

	// isEven body: if n == 0 then true else isOdd(n-1)
	isEvenBody := &core.If{
		Cond: &core.BinOp{
			Op:    "==",
			Left:  &core.Var{Name: "n"},
			Right: &core.Lit{Kind: core.IntLit, Value: 0},
		},
		Then: &core.Lit{Kind: core.BoolLit, Value: true},
		Else: &core.App{
			Func: &core.Var{Name: "isOdd"},
			Args: []core.CoreExpr{
				&core.BinOp{
					Op:    "-",
					Left:  &core.Var{Name: "n"},
					Right: &core.Lit{Kind: core.IntLit, Value: 1},
				},
			},
		},
	}

	// isOdd body: if n == 0 then false else isEven(n-1)
	isOddBody := &core.If{
		Cond: &core.BinOp{
			Op:    "==",
			Left:  &core.Var{Name: "n"},
			Right: &core.Lit{Kind: core.IntLit, Value: 0},
		},
		Then: &core.Lit{Kind: core.BoolLit, Value: false},
		Else: &core.App{
			Func: &core.Var{Name: "isEven"},
			Args: []core.CoreExpr{
				&core.BinOp{
					Op:    "-",
					Left:  &core.Var{Name: "n"},
					Right: &core.Lit{Kind: core.IntLit, Value: 1},
				},
			},
		},
	}

	// letrec isEven = ..., isOdd = ... in isEven(42)
	letrec := &core.LetRec{
		Bindings: []core.RecBinding{
			{Name: "isEven", Value: &core.Lambda{Params: []string{"n"}, Body: isEvenBody}},
			{Name: "isOdd", Value: &core.Lambda{Params: []string{"n"}, Body: isOddBody}},
		},
		Body: &core.App{
			Func: &core.Var{Name: "isEven"},
			Args: []core.CoreExpr{
				&core.Lit{Kind: core.IntLit, Value: 42},
			},
		},
	}

	evaluator := NewCoreEvaluator()
	evaluator.SetExperimentalBinopShim(true)
	result, err := evaluator.evalCore(letrec)

	if err != nil {
		t.Fatalf("Expected mutual recursion to succeed, got error: %v", err)
	}

	boolVal, ok := result.(*BoolValue)
	if !ok {
		t.Fatalf("Expected BoolValue, got %T: %v", result, result)
	}

	if !boolVal.Value {
		t.Errorf("Expected isEven(42) = true, got false")
	}
}

// TestStackOverflow tests that infinite recursion triggers depth guard
func TestStackOverflow(t *testing.T) {
	// Build: letrec loop = λn. loop(n+1) in loop(0)
	// Expected: RT_REC_003 error (max recursion depth exceeded)

	// loop body: loop(n+1)
	loopBody := &core.App{
		Func: &core.Var{Name: "loop"},
		Args: []core.CoreExpr{
			&core.BinOp{
				Op:    "+",
				Left:  &core.Var{Name: "n"},
				Right: &core.Lit{Kind: core.IntLit, Value: 1},
			},
		},
	}

	// letrec loop = λn. loop(n+1) in loop(0)
	letrec := &core.LetRec{
		Bindings: []core.RecBinding{
			{Name: "loop", Value: &core.Lambda{Params: []string{"n"}, Body: loopBody}},
		},
		Body: &core.App{
			Func: &core.Var{Name: "loop"},
			Args: []core.CoreExpr{
				&core.Lit{Kind: core.IntLit, Value: 0},
			},
		},
	}

	evaluator := NewCoreEvaluator()
	evaluator.SetExperimentalBinopShim(true)
	evaluator.SetMaxRecursionDepth(100) // Set low limit for faster test
	_, err := evaluator.evalCore(letrec)

	if err == nil {
		t.Fatal("Expected stack overflow error, got nil")
	}

	// Check for RT_REC_003 error code
	expectedErrSubstring := "RT_REC_003"
	if !contains(err.Error(), expectedErrSubstring) {
		t.Errorf("Expected error containing '%s', got: %v", expectedErrSubstring, err)
	}
}

// TestDeepRecursion tests that reasonably deep recursion works
func TestDeepRecursion(t *testing.T) {
	// Build: letrec sum = λn. if n <= 0 then 0 else n + sum(n-1) in sum(500)
	// With depth limit 1000, this should succeed
	// Expected: 125250

	sumBody := &core.If{
		Cond: &core.BinOp{
			Op:    "<=",
			Left:  &core.Var{Name: "n"},
			Right: &core.Lit{Kind: core.IntLit, Value: 0},
		},
		Then: &core.Lit{Kind: core.IntLit, Value: 0},
		Else: &core.BinOp{
			Op:   "+",
			Left: &core.Var{Name: "n"},
			Right: &core.App{
				Func: &core.Var{Name: "sum"},
				Args: []core.CoreExpr{
					&core.BinOp{
						Op:    "-",
						Left:  &core.Var{Name: "n"},
						Right: &core.Lit{Kind: core.IntLit, Value: 1},
					},
				},
			},
		},
	}

	letrec := &core.LetRec{
		Bindings: []core.RecBinding{
			{Name: "sum", Value: &core.Lambda{Params: []string{"n"}, Body: sumBody}},
		},
		Body: &core.App{
			Func: &core.Var{Name: "sum"},
			Args: []core.CoreExpr{
				&core.Lit{Kind: core.IntLit, Value: 500},
			},
		},
	}

	evaluator := NewCoreEvaluator()
	evaluator.SetExperimentalBinopShim(true)
	evaluator.SetMaxRecursionDepth(1000) // Should be enough for sum(500)
	result, err := evaluator.evalCore(letrec)

	if err != nil {
		t.Fatalf("Expected deep recursion to succeed, got error: %v", err)
	}

	intVal, ok := result.(*IntValue)
	if !ok {
		t.Fatalf("Expected IntValue, got %T: %v", result, result)
	}

	expected := 500 * 501 / 2 // Sum formula: n*(n+1)/2
	if intVal.Value != expected {
		t.Errorf("Expected sum(500) = %d, got %d", expected, intVal.Value)
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
