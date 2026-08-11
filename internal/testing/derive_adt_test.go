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

func parseDeriveFixture(t *testing.T, src string) (*Runner, *ast.File) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "fixture.ail")
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	p := parser.New(lexer.New(src, path))
	file := p.ParseFile()
	if errs := p.Errors(); len(errs) != 0 {
		t.Fatalf("parse fixture: %v", errs)
	}
	r := NewRunner(path)
	r.executor.SetSourceFile(file)
	return r, file
}

func TestM4B1_6_ADTConstructorCoverage(t *testing.T) {
	r, _ := parseDeriveFixture(t, `module examples/runnable/contracts/park
export type Season = LOW_SEASON | HIGH_SEASON
export func main() -> int ! {} { 0 }
`)
	gen, _ := r.createGeneratorForType(&ast.SimpleType{Name: "Season"})
	if gen == nil {
		t.Fatal("Season generator is nil")
	}
	seen := map[string]bool{}
	rng := newRNG(42)
	for i := 0; i < 100; i++ {
		value, ok := gen.Generate(rng).(*eval.TaggedValue)
		if !ok {
			t.Fatalf("draw %d: expected TaggedValue", i)
		}
		seen[value.CtorName] = true
	}
	for _, ctor := range []string{"LOW_SEASON", "HIGH_SEASON"} {
		if !seen[ctor] {
			t.Errorf("constructor %s was never generated; seen=%v", ctor, seen)
		}
	}
}

func TestM4B1_7_NullaryAndNarySplice(t *testing.T) {
	result := runEnsuresFromSource(t, `module shade_splice
export type Shade = Light | Dark(int)
export func valid(s: Shade) -> bool ! {}
ensures { result == true } {
  match s { Light => true, Dark(level) => level == level }
}
export func main() -> int ! {} { 0 }
`)
	property := propertyByName(t, result, "valid_property_1")
	if property.Status != StatusPass || property.TestsRun != 100 {
		t.Fatalf("property=%+v, want 100-case pass", property)
	}
}

// maxDerivedDocNodes bounds B1-9's size assertion. The draw sequence below is
// fully deterministic (newRNG(42), a fixed 200 draws), so this can be tight
// without flaking: the measured max over that exact sequence is 7. It is
// deliberately NOT set to a round "obviously safe" number — at 40 the assertion
// still caught the plan's named mutation (restoring unlimited MaxSize inside a
// derivation, which reaches 197) but sailed past a modest one: widening the
// per-level cap to ctx.depth+3 tops out at 13 and would have passed unnoticed.
// A bound that only catches the catastrophic regression is not a size budget.
const maxDerivedDocNodes = 10

func TestM4B1_9_MutualRecursionSizeBound(t *testing.T) {
	r, _ := parseDeriveFixture(t, `module doc_cycle
export type Block = Para(string) | Container({ blocks: list[Block], kind: string })
export type Doc = { title: string, blocks: list[Block] }
export func main() -> int ! {} { 0 }
`)
	gen, _ := r.createGeneratorForType(&ast.SimpleType{Name: "Doc"})
	if gen == nil {
		t.Fatal("Doc generator is nil")
	}
	rng := newRNG(42)
	for i := 0; i < 200; i++ {
		n := derivedValueNodes(gen.Generate(rng))
		if n > maxDerivedDocNodes {
			t.Fatalf("draw %d has %d nodes, limit is %d", i, n, maxDerivedDocNodes)
		}
	}
}

func derivedValueNodes(value eval.Value) int {
	switch v := value.(type) {
	case *eval.ListValue:
		n := 1
		for _, elem := range v.Elements {
			n += derivedValueNodes(elem)
		}
		return n
	case *eval.RecordValue:
		n := 1
		for _, field := range v.Fields {
			n += derivedValueNodes(field)
		}
		return n
	case *eval.TupleValue:
		n := 1
		for _, elem := range v.Elements {
			n += derivedValueNodes(elem)
		}
		return n
	case *eval.TaggedValue:
		n := 1
		for _, field := range v.Fields {
			n += derivedValueNodes(field)
		}
		return n
	default:
		return 1
	}
}

func TestM4B1_12_EndlessADTIsNoGenerator(t *testing.T) {
	result := runEnsuresFromSource(t, `module endless
export type Endless = Wrap(Endless)
export func impossible(x: Endless) -> bool ! {}
ensures { result == true } { true }
export func main() -> int ! {} { 0 }
`)
	property := propertyByName(t, result, "impossible_property_1")
	if property.Status != StatusSkip || property.SkipKind != SkipKindNoGenerator {
		t.Fatalf("property=%+v, want no_generator skip", property)
	}
}

