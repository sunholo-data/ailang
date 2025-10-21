package builtins

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/sunholo/ailang/internal/effects/testctx"
	"github.com/sunholo/ailang/internal/eval"
)

// Helper to extract value from Result type
func extractOk(result eval.Value) eval.Value {
	ctor, ok := result.(*eval.TaggedValue)
	if !ok || ctor.CtorName != "Ok" || len(ctor.Fields) != 1 {
		return nil
	}
	return ctor.Fields[0]
}

func extractErr(result eval.Value) string {
	ctor, ok := result.(*eval.TaggedValue)
	if !ok || ctor.CtorName != "Err" || len(ctor.Fields) != 1 {
		return ""
	}
	strVal, ok := ctor.Fields[0].(*eval.StringValue)
	if !ok {
		return ""
	}
	return strVal.Value
}

func isOk(result eval.Value) bool {
	ctor, ok := result.(*eval.TaggedValue)
	return ok && ctor.CtorName == "Ok"
}

// Streaming Builder Tests

func TestJSONBuilder_Null(t *testing.T) {
	builder := newJSONBuilder("null")
	result, err := builder.build()
	require.NoError(t, err)

	ctor, ok := result.(*eval.TaggedValue)
	require.True(t, ok)
	assert.Equal(t, "JNull", ctor.CtorName)
	assert.Len(t, ctor.Fields, 0)
}

func TestJSONBuilder_True(t *testing.T) {
	builder := newJSONBuilder("true")
	result, err := builder.build()
	require.NoError(t, err)

	ctor, ok := result.(*eval.TaggedValue)
	require.True(t, ok)
	assert.Equal(t, "JBool", ctor.CtorName)
	require.Len(t, ctor.Fields, 1)

	boolVal, ok := ctor.Fields[0].(*eval.BoolValue)
	require.True(t, ok)
	assert.True(t, boolVal.Value)
}

func TestJSONBuilder_False(t *testing.T) {
	builder := newJSONBuilder("false")
	result, err := builder.build()
	require.NoError(t, err)

	ctor, ok := result.(*eval.TaggedValue)
	require.True(t, ok)
	assert.Equal(t, "JBool", ctor.CtorName)
	require.Len(t, ctor.Fields, 1)

	boolVal, ok := ctor.Fields[0].(*eval.BoolValue)
	require.True(t, ok)
	assert.False(t, boolVal.Value)
}

func TestJSONBuilder_Integer(t *testing.T) {
	builder := newJSONBuilder("42")
	result, err := builder.build()
	require.NoError(t, err)

	ctor, ok := result.(*eval.TaggedValue)
	require.True(t, ok)
	assert.Equal(t, "JNumber", ctor.CtorName)
	require.Len(t, ctor.Fields, 1)

	floatVal, ok := ctor.Fields[0].(*eval.FloatValue)
	require.True(t, ok)
	assert.Equal(t, 42.0, floatVal.Value)
}

func TestJSONBuilder_Float(t *testing.T) {
	builder := newJSONBuilder("3.14")
	result, err := builder.build()
	require.NoError(t, err)

	ctor, ok := result.(*eval.TaggedValue)
	require.True(t, ok)
	assert.Equal(t, "JNumber", ctor.CtorName)
	require.Len(t, ctor.Fields, 1)

	floatVal, ok := ctor.Fields[0].(*eval.FloatValue)
	require.True(t, ok)
	assert.InDelta(t, 3.14, floatVal.Value, 0.001)
}

func TestJSONBuilder_Scientific(t *testing.T) {
	builder := newJSONBuilder("1.5e10")
	result, err := builder.build()
	require.NoError(t, err)

	ctor, ok := result.(*eval.TaggedValue)
	require.True(t, ok)
	assert.Equal(t, "JNumber", ctor.CtorName)
	require.Len(t, ctor.Fields, 1)

	floatVal, ok := ctor.Fields[0].(*eval.FloatValue)
	require.True(t, ok)
	assert.Equal(t, 1.5e10, floatVal.Value)
}

func TestJSONBuilder_String(t *testing.T) {
	builder := newJSONBuilder(`"hello"`)
	result, err := builder.build()
	require.NoError(t, err)

	ctor, ok := result.(*eval.TaggedValue)
	require.True(t, ok)
	assert.Equal(t, "JString", ctor.CtorName)
	require.Len(t, ctor.Fields, 1)

	strVal, ok := ctor.Fields[0].(*eval.StringValue)
	require.True(t, ok)
	assert.Equal(t, "hello", strVal.Value)
}

