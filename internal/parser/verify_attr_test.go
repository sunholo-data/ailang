package parser

import (
	"strings"
	"testing"

	"github.com/sunholo/ailang/internal/ast"
	"github.com/sunholo/ailang/internal/lexer"
)

// TestVerifyAttribute_BasicDepth tests that @verify(depth: 5) parses and
// attaches to the next function declaration.
func TestVerifyAttribute_BasicDepth(t *testing.T) {
	input := `
@verify(depth: 5)
func fibonacci(n: int) -> int ! {} {
  if n <= 1 then n else fibonacci(n - 1) + fibonacci(n - 2)
}`

	l := lexer.New(input, "test.ail")
	p := New(l)
	file := p.ParseFile()

	if len(p.Errors()) > 0 {
		for _, err := range p.Errors() {
			t.Errorf("parser error: %v", err)
		}
		t.FailNow()
	}

	if len(file.Funcs) != 1 {
		t.Fatalf("expected 1 function, got %d", len(file.Funcs))
	}

	fn := file.Funcs[0]
	if fn.Name != "fibonacci" {
		t.Errorf("expected function name 'fibonacci', got %q", fn.Name)
	}

	if fn.VerifyDepth == nil {
		t.Fatal("expected VerifyDepth to be set, got nil")
	}

	if *fn.VerifyDepth != 5 {
		t.Errorf("expected VerifyDepth=5, got %d", *fn.VerifyDepth)
	}
}

// TestVerifyAttribute_DepthTwo tests a smaller depth value.
func TestVerifyAttribute_DepthTwo(t *testing.T) {
	input := `
@verify(depth: 2)
func f(x: int) -> int ! {}
requires { x >= 0 }
ensures { result >= 0 }
{
  if x <= 0 then 0 else f(x - 1) + 1
}`

	l := lexer.New(input, "test.ail")
	p := New(l)
	file := p.ParseFile()

	if len(p.Errors()) > 0 {
		for _, err := range p.Errors() {
			t.Errorf("parser error: %v", err)
		}
		t.FailNow()
	}

	if len(file.Funcs) != 1 {
		t.Fatalf("expected 1 function, got %d", len(file.Funcs))
	}

	fn := file.Funcs[0]
	if fn.VerifyDepth == nil {
		t.Fatal("expected VerifyDepth to be set, got nil")
	}

	if *fn.VerifyDepth != 2 {
		t.Errorf("expected VerifyDepth=2, got %d", *fn.VerifyDepth)
	}
}

// TestVerifyAttribute_DepthZero tests @verify(depth: 0) which means disable unrolling.
func TestVerifyAttribute_DepthZero(t *testing.T) {
	input := `
@verify(depth: 0)
func g(x: int) -> int ! {} { x + 1 }`

	l := lexer.New(input, "test.ail")
	p := New(l)
	file := p.ParseFile()

	if len(p.Errors()) > 0 {
		for _, err := range p.Errors() {
			t.Errorf("parser error: %v", err)
		}
		t.FailNow()
	}

	if len(file.Funcs) != 1 {
		t.Fatalf("expected 1 function, got %d", len(file.Funcs))
	}

	fn := file.Funcs[0]
	if fn.VerifyDepth == nil {
		t.Fatal("expected VerifyDepth to be set (even for depth 0), got nil")
	}

	if *fn.VerifyDepth != 0 {
		t.Errorf("expected VerifyDepth=0, got %d", *fn.VerifyDepth)
	}
}

// TestVerifyAttribute_NoAttribute tests that a function without @verify has nil depth.
func TestVerifyAttribute_NoAttribute(t *testing.T) {
	input := `func h(x: int) -> int ! {} { x + 1 }`

	l := lexer.New(input, "test.ail")
	p := New(l)
	file := p.ParseFile()

	if len(p.Errors()) > 0 {
		for _, err := range p.Errors() {
			t.Errorf("parser error: %v", err)
		}
		t.FailNow()
	}

	if len(file.Funcs) != 1 {
		t.Fatalf("expected 1 function, got %d", len(file.Funcs))
	}

	fn := file.Funcs[0]
	if fn.VerifyDepth != nil {
		t.Errorf("expected VerifyDepth=nil for function without @verify, got %d", *fn.VerifyDepth)
	}
}

