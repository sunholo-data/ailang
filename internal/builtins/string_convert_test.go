package builtins

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/sunholo-data/ailang/internal/eval"
)

func TestFloatToStrRegistered(t *testing.T) {
	spec, ok := GetSpec("_string_floatToStr")
	require.True(t, ok, "_string_floatToStr should be registered")

	assert.Equal(t, "std/string", spec.Module)
	assert.Equal(t, "_string_floatToStr", spec.Name)
	assert.Equal(t, 1, spec.NumArgs)
	assert.True(t, spec.IsPure)
	assert.Equal(t, "", spec.Effect)
}

func TestFloatToStrType(t *testing.T) {
	spec, ok := GetSpec("_string_floatToStr")
	require.True(t, ok)

	typ := spec.Type()
	require.NotNil(t, typ)

	// Should be float -> string
	assert.Contains(t, typ.String(), "float")
	assert.Contains(t, typ.String(), "string")
}

func TestFloatToStrImpl(t *testing.T) {
	spec, ok := GetSpec("_string_floatToStr")
	require.True(t, ok)

	tests := []struct {
		name     string
		input    float64
		expected string
	}{
		{"positive", 3.14, "3.14"},
		{"negative", -0.5, "-0.5"},
		{"integer_value", 42.0, "42"},
		{"zero", 0.0, "0"},
		{"small", 0.0001, "0.0001"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := []eval.Value{&eval.FloatValue{Value: tt.input}}
			result, err := spec.Impl(nil, args)
			require.NoError(t, err)

			strVal, ok := result.(*eval.StringValue)
			require.True(t, ok, "expected StringValue")
			assert.Equal(t, tt.expected, strVal.Value)
		})
	}
}

func TestIntToStrRegistered(t *testing.T) {
	spec, ok := GetSpec("_string_intToStr")
	require.True(t, ok, "_string_intToStr should be registered")

	assert.Equal(t, "std/string", spec.Module)
	assert.Equal(t, "_string_intToStr", spec.Name)
	assert.Equal(t, 1, spec.NumArgs)
	assert.True(t, spec.IsPure)
	assert.Equal(t, "", spec.Effect)
}

func TestIntToStrType(t *testing.T) {
	spec, ok := GetSpec("_string_intToStr")
	require.True(t, ok)

	typ := spec.Type()
	require.NotNil(t, typ)

	// Should be int -> string
	assert.Contains(t, typ.String(), "int")
	assert.Contains(t, typ.String(), "string")
}

func TestIntToStrImpl(t *testing.T) {
	spec, ok := GetSpec("_string_intToStr")
	require.True(t, ok)

	tests := []struct {
		name     string
		input    int
		expected string
	}{
		{"positive", 42, "42"},
		{"negative", -100, "-100"},
		{"zero", 0, "0"},
		{"large", 1234567890, "1234567890"},
		{"negative_large", -987654321, "-987654321"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := []eval.Value{&eval.IntValue{Value: tt.input}}
			result, err := spec.Impl(nil, args)
			require.NoError(t, err)

			strVal, ok := result.(*eval.StringValue)
			require.True(t, ok, "expected StringValue")
			assert.Equal(t, tt.expected, strVal.Value)
		})
	}
}

func TestStringConversionMetadata(t *testing.T) {
	builtins := []string{"_string_floatToStr", "_string_intToStr"}

	for _, name := range builtins {
		t.Run(name, func(t *testing.T) {
			spec, ok := GetSpec(name)
			require.True(t, ok, "%s should be registered", name)

			require.NotNil(t, spec.Metadata, "%s should have metadata", name)
			assert.Equal(t, "v0.5.10", spec.Metadata.Since)
			assert.Equal(t, "string", spec.Metadata.Category)
			assert.Contains(t, spec.Metadata.Tags, "conversion")
		})
	}
}
