package iface

import (
	"bytes"
	"strings"
	"testing"
)

// #2290: class constraints are part of an export's interface. Adding one
// rejects callers that passed, so both identities — interface_hash
// (HashProjection) and the release classifier's SignatureSet — must see it,
// and an export without constraints must hash and sign exactly as before.
func TestConstraintsChangeIdentity(t *testing.T) {
	plain := &InterfaceJSON{Module: "q/lib", Schema: "ailang.iface/v1",
		Funcs: []FuncJSON{{Name: "eqq", Type: "(a,a)->bool", Effects: []string{}, Pure: true}}}
	constrained := &InterfaceJSON{Module: "q/lib", Schema: "ailang.iface/v1",
		Funcs: []FuncJSON{{Name: "eqq", Type: "(a,a)->bool", Constraints: []string{"Eq[a]"}, Effects: []string{}, Pure: true}}}

	hp, err := HashProjection(plain)
	if err != nil {
		t.Fatal(err)
	}
	hc, err := HashProjection(constrained)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(hp, hc) {
		t.Fatalf("a constraint must change the hash projection: %s", hp)
	}
	if strings.Contains(string(hp), "constraints") {
		t.Errorf("an unconstrained export must project as before (no constraints key): %s", hp)
	}

	sp, sc := SignatureSet(plain), SignatureSet(constrained)
	if sp[0] != "q/lib:func:eqq:(a\\,a)->bool:" {
		t.Errorf("an unconstrained signature must be unchanged, got %q", sp[0])
	}
	if sp[0] == sc[0] || !strings.Contains(sc[0], "Eq[a] => ") {
		t.Errorf("a constraint must change the signature: %q vs %q", sp[0], sc[0])
	}
}
