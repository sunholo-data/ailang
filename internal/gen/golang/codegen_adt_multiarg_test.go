package golang

import (
	"strings"
	"testing"

	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/types"
)

// TestADTConstructorMultiArg tests that ADT constructors with multiple fields
// generate correct Go code without double-paren calls.
// This is a regression test for GitHub issue #52 (M-CODEGEN-ADT-DOUBLE-PAREN).
func TestADTConstructorMultiArg(t *testing.T) {
	// Simulate: type DrawCmd = | Clear | Viewport(id: int, x: float, y: float, w: float, h: float)
	// And a function: let makeViewport = \id. \x. \y. \w. \h. Viewport(id, x, y, w, h)

	// Create the Lambda body: App($adt.make_DrawCmd_Viewport, [id, x, y, w, h])
	appNode := &core.App{
		CoreNode: core.CoreNode{NodeID: 10},
		Func: &core.VarGlobal{
			CoreNode: core.CoreNode{NodeID: 11},
			Ref:      core.GlobalRef{Module: "$adt", Name: "make_DrawCmd_Viewport"},
		},
		Args: []core.CoreExpr{
			&core.Var{CoreNode: core.CoreNode{NodeID: 12}, Name: "id"},
			&core.Var{CoreNode: core.CoreNode{NodeID: 13}, Name: "x"},
			&core.Var{CoreNode: core.CoreNode{NodeID: 14}, Name: "y"},
			&core.Var{CoreNode: core.CoreNode{NodeID: 15}, Name: "w"},
			&core.Var{CoreNode: core.CoreNode{NodeID: 16}, Name: "h"},
		},
	}

	// Create the Lambda: \id. \x. \y. \w. \h. Viewport(id, x, y, w, h)
	lam := &core.Lambda{
		CoreNode: core.CoreNode{NodeID: 1},
		Params:   []string{"id", "x", "y", "w", "h"},
		Body:     appNode,
	}

	prog := &core.Program{
		Decls: []core.CoreExpr{
			&core.Let{
				CoreNode: core.CoreNode{NodeID: 0},
				Name:     "makeViewport",
				Value:    lam,
				Body:     &core.Var{Name: "makeViewport"},
			},
		},
		Meta: map[string]*core.DeclMeta{
			"makeViewport": {IsExport: true},
		},
	}

	gen := New("test")

	// Register the ADT constructor with types
	gen.RegisterADTConstructorWithTypes("DrawCmd", "Viewport", []string{
		"int64", "float64", "float64", "float64", "float64",
	})

	// Set up CoreTypeInfo for the Lambda
	cti := types.CoreTypeInfo{
		1: &types.TFunc2{
			Params: []types.Type{
				&types.TCon{Name: "int"},
				&types.TCon{Name: "float"},
				&types.TCon{Name: "float"},
				&types.TCon{Name: "float"},
				&types.TCon{Name: "float"},
			},
			Return: &types.TCon{Name: "DrawCmd"},
		},
	}
	gen.SetCoreTypeInfo(cti)

	code, err := gen.Generate(prog)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	codeStr := string(code)

	// Bug check: Should NOT have double-paren pattern "()()"
	if strings.Contains(codeStr, "()()") {
		t.Errorf("BUG: Generated code contains '()()' double-paren pattern!\n\nGenerated code:\n%s", codeStr)
	}

	// Should have proper constructor call with arguments
	// The _impl function should call NewDrawCmdViewport with args
	if !strings.Contains(codeStr, "NewDrawCmdViewport(") {
		t.Errorf("Missing NewDrawCmdViewport( call, got:\n%s", codeStr)
	}

	// Should NOT have empty NewDrawCmdViewport() call (nullary pattern)
	// Unless followed immediately by arguments
	if strings.Contains(codeStr, "NewDrawCmdViewport()") && !strings.Contains(codeStr, "NewDrawCmdViewport()(") {
		// Check if there's actually an empty call
		if strings.Count(codeStr, "NewDrawCmdViewport()") > 0 &&
			!strings.Contains(codeStr, "NewDrawCmdViewport(id") &&
			!strings.Contains(codeStr, "NewDrawCmdViewport(x") {
			t.Errorf("Found empty NewDrawCmdViewport() call, expected args, got:\n%s", codeStr)
		}
	}

	t.Logf("Generated code:\n%s", codeStr)
}

