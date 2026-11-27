package iface

import (
	"reflect"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/types"
)

// Unit tests for the facade re-export table (M-PKG-PARALLEL-AGENTS C1).

func intT() types.Type { return &types.TCon{Name: "int"} }

func intItem(name string) *IfaceItem {
	sc := &types.Scheme{Type: intT()}
	return &IfaceItem{Name: name, Type: sc, IFC: types.IFCSummaryFromScheme(sc)}
}

// facadeIface: lib/facade declares `own` and type Local, and re-exports the
// function `area`, the type Shape (with constructors Square, Rect) and the
// alias Size from lib/shapes.
func facadeIface() *Iface {
	f := NewIface("lib/facade")
	f.Exports["own"] = intItem("own")
	f.Types["Local"] = &TypeExport{Name: "Local"}
	f.Constructors["MkLocal"] = &ConstructorScheme{TypeName: "Local", CtorName: "MkLocal"}
	rx := NewReexports("lib/facade")
	rx.Items.Exports["area"] = intItem("area")
	rx.Items.Exports["own"] = intItem("shadowed")
	rx.Items.Types["Shape"] = &TypeExport{Name: "Shape"}
	rx.Items.Constructors["Square"] = &ConstructorScheme{TypeName: "Shape", CtorName: "Square", Arity: 1}
	rx.Items.Constructors["Rect"] = &ConstructorScheme{TypeName: "Shape", CtorName: "Rect", Arity: 2}
	rx.Items.Types["Size"] = &TypeExport{Name: "Size"}
	rx.Items.TypeAliases["Size"] = intT()
	rx.Items.AliasParams["Size"] = []string{"a"}
	rx.Origins["area"] = ReexportOrigin{Module: "lib/shapes", Name: "area", MovedTo: "new/shapes"}
	rx.Origins["Shape"] = ReexportOrigin{Module: "lib/shapes", Name: "Shape"}
	f.Reexported = rx
	return f
}

func TestLookup_OwnAndReexported(t *testing.T) {
	f := facadeIface()
	fn, typ, ctor, ctors := f.Lookup("area")
	if fn == nil || typ != nil || ctor != nil || ctors != nil {
		t.Fatalf("area: fn=%v typ=%v ctor=%v ctors=%v", fn, typ, ctor, ctors)
	}
	if fn, _, _, _ := f.Lookup("own"); fn == nil || fn.Name != "own" {
		t.Fatalf("own declaration must win over a re-export of the same name, got %+v", fn)
	}
	_, typ, _, ctors = f.Lookup("Shape")
	if typ == nil || len(ctors) != 2 || ctors["Square"] == nil || ctors["Rect"] == nil {
		t.Fatalf("Shape: typ=%v ctors=%v", typ, ctors)
	}
	_, typ, _, ctors = f.Lookup("Local")
	if typ == nil || len(ctors) != 1 || ctors["MkLocal"] == nil {
		t.Fatalf("own type Local: typ=%v ctors=%v", typ, ctors)
	}
	if _, _, ctor, _ := f.Lookup("Rect"); ctor == nil || ctor.Arity != 2 {
		t.Fatalf("Rect constructor: %+v", ctor)
	}
	if fn, typ, ctor, ctors := f.Lookup("missing"); fn != nil || typ != nil || ctor != nil || ctors != nil {
		t.Fatal("missing name found")
	}
	// A plain interface (no re-exports) looks up its own entries only.
	plain := NewIface("lib/plain")
	plain.Exports["f"] = &IfaceItem{Name: "f"}
	if fn, _, _, _ := plain.Lookup("f"); fn == nil {
		t.Fatal("plain lookup")
	}
}

func TestLookupAlias(t *testing.T) {
	f := facadeIface()
	body, params, ok := f.LookupAlias("Size")
	if !ok || body == nil || !reflect.DeepEqual(params, []string{"a"}) {
		t.Fatalf("Size: %v %v %v", body, params, ok)
	}
	f.TypeAliases["Own"] = intT()
	if _, params, ok := f.LookupAlias("Own"); !ok || params != nil {
		t.Fatalf("own alias: %v %v", params, ok)
	}
	if _, _, ok := f.LookupAlias("Shape"); ok {
		t.Fatal("Shape is not an alias")
	}
	if _, _, ok := NewIface("x").LookupAlias("Size"); ok {
		t.Fatal("alias found on an empty interface")
	}
}

func TestWithReexports_MergesWithoutMutating(t *testing.T) {
	f := facadeIface()
	merged := f.withReexports()
	for _, name := range []string{"own", "area"} {
		if merged.Exports[name] == nil {
			t.Errorf("merged exports miss %s", name)
		}
	}
	if merged.Exports["own"].Name != "own" {
		t.Errorf("declaration must win: %+v", merged.Exports["own"])
	}
	if merged.Types["Shape"] == nil || merged.Constructors["Square"] == nil || merged.TypeAliases["Size"] == nil || merged.AliasParams["Size"] == nil {
		t.Errorf("merged interface misses re-exported type entries: %+v", merged)
	}
	if f.Exports["area"] != nil || f.Constructors["Square"] != nil {
		t.Error("withReexports modified the facade's own maps")
	}
	plain := NewIface("lib/plain")
	if plain.withReexports() != plain {
		t.Error("an interface without re-exports must render as itself")
	}
}

func TestReexportDigestLines(t *testing.T) {
	if lines := NewIface("lib/plain").reexportDigestLines(); lines != nil {
		t.Fatalf("plain: %v", lines)
	}
	empty := NewIface("lib/f")
	empty.Reexported = NewReexports("lib/f")
	if lines := empty.reexportDigestLines(); lines != nil {
		t.Fatalf("empty table: %v", lines)
	}
	want := []string{"Shape=lib/shapes.Shape@", "area=lib/shapes.area@new/shapes"}
	if got := facadeIface().reexportDigestLines(); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestToNormalizedJSON_RendersReexports(t *testing.T) {
	raw, err := facadeIface().ToNormalizedJSON()
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{`"area"`, `"Shape"`, `"Square"`, `"own"`} {
		if !strings.Contains(string(raw), name) {
			t.Errorf("rendered interface misses %s:\n%s", name, raw)
		}
	}
}
