package repl

import (
	"strings"
	"testing"
)

// #1719: the browser path gives the same verdict as the CLI for a sink reached
// through a function value (alias, higher-order parameter, wrapper lambda),
// because both call types.CheckModuleIFC. Before the fix all three loaded.
func TestLoadModule_IFCFunctionValueSink_Rejected(t *testing.T) {
	prelude := `
func fromInbox(s: string) -> string<email> ! {} { s }
func gate(body: string{not email}) -> string ! {} { body }
func apply1(f: string -> string, x: string) -> string ! {} { f(x) }
`
	for name, body := range map[string]string{
		"alias":  `let g = gate in g(fromInbox("hi"))`,
		"hof":    `apply1(gate, fromInbox("hi"))`,
		"lambda": `let w = \y. gate(y) in w(fromInbox("hi"))`,
	} {
		t.Run(name, func(t *testing.T) {
			reg := NewModuleRegistry()
			src := "module fv_" + name + "\n" + prelude + "func leak() -> string ! {} { " + body + " }\n"
			if _, err := reg.LoadModule("fv_"+name, src); err == nil || !strings.Contains(err.Error(), "information-flow") {
				t.Fatalf("LoadModule must reject %s with an information-flow error, got %v", body, err)
			}
		})
	}
	reg := NewModuleRegistry()
	clean := "module fv_clean\n" + prelude + `func ok() -> string ! {} { let g = gate in g(apply1(gate, "clean")) }` + "\n"
	if _, err := reg.LoadModule("fv_clean", clean); err != nil {
		t.Fatalf("LoadModule rejected a clean function-value module: %v", err)
	}
}
