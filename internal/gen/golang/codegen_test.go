package golang

import (
	"strings"
	"testing"

	"github.com/sunholo/ailang/internal/core"
)

func TestGeneratePureFunction_Factorial(t *testing.T) {
	// let rec factorial = \n. if n <= 1 then 1 else n * factorial(n - 1)
	prog := &core.Program{
		Decls: []core.CoreExpr{
			&core.LetRec{
				Bindings: []core.RecBinding{
					{
						Name: "factorial",
						Value: &core.Lambda{
							Params: []string{"n"},
							Body: &core.If{
								Cond: &core.BinOp{
									Op:    "<=",
									Left:  &core.Var{Name: "n"},
									Right: &core.Lit{Kind: core.IntLit, Value: int64(1)},
								},
								Then: &core.Lit{Kind: core.IntLit, Value: int64(1)},
								Else: &core.BinOp{
									Op:   "*",
									Left: &core.Var{Name: "n"},
									Right: &core.App{
										Func: &core.Var{Name: "factorial"},
										Args: []core.CoreExpr{
											&core.BinOp{
												Op:    "-",
												Left:  &core.Var{Name: "n"},
												Right: &core.Lit{Kind: core.IntLit, Value: int64(1)},
											},
										},
									},
								},
							},
						},
					},
				},
				Body: &core.Var{Name: "factorial"},
			},
		},
	}

	gen := New("game")
	code, err := gen.Generate(prog)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	codeStr := string(code)

	// Check for function declaration (private by default until A4 export syntax)
	if !strings.Contains(codeStr, "func factorial") {
		t.Errorf("Missing factorial function, got:\n%s", codeStr)
	}

	// Check for recursive call
	if !strings.Contains(codeStr, "factorial") || !strings.Contains(codeStr, "(") {
		t.Errorf("Missing recursive call pattern")
	}

	// Verify it's valid Go (go/format passed)
	if !strings.Contains(codeStr, "package game") {
		t.Errorf("Missing package declaration")
	}
}

func TestGeneratePureFunction_SimpleLet(t *testing.T) {
	// let double = \x. x * 2
	prog := &core.Program{
		Decls: []core.CoreExpr{
			&core.Let{
				Name: "double",
				Value: &core.Lambda{
					Params: []string{"x"},
					Body: &core.BinOp{
						Op:    "*",
						Left:  &core.Var{Name: "x"},
						Right: &core.Lit{Kind: core.IntLit, Value: int64(2)},
					},
				},
				Body: &core.Var{Name: "double"},
			},
		},
	}

	gen := New("math")
	code, err := gen.Generate(prog)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	codeStr := string(code)

	// Check for function (private by default until A4 export syntax)
	if !strings.Contains(codeStr, "func double") {
		t.Errorf("Missing double function, got:\n%s", codeStr)
	}

	// Check for multiplication
	if !strings.Contains(codeStr, "*") {
		t.Errorf("Missing multiplication operator")
	}
}

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
	if !strings.Contains(codeStr, "return 1") {
		t.Errorf("Missing then branch")
	}
	if !strings.Contains(codeStr, "return 2") {
		t.Errorf("Missing else branch")
	}
}

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
	if !strings.Contains(codeStr, "x :=") {
		t.Errorf("Missing x binding, got:\n%s", codeStr)
	}
	if !strings.Contains(codeStr, "y :=") {
		t.Errorf("Missing y binding, got:\n%s", codeStr)
	}
}

func TestGenerateFunctionApplication(t *testing.T) {
	// f(1, 2)
	prog := &core.Program{
		Decls: []core.CoreExpr{
			&core.Let{
				Name: "result",
				Value: &core.App{
					Func: &core.Var{Name: "f"},
					Args: []core.CoreExpr{
						&core.Lit{Kind: core.IntLit, Value: int64(1)},
						&core.Lit{Kind: core.IntLit, Value: int64(2)},
					},
				},
				Body: &core.Var{Name: "result"},
			},
		},
	}

	gen := New("test")
	code, err := gen.Generate(prog)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	codeStr := string(code)

	// Check for function call syntax - with CallFunc wrapper for lambda variables
	if !strings.Contains(codeStr, "1, 2)") {
		t.Errorf("Missing function call with args, got:\n%s", codeStr)
	}
}

