package iface

import (
	"encoding/json"
	"reflect"
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
		"example/shapes:ctor:Shape:Rectangle(float, float)",
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

	if _, err := HashProjection(j); err != nil {
		t.Fatalf("HashProjection: %v", err)
	}
	if !reflect.DeepEqual(j, want) {
		before, _ := json.Marshal(want)
		after, _ := json.Marshal(j)
		t.Fatalf("HashProjection mutated input:\nbefore %s\n after %s", before, after)
	}
}
