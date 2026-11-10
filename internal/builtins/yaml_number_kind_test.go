package builtins

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/sunholo-data/ailang/internal/eval"
)

// YAML has its own int/float distinction (the resolved !!int / !!float tag).
// The bridge must carry it into the JSON text, so std/json gives JInt for a
// YAML integer and JNumber for a YAML float — never "is the float whole".
func TestYAMLToJSONKeepsFloatSyntax(t *testing.T) {
	cases := map[string]string{
		"1.0":                  "1.0",
		"1e3":                  "1000.0",
		"1.0e+3":               "1000.0",
		"2.50":                 "2.5",
		"-0.0":                 "-0.0",
		"1e21":                 "1e+21",
		"!!float 3":            "3.0",
		"1":                    "1",
		"0x10":                 "16",
		"0o17":                 "15",
		"9007199254740993":     "9007199254740993",
		"12345678901234567890": "12345678901234567890",
		"!!int \"7\"":          "7",
		"[1, 1.0]":             "[1,1.0]",
		"a: 1.0\nb: 2":         `{"a":1.0,"b":2}`,
	}
	for in, want := range cases {
		require.Equal(t, want, expectYAMLOk(t, callYAMLToJSON(t, in)), "yamlToJson(%q)", in)
	}
}

// std/yaml.decode is yamlToJson + std/json.decode; check the constructor.
func TestYAMLDecodeNumberConstructors(t *testing.T) {
	for in, want := range map[string]string{
		"1.0": "JNumber", "1e3": "JNumber", "1.0e+3": "JNumber", "!!float 3": "JNumber",
		"1": "JInt", "0x10": "JInt", "1_000": "JInt", "9007199254740993": "JInt",
		"12345678901234567890": "JNumber",
	} {
		text := expectYAMLOk(t, callYAMLToJSON(t, in))
		res, err := jsonDecodeImpl(nil, []eval.Value{&eval.StringValue{Value: text}})
		require.NoError(t, err)
		got := extractOk(res)
		require.NotNil(t, got, "decode(%q)", text)
		require.Equal(t, want, got.(*eval.TaggedValue).CtorName, "yaml %q -> json %q", in, text)
	}
}

// Json -> YAML -> Json keeps the constructor: a whole-number JNumber is
// written with float syntax so YAML reads it back as a float.
func TestYAMLEncodeRoundTripKeepsConstructor(t *testing.T) {
	jint := &eval.TaggedValue{TypeName: "Json", CtorName: "JInt", Fields: []eval.Value{&eval.IntValue{Value: 3}}}
	for _, v := range []eval.Value{makeJNumberFloat(1), makeJNumberFloat(1e20), makeJNumberFloat(math.Copysign(0, -1)), makeJNumberFloat(2.5), jint} {
		text := expectYAMLOk(t, callYAMLEncode(t, v))
		back := expectYAMLOk(t, callYAMLToJSON(t, text))
		res, err := jsonDecodeImpl(nil, []eval.Value{&eval.StringValue{Value: back}})
		require.NoError(t, err)
		require.Equal(t, v.(*eval.TaggedValue).CtorName, extractOk(res).(*eval.TaggedValue).CtorName, "yaml %q -> json %q", text, back)
	}
}
