package types

import (
	"strings"
	"testing"
)

// TestSafeTypeStringTruncatesOnDeepTypes verifies that SafeTypeString
// produces a truncated output for deeply nested types rather than hanging.
func TestSafeTypeStringTruncatesOnDeepTypes(t *testing.T) {
	// Build a deeply nested type: Foo[Foo[Foo[...]]]
	// Depth exceeds MaxStringifyDepth (100)
	deep := Type(&TCon{Name: "int"})
	for i := 0; i < MaxStringifyDepth+50; i++ {
		deep = &TApp{
			Constructor: &TCon{Name: "Foo"},
			Args:        []Type{deep},
		}
	}

	result := SafeTypeString(deep)

	// Should contain truncation marker, not hang
	if !strings.Contains(result, "...depth limit") {
		t.Errorf("expected truncation marker '...depth limit', got: %s", result)
	}

	// Result should be reasonable length (not infinitely long)
	if len(result) > 10000 {
		t.Errorf("result too long (%d chars), possible infinite loop", len(result))
	}
}

// TestSafeTypeStringTruncatesOnCyclicTypes verifies that SafeTypeString
// handles cyclic types (mu-types) without hanging.
func TestSafeTypeStringTruncatesOnCyclicTypes(t *testing.T) {
	// Create a cyclic type: μX. Foo[X]
	// This simulates a recursive ADT like List[a]
	cyclic := &TApp{
		Constructor: &TCon{Name: "Foo"},
		Args:        []Type{nil}, // Will be set to cyclic
	}
	cyclic.Args[0] = cyclic // Create cycle

	result := SafeTypeString(cyclic)

	// Should truncate with cycle marker, not hang forever
	if !strings.Contains(result, "...cycle") {
		t.Errorf("expected cycle marker '...cycle', got: %s", result)
	}

	// Result should be reasonable length
	if len(result) > 1000 {
		t.Errorf("result too long (%d chars), possible infinite loop", len(result))
	}
}

// TestSafeTypeStringNormalTypes verifies SafeTypeString works correctly
// for normal (non-cyclic, non-deep) types.
func TestSafeTypeStringNormalTypes(t *testing.T) {
	tests := []struct {
		name     string
		typ      Type
		expected string
	}{
		{
			name:     "simple int",
			typ:      &TCon{Name: "int"},
			expected: "int",
		},
		{
			name:     "type variable",
			typ:      &TVar{Name: "a"},
			expected: "a",
		},
		{
			name:     "list of int",
			typ:      &TList{Element: &TCon{Name: "int"}},
			expected: "[int]",
		},
		{
			name: "function type",
			typ: &TFunc2{
				Params: []Type{&TCon{Name: "int"}},
				Return: &TCon{Name: "string"},
			},
			expected: "int -> string",
		},
		{
			name: "tuple type",
			typ: &TTuple{
				Elements: []Type{&TCon{Name: "int"}, &TCon{Name: "string"}},
			},
			expected: "(int, string)",
		},
		{
			name: "type application",
			typ: &TApp{
				Constructor: &TCon{Name: "Maybe"},
				Args:        []Type{&TCon{Name: "int"}},
			},
			expected: "Maybe[int]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SafeTypeString(tt.typ)
			if result != tt.expected {
				t.Errorf("SafeTypeString(%s) = %q, want %q", tt.name, result, tt.expected)
			}
		})
	}
}

// TestSafeTypeStringNil verifies SafeTypeString handles nil input gracefully.
func TestSafeTypeStringNil(t *testing.T) {
	result := SafeTypeString(nil)
	if result != "nil" {
		t.Errorf("SafeTypeString(nil) = %q, want \"nil\"", result)
	}
}

// TestTruncatedTypeString verifies the length-limited wrapper.
func TestTruncatedTypeString(t *testing.T) {
	// Create a moderately complex type
	complex := &TFunc2{
		Params: []Type{
			&TRecord{Fields: map[string]Type{
				"name": &TCon{Name: "string"},
				"age":  &TCon{Name: "int"},
			}},
		},
		Return: &TCon{Name: "bool"},
	}

	// Full result
	full := SafeTypeString(complex)

	// Truncated to 20 chars
	truncated := TruncatedTypeString(complex, 20)

	if len(truncated) > 20 {
		t.Errorf("TruncatedTypeString should be <= 20 chars, got %d", len(truncated))
	}

	if len(full) > 20 && !strings.HasSuffix(truncated, "...") {
		t.Errorf("TruncatedTypeString should end with '...' when truncated")
	}
}