func TestJSONBuilder_StringEscapes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"newline", `"hello\nworld"`, "hello\nworld"},
		{"tab", `"hello\tworld"`, "hello\tworld"},
		{"quote", `"say \"hi\""`, `say "hi"`},
		{"backslash", `"path\\file"`, `path\file`},
		{"unicode", `"\u0048\u0065\u006C\u006C\u006F"`, "Hello"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := newJSONBuilder(tt.input)
			result, err := builder.build()
			require.NoError(t, err)

			ctor, ok := result.(*eval.TaggedValue)
			require.True(t, ok)
			assert.Equal(t, "JString", ctor.CtorName)

			strVal, ok := ctor.Fields[0].(*eval.StringValue)
			require.True(t, ok)
			assert.Equal(t, tt.expected, strVal.Value)
		})
	}
}

func TestJSONBuilder_EmptyArray(t *testing.T) {
	builder := newJSONBuilder("[]")
	result, err := builder.build()
	require.NoError(t, err)

	ctor, ok := result.(*eval.TaggedValue)
	require.True(t, ok)
	assert.Equal(t, "JArray", ctor.CtorName)
	require.Len(t, ctor.Fields, 1)

	listVal, ok := ctor.Fields[0].(*eval.ListValue)
	require.True(t, ok)
	assert.Len(t, listVal.Elements, 0)
}

func TestJSONBuilder_ArraySingle(t *testing.T) {
	builder := newJSONBuilder("[1]")
	result, err := builder.build()
	require.NoError(t, err)

	ctor, ok := result.(*eval.TaggedValue)
	require.True(t, ok)
	assert.Equal(t, "JArray", ctor.CtorName)

	listVal, ok := ctor.Fields[0].(*eval.ListValue)
	require.True(t, ok)
	require.Len(t, listVal.Elements, 1)

	elem, ok := listVal.Elements[0].(*eval.TaggedValue)
	require.True(t, ok)
	assert.Equal(t, "JNumber", elem.CtorName)
}

func TestJSONBuilder_ArrayMultiple(t *testing.T) {
	builder := newJSONBuilder("[1,2,3]")
	result, err := builder.build()
	require.NoError(t, err)

	ctor, ok := result.(*eval.TaggedValue)
	require.True(t, ok)
	assert.Equal(t, "JArray", ctor.CtorName)

	listVal, ok := ctor.Fields[0].(*eval.ListValue)
	require.True(t, ok)
	assert.Len(t, listVal.Elements, 3)
}

func TestJSONBuilder_ArrayNested(t *testing.T) {
	builder := newJSONBuilder("[[1],[2]]")
	result, err := builder.build()
	require.NoError(t, err)

	ctor, ok := result.(*eval.TaggedValue)
	require.True(t, ok)
	assert.Equal(t, "JArray", ctor.CtorName)

	listVal, ok := ctor.Fields[0].(*eval.ListValue)
	require.True(t, ok)
	require.Len(t, listVal.Elements, 2)

	// Check first nested array
	nested, ok := listVal.Elements[0].(*eval.TaggedValue)
	require.True(t, ok)
	assert.Equal(t, "JArray", nested.CtorName)
}

func TestJSONBuilder_EmptyObject(t *testing.T) {
	builder := newJSONBuilder("{}")
	result, err := builder.build()
	require.NoError(t, err)

	ctor, ok := result.(*eval.TaggedValue)
	require.True(t, ok)
	assert.Equal(t, "JObject", ctor.CtorName)
	require.Len(t, ctor.Fields, 1)

	listVal, ok := ctor.Fields[0].(*eval.ListValue)
	require.True(t, ok)
	assert.Len(t, listVal.Elements, 0)
}

func TestJSONBuilder_ObjectSingle(t *testing.T) {
	builder := newJSONBuilder(`{"a":1}`)
	result, err := builder.build()
	require.NoError(t, err)

	ctor, ok := result.(*eval.TaggedValue)
	require.True(t, ok)
	assert.Equal(t, "JObject", ctor.CtorName)

	listVal, ok := ctor.Fields[0].(*eval.ListValue)
	require.True(t, ok)
	require.Len(t, listVal.Elements, 1)

	// Check key-value pair
	kvRec, ok := listVal.Elements[0].(*eval.RecordValue)
	require.True(t, ok)

	keyVal, ok := kvRec.Fields["key"].(*eval.StringValue)
	require.True(t, ok)
	assert.Equal(t, "a", keyVal.Value)

	valCtor, ok := kvRec.Fields["value"].(*eval.TaggedValue)
	require.True(t, ok)
	assert.Equal(t, "JNumber", valCtor.CtorName)
}

