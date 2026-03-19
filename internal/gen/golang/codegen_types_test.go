package golang

import (
	"strings"
	"testing"

	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/types"
)

// M-DX23: Test typed function signatures with CoreTypeInfo
func TestGenerateTypedFunctionSignature(t *testing.T) {
	// Create a Lambda with a known NodeID
	lam := &core.Lambda{
		CoreNode: core.CoreNode{NodeID: 42}, // Known NodeID
		Params:   []string{"world"},
		Body:     &core.Var{Name: "world"}, // Simple identity function
	}

	prog := &core.Program{
		Decls: []core.CoreExpr{
			&core.Let{
				Name:  "step",
				Value: lam,
				Body:  &core.Var{Name: "step"},
			},
		},
		Meta: map[string]*core.DeclMeta{
			"step": {IsExport: true},
		},
	}

	// Set up CoreTypeInfo with a typed function signature
	// step: World -> World (AILANG), *World -> *World (Go)
	// M-DX25.6: ADT types map to pointers in Go
	coreTypeInfo := make(types.CoreTypeInfo)
	worldType := &types.TCon{Name: "World"}
	funcType := &types.TFunc2{
		Params:    []types.Type{worldType},
		EffectRow: nil,
		Return:    worldType,
	}
	coreTypeInfo[42] = funcType // Map Lambda's NodeID to its type

	gen := New("game")
	gen.SetCoreTypeInfo(coreTypeInfo)

	code, err := gen.Generate(prog)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	codeStr := string(code)

	// Should have typed parameter (World pointer not interface{})
	// M-DX25.6: ADT types are represented as pointers in Go
	if !strings.Contains(codeStr, "world *World") {
		t.Errorf("Expected typed parameter 'world *World', got:\n%s", codeStr)
	}

	// Should have typed return type (World pointer not interface{})
	// M-DX25.6: ADT types are represented as pointers in Go
	if !strings.Contains(codeStr, ") *World {") {
		t.Errorf("Expected typed return type '*World', got:\n%s", codeStr)
	}
}

// M-DX23: Test fallback to interface{} when CoreTypeInfo is not available
func TestGenerateFallbackToInterface(t *testing.T) {
	lam := &core.Lambda{
		CoreNode: core.CoreNode{NodeID: 100},
		Params:   []string{"x"},
		Body:     &core.Var{Name: "x"},
	}

	prog := &core.Program{
		Decls: []core.CoreExpr{
			&core.Let{
				Name:  "identity",
				Value: lam,
				Body:  &core.Var{Name: "identity"},
			},
		},
	}

	// No CoreTypeInfo set - should fall back to interface{}
	gen := New("test")
	code, err := gen.Generate(prog)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	codeStr := string(code)

	// Should have interface{} parameter (fallback)
	if !strings.Contains(codeStr, "x interface{}") {
		t.Errorf("Expected fallback to 'x interface{}', got:\n%s", codeStr)
	}

	// Should have interface{} return type (fallback)
	if !strings.Contains(codeStr, ") interface{} {") {
		t.Errorf("Expected fallback return type 'interface{}', got:\n%s", codeStr)
	}
}

// M-DX25.2: Test typed let bindings with CoreTypeInfo
// Uses nested let inside a function to test the IIFE pattern
func TestTypedLetBindings(t *testing.T) {
	// func test() bool { let x = true in x }
	valueLit := &core.Lit{
		CoreNode: core.CoreNode{NodeID: 102}, // Value expression has its own NodeID
		Kind:     core.BoolLit,
		Value:    true,
	}
	nestedLet := &core.Let{
		CoreNode: core.CoreNode{NodeID: 100},
		Name:     "x",
		Value:    valueLit,
		Body:     &core.Var{Name: "x"},
	}

	// Wrap in a function declaration (let test = \() -> nestedLet)
	lam := &core.Lambda{
		CoreNode: core.CoreNode{NodeID: 101},
		Params:   []string{},
		Body:     nestedLet,
	}

	topLevelLet := &core.Let{
		Name:  "test",
		Value: lam,
		Body:  &core.Var{Name: "test"},
	}

	prog := &core.Program{
		Decls: []core.CoreExpr{topLevelLet},
	}

	// Set up CoreTypeInfo:
	// - NodeID 100 (let expression) -> bool (the body's type = return type)
	// - NodeID 101 (lambda) -> () -> bool (function type for wrapper signature)
	// - NodeID 102 (value expression) -> bool (the variable's type)
	coreTypeInfo := make(types.CoreTypeInfo)
	coreTypeInfo[100] = &types.TCon{Name: "bool"}                                                // Let expression (body) type
	coreTypeInfo[101] = &types.TFunc2{Params: []types.Type{}, Return: &types.TCon{Name: "bool"}} // Lambda type for wrapper
	coreTypeInfo[102] = &types.TCon{Name: "bool"}                                                // Value expression type

	gen := New("test")
	gen.SetCoreTypeInfo(coreTypeInfo)
	code, err := gen.Generate(prog)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	codeStr := string(code)

	// M-DX26: Now generates _impl (interface{}) and wrapper (typed)
	// The wrapper should return bool, calling _impl and asserting
	if !strings.Contains(codeStr, "func test() bool {") {
		t.Errorf("Expected typed wrapper 'func test() bool {', got:\n%s", codeStr)
	}

	// M-DX26: _impl should return interface{}
	if !strings.Contains(codeStr, "func test_impl() interface{} {") {
		t.Errorf("Expected _impl function 'func test_impl() interface{} {', got:\n%s", codeStr)
	}
}

