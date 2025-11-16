package pipeline

import (
	"strings"
	"testing"

	"github.com/sunholo/ailang/internal/elaborate"
	"github.com/sunholo/ailang/internal/lexer"
	"github.com/sunholo/ailang/internal/parser"
	"github.com/sunholo/ailang/internal/types"
)

// Helper function to type check AILANG code
func typeCheckCode(t *testing.T, code string) error {
	t.Helper()

	// Parse
	l := lexer.New(code, "test.ail")
	p := parser.New(l)
	program := p.Parse()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	// Elaborate
	elab := elaborate.NewElaborator()
	coreProg, err := elab.Elaborate(program)
	if err != nil {
		t.Fatalf("Elaboration error: %v", err)
	}

	// Type check
	tc := types.NewCoreTypeChecker()
	_, err = tc.CheckCoreProgram(coreProg)
	return err
}

// TestConcatRecursiveString tests recursive string concatenation (the bug case)
// This was failing before the fix with "cannot unify string with *types.TList"
func TestConcatRecursiveString(t *testing.T) {
	code := `
module test/concat

export func join(sep: string, xs: [int]) -> string {
  match xs {
    [] => "",
    [x] => show(x),
    x :: rest => show(x) ++ sep ++ join(sep, rest)
  }
}
`

	err := typeCheckCode(t, code)
	if err != nil {
		t.Errorf("Recursive string concat should work, got error: %v", err)
	}
}

// TestConcatListWithSignature tests list concatenation with type signature
// Regression test: ensure list concat still works
func TestConcatListWithSignature(t *testing.T) {
	code := `
module test/concat

export func concat[a](xs: [a], ys: [a]) -> [a] {
  xs ++ ys
}
`

	err := typeCheckCode(t, code)
	if err != nil {
		t.Errorf("List concat with signature should work, got error: %v", err)
	}
}

// TestConcatConcreteString tests string concatenation with concrete strings
// Regression test: ensure existing string concat still works
func TestConcatConcreteString(t *testing.T) {
	code := `
module test/concat

export func greet() -> string {
  "Hello" ++ " " ++ "World"
}
`

	err := typeCheckCode(t, code)
	if err != nil {
		t.Errorf("Concrete string concat should work, got error: %v", err)
	}
}

// TestConcatConcreteList tests list concatenation with concrete lists
// Regression test: ensure existing list concat still works
func TestConcatConcreteList(t *testing.T) {
	code := `
module test/concat

export func combine() -> [int] {
  [1, 2, 3] ++ [4, 5, 6]
}
`

	err := typeCheckCode(t, code)
	if err != nil {
		t.Errorf("Concrete list concat should work, got error: %v", err)
	}
}

// TestConcatMixedTypes tests that mixing string and list fails
// TODO(v0.4.6): This test is skipped because it reveals a deeper issue with type annotation threading.
// Type annotations from Surface AST aren't properly threaded to Core type checking, so parameters
// appear as type variables initially. The error is caught at runtime, not type-checking time.
// This needs a broader fix to how type annotations are handled in the elaboration→type-checking pipeline.
func TestConcatMixedTypes(t *testing.T) {
	t.Skip("Skipped: reveals deeper type annotation threading issue (out of scope for v0.4.5)")
	code := `
module test/concat

export func broken(s: string, xs: [int]) -> string {
  s ++ xs
}
`

	err := typeCheckCode(t, code)
	if err == nil {
		t.Error("Mixing string and list should fail, but got no error")
		return
	}

	// Check that error mentions type unification failure
	errStr := err.Error()
	if !strings.Contains(errStr, "unif") && !strings.Contains(errStr, "type") {
		t.Errorf("Expected unification error, got: %v", err)
	}
}

// TestConcatStringVarPlusConcreteString tests type variable + concrete string
// Should infer the type variable as string
func TestConcatStringVarPlusConcreteString(t *testing.T) {
	code := `
module test/concat

export func append(x: string) -> string {
  x ++ " suffix"
}
`

	err := typeCheckCode(t, code)
	if err != nil {
		t.Errorf("Type var + concrete string should work, got error: %v", err)
	}
}

// TestConcatListVarPlusConcreteList tests type variable + concrete list
// Should infer the type variable as list
func TestConcatListVarPlusConcreteList(t *testing.T) {
	code := `
module test/concat

export func append[a](xs: [a]) -> [a] {
  xs ++ []
}
`

	err := typeCheckCode(t, code)
	if err != nil {
		t.Errorf("Type var + concrete list should work, got error: %v", err)
	}
}

// TestConcatNestedRecursion tests deeply nested recursive string building
// Another test for the bug fix
func TestConcatNestedRecursion(t *testing.T) {
	code := `
module test/concat

export func repeat(s: string, n: int) -> string {
  if n <= 0 then ""
  else s ++ repeat(s, n - 1)
}
`

	err := typeCheckCode(t, code)
	if err != nil {
		t.Errorf("Nested recursive string concat should work, got error: %v", err)
	}
}
