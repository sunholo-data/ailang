package bytecode

import "testing"

func TestMapValueIdentityAndPersistence(t *testing.T) {
	m := NewMap(nil)
	a, err := m.AsMap().Insert(NewInt(1), NewString("one"))
	if err != nil {
		t.Fatal(err)
	}
	b, err := a.AsMap().Insert(NewString("1"), NewString("string"))
	if err != nil {
		t.Fatal(err)
	}
	if len(m.AsMap().Entries) != 0 || len(a.AsMap().Entries) != 1 || len(b.AsMap().Entries) != 2 {
		t.Fatal("mutated original")
	}
	if b.String() != `Map{1: "one", "1": "string"}` {
		t.Fatal(b.String())
	}
	if !b.Equal(b) || b.Equal(a) {
		t.Fatal("equality")
	}
	if _, err := a.AsMap().Insert(NewFloat(1), Unit()); err == nil {
		t.Fatal("float key accepted")
	}
}
