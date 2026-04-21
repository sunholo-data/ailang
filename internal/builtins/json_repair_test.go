package builtins

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sunholo-data/ailang/internal/effects/testctx"
	"github.com/sunholo-data/ailang/internal/eval"
)

func TestJSONRepairValidJSON(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"object", `{"name": "Alice", "age": 30}`},
		{"array", `[1, 2, 3]`},
		{"string", `"hello"`},
		{"number", `42`},
		{"true", `true`},
		{"false", `false`},
		{"null", `null`},
		{"nested", `{"a": [1, {"b": true}]}`},
		{"empty object", `{}`},
		{"empty array", `[]`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := repairJSON(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.input, result)
		})
	}
}

func TestJSONRepairWhitespacePadding(t *testing.T) {
	// Gemini structured output pads with trailing spaces
	input := `{"name": "Alice"}` + "    \t\n   "
	result, err := repairJSON(input)
	require.NoError(t, err)
	assert.Equal(t, `{"name": "Alice"}`, result)
}

func TestJSONRepairLeadingWhitespace(t *testing.T) {
	input := "  \n\t" + `{"name": "Alice"}`
	result, err := repairJSON(input)
	require.NoError(t, err)
	assert.Equal(t, `{"name": "Alice"}`, result)
}

func TestJSONRepairUnclosedString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains string // expected to contain this substring
	}{
		{
			"simple string",
			`{"key": "val`,
			`"val"`, // string should be closed
		},
		{
			"object with unclosed value",
			`{"name": "Alice`, // truncated mid-value
			`"Alice"`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := repairJSON(tt.input)
			require.NoError(t, err)
			assert.Contains(t, result, tt.contains)
			// Result should be valid JSON
			assert.True(t, isValidJSON(result), "repaired JSON should be valid: %s", result)
		})
	}
}

func TestJSONRepairUnclosedArray(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple", `[1, 2, 3`, `[1, 2, 3]`},
		{"nested", `[1, [2, 3`, `[1, [2, 3]]`},
		{"with trailing comma", `[1, 2,`, `[1, 2]`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := repairJSON(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestJSONRepairUnclosedObject(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple", `{"a": 1`, `{"a": 1}`},
		{"nested", `{"a": {"b": 2`, `{"a": {"b": 2}}`},
		{"with trailing comma", `{"a": 1,`, `{"a": 1}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := repairJSON(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestJSONRepairTrailingComma(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"object", `{"a": 1,}`, `{"a": 1}`},
		{"array", `[1, 2, 3,]`, `[1, 2, 3]`},
		{"nested object", `{"a": [1,],}`, `{"a": [1]}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := repairJSON(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestJSONRepairTruncatedKeyword(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"tru", `{"done": tru`, `{"done": true}`},
		{"fals", `{"done": fals`, `{"done": false}`},
		{"nul", `{"val": nul`, `{"val": null}`},
		{"fal", `[fal`, `[false]`},
		{"tr", `[tr`, `[true]`},
		{"nu", `[nu`, `[null]`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := repairJSON(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
			assert.True(t, isValidJSON(result), "repaired JSON should be valid: %s", result)
		})
	}
}

func TestJSONRepairMixedTruncation(t *testing.T) {
	// Object with unclosed string inside unclosed array inside unclosed object
	input := `{"items": [{"name": "Alic`
	result, err := repairJSON(input)
	require.NoError(t, err)
	assert.True(t, isValidJSON(result), "repaired JSON should be valid: %s", result)
}

func TestJSONRepairEmptyInput(t *testing.T) {
	_, err := repairJSON("")
	assert.Error(t, err)

	_, err = repairJSON("   ")
	assert.Error(t, err)
}

func TestJSONRepairKeyWithoutValue(t *testing.T) {
	// Object truncated right after a colon
	input := `{"name":`
	result, err := repairJSON(input)
	require.NoError(t, err)
	assert.True(t, isValidJSON(result), "repaired JSON should be valid: %s", result)
	assert.Contains(t, result, "null")
}

func TestJSONRepairBuiltinImpl(t *testing.T) {
	ctx := testctx.NewMockEffContext()

	// Test with valid JSON
	result, err := jsonRepairImpl(ctx.EffContext, []eval.Value{
		&eval.StringValue{Value: `{"key": "value"}`},
	})
	require.NoError(t, err)
	tagged, ok := result.(*eval.TaggedValue)
	require.True(t, ok)
	assert.Equal(t, "Ok", tagged.CtorName)
	assert.Equal(t, `{"key": "value"}`, testctx.GetString(tagged.Fields[0]))

	// Test with truncated JSON
	result, err = jsonRepairImpl(ctx.EffContext, []eval.Value{
		&eval.StringValue{Value: `{"key": "val`},
	})
	require.NoError(t, err)
	tagged, ok = result.(*eval.TaggedValue)
	require.True(t, ok)
	assert.Equal(t, "Ok", tagged.CtorName)
	repaired := testctx.GetString(tagged.Fields[0])
	assert.True(t, isValidJSON(repaired), "repaired JSON should be valid: %s", repaired)

	// Test with empty input
	result, err = jsonRepairImpl(ctx.EffContext, []eval.Value{
		&eval.StringValue{Value: ""},
	})
	require.NoError(t, err)
	tagged, ok = result.(*eval.TaggedValue)
	require.True(t, ok)
	assert.Equal(t, "Err", tagged.CtorName)
}

func TestJSONRepairDanglingBackslash(t *testing.T) {
	// String truncated after a backslash
	input := `{"path": "C:\`
	result, err := repairJSON(input)
	require.NoError(t, err)
	assert.True(t, isValidJSON(result), "repaired JSON should be valid: %s", result)
}

func TestJSONRepairLargeWhitespacePadding(t *testing.T) {
	// Simulates the Gemini bug: valid JSON followed by ~4KB of spaces
	json := `{"content": "extracted text from PDF document"}`
	padding := make([]byte, 4096)
	for i := range padding {
		padding[i] = ' '
	}
	input := json + string(padding)

	result, err := repairJSON(input)
	require.NoError(t, err)
	assert.Equal(t, json, result)
}
