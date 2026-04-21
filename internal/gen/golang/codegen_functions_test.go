package golang

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/core"
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

	// M-ZERO-ARG: The typed wrapper should have NO parameter for unit-typed params
	// and should pass struct{}{} to the _impl function
	if strings.Contains(codeStr, "func TileFloor(_unused0") {
		t.Errorf("Wrapper should not expose unit param in public API, got:\n%s", codeStr)
	}
	if !strings.Contains(codeStr, "func TileFloor() interface{}") {
		t.Errorf("Expected wrapper with no parameters, got:\n%s", codeStr)
	}
	if !strings.Contains(codeStr, "return tileFloor_impl(struct{}{})") {
		t.Errorf("Expected wrapper to pass struct{}{} to _impl, got:\n%s", codeStr)
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
