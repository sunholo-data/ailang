package pipeline

import (
	"strings"
	"testing"
)

// #2371: the Option/Result prelude (Option, Some, None, Result, Ok, Err) is in
// scope in every module but the standard library's, with the entry-module
// rules: lowest precedence, a module's own declaration of a prelude type or
// constructor drops that prelude module, an explicit import of the name wins.
// Each case is a library module (no main) checked through an importing root.
func TestPreludeInLibraryModules(t *testing.T) {
	const q = "module lib/q\nexport type Q = Some(int) | QN\nexport pure func qmk(i: int) -> Q = if i > 0 then Some(i) else QN\n"
	for _, c := range []struct {
		name, lib string
		want      string // "" = checks
	}{
		{"types and constructors without an import",
			"pure func f(o: Option[int], r: Result[int, string]) -> int = match o { Some(n) => n, None => match r { Ok(m) => m, Err(_) => 0 } }\n" +
				"export pure func run() -> int = f(None, Ok(5))\n", ""},
		{"a helper over Some is generic",
			"export pure func run() -> bool {\n  let isSome = \\o. match o { Some(_) => true, _ => false };\n  isSome(Some(1)) && isSome(Some(\"s\"))\n}\n", ""},
		{"an explicit import of std/option is still valid",
			"import std/option (Option, Some, None)\nexport pure func run() -> Option[int] = Some(1)\n", ""},
		{"a local type declaring Some shadows the Option prelude",
			"type M = Some(int) | Nada\npure func f(m: M) -> int = match m { Some(n) => n, Nada => 0 }\n" +
				"export pure func run() -> int = f(Some(3)) + (match Ok(1) { Ok(n) => n, Err(_) => 0 })\n", ""},
		{"... and drops None with it, as in an entry module",
			"type M = Some(int) | Nada\nexport pure func run(o: Option[int]) -> int = match o { None => 0, _ => 1 }\n", "TC_MATCH_001"},
		{"an explicit import of another type's Some wins",
			"import lib/q (Q, Some, QN)\npure func f(x: Q) -> int = match x { Some(n) => n, QN => 0 }\nexport pure func run() -> int = f(Some(4))\n", ""},
		{"... while Option and None stay the prelude's",
			"import lib/q (Q, Some, QN)\nexport pure func run(o: Option[int]) -> int = match o { Some(n) => n, None => 0 }\n",
			"belongs to ADT 'Q', not 'Option'"},
	} {
		t.Run(c.name, func(t *testing.T) {
			check := byNameFixture(t, map[string]string{
				"lib/q.ail": q,
				"lib/m.ail": "module lib/m\n" + c.lib,
				"main.ail":  "module main\nimport lib/m (run)\nexport pure func go() -> int = 0\n",
			})
			got := check("main.ail")
			switch {
			case c.want == "" && got != "ok":
				t.Fatalf("must check, got %s", got)
			case c.want != "" && !strings.Contains(got, c.want):
				t.Fatalf("want an error containing %q, got %s", c.want, got)
			}
		})
	}
}