// M-DX25.2: Test let bindings with value that produces interface{}
func TestTypedLetBindingsWithAssertion(t *testing.T) {
	// func test() int64 { let x = 1 + 2 in x }
	binOp := &core.BinOp{
		CoreNode: core.CoreNode{NodeID: 201},
		Op:       "+",
		Left:     &core.Lit{Kind: core.IntLit, Value: int64(1)},
		Right:    &core.Lit{Kind: core.IntLit, Value: int64(2)},
	}
	nestedLet := &core.Let{
		CoreNode: core.CoreNode{NodeID: 200},
		Name:     "x",
		Value:    binOp,
		Body:     &core.Var{Name: "x"},
	}

	// Wrap in a function
	lam := &core.Lambda{
		CoreNode: core.CoreNode{NodeID: 202},
		Params:   []string{},
		Body:     nestedLet,
	}

	topLevelLet := &core.Let{
		Name:  "test",
		Value: lam,
		Body:  &core.Var{Name: "test"},
	}

	prog := &core.Program{
		Decls: []core.CoreExpr{topLevelLet},
	}

	// Set up CoreTypeInfo:
	// - NodeID 200 (let expression) -> int (the body's type = return type)
	// - NodeID 201 (BinOp value) -> int (the variable's type)
	// - NodeID 202 (lambda) -> () -> int (function type for wrapper signature)
	coreTypeInfo := make(types.CoreTypeInfo)
	coreTypeInfo[200] = &types.TCon{Name: "int"}                                                // Let expression (body) type
	coreTypeInfo[201] = &types.TCon{Name: "int"}                                                // Value expression type (BinOp)
	coreTypeInfo[202] = &types.TFunc2{Params: []types.Type{}, Return: &types.TCon{Name: "int"}} // Lambda type for wrapper

	gen := New("test")
	gen.SetCoreTypeInfo(coreTypeInfo)
	code, err := gen.Generate(prog)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	codeStr := string(code)

	// M-DX26: Now generates _impl (interface{}) and wrapper (typed)
	// The wrapper should return int64, calling _impl and asserting
	if !strings.Contains(codeStr, "func test() int64 {") {
		t.Errorf("Expected typed wrapper 'func test() int64 {', got:\n%s", codeStr)
	}

	// M-DX26: _impl should return interface{}
	if !strings.Contains(codeStr, "func test_impl() interface{} {") {
		t.Errorf("Expected _impl function 'func test_impl() interface{} {', got:\n%s", codeStr)
	}

	// M-DX26: The wrapper SHOULD have type assertion since _impl returns interface{}
	if !strings.Contains(codeStr, ".(int64)") {
		t.Errorf("Wrapper should have type assertion '.(int64)', got:\n%s", codeStr)
	}
}

// M-DX25.2: Test fallback to interface{} when CoreTypeInfo is not available for let
func TestTypedLetBindingsFallback(t *testing.T) {
	// func test() { let x = true in x } (no type info)
	// M-CODEGEN-V2: This now generates flat code, not nested IIFEs
	nestedLet := &core.Let{
		CoreNode: core.CoreNode{NodeID: 300},
		Name:     "x",
		Value:    &core.Lit{Kind: core.BoolLit, Value: true},
		Body:     &core.Var{Name: "x"},
	}

	// Wrap in a function
	lam := &core.Lambda{
		CoreNode: core.CoreNode{NodeID: 301},
		Params:   []string{},
		Body:     nestedLet,
	}

	topLevelLet := &core.Let{
		Name:  "test",
		Value: lam,
		Body:  &core.Var{Name: "test"},
	}

	prog := &core.Program{
		Decls: []core.CoreExpr{topLevelLet},
	}

	// No CoreTypeInfo set - should fall back to interface{}
	gen := New("test")
	code, err := gen.Generate(prog)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	codeStr := string(code)

	// M-CODEGEN-V2: Should generate flat function body with interface{} variable
	if !strings.Contains(codeStr, "var x interface{} =") {
		t.Errorf("Expected 'var x interface{} =', got:\n%s", codeStr)
	}

	// M-CODEGEN-V2: Should NOT generate nested IIFEs (the whole point of the fix!)
	// Count occurrences of IIFE pattern - should be zero in function body
	iifeCount := strings.Count(codeStr, "return func() interface{} {")
	if iifeCount > 0 {
		t.Errorf("Expected no nested IIFEs in function body, found %d", iifeCount)
	}
}
