package golang

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/core"
)

// TestStringConvBuiltinMapping tests that string conversion builtins are mapped correctly.
// M-CODEGEN-STDLIB-STRING: Verifies floatToStr, intToStr generate correct Go code.
func TestStringConvBuiltinMapping(t *testing.T) {
	tests := []struct {
		name     string
		builtin  string
		expected StringConvKind
	}{
		// Builtins (underscore prefix)
		{"floatToStr builtin", "_string_floatToStr", StringConvFloatToStr},
		{"intToStr builtin", "_string_intToStr", StringConvIntToStr},

		// stdlib wrappers
		{"floatToStr wrapper", "floatToStr", StringConvFloatToStr},
		{"intToStr wrapper", "intToStr", StringConvIntToStr},

		// Non-string-conv should return None
		{"random func", "some_random_func", StringConvNone},
		{"math func", "_math_sin", StringConvNone},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := New("test")
			result := g.getStringConvFunction(&core.VarGlobal{
				Ref: core.GlobalRef{Name: tt.builtin},
			})
			if result != tt.expected {
				t.Errorf("getStringConvFunction(%q) = %v, want %v", tt.builtin, result, tt.expected)
			}
			if tt.expected != StringConvNone && !g.needsStrconvImport {
				t.Errorf("needsStrconvImport should be true after mapping %q", tt.builtin)
			}
			if tt.expected == StringConvNone && g.needsStrconvImport {
				t.Errorf("needsStrconvImport should be false for non-string-conv %q", tt.builtin)
			}
		})
	}
}

// TestFloatToStrGeneration tests that floatToStr generates correct strconv.FormatFloat call.
func TestFloatToStrGeneration(t *testing.T) {
	g := New("test")

	// Create a program that calls floatToStr(x)
	prog := &core.Program{
		Decls: []core.CoreExpr{
			&core.Let{
				Name: "formatFloat",
				Value: &core.Lambda{
					Params: []string{"x"},
					Body: &core.App{
						Func: &core.VarGlobal{
							Ref: core.GlobalRef{Module: "std/string", Name: "_string_floatToStr"},
						},
						Args: []core.CoreExpr{&core.Var{Name: "x"}},
					},
				},
				Body: &core.Var{Name: "formatFloat"},
			},
		},
	}

	code, err := g.Generate(prog)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	codeStr := string(code)

	// Check that strconv.FormatFloat is used
	if !strings.Contains(codeStr, "strconv.FormatFloat") {
		t.Error("Generated code should contain strconv.FormatFloat for floatToStr call")
		t.Logf("Generated code:\n%s", codeStr)
	}

	// Check that proper arguments are passed
	if !strings.Contains(codeStr, "'g', -1, 64") {
		t.Error("Generated code should contain 'g', -1, 64 arguments for FormatFloat")
		t.Logf("Generated code:\n%s", codeStr)
	}

	// Check that strconv import is present
	if !strings.Contains(codeStr, `"strconv"`) {
		t.Error("Generated code should contain strconv import")
		t.Logf("Generated code:\n%s", codeStr)
	}
}

// TestIntToStrGeneration tests that intToStr generates correct strconv.Itoa call.
func TestIntToStrGeneration(t *testing.T) {
	g := New("test")

	// Create a program that calls intToStr(n)
	prog := &core.Program{
		Decls: []core.CoreExpr{
			&core.Let{
				Name: "formatInt",
				Value: &core.Lambda{
					Params: []string{"n"},
					Body: &core.App{
						Func: &core.VarGlobal{
							Ref: core.GlobalRef{Module: "std/string", Name: "_string_intToStr"},
						},
						Args: []core.CoreExpr{&core.Var{Name: "n"}},
					},
				},
				Body: &core.Var{Name: "formatInt"},
			},
		},
	}

	code, err := g.Generate(prog)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	codeStr := string(code)

	// Check that strconv.Itoa is used
	if !strings.Contains(codeStr, "strconv.Itoa") {
		t.Error("Generated code should contain strconv.Itoa for intToStr call")
		t.Logf("Generated code:\n%s", codeStr)
	}

	// Check that int() conversion is present (Itoa needs int, not int64)
	if !strings.Contains(codeStr, "int(") {
		t.Error("Generated code should contain int() conversion for Itoa")
		t.Logf("Generated code:\n%s", codeStr)
	}

	// Check that strconv import is present
	if !strings.Contains(codeStr, `"strconv"`) {
		t.Error("Generated code should contain strconv import")
		t.Logf("Generated code:\n%s", codeStr)
	}
}

