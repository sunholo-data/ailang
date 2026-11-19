package iface

import (
	"bytes"
	"testing"
)

// Derived purity (#1443) must not move package identity: the effect row and type
// that determine it are already hashed, and the pre-#1443 flag was always true.
func TestHashProjection_PurityDoesNotChangeIdentity(t *testing.T) {
	mk := func(pure bool) *InterfaceJSON {
		return &InterfaceJSON{
			Module: "examples/docs/module_greet",
			Schema: "ailang.iface/v1",
			Types:  []TypeJSON{},
			Funcs:  []FuncJSON{{Name: "greet", Type: "(string)->()!{IO}", Effects: []string{"IO"}, Pure: pure}},
		}
	}
	stub, err := HashProjection(mk(true))
	if err != nil {
		t.Fatal(err)
	}
	derived, err := HashProjection(mk(false))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(stub, derived) {
		t.Fatalf("purity changed the identity projection:\nstub    %s\nderived %s", stub, derived)
	}
	widened := mk(false)
	widened.Funcs[0].Effects = []string{"FS", "IO"}
	got, err := HashProjection(widened)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(stub, got) {
		t.Fatal("an effect-row change must still change the identity projection")
	}
}
