package elaborate

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/types"
	"github.com/sunholo-data/ailang/internal/types/exhaust"
)

// #2371: a bare `Ok` pattern whose in-scope Ok is the prelude's Ok(a) is an
// arity error; when a type in the import closure declares a nullary Ok, the
// error names the import that makes it that type's.
func TestBareCtorArityErrorNamesTheClosureImport(t *testing.T) {
	universe := types.MatchUniverse{
		types.NominalKey("std/result", "Result"): {{Name: "Err", Arity: 1}, {Name: "Ok", Arity: 1}},
		types.NominalKey("lib/st", "Status"):     {{Name: "Err", Arity: 1}, {Name: "Ok", Arity: 0}, {Name: "Pending", Arity: 0}},
		types.NominalKey("lib/r", "R2"):          {{Name: "Bad", Arity: 0}, {Name: "Ok", Arity: 1}},
	}
	for _, c := range []struct {
		name     string
		universe types.MatchUniverse
		want     string // "" = no hint
	}{
		{"a nullary namesake in the closure", universe,
			" — if you mean lib/st's Status.Ok, import it: import lib/st (Status, Err, Ok, Pending)"},
		{"no closure table (single-file path)", nil, ""},
		{"only namesakes that take arguments", types.MatchUniverse{
			types.NominalKey("std/result", "Result"): universe[types.NominalKey("std/result", "Result")],
			types.NominalKey("lib/r", "R2"):          universe[types.NominalKey("lib/r", "R2")],
		}, ""},
	} {
		t.Run(c.name, func(t *testing.T) {
			e := NewElaborator()
			e.RegisterImportedConstructorAs("Ok", "Ok", "std/result", "Result", 1, 2)
			if c.universe != nil {
				e.SetClosureConstructors(c.universe)
			}
			_, err := e.elaboratePattern(&ast.Identifier{Name: "Ok"})
			if err == nil {
				t.Fatal("a bare Ok(a) pattern must be an arity error")
			}
			msg := err.Error()
			if !strings.Contains(msg, "constructor Ok in pattern expects 1 argument(s)") {
				t.Fatalf("not the arity error: %s", msg)
			}
			if c.want == "" && strings.Contains(msg, "import it") || c.want != "" && !strings.HasSuffix(msg, c.want) {
				t.Fatalf("hint: want %q, got %s", c.want, msg)
			}
		})
	}
}

// The candidates exclude the in-scope type itself.
func TestArityImportHintExcludesTheInScopeType(t *testing.T) {
	e := NewElaborator()
	e.SetClosureConstructors(types.MatchUniverse{
		types.NominalKey("lib/a", "A"): {exhaust.CtorSig{Name: "K", Arity: 0}},
	})
	if h := e.arityImportHint("K", "lib/a", "A", 0); h != "" {
		t.Errorf("the in-scope type is not a hint, got %q", h)
	}
	if h := e.arityImportHint("K", "lib/b", "B", 0); !strings.Contains(h, "import lib/a (A, K)") {
		t.Errorf("want lib/a's import, got %q", h)
	}
}
