package repl

import (
	"strings"
	"testing"
)

// #2273 on the REPL/WASM module registry (module_registry_load.go): a module
// it loads gets the same rigid-signature check as the CLI pipeline.
func TestModuleRegistry_RigidSignature(t *testing.T) {
	reg := NewModuleRegistry()
	_, err := reg.LoadModule("rig", "module rig\n\nexport pure func toInt[a](x: a) -> int = x\n")
	if err == nil || !strings.Contains(err.Error(), "TC_RIGID_001") {
		t.Fatalf("want TC_RIGID_001, got %v", err)
	}
	ok := NewModuleRegistry()
	if _, err := ok.LoadModule("ok", "module ok\n\nexport pure func ident[a](x: a) -> a = x\n"); err != nil {
		t.Fatalf("correct module rejected: %v", err)
	}
	// The scheme the registry keeps reads as written: `a`, never the
	// declaration-unique inference name `a@{ident}`.
	exp, err := ok.GetExport("ok", "ident")
	if err != nil || exp.Scheme == nil {
		t.Fatalf("ident not exported: %v", err)
	}
	if s := exp.Scheme.String(); strings.Contains(s, "@{") || !strings.Contains(s, "a") {
		t.Errorf("scheme %q does not use the written name", s)
	}
}
