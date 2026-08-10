package testing

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/lexer"
	"github.com/sunholo-data/ailang/internal/parser"
)

// deriveRunnerFromSource parses src into a Runner whose executor holds the
// source file, so same-file named-type derivation can resolve TypeDecls. The
// genForType seam stays bound to the built-in createGeneratorForType.
func deriveRunnerFromSource(t *testing.T, src string) *Runner {
	t.Helper()
	path := filepath.Join(t.TempDir(), "derive.ail")
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
	return runner
}

// deriveOnce derives typ via the production seam and draws a single value from
// a seeded RNG, returning the derived generator and the value.
func deriveOnce(t *testing.T, r *Runner, typ ast.Type) (Generator, eval.Value) {
	t.Helper()
	gen, shrink := r.genForTypeSeam(typ)
	if gen == nil {
		t.Fatalf("expected generator for %v, got nil", typ)
	}
	if shrink == nil {
		t.Fatalf("expected shrinker for %v, got nil", typ)
	}
	rng := newRNG(42)
	return gen, gen.Generate(rng)
}

// TestDerive_UnitParamM3KillsOnMissingArm derives `()` — the explicit unit arm
// added in M3. Without it, `()` in parameter position stays a vacuous skip.
func TestDerive_UnitParam(t *testing.T) {
	r := NewRunner("test.ail")
	gen, shrink := r.createGeneratorForType(&ast.SimpleType{Name: "()"})
	if gen == nil {
		t.Fatal("expected generator for (), got nil")
	}
	if shrink == nil {
		t.Fatal("expected shrinker for (), got nil")
	}
	rng := newRNG(42)
	val := gen.Generate(rng)
	if _, ok := val.(*eval.UnitValue); !ok {
		t.Fatalf("expected UnitValue, got %T", val)
	}
}

// TestDerive_AnonymousRecord derives an anonymous record type — the single most
// common vacuous shape in the corpus (§1.4: {x:int,y:int} everywhere).
func TestDerive_AnonymousRecord(t *testing.T) {
	r := NewRunner("test.ail")
	_, val := deriveOnce(t, r, &ast.RecordType{
		Fields: []*ast.RecordField{
			{Name: "x", Type: &ast.SimpleType{Name: "int"}},
			{Name: "y", Type: &ast.SimpleType{Name: "int"}},
		},
	})
	rec, ok := val.(*eval.RecordValue)
	if !ok {
		t.Fatalf("expected RecordValue, got %T", val)
	}
	if len(rec.Fields) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(rec.Fields))
	}
	for _, name := range []string{"x", "y"} {
		if _, ok := rec.Fields[name].(*eval.IntValue); !ok {
			t.Errorf("field %q: expected IntValue, got %T", name, rec.Fields[name])
		}
	}
}

// TestDerive_NestedAnonymousRecord derives a nested anonymous record — the
// B1-8 shape ({ name: string, pos: { x: int, y: int } }).
func TestDerive_NestedAnonymousRecord(t *testing.T) {
	r := NewRunner("test.ail")
	_, val := deriveOnce(t, r, &ast.RecordType{
		Fields: []*ast.RecordField{
			{Name: "name", Type: &ast.SimpleType{Name: "string"}},
			{
				Name: "pos",
				Type: &ast.RecordType{
					Fields: []*ast.RecordField{
						{Name: "x", Type: &ast.SimpleType{Name: "int"}},
						{Name: "y", Type: &ast.SimpleType{Name: "int"}},
					},
				},
			},
		},
	})
	rec, ok := val.(*eval.RecordValue)
	if !ok {
		t.Fatalf("expected RecordValue, got %T", val)
	}
	if _, ok := rec.Fields["name"].(*eval.StringValue); !ok {
		t.Errorf("name: expected StringValue, got %T", rec.Fields["name"])
	}
	pos, ok := rec.Fields["pos"].(*eval.RecordValue)
	if !ok {
		t.Fatalf("pos: expected nested RecordValue, got %T", rec.Fields["pos"])
	}
	if len(pos.Fields) != 2 {
		t.Fatalf("pos: expected 2 fields, got %d", len(pos.Fields))
	}
	for _, name := range []string{"x", "y"} {
		if _, ok := pos.Fields[name].(*eval.IntValue); !ok {
			t.Errorf("pos.%q: expected IntValue, got %T", name, pos.Fields[name])
		}
	}
}

