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

const maxDerivedDocNodes = 40

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
