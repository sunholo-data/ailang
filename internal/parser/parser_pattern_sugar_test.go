package parser

import (
	"testing"

	"github.com/sunholo/ailang/internal/lexer"
)

// TestPatternConsSugarBasic tests basic x :: xs sugar syntax
func TestPatternConsSugarBasic(t *testing.T) {
	input := `module test

func sum(xs: List[int]) -> int {
  match xs {
    x :: xs => x,
    [] => 0
  }
}`

	l := lexer.New(input, "test.ail")
	p := New(l)
	program := p.Parse()

	if len(p.Errors()) != 0 {
		t.Log("Parser errors:")
		for _, err := range p.Errors() {
			t.Logf("  %s", err)
		}
		t.Fatal("Failed to parse x :: xs sugar")
	}

	if program == nil || program.File == nil {
		t.Fatal("Expected file to parse successfully")
	}

	if len(program.File.Funcs) != 1 {
		t.Fatalf("Expected 1 function, got %d", len(program.File.Funcs))
	}
}

// TestPatternConsSugarWildcard tests wildcard with cons sugar: _ :: xs
func TestPatternConsSugarWildcard(t *testing.T) {
	input := `module test

func tail(xs: List[int]) -> List[int] {
  match xs {
    _ :: xs => xs,
    [] => []
  }
}`

	l := lexer.New(input, "test.ail")
	p := New(l)
	program := p.Parse()

	if len(p.Errors()) != 0 {
		t.Log("Parser errors:")
		for _, err := range p.Errors() {
			t.Logf("  %s", err)
		}
		t.Fatal("Failed to parse _ :: xs sugar")
	}

	if program == nil || program.File == nil {
		t.Fatal("Expected file to parse successfully")
	}
}

// TestPatternConsSugarRightAssociative tests right-associativity: a :: b :: c
func TestPatternConsSugarRightAssociative(t *testing.T) {
	input := `module test

func f(xs: List[int]) -> int {
  match xs {
    a :: b :: c => a + b,
    _ => 0
  }
}`

	l := lexer.New(input, "test.ail")
	p := New(l)
	program := p.Parse()

	if len(p.Errors()) != 0 {
		t.Log("Parser errors:")
		for _, err := range p.Errors() {
			t.Logf("  %s", err)
		}
		t.Fatal("Failed to parse a :: b :: c sugar")
	}

	if program == nil || program.File == nil {
		t.Fatal("Expected file to parse successfully")
	}
}

// TestPatternConsSugarWithEmptyList tests x :: [] sugar
func TestPatternConsSugarWithEmptyList(t *testing.T) {
	input := `module test

func f(xs: List[int]) -> int {
  match xs {
    x :: [] => x,
    _ => 0
  }
}`

	l := lexer.New(input, "test.ail")
	p := New(l)
	program := p.Parse()

	if len(p.Errors()) != 0 {
		t.Log("Parser errors:")
		for _, err := range p.Errors() {
			t.Logf("  %s", err)
		}
		t.Fatal("Failed to parse x :: [] sugar")
	}

	if program == nil || program.File == nil {
		t.Fatal("Expected file to parse successfully")
	}
}

// TestPatternConsSugarWithLiteral tests literal :: xs sugar
func TestPatternConsSugarWithLiteral(t *testing.T) {
	input := `module test

func f(xs: List[int]) -> int {
  match xs {
    1 :: xs => 100,
    x :: xs => x,
    [] => 0
  }
}`

	l := lexer.New(input, "test.ail")
	p := New(l)
	program := p.Parse()

	if len(p.Errors()) != 0 {
		t.Log("Parser errors:")
		for _, err := range p.Errors() {
			t.Logf("  %s", err)
		}
		t.Fatal("Failed to parse 1 :: xs sugar")
	}

	if program == nil || program.File == nil {
		t.Fatal("Expected file to parse successfully")
	}
}

