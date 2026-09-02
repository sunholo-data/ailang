package iface

import (
	"encoding/json"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func TestHashProjection_ExcludesAlias(t *testing.T) {
	j := &InterfaceJSON{
		Module: "example/cards",
		Schema: "ailang.iface/v1",
		Types: []TypeJSON{{
			Name: "Card", Params: []string{"a"}, Alias: "{value: a}",
		}},
		Funcs: []FuncJSON{{Name: "make", Type: "a -> Card[a]", Effects: []string{}, Pure: true}},
	}

	// Anti-vacuity floor: the negative assertion below passes trivially if the fixture
	// carries no alias, so an empty fixture is an INSTRUMENT FAILURE, not a pass.
	if j.Types[0].Alias == "" {
		t.Fatal("instrument failure: fixture must carry a non-empty Alias")
	}

	got, err := HashProjection(j)
	if err != nil {
		t.Fatalf("HashProjection: %v", err)
	}
	if strings.Contains(string(got), "alias") || strings.Contains(string(got), "{value: a}") {
		t.Fatalf("projection retained alias: %s", got)
	}
	for _, want := range []string{`"module":"example/cards"`, `"schema":"ailang.iface/v1"`, `"name":"Card"`, `"params":["a"]`, `"name":"make"`, `"type":"a -\u003e Card[a]"`, `"pure":true`} {
		if !strings.Contains(string(got), want) {
			t.Errorf("projection %s does not contain %s", got, want)
		}
	}
	if _, err := HashProjection(nil); err == nil {
		t.Error("HashProjection(nil) returned nil error")
	}
}

func TestHashProjection_Deterministic(t *testing.T) {
	want := `{"module":"example/shapes","types":[{"name":"Color","ctors":["Blue","Red"]},{"name":"Shape","params":["a"],"ctors":["Circle(float)","Rectangle(float, float)"]}],"funcs":[{"name":"area","type":"Shape[a] -\u003e float","effects":["IO","State"],"pure":false},{"name":"paint","type":"(Shape[a], Color) -\u003e Shape[a]","effects":[],"pure":true}],"schema":"ailang.iface/v1"}`

	for i := 0; i < 128; i++ {
		typesByName := map[string]TypeJSON{
			"Shape": {Name: "Shape", Params: []string{"a"}, Ctors: []string{"Rectangle(float, float)", "Circle(float)"}},
			"Color": {Name: "Color", Ctors: []string{"Red", "Blue"}},
		}
		funcsByName := map[string]FuncJSON{
			"paint": {Name: "paint", Type: "(Shape[a], Color) -> Shape[a]", Effects: []string{}, Pure: true},
			"area":  {Name: "area", Type: "Shape[a] -> float", Effects: []string{"State", "IO"}, Pure: false},
		}
		j := &InterfaceJSON{Module: "example/shapes", Schema: "ailang.iface/v1"}
		for _, typ := range typesByName {
			j.Types = append(j.Types, typ)
		}
		for _, fn := range funcsByName {
			j.Funcs = append(j.Funcs, fn)
		}

		got, err := HashProjection(j)
		if err != nil {
			t.Fatalf("iteration %d: HashProjection: %v", i, err)
		}
		if string(got) != want {
			t.Fatalf("iteration %d:\n got %s\nwant %s", i, got, want)
		}
	}
}

func TestSignatureSet_SortedAndStable(t *testing.T) {
	j := &InterfaceJSON{
		Module: "example/shapes",
		Types: []TypeJSON{
			{Name: "Shape", Params: []string{"a"}, Ctors: []string{"Rectangle(float, float)", "Circle(float)"}},
			{Name: "AliasOnly", Alias: "{hidden: string}"},
		},
		Funcs: []FuncJSON{
			{Name: "paint", Type: "Shape[a] -> Shape[a]", Effects: []string{"State", "IO"}, Pure: false},
			{Name: "area", Type: "Shape[a] -> float", Effects: []string{}, Pure: true},
		},
	}
	want := []string{
		"example/shapes:ctor:Shape:Circle(float)",
		`example/shapes:ctor:Shape:Rectangle(float\, float)`, // comma escaped — see escapeSigField
		"example/shapes:func:area:Shape[a] -> float:",
		"example/shapes:func:paint:Shape[a] -> Shape[a]:IO,State",
		"example/shapes:type:AliasOnly/0",
		"example/shapes:type:Shape/1",
	}

	if got := SignatureSet(j); !reflect.DeepEqual(got, want) {
		t.Fatalf("SignatureSet() = %#v, want %#v", got, want)
	}
	if got := SignatureSet(&InterfaceJSON{}); got == nil || len(got) != 0 {
		t.Fatalf("SignatureSet(empty) = %#v, want non-nil empty slice", got)
	}
	if got := SignatureSet(nil); got != nil {
		t.Fatalf("SignatureSet(nil) = %#v, want nil", got)
	}
}

func TestHashProjection_DoesNotMutateInput(t *testing.T) {
	j := &InterfaceJSON{
		Module: "example/shapes",
		Schema: "ailang.iface/v1",
		Types:  []TypeJSON{{Name: "Shape", Params: []string{"z", "a"}, Ctors: []string{"Square", "Circle"}, Alias: "kept by caller"}},
		Funcs:  []FuncJSON{{Name: "draw", Type: "Shape -> unit", Effects: []string{"State", "IO"}, Pure: false}},
	}
	want := &InterfaceJSON{
		Module: "example/shapes",
		Schema: "ailang.iface/v1",
		Types:  []TypeJSON{{Name: "Shape", Params: []string{"z", "a"}, Ctors: []string{"Square", "Circle"}, Alias: "kept by caller"}},
		Funcs:  []FuncJSON{{Name: "draw", Type: "Shape -> unit", Effects: []string{"State", "IO"}, Pure: false}},
	}

	// Anti-vacuity floor (iteration 311 evaluator): this arm only detects a Ctors-aliasing
	// in-place sort while the fixture's Ctors are OUT of order. Re-running the same real bug
	// against an already-sorted fixture made it invisible — so a well-meaning "tidy the
	// fixture alphabetically" edit would silently disable the check with no signal.
	if sort.StringsAreSorted(j.Types[0].Ctors) {
		t.Fatal("instrument failure: fixture Ctors must be deliberately UNSORTED")
	}

	if _, err := HashProjection(j); err != nil {
		t.Fatalf("HashProjection: %v", err)
	}
	if !reflect.DeepEqual(j, want) {
		before, _ := json.Marshal(want)
		after, _ := json.Marshal(j)
		t.Fatalf("HashProjection mutated input:\nbefore %s\n after %s", before, after)
	}
}

// TestSignatureSet_EncodingIsInjective pins the fix for the collision the iteration-311
// sprint evaluator found and the controller reproduced: before escapeSigField, the pair
// below produced the byte-identical signature "mod:func:run:A:B:". M5 diffs these sets to
// classify a release, so a collision is a wrong ChangeClass, not a cosmetic defect.
func TestSignatureSet_EncodingIsInjective(t *testing.T) {
	cases := []struct {
		name string
		a, b *InterfaceJSON
	}{
		{
			"type-colon vs effect-colon",
			&InterfaceJSON{Module: "mod", Funcs: []FuncJSON{{Name: "run", Type: "A:B", Effects: []string{}}}},
			&InterfaceJSON{Module: "mod", Funcs: []FuncJSON{{Name: "run", Type: "A", Effects: []string{"B:"}}}},
		},
		{
			"one effect with a comma vs two effects",
			&InterfaceJSON{Module: "mod", Funcs: []FuncJSON{{Name: "run", Type: "T", Effects: []string{"IO,FS"}}}},
			&InterfaceJSON{Module: "mod", Funcs: []FuncJSON{{Name: "run", Type: "T", Effects: []string{"IO", "FS"}}}},
		},
		{
			"colon in a ctor vs in the type name",
			&InterfaceJSON{Module: "mod", Types: []TypeJSON{{Name: "T", Ctors: []string{"C:D"}}}},
			&InterfaceJSON{Module: "mod", Types: []TypeJSON{{Name: "T:C", Ctors: []string{"D"}}}},
		},
	}
	for _, c := range cases {
		sa, sb := SignatureSet(c.a), SignatureSet(c.b)
		if reflect.DeepEqual(sa, sb) {
			t.Errorf("%s: distinct interfaces share a signature set %#v", c.name, sa)
		}
	}

	// Control: the escaping must not break the ordinary case, and it must be stable.
	plain := &InterfaceJSON{Module: "mod", Funcs: []FuncJSON{{Name: "run", Type: "int -> int", Effects: []string{"IO"}}}}
	want := []string{"mod:func:run:int -> int:IO"}
	if got := SignatureSet(plain); !reflect.DeepEqual(got, want) {
		t.Errorf("unescaped input changed shape: got %#v want %#v", got, want)
	}
	if !reflect.DeepEqual(SignatureSet(plain), SignatureSet(plain)) {
		t.Error("SignatureSet is not stable across calls")
	}
}
