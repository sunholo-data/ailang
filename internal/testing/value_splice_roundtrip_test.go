package testing

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/lexer"
	"github.com/sunholo-data/ailang/internal/parser"
)

// newRoundTripEvaluator parses an ADT-bearing source so the tagged round-trip
// values can resolve their constructors in the evaluator env (injectADTConstructors).
func newRoundTripEvaluator(t *testing.T) (*eval.CoreEvaluator, *Runner) {
	t.Helper()
	const src = `module roundtrip
type Season = SPRING | SUMMER
type Block = Para(string) | Container({ blocks: list[string] })
export func main() -> int ! {} { 0 }
`
	path := filepath.Join(t.TempDir(), "roundtrip.ail")
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	l := lexer.New(src, path)
	p := parser.New(l)
	file := p.ParseFile()
	if errs := p.Errors(); len(errs) > 0 {
		t.Fatalf("parse errors: %v", errs)
	}
	runner := NewRunner(path)
	runner.executor.SetSourceFile(file)
	evaluator := eval.NewCoreEvaluator()
	runner.executor.injectADTConstructors(evaluator)
	return evaluator, runner
}

// assertRoundTrip splices v to an AST, converts to Core, evaluates it, and
// asserts the resulting eval.Value is structurally equal to v.
func assertRoundTrip(t *testing.T, r *Runner, evaluator *eval.CoreEvaluator, v eval.Value) {
	t.Helper()
	lit, err := r.valueToLiteral(v)
	if err != nil {
		t.Fatalf("valueToLiteral(%T): %v", v, err)
	}
	result, err := evaluator.Eval(astExprToCore(lit))
	if err != nil {
		t.Fatalf("evaluate spliced %T: %v", v, err)
	}
	if !equalValues(result, v) {
		t.Fatalf("round-trip mismatch: got %v, want %v", result, v)
	}
}

// TestB12_RoundTrip (acceptance criterion B1-2): for each of the five splice
// shapes, astExprToCore(valueToLiteral(v)) evaluated through the evaluator is
// structurally equal to v. The tagged cases resolve their constructors via
// injectADTConstructors. This reads one full step past the AST, so an AST that
// is well-formed but semantically wrong still reds.
func TestB12_RoundTrip(t *testing.T) {
	evaluator, r := newRoundTripEvaluator(t)

	// 1. unit
	assertRoundTrip(t, r, evaluator, &eval.UnitValue{})

	// 2. record (sorted fields survive the map round-trip)
	assertRoundTrip(t, r, evaluator, &eval.RecordValue{
		Fields: map[string]eval.Value{
			"x": &eval.IntValue{Value: 1},
			"y": &eval.BoolValue{Value: true},
		},
	})

	// 3. tuple
	assertRoundTrip(t, r, evaluator, &eval.TupleValue{
		Elements: []eval.Value{
			&eval.StringValue{Value: "hi"},
			&eval.IntValue{Value: 7},
		},
	})

	// 4. nullary tagged → bare identifier
	assertRoundTrip(t, r, evaluator, &eval.TaggedValue{
		ModulePath: "test",
		TypeName:   "Season",
		CtorName:   "SPRING",
		Fields:     []eval.Value{},
	})

	// 5. n-ary tagged → constructor closure re-applied
	assertRoundTrip(t, r, evaluator, &eval.TaggedValue{
		ModulePath: "test",
		TypeName:   "Block",
		CtorName:   "Para",
		Fields: []eval.Value{
			&eval.StringValue{Value: "hello"},
		},
	})
}
