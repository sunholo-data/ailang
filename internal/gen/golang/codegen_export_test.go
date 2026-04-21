package golang

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/core"
)

func TestExportFunc_GeneratesPascalCase(t *testing.T) {
	// export func add(x, y) { x + y }
	prog := &core.Program{
		Decls: []core.CoreExpr{
			&core.Let{
				Name: "add",
				Value: &core.Lambda{
					Params: []string{"x", "y"},
					Body: &core.BinOp{
						Op:    "+",
						Left:  &core.Var{Name: "x"},
						Right: &core.Var{Name: "y"},
					},
				},
				Body: &core.Var{Name: "add"},
			},
		},
		Meta: map[string]*core.DeclMeta{
			"add": {Name: "add", IsExport: true},
		},
	}

	gen := New("math")
	code, err := gen.Generate(prog)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	codeStr := string(code)

	// Should have PascalCase exported function
	if !strings.Contains(codeStr, "func Add(") {
		t.Errorf("Expected exported 'func Add(', got:\n%s", codeStr)
	}
}

func TestNonExportFunc_GeneratesCamelCase(t *testing.T) {
	// func helper(x) { x * 2 }
	prog := &core.Program{
		Decls: []core.CoreExpr{
			&core.Let{
				Name: "helper",
				Value: &core.Lambda{
					Params: []string{"x"},
					Body: &core.BinOp{
						Op:    "*",
						Left:  &core.Var{Name: "x"},
						Right: &core.Lit{Kind: core.IntLit, Value: int64(2)},
					},
				},
				Body: &core.Var{Name: "helper"},
			},
		},
		Meta: map[string]*core.DeclMeta{
			"helper": {Name: "helper", IsExport: false},
		},
	}

	gen := New("util")
	code, err := gen.Generate(prog)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	codeStr := string(code)

	// Should have camelCase private function
	if !strings.Contains(codeStr, "func helper(") {
		t.Errorf("Expected private 'func helper(', got:\n%s", codeStr)
	}

	// Should NOT have PascalCase
	if strings.Contains(codeStr, "func Helper(") {
		t.Errorf("Should not export private function, got:\n%s", codeStr)
	}
}
