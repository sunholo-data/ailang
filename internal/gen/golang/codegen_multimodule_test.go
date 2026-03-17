package golang

import (
	"strings"
	"testing"

	"github.com/sunholo/ailang/internal/core"
)

// TestNonLambdaLetBindingModulePrefix verifies that non-lambda let bindings
// (constants/variables) get module-prefixed Go names when moduleName is set.
// Bug: generateTopLevelLet did not apply moduleName__ prefix, causing
// "redeclared in this block" when two modules have the same constant name.
func TestNonLambdaLetBindingModulePrefix(t *testing.T) {
	prog := &core.Program{
		Decls: []core.CoreExpr{
			&core.Let{
				Name:  "maxWords",
				Value: &core.Lit{Kind: core.IntLit, Value: int64(5000)},
				Body:  &core.Var{Name: "maxWords"},
			},
		},
	}

	gen := New("testpkg")
	gen.SetModuleName("eval")
	code, err := gen.Generate(prog)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	output := string(code)
	// Non-exported constant should have module prefix
	if !strings.Contains(output, "eval__maxWords") {
		t.Errorf("Expected module-prefixed var name 'eval__maxWords', got:\n%s", output)
	}
}

// TestNonLambdaLetBindingDedup verifies that duplicate Let nodes for the same
// binding only emit one var declaration.
// Bug: Core lowering can produce multiple Let nodes per binding when inlining.
func TestNonLambdaLetBindingDedup(t *testing.T) {
	prog := &core.Program{
		Decls: []core.CoreExpr{
			&core.Let{
				Name:  "maxWords",
				Value: &core.Lit{Kind: core.IntLit, Value: int64(5000)},
				Body:  &core.Var{Name: "maxWords"},
			},
			// Duplicate — same binding emitted again by Core lowering
			&core.Let{
				Name:  "maxWords",
				Value: &core.Lit{Kind: core.IntLit, Value: int64(5000)},
				Body:  &core.Var{Name: "maxWords"},
			},
			// Third copy
			&core.Let{
				Name:  "maxWords",
				Value: &core.Lit{Kind: core.IntLit, Value: int64(5000)},
				Body:  &core.Var{Name: "maxWords"},
			},
		},
	}

	gen := New("testpkg")
	gen.SetModuleName("eval")
	code, err := gen.Generate(prog)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	output := string(code)
	// Should appear exactly once
	count := strings.Count(output, "eval__maxWords")
	if count != 1 {
		t.Errorf("Expected exactly 1 occurrence of eval__maxWords, got %d:\n%s", count, output)
	}
}

// TestNonLambdaLetBindingExportedAlsoPrefixed verifies that exported let bindings
// ALSO get module-prefixed when moduleName is set (multi-module mode).
// This prevents collisions when multiple modules export the same name.
func TestNonLambdaLetBindingExportedAlsoPrefixed(t *testing.T) {
	prog := &core.Program{
		Decls: []core.CoreExpr{
			&core.Let{
				Name:  "MaxWords",
				Value: &core.Lit{Kind: core.IntLit, Value: int64(5000)},
				Body:  &core.Var{Name: "MaxWords"},
			},
		},
		Meta: map[string]*core.DeclMeta{
			"MaxWords": {IsExport: true},
		},
	}

	gen := New("testpkg")
	gen.SetModuleName("eval")
	code, err := gen.Generate(prog)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	output := string(code)
	// In multi-module mode, even exported vars get prefixed
	if !strings.Contains(output, "eval__MaxWords") {
		t.Errorf("Expected module-prefixed 'eval__MaxWords', got:\n%s", output)
	}
}

// TestNonLambdaLetBindingNoModuleNoPrefix verifies that without moduleName,
// no prefix is applied (single-module compilation).
func TestNonLambdaLetBindingNoModuleNoPrefix(t *testing.T) {
	prog := &core.Program{
		Decls: []core.CoreExpr{
			&core.Let{
				Name:  "maxWords",
				Value: &core.Lit{Kind: core.IntLit, Value: int64(5000)},
				Body:  &core.Var{Name: "maxWords"},
			},
		},
	}

	gen := New("testpkg")
	// No SetModuleName — single-module mode
	code, err := gen.Generate(prog)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	output := string(code)
	if strings.Contains(output, "__") {
		t.Errorf("Single-module should not have module prefix, got:\n%s", output)
	}
}

