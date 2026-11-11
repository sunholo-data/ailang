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

// TestTypeScope_LocalAndExplicitBeatEveryBuiltinName: a module may declare,
// or import by name, a type whose name is a built-in (Map, List, Int, ...), a
// prelude name or a builtin-home name (Json, DOMEvent, ...); its own or the
// imported one wins. The names come from the tables themselves, so a new
// built-in cannot be missed.
func TestTypeScope_LocalAndExplicitBeatEveryBuiltinName(t *testing.T) {
	var names []string
	names = append(names, ModulelessNames()...)
	names = append(names, PreludeTypeNames()...)
	names = append(names, BuiltinHomeNames()...)
	if len(names) < 30 {
		t.Fatalf("only %d built-in names: the tables are not being read", len(names))
	}
	for _, n := range names {
		local := NewTypeScope("m")
		local.BindLocal(n)
		if got, err := local.Lookup(n); err != nil || got != (TypeRef{Module: "m", Name: n}) {
			t.Errorf("local %s: got %+v %v, want m's", n, got, err)
		}
		imp := NewTypeScope("m")
		imp.BindExplicit(n, TypeRef{Module: "q/a", Name: n})
		if got, err := imp.Lookup(n); err != nil || got != (TypeRef{Module: "q/a", Name: n}) {
			t.Errorf("imported %s: got %+v %v, want q/a's", n, got, err)
		}
	}
}

// TestTypeScope_EachStepBeatsTheNext pins the precedence order step by step:
// each scope binds a name at two adjacent steps and must answer the earlier.
func TestTypeScope_EachStepBeatsTheNext(t *testing.T) {
	type step struct {
		name string
		bind func(s *TypeScope, n string)
		want func(n string) TypeRef
	}
	local := step{"local", func(s *TypeScope, n string) { s.BindLocal(n) }, func(n string) TypeRef { return TypeRef{"m", n} }}
	explicit := step{"explicit", func(s *TypeScope, n string) { s.BindExplicit(n, TypeRef{"q/exp", n}) }, func(n string) TypeRef { return TypeRef{"q/exp", n} }}
	direct := step{"direct", func(s *TypeScope, n string) { s.AddDirect(TypeRef{"q/dir", n}) }, func(n string) TypeRef { return TypeRef{"q/dir", n} }}
	closure := step{"closure", func(s *TypeScope, n string) { s.AddClosure(TypeRef{"q/clo", n}) }, func(n string) TypeRef { return TypeRef{"q/clo", n} }}
	pairs := []struct {
		name          string
		first, second step
	}{
		{"T", local, explicit},
		{"T", explicit, direct},
		{"Map", explicit, direct}, // explicit beats a built-in name too
		{"Option", local, direct},
	}
	for _, p := range pairs {
		s := NewTypeScope("m")
		p.second.bind(s, p.name)
		p.first.bind(s, p.name)
		if got, err := s.Lookup(p.name); err != nil || got != p.first.want(p.name) {
			t.Errorf("%s: %s must beat %s, got %+v %v", p.name, p.first.name, p.second.name, got, err)
		}
	}
	// Built-in (step 4) beats direct and closure; prelude (5) beats them too.
	for _, n := range []string{"Map", "List", "Int", "Option", "Result"} {
		s := NewTypeScope("m")
		direct.bind(s, n)
		closure.bind(s, n)
		got, err := s.Lookup(n)
		if err != nil || got.Module == "q/dir" || got.Module == "q/clo" {
			t.Errorf("%s: a built-in/prelude name must beat the auto-import channels, got %+v %v", n, got, err)
		}
	}
	// Direct and closure are ONE step: a candidate in each is ambiguous.
	s0 := NewTypeScope("m")
	direct.bind(s0, "T")
	closure.bind(s0, "T")
	if _, err := s0.Lookup("T"); err == nil {
		t.Error("a direct dependency's T and a closure module's T must be ambiguous, not decided by which is nearer")
	}
	// The auto-import step beats the builtin home table; the table beats "this module".
	s := NewTypeScope("m")
	closure.bind(s, "Json")
	if got, _ := s.Lookup("Json"); got != (TypeRef{"q/clo", "Json"}) {
		t.Errorf("an imported/reachable declaration must beat the builtin home table, got %+v", got)
	}
	if got, _ := NewTypeScope("m").Lookup("Json"); got != (TypeRef{"std/json", "Json"}) {
		t.Errorf("builtin home must beat this module, got %+v", got)
	}
	if got, _ := NewTypeScope("m").Lookup("Nope"); got != (TypeRef{"m", "Nope"}) {
		t.Errorf("an unbound name is this module's, got %+v", got)
	}
}

// TestTypeScope_AmbiguityIsAnErrorNotAPick: two candidates at the direct or
// closure step are TC_AMBIGUOUS_TYPE_001 naming both, in an order that does
// not depend on the order they were added.
func TestTypeScope_AmbiguityIsAnErrorNotAPick(t *testing.T) {
	for _, add := range []func(*TypeScope, TypeRef){(*TypeScope).AddDirect, (*TypeScope).AddClosure} {
		var msgs []string
		for _, order := range [][]string{{"q/a", "q/b"}, {"q/b", "q/a"}} {
			s := NewTypeScope("m")
			for _, mod := range order {
				add(s, TypeRef{mod, "Leg"})
				add(s, TypeRef{mod, "Leg"}) // the same declaration twice is one candidate
			}
			_, err := s.Lookup("Leg")
			if err == nil {
				t.Fatal("two candidates must be TC_AMBIGUOUS_TYPE_001")
			}
			for _, want := range []string{TC_AMBIGUOUS_TYPE_001, "q/a", "q/b", "Suggestion:"} {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("missing %q: %v", want, err)
				}
			}
			msgs = append(msgs, err.Error())
		}
		if msgs[0] != msgs[1] {
			t.Errorf("diagnostic depends on order:\n%s\n%s", msgs[0], msgs[1])
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
