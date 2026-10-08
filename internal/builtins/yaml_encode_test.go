package builtins

import (
	"math"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/sunholo-data/ailang/internal/eval"
)

func callYAMLEncode(t *testing.T, v eval.Value) eval.Value {
	t.Helper()
	result, err := yamlEncodeImpl(nil, []eval.Value{v})
	require.NoError(t, err)
	return result
}

func TestYAMLEncodeFixtures(t *testing.T) {
	cases := []struct {
		name  string
		value eval.Value
		want  string
	}{
		{"null", makeJNull(), "null\n"},
		{"true", makeJBool(true), "true\n"},
		{"false", makeJBool(false), "false\n"},
		{"empty string", makeJString(""), "\"\"\n"},
		{"empty array", makeJArray(nil), "[]\n"},
		{"empty object", makeJObject(nil), "{}\n"},
		{"escapes", makeJString("a\n\"\\\b\f\r\t\x00"), "\"a\\n\\\"\\\\\\b\\f\\r\\t\\u0000\"\n"},
		{"unicode", makeJString("Fysik 日本 🎉"), "\"Fysik 日本 🎉\"\n"},
		{"ordered", makeJObject([]eval.Value{makeKV("z", makeJNull()), makeKV("a", makeJBool(true))}), "\"z\": null\n\"a\": true\n"},
		{"nested", makeJObject([]eval.Value{makeKV("items", makeJArray([]eval.Value{
			makeJObject([]eval.Value{makeKV("x", makeJNumberFloat(1)), makeKV("y", makeJArray([]eval.Value{makeJArray([]eval.Value{makeJString("v")}), makeJObject(nil)}))}),
			makeJArray([]eval.Value{makeJBool(false), makeJNull()}),
		}))}), "\"items\":\n  - \"x\": 1\n    \"y\":\n      - - \"v\"\n      - {}\n  - - false\n    - null\n"},
		{"map in map", makeJObject([]eval.Value{makeKV("", makeJObject([]eval.Value{makeKV("a: b\n#", makeJArray(nil))}))}), "\"\":\n  \"a: b\\n#\": []\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			text := expectYAMLOk(t, callYAMLEncode(t, tc.value))
			require.Equal(t, tc.want, text)
			decoded := expectYAMLOk(t, callYAMLToJSON(t, text))
			// JSONEq compares recursively without assuming the decoder preserves object order.
			encoded, err := jsonEncodeImpl(nil, []eval.Value{tc.value})
			require.NoError(t, err)
			require.JSONEq(t, encoded.(*eval.StringValue).Value, decoded)
			for i := 0; i < 25; i++ {
				require.Equal(t, text, expectYAMLOk(t, callYAMLEncode(t, tc.value)))
			}
		})
	}
}

func TestYAMLEncodeStrings(t *testing.T) {
	for _, s := range []string{"true", "42", "null", "a: b", "#x", " leading space", "---", "yes", "2026-10-07", "\x7f", "\u0085", "\u2028", "\u2029", "\ufffe", "\uffff"} {
		t.Run(strconv.Quote(s), func(t *testing.T) {
			text := expectYAMLOk(t, callYAMLEncode(t, makeJString(s)))
			decoded := expectYAMLOk(t, callYAMLToJSON(t, text))
			result, err := jsonDecodeImpl(nil, []eval.Value{&eval.StringValue{Value: decoded}})
			require.NoError(t, err)
			require.Equal(t, s, result.(*eval.TaggedValue).Fields[0].(*eval.TaggedValue).Fields[0].(*eval.StringValue).Value)
		})
	}
}

func TestYAMLEncodeNumbers(t *testing.T) {
	for _, f := range []float64{0, math.Copysign(0, -1), 42, 42.5, 1e-6, 1e-7, 1e20, 1e21, math.SmallestNonzeroFloat64, math.MaxFloat64, -math.MaxFloat64} {
		t.Run(strconv.FormatFloat(f, 'g', -1, 64), func(t *testing.T) {
			text := expectYAMLOk(t, callYAMLEncode(t, makeJNumberFloat(f)))
			require.Equal(t, eval.FormatJSONNumber(f)+"\n", text)
			decoded := expectYAMLOk(t, callYAMLToJSON(t, text))
			result, err := jsonDecodeImpl(nil, []eval.Value{&eval.StringValue{Value: decoded}})
			require.NoError(t, err)
			got := result.(*eval.TaggedValue).Fields[0].(*eval.TaggedValue).Fields[0].(*eval.FloatValue).Value
			require.Equal(t, f, got)
			require.Equal(t, math.Signbit(f), math.Signbit(got))
		})
	}
	// Portable large integer: no narrowing through float64 during emission.
	if strconv.IntSize == 64 {
		n, err := strconv.Atoi("9007199254740993")
		require.NoError(t, err)
		v := &eval.TaggedValue{TypeName: "Json", CtorName: "JNumber", Fields: []eval.Value{&eval.IntValue{Value: n}}}
		text := expectYAMLOk(t, callYAMLEncode(t, v))
		require.Equal(t, "9007199254740993\n", text)
		require.Equal(t, "9007199254740993", expectYAMLOk(t, callYAMLToJSON(t, text)))
	}
}

