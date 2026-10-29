package iface

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/types"
)

func TestIFCSummaryBuilder(t *testing.T) {
	env := types.NewTypeEnv()
	env = env.ExtendScheme("gate", &types.Scheme{Type: &types.TFunc2{Params: []types.Type{types.TString}, Return: types.TString}})
	prog := &core.Program{Decls: []core.CoreExpr{&core.Let{Name: "gate", Value: &core.Lit{Kind: core.StringLit, Value: ""}}}, Meta: map[string]*core.DeclMeta{"gate": {IsExport: true}}}
	file := &ast.File{Funcs: []*ast.FuncDecl{{Name: "gate", Params: []*ast.Param{{Name: "input", Type: &ast.LabelledType{Base: &ast.SimpleType{Name: "string"}, Refinement: &ast.RefinementExpr{NotLabel: "email"}}}}, ReturnType: &ast.SimpleType{Name: "string"}, Body: &ast.FuncCall{Func: &ast.Identifier{Name: "producer"}}}}}
	imported := types.EmptyIFCSummary()
	imported.Intrinsic = types.EncodeIFCLabel(types.LabelConst("secret"))
	got, err := NewBuilder("test/lib", env).WithIFCImports(map[string]*types.ImportedIFCSig{"producer": imported}).Build(prog, nil, file)
	if err != nil {
		t.Fatal(err)
	}
	s := got.Exports["gate"].IFC
	if got.Schema != SchemaVersion || s.Params[0].Forbidden[0] != "email" || !types.LabelSubsumes(s.Intrinsic.Decode(), types.LabelConst("secret")) {
		t.Fatalf("lost metadata: %+v", s)
	}
}

func TestIFCSummaryDigestAndCodecs(t *testing.T) {
	base := NewIface("test/lib")
	base.AddExport("f", &types.Scheme{Type: &types.TFunc2{Params: []types.Type{types.TString}, Return: types.TString}}, true)
	base.AddExport("clean", &types.Scheme{Type: types.TInt}, true)
	s := base.Exports["f"].IFC
	s.Params = []types.IFCParam{{Applied: types.EncodeIFCLabel(types.LabelBottom()), Name: "x", Label: types.EncodeIFCLabel(types.LabelConst("email")), Forbidden: []string{"email", "secret"}}}
	s.Params[0].Applied = types.EncodeIFCLabel(types.LabelJoin(types.LabelConst("arg"), types.LabelVar("L")))
	s.Return = types.EncodeIFCLabel(types.LabelConst("clean"))
	s.HasReturnLabel = true
	s.Intrinsic = types.EncodeIFCLabel(types.LabelJoin(types.LabelConst("secret"), types.LabelVar("L")))
	s.Declassify = []string{"email"}
	b := NewBuilder(base.Module, nil)
	digest, err := b.computeDigest(base)
	if err != nil {
		t.Fatal(err)
	}
	mutations := []func(*types.ImportedIFCSig){
		func(s *types.ImportedIFCSig) { s.Params[0].Applied = types.EncodeIFCLabel(types.LabelConst("email")) },
		func(s *types.ImportedIFCSig) { s.Params[0].Name = "other" },
		func(s *types.ImportedIFCSig) { s.Params[0].Forbidden = []string{"other"} },
		func(s *types.ImportedIFCSig) { s.Params[0].Label = types.EncodeIFCLabel(types.LabelConst("other")) },
		func(s *types.ImportedIFCSig) { s.Return = types.EncodeIFCLabel(types.LabelConst("other")) },
		func(s *types.ImportedIFCSig) { s.HasReturnLabel = false },
		func(s *types.ImportedIFCSig) { s.Intrinsic = types.EncodeIFCLabel(types.LabelConst("other")) },
		func(s *types.ImportedIFCSig) { s.Declassify = []string{"*"} },
	}
	for n, mutate := range mutations {
		raw, _ := json.Marshal(s)
		var copy types.ImportedIFCSig
		if err := json.Unmarshal(raw, &copy); err != nil {
			t.Fatal(err)
		}
		mutate(&copy)
		base.Exports["f"].IFC = &copy
		got, err := b.computeDigest(base)
		if err != nil || got == digest {
			t.Fatalf("field %d absent from digest: %v", n, err)
		}
	}
	base.Exports["f"].IFC = s
	swapped := NewIface(base.Module)
	swapped.Exports["clean"] = base.Exports["clean"]
	swapped.Exports["f"] = base.Exports["f"]
	if got, _ := b.computeDigest(swapped); got != digest {
		t.Fatal("map order changed digest")
	}
	raw, err := base.ToNormalizedJSON()
	if err != nil {
		t.Fatal(err)
	}
	var compact InterfaceJSON
	if err := json.Unmarshal(raw, &compact); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(compact.Funcs[1].IFC, s) {
		t.Fatal("compact metadata lost")
	}
	projection, err := HashProjection(&compact)
	if err != nil || !bytes.Contains(projection, []byte(`"forbidden":["email","secret"]`)) {
		t.Fatalf("published hash metadata lost: %s %v", projection, err)
	}
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(base); err != nil {
		t.Fatal(err)
	}
	var decoded Iface
	if err := gob.NewDecoder(&buf).Decode(&decoded); err != nil {
		t.Fatal(err)
	}
	if err := decoded.Exports["clean"].IFC.Validate(); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(decoded.Exports["f"].IFC, s) {
		t.Fatal("gob metadata lost")
	}
}

func TestIFCCompactRejectsMissingMetadata(t *testing.T) {
	for _, raw := range []string{
		`{"module":"lib","schema":"ailang.iface/v2","funcs":[{"name":"sink"}]}`,
		`{"module":"lib","schema":"ailang.iface/v2","funcs":[{"name":"sink","ifc":null}]}`,
	} {
		var decoded InterfaceJSON
		if err := json.Unmarshal([]byte(raw), &decoded); err == nil {
			t.Fatal("missing published summary accepted")
		}
	}
	i := NewIface("lib")
	i.AddExport("sink", &types.Scheme{Type: types.TString}, true)
	i.Exports["sink"].IFC = nil
	if _, err := i.ToNormalizedJSON(); err == nil {
		t.Fatal("missing summary published")
	}
}
