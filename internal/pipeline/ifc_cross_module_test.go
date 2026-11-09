package pipeline

import (
	"os"
	"strings"
	"testing"
)

// M-IFC-CROSS-MODULE M3 (#1134): a library's IFC contract — sink refinements,
// positive parameter labels, declared/intrinsic return labels and Declassify
// authority — binds its importers exactly as it binds callers in its own module.

const ifcXLib = `module lib
import std/secret (secret)
export pure func gate(body: string{not email}) -> string { body }
export pure func fromInbox(raw: string) -> string<email> { raw }
export func fetch(ref: string) -> string ! {Secret} { secret(ref, "test") }
export pure func q(s: string<sqlsafe>) -> string { s }
export func scrub(raw: string) -> string ! {Declassify[label=email]} { raw }
export func bare(raw: string) -> string ! {Declassify} { raw }
export pure func ident(x: string) -> string { x }
export func apply(f: string -> string, x: string) -> string { f(x) }
`

// mid re-exports lib's taint through an unannotated wrapper.
const ifcXMid = `module mid
import lib (fetch)
export func get(ref: string) -> string ! {Secret} { fetch(ref) }
`

type ifcXRow struct {
	name string
	app  string
	want string // "" = must compile; otherwise a substring of the IFC error
}

var ifcXRows = []ifcXRow{
	{"selective sink (#1134 repro)", `import lib (gate)
pure func src(raw: string) -> string<email> { raw }
export func main() -> string { gate(src("evil")) }`, `reaches parameter "body" of imported function lib.gate`},
	{"imported source into imported sink", `import lib (gate, fromInbox)
export func main() -> string { gate(fromInbox("evil")) }`, "forbids it (string{not email})"},
	{"renamed import", `import lib (gate as g, fromInbox)
export func main() -> string { g(fromInbox("evil")) }`, "imported function lib.gate"},
	{"qualified module alias", `import lib as L
export func main() -> string { L.gate(L.fromInbox("evil")) }`, "imported function lib.gate"},
	{"declared return into local sink", `import lib (fromInbox)
pure func sink(b: string{not email}) -> string { b }
export func main() -> string { sink(fromInbox("evil")) }`, `parameter "b" which forbids it`},
	{"intrinsic return into local sink", `import lib (fetch)
pure func sink(b: string{not secret}) -> string { b }
export func main() -> string ! {Secret} { sink(fetch("op://x")) }`, "labelled <secret>"},
	{"transitive intrinsic through mid", `import mid (get)
pure func sink(b: string{not secret}) -> string { b }
export func main() -> string ! {Secret} { sink(get("op://x")) }`, "labelled <secret>"},
	{"imported Check C", `import lib (q, fromInbox)
export func main() -> string { q(fromInbox("evil")) }`, "does not cover it"},
	{"transparent import joins args", `import lib (gate, ident, fromInbox)
export func main() -> string { gate(ident(fromInbox("evil"))) }`, "forbids it"},
	{"closure body taint", `import lib (gate, fromInbox)
export func main() -> string { let f = \u. fromInbox("evil") in gate(f(0)) }`, "forbids it"},
	{"local unannotated wrapper", `import lib (gate, fromInbox)
func forward(x: string) -> string { gate(x) }
export func main() -> string { forward(fromInbox("evil")) }`, "forbids it"},
	{"imported sink as HOF argument", `import lib (apply, gate, fromInbox)
export func main() -> string { apply(gate, fromInbox("evil")) }`, "forbids it"},
	{"imported sink via alias", `import lib (gate, fromInbox)
export func main() -> string { let g = gate in g(fromInbox("evil")) }`, "forbids it"},
	{"scoped declassifier does not cover other labels", `import lib (scrub, fetch)
pure func sink(b: string{not secret}) -> string { b }
export func main() -> string ! {Secret, Declassify[label=email]} { sink(scrub(fetch("op://x"))) }`, "labelled <secret>"},

	{"clean literal into imported sink", `import lib (gate)
export func main() -> string { gate("hello") }`, ""},
	{"literal into imported positive label", `import lib (q)
export func main() -> string { q("select 1") }`, ""},
	{"scoped imported declassify", `import lib (gate, scrub, fromInbox)
export func main() -> string ! {Declassify[label=email]} { gate(scrub(fromInbox("evil"))) }`, ""},
	{"bare imported declassify", `import lib (gate, bare, fromInbox)
export func main() -> string ! {Declassify} { gate(bare(fromInbox("evil"))) }`, ""},
	{"lexical shadow of imported sink", `import lib (gate, fromInbox)
export func main() -> string { let gate = \x. x in gate(fromInbox("evil")) }`, ""},
	{"unrelated label into imported sink", `import lib (gate, fetch)
export func main() -> string ! {Secret} { gate(fetch("op://x")) }`, ""},
}