// TestVerifyAttribute_WithExport tests @verify before an exported function.
func TestVerifyAttribute_WithExport(t *testing.T) {
	input := `
@verify(depth: 3)
export func add(x: int, y: int) -> int ! {} { x + y }`

	l := lexer.New(input, "test.ail")
	p := New(l)
	file := p.ParseFile()

	if len(p.Errors()) > 0 {
		for _, err := range p.Errors() {
			t.Errorf("parser error: %v", err)
		}
		t.FailNow()
	}

	if len(file.Funcs) != 1 {
		t.Fatalf("expected 1 function, got %d", len(file.Funcs))
	}

	fn := file.Funcs[0]
	if fn.Name != "add" {
		t.Errorf("expected function name 'add', got %q", fn.Name)
	}
	if !fn.IsExport {
		t.Error("expected function to be exported")
	}
	if fn.VerifyDepth == nil {
		t.Fatal("expected VerifyDepth to be set, got nil")
	}
	if *fn.VerifyDepth != 3 {
		t.Errorf("expected VerifyDepth=3, got %d", *fn.VerifyDepth)
	}
}

// TestVerifyAttribute_WithPure tests @verify before a pure function.
func TestVerifyAttribute_WithPure(t *testing.T) {
	input := `
@verify(depth: 7)
pure func multiply(x: int, y: int) -> int { x * y }`

	l := lexer.New(input, "test.ail")
	p := New(l)
	file := p.ParseFile()

	if len(p.Errors()) > 0 {
		for _, err := range p.Errors() {
			t.Errorf("parser error: %v", err)
		}
		t.FailNow()
	}

	if len(file.Funcs) != 1 {
		t.Fatalf("expected 1 function, got %d", len(file.Funcs))
	}

	fn := file.Funcs[0]
	if !fn.IsPure {
		t.Error("expected function to be pure")
	}
	if fn.VerifyDepth == nil {
		t.Fatal("expected VerifyDepth to be set, got nil")
	}
	if *fn.VerifyDepth != 7 {
		t.Errorf("expected VerifyDepth=7, got %d", *fn.VerifyDepth)
	}
}

// TestVerifyAttribute_MixedWithWithout tests multiple functions, some with @verify
// and some without, ensuring the attribute only applies to the immediately following function.
func TestVerifyAttribute_MixedWithWithout(t *testing.T) {
	input := `
@verify(depth: 5)
func fib(n: int) -> int ! {} {
  if n <= 1 then n else fib(n - 1) + fib(n - 2)
}

func add(x: int, y: int) -> int ! {} { x + y }

@verify(depth: 2)
func fact(n: int) -> int ! {} {
  if n <= 1 then 1 else n * fact(n - 1)
}`

	l := lexer.New(input, "test.ail")
	p := New(l)
	file := p.ParseFile()

	if len(p.Errors()) > 0 {
		for _, err := range p.Errors() {
			t.Errorf("parser error: %v", err)
		}
		t.FailNow()
	}

	if len(file.Funcs) != 3 {
		t.Fatalf("expected 3 functions, got %d", len(file.Funcs))
	}

	// fib has depth 5
	fib := file.Funcs[0]
	if fib.Name != "fib" {
		t.Errorf("expected first function to be 'fib', got %q", fib.Name)
	}
	if fib.VerifyDepth == nil || *fib.VerifyDepth != 5 {
		t.Errorf("expected fib VerifyDepth=5, got %v", fib.VerifyDepth)
	}

	// add has no attribute
	add := file.Funcs[1]
	if add.Name != "add" {
		t.Errorf("expected second function to be 'add', got %q", add.Name)
	}
	if add.VerifyDepth != nil {
		t.Errorf("expected add VerifyDepth=nil, got %d", *add.VerifyDepth)
	}

	// fact has depth 2
	fact := file.Funcs[2]
	if fact.Name != "fact" {
		t.Errorf("expected third function to be 'fact', got %q", fact.Name)
	}
	if fact.VerifyDepth == nil || *fact.VerifyDepth != 2 {
		t.Errorf("expected fact VerifyDepth=2, got %v", fact.VerifyDepth)
	}
}

