package builtins

import (
	"encoding/json"
	"gopkg.in/yaml.v3"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/sunholo-data/ailang/internal/effects/testctx"
	"github.com/sunholo-data/ailang/internal/eval"
)

// callYAMLToJSON invokes the builtin with a string and returns the result value.
func callYAMLToJSON(t *testing.T, input string) eval.Value {
	t.Helper()
	ctx := testctx.NewMockEffContext()
	result, err := yamlToJSONImpl(ctx.EffContext, []eval.Value{
		&eval.StringValue{Value: input},
	})
	require.NoError(t, err)
	return result
}

// expectOk asserts the result is Ok(string) and returns the wrapped JSON string.
func expectYAMLOk(t *testing.T, result eval.Value) string {
	t.Helper()
	tv, ok := result.(*eval.TaggedValue)
	require.True(t, ok, "expected TaggedValue, got %T", result)
	require.Equal(t, "Ok", tv.CtorName, "expected Ok, got %s", tv.CtorName)
	require.Len(t, tv.Fields, 1)
	sv, ok := tv.Fields[0].(*eval.StringValue)
	require.True(t, ok, "expected StringValue inside Ok, got %T", tv.Fields[0])
	return sv.Value
}

// expectErr asserts the result is Err(string) and returns the message.
func expectYAMLErr(t *testing.T, result eval.Value) string {
	t.Helper()
	tv, ok := result.(*eval.TaggedValue)
	require.True(t, ok, "expected TaggedValue, got %T", result)
	require.Equal(t, "Err", tv.CtorName, "expected Err, got %s", tv.CtorName)
	require.Len(t, tv.Fields, 1)
	sv, ok := tv.Fields[0].(*eval.StringValue)
	require.True(t, ok, "expected StringValue inside Err, got %T", tv.Fields[0])
	return sv.Value
}

