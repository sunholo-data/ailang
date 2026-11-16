package pipeline

import (
	"fmt"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/loader"
)

// Review round 2 (probes pre6, pre10): the prelude the loader injects into an
// entry module (import std/option (Option, Some, None), import std/result
// (Result, Ok, Err)) is the LOWEST-precedence binding of its names. A user's
// own declaration or explicit import of any prelude name wins silently, as on
// dev — for types and constructors, in entry and non-entry modules — and no
// diagnostic ever points at the injected import (position 0:0). The names are
// read from the prelude definition itself.
func TestNominalIdentity_UserBindingBeatsThePrelude(t *testing.T) {
	var names []string
	for _, mod := range loader.EntryPreludeModules() {
		names = append(names, loader.EntryPreludeSymbols(mod)...)
	}
	if len(names) < 6 {
		t.Fatalf("prelude names not read: %v", names)
	}
	for _, n := range names {
		isType := n == "Option" || n == "Result"
		var decl, use, imp string
		if isType {
			decl = fmt.Sprintf("export type %s = Mk(int) | Nope\nexport pure func mk() -> %s = Mk(5)\n", n, n)
			use = fmt.Sprintf("pure func g(x: %s) -> int = match x { Mk(k) => k, Nope => 0 }\n", n)
			imp = fmt.Sprintf("import m/lib (%s, Mk, Nope, mk)\n", n)
		} else {
			decl = fmt.Sprintf("export type Outcome = %s(int) | Other\nexport pure func mk() -> Outcome = %s(5)\n", n, n)
			use = fmt.Sprintf("pure func g(x: Outcome) -> int = match x { %s(k) => k, Other => 0 }\n", n)
			imp = fmt.Sprintf("import m/lib (Outcome, %s, Other, mk)\n", n)
		}
		entry := "export func main() -> () ! {IO} = println(show(g(mk())))\n"
		library := "export pure func run() -> int = g(mk())\n"
		for _, c := range []struct {
			name  string
			files map[string]string
		}{
			{"explicit import, entry", map[string]string{"m/lib.ail": "module m/lib\n" + decl, "m/main.ail": "module m/main\n" + imp + use + entry}},
			{"explicit import, library", map[string]string{"m/lib.ail": "module m/lib\n" + decl, "m/main.ail": "module m/main\n" + imp + use + library}},
			{"local declaration, entry", map[string]string{"m/main.ail": "module m/main\n" + strings.ReplaceAll(decl, "export type", "type") + use + entry}},
			{"local declaration, library", map[string]string{"m/main.ail": "module m/main\n" + strings.ReplaceAll(decl, "export type", "type") + use + library}},
		} {
			t.Run(n+"/"+c.name, func(t *testing.T) {
				root := t.TempDir()
				t.Chdir(root)
				writeNominalFiles(t, root, c.files)
				_, err := Run(Config{Mode: ModeCheck, NoCache: true}, Source{Filename: "m/main.ail"})
				if err != nil {
					if strings.Contains(err.Error(), ":0:0") {
						t.Fatalf("a diagnostic points at injected code (0:0): %v", err)
					}
					t.Fatalf("the user's %s must win over the prelude's: %v", n, err)
				}
			})
		}
	}
}
