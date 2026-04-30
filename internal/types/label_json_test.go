package types

import (
	"encoding/json"
	"testing"
)

// roundTripLabel marshals a Label to JSON and unmarshals it back.
func roundTripLabel(t *testing.T, l Label) Label {
	t.Helper()
	data, err := MarshalLabel(l)
	if err != nil {
		t.Fatalf("MarshalLabel(%v): %v", l, err)
	}
	got, err := UnmarshalLabel(data)
	if err != nil {
		t.Fatalf("UnmarshalLabel(%s): %v", data, err)
	}
	return got
}

func TestLabelJSON_Bottom(t *testing.T) {
	got := roundTripLabel(t, LabelBottom())
	if !LabelEqual(got, LabelBottom()) {
		t.Errorf("⊥ round-trip failed: got %v", got)
	}
}

func TestLabelJSON_Const(t *testing.T) {
	in := LabelConst("email")
	got := roundTripLabel(t, in)
	if !LabelEqual(got, in) {
		t.Errorf("<email> round-trip failed: got %v", got)
	}
}

func TestLabelJSON_Var(t *testing.T) {
	in := LabelVar("α")
	got := roundTripLabel(t, in)
	if !LabelEqual(got, in) {
		t.Errorf("var round-trip failed: got %v", got)
	}
}

func TestLabelJSON_Join(t *testing.T) {
	in := LabelJoin(LabelConst("email"), LabelConst("user"))
	got := roundTripLabel(t, in)
	if !LabelEqual(got, in) {
		t.Errorf("join round-trip failed: got %v want %v", got, in)
	}
}

func TestLabelJSON_NestedJoin(t *testing.T) {
	// Three constants — flattens to canonical form on join.
	in := LabelJoin(LabelJoin(LabelConst("a"), LabelConst("b")), LabelConst("c"))
	got := roundTripLabel(t, in)
	if !LabelEqual(got, in) {
		t.Errorf("nested join round-trip failed: got %v want %v", got, in)
	}
}

func TestLabelJSON_NilDefaultsToBottom(t *testing.T) {
	// nil/null/missing data must default to ⊥ for backwards compat with
	// pre-label ifaces.
	cases := [][]byte{nil, []byte("null"), []byte("")}
	for _, data := range cases {
		got, err := UnmarshalLabel(data)
		if err != nil {
			t.Errorf("UnmarshalLabel(%q) error: %v", data, err)
			continue
		}
		if !LabelEqual(got, LabelBottom()) {
			t.Errorf("UnmarshalLabel(%q) = %v, want ⊥", data, got)
		}
	}
}

// TestTLabelledJSON_RoundTrip verifies that a labelled type wraps and
// unwraps cleanly through Type marshal/unmarshal.
func TestTLabelledJSON_RoundTrip(t *testing.T) {
	in := &TLabelled{Inner: TString, L: LabelConst("email")}
	data, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	got, err := UnmarshalType(data)
	if err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	gotL, ok := got.(*TLabelled)
	if !ok {
		t.Fatalf("expected *TLabelled, got %T", got)
	}
	if !gotL.Inner.Equals(in.Inner) {
		t.Errorf("inner mismatch: got %v want %v", gotL.Inner, in.Inner)
	}
	if !LabelEqual(gotL.L, in.L) {
		t.Errorf("label mismatch: got %v want %v", gotL.L, in.L)
	}
}

// TestTLabelledJSON_InScheme verifies that a labelled return type survives
// the same Scheme round-trip used by iface caching.
func TestTLabelledJSON_InScheme(t *testing.T) {
	// fetchMail :: () -> string<email>
	scheme := &Scheme{
		TypeVars: nil,
		Type: &TFunc2{
			Params:    nil,
			EffectRow: EmptyEffectRow(),
			Return:    &TLabelled{Inner: TString, L: LabelConst("email")},
		},
	}

	data, err := MarshalScheme(scheme)
	if err != nil {
		t.Fatalf("MarshalScheme: %v", err)
	}

	got, err := UnmarshalScheme(data)
	if err != nil {
		t.Fatalf("UnmarshalScheme: %v", err)
	}

	fn, ok := got.Type.(*TFunc2)
	if !ok {
		t.Fatalf("expected *TFunc2 return, got %T", got.Type)
	}
	ret, ok := fn.Return.(*TLabelled)
	if !ok {
		t.Fatalf("expected *TLabelled return, got %T", fn.Return)
	}
	if !LabelEqual(ret.L, LabelConst("email")) {
		t.Errorf("return label = %v, want <email>", ret.L)
	}
}

// TestTLabelledJSON_BackwardsCompat verifies that an iface JSON without
// any label fields (the legacy format) loads as plain types — no labels
// appear, and LabelOf returns ⊥.
func TestTLabelledJSON_BackwardsCompat(t *testing.T) {
	// Old-style iface JSON for a plain string return — no tlabelled wrapper.
	legacy := []byte(`{"tag":"tcon","data":{"Name":"string"}}`)
	got, err := UnmarshalType(legacy)
	if err != nil {
		t.Fatalf("UnmarshalType(legacy): %v", err)
	}
	if _, isLabelled := got.(*TLabelled); isLabelled {
		t.Errorf("plain TCon should NOT decode as TLabelled")
	}
	if !LabelEqual(LabelOf(got), LabelBottom()) {
		t.Errorf("LabelOf(plain) = %v, want ⊥", LabelOf(got))
	}
}
