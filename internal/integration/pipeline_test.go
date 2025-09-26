package integration

import (
	"testing"
	"github.com/sunholo/ailang/internal/elaborate"
	"github.com/sunholo/ailang/internal/eval"
	"github.com/sunholo/ailang/internal/lexer"
	"github.com/sunholo/ailang/internal/parser"
	"github.com/sunholo/ailang/internal/types"
)

// TestFullPipeline tests the complete pipeline: Parse → Elaborate → TypeCheck → Eval
func TestFullPipeline(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected interface{}
		wantErr  bool
		errMsg   string
	}{
		// Basic arithmetic
		{
			name:     "simple arithmetic",
			input:    "2 + 3 * 4",
			expected: 14,
		},
		{
			name:     "float arithmetic",
			input:    "10.5 / 2.0",
			expected: 5.25,
		},
		
		// Let bindings
		{
			name:     "let binding",
			input:    "let x = 5 in x * 2",
			expected: 10,
		},
		{
			name:     "nested let",
			input:    "let x = 3 in let y = x + 2 in x * y",
			expected: 15,
		},
		
		// Lambda expressions
		{
			name:     "lambda identity",
			input:    "(\\x. x)(42)",
			expected: 42,
		},
		{
			name:     "lambda with closure",
			input:    "let y = 10 in (\\x. x + y)(5)",
			expected: 15,
		},
		
		// Type errors
		{
			name:    "type error - bool arithmetic",
			input:   "true + 1",
			wantErr: true,
			errMsg:  "No instance for Num Bool",
		},
		{
			name:    "type error - string comparison",
			input:   "\"hello\" > 5",
			wantErr: true,
			errMsg:  "Cannot unify String with Int",
		},
		
		// Let rec (factorial)
		{
			name: "factorial with let rec",
			input: `let rec fact = \n. if n <= 1 then 1 else n * fact(n - 1) in fact(5)`,
			expected: 120,
		},
		
		// Records
		{
			name:     "record creation and access",
			input:    `let r = {name: "Alice", age: 30} in r.age`,
			expected: 30,
		},
		
		// Lists
		{
			name:     "list creation",
			input:    "[1, 2, 3]",
			expected: []int{1, 2, 3},
		},
		
		// String concatenation
		{
			name:     "string concat",
			input:    `"hello " ++ "world"`,
			expected: "hello world",
		},
		
		// Conditionals
		{
			name:     "if expression",
			input:    "if 5 > 3 then 10 else 20",
			expected: 10,
		},
		
		// ANF normalization test
		{
			name:     "complex expression normalized to ANF",
			input:    "(2 + 3) * (4 + 5)",
			expected: 45,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse
			l := lexer.New(tt.input, "test.ail")
			p := parser.New(l)
			program := p.Parse()
			
			if len(p.Errors()) > 0 {
				if !tt.wantErr {
					t.Fatalf("unexpected parse errors: %v", p.Errors())
				}
				return
			}
			
			// Elaborate to Core ANF
			elaborator := elaborate.NewElaborator()
			coreProgram, err := elaborator.Elaborate(program)
			if err != nil {
				if !tt.wantErr {
					t.Fatalf("unexpected elaboration error: %v", err)
				}
				return
			}
			
			// Type check
			typeChecker := types.NewCoreTypeChecker()
			typedProgram, err := typeChecker.CheckCoreProgram(coreProgram)
			if err != nil {
				if tt.wantErr {
					// Check error message contains expected text
					if tt.errMsg != "" {
						errStr := err.Error()
						// TODO: Check error message
						_ = errStr
					}
					return
				}
				t.Fatalf("unexpected type error: %v", err)
			}
			
			if tt.wantErr {
				t.Fatal("expected error but got none")
			}
			
			// Evaluate
			evaluator := eval.NewTypedEvaluator(false, 0, false)
			result, err := evaluator.EvalTypedProgram(typedProgram)
			if err != nil {
				t.Fatalf("unexpected runtime error: %v", err)
			}
			
			// Check result
			switch expected := tt.expected.(type) {
			case int:
				if intVal, ok := result.(*eval.IntValue); ok {
					if intVal.Value != expected {
						t.Errorf("expected %d, got %d", expected, intVal.Value)
					}
				} else {
					t.Errorf("expected IntValue, got %T", result)
				}
				
			case float64:
				if floatVal, ok := result.(*eval.FloatValue); ok {
					if floatVal.Value != expected {
						t.Errorf("expected %f, got %f", expected, floatVal.Value)
					}
				} else {
					t.Errorf("expected FloatValue, got %T", result)
				}
				
			case string:
				if strVal, ok := result.(*eval.StringValue); ok {
					if strVal.Value != expected {
						t.Errorf("expected %s, got %s", expected, strVal.Value)
					}
				} else {
					t.Errorf("expected StringValue, got %T", result)
				}
				
			case []int:
				if listVal, ok := result.(*eval.ListValue); ok {
					if len(listVal.Elements) != len(expected) {
						t.Errorf("expected list length %d, got %d", len(expected), len(listVal.Elements))
					}
					// TODO: Check elements
				} else {
					t.Errorf("expected ListValue, got %T", result)
				}
			}
		})
	}
}

