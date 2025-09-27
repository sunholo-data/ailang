package types

import (
	"testing"
)

// TestMakeDictionaryKey_Canonicalizes verifies that MakeDictionaryKey 
// properly normalizes type names (int→Int, float→Float, etc.)
func TestMakeDictionaryKey_Canonicalizes(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
		className string
		typ       Type
		method    string
		want      string
	}{
		{
			name:      "int normalizes to Int",
			namespace: "prelude",
			className: "Num",
			typ:       &TCon{Name: "int"},
			method:    "add",
			want:      "prelude::Num::Int::add",
		},
		{
			name:      "float normalizes to Float",
			namespace: "prelude",
			className: "Fractional",
			typ:       &TCon{Name: "float"},
			method:    "divide",
			want:      "prelude::Fractional::Float::divide",
		},
		{
			name:      "string normalizes to String",
			namespace: "prelude",
			className: "Eq",
			typ:       &TCon{Name: "string"},
			method:    "eq",
			want:      "prelude::Eq::String::eq",
		},
		{
			name:      "bool normalizes to Bool",
			namespace: "prelude",
			className: "Ord",
			typ:       &TCon{Name: "bool"},
			method:    "lt",
			want:      "prelude::Ord::Bool::lt",
		},
		{
			name:      "already normalized Int stays Int",
			namespace: "prelude",
			className: "Num",
			typ:       &TCon{Name: "Int"},
			method:    "mul",
			want:      "prelude::Num::Int::mul",
		},
		{
			name:      "dictionary reference (no method)",
			namespace: "prelude",
			className: "Ord",
			typ:       &TCon{Name: "float"},
			method:    "",
			want:      "prelude::Ord::Float",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MakeDictionaryKey(tt.namespace, tt.className, tt.typ, tt.method)
			if got != tt.want {
				t.Errorf("MakeDictionaryKey(%q, %q, %v, %q) = %q, want %q",
					tt.namespace, tt.className, tt.typ, tt.method, got, tt.want)
			}
		})
	}
}

// TestInstanceEnv_CanonicalKey verifies that the InstanceEnv's canonicalKey
// function properly normalizes type names and uses double-colon separator
func TestInstanceEnv_CanonicalKey(t *testing.T) {
	tests := []struct {
		name      string
		className string
		typ       Type
		want      string
	}{
		{
			name:      "Eq with string type",
			className: "Eq",
			typ:       &TCon{Name: "string"},
			want:      "Eq::String",
		},
		{
			name:      "Ord with int type",
			className: "Ord",
			typ:       &TCon{Name: "int"},
			want:      "Ord::Int",
		},
		{
			name:      "Num with float type",
			className: "Num",
			typ:       &TCon{Name: "float"},
			want:      "Num::Float",
		},
		{
			name:      "Show with bool type",
			className: "Show",
			typ:       &TCon{Name: "bool"},
			want:      "Show::Bool",
		},
		{
			name:      "already normalized type",
			className: "Eq",
			typ:       &TCon{Name: "Int"},
			want:      "Eq::Int",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := canonicalKey(tt.className, tt.typ)
			if got != tt.want {
				t.Errorf("canonicalKey(%q, %v) = %q, want %q",
					tt.className, tt.typ, got, tt.want)
			}
		})
	}
}

// TestKeyConsistency verifies that all key generation methods produce
// consistent results for the same inputs
func TestKeyConsistency(t *testing.T) {
	// Test that different paths to create the same key produce identical results
	typ := &TCon{Name: "int"}
	className := "Num"
	namespace := "prelude"
	method := "add"
	
	// Create key using MakeDictionaryKey
	key1 := MakeDictionaryKey(namespace, className, typ, method)
	
	// What the key should normalize to
	expectedKey := "prelude::Num::Int::add"
	
	if key1 != expectedKey {
		t.Errorf("MakeDictionaryKey produced %q, expected %q", key1, expectedKey)
	}
	
	// Verify that normalizing "int" always produces "Int"
	normalized := NormalizeTypeName(typ)
	if normalized != "Int" {
		t.Errorf("NormalizeTypeName(%v) = %q, expected \"Int\"", typ, normalized)
	}
	
	// Test with float type
	floatTyp := &TCon{Name: "float"}
	floatKey := MakeDictionaryKey(namespace, "Fractional", floatTyp, "divide")
	expectedFloatKey := "prelude::Fractional::Float::divide"
	
	if floatKey != expectedFloatKey {
		t.Errorf("MakeDictionaryKey for float produced %q, expected %q", floatKey, expectedFloatKey)
	}
}