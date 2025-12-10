package golang

import (
	"strings"
	"testing"

	"github.com/sunholo/ailang/internal/core"
)

// TestMathBuiltinMapping tests that math builtins are mapped to Go math package.
// M-CODEGEN-STDLIB-MATH: Verifies PI, sin, cos, etc. generate correct Go code.
func TestMathBuiltinMapping(t *testing.T) {
	tests := []struct {
		name     string
		builtin  string
		expected string
	}{
		// Constants
		{"PI builtin", "_math_PI", "math.Pi"},
		{"PI wrapper", "PI", "math.Pi"},
		{"E builtin", "_math_E", "math.E"},
		{"E wrapper", "E", "math.E"},

		// Trig functions
		{"sin builtin", "_math_sin", "math.Sin"},
		{"sin wrapper", "sin", "math.Sin"},
		{"cos builtin", "_math_cos", "math.Cos"},
		{"cos wrapper", "cos", "math.Cos"},
		{"tan builtin", "_math_tan", "math.Tan"},
		{"tan wrapper", "tan", "math.Tan"},

		// Inverse trig
		{"asin", "_math_asin", "math.Asin"},
		{"acos", "_math_acos", "math.Acos"},
		{"atan", "_math_atan", "math.Atan"},
		{"atan2", "_math_atan2", "math.Atan2"},

		// Exponential/log
		{"exp", "_math_exp", "math.Exp"},
		{"log", "_math_log", "math.Log"},
		{"log10", "_math_log10", "math.Log10"},
		{"pow", "_math_pow", "math.Pow"},
		{"sqrt", "_math_sqrt", "math.Sqrt"},

		// Rounding
		{"ceil", "_math_ceil", "math.Ceil"},
		{"floor", "_math_floor", "math.Floor"},
		{"round", "_math_round", "math.Round"},

		// Utility
		{"abs_Float", "_math_abs_Float", "math.Abs"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := New("test")
			result := g.mapPureMathBuiltin(tt.builtin)
			if result != tt.expected {
				t.Errorf("mapPureMathBuiltin(%q) = %q, want %q", tt.builtin, result, tt.expected)
			}
			if result != "" && !g.needsMathImport {
				t.Errorf("needsMathImport should be true after mapping %q", tt.builtin)
			}
		})
	}
}

// TestMathImportGeneration tests that math import is added when needed.
func TestMathImportGeneration(t *testing.T) {
	g := New("test")

	// Create a simple program with a math constant reference
	prog := &core.Program{
		Decls: []core.CoreExpr{
			&core.Let{
				Name: "usePI",
				Value: &core.VarGlobal{
					Ref: core.GlobalRef{Module: "std/math", Name: "PI"},
				},
				Body: &core.Var{Name: "usePI"},
			},
		},
	}

	code, err := g.Generate(prog)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	codeStr := string(code)

	// Check that math import is present
	if !strings.Contains(codeStr, `"math"`) {
		t.Error("Generated code should contain math import")
	}

	// Check that PI is mapped to math.Pi
	if !strings.Contains(codeStr, "math.Pi") {
		t.Error("Generated code should contain math.Pi")
	}
}

// TestNoMathImportWhenNotNeeded tests that math import is not added when not needed.
func TestNoMathImportWhenNotNeeded(t *testing.T) {
	g := New("test")

	// Create a simple program without math functions
	prog := &core.Program{
		Decls: []core.CoreExpr{
			&core.Let{
				Name:  "simpleFunc",
				Value: &core.Lit{Kind: core.IntLit, Value: int64(42)},
				Body:  &core.Var{Name: "simpleFunc"},
			},
		},
	}

	code, err := g.Generate(prog)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	codeStr := string(code)

	// Check that math import is NOT present
	if strings.Contains(codeStr, `"math"`) {
		t.Error("Generated code should NOT contain math import when not needed")
	}
}

// TestMathFunctionCall tests that math functions are called correctly.
func TestMathFunctionCall(t *testing.T) {
	g := New("test")

	// Create a program that calls sin(x)
	prog := &core.Program{
		Decls: []core.CoreExpr{
			&core.Let{
				Name: "computeSin",
				Value: &core.Lambda{
					Params: []string{"x"},
					Body: &core.App{
						Func: &core.VarGlobal{
							Ref: core.GlobalRef{Module: "std/math", Name: "sin"},
						},
						Args: []core.CoreExpr{&core.Var{Name: "x"}},
					},
				},
				Body: &core.Var{Name: "computeSin"},
			},
		},
	}

	code, err := g.Generate(prog)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	codeStr := string(code)

	// Check that math.Sin is used
	if !strings.Contains(codeStr, "math.Sin") {
		t.Error("Generated code should contain math.Sin for sin() call")
	}

	// Check that math import is present
	if !strings.Contains(codeStr, `"math"`) {
		t.Error("Generated code should contain math import")
	}
}

// TestMathNonBuiltin tests that non-math builtins don't trigger math import.
func TestMathNonBuiltin(t *testing.T) {
	g := New("test")
	result := g.mapPureMathBuiltin("some_random_func")
	if result != "" {
		t.Errorf("mapPureMathBuiltin should return empty for non-math builtin, got %q", result)
	}
	if g.needsMathImport {
		t.Error("needsMathImport should be false for non-math builtin")
	}
}

// TestMathImportWithSkipRuntimeHelpers tests that math import is added even when
// skipRuntimeHelpers is true (multi-file compilation mode).
// M-CODEGEN-STDLIB-MATH: Bug #26 followup fix.
func TestMathImportWithSkipRuntimeHelpers(t *testing.T) {
	g := New("test")
	g.SetSkipRuntimeHelpers(true) // Multi-file mode

	// Create a program that uses PI
	prog := &core.Program{
		Decls: []core.CoreExpr{
			&core.Let{
				Name: "usePI",
				Value: &core.VarGlobal{
					Ref: core.GlobalRef{Module: "std/math", Name: "PI"},
				},
				Body: &core.Var{Name: "usePI"},
			},
		},
	}

	code, err := g.Generate(prog)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	codeStr := string(code)

	// Check that math import is present even with skipRuntimeHelpers=true
	if !strings.Contains(codeStr, `"math"`) {
		t.Error("Generated code should contain math import even with skipRuntimeHelpers=true")
		t.Logf("Generated code:\n%s", codeStr)
	}

	// Check that math.Pi is used
	if !strings.Contains(codeStr, "math.Pi") {
		t.Error("Generated code should contain math.Pi")
	}

	// Check that other runtime imports (fmt, reflect, strings) are NOT present
	if strings.Contains(codeStr, `"fmt"`) {
		t.Error("Generated code should NOT contain fmt import when skipRuntimeHelpers=true")
	}
	if strings.Contains(codeStr, `"reflect"`) {
		t.Error("Generated code should NOT contain reflect import when skipRuntimeHelpers=true")
	}
}