func TestYAMLEncodeErrorsAndDuplicates(t *testing.T) {
	for _, f := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		for _, v := range []eval.Value{makeJNumberFloat(f), makeJArray([]eval.Value{makeJNumberFloat(f)}), makeJObject([]eval.Value{makeKV("bad", makeJNumberFloat(f))})} {
			require.Contains(t, expectYAMLErr(t, callYAMLEncode(t, v)), "non-finite")
		}
	}
	v := makeJObject([]eval.Value{makeKV("dup", makeJNull()), makeKV("dup", makeJBool(true))})
	text := expectYAMLOk(t, callYAMLEncode(t, v))
	require.Equal(t, "\"dup\": null\n\"dup\": true\n", text)
	require.Contains(t, expectYAMLErr(t, callYAMLToJSON(t, text)), "dup")
}

func TestYAMLEncodeInvalidADT(t *testing.T) {
	cases := []eval.Value{
		&eval.StringValue{Value: "not Json"},
		&eval.TaggedValue{TypeName: "Other", CtorName: "JNull"},
		&eval.TaggedValue{TypeName: "Json", CtorName: "JArray"},
		&eval.TaggedValue{TypeName: "Json", CtorName: "JObject", Fields: []eval.Value{&eval.StringValue{}}},
		makeJObject([]eval.Value{makeJNull()}),
		makeJObject([]eval.Value{&eval.RecordValue{Fields: map[string]eval.Value{"key": &eval.IntValue{Value: 1}}}}),
		makeJObject([]eval.Value{&eval.RecordValue{Fields: map[string]eval.Value{"key": &eval.StringValue{Value: "x"}}}}),
		&eval.TaggedValue{TypeName: "Json", CtorName: "JString", Fields: []eval.Value{&eval.IntValue{}}},
		&eval.TaggedValue{TypeName: "Json", CtorName: "Unknown"},
	}
	for _, v := range cases {
		require.Contains(t, expectYAMLErr(t, callYAMLEncode(t, v)), "yaml encode:")
	}
}

func TestYAMLEncodeRegistration(t *testing.T) {
	spec, ok := GetSpec("_yaml_encode")
	require.True(t, ok)
	require.True(t, spec.IsPure)
	require.Empty(t, spec.Effect)
	require.Equal(t, "std/yaml", spec.Module)
	require.Equal(t, 1, spec.NumArgs)
	require.Equal(t, "Json -> Result[string, string]", spec.Type().String())
}

func TestYAMLEncodeControlKeys(t *testing.T) {
	for r := rune(0); r <= 0x9f; r++ {
		s := string(r)
		v := makeJObject([]eval.Value{makeKV(s, makeJString(s))})
		text := expectYAMLOk(t, callYAMLEncode(t, v))
		jsonText := expectYAMLOk(t, callYAMLToJSON(t, text))
		encoded, err := jsonEncodeImpl(nil, []eval.Value{v})
		require.NoError(t, err)
		require.JSONEq(t, encoded.(*eval.StringValue).Value, jsonText, "codepoint U+%04X", r)
	}
}

func TestYAMLEncodeLongKeys(t *testing.T) {
	for _, key := range []string{strings.Repeat("x", 1022), strings.Repeat("x", 1024), strings.Repeat("\n", 200), strings.Repeat("日本", 600)} {
		v := makeJObject([]eval.Value{makeKV(key, makeJArray([]eval.Value{makeJNull()})), makeKV("z", makeJBool(true))})
		for _, value := range []eval.Value{v, makeJArray([]eval.Value{v})} {
			text := expectYAMLOk(t, callYAMLEncode(t, value))
			result := expectYAMLOk(t, callYAMLToJSON(t, text))
			expected, err := jsonEncodeImpl(nil, []eval.Value{value})
			require.NoError(t, err)
			require.JSONEq(t, expected.(*eval.StringValue).Value, result)
		}
	}
}