func TestJSONBuilder_ObjectMultiple(t *testing.T) {
	builder := newJSONBuilder(`{"a":1,"b":2}`)
	result, err := builder.build()
	require.NoError(t, err)

	ctor, ok := result.(*eval.TaggedValue)
	require.True(t, ok)
	assert.Equal(t, "JObject", ctor.CtorName)

	listVal, ok := ctor.Fields[0].(*eval.ListValue)
	require.True(t, ok)
	assert.Len(t, listVal.Elements, 2)
}

func TestJSONBuilder_ObjectKeyOrder(t *testing.T) {
	// Order preservation is critical for deterministic round-trip
	builder := newJSONBuilder(`{"z":1,"a":2,"m":3}`)
	result, err := builder.build()
	require.NoError(t, err)

	ctor, ok := result.(*eval.TaggedValue)
	require.True(t, ok)
	assert.Equal(t, "JObject", ctor.CtorName)

	listVal, ok := ctor.Fields[0].(*eval.ListValue)
	require.True(t, ok)
	require.Len(t, listVal.Elements, 3)

	// Verify keys are in source order
	keys := []string{}
	for _, elem := range listVal.Elements {
		kvRec, ok := elem.(*eval.RecordValue)
		require.True(t, ok)
		keyVal, ok := kvRec.Fields["key"].(*eval.StringValue)
		require.True(t, ok)
		keys = append(keys, keyVal.Value)
	}

	assert.Equal(t, []string{"z", "a", "m"}, keys, "Keys should preserve source order")
}

func TestJSONBuilder_ObjectNested(t *testing.T) {
	builder := newJSONBuilder(`{"a":{"b":1}}`)
	result, err := builder.build()
	require.NoError(t, err)

	ctor, ok := result.(*eval.TaggedValue)
	require.True(t, ok)
	assert.Equal(t, "JObject", ctor.CtorName)

	listVal, ok := ctor.Fields[0].(*eval.ListValue)
	require.True(t, ok)
	require.Len(t, listVal.Elements, 1)

	// Check nested object
	kvRec, ok := listVal.Elements[0].(*eval.RecordValue)
	require.True(t, ok)

	nested, ok := kvRec.Fields["value"].(*eval.TaggedValue)
	require.True(t, ok)
	assert.Equal(t, "JObject", nested.CtorName)
}

func TestJSONBuilder_Mixed(t *testing.T) {
	builder := newJSONBuilder(`[{"a":[1,2]},{"b":null}]`)
	result, err := builder.build()
	require.NoError(t, err)

	ctor, ok := result.(*eval.TaggedValue)
	require.True(t, ok)
	assert.Equal(t, "JArray", ctor.CtorName)
}

func TestJSONBuilder_DeeplyNested(t *testing.T) {
	builder := newJSONBuilder(`{"a":{"b":{"c":{"d":{"e":1}}}}}`)
	result, err := builder.build()
	require.NoError(t, err)

	ctor, ok := result.(*eval.TaggedValue)
	require.True(t, ok)
	assert.Equal(t, "JObject", ctor.CtorName)
}

// Error Tests

