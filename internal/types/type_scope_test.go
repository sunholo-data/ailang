package types

import (
	"strings"
	"testing"
)

func TestMissingBuiltinHomes_EveryShape(t *testing.T) {
	bad := &TCon{Name: "Unhomed"}
	shapes := []Type{
		bad,
		&TApp{Constructor: BuiltinCon("Result"), Args: []Type{bad, TInt}},
		&TFunc2{Params: []Type{bad}, Return: TInt},
		&TFunc2{Params: []Type{TInt}, Return: bad},
		&TList{Element: bad},
		&TArray{Element: bad},
		&TMap{Key: TString, Value: bad},
		&TTuple{Elements: []Type{TInt, bad}},
		&TRecord{Fields: map[string]Type{"x": bad}},
		&TRecordOpen{Fields: map[string]Type{"x": bad}},
		&TLabelled{Inner: bad, L: LabelBottom()},
	}
	for _, s := range shapes {
		if got := MissingBuiltinHomes(s); len(got) != 1 || got[0] != "Unhomed" {
			t.Errorf("MissingBuiltinHomes(%T) = %v, want [Unhomed]", s, got)
		}
	}
	ok := &TFunc2{Params: []Type{BuiltinCon("Json"), BuiltinCon("Map"), TInt}, Return: BuiltinCon("Option")}
	if got := MissingBuiltinHomes(ok); len(got) != 0 {
		t.Errorf("homed signature reported %v", got)
	}
	if BuiltinCon("NotAHomedName").Module != "" {
		t.Error("an unhomed builtin name must carry no module (no $builtin fallback)")
	}
}

func TestAmbiguousTypeError_Text(t *testing.T) {
	e := &AmbiguousTypeError{Name: "Leg", Candidates: []TypeRef{{"pkg/x/a-b", "Leg"}, {"pkg/y/9z", "Leg"}}, Transitive: true}
	msg := e.Error()
	for _, want := range []string{TC_AMBIGUOUS_TYPE_001, "pkg/x/a-b, pkg/y/9z", "reaches through its imports", "import pkg/x/a-b (Leg)", "Leg as OtherLeg"} {
		if !strings.Contains(msg, want) {
			t.Errorf("missing %q in %s", want, msg)
		}
	}
	e = &AmbiguousTypeError{Name: "Leg", Candidates: []TypeRef{{"q/a", "Leg"}, {"q/b", "Leg"}}}
	if msg := e.Error(); !strings.Contains(msg, "the modules this one imports export") || !strings.Contains(msg, "Leg as BLeg") {
		t.Errorf("direct-dependency text: %s", msg)
	}
}

func TestAmbiguityInstrument_ObservesAndCanDowngrade(t *testing.T) {
	var events []AmbiguityEvent
	restore := SetAmbiguityInstrument(func(ev AmbiguityEvent) { events = append(events, ev) }, false)
	s := NewTypeScope("m")
	s.AddClosure(TypeRef{"q/b", "T"})
	s.AddClosure(TypeRef{"q/a", "T"})
	got, err := s.Lookup("T")
	if err != nil || got.Module != "q/a" {
		t.Fatalf("with errors off the first candidate in module order is used: %+v %v", got, err)
	}
	if AmbiguityIsError() {
		t.Fatal("instrument should have turned errors off")
	}
	restore()
	if !AmbiguityIsError() {
		t.Fatal("restore must turn errors back on")
	}
	if len(events) != 1 || events[0].Kind != "type-closure" || events[0].Name != "T" || len(events[0].Candidates) != 2 {
		t.Fatalf("events = %+v", events)
	}
	ObserveAmbiguity(AmbiguityEvent{Kind: "explicit-dup"}) // no observer installed: a no-op
}

func TestNominalMismatchMsg_BuiltinSide(t *testing.T) {
	msg := nominalMismatchMsg(&TCon{Name: "Map"}, &TCon{Name: "Map", Module: "x/m"})
	if !strings.Contains(msg, "the built-in Map") || !strings.Contains(msg, "Map (defined in x/m)") || strings.Contains(msg, "<no module>") {
		t.Errorf("built-in side: %s", msg)
	}
	msg = nominalMismatchMsg(&TCon{Name: "DOMEvent", Module: BuiltinModule}, &TCon{Name: "DOMEvent"})
	if !strings.Contains(msg, "use one DOMEvent on both sides") {
		t.Errorf("no module to import from: %s", msg)
	}
	if displayModule("") != "the built-in" || displayModule("q/a") != "q/a" {
		t.Error("displayModule")
	}
}

func TestAmbiguousCtorError_Text(t *testing.T) {
	e := NewAmbiguousCtorError("C", []TypeRef{{"q/a", "T"}, {"q/b", "T"}}, "at q/m:4:2")
	msg := e.Error()
	for _, want := range []string{TC_AMBIGUOUS_CTOR_001, "at q/m:4:2", "q/a's T and q/b's T", "import q/a (C)"} {
		if !strings.Contains(msg, want) {
			t.Errorf("missing %q in %s", want, msg)
		}
	}
}
