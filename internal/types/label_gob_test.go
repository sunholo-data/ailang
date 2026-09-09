package types

import (
	"bytes"
	"encoding/gob"
	"testing"
)

// M-TAINT-TYPES: TLabelled must survive a gob round trip. Without a codec, every
// compile of an IFC-labelled module failed its cache write and silently fell back
// to fresh compilation (inbox_1788847828665_f0c5db58) — a correctness-preserving
// but permanently-cold cache.
//
// The label implementations are unexported value types with unexported fields,
// which gob refuses, so registering them would not have been enough: the encoder
// would find the type and serialise an empty label. These tests therefore assert
// the label's CONTENT survives, not merely that encoding returns no error.
func TestTLabelledGobRoundTrip(t *testing.T) {
	cases := []struct {
		name string
		in   *TLabelled
	}{
		{"const label", &TLabelled{Inner: &TCon{Name: "string"}, L: LabelConst("secret")}},
		{"different label", &TLabelled{Inner: &TCon{Name: "string"}, L: LabelConst("client")}},
		{"non-string inner", &TLabelled{Inner: &TCon{Name: "int"}, L: LabelConst("pii")}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := gob.NewEncoder(&buf).Encode(tc.in); err != nil {
				t.Fatalf("encode: %v", err)
			}
			var got *TLabelled
			if err := gob.NewDecoder(&buf).Decode(&got); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if got.String() != tc.in.String() {
				t.Errorf("round trip changed the type: got %q, want %q", got.String(), tc.in.String())
			}
			if got.L == nil {
				t.Fatal("label came back nil — the codec dropped it")
			}
			if got.L.String() != tc.in.L.String() {
				t.Errorf("label content lost: got %q, want %q", got.L.String(), tc.in.L.String())
			}
		})
	}
}

// TestTLabelledGobThroughInterface: the cache encodes types as the Type
// interface, not as a concrete *TLabelled, which is the path that actually
// failed in the field.
func TestTLabelledGobThroughInterface(t *testing.T) {
	var in Type = &TLabelled{Inner: &TCon{Name: "string"}, L: LabelConst("secret")}
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(&in); err != nil {
		t.Fatalf("encode through Type interface: %v", err)
	}
	var got Type
	if err := gob.NewDecoder(&buf).Decode(&got); err != nil {
		t.Fatalf("decode through Type interface: %v", err)
	}
	lab, ok := got.(*TLabelled)
	if !ok {
		t.Fatalf("came back as %T, want *TLabelled", got)
	}
	if lab.L == nil || lab.L.String() != LabelConst("secret").String() {
		t.Errorf("label lost through the interface path: %v", lab.L)
	}
}
