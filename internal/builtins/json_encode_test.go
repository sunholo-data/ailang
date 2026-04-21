package builtins

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/sunholo-data/ailang/internal/effects/testctx"
	"github.com/sunholo-data/ailang/internal/eval"
)

// Test individual JSON types
// Note: extractOk, extractErr, isOk helpers are defined in json_decode_test.go

func TestJSONEncodeNull(t *testing.T) {
	ctx := testctx.NewMockEffContext()

	jnull := makeJNull()
	result, err := jsonEncodeImpl(ctx.EffContext, []eval.Value{jnull})

	require.NoError(t, err)
	assert.Equal(t, "null", testctx.GetString(result))
}

func TestJSONEncodeBoolTrue(t *testing.T) {
	ctx := testctx.NewMockEffContext()

	jtrue := makeJBool(true)
	result, err := jsonEncodeImpl(ctx.EffContext, []eval.Value{jtrue})

	require.NoError(t, err)
	assert.Equal(t, "true", testctx.GetString(result))
}

func TestJSONEncodeBoolFalse(t *testing.T) {
	ctx := testctx.NewMockEffContext()

	jfalse := makeJBool(false)
	result, err := jsonEncodeImpl(ctx.EffContext, []eval.Value{jfalse})

	require.NoError(t, err)
	assert.Equal(t, "false", testctx.GetString(result))
}

func TestJSONEncodeNumber(t *testing.T) {
	ctx := testctx.NewMockEffContext()

	tests := []struct {
		name     string
		input    float64
		expected string
	}{
		{"integer", 42.0, "42"},
		{"negative", -17.0, "-17"},
		{"zero", 0.0, "0"},
		{"float", 3.14, "3.14"},
		{"negative float", -2.5, "-2.5"},
		{"small float", 0.001, "0.001"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jnum := makeJNumberFloat(tt.input)
			result, err := jsonEncodeImpl(ctx.EffContext, []eval.Value{jnum})

			require.NoError(t, err)
			assert.Equal(t, tt.expected, testctx.GetString(result))
		})
	}
}

// TestJSONEncodeNumberWithIntValue verifies that JNumber with IntValue
// (as produced by WASM bridge round-trip: JS number → jsToAILANGValue → IntValue)
// encodes correctly instead of returning "JNumber field expected FloatValue".
func TestJSONEncodeNumberWithIntValue(t *testing.T) {
	ctx := testctx.NewMockEffContext()

	tests := []struct {
		name     string
		input    int
		expected string
	}{
		{"positive", 42, "42"},
		{"negative", -17, "-17"},
		{"zero", 0, "0"},
		{"large", 1000000, "1000000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create JNumber with IntValue (as WASM bridge produces)
			jnum := &eval.TaggedValue{
				ModulePath: "std/json",
				TypeName:   "Json",
				CtorName:   "JNumber",
				Fields:     []eval.Value{&eval.IntValue{Value: tt.input}},
			}
			result, err := jsonEncodeImpl(ctx.EffContext, []eval.Value{jnum})

			require.NoError(t, err)
			assert.Equal(t, tt.expected, testctx.GetString(result))
		})
	}
}

func TestJSONEncodeString(t *testing.T) {
	ctx := testctx.NewMockEffContext()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple", "hello", `"hello"`},
		{"empty", "", `""`},
		{"with space", "hello world", `"hello world"`},
		{"with unicode", "hello 世界", `"hello 世界"`},
		{"with emoji", "🎉🎊", `"🎉🎊"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jstr := makeJString(tt.input)
			result, err := jsonEncodeImpl(ctx.EffContext, []eval.Value{jstr})

			require.NoError(t, err)
			assert.Equal(t, tt.expected, testctx.GetString(result))
		})
	}
}

func TestJSONEncodeStringEscaping(t *testing.T) {
	ctx := testctx.NewMockEffContext()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"quote", `hello"world`, `"hello\"world"`},
		{"backslash", `hello\world`, `"hello\\world"`},
		{"newline", "hello\nworld", `"hello\nworld"`},
		{"tab", "hello\tworld", `"hello\tworld"`},
		{"carriage return", "hello\rworld", `"hello\rworld"`},
		{"backspace", "hello\bworld", `"hello\bworld"`},
		{"form feed", "hello\fworld", `"hello\fworld"`},
		{"control char", "hello\x01world", `"hello\u0001world"`},
		{"multiple escapes", "\"hello\"\n\t", `"\"hello\"\n\t"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jstr := makeJString(tt.input)
			result, err := jsonEncodeImpl(ctx.EffContext, []eval.Value{jstr})

			require.NoError(t, err)
			assert.Equal(t, tt.expected, testctx.GetString(result))
		})
	}
}

