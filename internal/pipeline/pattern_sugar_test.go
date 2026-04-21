package pipeline

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/elaborate"
	"github.com/sunholo-data/ailang/internal/lexer"
	"github.com/sunholo-data/ailang/internal/parser"
)

// TestPatternSugarIntegration tests parse → elaborate pipeline with cons sugar
func TestPatternSugarIntegration(t *testing.T) {
	source := `module test

func head(xs: List[int]) -> int {
  match xs {
    x :: xs => x,
    [] => 0
  }
}`

	// Parse
	l := lexer.New(source, "test.ail")
	p := parser.New(l)
	program := p.Parse()

	if len(p.Errors()) != 0 {
		t.Fatalf("Parse errors: %v", p.Errors())
	}

	// Elaborate
	elab := elaborate.NewElaborator()
	_, err := elab.Elaborate(program)
	if err != nil {
		t.Fatalf("Elaboration failed: %v", err)
	}
}

// TestPatternSugarRightAssociativeIntegration tests a :: b :: c syntax
func TestPatternSugarRightAssociativeIntegration(t *testing.T) {
	source := `module test

func sumTwo(xs: List[int]) -> int {
  match xs {
    a :: b :: c => a + b,
    _ => 0
  }
}`

	l := lexer.New(source, "test.ail")
	p := parser.New(l)
	program := p.Parse()

	if len(p.Errors()) != 0 {
		t.Fatalf("Parse errors: %v", p.Errors())
	}

	elab := elaborate.NewElaborator()
	_, err := elab.Elaborate(program)
	if err != nil {
		t.Fatalf("Elaboration failed: %v", err)
	}
}

// TestPatternSugarMixedFormsIntegration tests mixing sugar and canonical
func TestPatternSugarMixedFormsIntegration(t *testing.T) {
	source := `module test

func test(xs: List[int]) -> int {
  match xs {
    x :: [] => x,
    ::(a, ::(b, rest)) => a + b,
    _ => 0
  }
}`

	l := lexer.New(source, "test.ail")
	p := parser.New(l)
	program := p.Parse()

	if len(p.Errors()) != 0 {
		t.Fatalf("Parse errors: %v", p.Errors())
	}

	elab := elaborate.NewElaborator()
	_, err := elab.Elaborate(program)
	if err != nil {
		t.Fatalf("Elaboration failed: %v", err)
	}
}

// TestPatternSugarStrictModeIntegration tests that strict mode rejects sugar
func TestPatternSugarStrictModeIntegration(t *testing.T) {
	source := `module test

func head(xs: List[int]) -> int {
  match xs {
    x :: xs => x,
    [] => 0
  }
}`

	l := lexer.New(source, "test.ail")
	p := parser.New(l)
	p.SetStrictSyntaxMode(true) // Enable strict mode

	_ = p.Parse()

	// Should have parse errors in strict mode
	if len(p.Errors()) == 0 {
		t.Fatal("Expected parse errors in strict mode")
	}

	// Verify we got a sugar error
	found := false
	for _, err := range p.Errors() {
		errStr := err.Error()
		if len(errStr) >= 18 && errStr[:18] == "SUGAR_CONS_PATTERN" {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Expected SUGAR_CONS_PATTERN error in strict mode, got: %v", p.Errors())
	}
}

// TestPatternSugarCanonicalInStrictModeIntegration verifies canonical form works in strict mode
func TestPatternSugarCanonicalInStrictModeIntegration(t *testing.T) {
	source := `module test

func head(xs: List[int]) -> int {
  match xs {
    ::(x, xs) => x,
    [] => 0
  }
}`

	l := lexer.New(source, "test.ail")
	p := parser.New(l)
	p.SetStrictSyntaxMode(true) // Enable strict mode

	program := p.Parse()

	// Canonical form should work in strict mode
	if len(p.Errors()) != 0 {
		t.Fatalf("Strict mode should accept canonical form: %v", p.Errors())
	}

	elab := elaborate.NewElaborator()
	_, err := elab.Elaborate(program)
	if err != nil {
		t.Fatalf("Elaboration failed: %v", err)
	}
}
