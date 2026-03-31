package types

import (
	"testing"
)

// TestTMap_String tests the String() representation
func TestTMap_String(t *testing.T) {
	tm := &TMap{Key: TString, Value: TInt}
	if s := tm.String(); s != "Map[string, int]" {
		t.Errorf("TMap.String() = %q, want %q", s, "Map[string, int]")
	}

	// Nested: Map[string, Map[int, bool]]
	inner := &TMap{Key: TInt, Value: TBool}
	outer := &TMap{Key: TString, Value: inner}
	if s := outer.String(); s != "Map[string, Map[int, bool]]" {
		t.Errorf("nested TMap.String() = %q, want %q", s, "Map[string, Map[int, bool]]")
	}
}

// TestTMap_Equals tests structural equality
func TestTMap_Equals(t *testing.T) {
	a := &TMap{Key: TString, Value: TInt}
	b := &TMap{Key: TString, Value: TInt}
	c := &TMap{Key: TInt, Value: TString}
	d := &TMap{Key: TString, Value: TBool}

	if !a.Equals(b) {
		t.Error("identical TMap should be equal")
	}
	if a.Equals(c) {
		t.Error("Map[string,int] should not equal Map[int,string]")
	}
	if a.Equals(d) {
		t.Error("Map[string,int] should not equal Map[string,bool]")
	}
	if a.Equals(TInt) {
		t.Error("TMap should not equal TInt")
	}
}

// TestTMap_Substitute tests type variable substitution
func TestTMap_Substitute(t *testing.T) {
	k := &TVar2{Name: "k", Kind: Star}
	v := &TVar2{Name: "v", Kind: Star}
	tm := &TMap{Key: k, Value: v}

	subs := map[string]Type{
		"k": TString,
		"v": TInt,
	}
	result := tm.Substitute(subs)
	expected := &TMap{Key: TString, Value: TInt}

	if !result.Equals(expected) {
		t.Errorf("after substitution got %s, want %s", result, expected)
	}

	// Partial substitution: only k
	partial := map[string]Type{"k": TBool}
	partialResult := tm.Substitute(partial)
	if partialMap, ok := partialResult.(*TMap); !ok {
		t.Errorf("partial substitution returned %T, want *TMap", partialResult)
	} else {
		if !partialMap.Key.Equals(TBool) {
			t.Errorf("partial sub key = %s, want bool", partialMap.Key)
		}
		if !partialMap.Value.Equals(v) {
			t.Errorf("partial sub value = %s, want %s (unchanged)", partialMap.Value, v)
		}
	}
}

// TestTMap_Head tests TypeHead classification
func TestTMap_Head(t *testing.T) {
	tm := &TMap{Key: TString, Value: TInt}
	h := Head(tm)
	if h != HeadMap {
		t.Errorf("Head(TMap) = %v, want HeadMap", h)
	}
}

// TestTMap_Builder tests the Builder.Map() helper
func TestTMap_Builder(t *testing.T) {
	b := NewBuilder()
	result := b.Map(b.String(), b.Int())

	tm, ok := result.(*TMap)
	if !ok {
		t.Fatalf("Builder.Map() returned %T, want *TMap", result)
	}
	if !tm.Key.Equals(TString) {
		t.Errorf("key = %s, want string", tm.Key)
	}
	if !tm.Value.Equals(TInt) {
		t.Errorf("value = %s, want int", tm.Value)
	}
}

// TestTMap_Unify tests TMap unification
func TestTMap_Unify(t *testing.T) {
	u := NewUnifier()

	// TMap[string, int] unifies with TMap[string, int]
	a := &TMap{Key: TString, Value: TInt}
	b := &TMap{Key: TString, Value: TInt}
	sub, err := u.Unify(a, b, Substitution{})
	if err != nil {
		t.Fatalf("unify identical TMap: %v", err)
	}
	_ = sub

	// TMap[k, v] unifies with TMap[string, int] binding k=string, v=int
	k := &TVar2{Name: "k", Kind: Star}
	v := &TVar2{Name: "v", Kind: Star}
	generic := &TMap{Key: k, Value: v}
	concrete := &TMap{Key: TString, Value: TInt}
	sub2, err := u.Unify(generic, concrete, Substitution{})
	if err != nil {
		t.Fatalf("unify generic TMap: %v", err)
	}
	resolvedK := ApplySubstitution(sub2, k)
	resolvedV := ApplySubstitution(sub2, v)
	if !resolvedK.Equals(TString) {
		t.Errorf("k resolved to %s, want string", resolvedK)
	}
	if !resolvedV.Equals(TInt) {
		t.Errorf("v resolved to %s, want int", resolvedV)
	}
}

// TestTMap_UnifyWithTApp tests bidirectional TMap ↔ TApp("Map",...) unification
func TestTMap_UnifyWithTApp(t *testing.T) {
	u := NewUnifier()

	tmap := &TMap{Key: TString, Value: TInt}
	tapp := &TApp{
		Constructor: &TCon{Name: "Map"},
		Args:        []Type{TString, TInt},
	}

	// TMap -> TApp direction
	sub, err := u.Unify(tmap, tapp, Substitution{})
	if err != nil {
		t.Fatalf("unify TMap with TApp: %v", err)
	}
	_ = sub

	// TApp -> TMap direction
	sub2, err := u.Unify(tapp, tmap, Substitution{})
	if err != nil {
		t.Fatalf("unify TApp with TMap: %v", err)
	}
	_ = sub2
}

// TestTMap_StringDeterministic runs with -count=20 to verify String() is deterministic.
func TestTMap_StringDeterministic(t *testing.T) {
	tm := &TMap{Key: TString, Value: &TMap{Key: TInt, Value: TBool}}
	expected := "Map[string, Map[int, bool]]"
	for i := 0; i < 10; i++ {
		got := tm.String()
		if got != expected {
			t.Errorf("iteration %d: String() = %q, want %q", i, got, expected)
		}
	}
}
