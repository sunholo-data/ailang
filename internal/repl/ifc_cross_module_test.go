package repl

import (
	"strings"
	"testing"
)

// M-IFC-CROSS-MODULE M3 (#1134): the REPL / WASM module registry enforces
// imported IFC contracts with the same verdicts as the CLI
// (internal/pipeline/cache_ifc_imported_test.go), and a rejected module
// publishes nothing.
const replIFCLib = `module lib
export func gate(input: string{not secret}) -> bool ! {} { input != "" }
export func leak() -> string<secret> ! {} { "hunter2" }
export func wrapped() -> string ! {} { leak() }
export func redact(s: string) -> string ! {Declassify} { "***" }
`

func TestLoadModule_ImportedIFCContracts(t *testing.T) {
	for _, tc := range []struct {
		name, body, want string
	}{
		{"imported sink", `import lib (gate, leak)
export func main() -> bool ! {} { gate(leak()) }`, `of imported function lib.gate, which forbids it`},
		{"renamed import", `import lib (gate as check, wrapped)
export func main() -> bool ! {} { check(wrapped()) }`, `of imported function lib.gate, which forbids it`},
		{"module alias", `import lib as L
export func main() -> bool ! {} { L.gate(L.leak()) }`, `of imported function lib.gate, which forbids it`},
		{"imported result into local sink", `import lib (leak)
func sink(x: string{not secret}) -> string ! {} { x }
export func main() -> string ! {} { sink(leak()) }`, `reaches parameter "x" which forbids it`},
		{"clean literal", `import lib (gate)
export func main() -> bool ! {} { gate("ok") }`, ""},
		{"declassified", `import lib (gate, leak, redact)
export func main() -> bool ! {} { gate(redact(leak())) }`, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reg := NewModuleRegistry()
			if _, err := reg.LoadModule("lib", replIFCLib); err != nil {
				t.Fatalf("load lib: %v", err)
			}
			if reg.modules["lib"].Exports["gate"].IFC.Params[0].Forbidden[0] != "secret" {
				t.Fatal("lib export lost its IFC summary")
			}
			_, err := reg.LoadModule("app", "module app\n"+tc.body+"\n")
			if tc.want == "" {
				if err != nil {
					t.Fatalf("clean module rejected: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("want %q, got %v", tc.want, err)
			}
			if _, published := reg.modules["app"]; published {
				t.Fatal("a rejected module must publish nothing")
			}
		})
	}
}