func TestJSONEncodeArray(t *testing.T) {
	ctx := testctx.NewMockEffContext()

	tests := []struct {
		name     string
		input    []eval.Value
		expected string
	}{
		{"empty", []eval.Value{}, "[]"},
		{"single number", []eval.Value{makeJNumberFloat(42.0)}, "[42]"},
		{"multiple numbers", []eval.Value{makeJNumberFloat(1.0), makeJNumberFloat(2.0), makeJNumberFloat(3.0)}, "[1,2,3]"},
		{"mixed types", []eval.Value{makeJNumberFloat(42.0), makeJString("hello"), makeJBool(true), makeJNull()}, `[42,"hello",true,null]`},
		{"nested array", []eval.Value{makeJArray([]eval.Value{makeJNumberFloat(1.0), makeJNumberFloat(2.0)})}, "[[1,2]]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jarr := makeJArray(tt.input)
			result, err := jsonEncodeImpl(ctx.EffContext, []eval.Value{jarr})

			require.NoError(t, err)
			assert.Equal(t, tt.expected, testctx.GetString(result))
		})
	}
}

func TestJSONEncodeObject(t *testing.T) {
	ctx := testctx.NewMockEffContext()

	tests := []struct {
		name     string
		input    []eval.Value
		expected string
	}{
		{"empty", []eval.Value{}, "{}"},
		{"single key", []eval.Value{makeKV("name", makeJString("Alice"))}, `{"name":"Alice"}`},
		{"multiple keys", []eval.Value{
			makeKV("name", makeJString("Alice")),
			makeKV("age", makeJNumberFloat(30.0)),
		}, `{"name":"Alice","age":30}`},
		{"nested object", []eval.Value{
			makeKV("person", makeJObject([]eval.Value{
				makeKV("name", makeJString("Bob")),
			})),
		}, `{"person":{"name":"Bob"}}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jobj := makeJObject(tt.input)
			result, err := jsonEncodeImpl(ctx.EffContext, []eval.Value{jobj})

			require.NoError(t, err)
			assert.Equal(t, tt.expected, testctx.GetString(result))
		})
	}
}

// Roundtrip tests - decode(encode(x)) == Ok(x)

func TestRoundtripNull(t *testing.T) {
	ctx := testctx.NewMockEffContext()

	original := makeJNull()

	// Encode
	encoded, err := jsonEncodeImpl(ctx.EffContext, []eval.Value{original})
	require.NoError(t, err)

	// Decode
	decoded, err := jsonDecodeImpl(ctx.EffContext, []eval.Value{encoded})
	require.NoError(t, err)

	// Verify it's Ok(Json)
	result := extractOk(decoded)

	// Compare structure (both should be JNull)
	origTagged := original.(*eval.TaggedValue)
	resultTagged := result.(*eval.TaggedValue)
	assert.Equal(t, origTagged.CtorName, resultTagged.CtorName)
}

func TestRoundtripNumber(t *testing.T) {
	ctx := testctx.NewMockEffContext()

	tests := []float64{42.0, 3.14, -17.5, 0.0}

	for _, num := range tests {
		t.Run("", func(t *testing.T) {
			original := makeJNumberFloat(num)

			// Encode
			encoded, err := jsonEncodeImpl(ctx.EffContext, []eval.Value{original})
			require.NoError(t, err)

			// Decode
			decoded, err := jsonDecodeImpl(ctx.EffContext, []eval.Value{encoded})
			require.NoError(t, err)

			result := extractOk(decoded)

			// Extract the number value
			resultTagged := result.(*eval.TaggedValue)
			assert.Equal(t, "JNumber", resultTagged.CtorName)
			resultFloat := resultTagged.Fields[0].(*eval.FloatValue).Value
			assert.Equal(t, num, resultFloat)
		})
	}
}

func TestRoundtripString(t *testing.T) {
	ctx := testctx.NewMockEffContext()

	tests := []string{"hello", "hello\nworld", `quote"test`, "emoji🎉"}

	for _, str := range tests {
		t.Run("", func(t *testing.T) {
			original := makeJString(str)

			// Encode
			encoded, err := jsonEncodeImpl(ctx.EffContext, []eval.Value{original})
			require.NoError(t, err)

			// Decode
			decoded, err := jsonDecodeImpl(ctx.EffContext, []eval.Value{encoded})
			require.NoError(t, err)

			result := extractOk(decoded)

			// Extract the string value
			resultTagged := result.(*eval.TaggedValue)
			assert.Equal(t, "JString", resultTagged.CtorName)
			resultStr := resultTagged.Fields[0].(*eval.StringValue).Value
			assert.Equal(t, str, resultStr)
		})
	}
}

func TestRoundtripArray(t *testing.T) {
	ctx := testctx.NewMockEffContext()

	original := makeJArray([]eval.Value{
		makeJNumberFloat(1.0),
		makeJString("two"),
		makeJBool(true),
	})

	// Encode
	encoded, err := jsonEncodeImpl(ctx.EffContext, []eval.Value{original})
	require.NoError(t, err)

	// Decode
	decoded, err := jsonDecodeImpl(ctx.EffContext, []eval.Value{encoded})
	require.NoError(t, err)

	result := extractOk(decoded)

	// Verify structure
	resultTagged := result.(*eval.TaggedValue)
	assert.Equal(t, "JArray", resultTagged.CtorName)
	resultList := resultTagged.Fields[0].(*eval.ListValue)
	assert.Equal(t, 3, len(resultList.Elements))
}

func TestRoundtripObject(t *testing.T) {
	ctx := testctx.NewMockEffContext()

	original := makeJObject([]eval.Value{
		makeKV("name", makeJString("Alice")),
		makeKV("age", makeJNumberFloat(30.0)),
		makeKV("active", makeJBool(true)),
	})

	// Encode
	encoded, err := jsonEncodeImpl(ctx.EffContext, []eval.Value{original})
	require.NoError(t, err)
	jsonStr := testctx.GetString(encoded)

	// Decode
	decoded, err := jsonDecodeImpl(ctx.EffContext, []eval.Value{encoded})
	require.NoError(t, err)

	result := extractOk(decoded)

	// Parse both as Go maps to compare structure (order-independent)
	var originalMap, resultMap map[string]interface{}
	err = json.Unmarshal([]byte(jsonStr), &originalMap)
	require.NoError(t, err)

	// Re-encode the decoded value to compare
	reencoded, err := jsonEncodeImpl(ctx.EffContext, []eval.Value{result})
	require.NoError(t, err)
	err = json.Unmarshal([]byte(testctx.GetString(reencoded)), &resultMap)
	require.NoError(t, err)

	assert.Equal(t, originalMap, resultMap)
}

// Helper functions to construct Json ADT values
// Note: makeJNull, makeJBool, makeJString are already defined in json_decode.go

func makeJNumberFloat(f float64) eval.Value {
	return &eval.TaggedValue{
		ModulePath: "std/json",
		TypeName:   "Json",
		CtorName:   "JNumber",
		Fields:     []eval.Value{&eval.FloatValue{Value: f}},
	}
}

func makeJArray(elements []eval.Value) eval.Value {
	return &eval.TaggedValue{
		ModulePath: "std/json",
		TypeName:   "Json",
		CtorName:   "JArray",
		Fields:     []eval.Value{&eval.ListValue{Elements: elements}},
	}
}

func makeJObject(kvPairs []eval.Value) eval.Value {
	return &eval.TaggedValue{
		ModulePath: "std/json",
		TypeName:   "Json",
		CtorName:   "JObject",
		Fields:     []eval.Value{&eval.ListValue{Elements: kvPairs}},
	}
}

func makeKV(key string, value eval.Value) eval.Value {
	return &eval.RecordValue{
		Fields: map[string]eval.Value{
			"key":   &eval.StringValue{Value: key},
			"value": value,
		},
	}
}
