package elaborate

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
)

// The MOD015 fix line must be a valid import a model can paste: value aliases
// stay lowercase, constructor aliases uppercase, a module alias is kept.
// Behaviour through the full pipeline: internal/pipeline/import_local_collision_test.go.
func TestMOD015AliasSuggestion(t *testing.T) {
	cases := []struct {
		b    importBinding
		want string
	}{
		{importBinding{name: "map", orig: "map", path: "std/list"}, "import std/list (map as listMap)"},
		{importBinding{name: "tick", orig: "tock", path: "./a"}, "import ./a (tock as aTick)"},
		{importBinding{name: "Red", orig: "Red", path: "pkg/x/colors", kind: importCtor}, "import pkg/x/colors (Red as ColorsRed)"},
		{importBinding{name: "onEvent", orig: "onEvent", path: "std/stream", mod: "Stream"}, "import std/stream as Stream (onEvent as streamOnEvent)"},
		{importBinding{name: "f", orig: "f", path: "./2d"}, "import ./2d (f as importedF)"},
	}
	for _, c := range cases {
		if got := aliasSuggestion(c.b); got != c.want {
			t.Errorf("aliasSuggestion(%+v) = %q, want %q", c.b, got, c.want)
		}
	}
}

func TestMOD015LetVsFuncLeftToMOD007(t *testing.T) {
	// A name that is both a let and a func is MOD007's to report; the MOD015
	// check must still see it as a local (as the func) and not panic.
	funcs := []*FuncSig{{Name: "tick", FuncDecl: &ast.FuncDecl{Name: "tick", Pos: ast.Pos{Line: 5, Column: 1}}}}
	lets := []*ModuleLet{{Name: "tick", Pos: ast.Pos{Line: 4, Column: 1}}}
	binds := []importBinding{{name: "tick", orig: "tick", path: "dep", pos: ast.Pos{Line: 2, Column: 1}}}
	err := checkImportCollisions(&ast.File{Path: "m.ail"}, binds, lets, funcs)
	if err == nil || !strings.Contains(err.Error(), "as a func (at m.ail:5:1)") {
		t.Fatalf("want MOD015 citing the func, got %v", err)
	}
}
