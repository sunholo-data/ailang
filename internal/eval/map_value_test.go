package eval

import (
	"testing"
)

// TestMapKey_IntKeys tests canonical encoding of int keys
func TestMapKey_IntKeys(t *testing.T) {
	tests := []struct {
		value    int
		expected string
	}{
		{0, "i:0"},
		{42, "i:42"},
		{-1, "i:-1"},
		{1000000, "i:1000000"},
	}
	for _, tt := range tests {
		key, err := MapKey(&IntValue{Value: tt.value})
		if err != nil {
			t.Fatalf("MapKey(int %d) error: %v", tt.value, err)
		}
		if key != tt.expected {
			t.Errorf("MapKey(int %d) = %q, want %q", tt.value, key, tt.expected)
		}
	}
}

// TestMapKey_StringKeys tests canonical encoding of string keys
func TestMapKey_StringKeys(t *testing.T) {
	tests := []struct {
		value    string
		expected string
	}{
		{"", "s:"},
		{"hello", "s:hello"},
		{"key with spaces", "s:key with spaces"},
		{"i:42", "s:i:42"}, // string that looks like int encoding — must not collide
	}
	for _, tt := range tests {
		key, err := MapKey(&StringValue{Value: tt.value})
		if err != nil {
			t.Fatalf("MapKey(string %q) error: %v", tt.value, err)
		}
		if key != tt.expected {
			t.Errorf("MapKey(string %q) = %q, want %q", tt.value, key, tt.expected)
		}
	}
}

// TestMapKey_BoolKeys tests canonical encoding of bool keys
func TestMapKey_BoolKeys(t *testing.T) {
	keyTrue, err := MapKey(&BoolValue{Value: true})
	if err != nil {
		t.Fatalf("MapKey(true) error: %v", err)
	}
	if keyTrue != "b:true" {
		t.Errorf("MapKey(true) = %q, want %q", keyTrue, "b:true")
	}

	keyFalse, err := MapKey(&BoolValue{Value: false})
	if err != nil {
		t.Fatalf("MapKey(false) error: %v", err)
	}
	if keyFalse != "b:false" {
		t.Errorf("MapKey(false) = %q, want %q", keyFalse, "b:false")
	}

	if keyTrue == keyFalse {
		t.Error("MapKey(true) and MapKey(false) must differ")
	}
}

// TestMapKey_NoCollisions verifies different types with similar-looking values don't collide
func TestMapKey_NoCollisions(t *testing.T) {
	keys := make(map[string]string)

	vals := []struct {
		label string
		val   Value
	}{
		{"int 0", &IntValue{Value: 0}},
		{"string '0'", &StringValue{Value: "0"}},
		{"bool false", &BoolValue{Value: false}},
		{"int 1", &IntValue{Value: 1}},
		{"string '1'", &StringValue{Value: "1"}},
		{"bool true", &BoolValue{Value: true}},
		{"string 'true'", &StringValue{Value: "true"}},
		{"string 'false'", &StringValue{Value: "false"}},
		{"string 'i:42'", &StringValue{Value: "i:42"}},
		{"int 42", &IntValue{Value: 42}},
	}

	for _, v := range vals {
		k, err := MapKey(v.val)
		if err != nil {
			t.Fatalf("MapKey(%s) error: %v", v.label, err)
		}
		if existing, ok := keys[k]; ok {
			t.Errorf("COLLISION: MapKey(%s) == MapKey(%s) == %q", v.label, existing, k)
		}
		keys[k] = v.label
	}
}

// TestMapKey_UnsupportedTypes tests that unsupported types return error
func TestMapKey_UnsupportedTypes(t *testing.T) {
	unsupported := []Value{
		&UnitValue{},
		&ListValue{Elements: []Value{}},
		&FloatValue{Value: 3.14},
	}
	for _, v := range unsupported {
		_, err := MapKey(v)
		if err == nil {
			t.Errorf("MapKey(%T) should return error for unsupported type", v)
		}
	}
}

// TestMapValue_EmptyMap tests operations on empty map
func TestMapValue_EmptyMap(t *testing.T) {
	m := &MapValue{Entries: make(map[string]*MapEntry)}

	if m.Size() != 0 {
		t.Errorf("empty map Size() = %d, want 0", m.Size())
	}
	if m.String() != "Map{}" {
		t.Errorf("empty map String() = %q, want %q", m.String(), "Map{}")
	}
	if _, found := m.Lookup(&StringValue{Value: "x"}); found {
		t.Error("Lookup on empty map should return false")
	}
}

