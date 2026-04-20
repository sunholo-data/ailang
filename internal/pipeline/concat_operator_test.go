package pipeline

import (
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

// TestConcatListWithSignature tests list concatenation with type signature
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

// TestConcatConcreteList tests list concatenation with concrete lists
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

// TestConcatListVarPlusConcreteList tests type variable + concrete list
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
