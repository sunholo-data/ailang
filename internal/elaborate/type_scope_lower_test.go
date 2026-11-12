package elaborate

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/lexer"
	"github.com/sunholo-data/ailang/internal/parser"
	"github.com/sunholo-data/ailang/internal/types"
)

// #2211: an applied lowercase name that names no type (`zz[int]`) is
// TC_UNBOUND_TYPE_001 only when the pipeline installed the module's scope,
// which knows its imports. A file-derived scope (REPL, runtime fallback)
// cannot tell an imported type from a value, so it reports nothing.
func TestTypeScope_AppliedLowerUnbound_OnlyWithInstalledScope(t *testing.T) {
	src := "module m\npure func f(x: zz[int]) -> int = 0\n"
	elab := func(install bool) error {
		p := parser.New(lexer.New(src, "m.ail"))
		file := p.ParseFile()
		if errs := p.Errors(); len(errs) > 0 {
			t.Fatalf("parse: %v", errs)
		}
		e := NewElaboratorWithPath("m")
		if install {
			e.SetTypeScope(types.NewTypeScope("m"))
		}
		_, err := e.ElaborateFile(file)
		return err
	}
	if err := elab(false); err != nil {
		t.Errorf("file-derived scope must not report: %v", err)
	}
	err := elab(true)
	if err == nil || !strings.Contains(err.Error(), types.TC_UNBOUND_TYPE_001) || !strings.Contains(err.Error(), "no type constructor 'zz'") {
		t.Errorf("installed scope: want TC_UNBOUND_TYPE_001 naming zz, got %v", err)
	}
}

// A bare lowercase name the scope does not declare (an import by name that is
// a value) is a type variable; one it declares is that type.
func TestTypeScope_SimpleTypeLowerDemotion(t *testing.T) {
	e := NewElaboratorWithPath("m")
	s := types.NewTypeScope("m")
	s.BindLocal("point")
	e.SetTypeScope(s)
	if tv, ok := e.simpleType("map", ast.Pos{}).(*types.TVar2); !ok || tv.Name != "map" {
		t.Errorf("undeclared lowercase: want TVar2 map, got %v", e.simpleType("map", ast.Pos{}))
	}
	if tc, ok := e.simpleType("point", ast.Pos{}).(*types.TCon); !ok || tc.Name != "point" || tc.Module != "m" {
		t.Errorf("declared lowercase: want TCon m::point, got %v", e.simpleType("point", ast.Pos{}))
	}
	if tc, ok := e.simpleType("Foo", ast.Pos{}).(*types.TCon); !ok || tc.Name != "Foo" {
		t.Errorf("uppercase: want TCon Foo, got %v", e.simpleType("Foo", ast.Pos{}))
	}
}

// TC_UNBOUND_TYPE_003 on every path that elaborates a file: a free variable
// in a type declaration is reported with a file-derived scope too; a value
// imported by name is reported only when the installed scope can tell it is
// not a type.
func TestTypeDeclVars_Paths(t *testing.T) {
	elab := func(src string, scope *types.TypeScope) error {
		p := parser.New(lexer.New(src, "m.ail"))
		file := p.ParseFile()
		if errs := p.Errors(); len(errs) > 0 {
			t.Fatalf("parse: %v", errs)
		}
		e := NewElaboratorWithPath("m")
		if scope != nil {
			e.SetTypeScope(scope)
		}
		_, err := e.ElaborateFile(file)
		return err
	}
	free := "module m\ntype Count = integer\npure func f(x: int) -> int = x\n"
	if err := elab(free, nil); err == nil || !strings.Contains(err.Error(), types.TC_UNBOUND_TYPE_003) || !strings.Contains(err.Error(), "did you mean the type `int`?") {
		t.Errorf("file-derived scope, free variable: %v", err)
	}
	value := "module m\nimport std/list (map)\ntype F = (map) -> int\npure func f(x: int) -> int = x\n"
	if err := elab(value, nil); err != nil {
		t.Errorf("file-derived scope cannot tell a value import; must not report: %v", err)
	}
	if err := elab(value, types.NewTypeScope("m")); err == nil || !strings.Contains(err.Error(), "'map' is not a parameter of F") {
		t.Errorf("installed scope, value import: %v", err)
	}
	ok := "module m\ntype Box[a] = B(a) | E\ntype pair = {x: pt, y: Box[pt]}\ntype pt = {n: int}\npure func f(x: int) -> int = x\n"
	if err := elab(ok, nil); err != nil {
		t.Errorf("parameters and local lowercase types: %v", err)
	}
}

func TestClosestTypeName(t *testing.T) {
	cands := []string{"int", "string", "()", "Count", "Point", "Handler"}
	for _, c := range []struct{ name, decl, want string }{
		{"integer", "Count", "int"}, // prefix
		{"strng", "X", "string"},    // edit distance 1
		{"point", "Seg", "Point"},   // case only
		{"handler", "Handler", ""},  // never the declaration itself
		{"a", "Any", ""},            // too short for an edit distance (not "()")
		{"cnt", "X", ""},            // too short
		{"COUNT", "X", "Count"},     // case only, short names too
		{"zzzzzz", "X", ""},         // nothing close
	} {
		if got := closestTypeName(c.name, c.decl, cands); got != c.want {
			t.Errorf("closestTypeName(%q, %q) = %q, want %q", c.name, c.decl, got, c.want)
		}
	}
}