// TestMapValue_InsertAndLookup tests basic insert/lookup
func TestMapValue_InsertAndLookup(t *testing.T) {
	m := &MapValue{Entries: make(map[string]*MapEntry)}

	m2, err := m.Insert(&StringValue{Value: "a"}, &IntValue{Value: 1})
	if err != nil {
		t.Fatalf("Insert error: %v", err)
	}
	if m2.Size() != 1 {
		t.Errorf("after insert, Size() = %d, want 1", m2.Size())
	}

	val, found := m2.Lookup(&StringValue{Value: "a"})
	if !found {
		t.Fatal("Lookup('a') not found after insert")
	}
	if iv, ok := val.(*IntValue); !ok || iv.Value != 1 {
		t.Errorf("Lookup('a') = %v, want IntValue(1)", val)
	}

	// Original map unchanged (immutability)
	if m.Size() != 0 {
		t.Error("original map was mutated by Insert")
	}
}

// TestMapValue_InsertOverwrite tests that insert overwrites existing keys
func TestMapValue_InsertOverwrite(t *testing.T) {
	m := &MapValue{Entries: make(map[string]*MapEntry)}
	m, _ = m.Insert(&StringValue{Value: "x"}, &IntValue{Value: 1})
	m2, _ := m.Insert(&StringValue{Value: "x"}, &IntValue{Value: 99})

	val, found := m2.Lookup(&StringValue{Value: "x"})
	if !found {
		t.Fatal("key 'x' not found after overwrite")
	}
	if iv, ok := val.(*IntValue); !ok || iv.Value != 99 {
		t.Errorf("after overwrite, Lookup('x') = %v, want 99", val)
	}
	if m2.Size() != 1 {
		t.Errorf("after overwrite, Size() = %d, want 1", m2.Size())
	}
}

// TestMapValue_Remove tests key removal
func TestMapValue_Remove(t *testing.T) {
	m := &MapValue{Entries: make(map[string]*MapEntry)}
	m, _ = m.Insert(&StringValue{Value: "a"}, &IntValue{Value: 1})
	m, _ = m.Insert(&StringValue{Value: "b"}, &IntValue{Value: 2})

	m2 := m.Remove(&StringValue{Value: "a"})
	if m2.Size() != 1 {
		t.Errorf("after remove, Size() = %d, want 1", m2.Size())
	}
	if _, found := m2.Lookup(&StringValue{Value: "a"}); found {
		t.Error("removed key 'a' still found")
	}
	if _, found := m2.Lookup(&StringValue{Value: "b"}); !found {
		t.Error("key 'b' should still exist after removing 'a'")
	}

	// Original unchanged
	if m.Size() != 2 {
		t.Error("original map was mutated by Remove")
	}
}

// TestMapValue_RemoveNonexistent tests removing a key that doesn't exist
func TestMapValue_RemoveNonexistent(t *testing.T) {
	m := &MapValue{Entries: make(map[string]*MapEntry)}
	m, _ = m.Insert(&StringValue{Value: "a"}, &IntValue{Value: 1})

	m2 := m.Remove(&StringValue{Value: "z"})
	if m2.Size() != 1 {
		t.Errorf("removing nonexistent key changed size: %d", m2.Size())
	}
}

// TestMapValue_StringDeterministic runs with -count=20 to verify deterministic output.
// Go map iteration is random, so this catches nondeterminism in String().
func TestMapValue_StringDeterministic(t *testing.T) {
	m := &MapValue{Entries: make(map[string]*MapEntry)}
	m, _ = m.Insert(&StringValue{Value: "cherry"}, &IntValue{Value: 3})
	m, _ = m.Insert(&StringValue{Value: "apple"}, &IntValue{Value: 1})
	m, _ = m.Insert(&StringValue{Value: "banana"}, &IntValue{Value: 2})
	m, _ = m.Insert(&IntValue{Value: 10}, &StringValue{Value: "ten"})
	m, _ = m.Insert(&BoolValue{Value: true}, &StringValue{Value: "yes"})

	// With 5 entries and mixed key types, nondeterministic iteration would produce
	// different orderings across runs. Canonical keys sort lexicographically:
	// "b:true" < "i:10" < "s:apple" < "s:banana" < "s:cherry"
	// Note: StringValue.String() returns raw value without quotes
	expected := `Map{true: yes, 10: ten, apple: 1, banana: 2, cherry: 3}`
	got := m.String()
	if got != expected {
		t.Errorf("String() = %q\n  want   %q", got, expected)
	}
}
