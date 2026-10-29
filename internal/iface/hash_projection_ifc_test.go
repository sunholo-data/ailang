package iface

import (
	"bytes"
	"testing"

	"github.com/sunholo-data/ailang/internal/types"
)

func unlabelledSummary() *types.ImportedIFCSig {
	s := types.EmptyIFCSummary()
	s.Params = []types.IFCParam{{Applied: types.EncodeIFCLabel(types.LabelBottom()), Forbidden: []string{}, Name: "name", Label: types.EncodeIFCLabel(types.LabelBottom())}}
	return s
}

func greetIface(schema string, ifc *types.ImportedIFCSig) *InterfaceJSON {
	return &InterfaceJSON{
		Module: "examples/docs/module_greet",
		Schema: schema,
		Types:  []TypeJSON{},
		Funcs:  []FuncJSON{{IFC: ifc, Name: "greet", Type: "(string)->()!{IO}", Effects: []string{"IO"}, Pure: true}},
	}
}

// A package whose exports carry no labels must keep the identity it had before
// IFC summaries and the v2 iface schema existed; otherwise every publish from a
// binary on the other side of the change is refused with PUB005.
func TestHashProjection_UnlabelledSummaryKeepsPreSummaryIdentity(t *testing.T) {
	before, err := HashProjection(greetIface("ailang.iface/v1", nil))
	if err != nil {
		t.Fatal(err)
	}
	after, err := HashProjection(greetIface(SchemaVersion, unlabelledSummary()))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatalf("unlabelled v2 export changed the identity projection:\nbefore %s\nafter  %s", before, after)
	}
}

// A label is part of the security interface, so each kind of label must change
// the identity projection.
func TestHashProjection_LabelsChangeIdentity(t *testing.T) {
	base, err := HashProjection(greetIface(SchemaVersion, unlabelledSummary()))
	if err != nil {
		t.Fatal(err)
	}
	secret := types.EncodeIFCLabel(types.LabelConst("secret"))
	cases := map[string]func(s *types.ImportedIFCSig){
		"param label":        func(s *types.ImportedIFCSig) { s.Params[0].Label = secret },
		"applied":            func(s *types.ImportedIFCSig) { s.Params[0].Applied = secret },
		"multiple forbidden": func(s *types.ImportedIFCSig) { s.Params[0].Forbidden = []string{"email", "secret"} },
		"forbidden sink":     func(s *types.ImportedIFCSig) { s.Params[0].Forbidden = []string{"secret"} },
		"declared return":    func(s *types.ImportedIFCSig) { s.Return, s.HasReturnLabel = secret, true },
		"intrinsic":          func(s *types.ImportedIFCSig) { s.Intrinsic = secret },
		"declassify":         func(s *types.ImportedIFCSig) { s.Declassify = []string{"secret"} },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			s := unlabelledSummary()
			mutate(s)
			if s.IsUnlabelled() {
				t.Fatal("instrument failure: mutated summary still reports unlabelled")
			}
			got, err := HashProjection(greetIface(SchemaVersion, s))
			if err != nil {
				t.Fatal(err)
			}
			if bytes.Equal(base, got) {
				t.Fatalf("%s did not change the identity projection", name)
			}
		})
	}
}

func TestIsUnlabelled(t *testing.T) {
	if (*types.ImportedIFCSig)(nil).IsUnlabelled() {
		t.Error("nil summary is missing metadata, not unlabelled")
	}
	if !types.EmptyIFCSummary().IsUnlabelled() {
		t.Error("empty summary should be unlabelled")
	}
	s := unlabelledSummary()
	s.Params[0].Name = "renamed"
	if !s.IsUnlabelled() {
		t.Error("a parameter name is not security information")
	}
}

func TestHashProjectionRejectsMalformedIFC(t *testing.T) {
	for _, mutate := range []func(*types.ImportedIFCSig){
		func(s *types.ImportedIFCSig) { s.Params[0].Forbidden = nil },
		func(s *types.ImportedIFCSig) { s.Params[0].Forbidden = []string{"secret", "email"} },
		func(s *types.ImportedIFCSig) { s.Params[0].Applied = types.IFCLabel{} },
	} {
		s := unlabelledSummary()
		mutate(s)
		if s.IsUnlabelled() {
			t.Fatal("malformed security metadata reported clean")
		}
		if _, err := HashProjection(greetIface(SchemaVersion, s)); err == nil {
			t.Fatal("malformed security metadata hashed")
		}
	}
	if _, err := HashProjection(greetIface(SchemaVersion, nil)); err == nil {
		t.Fatal("missing v2 metadata hashed")
	}
}
