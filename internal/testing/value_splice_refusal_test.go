package testing

import (
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

// TestRefusal_N4_Forall drives the forall path (runProperty → bindPropertyValues)
// with an injected generator that yields an unspliceable value and asserts
// StatusFail. The refusal happens inside bindPropertyValues, before the legacy
// EvaluateExpression path is touched.
func TestRefusal_N4_Forall(t *testing.T) {
	r := newSpliceRunner()
	r.genForType = func(typ ast.Type) (Generator, Shrinker) {
		return &funcValueGenerator{}, NewNoOpShrinker()
	}
	prop := &ast.Property{
		Name: "forallRefusal",
		Kind: ast.PropertyKind, // any non-ensures/requires kind dispatches to the forall path
		Binders: []*ast.Binder{
			{Name: "x", Type: &ast.SimpleType{Name: "int"}},
		},
		Expr: &ast.Literal{Kind: ast.BoolLit, Value: true},
	}
	pc := PropertyCase{Name: "forallRefusal", Property: prop}
	result := r.runProperty(pc)
	if result.Status != StatusFail {
		t.Fatalf("forall refusal: expected StatusFail, got %v (error: %s)", result.Status, result.Error)
	}
	if !strings.Contains(result.Error, "no literal splice") {
		t.Errorf("forall refusal error = %q, want it to contain \"no literal splice\"", result.Error)
	}
}

// TestRefusal_N5_Shrink drives shrinkCounterexample directly with a shrinker
// that yields an unspliceable candidate. The splice refusal must be skipped
// (best-effort continue): the function returns a minimal slice and does not
// panic, and it must not change the reported verdict.
func TestRefusal_N5_Shrink(t *testing.T) {
	r := newSpliceRunner()
	prop := &ast.Property{
		Name:    "shrinkRefusal",
		Kind:    ast.PropertyKind,
		Binders: []*ast.Binder{{Name: "x", Type: &ast.SimpleType{Name: "int"}}},
		Expr:    &ast.Literal{Kind: ast.BoolLit, Value: true},
	}
	failing := []eval.Value{&eval.IntValue{Value: 42}}
	// funcValueShrinker yields an unspliceable FunctionValue, so bindPropertyValues
	// refuses it; shrinkCounterexample must continue and keep the original value.
	minimal := r.shrinkCounterexample(prop, failing, []Shrinker{&funcValueShrinker{}})
	if len(minimal) != 1 {
		t.Fatalf("expected minimal slice of length 1, got %d", len(minimal))
	}
	if iv, ok := minimal[0].(*eval.IntValue); !ok || iv.Value != 42 {
		t.Errorf("expected minimal value to be the original int 42, got %T %v", minimal[0], minimal[0])
	}
}

// TestShrinkNilExprContract pins the implicit contract that makes the N-5 guard
// in shrinkCounterexample redundant rather than load-bearing.
//
// Measured at iteration 169: neutering that guard to `if false && err != nil`
// leaves the entire internal/testing package green, because a splice refusal
// makes bindPropertyValues return (nil, err) and EvaluateExpression then errors
// on the nil expression — the adjacent error branch continues for the same
// effect. That redundancy is only true while EvaluateExpression ERRORS on nil
// instead of panicking, which is an accident of its string-formatting
// implementation, not a stated contract. This test states it. If it ever reds,
// the N-5 guard has become the thing standing between a shrink attempt and a
// panic, and its neutering mutation will start killing.
func TestShrinkNilExprContract(t *testing.T) {
	r := newSpliceRunner()
	defer func() {
		if p := recover(); p != nil {
			t.Fatalf("EvaluateExpression(nil) panicked (%v); the N-5 guard in "+
				"shrinkCounterexample is now load-bearing — re-run its neutering "+
				"mutation and update the DECLARED REDUNDANT note in runner.go", p)
		}
	}()
	if _, err := r.executor.EvaluateExpression(nil); err == nil {
		t.Fatal("EvaluateExpression(nil) returned a nil error; the N-5 guard's " +
			"redundancy rests on this erroring — see runner.go's DECLARED REDUNDANT note")
	}
}
