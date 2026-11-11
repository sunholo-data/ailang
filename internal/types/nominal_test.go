package types

import (
	"encoding/json"
	"strings"
	"testing"
)

// M-V1-NOMINAL-TYPE-IDENTITY: identity is (Module, Name); "" is not a wildcard.
func TestNominalSame_ModuleIsIdentity(t *testing.T) {
	a := &TCon{Name: "T", Module: "q/a"}
	b := &TCon{Name: "T", Module: "q/b"}
	a2 := &TCon{Name: "T", Module: "q/a"}
	bare := &TCon{Name: "T"}
	if !a.Equals(a2) || !SafeEquals(a, a2) {
		t.Fatal("same module and name must be equal")
	}
	for _, pair := range [][2]*TCon{{a, b}, {a, bare}, {bare, b}} {
		if pair[0].Equals(pair[1]) || SafeEquals(pair[0], pair[1]) {
			t.Fatalf("%s(%s) must differ from %s(%s)", pair[0].Name, pair[0].Module, pair[1].Name, pair[1].Module)
		}
	}
	if a.String() != "T" {
		t.Fatalf("display must stay the bare name, got %q", a.String())
	}
}

func TestUnify_NominalMismatchIsCodedAndNamesBothModules(t *testing.T) {
	u := NewUnifier()
	_, err := u.Unify(&TCon{Name: "T", Module: "q/a"}, &TCon{Name: "T", Module: "q/b"}, Substitution{})
	if err == nil {
		t.Fatal("two modules' T unified")
	}
	for _, want := range []string{TC_NOMINAL_001, "T (defined in q/a)", "T (defined in q/b)", "Suggestion:"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("missing %q in %v", want, err)
		}
	}
	// Type-application heads go through the same comparison.
	_, err = u.Unify(&TApp{Constructor: &TCon{Name: "Box", Module: "a"}, Args: []Type{TInt}},
		&TApp{Constructor: &TCon{Name: "Box", Module: "b"}, Args: []Type{TInt}}, Substitution{})
	if err == nil || !strings.Contains(err.Error(), TC_NOMINAL_001) {
		t.Fatalf("applied heads of two modules must not unify: %v", err)
	}
	// Unrelated names keep the ordinary message.
	_, err = u.Unify(&TCon{Name: "T", Module: "q/a"}, &TCon{Name: "U", Module: "q/a"}, Substitution{})
	if err == nil || strings.Contains(err.Error(), TC_NOMINAL_001) {
		t.Fatalf("different names are not a nominal-identity error: %v", err)
	}
}

func TestUnify_HalfQualifiedPairIsInternalError(t *testing.T) {
	prev := SetUnqualifiedNominalPanic(false)
	defer SetUnqualifiedNominalPanic(prev)
	_, err := NewUnifier().Unify(&TCon{Name: "T", Module: "q/a"}, &TCon{Name: "T"}, Substitution{})
	if err == nil || !strings.Contains(err.Error(), "internal compiler error") {
		t.Fatalf("a minting site that forgot the module must be loud, got %v", err)
	}
}

func TestUnqualifiedNominalHook_Panics(t *testing.T) {
	prev := SetUnqualifiedNominalPanic(true)
	defer SetUnqualifiedNominalPanic(prev)
	defer func() {
		if r := recover(); r == nil || !strings.Contains(r.(string), "Widget") {
			t.Fatalf("hook did not fire: %v", r)
		}
	}()
	_, _ = NewUnifier().Unify(&TCon{Name: "Widget"}, &TCon{Name: "Widget"}, Substitution{})
}

func TestUnqualifiedNominalHook_IgnoresModuleless(t *testing.T) {
	prev := SetUnqualifiedNominalPanic(true)
	defer SetUnqualifiedNominalPanic(prev)
	if _, err := NewUnifier().Unify(TInt, &TCon{Name: "int"}, Substitution{}); err != nil {
		t.Fatal(err)
	}
}

func TestTConJSON_RoundTripsModule(t *testing.T) {
	for _, in := range []*TCon{{Name: "T", Module: "pkg/x/y"}, {Name: "int"}} {
		raw, err := json.Marshal(in)
		if err != nil {
			t.Fatal(err)
		}
		out, err := UnmarshalType(raw)
		if err != nil {
			t.Fatal(err)
		}
		if c := out.(*TCon); c.Name != in.Name || c.Module != in.Module {
			t.Fatalf("round trip %+v -> %+v", in, c)
		}
		if in.Module == "" && strings.Contains(string(raw), "Module") {
			t.Fatalf("module-less TCon must encode as before: %s", raw)
		}
	}
}

func TestTypeScope_Precedence(t *testing.T) {
	s := NewTypeScope("m")
	s.Bind("Option", TypeRef{Module: "m", Name: "Option"}) // local wins
	s.Bind("Option", TypeRef{Module: "std/option", Name: "Option"})
	s.Bind("U", TypeRef{Module: "q/a", Name: "T"}) // import q/a (T as U)
	cases := map[string]TypeRef{
		"Option":   {Module: "m", Name: "Option"},
		"U":        {Module: "q/a", Name: "T"},
		"Json":     {Module: "std/json", Name: "Json"},
		"DOMEvent": {Module: BuiltinModule, Name: "DOMEvent"},
		"Nope":     {Module: "m", Name: "Nope"},
		"int":      {Name: "int"},
		"Map":      {Name: "Map"},
	}
	for name, want := range cases {
		if got := s.Resolve(name); got != want {
			t.Errorf("Resolve(%s) = %+v, want %+v", name, got, want)
		}
	}
}

func TestNominalKey_RoundTrip(t *testing.T) {
	m, n := SplitNominalKey(NominalKey("pkg/a/b", "T"))
	if m != "pkg/a/b" || n != "T" {
		t.Fatalf("got %q %q", m, n)
	}
	if NominalKey("", "int") != "int" {
		t.Fatal("module-less key must be the bare name")
	}
}
