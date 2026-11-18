package iface

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/types"
)

// eqScheme is ∀a. Eq[a], Num[a], Eq[a] => (a, a) -> bool (a duplicate on purpose).
func eqScheme() *types.Scheme {
	a := &types.TVar2{Name: "t9", Kind: types.KStar{}}
	return &types.Scheme{
		TypeVars: []string{"t9"},
		Constraints: []types.Constraint{
			{Class: "Num", Type: a}, {Class: "Eq", Type: a}, {Class: "Eq", Type: a},
		},
		Type: &types.TFunc2{Params: []types.Type{a, a}, Return: types.TBool},
	}
}

// #2290: the export keeps its constraints (canonicalizeScheme), the digest
// moves with them (schemeToString), and the JSON lists them sorted,
// de-duplicated and named like the type (canonicalizeSchemeParts).
func TestExportKeepsConstraints(t *testing.T) {
	b := NewBuilder("q/lib", nil)
	got, err := b.canonicalizeScheme(eqScheme())
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Constraints) != 3 {
		t.Fatalf("canonicalizeScheme dropped constraints: %v", got.Constraints)
	}
	plain := &types.Scheme{TypeVars: got.TypeVars, Type: got.Type}
	if p, c := b.schemeToString(plain), b.schemeToString(got); p == c || !strings.Contains(c, "Eq[t9], Eq[t9], Num[t9] => ") {
		t.Errorf("digest string must carry the sorted constraints: %q vs %q", p, c)
	}
	if s := b.schemeToString(&types.Scheme{Type: types.TInt}); s != "int" {
		t.Errorf("an unquantified, unconstrained scheme renders as its type, got %q", s)
	}

	zero := &types.Scheme{Type: types.TInt}
	i := &Iface{Module: "q/lib", Schema: SchemaVersion, Exports: map[string]*IfaceItem{
		"eqq":  {Name: "eqq", Type: got, Purity: true, IFC: types.IFCSummaryFromScheme(got)},
		"zero": {Name: "zero", Type: zero, Purity: true, IFC: types.IFCSummaryFromScheme(zero)},
	}}
	data, err := i.ToNormalizedJSON()
	if err != nil {
		t.Fatal(err)
	}
	var j InterfaceJSON
	if err := json.Unmarshal(data, &j); err != nil {
		t.Fatal(err)
	}
	for _, f := range j.Funcs {
		switch f.Name {
		case "eqq":
			if f.Type != "(a,a)->bool" || strings.Join(f.Constraints, " ") != "Eq[a] Num[a]" {
				t.Errorf("eqq: got %s %v", f.Type, f.Constraints)
			}
		case "zero":
			if f.Constraints != nil {
				t.Errorf("an unconstrained export must omit constraints, got %v", f.Constraints)
			}
		}
	}
	if typ, cs, lits := canonicalizeSchemeParts(nil); typ != "unknown" || cs != nil || lits != nil {
		t.Errorf("nil scheme: got %q %v %v", typ, cs, lits)
	}
}