func ifcXFixture(t *testing.T, app string) {
	t.Helper()
	t.Chdir(t.TempDir())
	t.Setenv("AILANG_NO_CACHE", "1")
	t.Setenv("AILANG_STDLIB_PATH", "")
	for name, src := range map[string]string{"lib.ail": ifcXLib, "mid.ail": ifcXMid, "app.ail": "module app\n" + app + "\n"} {
		if err := os.WriteFile(name, []byte(src), 0600); err != nil {
			t.Fatal(err)
		}
	}
}

func TestIFCCrossModuleContracts(t *testing.T) {
	for _, row := range ifcXRows {
		t.Run(row.name, func(t *testing.T) {
			ifcXFixture(t, row.app)
			_, err := Run(Config{Mode: ModeCheck}, Source{Filename: "app.ail"})
			if row.want == "" {
				if err != nil {
					t.Fatalf("clean control rejected: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("leak accepted; want IFC error containing %q", row.want)
			}
			msg := err.Error()
			if !strings.Contains(msg, "information-flow violation") || !strings.Contains(msg, row.want) {
				t.Fatalf("want IFC error containing %q, got: %v", row.want, err)
			}
			if !strings.Contains(msg, "app.ail:") {
				t.Fatalf("error does not name the caller location: %v", err)
			}
		})
	}
}

// A dependency served from the compile cache still binds its importer: the
// cached interface carries the summary the caller's gate reads.
func TestIFCCrossModuleWarmDependency(t *testing.T) {
	ifcXFixture(t, "import lib (gate)\nexport func main() -> string { gate(\"hello\") }")
	t.Setenv("AILANG_NO_CACHE", "")
	t.Setenv("AILANG_CACHE_DIR", "")
	cfg := Config{Mode: ModeCheck}
	src := Source{Filename: "app.ail"}
	if _, err := runModuleWithCacheDependencies(t.Context(), cfg, src, productionCacheDependencies()); err != nil {
		t.Fatalf("cold clean caller: %v", err)
	}
	leak := "module app\nimport lib (gate, fromInbox)\nexport func main() -> string { gate(fromInbox(\"evil\")) }\n"
	if err := os.WriteFile("app.ail", []byte(leak), 0600); err != nil {
		t.Fatal(err)
	}
	for _, run := range []string{"warm lib, edited caller", "warm lib, warm caller attempt"} {
		_, err := runModuleWithCacheDependencies(t.Context(), cfg, src, productionCacheDependencies())
		if err == nil || !strings.Contains(err.Error(), "imported function lib.gate") {
			t.Fatalf("%s: leak accepted or wrong error: %v", run, err)
		}
	}
}

// The published cross-module examples keep their documented verdicts.
func TestIFCCrossModuleExamples(t *testing.T) {
	t.Chdir("../..")
	t.Setenv("AILANG_NO_CACHE", "1")
	for _, tc := range []struct{ file, want string }{
		{"examples/runnable/secrets/ifc_xmod_lib.ail", ""},
		{"examples/runnable/secrets/ifc_xmod_app.ail", ""},
		{"examples/runnable/secrets/ifc_xmod_leak.ail", `reaches parameter "body" of imported function examples/runnable/secrets/ifc_xmod_lib.deliver`},
	} {
		_, err := Run(Config{Mode: ModeCheck}, Source{Filename: tc.file})
		switch {
		case tc.want == "" && err != nil:
			t.Errorf("%s: %v", tc.file, err)
		case tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)):
			t.Errorf("%s: want IFC error containing %q, got %v", tc.file, tc.want, err)
		}
	}
}
