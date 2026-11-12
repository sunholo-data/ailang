package core

import "testing"

func TestADTFactoryName_RoundTrip(t *testing.T) {
	cases := []struct {
		module, typ, ctor, name string
	}{
		{"q/a", "T", "A", "make_T_A@q/a"},
		{"pkg/acme/shapes/types/shape", "Shape", "Circle", "make_Shape_Circle@pkg/acme/shapes/types/shape"},
		{"std/result", "Season", "LOW_SEASON", "make_Season_LOW_SEASON@std/result"},
		{"", "Option", "Some", "make_Option_Some"},
	}
	for _, c := range cases {
		name := ADTFactoryName(c.module, c.typ, c.ctor)
		if name != c.name {
			t.Errorf("ADTFactoryName(%q,%q,%q) = %q, want %q", c.module, c.typ, c.ctor, name, c.name)
		}
		if key := ADTFactoryKey(c.module, c.typ, c.ctor); key != "$adt."+c.name {
			t.Errorf("ADTFactoryKey = %q", key)
		}
		m, ty, ct, ok := ParseADTFactoryName(name)
		if !ok || m != c.module || ty != c.typ || ct != c.ctor {
			t.Errorf("ParseADTFactoryName(%q) = %q %q %q %v", name, m, ty, ct, ok)
		}
		base, mod := SplitADTFactoryModule(name)
		if mod != c.module || ADTFactoryDisplayName(name) != base || base != "make_"+c.typ+"_"+c.ctor {
			t.Errorf("SplitADTFactoryModule(%q) = %q %q", name, base, mod)
		}
		if !IsADTFactory(GlobalRef{Module: ADTModule, Name: name}) {
			t.Errorf("IsADTFactory(%q) = false", name)
		}
	}
	for _, bad := range []string{"", "make_", "make_T", "make__A", "makeT_A", "make_T_@q/a"} {
		if _, _, _, ok := ParseADTFactoryName(bad); ok {
			t.Errorf("ParseADTFactoryName(%q) accepted a malformed name", bad)
		}
	}
	if IsADTFactory(GlobalRef{Module: "$builtin", Name: "make_T_A"}) {
		t.Error("a non-$adt ref is not a factory")
	}
}