// TestDerive_Tuple derives a tuple type (int, string).
func TestDerive_Tuple(t *testing.T) {
	r := NewRunner("test.ail")
	_, val := deriveOnce(t, r, &ast.TupleType{
		Elements: []ast.Type{
			&ast.SimpleType{Name: "int"},
			&ast.SimpleType{Name: "string"},
		},
	})
	tup, ok := val.(*eval.TupleValue)
	if !ok {
		t.Fatalf("expected TupleValue, got %T", val)
	}
	if len(tup.Elements) != 2 {
		t.Fatalf("expected 2 elements, got %d", len(tup.Elements))
	}
	if _, ok := tup.Elements[0].(*eval.IntValue); !ok {
		t.Errorf("element 0: expected IntValue, got %T", tup.Elements[0])
	}
	if _, ok := tup.Elements[1].(*eval.StringValue); !ok {
		t.Errorf("element 1: expected StringValue, got %T", tup.Elements[1])
	}
}

// TestDerive_NamedRecordFromSourceFile derives a same-file named record type
// (`type Point = { x: int, y: int }`) via the TypeDecl lookup.
func TestDerive_NamedRecordFromSourceFile(t *testing.T) {
	src := `module derive_named

export type Point = { x: int, y: int }

export func main() -> int ! {} { 0 }
`
	r := deriveRunnerFromSource(t, src)
	_, val := deriveOnce(t, r, &ast.SimpleType{Name: "Point"})
	rec, ok := val.(*eval.RecordValue)
	if !ok {
		t.Fatalf("expected RecordValue, got %T", val)
	}
	if len(rec.Fields) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(rec.Fields))
	}
}

// TestDerive_NamedTypeMissing_SourceNil pins B1-10's honest-skip half: a named
// type with no source file on the executor stays a vacuous skip (nil, nil) —
// it must never fall back to a fabricated unit generator.
func TestDerive_NamedTypeMissing_SourceNil(t *testing.T) {
	r := NewRunner("test.ail") // executor.sourceFile is nil here
	gen, shrink := r.createGeneratorForType(&ast.SimpleType{Name: "Cell"})
	if gen != nil || shrink != nil {
		t.Fatalf("expected nil, nil for unresolvable named type, got generator=%T shrinker=%T", gen, shrink)
	}
}

// TestDerive_NamedTypeMissing_NoDecl pins the same honest-skip behaviour when a
// source file exists but contains no matching TypeDecl (imported types).
func TestDerive_NamedTypeMissing_NoDecl(t *testing.T) {
	src := `module derive_missing

export func main() -> int ! {} { 0 }
`
	r := deriveRunnerFromSource(t, src)
	gen, shrink := r.createGeneratorForType(&ast.SimpleType{Name: "Cell"})
	if gen != nil || shrink != nil {
		t.Fatalf("expected nil, nil for undeclared named type, got generator=%T shrinker=%T", gen, shrink)
	}
}

// TestDerive_TypeAliasToTuple covers §1.5's TypeAlias omission: `type Pair =
// (int, int)` parses to TypeAlias{Target: TupleType} and must recurse on the
// target, or every alias-typed parameter stays vacuous.
func TestDerive_TypeAliasToTuple(t *testing.T) {
	src := `module derive_alias_tuple

export type Pair = (int, int)

export func main() -> int ! {} { 0 }
`
	r := deriveRunnerFromSource(t, src)
	_, val := deriveOnce(t, r, &ast.SimpleType{Name: "Pair"})
	tup, ok := val.(*eval.TupleValue)
	if !ok {
		t.Fatalf("expected TupleValue via alias target, got %T", val)
	}
	if len(tup.Elements) != 2 {
		t.Fatalf("expected 2 elements, got %d", len(tup.Elements))
	}
}

// TestDerive_TypeAliasToScalar covers `type UserId = int` (TypeAlias over a
// scalar) — without the alias arm, UserId-typed parameters stay vacuous.
func TestDerive_TypeAliasToScalar(t *testing.T) {
	src := `module derive_alias_scalar

export type UserId = int

export func main() -> int ! {} { 0 }
`
	r := deriveRunnerFromSource(t, src)
	_, val := deriveOnce(t, r, &ast.SimpleType{Name: "UserId"})
	if _, ok := val.(*eval.IntValue); !ok {
		t.Fatalf("expected IntValue via alias target, got %T", val)
	}
}

