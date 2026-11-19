package repl

import (
	"strings"
	"testing"
)

// M-IFC-CROSS-MODULE M3 (#1134): the browser path checks an importer against
// the IFC contracts of modules registered earlier, with the CLI's verdicts
// (internal/pipeline/ifc_cross_module_test.go holds the full matrix).
const replIFCLib = `module lib
export func gate(body: string{not email}) -> string ! {} { body }
export func fromInbox(raw: string) -> string<email> ! {} { raw }
export func q(s: string<sqlsafe>) -> string ! {} { s }
export func scrub(raw: string) -> string ! {Declassify[label=email]} { raw }
export func ident(x: string) -> string ! {} { x }
`

// mid wraps lib's labelled source: its exported summary must carry the taint.
const replIFCMid = `module mid
import lib (fromInbox)
export func inbox(raw: string) -> string ! {} { fromInbox(raw) }
`

func TestLoadModule_IFCCrossModule(t *testing.T) {
	for _, tc := range []struct {
		name, app, want string
	}{
		{"selective sink", "import lib (gate, fromInbox)\nexport func leak() -> string ! {} { gate(fromInbox(\"evil\")) }", "imported function lib.gate"},
		{"renamed", "import lib (gate as g, fromInbox)\nexport func leak() -> string ! {} { g(fromInbox(\"evil\")) }", "imported function lib.gate"},
		{"qualified", "import lib as L\nexport func leak() -> string ! {} { L.gate(L.fromInbox(\"evil\")) }", "imported function lib.gate"},
		{"declared return into local sink", "import lib (fromInbox)\nfunc sink(b: string{not email}) -> string ! {} { b }\nexport func leak() -> string ! {} { sink(fromInbox(\"evil\")) }", "forbids it"},
		{"transitive taint through mid", "import mid (inbox)\nimport lib (gate)\nexport func leak() -> string ! {} { gate(inbox(\"evil\")) }", "forbids it"},
		{"imported Check C", "import lib (q, fromInbox)\nexport func leak() -> string ! {} { q(fromInbox(\"evil\")) }", "does not cover it"},
		{"transparent import", "import lib (gate, ident, fromInbox)\nexport func leak() -> string ! {} { gate(ident(fromInbox(\"evil\"))) }", "forbids it"},
		{"clean literal", "import lib (gate)\nexport func ok() -> string ! {} { gate(\"hello\") }", ""},
		{"scoped declassify", "import lib (gate, scrub, fromInbox)\nexport func ok() -> string ! {Declassify[label=email]} { gate(scrub(fromInbox(\"evil\"))) }", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reg := NewModuleRegistry()
			for _, dep := range []struct{ name, src string }{{"lib", replIFCLib}, {"mid", replIFCMid}} {
				if _, err := reg.LoadModule(dep.name, dep.src); err != nil {
					t.Fatalf("load %s: %v", dep.name, err)
				}
			}
			_, err := reg.LoadModule("app", "module app\n"+tc.app+"\n")
			if tc.want == "" {
				if err != nil {
					t.Fatalf("clean importer rejected: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), "information-flow violation") || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("want information-flow error containing %q, got %v", tc.want, err)
			}
			if _, published := reg.modules["app"]; published {
				t.Fatal("rejected importer published its exports")
			}
		})
	}
}
