package golang

import (
	"strings"
	"testing"

	"github.com/sunholo/ailang/internal/core"
)

func TestGenerateList(t *testing.T) {
	// [1, 2, 3]
	prog := &core.Program{
		Decls: []core.CoreExpr{
			&core.Let{
				Name: "nums",
				Value: &core.List{
					Elements: []core.CoreExpr{
						&core.Lit{Kind: core.IntLit, Value: int64(1)},
						&core.Lit{Kind: core.IntLit, Value: int64(2)},
						&core.Lit{Kind: core.IntLit, Value: int64(3)},
					},
				},
				Body: &core.Var{Name: "nums"},
			},
		},
	}

	gen := New("test")
	code, err := gen.Generate(prog)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	codeStr := string(code)

	// Check for slice literal
	if !strings.Contains(codeStr, "[]interface{}") {
		t.Errorf("Missing slice type")
	}
	if !strings.Contains(codeStr, "1") && !strings.Contains(codeStr, "2") && !strings.Contains(codeStr, "3") {
		t.Errorf("Missing list elements")
	}
}

func TestGenerateRecord(t *testing.T) {
	// { x: 10, y: 20 }
	prog := &core.Program{
		Decls: []core.CoreExpr{
			&core.Let{
				Name: "point",
				Value: &core.Record{
					Fields: map[string]core.CoreExpr{
						"x": &core.Lit{Kind: core.IntLit, Value: int64(10)},
						"y": &core.Lit{Kind: core.IntLit, Value: int64(20)},
					},
				},
				Body: &core.Var{Name: "point"},
			},
		},
	}

	gen := New("test")
	code, err := gen.Generate(prog)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	codeStr := string(code)

	// Check for map
	if !strings.Contains(codeStr, "map[string]interface{}") {
		t.Errorf("Missing map type")
	}
}

func TestGenerateNestedLet(t *testing.T) {
	// Test nested let inside a function:
	// let f = \_ . let x = 1 in let y = 2 in x + y
	prog := &core.Program{
		Decls: []core.CoreExpr{
			&core.Let{
				Name: "f",
				Value: &core.Lambda{
					Params: []string{"_"},
					Body: &core.Let{
						Name:  "x",
						Value: &core.Lit{Kind: core.IntLit, Value: int64(1)},
						Body: &core.Let{
							Name:  "y",
							Value: &core.Lit{Kind: core.IntLit, Value: int64(2)},
							Body: &core.BinOp{
								Op:    "+",
								Left:  &core.Var{Name: "x"},
								Right: &core.Var{Name: "y"},
							},
						},
					},
				},
				Body: &core.Var{Name: "f"},
			},
		},
	}

	gen := New("test")
	code, err := gen.Generate(prog)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	codeStr := string(code)

	// Should have nested let bindings inside the generated IIFE
	// M-DX13.3: Uses "var x interface{}" to allow type assertions on concrete values
	if !strings.Contains(codeStr, "var x interface{} =") {
		t.Errorf("Missing x binding, got:\n%s", codeStr)
	}
	if !strings.Contains(codeStr, "var y interface{} =") {
		t.Errorf("Missing y binding, got:\n%s", codeStr)
	}
}
