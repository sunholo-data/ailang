package elaborate

import (
	"reflect"
	"sort"
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/core"
)

func TestASTPatternBinders(t *testing.T) {
	id := func(s string) *ast.Identifier { return &ast.Identifier{Name: s} }
	cases := []struct {
		name string
		pat  ast.Pattern
		want []string
	}{
		{"nil", nil, nil},
		{"lower", id("x"), []string{"x"}},
		{"upper", id("None"), nil},
		{"underscore", id("_"), nil},
		{"wildcard", &ast.WildcardPattern{}, nil},
		{"literal", &ast.Literal{Value: 1}, nil},
		{"constructor", &ast.ConstructorPattern{Name: "Some", Patterns: []ast.Pattern{&ast.ConstructorPattern{Name: "Pair", Patterns: []ast.Pattern{id("x"), id("y")}}}}, []string{"x", "y"}},
		{"cons", &ast.ConsPattern{Head: id("h"), Tail: id("t")}, []string{"h", "t"}},
		{"list_rest", &ast.ListPattern{Elements: []ast.Pattern{id("x")}, Rest: id("xs")}, []string{"x", "xs"}},
		{"tuple", &ast.TuplePattern{Elements: []ast.Pattern{id("x"), id("None")}}, []string{"x"}},
		{"record", &ast.RecordPattern{Fields: []*ast.FieldPattern{{Name: "field", Pattern: id("x")}}}, []string{"x"}},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got := astPatternBinders(tt.pat)
			sort.Strings(got)
			sort.Strings(tt.want)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("binders = %v, want %v", got, tt.want)
			}
			if tt.pat == nil {
				return
			}
			e := NewElaborator()
			pat := tt.pat
			// The parser represents cons as ConstructorPattern; the legacy
			// ConsPattern node has no direct elaborator case.
			if cons, ok := pat.(*ast.ConsPattern); ok {
				pat = &ast.ConstructorPattern{Name: "::", Patterns: []ast.Pattern{cons.Head, cons.Tail}}
			}
			p, err := e.elaboratePattern(pat)
			if err != nil {
				t.Fatal(err)
			}
			coreNames := corePatternBinders(p)
			sort.Strings(coreNames)
			if !reflect.DeepEqual(got, coreNames) {
				t.Fatalf("surface %v != core %v", got, coreNames)
			}
		})
	}
}

func TestCallGraphShadowing(t *testing.T) {
	id := func(s string) *ast.Identifier { return &ast.Identifier{Name: s} }
	for _, tt := range []struct {
		name string
		body ast.Expr
		edge bool
	}{
		{"pattern", &ast.Match{Expr: id("xs"), Cases: []*ast.Case{{Pattern: id("t"), Body: id("t")}}}, false},
		{"lambda", &ast.Lambda{Params: []*ast.Param{{Name: "t"}}, Body: id("t")}, false},
		{"let", &ast.Let{Name: "t", Value: id("xs"), Body: id("t")}, false},
		{"free", id("t"), true},
		{"let_value", &ast.Let{Name: "t", Value: id("t"), Body: id("t")}, true},
		{"block_letrec_self", &ast.Block{Exprs: []ast.Expr{&ast.LetRec{Name: "t", Value: &ast.Lambda{Body: id("t")}}, id("t")}}, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			each := &FuncSig{Name: "each", Body: tt.body}
			sibling := &FuncSig{Name: "t", Body: id("each")}
			graph := BuildCallGraph([]*FuncSig{each, sibling}, nil, map[string]*FuncSig{"each": each, "t": sibling}, nil)
			if got := len(graph.edges["each"]) > 0; got != tt.edge {
				t.Fatalf("each edges=%v, want edge=%v", graph.edges["each"], tt.edge)
			}
			if len(graph.edges["t"]) != 1 || graph.edges["t"][0] != "each" {
				t.Fatalf("genuine edge lost: %v", graph.edges)
			}
			if got := len(graph.SCCs()) == 1; got != tt.edge {
				t.Fatalf("incorrect SCCs: %v", graph.SCCs())
			}
			if got := isSelfRecursive("t", map[string]*FuncSig{"t": {Name: "t", Body: tt.body}}, nil); got != tt.edge {
				t.Fatalf("self recursion=%v, want %v", got, tt.edge)
			}
		})
	}
	// Module letrec values keep a genuine self-reference.
	value := &ast.LetRec{Name: "x", Value: &ast.Lambda{Body: id("x")}}
	if !isSelfRecursive("x", nil, map[string]*ModuleLet{"x": {Name: "x", Value: value}}) {
		t.Fatal("module letrec self-edge lost")
	}
}

func TestModuleLetRecEmission(t *testing.T) {
	id := &ast.Identifier{Name: "x"}
	prog := &ast.Program{File: &ast.File{Statements: []ast.Node{
		&ast.LetRec{Name: "x", Value: &ast.Lambda{Params: []*ast.Param{{Name: "n"}}, Body: &ast.FuncCall{Func: id, Args: []ast.Expr{&ast.Identifier{Name: "n"}}}}},
	}}}
	out, err := NewElaborator().Elaborate(prog)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Decls) != 1 {
		t.Fatalf("decls=%v", out.Decls)
	}
	if _, ok := out.Decls[0].(*core.LetRec); !ok {
		t.Fatalf("module recursion emitted as %T", out.Decls[0])
	}
}

// A module function's parameters are bound for its whole body (#2078 review).
func TestCallGraphFuncParamShadowing(t *testing.T) {
	id := func(s string) *ast.Identifier { return &ast.Identifier{Name: s} }
	each := &FuncSig{Name: "each", Params: []string{"t"}, Body: id("t")}
	sibling := &FuncSig{Name: "t", Body: id("each")}
	graph := BuildCallGraph([]*FuncSig{each, sibling}, nil, map[string]*FuncSig{"each": each, "t": sibling}, nil)
	if len(graph.edges["each"]) != 0 {
		t.Fatalf("parameter t created a false edge: %v", graph.edges["each"])
	}
	if len(graph.SCCs()) != 2 {
		t.Fatalf("false SCC: %v", graph.SCCs())
	}
	self := &FuncSig{Name: "t", Params: []string{"t"}, Body: id("t")}
	if isSelfRecursive("t", map[string]*FuncSig{"t": self}, nil) {
		t.Fatal("parameter named like its function read as self-recursion")
	}
}