func TestM4B1_13_TopLevelScalarListPreserved(t *testing.T) {
	result := runEnsuresFromSource(t, `module lane_a_list
export func same(xs: list[int]) -> bool ! {}
ensures { result == true } { true }
export func main() -> int ! {} { 0 }
`)
	property := propertyByName(t, result, "same_property_1")
	if property.Status != StatusPass || property.TestsRun != 100 {
		t.Fatalf("property=%+v, want exact 100-case pass", property)
	}
	r := NewRunner("list.ail")
	gen, _ := r.createGeneratorForType(&ast.TypeApp{
		Constructor: "list",
		Args:        []ast.Type{&ast.SimpleType{Name: "int"}},
	})
	rng := newRNG(42)
	maxLen := 0
	for i := 0; i < 100; i++ {
		length := len(gen.Generate(rng).(*eval.ListValue).Elements)
		if length > maxLen {
			maxLen = length
		}
	}
	if maxLen <= maxDeriveDepth {
		t.Fatalf("top-level list range was depth-capped: max length=%d", maxLen)
	}
}

func TestM4_ADTGeneratorCarriesRealIdentity(t *testing.T) {
	gen := NewADTGenerator("Some", []Generator{NewIntGenerator(1, 1)}, true,
		"pkg/example", "Option")
	value := gen.Generate(newRNG(42)).(*eval.TaggedValue)
	if value.ModulePath != "pkg/example" || value.TypeName != "Option" {
		t.Fatalf("identity=%s/%s, want pkg/example/Option", value.ModulePath, value.TypeName)
	}
}

func TestM4_TypeAppSubstitution(t *testing.T) {
	r, _ := parseDeriveFixture(t, `module generic_box
export type Box[a] = Boxed(a)
export func main() -> int ! {} { 0 }
`)
	gen, _ := r.createGeneratorForType(&ast.TypeApp{
		Constructor: "Box",
		Args:        []ast.Type{&ast.SimpleType{Name: "int"}},
	})
	if gen == nil {
		t.Fatal("Box[int] generator is nil")
	}
	value := gen.Generate(newRNG(42)).(*eval.TaggedValue)
	if _, ok := value.Fields[0].(*eval.IntValue); !ok {
		t.Fatalf("substituted field type=%T, want IntValue", value.Fields[0])
	}
}

// TestM4_UnderivableConstructorRefusesWholeADT pins plan §3 M4 change 1: if ANY
// field of ANY constructor is underivable, the WHOLE type refuses. Dropping just
// the offending constructor would still produce a working generator — one that
// silently draws from a biased sub-distribution, never emitting Bad. That is the
// one defect in this milestone with no visible symptom: every other test still
// passes, the corpus still runs, and the property tests quietly stop covering a
// constructor. Mutating the refusal to `continue` past the bad constructor leaves
// the entire package green without this test.
func TestM4_UnderivableConstructorRefusesWholeADT(t *testing.T) {
	r, _ := parseDeriveFixture(t, `module mixed_ctors
export type Mix = Good(int) | Bad(ImportedThing)
export func main() -> int ! {} { 0 }
`)
	gen, shrink := r.createGeneratorForType(&ast.SimpleType{Name: "Mix"})
	if gen != nil || shrink != nil {
		t.Fatalf("a constructor with an underivable field must refuse the whole ADT, "+
			"got generator=%T shrinker=%T (a non-nil generator here draws only from "+
			"the derivable constructors, biasing the distribution invisibly)", gen, shrink)
	}
}

// TestM4_IndirectRecursionThroughNamedTypeIsBounded covers the indirect arm of
// the recursion detector (typeDefReferencesDecl). B1-9's Doc/Block fixture
// recurses through an INLINE anonymous record, so it reaches the target through
// typeReferencesDecl's RecordType arm and never consults a named intermediary —
// leaving the whole named-type path at 0% coverage while looking well tested.
//
// Here Branch's field is a NAMED type whose own definition points back at Node.
// If that indirection is not followed, Branch is classified non-recursive, the
// depth bound never restricts it, and derivation runs away. The size assertion
// is what catches that: it is downstream of the classification, not adjacent to
// it. Neutering typeDefReferencesDecl to `return false` reds this test.
func TestM4_IndirectRecursionThroughNamedTypeIsBounded(t *testing.T) {
	r, _ := parseDeriveFixture(t, `module indirect_rec
export type Node = Leaf | Branch(Wrapper)
export type Wrapper = { node: Node }
export func main() -> int ! {} { 0 }
`)
	gen, _ := r.createGeneratorForType(&ast.SimpleType{Name: "Node"})
	if gen == nil {
		t.Fatal("Node generator is nil: an ADT recursive through a named type must still derive")
	}
	rng := newRNG(7)
	for i := 0; i < 200; i++ {
		if n := derivedValueNodes(gen.Generate(rng)); n > maxDerivedDocNodes {
			t.Fatalf("draw %d has %d nodes, limit is %d — indirect recursion through a "+
				"named type was not detected, so the depth bound never restricted Branch",
				i, n, maxDerivedDocNodes)
		}
	}
}