// TestDerive_TypeAliasToList covers `type Names = [string]` (TypeAlias over a
// list TypeApp) — the alias arm recurses into the preserved Lane A list arm.
func TestDerive_TypeAliasToList(t *testing.T) {
	src := `module derive_alias_list

export type Names = [string]

export func main() -> int ! {} { 0 }
`
	r := deriveRunnerFromSource(t, src)
	_, val := deriveOnce(t, r, &ast.SimpleType{Name: "Names"})
	if _, ok := val.(*eval.ListValue); !ok {
		t.Fatalf("expected ListValue via alias target, got %T", val)
	}
}

// TestDerive_UnderivableFieldKillsWholeRecord pins the nil-on-any-underivable-
// field rule: an ADT field (M4 territory) makes the entire record underivable
// rather than silently substituting the other fields.
func TestDerive_UnderivableFieldKillsWholeRecord(t *testing.T) {
	src := `module derive_underv

export type Shade = Light | Dark(level: int)

export type Rec = { ok: int, shade: Shade }

export func main() -> int ! {} { 0 }
`
	r := deriveRunnerFromSource(t, src)
	gen, shrink := r.createGeneratorForType(&ast.SimpleType{Name: "Rec"})
	if gen != nil || shrink != nil {
		t.Fatalf("expected nil, nil for record with underivable ADT field, got generator=%T shrinker=%T", gen, shrink)
	}
}

// TestDerive_ADTIsM4 pins that AlgebraicType definitions do NOT derive in M3 —
// they stay honest vacuous skips until M4's arm lands.
func TestDerive_ADTIsM4(t *testing.T) {
	src := `module derive_adt_m4

export type Shade = Light | Dark(level: int)

export func main() -> int ! {} { 0 }
`
	r := deriveRunnerFromSource(t, src)
	gen, shrink := r.createGeneratorForType(&ast.SimpleType{Name: "Shade"})
	if gen != nil || shrink != nil {
		t.Fatalf("expected nil, nil for ADT at M3, got generator=%T shrinker=%T", gen, shrink)
	}
}

// TestDerive_ListArmPreserved pins that the Lane A list arm is untouched:
// list[int] still derives, list[Shade] (ADT element, M4) still fails.
func TestDerive_ListArmPreserved(t *testing.T) {
	src := `module derive_list

export type Shade = Light | Dark(level: int)

export func main() -> int ! {} { 0 }
`
	r := deriveRunnerFromSource(t, src)
	gen, _ := r.createGeneratorForType(&ast.TypeApp{
		Constructor: "list",
		Args:        []ast.Type{&ast.SimpleType{Name: "int"}},
	})
	if gen == nil {
		t.Fatal("expected list[int] to keep deriving (Lane A arm preserved)")
	}
	gen, _ = r.createGeneratorForType(&ast.TypeApp{
		Constructor: "list",
		Args:        []ast.Type{&ast.SimpleType{Name: "Shade"}},
	})
	if gen != nil {
		t.Fatal("expected list[Shade] to stay underivable at M3 (ADT element)")
	}
}

// TestDerive_RecursiveNamedRecordBounded pins the depth budget: a
// self-referential named record cannot recurse forever at M3 — it degrades to
// an honest nil, nil rather than a stack overflow.
func TestDerive_RecursiveNamedRecordBounded(t *testing.T) {
	src := `module derive_rec

export type L = { val: int, next: L }

export func main() -> int ! {} { 0 }
`
	r := deriveRunnerFromSource(t, src)
	gen, shrink := r.createGeneratorForType(&ast.SimpleType{Name: "L"})
	if gen != nil || shrink != nil {
		t.Fatalf("expected nil, nil for unbounded recursive record at depth budget, got generator=%T shrinker=%T", gen, shrink)
	}
}

// TestDerive_UnderivableElemKillsWholeTuple pins the tuple arm's refusal
// branch: an underivable element (an ADT, which is M4) makes the whole tuple
// underivable — nil, nil, never a partially-derived tuple. Added at
// iteration-170 review: neutering this branch (`if false && elemGen == nil`)
// left the entire package green, so nothing else protects it.
func TestDerive_UnderivableElemKillsWholeTuple(t *testing.T) {
	src := `module derive_tuple_refuse

export type Color = Red | Green

export func main() -> int ! {} { 0 }
`
	r := deriveRunnerFromSource(t, src)
	typ := &ast.TupleType{Elements: []ast.Type{
		&ast.SimpleType{Name: "int"},
		&ast.SimpleType{Name: "Color"},
	}}
	gen, shrink := r.createGeneratorForType(typ)
	if gen != nil || shrink != nil {
		t.Fatalf("expected nil, nil for tuple with underivable element, got generator=%T shrinker=%T", gen, shrink)
	}
}
