package pipeline

import (
	"errors"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/iface"
	"github.com/sunholo-data/ailang/internal/lexer"
	"github.com/sunholo-data/ailang/internal/parser"
	"github.com/sunholo-data/ailang/internal/types"
)

func parseForTypeNames(t *testing.T, src string) *ast.File {
	t.Helper()
	p := parser.New(lexer.New(src, "q/main.ail"))
	f := p.ParseFile()
	if errs := p.Errors(); len(errs) > 0 {
		t.Fatalf("parse: %v", errs)
	}
	return f
}

// q/lib exports a lowercase type `t` (and its constructor's ADT `w`) and a
// value `mk`.
func typeNamesLib() map[string]*iface.Iface {
	lib := iface.NewIface("q/lib")
	lib.AddType("t", 0)
	lib.AddConstructor("w", "W", nil, &types.TCon{Name: "w", Module: "q/lib"})
	return map[string]*iface.Iface{"q/lib": lib}
}

func warnStrings(ws []interface{ String() string }) string {
	var out []string
	for _, w := range ws {
		out = append(out, w.String())
	}
	return strings.Join(out, "\n")
}

func TestTypeNameWarnings_Unit(t *testing.T) {
	ifaces := typeNamesLib()
	get := func(p string) *iface.Iface { return ifaces[p] }
	cases := []struct {
		name, src string
		want      []string // substrings, one per expected warning; nil = none
	}{
		{"implicit var named like an exported type, once per declaration",
			"import q/lib (mk)\npure func f(x: t, y: t) -> t = x\npure func g(x: t) -> int = 0",
			[]string{"'t' is treated as a type variable", "'t' is treated as a type variable"}},
		{"a constructor's ADT counts as an exported type",
			"import q/lib (mk)\npure func f(x: w) -> int = 0", []string{"q/lib exports a type 'w'"}},
		{"value imported by name, aliased",
			"import q/lib (mk as maker)\npure func f(x: maker) -> int = 0",
			[]string{"'maker' is treated as a type variable here; it is imported by name from q/lib, but as a value"}},
		{"type imported by name: no warning", "import q/lib (t)\npure func f(x: t) -> int = 0", nil},
		{"bound parameter: no warning", "import q/lib (mk)\npure func f[t](x: t) -> t = x", nil},
		{"unknown module interface: no warning", "import q/other (v)\npure func f(x: v, y: z) -> int = 0", nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ws := typeNameWarnings(parseForTypeNames(t, c.src), get)
			var got []interface{ String() string }
			for _, w := range ws {
				got = append(got, w)
			}
			if len(ws) != len(c.want) {
				t.Fatalf("want %d warnings, got %d:\n%s", len(c.want), len(ws), warnStrings(got))
			}
			for i, w := range ws {
				tw := w.(*TypeNameWarning)
				if tw.Code() != types.TC_UNBOUND_TYPE_002 || !strings.Contains(w.String(), c.want[i]) || tw.Suggestion() == "" {
					t.Errorf("warning %d: want %q, got %s", i, c.want[i], w.String())
				}
				if l, _ := tw.Position(); l == 0 || tw.WarningFile() != "q/main.ail" {
					t.Errorf("warning %d: bad location %+v", i, tw.Location)
				}
			}
		})
	}
}

// DetectTypeNameWarnings skips an installed dependency's modules and units
// without a parsed source.
func TestDetectTypeNameWarnings_SkipsDependencies(t *testing.T) {
	ifaces := typeNamesLib()
	get := func(p string) *iface.Iface { return ifaces[p] }
	src := "import q/lib (mk)\npure func f(x: t) -> int = 0"
	units := map[string]*CompileUnit{
		"q/main":            {ID: "q/main", Surface: parseForTypeNames(t, src)},
		"pkg/other/lib/mod": {ID: "pkg/other/lib/mod", Surface: parseForTypeNames(t, src)},
		"q/nosurface":       {ID: "q/nosurface"},
	}
	ws := DetectTypeNameWarnings(units, get)
	if len(ws) != 1 {
		t.Fatalf("want 1 warning (q/main only), got %d", len(ws))
	}
}

func TestWithLowerTypeNote(t *testing.T) {
	base := errors.New("type error in q/main (decl 1): boom")
	mixed := parseForTypeNames(t, "type acc = {n: int}\npure func f(g: (acc, elem) -> acc) -> int = 0")
	err := withLowerTypeNote(base, mixed)
	if !errors.Is(err, base) || !strings.Contains(err.Error(), "Note: 'acc' in the signature of f is the type declared at q/main.ail:1:1") {
		t.Errorf("mixed signature: %v", err)
	}
	// No implicit variable beside the declared type: no note.
	plain := parseForTypeNames(t, "type acc = {n: int}\npure func f(g: acc) -> acc = g")
	if got := withLowerTypeNote(base, plain); got != base {
		t.Errorf("plain signature got a note: %v", got)
	}
	if withLowerTypeNote(nil, mixed) != nil || withLowerTypeNote(base, nil) != base {
		t.Error("nil error or file must pass through")
	}
}
