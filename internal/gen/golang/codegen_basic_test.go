package golang

import (
	"strings"
	"testing"

	"github.com/sunholo/ailang/internal/core"
)

func TestGenerateBinaryOperators(t *testing.T) {
	tests := []struct {
		name     string
		op       string
		expected string
	}{
		{"add", "+", "+"},
		{"sub", "-", "-"},
		{"mul", "*", "*"},
		{"div", "/", "/"},
		{"eq", "==", "=="},
		{"ne", "!=", "!="},
		{"lt", "<", "<"},
		{"gt", ">", ">"},
		{"le", "<=", "<="},
		{"ge", ">=", ">="},
		{"and", "&&", "&&"},
		{"or", "||", "||"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prog := &core.Program{
				Decls: []core.CoreExpr{
					&core.Let{
						Name: "test",
						Value: &core.Lambda{
							Params: []string{"a", "b"},
							Body: &core.BinOp{
								Op:    tt.op,
								Left:  &core.Var{Name: "a"},
								Right: &core.Var{Name: "b"},
							},
						},
						Body: &core.Var{Name: "test"},
					},
				},
			}

			gen := New("ops")
			code, err := gen.Generate(prog)
			if err != nil {
				t.Fatalf("Generate failed: %v", err)
			}

			if !strings.Contains(string(code), tt.expected) {
				t.Errorf("Missing %s operator in output", tt.expected)
			}
		})
	}
}

func TestGenerateLiterals(t *testing.T) {
	tests := []struct {
		name     string
		lit      *core.Lit
		expected string
	}{
		{"int", &core.Lit{Kind: core.IntLit, Value: int64(42)}, "42"},
		{"float", &core.Lit{Kind: core.FloatLit, Value: 3.14}, "3.14"},
		{"bool_true", &core.Lit{Kind: core.BoolLit, Value: true}, "true"},
		{"bool_false", &core.Lit{Kind: core.BoolLit, Value: false}, "false"},
		{"string", &core.Lit{Kind: core.StringLit, Value: "hello"}, `"hello"`},
		{"unit", &core.Lit{Kind: core.UnitLit, Value: nil}, "struct{}{}"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prog := &core.Program{
				Decls: []core.CoreExpr{
					&core.Let{
						Name:  "x",
						Value: tt.lit,
						Body:  &core.Var{Name: "x"},
					},
				},
			}

			gen := New("test")
			code, err := gen.Generate(prog)
			if err != nil {
				t.Fatalf("Generate failed: %v", err)
			}

			if !strings.Contains(string(code), tt.expected) {
				t.Errorf("Missing %s in output, got:\n%s", tt.expected, string(code))
			}
		})
	}
}

func TestGenerateIfExpression(t *testing.T) {
	// if true then 1 else 2
	prog := &core.Program{
		Decls: []core.CoreExpr{
			&core.Let{
				Name: "test",
				Value: &core.Lambda{
					Params: []string{"x"},
					Body: &core.If{
						Cond: &core.Var{Name: "x"},
						Then: &core.Lit{Kind: core.IntLit, Value: int64(1)},
						Else: &core.Lit{Kind: core.IntLit, Value: int64(2)},
					},
				},
				Body: &core.Var{Name: "test"},
			},
		},
	}

	gen := New("test")
	code, err := gen.Generate(prog)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	codeStr := string(code)

	// Check for if statement
	if !strings.Contains(codeStr, "if") {
		t.Errorf("Missing if statement")
	}

	// Check for both branches
	// M-DX17: Integer literals are now wrapped in int64()
	if !strings.Contains(codeStr, "return int64(1)") {
		t.Errorf("Missing then branch")
	}
	if !strings.Contains(codeStr, "return int64(2)") {
		t.Errorf("Missing else branch")
	}
}

func TestGenerateUnaryOperators(t *testing.T) {
	tests := []struct {
		name     string
		op       string
		expected string
	}{
		{"neg", "-", "-"},
		{"not", "!", "!"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prog := &core.Program{
				Decls: []core.CoreExpr{
					&core.Let{
						Name: "test",
						Value: &core.Lambda{
							Params: []string{"x"},
							Body: &core.UnOp{
								Op:      tt.op,
								Operand: &core.Var{Name: "x"},
							},
						},
						Body: &core.Var{Name: "test"},
					},
				},
			}

			gen := New("ops")
			code, err := gen.Generate(prog)
			if err != nil {
				t.Fatalf("Generate failed: %v", err)
			}

			if !strings.Contains(string(code), tt.expected) {
				t.Errorf("Missing %s operator in output, got:\n%s", tt.expected, string(code))
			}
		})
	}
}