func TestYAMLToJSON_Scalars(t *testing.T) {
	cases := []struct {
		name string
		yaml string
		want string
	}{
		{"string", "hello\n", `"hello"`},
		{"int", "42\n", `42`},
		{"float", "1.5\n", `1.5`},
		{"bool_true", "true\n", `true`},
		{"bool_false", "false\n", `false`},
		{"null_tilde", "~\n", `null`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := expectYAMLOk(t, callYAMLToJSON(t, tc.yaml))
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestYAMLToJSON_Empty(t *testing.T) {
	// Empty input decodes to nil, which marshals to the JSON null literal.
	got := expectYAMLOk(t, callYAMLToJSON(t, ""))
	assert.Equal(t, "null", got)
}

func TestYAMLToJSON_NullValue(t *testing.T) {
	got := expectYAMLOk(t, callYAMLToJSON(t, "x:\n"))
	assert.Equal(t, `{"x":null}`, got)
}

func TestYAMLToJSON_BlockMapping(t *testing.T) {
	// Mapping pairs retain document order, including nested mappings.
	in := "name: STX\ncount: 3\nnested:\n  x: 1.5\n  ok: true\n"
	want := `{"name":"STX","count":3,"nested":{"x":1.5,"ok":true}}`
	got := expectYAMLOk(t, callYAMLToJSON(t, in))
	assert.Equal(t, want, got)
}

func TestYAMLToJSON_Sequence(t *testing.T) {
	got := expectYAMLOk(t, callYAMLToJSON(t, "items:\n  - a\n  - b\n"))
	assert.Equal(t, `{"items":["a","b"]}`, got)
}

func TestYAMLToJSON_FlowStyleEqualsBlock(t *testing.T) {
	block := expectYAMLOk(t, callYAMLToJSON(t, "a: 1\nb:\n  - x\n  - y\n"))
	flow := expectYAMLOk(t, callYAMLToJSON(t, "{a: 1, b: [x, y]}\n"))
	assert.Equal(t, block, flow)
	assert.Equal(t, `{"a":1,"b":["x","y"]}`, block)
}

func TestYAMLToJSON_AnchorsResolved(t *testing.T) {
	// Anchors/aliases are resolved during decode (they work), but not preserved.
	got := expectYAMLOk(t, callYAMLToJSON(t, "a: &x 1\nb: *x\n"))
	assert.Equal(t, `{"a":1,"b":1}`, got)
}

func TestYAMLToJSON_MultiDocReadsFirstOnly(t *testing.T) {
	// Documented single-document behavior: only the first document is read.
	got := expectYAMLOk(t, callYAMLToJSON(t, "a: 1\n---\nb: 2\n"))
	assert.Equal(t, `{"a":1}`, got)
}

func TestYAMLToJSON_NonStringKeyIsErr(t *testing.T) {
	// A non-string map key has no JSON representation → loud failure, no coercion.
	msg := expectYAMLErr(t, callYAMLToJSON(t, "1: a\n2: b\n"))
	assert.Contains(t, msg, "yaml:")
}

func TestYAMLToJSON_NaNIsErr(t *testing.T) {
	msg := expectYAMLErr(t, callYAMLToJSON(t, "x: .nan\n"))
	assert.Contains(t, msg, "yaml:")
}

func TestYAMLToJSON_InfIsErr(t *testing.T) {
	msg := expectYAMLErr(t, callYAMLToJSON(t, "x: .inf\n"))
	assert.Contains(t, msg, "yaml:")
}

func TestYAMLToJSON_MalformedIndentationIsErr(t *testing.T) {
	// Bad indentation is a YAML parse error.
	msg := expectYAMLErr(t, callYAMLToJSON(t, "a: 1\n  b: 2\n"))
	assert.Contains(t, msg, "yaml:")
}

func TestYAMLToJSON_NonStringArgErrors(t *testing.T) {
	ctx := testctx.NewMockEffContext()
	_, err := yamlToJSONImpl(ctx.EffContext, []eval.Value{&eval.IntValue{Value: 5}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expected string")
}

// TestYAMLToJSON_Deterministic guards against Go map iteration nondeterminism:
// object key ordering must be stable across many passes (the Node tree preserves document order).
func TestYAMLToJSON_Deterministic(t *testing.T) {
	in := "z: 1\na: 2\nm: 3\nnested:\n  q: 4\n  b: 5\n"
	want := expectYAMLOk(t, callYAMLToJSON(t, in))
	for i := 0; i < 100; i++ {
		got := expectYAMLOk(t, callYAMLToJSON(t, in))
		require.Equal(t, want, got, "output changed on pass %d", i)
	}
}

func TestYAMLToJSON_DocumentOrder(t *testing.T) {
	for _, input := range []string{"b: 1\na: 2\nc: 3\n", "{b: 1, a: 2, c: 3}"} {
		require.Equal(t, `{"b":1,"a":2,"c":3}`, expectYAMLOk(t, callYAMLToJSON(t, input)))
	}
	require.Equal(t, `{"z":[{"b":1,"a":2}],"a":{"c":3,"b":2}}`, expectYAMLOk(t, callYAMLToJSON(t, "z: [{b: 1, a: 2}]\na: {c: 3, b: 2}")))
}

func TestYAMLToJSON_MergesAndAliases(t *testing.T) {
	cases := []struct{ input, want string }{
		{"base: &b {z: 1, a: 2}\ncopy: *b", `{"base":{"z":1,"a":2},"copy":{"z":1,"a":2}}`},
		{"base: &b [x, y]\ncopy: *b", `{"base":["x","y"],"copy":["x","y"]}`},
		{"base: &b {z: 1, a: 2}\nchild: {c: 3, <<: *b, a: 4}", `{"base":{"z":1,"a":2},"child":{"c":3,"z":1,"a":4}}`},
		{"base: &b {z: 1, a: 2}\nchild: {a: 4, <<: *b, c: 3}", `{"base":{"z":1,"a":2},"child":{"a":4,"z":1,"c":3}}`},
		{"a: &a {z: 1, b: 2}\nb: &b {b: 3, c: 4}\nx: {<<: [*a, *b]}", `{"a":{"z":1,"b":2},"b":{"b":3,"c":4},"x":{"z":1,"b":2,"c":4}}`},
		{"a: &a {z: 1, b: 2}\nb: &b {c: 3, <<: *a}\nx: {<<: *b, d: 4}", `{"a":{"z":1,"b":2},"b":{"c":3,"z":1,"b":2},"x":{"c":3,"z":1,"b":2,"d":4}}`},
		{"'<<': literal\n'1': a", `{"\u003c\u003c":"literal","1":"a"}`},
	}
	for _, tc := range cases {
		require.Equal(t, tc.want, expectYAMLOk(t, callYAMLToJSON(t, tc.input)))
	}
}

// Compare scalar bytes with the shipped generic decoder, without involving order.
func TestYAMLToJSON_ScalarCompatibility(t *testing.T) {
	for _, input := range []string{
		"\n", "{}", "[]", "!!str 1", "!!int '5'", "2026-10-07",
		"9007199254740993", "0x1F", "0o17", "1:30", "|\n  line1\n  line2\n",
		"'<>&\"'", "{'<>&\"': '<>&'}", "!custom text",
	} {
		t.Run(input, func(t *testing.T) {
			var old interface{}
			require.NoError(t, yaml.Unmarshal([]byte(input), &old))
			want, err := json.Marshal(old)
			require.NoError(t, err)
			require.Equal(t, string(want), expectYAMLOk(t, callYAMLToJSON(t, input)))
		})
	}
}

func TestYAMLToJSON_ValidationCompatibility(t *testing.T) {
	for _, input := range []string{
		"a: 1\na: 2", "<<: {}\n<<: {}", "true: x", "~: x", "? [a, b]\n: x",
		"x: -.inf", "x: .nan", "x: .inf", "a: &a {b: *a}", "a: &a [*a]",
		"<<: 1", "<<: [1]", "<<: [ {} , [] ]", "a: !!int nope",
		"base: &b {1: x}\nx: {<<: *b}",
	} {
		t.Run(input, func(t *testing.T) {
			var old interface{}
			err := yaml.Unmarshal([]byte(input), &old)
			if err == nil {
				_, err = json.Marshal(old)
			}
			require.Error(t, err, "baseline must reject fixture")
			require.Contains(t, expectYAMLErr(t, callYAMLToJSON(t, input)), "yaml:")
		})
	}
	require.Equal(t, "yaml: yaml: map merge requires map or sequence of maps as the value", expectYAMLErr(t, callYAMLToJSON(t, "<<: 1")))
}

func TestYAMLNodeMechanics(t *testing.T) {
	var node yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte("base: &b {z: 1}\nchild: {<<: *b}\n---\nignored: 2"), &node))
	root := node.Content[0]
	require.Len(t, root.Content, 4)
	merge := root.Content[3]
	require.Equal(t, "!!merge", merge.Content[0].Tag)
	require.Equal(t, yaml.AliasNode, merge.Content[1].Kind)
	require.Same(t, root.Content[1], merge.Content[1].Alias)
}