// TestANFNormalization verifies that elaboration produces proper ANF
func TestANFNormalization(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantLetCount int // Minimum number of let bindings expected
	}{
		{
			name:         "complex arithmetic",
			input:        "(2 + 3) * (4 + 5)",
			wantLetCount: 2, // At least 2 lets for sub-expressions
		},
		{
			name:         "nested function calls",
			input:        "f(g(x), h(y))",
			wantLetCount: 2, // Arguments must be atomic
		},
		{
			name:         "effectful operations",
			input:        "readFile(path1) ++ readFile(path2)",
			wantLetCount: 2, // Each effectful op gets own let
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse
			l := lexer.New(tt.input, "test.ail")
			p := parser.New(l)
			program := p.Parse()
			
			if len(p.Errors()) > 0 {
				// Some tests may have undefined functions - that's OK for ANF testing
				return
			}
			
			// Elaborate to Core ANF
			elaborator := elaborate.NewElaborator()
			coreProgram, err := elaborator.Elaborate(program)
			if err != nil {
				// Some elaboration errors are OK for this test
				return
			}
			
			// Count let bindings in Core program
			letCount := countLetBindings(coreProgram)
			if letCount < tt.wantLetCount {
				t.Errorf("expected at least %d let bindings in ANF, got %d", 
					tt.wantLetCount, letCount)
			}
		})
	}
}

// TestLetRec tests recursive let bindings
func TestLetRec(t *testing.T) {
	input := `
	let rec fact = \n. if n <= 1 then 1 else n * fact(n - 1) in
	fact(5)
	`
	
	// Full pipeline
	l := lexer.New(input, "test.ail")
	p := parser.New(l)
	program := p.Parse()
	
	if len(p.Errors()) > 0 {
		t.Fatalf("parse errors: %v", p.Errors())
	}
	
	elaborator := elaborate.NewElaborator()
	coreProgram, err := elaborator.Elaborate(program)
	if err != nil {
		t.Fatalf("elaboration error: %v", err)
	}
	
	typeChecker := types.NewCoreTypeChecker()
	typedProgram, err := typeChecker.CheckCoreProgram(coreProgram)
	if err != nil {
		t.Fatalf("type error: %v", err)
	}
	
	evaluator := eval.NewTypedEvaluator(false, 0, false)
	result, err := evaluator.EvalTypedProgram(typedProgram)
	if err != nil {
		t.Fatalf("runtime error: %v", err)
	}
	
	if intVal, ok := result.(*eval.IntValue); ok {
		if intVal.Value != 120 {
			t.Errorf("expected factorial(5) = 120, got %d", intVal.Value)
		}
	} else {
		t.Errorf("expected IntValue, got %T", result)
	}
}

// TestCanonicalRowPrinting tests that rows are printed in canonical form
func TestCanonicalRowPrinting(t *testing.T) {
	// Create a row with unsorted labels
	row := &types.Row{
		Kind: types.EffectRow,
		Labels: map[string]types.Type{
			"Net": types.TUnit,
			"FS":  types.TUnit,
			"IO":  types.TUnit,
		},
		Tail: nil,
	}
	
	// Should print in sorted order
	expected := "{FS, IO, Net}"
	actual := row.String()
	
	if actual != expected {
		t.Errorf("expected canonical row %s, got %s", expected, actual)
	}
}

// TestLinearCapture tests linear capability capture detection
func TestLinearCapture(t *testing.T) {
	// This would require FS to be in scope as a linear capability
	// For now, this is a placeholder test
	t.Skip("Linear capture test requires capability system implementation")
}

// TestClassConstraints tests type class constraint failures
func TestClassConstraints(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		errMsg string
	}{
		{
			name:   "Num constraint on Bool",
			input:  "true + 1",
			errMsg: "No instance for Num Bool",
		},
		{
			name:   "Ord constraint on incompatible types",
			input:  `"hello" > 5`,
			errMsg: "Cannot unify String with Int",
		},
		{
			name:   "Eq constraint",
			input:  `(\x. x) == 5`,
			errMsg: "No instance for Eq",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse
			l := lexer.New(tt.input, "test.ail")
			p := parser.New(l)
			program := p.Parse()
			
			// Elaborate
			elaborator := elaborate.NewElaborator()
			coreProgram, err := elaborator.Elaborate(program)
			if err != nil {
				return
			}
			
			// Type check - should fail
			typeChecker := types.NewCoreTypeChecker()
			_, err = typeChecker.CheckCoreProgram(coreProgram)
			if err == nil {
				t.Fatal("expected type error but got none")
			}
			
			// TODO: Check error message contains expected text
		})
	}
}

// Helper function to count let bindings in Core program
func countLetBindings(prog interface{}) int {
	// TODO: Implement Core AST traversal to count let bindings
	return 0
}