package testing

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/eval"
)

// TestDerivedRecordGenerator_DeterministicDraws is acceptance B1-4: the
// derived generator for {x:int, y:int, s:string}, drawing once from a fresh
// newRNG(42), 200 times, yields 200 byte-identical RecordValues.
//
// The write the assertion reads is the field→value assignment in the produced
// RecordValue — exactly what changes when the per-field RNG draw order varies.
// RecordGenerator used to range over the fieldGens Go map (random order per
// range statement), so the order of RNG draws varied run-to-run and a fixed
// seed no longer reproduced a fixed record (F-3). The fix sorts the field
// names once in NewRecordGenerator and always draws in that order.
//
// Canonical serialisation: RecordValue.String() sorts field keys before
// emitting, so two values whose field→value assignments are identical produce
// identical bytes regardless of map storage order; two values whose *draw
// order* differed (and therefore assigned different values to fields) differ.
func TestDerivedRecordGenerator_DeterministicDraws(t *testing.T) {
	// Build through the production derivation path, exactly as the contract
	// harnesses do at the three generator call sites.
	r := NewRunner("test.ail")
	gen, shrink := r.createGeneratorForType(&ast.RecordType{
		Fields: []*ast.RecordField{
			{Name: "x", Type: &ast.SimpleType{Name: "int"}},
			{Name: "y", Type: &ast.SimpleType{Name: "int"}},
			{Name: "s", Type: &ast.SimpleType{Name: "string"}},
		},
	})
	if gen == nil {
		t.Fatal("expected derived generator for {x:int, y:int, s:string}")
	}
	if shrink == nil {
		t.Fatal("expected derived shrinker for {x:int, y:int, s:string}")
	}

	const draws = 200
	first := ""
	for i := 0; i < draws; i++ {
		rng := newRNG(42)
		val := gen.Generate(rng)
		rec, ok := val.(*eval.RecordValue)
		if !ok {
			t.Fatalf("draw %d: expected RecordValue, got %T", i, val)
		}
		if len(rec.Fields) != 3 {
			t.Fatalf("draw %d: expected 3 fields, got %d", i, len(rec.Fields))
		}
		s := rec.String()
		if i == 0 {
			first = s
			continue
		}
		if s != first {
			t.Fatalf("draw %d diverged from draw 0: got %s want %s", i, s, first)
		}
	}
}

// TestRecordGenerator_DeterministicAcrossInstances is the companion check on
// the plain combinator: two NewRecordGenerator instances over the same field
// map draw the same field→value assignment from the same seed. This is the
// direct unit target of the F-3 fix (the derived path above already routes
// through NewRecordGenerator, but a regression here must be attributable to
// the combinator, not the derivation).
func TestRecordGenerator_DeterministicAcrossInstances(t *testing.T) {
	fieldGens := map[string]Generator{
		"x": NewIntGenerator(-10, 10),
		"y": NewIntGenerator(-10, 10),
		"s": NewStringGenerator(0, 5, ""),
	}
	genA := NewRecordGenerator(fieldGens)
	genB := NewRecordGenerator(fieldGens)

	rngA := newRNG(7)
	rngB := newRNG(7)
	for i := 0; i < 50; i++ {
		a := genA.Generate(rngA)
		b := genB.Generate(rngB)
		if a.String() != b.String() {
			t.Fatalf("stream draw %d diverged between instances:\nA=%s\nB=%s", i, a.String(), b.String())
		}
	}
}
