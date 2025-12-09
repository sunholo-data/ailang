package golang

import (
	"strings"
	"testing"

	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/types"
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
	// M-DX17: Integer literals are now wrapped in int64()
	if !strings.Contains(codeStr, "return int64(1)") {
		t.Errorf("Missing then branch")
	}
	if !strings.Contains(codeStr, "return int64(2)") {
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
	// M-DX13.3: Uses "var x interface{}" to allow type assertions on concrete values
	if !strings.Contains(codeStr, "var x interface{} =") {
		t.Errorf("Missing x binding, got:\n%s", codeStr)
	}
	if !strings.Contains(codeStr, "var y interface{} =") {
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
	// M-DX17: Integer literals are now wrapped in int64()
	if !strings.Contains(codeStr, "int64(1), int64(2))") {
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
	// Register ADT constructors so the generator can determine the parent type
	gen.RegisterADTConstructor("Option", "Some", 1)
	gen.RegisterADTConstructor("Option", "None", 0)

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

	// Should have proper Kind-based case (not type-based)
	if !strings.Contains(codeStr, "OptionKind") {
		t.Errorf("Missing OptionKind in case clause, got:\n%s", codeStr)
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

// M-DX12: Test ADT slice converter generation
func TestGenerateADTSliceConverter(t *testing.T) {
	// Register an ADT type for slice conversion
	prog := &core.Program{
		Decls: []core.CoreExpr{},
	}

	gen := New("game")
	// Register ADT slice type - this is what happens when [DrawCmd] is encountered
	gen.RegisterADTSliceType("DrawCmd")
	gen.RegisterADTSliceType("Camera")

	code, err := gen.Generate(prog)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	codeStr := string(code)

	// Should have DrawCmd converter (M-BUGFIX: exported with capital C)
	if !strings.Contains(codeStr, "func ConvertToDrawCmdSlice(v interface{}) []*DrawCmd") {
		t.Errorf("Missing ConvertToDrawCmdSlice function, got:\n%s", codeStr)
	}

	// Should have Camera converter (M-BUGFIX: exported with capital C)
	if !strings.Contains(codeStr, "func ConvertToCameraSlice(v interface{}) []*Camera") {
		t.Errorf("Missing ConvertToCameraSlice function, got:\n%s", codeStr)
	}

	// Should have fail-fast panic
	if !strings.Contains(codeStr, "panic(fmt.Sprintf") {
		t.Errorf("Missing panic for fail-fast, got:\n%s", codeStr)
	}

	// Should have empty slice handling
	if !strings.Contains(codeStr, "[]*DrawCmd{}") {
		t.Errorf("Missing empty slice return, got:\n%s", codeStr)
	}

	// Should be deterministic (sorted alphabetically)
	cameraIdx := strings.Index(codeStr, "ConvertToCameraSlice")
	drawCmdIdx := strings.Index(codeStr, "ConvertToDrawCmdSlice")
	if cameraIdx == -1 || drawCmdIdx == -1 {
		t.Errorf("Missing converters")
	}
	if cameraIdx > drawCmdIdx {
		t.Errorf("Converters should be sorted alphabetically (Camera before DrawCmd), got Camera at %d, DrawCmd at %d", cameraIdx, drawCmdIdx)
	}
}

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
	coreTypeInfo[100] = &types.TCon{Name: "bool"}                                               // Let expression (body) type
	coreTypeInfo[101] = &types.TFunc{Params: []types.Type{}, Return: &types.TCon{Name: "bool"}} // Lambda type for wrapper
	coreTypeInfo[102] = &types.TCon{Name: "bool"}                                               // Value expression type

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
	coreTypeInfo[200] = &types.TCon{Name: "int"}                                               // Let expression (body) type
	coreTypeInfo[201] = &types.TCon{Name: "int"}                                               // Value expression type (BinOp)
	coreTypeInfo[202] = &types.TFunc{Params: []types.Type{}, Return: &types.TCon{Name: "int"}} // Lambda type for wrapper

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

	// Should fall back to interface{}
	if !strings.Contains(codeStr, "func() interface{} {") {
		t.Errorf("Expected fallback 'func() interface{} {', got:\n%s", codeStr)
	}

	if !strings.Contains(codeStr, "var x interface{} =") {
		t.Errorf("Expected fallback 'var x interface{} =', got:\n%s", codeStr)
	}
}

// TestBlankIdentifierParameter tests that blank identifiers (_) in function parameters
// are replaced with _unused0, _unused1, etc. to allow them to be passed as arguments.
// M-BUGFIX: This was causing "cannot use _ as value" errors in generated Go code.
func TestBlankIdentifierParameter(t *testing.T) {
	// let tileFloor = \_ . 42
	// Exported function with _ parameter
	prog := &core.Program{
		Decls: []core.CoreExpr{
			&core.Let{
				Name: "tileFloor",
				Value: &core.Lambda{
					Params: []string{"_"},
					Body:   &core.Lit{Kind: core.IntLit, Value: int64(42)},
				},
				Body: &core.Var{Name: "tileFloor"},
			},
		},
		Meta: map[string]*core.DeclMeta{
			"tileFloor": {IsExport: true},
		},
	}

	gen := New("test")
	code, err := gen.Generate(prog)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	codeStr := string(code)

	// The _impl function should have _unused0 parameter, not _
	if strings.Contains(codeStr, "tileFloor_impl(_ interface{})") {
		t.Errorf("_impl should not have bare _ parameter, got:\n%s", codeStr)
	}
	if !strings.Contains(codeStr, "tileFloor_impl(_unused0 interface{})") {
		t.Errorf("Expected _impl with _unused0 parameter, got:\n%s", codeStr)
	}

	// The typed wrapper should also have _unused0 and pass it correctly
	if strings.Contains(codeStr, "return tileFloor_impl(_)") {
		t.Errorf("Wrapper should not pass bare _ as argument (invalid Go), got:\n%s", codeStr)
	}
	if !strings.Contains(codeStr, "return tileFloor_impl(_unused0)") {
		t.Errorf("Expected wrapper to pass _unused0 as argument, got:\n%s", codeStr)
	}
}

// TestMultipleBlankIdentifiers tests that multiple _ parameters get unique names.
func TestMultipleBlankIdentifiers(t *testing.T) {
	// let f = \_ \_ . 42
	prog := &core.Program{
		Decls: []core.CoreExpr{
			&core.Let{
				Name: "f",
				Value: &core.Lambda{
					Params: []string{"_", "_"},
					Body:   &core.Lit{Kind: core.IntLit, Value: int64(42)},
				},
				Body: &core.Var{Name: "f"},
			},
		},
		Meta: map[string]*core.DeclMeta{
			"f": {IsExport: true},
		},
	}

	gen := New("test")
	code, err := gen.Generate(prog)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	codeStr := string(code)

	// Should have _unused0 and _unused1
	if !strings.Contains(codeStr, "_unused0") {
		t.Errorf("Expected _unused0 parameter, got:\n%s", codeStr)
	}
	if !strings.Contains(codeStr, "_unused1") {
		t.Errorf("Expected _unused1 parameter, got:\n%s", codeStr)
	}
}

// TestRecordReturnType tests M-BUGFIX: Functions returning record types should
// generate correct Go return types (e.g., *BridgeState instead of struct{}).
func TestRecordReturnType(t *testing.T) {
	// Create a Lambda that "returns a record type"
	// We need to set up CoreTypeInfo to indicate the return type is a TRecord
	lam := &core.Lambda{
		CoreNode: core.CoreNode{NodeID: 1},
		Params:   []string{"x"},
		Body:     &core.Lit{Kind: core.IntLit, Value: int64(42)}, // placeholder body
	}

	prog := &core.Program{
		Decls: []core.CoreExpr{
			&core.Let{
				Name:  "initBridge",
				Value: lam,
				Body:  &core.Var{Name: "initBridge"},
			},
		},
		Meta: map[string]*core.DeclMeta{
			"initBridge": {IsExport: true},
		},
	}

	gen := New("test")

	// Register a record type (simulating type BridgeState = { x: int, y: int })
	gen.RegisterRecordType("BridgeState", []string{"X", "Y"}, map[string]string{
		"X": "int64",
		"Y": "int64",
	})

	// Set up CoreTypeInfo with the Lambda returning a TRecord with matching fields
	cti := types.CoreTypeInfo{
		1: &types.TFunc{
			Params: []types.Type{&types.TCon{Name: "int"}},
			Return: &types.TRecord{
				Fields: map[string]types.Type{
					"x": &types.TCon{Name: "int"},
					"y": &types.TCon{Name: "int"},
				},
			},
		},
	}
	gen.SetCoreTypeInfo(cti)

	code, err := gen.Generate(prog)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	codeStr := string(code)

	// The typed wrapper should have *BridgeState as return type
	// Check for the specific function signature
	if !strings.Contains(codeStr, "func InitBridge(x int64) *BridgeState") {
		t.Errorf("Expected 'func InitBridge(x int64) *BridgeState', got:\n%s", codeStr)
	}
	// Make sure the return type is not struct{} in the InitBridge function
	if strings.Contains(codeStr, "func InitBridge(x int64) struct{}") {
		t.Errorf("Expected *BridgeState return type, but got struct{} in InitBridge")
	}
}
