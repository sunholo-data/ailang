package pipeline

import (
	"regexp"
	"strings"
	"testing"
)

// #2371's one accepted→rejected case: a library matching, by name and
// without importing its type, a user type's constructor called Ok, Err, None
// or Some now gets the prelude's constructor. The error names the one-step
// fix when a type in the import closure declares the constructor, and
// applying it verbatim makes the program check.

const preludeHintLibs = "module lib/st\nexport type Status = Ok | Err(string) | Pending\nexport type Maybe = Just(int) | None\n" +
	"export pure func stat(i: int) -> Status = if i == 0 then Ok else if i == 1 then Err(\"bad\") else Pending\n" +
	"export pure func mb(i: int) -> Maybe = if i > 0 then Just(i) else None\n"

var hintImport = regexp.MustCompile(`import it: (import \S+ \([^)]*\))`)

func TestPreludeShadowedByNameCtorNamesTheImport(t *testing.T) {
	for _, c := range []struct {
		name, body, want, fix string
	}{
		{"a nullary Ok of the user's type (arity error)",
			"pure func f(i: int) -> int = match ws(i) { Ok => 1, Err(_) => 2, _ => 3 }\n",
			"constructor Ok in pattern expects 1 argument(s)",
			"import lib/st (Status, Err, Ok, Pending)"},
		{"None of the user's type (foreign-constructor error)",
			"pure func f(i: int) -> int = match wm(i) { Just(v) => v, None => 0 }\n",
			"match arm constructor 'None' belongs to ADT 'Option', not 'Maybe'",
			"import lib/st (Maybe, Just, None)"},
	} {
		t.Run(c.name, func(t *testing.T) {
			probe := func(imports string) string {
				return "module probe\nimport lib/w (ws, wm)\n" + imports + c.body +
					"export pure func summary() -> string = show([f(0), f(1), f(2)])\n"
			}
			files := map[string]string{
				"lib/st.ail": preludeHintLibs,
				"lib/w.ail":  "module lib/w\nimport lib/st (stat, mb)\nexport pure func ws(i: int) = stat(i)\nexport pure func wm(i: int) = mb(i)\n",
				"probe.ail":  probe(""),
			}
			got := byNameFixture(t, files)("probe.ail")
			if !strings.Contains(got, c.want) {
				t.Fatalf("want %q, got %s", c.want, got)
			}
			m := hintImport.FindStringSubmatch(got)
			if m == nil || m[1] != c.fix {
				t.Fatalf("want the hint %q, got %s", c.fix, got)
			}
			files["probe.ail"] = probe(m[1] + "\n") // apply the hint verbatim
			if fixed := byNameFixture(t, files)("probe.ail"); fixed != "ok" {
				t.Fatalf("applying %q must make it check, got %s", m[1], fixed)
			}
		})
	}
}

// No hint when no type in the closure declares the name: the plain errors
// are unchanged.
func TestPreludeArityErrorWithoutClosureNamesakeHasNoHint(t *testing.T) {
	got := byNameFixture(t, map[string]string{
		"probe.ail": "module probe\npure func f(r: Result[int, string]) -> int = match r { Ok => 1, _ => 2 }\n",
	})("probe.ail")
	if !strings.Contains(got, "constructor Ok in pattern expects 1 argument(s)") || strings.Contains(got, "import it") {
		t.Fatalf("want the plain arity error, got %s", got)
	}
	// ... nor when the closure's only other Ok takes arguments too: importing
	// it would not make a bare `Ok` pattern right.
	got = byNameFixture(t, map[string]string{
		"lib/r.ail": "module lib/r\nexport type R2 = Ok(int) | Bad\nexport pure func r2() -> R2 = Bad\n",
		"probe.ail": "module probe\nimport lib/r (r2)\npure func f(r: Result[int, string]) -> int = match r { Ok => 1, _ => 2 }\n",
	})("probe.ail")
	if !strings.Contains(got, "constructor Ok in pattern expects 1 argument(s)") || strings.Contains(got, "import it") {
		t.Fatalf("want the plain arity error (lib/r's Ok takes an argument), got %s", got)
	}
}

// A file of top-level expressions gets no prelude (#2372): an expression
// using a prelude name is "undefined variable" with the import that fixes it,
// as getOrElse already had.
func TestUndefinedPreludeNameNamesItsImport(t *testing.T) {
	for name, want := range map[string]string{
		"Some(1)": "import std/option (Option, Some, None)",
		"None":    "import std/option (Option, Some, None)",
		"Ok(1)":   "import std/result (Result, Ok, Err)",
		"Err(1)":  "import std/result (Result, Ok, Err)",
	} {
		got := byNameFixture(t, map[string]string{"probe.ail": "module probe\n\nlet x = " + name + " in x\n"})("probe.ail")
		if !strings.Contains(got, "undefined variable") || !strings.Contains(got, "add `"+want+"`") {
			t.Errorf("%s: want undefined variable with `%s`, got %s", name, want, got)
		}
	}
}