func TestJSONBuilder_InvalidSyntax(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"unclosed array", "[1,2"},
		{"unclosed object", `{"a":1`},
		{"trailing comma array", "[1,]"},
		{"trailing comma object", `{"a":1,}`},
		{"missing colon", `{"a" 1}`},
		{"empty", ""},
		{"just comma", "{,}"},
		{"invalid value", `{"a":}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := newJSONBuilder(tt.input)
			_, err := builder.build()
			assert.Error(t, err, "Expected error for invalid JSON")
		})
	}
}

func TestJSONBuilder_WhitespaceOnly(t *testing.T) {
	builder := newJSONBuilder("   \n\t  ")
	_, err := builder.build()
	assert.Error(t, err, "Expected error for whitespace-only input")
}

func TestJSONBuilder_UnicodeEmoji(t *testing.T) {
	builder := newJSONBuilder(`{"emoji":"🎉"}`)
	result, err := builder.build()
	require.NoError(t, err)

	ctor, ok := result.(*eval.TaggedValue)
	require.True(t, ok)
	assert.Equal(t, "JObject", ctor.CtorName)
}

// Integration Tests - Testing the builtin with Result type wrapping

func TestJSONDecodeBuiltin_ValidJSON(t *testing.T) {
	ctx := testctx.NewMockEffContext()

	tests := []struct {
		name     string
		input    string
		expected string // Expected constructor name
	}{
		{"null", "null", "JNull"},
		{"true", "true", "JBool"},
		{"false", "false", "JBool"},
		{"integer", "42", "JNumber"},
		{"float", "3.14", "JNumber"},
		{"string", `"hello"`, "JString"},
		{"array", "[1,2,3]", "JArray"},
		{"object", `{"a":1}`, "JObject"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := jsonDecodeImpl(ctx.EffContext, []eval.Value{
				&eval.StringValue{Value: tt.input},
			})
			require.NoError(t, err)

			// Check it's an Ok result
			okCtor, ok := result.(*eval.TaggedValue)
			require.True(t, ok, "Expected TaggedValue, got %T", result)
			assert.Equal(t, "Ok", okCtor.CtorName)
			require.Len(t, okCtor.Fields, 1)

			// Check the wrapped JSON value
			jsonVal, ok := okCtor.Fields[0].(*eval.TaggedValue)
			require.True(t, ok)
			assert.Equal(t, tt.expected, jsonVal.CtorName)
		})
	}
}

func TestJSONDecodeBuiltin_InvalidJSON(t *testing.T) {
	ctx := testctx.NewMockEffContext()

	tests := []struct {
		name  string
		input string
	}{
		{"unclosed array", "[1,2"},
		{"unclosed object", `{"a":1`},
		{"trailing comma", "[1,]"},
		{"empty", ""},
		{"whitespace only", "   \n\t  "},
		{"invalid value", `{"a":}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := jsonDecodeImpl(ctx.EffContext, []eval.Value{
				&eval.StringValue{Value: tt.input},
			})
			require.NoError(t, err, "Builtin should not return Go error")

			// Check it's an Err result
			errCtor, ok := result.(*eval.TaggedValue)
			require.True(t, ok)
			assert.Equal(t, "Err", errCtor.CtorName)
			require.Len(t, errCtor.Fields, 1)

			// Check the error message is a string
			errMsg, ok := errCtor.Fields[0].(*eval.StringValue)
			require.True(t, ok)
			assert.NotEmpty(t, errMsg.Value, "Error message should not be empty")
		})
	}
}

func TestJSONDecodeBuiltin_KeyOrderPreservation(t *testing.T) {
	ctx := testctx.NewMockEffContext()

	// Object with non-alphabetical key order
	input := `{"z":1,"a":2,"m":3}`

	result, err := jsonDecodeImpl(ctx.EffContext, []eval.Value{
		&eval.StringValue{Value: input},
	})
	require.NoError(t, err)

	// Extract Ok result
	okCtor := result.(*eval.TaggedValue)
	require.Equal(t, "Ok", okCtor.CtorName)

	// Extract JObject
	jsonVal := okCtor.Fields[0].(*eval.TaggedValue)
	require.Equal(t, "JObject", jsonVal.CtorName)

	// Extract key-value list
	listVal := jsonVal.Fields[0].(*eval.ListValue)
	require.Len(t, listVal.Elements, 3)

	// Verify keys are in source order
	keys := []string{}
	for _, elem := range listVal.Elements {
		kvRec, ok := elem.(*eval.RecordValue)
		require.True(t, ok)
		keyVal, ok := kvRec.Fields["key"].(*eval.StringValue)
		require.True(t, ok)
		keys = append(keys, keyVal.Value)
	}

	assert.Equal(t, []string{"z", "a", "m"}, keys, "Keys should preserve source order")
}

func TestJSONDecodeBuiltin_UnicodeSupport(t *testing.T) {
	ctx := testctx.NewMockEffContext()

	// Test Unicode escapes and emoji
	input := `{"greeting":"Hello \u0057orld","emoji":"🎉"}`

	result, err := jsonDecodeImpl(ctx.EffContext, []eval.Value{
		&eval.StringValue{Value: input},
	})
	require.NoError(t, err)

	// Extract Ok result
	okCtor := result.(*eval.TaggedValue)
	require.Equal(t, "Ok", okCtor.CtorName)

	// Extract JObject
	jsonVal := okCtor.Fields[0].(*eval.TaggedValue)
	assert.Equal(t, "JObject", jsonVal.CtorName)
}
