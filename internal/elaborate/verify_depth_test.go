package elaborate

import (
	"testing"

	"github.com/sunholo/ailang/internal/lexer"
	"github.com/sunholo/ailang/internal/parser"
)

// TestVerifyDepthElaboration tests that per-function @verify(depth: N)
// attribute propagates from AST to Core DeclMeta during elaboration.
func TestVerifyDepthElaboration(t *testing.T) {
	input := `module test

@verify(depth: 5)
export func fib(n: int) -> int ! {} {
  if n <= 1 then n else fib(n - 1) + fib(n - 2)
}

export func add(x: int, y: int) -> int ! {} { x + y }
`

	l := lexer.New(input, "test.ail")
	p := parser.New(l)
	file := p.ParseFile()

	if len(p.Errors()) > 0 {
		for _, err := range p.Errors() {
			t.Errorf("parser error: %v", err)
		}
		t.FailNow()
	}

	// Elaborate
	elab := NewElaborator()
	prog, err := elab.ElaborateFile(file)
	if err != nil {
		t.Fatalf("elaboration error: %v", err)
	}

	// Check fib has VerifyDepth=5
	fibMeta, ok := prog.Meta["fib"]
	if !ok {
		t.Fatal("expected DeclMeta for 'fib', not found")
	}
	if fibMeta.VerifyDepth != 5 {
		t.Errorf("expected fib VerifyDepth=5, got %d", fibMeta.VerifyDepth)
	}

	// Check add has VerifyDepth=0 (default)
	addMeta, ok := prog.Meta["add"]
	if !ok {
		t.Fatal("expected DeclMeta for 'add', not found")
	}
	if addMeta.VerifyDepth != 0 {
		t.Errorf("expected add VerifyDepth=0 (default), got %d", addMeta.VerifyDepth)
	}
}

// TestVerifyDepthElaboration_MultipleWithContracts tests that @verify(depth: N)
// works alongside requires/ensures contracts.
func TestVerifyDepthElaboration_MultipleWithContracts(t *testing.T) {
	input := `module test

@verify(depth: 3)
export func safe_inc(x: int) -> int ! {}
requires { x >= 0 }
ensures { result > x }
{
  x + 1
}
`

	l := lexer.New(input, "test.ail")
	p := parser.New(l)
	file := p.ParseFile()

	if len(p.Errors()) > 0 {
		for _, err := range p.Errors() {
			t.Errorf("parser error: %v", err)
		}
		t.FailNow()
	}

	elab := NewElaborator()
	prog, err := elab.ElaborateFile(file)
	if err != nil {
		t.Fatalf("elaboration error: %v", err)
	}

	meta, ok := prog.Meta["safe_inc"]
	if !ok {
		t.Fatal("expected DeclMeta for 'safe_inc', not found")
	}

	// Check depth
	if meta.VerifyDepth != 3 {
		t.Errorf("expected VerifyDepth=3, got %d", meta.VerifyDepth)
	}

	// Check contracts are still present
	if len(meta.Contracts) != 2 {
		t.Errorf("expected 2 contracts (requires + ensures), got %d", len(meta.Contracts))
	}
}