// TestADTConstructorUnregistered tests what happens when ADT constructor is NOT registered.
// This reproduces GitHub issue #52 (M-CODEGEN-ADT-DOUBLE-PAREN).
func TestADTConstructorUnregistered(t *testing.T) {
	// Same setup as TestADTConstructorMultiArg, but WITHOUT registering the constructor
	appNode := &core.App{
		CoreNode: core.CoreNode{NodeID: 10},
		Func: &core.VarGlobal{
			CoreNode: core.CoreNode{NodeID: 11},
			Ref:      core.GlobalRef{Module: "$adt", Name: "make_DrawCmd_Viewport"},
		},
		Args: []core.CoreExpr{
			&core.Var{CoreNode: core.CoreNode{NodeID: 12}, Name: "id"},
			&core.Var{CoreNode: core.CoreNode{NodeID: 13}, Name: "x"},
			&core.Var{CoreNode: core.CoreNode{NodeID: 14}, Name: "y"},
			&core.Var{CoreNode: core.CoreNode{NodeID: 15}, Name: "w"},
			&core.Var{CoreNode: core.CoreNode{NodeID: 16}, Name: "h"},
		},
	}

	lam := &core.Lambda{
		CoreNode: core.CoreNode{NodeID: 1},
		Params:   []string{"id", "x", "y", "w", "h"},
		Body:     appNode,
	}

	prog := &core.Program{
		Decls: []core.CoreExpr{
			&core.Let{
				CoreNode: core.CoreNode{NodeID: 0},
				Name:     "makeViewport",
				Value:    lam,
				Body:     &core.Var{Name: "makeViewport"},
			},
		},
		Meta: map[string]*core.DeclMeta{
			"makeViewport": {IsExport: true},
		},
	}

	gen := New("test")

	// DO NOT register the constructor - this simulates the bug condition
	// gen.RegisterADTConstructorWithTypes("DrawCmd", "Viewport", []string{...})

	cti := types.CoreTypeInfo{
		1: &types.TFunc2{
			Params: []types.Type{
				&types.TCon{Name: "int"},
				&types.TCon{Name: "float"},
				&types.TCon{Name: "float"},
				&types.TCon{Name: "float"},
				&types.TCon{Name: "float"},
			},
			Return: &types.TCon{Name: "DrawCmd"},
		},
	}
	gen.SetCoreTypeInfo(cti)

	code, err := gen.Generate(prog)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	codeStr := string(code)

	// Check for the bug: double-paren pattern "()()"
	if strings.Contains(codeStr, "()()") {
		t.Errorf("BUG REPRODUCED: Generated code contains '()()' double-paren pattern!\n\nGenerated code (relevant part):\n%s",
			extractRelevantCode(codeStr, "makeViewport"))
	}

	// Even if constructor is not registered, we should still get reasonable output
	// (not crashing, and ideally still passing args)
	t.Logf("Generated code (makeViewport):\n%s", extractRelevantCode(codeStr, "makeViewport"))
}

// extractRelevantCode extracts functions containing the given name from code
func extractRelevantCode(code, funcName string) string {
	lines := strings.Split(code, "\n")
	var result []string
	inFunc := false
	braceCount := 0
	for _, line := range lines {
		if strings.Contains(line, funcName) && strings.Contains(line, "func") {
			inFunc = true
		}
		if inFunc {
			result = append(result, line)
			braceCount += strings.Count(line, "{") - strings.Count(line, "}")
			if braceCount == 0 && len(result) > 1 {
				result = append(result, "")
				inFunc = false
			}
		}
	}
	return strings.Join(result, "\n")
}

// TestADTConstructorNullary tests that nullary ADT constructors work correctly.
func TestADTConstructorNullary(t *testing.T) {
	// Simulate: type DrawCmd = | Clear | Viewport(...)
	// And: let clear = Clear

	varNode := &core.VarGlobal{
		CoreNode: core.CoreNode{NodeID: 10},
		Ref:      core.GlobalRef{Module: "$adt", Name: "make_DrawCmd_Clear"},
	}

	prog := &core.Program{
		Decls: []core.CoreExpr{
			&core.Let{
				CoreNode: core.CoreNode{NodeID: 0},
				Name:     "clear",
				Value:    varNode,
				Body:     &core.Var{Name: "clear"},
			},
		},
		Meta: map[string]*core.DeclMeta{
			"clear": {IsExport: true},
		},
	}

	gen := New("test")

	// Register Clear as nullary (0 fields)
	gen.RegisterADTConstructorWithTypes("DrawCmd", "Clear", []string{})

	code, err := gen.Generate(prog)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	codeStr := string(code)

	// Nullary constructor SHOULD have ()
	if !strings.Contains(codeStr, "NewDrawCmdClear()") {
		t.Errorf("Missing NewDrawCmdClear() call for nullary constructor, got:\n%s", codeStr)
	}

	// Should NOT have double-paren
	if strings.Contains(codeStr, "NewDrawCmdClear()()") {
		t.Errorf("BUG: Double-paren on nullary constructor, got:\n%s", codeStr)
	}

	t.Logf("Generated code:\n%s", codeStr)
}
