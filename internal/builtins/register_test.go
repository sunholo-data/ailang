package builtins

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/sunholo/ailang/internal/eval"
)

// TestStrLenRegistered tests that _str_len is registered
func TestStrLenRegistered(t *testing.T) {
	spec, ok := GetSpec("_str_len")
	require.True(t, ok, "_str_len should be registered")

	assert.Equal(t, "std/string", spec.Module)
	assert.Equal(t, "_str_len", spec.Name)
	assert.Equal(t, 1, spec.NumArgs)
	assert.True(t, spec.IsPure)
	assert.Equal(t, "", spec.Effect) // Pure function
}

// TestStrLenType tests that _str_len has correct type
func TestStrLenType(t *testing.T) {
	spec, ok := GetSpec("_str_len")
	require.True(t, ok)

	typ := spec.Type()
	require.NotNil(t, typ)

	// Should be a function type
	assert.Contains(t, typ.String(), "string")
	assert.Contains(t, typ.String(), "int")
}

// TestStrLenImpl tests the implementation
func TestStrLenImpl(t *testing.T) {
	spec, ok := GetSpec("_str_len")
	require.True(t, ok)

	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{"empty string", "", 0},
		{"ascii", "hello", 5},
		{"unicode", "世界", 2},    // 2 characters
		{"mixed", "hello世界", 7}, // 5 + 2
		{"emoji", "👋🌍", 2},      // 2 emoji
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := []eval.Value{&eval.StringValue{Value: tt.input}}
			result, err := spec.Impl(nil, args) // nil context for pure function

			require.NoError(t, err)
			intVal, ok := result.(*eval.IntValue)
			require.True(t, ok, "result should be IntValue")
			assert.Equal(t, tt.expected, intVal.Value)
		})
	}
}