// TestCrossModuleFunctionIsolation verifies that calling Generate for two
// different modules with the same function name does not cause collisions.
// Bug: The Generator reuses topLevelFuncs map across modules, so module B's
// function references may resolve to module A's Go function name.
func TestCrossModuleFunctionIsolation(t *testing.T) {
	makeProg := func(funcName string) *core.Program {
		return &core.Program{
			Decls: []core.CoreExpr{
				&core.Let{
					Name: funcName,
					Value: &core.Lambda{
						Params: []string{"x"},
						Body:   &core.Var{Name: "x"},
					},
					Body: &core.Var{Name: funcName},
				},
			},
		}
	}

	gen := New("testpkg")

	// Generate module A
	gen.SetModuleName("moduleA")
	codeA, err := gen.Generate(makeProg("parseComments"))
	if err != nil {
		t.Fatalf("Generate moduleA failed: %v", err)
	}

	// Reset per-module state before module B
	gen.ResetPerModuleState()

	// Generate module B with same function name
	gen.SetModuleName("moduleB")
	codeB, err := gen.Generate(makeProg("parseComments"))
	if err != nil {
		t.Fatalf("Generate moduleB failed: %v", err)
	}

	outputA := string(codeA)
	outputB := string(codeB)

	// Each module should have its own prefixed function
	if !strings.Contains(outputA, "moduleA__parseComments") {
		t.Errorf("Module A should have moduleA__parseComments, got:\n%s", outputA)
	}
	if !strings.Contains(outputB, "moduleB__parseComments") {
		t.Errorf("Module B should have moduleB__parseComments, got:\n%s", outputB)
	}

	// Module B should NOT reference module A's names
	if strings.Contains(outputB, "moduleA__") {
		t.Errorf("Module B should not contain moduleA__ references, got:\n%s", outputB)
	}
}

// TestNonLambdaLetBindingVarReference verifies that when a non-lambda let
// binding is module-prefixed, references to that variable also resolve
// to the prefixed name.
func TestNonLambdaLetBindingVarReference(t *testing.T) {
	prog := &core.Program{
		Decls: []core.CoreExpr{
			// let maxWords = 5000
			&core.Let{
				Name:  "maxWords",
				Value: &core.Lit{Kind: core.IntLit, Value: int64(5000)},
				Body:  &core.Var{Name: "maxWords"},
			},
			// let processWords = \text -> maxWords (reference to the constant)
			&core.Let{
				Name: "processWords",
				Value: &core.Lambda{
					Params: []string{"text"},
					Body: &core.VarGlobal{
						Ref: core.GlobalRef{Name: "maxWords"},
					},
				},
				Body: &core.Var{Name: "processWords"},
			},
		},
	}

	gen := New("testpkg")
	gen.SetModuleName("eval")
	code, err := gen.Generate(prog)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	output := string(code)
	// The var declaration should be prefixed
	if !strings.Contains(output, "eval__maxWords") {
		t.Errorf("Expected module-prefixed var 'eval__maxWords', got:\n%s", output)
	}
}

// TestListPatternMatchGeneratesValidGo verifies that match expressions with
// list patterns generate syntactically valid Go code (via if-else path, not switch).
// Bug: The switch/case fallback emitted `case []interface{}{}:` which is invalid Go.
func TestListPatternMatchGeneratesValidGo(t *testing.T) {
	prog := &core.Program{
		Decls: []core.CoreExpr{
			&core.Let{
				Name: "checkList",
				Value: &core.Lambda{
					Params: []string{"xs"},
					Body: &core.Match{
						Scrutinee: &core.Var{Name: "xs"},
						Arms: []core.MatchArm{
							{
								Pattern: &core.ListPattern{
									Elements: nil, // empty list pattern: []
									Tail:     nil,
								},
								Body: &core.Lit{Kind: core.StringLit, Value: "empty"},
							},
							{
								Pattern: &core.WildcardPattern{},
								Body:    &core.Lit{Kind: core.StringLit, Value: "non-empty"},
							},
						},
					},
				},
				Body: &core.Var{Name: "checkList"},
			},
		},
	}

	gen := New("testpkg")
	code, err := gen.Generate(prog)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	output := string(code)
	// Should use if-else path (len check), NOT switch/case
	if strings.Contains(output, "case []interface{}") {
		t.Errorf("List pattern should use if-else, not switch case. Got:\n%s", output)
	}
	// Should contain a length check for empty list
	if !strings.Contains(output, "ListLen(") && !strings.Contains(output, "len(") {
		t.Errorf("Expected length check for empty list pattern, got:\n%s", output)
	}
}