func TestGenerateLambdaExpression(t *testing.T) {
	// \x. x + 1
	prog := &core.Program{
		Decls: []core.CoreExpr{
			&core.Let{
				Name: "inc",
				Value: &core.Lambda{
					Params: []string{"x"},
					Body: &core.BinOp{
						Op:    "+",
						Left:  &core.Var{Name: "x"},
						Right: &core.Lit{Kind: core.IntLit, Value: int64(1)},
					},
				},
				Body: &core.Var{Name: "inc"},
			},
		},
	}

	gen := New("test")
	code, err := gen.Generate(prog)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	codeStr := string(code)

	// Check for anonymous function
	if !strings.Contains(codeStr, "func") {
		t.Errorf("Missing func keyword")
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

func TestCodePassesGoVet(t *testing.T) {
	// Generate a complex program and verify go/format accepted it
	// (go/format validates syntax, which is a subset of vet)
	prog := &core.Program{
		Decls: []core.CoreExpr{
			&core.LetRec{
				Bindings: []core.RecBinding{
					{
						Name: "sum",
						Value: &core.Lambda{
							Params: []string{"n"},
							Body: &core.If{
								Cond: &core.BinOp{
									Op:    "<=",
									Left:  &core.Var{Name: "n"},
									Right: &core.Lit{Kind: core.IntLit, Value: int64(0)},
								},
								Then: &core.Lit{Kind: core.IntLit, Value: int64(0)},
								Else: &core.BinOp{
									Op:   "+",
									Left: &core.Var{Name: "n"},
									Right: &core.App{
										Func: &core.Var{Name: "sum"},
										Args: []core.CoreExpr{
											&core.BinOp{
												Op:    "-",
												Left:  &core.Var{Name: "n"},
												Right: &core.Lit{Kind: core.IntLit, Value: int64(1)},
											},
										},
									},
								},
							},
						},
					},
				},
				Body: &core.Var{Name: "sum"},
			},
		},
	}

	gen := New("valid")
	code, err := gen.Generate(prog)
	if err != nil {
		t.Fatalf("Generate failed: %v\nThis indicates invalid Go syntax was generated", err)
	}

	// If we reach here, go/format passed which means syntax is valid
	if !strings.Contains(string(code), "package valid") {
		t.Errorf("Invalid package declaration")
	}
}

func TestGenerateMatch_ConstructorPattern(t *testing.T) {
	// match x with
	//   | Some(v) -> v
	//   | None -> 0
	prog := &core.Program{
		Decls: []core.CoreExpr{
			&core.Let{
				Name: "getValue",
				Value: &core.Lambda{
					Params: []string{"x"},
					Body: &core.Match{
						Scrutinee:  &core.Var{Name: "x"},
						Exhaustive: true,
						Arms: []core.MatchArm{
							{
								Pattern: &core.ConstructorPattern{
									Name: "Some",
									Args: []core.CorePattern{&core.VarPattern{Name: "v"}},
								},
								Body: &core.Var{Name: "v"},
							},
							{
								Pattern: &core.ConstructorPattern{
									Name: "None",
									Args: []core.CorePattern{},
								},
								Body: &core.Lit{Kind: core.IntLit, Value: int64(0)},
							},
						},
					},
				},
				Body: &core.Var{Name: "getValue"},
			},
		},
	}

	gen := New("test")
	code, err := gen.Generate(prog)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	codeStr := string(code)

	// Should have switch statement
	if !strings.Contains(codeStr, "switch") {
		t.Errorf("Missing switch statement, got:\n%s", codeStr)
	}

	// Should have case clauses
	if !strings.Contains(codeStr, "case") {
		t.Errorf("Missing case clause, got:\n%s", codeStr)
	}
}

func TestGenerateMatch_WildcardPattern(t *testing.T) {
	// match x with
	//   | _ -> 42
	prog := &core.Program{
		Decls: []core.CoreExpr{
			&core.Let{
				Name: "always42",
				Value: &core.Lambda{
					Params: []string{"x"},
					Body: &core.Match{
						Scrutinee:  &core.Var{Name: "x"},
						Exhaustive: true,
						Arms: []core.MatchArm{
							{
								Pattern: &core.WildcardPattern{},
								Body:    &core.Lit{Kind: core.IntLit, Value: int64(42)},
							},
						},
					},
				},
				Body: &core.Var{Name: "always42"},
			},
		},
	}

	gen := New("test")
	code, err := gen.Generate(prog)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	codeStr := string(code)

	// Should contain 42 as the result
	if !strings.Contains(codeStr, "42") {
		t.Errorf("Missing result value, got:\n%s", codeStr)
	}
}

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

func TestExportRecursiveFunc(t *testing.T) {
	// export func factorial(n) { if n <= 1 then 1 else n * factorial(n-1) }
	prog := &core.Program{
		Decls: []core.CoreExpr{
			&core.LetRec{
				Bindings: []core.RecBinding{
					{
						Name: "factorial",
						Value: &core.Lambda{
							Params: []string{"n"},
							Body: &core.If{
								Cond: &core.BinOp{
									Op:    "<=",
									Left:  &core.Var{Name: "n"},
									Right: &core.Lit{Kind: core.IntLit, Value: int64(1)},
								},
								Then: &core.Lit{Kind: core.IntLit, Value: int64(1)},
								Else: &core.BinOp{
									Op:   "*",
									Left: &core.Var{Name: "n"},
									Right: &core.App{
										Func: &core.Var{Name: "factorial"},
										Args: []core.CoreExpr{
											&core.BinOp{
												Op:    "-",
												Left:  &core.Var{Name: "n"},
												Right: &core.Lit{Kind: core.IntLit, Value: int64(1)},
											},
										},
									},
								},
							},
						},
					},
				},
				Body: &core.Var{Name: "factorial"},
			},
		},
		Meta: map[string]*core.DeclMeta{
			"factorial": {Name: "factorial", IsExport: true},
		},
	}

	gen := New("math")
	code, err := gen.Generate(prog)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	codeStr := string(code)

	// Should have PascalCase exported function
	if !strings.Contains(codeStr, "func Factorial(") {
		t.Errorf("Expected exported 'func Factorial(', got:\n%s", codeStr)
	}
}