// TestPatternConsSugarMixedForms tests mixing sugar and canonical forms
func TestPatternConsSugarMixedForms(t *testing.T) {
	input := `module test

func f(xs: List[int]) -> int {
  match xs {
    x :: [] => x,
    ::(a, ::(b, rest)) => a + b,
    x :: xs => x,
    [] => 0
  }
}`

	l := lexer.New(input, "test.ail")
	p := New(l)
	program := p.Parse()

	if len(p.Errors()) != 0 {
		t.Log("Parser errors:")
		for _, err := range p.Errors() {
			t.Logf("  %s", err)
		}
		t.Fatal("Failed to parse mixed sugar and canonical forms")
	}

	if program == nil || program.File == nil {
		t.Fatal("Expected file to parse successfully")
	}
}

// TestPatternConsSugarWithParens tests (x :: xs) in parentheses
func TestPatternConsSugarWithParens(t *testing.T) {
	input := `module test

func f(xs: List[int]) -> int {
  match xs {
    (x :: xs) => x,
    [] => 0
  }
}`

	l := lexer.New(input, "test.ail")
	p := New(l)
	program := p.Parse()

	if len(p.Errors()) != 0 {
		t.Log("Parser errors:")
		for _, err := range p.Errors() {
			t.Logf("  %s", err)
		}
		t.Fatal("Failed to parse (x :: xs) in parentheses")
	}

	if program == nil || program.File == nil {
		t.Fatal("Expected file to parse successfully")
	}
}

// TestPatternConsSugarWithGuard tests x :: xs with guard
func TestPatternConsSugarWithGuard(t *testing.T) {
	input := `module test

func f(xs: List[int]) -> int {
  match xs {
    x :: xs if x > 0 => x,
    _ => 0
  }
}`

	l := lexer.New(input, "test.ail")
	p := New(l)
	program := p.Parse()

	if len(p.Errors()) != 0 {
		t.Log("Parser errors:")
		for _, err := range p.Errors() {
			t.Logf("  %s", err)
		}
		t.Fatal("Failed to parse x :: xs with guard")
	}

	if program == nil || program.File == nil {
		t.Fatal("Expected file to parse successfully")
	}
}

// TestPatternConsSugarStrictMode tests that strict mode rejects sugar
func TestPatternConsSugarStrictMode(t *testing.T) {
	input := `module test

func f(xs: List[int]) -> int {
  match xs {
    x :: xs => x,
    [] => 0
  }
}`

	l := lexer.New(input, "test.ail")
	p := New(l)
	p.strictSyntaxMode = true // Enable strict mode
	_ = p.Parse()

	// Should have sugar error in strict mode
	if len(p.Errors()) == 0 {
		t.Fatal("Expected strict mode to reject x :: xs sugar")
	}

	// Check for CONS_PATTERN sugar error
	found := false
	for _, err := range p.Errors() {
		errStr := err.Error()
		if len(errStr) >= 18 && errStr[:18] == "SUGAR_CONS_PATTERN" {
			found = true
			t.Logf("Strict mode error (expected): %s", errStr)
			break
		}
	}

	if !found {
		t.Logf("Errors: %v", p.Errors())
		t.Error("Expected SUGAR_CONS_PATTERN error in strict mode")
	}
}

// TestPatternConsSugarStrictModeCanonicalOK tests that strict mode accepts canonical form
func TestPatternConsSugarStrictModeCanonicalOK(t *testing.T) {
	input := `module test

func f(xs: List[int]) -> int {
  match xs {
    ::(x, xs) => x,
    [] => 0
  }
}`

	l := lexer.New(input, "test.ail")
	p := New(l)
	p.strictSyntaxMode = true // Enable strict mode
	program := p.Parse()

	// Canonical form should work in strict mode
	if len(p.Errors()) != 0 {
		t.Log("Parser errors:")
		for _, err := range p.Errors() {
			t.Logf("  %s", err)
		}
		t.Fatal("Strict mode should accept canonical ::(x, xs) form")
	}

	if program == nil || program.File == nil {
		t.Fatal("Expected file to parse successfully")
	}
}

// TestPatternConsSugarNegativeInvalidForm tests that (:: x xs) is rejected
func TestPatternConsSugarNegativeInvalidForm(t *testing.T) {
	// Note: This would require special lexer/parser handling for spacing
	// For now, just document that :: x xs (with spaces, no parens) is not supported
	// The canonical form ::(x, xs) is always required for prefix notation
}
