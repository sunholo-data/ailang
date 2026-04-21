package golang

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/core"
)

// TestFlatCodegen_SimpleLetChain tests that a simple let chain generates flat code.
// M-CODEGEN-V2: This is the core test for the Block IR flattening.
func TestFlatCodegen_SimpleLetChain(t *testing.T) {
	// func test() { let x = 1 in let y = 2 in x + y }
	body := &core.BinOp{
		CoreNode: core.CoreNode{NodeID: 100},
		Op:       "+",
		Left:     &core.Var{CoreNode: core.CoreNode{NodeID: 101}, Name: "x"},
		Right:    &core.Var{CoreNode: core.CoreNode{NodeID: 102}, Name: "y"},
	}

	letY := &core.Let{
		CoreNode: core.CoreNode{NodeID: 103},
		Name:     "y",
		Value:    &core.Lit{CoreNode: core.CoreNode{NodeID: 104}, Kind: core.IntLit, Value: int64(2)},
		Body:     body,
	}

	letX := &core.Let{
		CoreNode: core.CoreNode{NodeID: 105},
		Name:     "x",
		Value:    &core.Lit{CoreNode: core.CoreNode{NodeID: 106}, Kind: core.IntLit, Value: int64(1)},
		Body:     letY,
	}

	lam := &core.Lambda{
		CoreNode: core.CoreNode{NodeID: 107},
		Params:   []string{},
		Body:     letX,
	}

	topLevelLet := &core.Let{
		Name:  "test",
		Value: lam,
		Body:  &core.Var{Name: "test"},
	}

	prog := &core.Program{
		Decls: []core.CoreExpr{topLevelLet},
	}

	gen := New("test")
	code, err := gen.Generate(prog)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	codeStr := string(code)

	// Should have flat variable declarations
	if !strings.Contains(codeStr, "var x interface{} =") {
		t.Errorf("Expected flat 'var x interface{} =', got:\n%s", codeStr)
	}
	if !strings.Contains(codeStr, "var y interface{} =") {
		t.Errorf("Expected flat 'var y interface{} =', got:\n%s", codeStr)
	}

	// Should NOT have nested IIFEs (the whole point!)
	iifeCount := strings.Count(codeStr, "return func() interface{} {")
	if iifeCount > 0 {
		t.Errorf("Expected no nested IIFEs, found %d in:\n%s", iifeCount, codeStr)
	}
}

// TestFlatCodegen_DeeplyNestedLets tests that 10 nested lets still generate flat code.
// M-CODEGEN-V2: This would have generated 10 levels of nesting before the fix.
func TestFlatCodegen_DeeplyNestedLets(t *testing.T) {
	// Build: let v0 = 0 in let v1 = 1 in ... let v9 = 9 in v0
	var body core.CoreExpr = &core.Var{CoreNode: core.CoreNode{NodeID: 1}, Name: "v0"}

	for i := 9; i >= 0; i-- {
		name := string(rune('a' + i)) // a, b, c, ..., j
		body = &core.Let{
			CoreNode: core.CoreNode{NodeID: uint64(10 + i)},
			Name:     name,
			Value:    &core.Lit{CoreNode: core.CoreNode{NodeID: uint64(20 + i)}, Kind: core.IntLit, Value: int64(i)},
			Body:     body,
		}
	}

	lam := &core.Lambda{
		CoreNode: core.CoreNode{NodeID: 100},
		Params:   []string{},
		Body:     body,
	}

	topLevelLet := &core.Let{
		Name:  "test",
		Value: lam,
		Body:  &core.Var{Name: "test"},
	}

	prog := &core.Program{
		Decls: []core.CoreExpr{topLevelLet},
	}

	gen := New("test")
	code, err := gen.Generate(prog)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	codeStr := string(code)

	// Count variable declarations - should be 10
	varCount := strings.Count(codeStr, "var ")
	// Note: there will be additional vars from runtime helpers, so we check for at least 10
	if varCount < 10 {
		t.Errorf("Expected at least 10 variable declarations, found %d", varCount)
	}

	// Should NOT have ANY nested IIFEs
	iifeCount := strings.Count(codeStr, "return func() interface{} {")
	if iifeCount > 0 {
		t.Errorf("Expected no nested IIFEs for 10 lets, found %d in:\n%s", iifeCount, codeStr)
	}

	// Check that the output lines are reasonable (not deeply nested)
	// With 10 lets, old code would have 10+ levels of nesting
	// New code should have much less - runtime helpers may have some nesting
	lines := strings.Split(codeStr, "\n")
	maxIndent := 0
	for _, line := range lines {
		indent := len(line) - len(strings.TrimLeft(line, "\t"))
		if indent > maxIndent {
			maxIndent = indent
		}
	}
	// Function body should have reasonable indentation (not 10+ from nested lets)
	// Allow up to 8 for runtime helpers that have some natural nesting
	if maxIndent > 8 {
		t.Errorf("Expected max indentation <= 8, got %d (suggests O(n) nesting)", maxIndent)
	}
}

