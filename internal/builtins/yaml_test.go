package builtins

import (
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
	// encoding/json sorts object keys, so the output is deterministic.
	in := "name: STX\ncount: 3\nnested:\n  x: 1.5\n  ok: true\n"
	want := `{"count":3,"name":"STX","nested":{"ok":true,"x":1.5}}`
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
// object key ordering must be stable across many passes (encoding/json sorts keys).
func TestYAMLToJSON_Deterministic(t *testing.T) {
	in := "z: 1\na: 2\nm: 3\nnested:\n  q: 4\n  b: 5\n"
	want := expectYAMLOk(t, callYAMLToJSON(t, in))
	for i := 0; i < 100; i++ {
		got := expectYAMLOk(t, callYAMLToJSON(t, in))
		require.Equal(t, want, got, "output changed on pass %d", i)
	}
}