// TestNoStrconvImportWhenNotNeeded tests that strconv import is not added when not needed.
func TestNoStrconvImportWhenNotNeeded(t *testing.T) {
	g := New("test")

	// Create a simple program without string conversion functions
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

	// Check that strconv import is NOT present
	if strings.Contains(codeStr, `"strconv"`) {
		t.Error("Generated code should NOT contain strconv import when not needed")
		t.Logf("Generated code:\n%s", codeStr)
	}
}

// TestStrconvImportWithSkipRuntimeHelpers tests that strconv import is added even when
// skipRuntimeHelpers is true (multi-file compilation mode).
func TestStrconvImportWithSkipRuntimeHelpers(t *testing.T) {
	g := New("test")
	g.SetSkipRuntimeHelpers(true) // Multi-file mode

	// Create a program that uses floatToStr
	prog := &core.Program{
		Decls: []core.CoreExpr{
			&core.Let{
				Name: "formatNum",
				Value: &core.App{
					Func: &core.VarGlobal{
						Ref: core.GlobalRef{Module: "std/string", Name: "_string_floatToStr"},
					},
					Args: []core.CoreExpr{&core.Lit{Kind: core.FloatLit, Value: 3.14}},
				},
				Body: &core.Var{Name: "formatNum"},
			},
		},
	}

	code, err := g.Generate(prog)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	codeStr := string(code)

	// Check that strconv import is present even with skipRuntimeHelpers=true
	if !strings.Contains(codeStr, `"strconv"`) {
		t.Error("Generated code should contain strconv import even with skipRuntimeHelpers=true")
		t.Logf("Generated code:\n%s", codeStr)
	}

	// Check that strconv.FormatFloat is used
	if !strings.Contains(codeStr, "strconv.FormatFloat") {
		t.Error("Generated code should contain strconv.FormatFloat")
		t.Logf("Generated code:\n%s", codeStr)
	}

	// Check that other runtime imports (fmt, reflect, strings) are NOT present
	if strings.Contains(codeStr, `"fmt"`) {
		t.Error("Generated code should NOT contain fmt import when skipRuntimeHelpers=true")
	}
	if strings.Contains(codeStr, `"reflect"`) {
		t.Error("Generated code should NOT contain reflect import when skipRuntimeHelpers=true")
	}
}

// TestStringConvWithTypeAssertion tests that string conversion functions have type assertions
// added for interface{} arguments.
func TestStringConvWithTypeAssertion(t *testing.T) {
	g := New("test")

	// Create a program that calls floatToStr(x) where x is a variable (interface{})
	prog := &core.Program{
		Decls: []core.CoreExpr{
			&core.Let{
				Name: "formatNum",
				Value: &core.Lambda{
					Params: []string{"num"},
					Body: &core.App{
						Func: &core.VarGlobal{
							Ref: core.GlobalRef{Module: "std/string", Name: "_string_floatToStr"},
						},
						Args: []core.CoreExpr{&core.Var{Name: "num"}},
					},
				},
				Body: &core.Var{Name: "formatNum"},
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
		t.Error("Generated code should contain .(float64) type assertion for floatToStr arg")
		t.Logf("Generated code:\n%s", codeStr)
	}
}

// TestIntToStrWithTypeAssertion tests that intToStr has int64 type assertion and int conversion.
func TestIntToStrWithTypeAssertion(t *testing.T) {
	g := New("test")

	// Create a program that calls intToStr(n) where n is a variable (interface{})
	prog := &core.Program{
		Decls: []core.CoreExpr{
			&core.Let{
				Name: "formatNum",
				Value: &core.Lambda{
					Params: []string{"num"},
					Body: &core.App{
						Func: &core.VarGlobal{
							Ref: core.GlobalRef{Module: "std/string", Name: "_string_intToStr"},
						},
						Args: []core.CoreExpr{&core.Var{Name: "num"}},
					},
				},
				Body: &core.Var{Name: "formatNum"},
			},
		},
	}

	code, err := g.Generate(prog)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	codeStr := string(code)

	// Should contain int64 type assertion for the argument
	if !strings.Contains(codeStr, ".(int64)") {
		t.Error("Generated code should contain .(int64) type assertion for intToStr arg")
		t.Logf("Generated code:\n%s", codeStr)
	}

	// Should contain int() conversion wrapper (strconv.Itoa needs int, not int64)
	if !strings.Contains(codeStr, "int(") {
		t.Error("Generated code should contain int() conversion wrapper for Itoa")
		t.Logf("Generated code:\n%s", codeStr)
	}
}