// TestFlatCodegen_NestedLetInValue tests that Let in Value position still works.
// M-CODEGEN-V2: Only top-level lets are flattened; nested lets in values use single IIFE.
func TestFlatCodegen_NestedLetInValue(t *testing.T) {
	// func test() { let x = (let y = 1 in y) in x }
	// The inner let is in VALUE position, so it should use an IIFE
	innerLet := &core.Let{
		CoreNode: core.CoreNode{NodeID: 200},
		Name:     "y",
		Value:    &core.Lit{CoreNode: core.CoreNode{NodeID: 201}, Kind: core.IntLit, Value: int64(1)},
		Body:     &core.Var{CoreNode: core.CoreNode{NodeID: 202}, Name: "y"},
	}

	outerLet := &core.Let{
		CoreNode: core.CoreNode{NodeID: 203},
		Name:     "x",
		Value:    innerLet, // Let in VALUE position
		Body:     &core.Var{CoreNode: core.CoreNode{NodeID: 204}, Name: "x"},
	}

	lam := &core.Lambda{
		CoreNode: core.CoreNode{NodeID: 205},
		Params:   []string{},
		Body:     outerLet,
	}

	topLevelLet := &core.Let{
		Name:  "test",
		Value: lam,
		Body:  &core.Var{Name: "test"},
	}

	prog := &core.Program{
		Decls: []core.CoreExpr{topLevelLet},
	}

	gen := New("test")
	code, err := gen.Generate(prog)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	codeStr := string(code)

	// The inner let (in Value position) SHOULD generate an IIFE
	// This is expected and correct - we're not trying to eliminate ALL IIFEs,
	// just nested chains of them
	if !strings.Contains(codeStr, "var x interface{} =") {
		t.Errorf("Expected 'var x interface{} =', got:\n%s", codeStr)
	}

	// The code should compile (we test this separately)
	// The key is that we don't get O(n) nesting - just 1 level for value-position lets
}

// TestFlatCodegen_NoLetBody tests that a function with no lets still works.
// M-CODEGEN-V2: Edge case - no lets means empty Block with just FinalExpr.
func TestFlatCodegen_NoLetBody(t *testing.T) {
	// func test() { 42 }
	lam := &core.Lambda{
		CoreNode: core.CoreNode{NodeID: 300},
		Params:   []string{},
		Body:     &core.Lit{CoreNode: core.CoreNode{NodeID: 301}, Kind: core.IntLit, Value: int64(42)},
	}

	topLevelLet := &core.Let{
		Name:  "test",
		Value: lam,
		Body:  &core.Var{Name: "test"},
	}

	prog := &core.Program{
		Decls: []core.CoreExpr{topLevelLet},
	}

	gen := New("test")
	code, err := gen.Generate(prog)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	codeStr := string(code)

	// Should have a simple return statement
	if !strings.Contains(codeStr, "return int64(42)") {
		t.Errorf("Expected 'return int64(42)', got:\n%s", codeStr)
	}

	// Should NOT have any IIFEs (no lets = no need for them)
	iifeCount := strings.Count(codeStr, "return func() interface{} {")
	if iifeCount > 0 {
		t.Errorf("Expected no IIFEs for function without lets, found %d", iifeCount)
	}
}