// TestVerifyAttribute_InvalidKey tests that @verify(foo: 5) produces an error.
func TestVerifyAttribute_InvalidKey(t *testing.T) {
	input := `
@verify(foo: 5)
func bar(x: int) -> int ! {} { x }`

	l := lexer.New(input, "test.ail")
	p := New(l)
	_ = p.ParseFile()

	if len(p.Errors()) == 0 {
		t.Fatal("expected parser error for invalid @verify key, got none")
	}

	found := false
	for _, err := range p.Errors() {
		if strings.Contains(err.Error(), "depth") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected error mentioning 'depth', got: %v", p.Errors())
	}
}

// TestVerifyAttribute_UnknownAttribute tests that @unknown(...) produces an error.
func TestVerifyAttribute_UnknownAttribute(t *testing.T) {
	input := `
@unknown(depth: 5)
func bar(x: int) -> int ! {} { x }`

	l := lexer.New(input, "test.ail")
	p := New(l)
	_ = p.ParseFile()

	if len(p.Errors()) == 0 {
		t.Fatal("expected parser error for unknown attribute, got none")
	}

	found := false
	for _, err := range p.Errors() {
		if strings.Contains(err.Error(), "unknown attribute") || strings.Contains(err.Error(), "@unknown") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected error about unknown attribute, got: %v", p.Errors())
	}
}

// TestVerifyAttribute_OutOfRange tests that @verify(depth: 15) produces an error.
func TestVerifyAttribute_OutOfRange(t *testing.T) {
	input := `
@verify(depth: 15)
func bar(x: int) -> int ! {} { x }`

	l := lexer.New(input, "test.ail")
	p := New(l)
	_ = p.ParseFile()

	if len(p.Errors()) == 0 {
		t.Fatal("expected parser error for depth out of range, got none")
	}

	found := false
	for _, err := range p.Errors() {
		if strings.Contains(err.Error(), "range") || strings.Contains(err.Error(), "0-10") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected error about range, got: %v", p.Errors())
	}
}

// TestVerifyAttribute_NotBeforeFunc tests that @verify before a non-function declaration
// produces an error.
func TestVerifyAttribute_NotBeforeFunc(t *testing.T) {
	input := `
@verify(depth: 5)
type Foo = Bar | Baz`

	l := lexer.New(input, "test.ail")
	p := New(l)
	_ = p.ParseFile()

	if len(p.Errors()) == 0 {
		t.Fatal("expected parser error for @verify before type decl, got none")
	}

	found := false
	for _, err := range p.Errors() {
		if strings.Contains(err.Error(), "function declaration") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected error about function declaration, got: %v", p.Errors())
	}
}

// TestVerifyAttribute_WithContracts tests that @verify works alongside requires/ensures.
func TestVerifyAttribute_WithContracts(t *testing.T) {
	input := `
@verify(depth: 4)
export func safe_div(x: int, y: int) -> int ! {}
requires { y != 0 }
ensures { result * y <= x }
{
  x / y
}`

	l := lexer.New(input, "test.ail")
	p := New(l)
	file := p.ParseFile()

	if len(p.Errors()) > 0 {
		for _, err := range p.Errors() {
			t.Errorf("parser error: %v", err)
		}
		t.FailNow()
	}

	if len(file.Funcs) != 1 {
		t.Fatalf("expected 1 function, got %d", len(file.Funcs))
	}

	fn := file.Funcs[0]
	if fn.VerifyDepth == nil || *fn.VerifyDepth != 4 {
		t.Errorf("expected VerifyDepth=4, got %v", fn.VerifyDepth)
	}

	// Check contracts are still parsed
	requiresCount := 0
	ensuresCount := 0
	for _, prop := range fn.Properties {
		switch prop.Kind {
		case ast.RequiresKind:
			requiresCount++
		case ast.EnsuresKind:
			ensuresCount++
		}
	}
	if requiresCount != 1 {
		t.Errorf("expected 1 requires contract, got %d", requiresCount)
	}
	if ensuresCount != 1 {
		t.Errorf("expected 1 ensures contract, got %d", ensuresCount)
	}
}
