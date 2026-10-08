package testing

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/lexer"
	"github.com/sunholo-data/ailang/internal/parser"
)

// unspliceableFuncValue returns a value kind valueToLiteral has no splice arm
// for (a *eval.FunctionValue), so the default refusal branch fires.
func unspliceableFuncValue() eval.Value {
	return &eval.FunctionValue{}
}

// funcValueGenerator is a test generator that always yields an unspliceable
// FunctionValue, driving the refusal branches at all three production call
// sites via the genForType seam.
type funcValueGenerator struct{}

func (g *funcValueGenerator) Generate(rng *rand.Rand) eval.Value { return unspliceableFuncValue() }

// funcValueShrinker yields an unspliceable FunctionValue as its single shrink
// candidate, exercising the shrink path's refusal branch (best-effort continue).
type funcValueShrinker struct{}

func (s *funcValueShrinker) Shrink(val eval.Value) []eval.Value {
	return []eval.Value{unspliceableFuncValue()}
}

// refusalRunnerFromSource parses src into a Runner with the genForType seam set
// to gen, plus the collected TestSuite, so each property path reaches the
// refusal branch with an injected refusal-producing generator.
func refusalRunnerFromSource(t *testing.T, src string, gen func(ast.Type) (Generator, Shrinker)) (*Runner, *TestSuite) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "refusal.ail")
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	l := lexer.New(src, path)
	p := parser.New(l)
	file := p.ParseFile()
	if errs := p.Errors(); len(errs) > 0 {
		t.Fatalf("parse errors: %v", errs)
	}
	suite := NewCollector(path).Collect(file)
	runner := NewRunner(path)
	runner.executor.SetSourceFile(file)
	runner.genForType = gen
	return runner, suite
}

// TestRefusal_N1_DirectCall pumps an unspliceable value straight into
// valueToLiteral and asserts a non-nil error that names the Go type. This is
// the default refusal arm itself; it needs no seam.
func TestRefusal_N1_DirectCall(t *testing.T) {
	r := newSpliceRunner()
	_, err := r.valueToLiteral(unspliceableFuncValue())
	if err == nil {
		t.Fatal("expected non-nil error for unspliceable value, got nil")
	}
	if !strings.Contains(err.Error(), "no literal splice") {
		t.Errorf("error = %q, want it to contain \"no literal splice\"", err.Error())
	}
	if !strings.Contains(err.Error(), "*eval.FunctionValue") {
		t.Errorf("error = %q, want it to name the Go type *eval.FunctionValue", err.Error())
	}
}

// TestRefusal_N2_Ensures drives the ensures path with an injected generator
// that yields an unspliceable value and asserts the splice refusal surfaces as
// StatusFail (never StatusSkip — a skip would re-open the vacuous hole).
func TestRefusal_N2_Ensures(t *testing.T) {
	src := `module refusal_ensures
export pure func f(x: int) -> int
  ensures { result == x }
{ x }
export func main() -> int ! {} { 0 }
`
	runner, suite := refusalRunnerFromSource(t, src, func(typ ast.Type) (Generator, Shrinker) {
		return &funcValueGenerator{}, NewNoOpShrinker()
	})
	result := runner.RunSuite(suite)
	if len(result.Properties) == 0 {
		t.Fatalf("expected at least one property result, got 0")
	}
	pr := result.Properties[0]
	if pr.Status != StatusFail {
		t.Fatalf("ensures refusal: expected StatusFail, got %v (error: %s)", pr.Status, pr.Error)
	}
	if !strings.Contains(pr.Error, "no literal splice") {
		t.Errorf("ensures refusal error = %q, want it to contain \"no literal splice\"", pr.Error)
	}
}

// TestRefusal_N3_Requires drives the requires path with an injected generator
// that yields an unspliceable value and asserts StatusFail.
func TestRefusal_N3_Requires(t *testing.T) {
	src := `module refusal_requires
export pure func f(x: int) -> int
  requires { x == x }
{ x }
export func main() -> int ! {} { 0 }
`
	runner, suite := refusalRunnerFromSource(t, src, func(typ ast.Type) (Generator, Shrinker) {
		return &funcValueGenerator{}, NewNoOpShrinker()
	})
	result := runner.RunSuite(suite)
	if len(result.Properties) == 0 {
		t.Fatalf("expected at least one property result, got 0")
	}
	pr := result.Properties[0]
	if pr.Status != StatusFail {
		t.Fatalf("requires refusal: expected StatusFail, got %v (error: %s)", pr.Status, pr.Error)
	}
	if !strings.Contains(pr.Error, "no literal splice") {
		t.Errorf("requires refusal error = %q, want it to contain \"no literal splice\"", pr.Error)
	}
}

// TestRefusal_N5_Shrink: a shrink candidate the predicate cannot evaluate
// (here a FunctionValue for an int binder, which errors) is skipped, and the
// original failing value stands. Since #624 the forall path passes values as
// arguments, so there is no splice to refuse; the skip is the contract.
func TestRefusal_N5_Shrink(t *testing.T) {
	failing := []eval.Value{&eval.IntValue{Value: 42}}
	call := func(args []eval.Value) (eval.Value, error) {
		if _, ok := args[0].(*eval.IntValue); !ok {
			return nil, fmt.Errorf("expected int, got %T", args[0])
		}
		return &eval.BoolValue{Value: false}, nil
	}
	minimal := shrinkCounterexample(call, failing, []Shrinker{&funcValueShrinker{}})
	if len(minimal) != 1 {
		t.Fatalf("expected minimal slice of length 1, got %d", len(minimal))
	}
	if iv, ok := minimal[0].(*eval.IntValue); !ok || iv.Value != 42 {
		t.Errorf("expected minimal value to be the original int 42, got %T %v", minimal[0], minimal[0])
	}
}
