package golang

import (
	"strings"
	"testing"

	"github.com/sunholo/ailang/internal/builtins"
	"github.com/sunholo/ailang/internal/core"
)

// TestMathBuiltinRegistryMapping tests that math builtins have registry specs.
// M-CODEGEN-SUSTAINABILITY: Verifies all math builtins are in the registry.
func TestMathBuiltinRegistryMapping(t *testing.T) {
	tests := []struct {
		name        string
		builtin     string
		containsStr string // substring that must appear in the Inline spec
	}{
		// Constants
		{"PI builtin", "_math_PI", "math.Pi"},
		{"E builtin", "_math_E", "math.E"},

		// Trig functions
		{"sin builtin", "_math_sin", "math.Sin"},
		{"cos builtin", "_math_cos", "math.Cos"},
		{"tan builtin", "_math_tan", "math.Tan"},

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
			spec := builtins.GetCodegenSpec(tt.builtin)
			if spec == nil {
				t.Fatalf("no codegen spec found for %q", tt.builtin)
			}
			if !strings.Contains(spec.Inline, tt.containsStr) {
				t.Errorf("registry spec for %q: Inline = %q, want it to contain %q", tt.builtin, spec.Inline, tt.containsStr)
			}
		})
	}
}

// TestMathStdlibNameResolution tests that stdlib names (sin, cos, PI) resolve via registry.
func TestMathStdlibNameResolution(t *testing.T) {
	stdlibNames := []string{"sin", "cos", "tan", "asin", "acos", "atan", "atan2",
		"exp", "log", "log10", "pow", "sqrt", "ceil", "floor", "round"}

	for _, name := range stdlibNames {
		t.Run(name, func(t *testing.T) {
			g := New("test")
			result := g.resolveBuiltinViaRegistry(name)
			if result == "" {
				t.Errorf("resolveBuiltinViaRegistry(%q) returned empty — not in registry", name)
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

// TestMathNonBuiltin tests that non-math builtins don't resolve via registry.
func TestMathNonBuiltin(t *testing.T) {
	g := New("test")
	result := g.resolveBuiltinViaRegistry("some_random_func")
	if result != "" {
		t.Errorf("resolveBuiltinViaRegistry should return empty for non-math builtin, got %q", result)
	}
	if g.needsMathImport {
		t.Error("needsMathImport should be false for non-math builtin")
	}
}

// TestMathConstantNotCalledAsFunction tests that math constants (PI, E) are not
// emitted with () when called as zero-arg functions in AILANG.
// M-CODEGEN-STDLIB-MATH: Bug #27 fix - PI() should generate math.Pi, not math.Pi().
func TestMathConstantNotCalledAsFunction(t *testing.T) {
	g := New("test")

	// Create a program that calls PI() - which should generate math.Pi (no parens)
	prog := &core.Program{
		Decls: []core.CoreExpr{
			&core.Let{
				Name: "area",
				Value: &core.App{
					Func: &core.VarGlobal{
						Ref: core.GlobalRef{Module: "std/math", Name: "PI"},
					},
					Args: []core.CoreExpr{}, // Zero args - AILANG calls PI as PI()
				},
				Body: &core.Var{Name: "area"},
			},
		},
	}

	code, err := g.Generate(prog)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	codeStr := string(code)

	// Should contain math.Pi (constant)
	if !strings.Contains(codeStr, "math.Pi") {
		t.Error("Generated code should contain math.Pi")
		t.Logf("Generated code:\n%s", codeStr)
	}

	// Should NOT contain math.Pi() (function call)
	if strings.Contains(codeStr, "math.Pi()") {
		t.Error("Generated code should NOT contain math.Pi() - PI is a constant, not a function")
		t.Logf("Generated code:\n%s", codeStr)
	}
}

// TestMathFunctionWithTypeAssertion tests that math functions have type assertions
// added for interface{} arguments.
// M-CODEGEN-STDLIB-MATH: Bug #27 fix - sin(x) should generate math.Sin(x.(float64)).
func TestMathFunctionWithTypeAssertion(t *testing.T) {
	g := New("test")

	// Create a program that calls sin(x) where x is a variable (interface{})
	prog := &core.Program{
		Decls: []core.CoreExpr{
			&core.Let{
				Name: "computeSin",
				Value: &core.Lambda{
					Params: []string{"angle"},
					Body: &core.App{
						Func: &core.VarGlobal{
							Ref: core.GlobalRef{Module: "std/math", Name: "sin"},
						},
						Args: []core.CoreExpr{&core.Var{Name: "angle"}},
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

	// Should contain type assertion for the argument
	if !strings.Contains(codeStr, ".(float64)") {
		t.Error("Generated code should contain .(float64) type assertion for math function arg")
		t.Logf("Generated code:\n%s", codeStr)
	}

	// Should contain math.Sin
	if !strings.Contains(codeStr, "math.Sin") {
		t.Error("Generated code should contain math.Sin")
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
