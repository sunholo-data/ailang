package builtins

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/sunholo-data/ailang/internal/eval"
)

// callURLEncode is a tiny test helper that drives netURLEncodeImpl with a string arg.
func callURLEncode(t *testing.T, s string) string {
	t.Helper()
	v, err := netURLEncodeImpl(nil, []eval.Value{&eval.StringValue{Value: s}})
	require.NoError(t, err)
	str, ok := v.(*eval.StringValue)
	require.True(t, ok, "_net_url_encode should return StringValue")
	return str.Value
}

// callURLEncodeForm drives netURLEncodeFormImpl with a slice of (name, value) pairs.
func callURLEncodeForm(t *testing.T, pairs ...[2]string) string {
	t.Helper()
	elems := make([]eval.Value, 0, len(pairs))
	for _, p := range pairs {
		elems = append(elems, &eval.RecordValue{
			Fields: map[string]eval.Value{
				"name":  &eval.StringValue{Value: p[0]},
				"value": &eval.StringValue{Value: p[1]},
			},
		})
	}
	v, err := netURLEncodeFormImpl(nil, []eval.Value{&eval.ListValue{Elements: elems}})
	require.NoError(t, err)
	str, ok := v.(*eval.StringValue)
	require.True(t, ok, "_net_url_encode_form should return StringValue")
	return str.Value
}

func TestNetURLEncode_OAuthEdgeCases(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{"equals", "=", "%3D"},
		{"plus", "+", "%2B"},
		{"slash", "/", "%2F"},
		{"ampersand", "&", "%26"},
		{"space_to_percent20", " ", "%20"},
		{"percent_first", "%", "%25"},
		{"unicode_cafe", "café", "caf%C3%A9"},
		{"empty", "", ""},
		{"safe_chars_passthrough", "abc123-_.~", "abc123-_.~"},
		{"oauth_secret_with_unsafe", "a/b=c+d&e", "a%2Fb%3Dc%2Bd%26e"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := callURLEncode(t, tc.input)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestNetURLEncode_RejectsNonString(t *testing.T) {
	_, err := netURLEncodeImpl(nil, []eval.Value{&eval.IntValue{Value: 42}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expected String")
}

func TestNetURLEncodeForm_EmptyList(t *testing.T) {
	got := callURLEncodeForm(t)
	assert.Equal(t, "", got)
}

func TestNetURLEncodeForm_SinglePair(t *testing.T) {
	got := callURLEncodeForm(t, [2]string{"k", "v"})
	assert.Equal(t, "k=v", got)
}

func TestNetURLEncodeForm_SpaceEncodedAsPlus(t *testing.T) {
	// Form spec: space -> +. This is the key asymmetry vs _net_url_encode.
	got := callURLEncodeForm(t, [2]string{"x", " "})
	assert.Equal(t, "x=+", got)
}

func TestNetURLEncodeForm_UnsafeValue(t *testing.T) {
	// Round-trip assertion from the design doc: value with =, &, / encodes correctly.
	got := callURLEncodeForm(t, [2]string{"k", "a=b&c"})
	assert.Equal(t, "k=a%3Db%26c", got)
}

func TestNetURLEncodeForm_OAuthLinkedInPayload(t *testing.T) {
	// Regression test for feedback msg 9e25539f (demos/linkedin OAuth token exchange).
	// client_secret contains URL-unsafe chars that the hand-rolled encoder had to
	// handle in a specific order.
	got := callURLEncodeForm(t,
		[2]string{"grant_type", "authorization_code"},
		[2]string{"code", "ABC=123/xyz+abc"},
		[2]string{"client_id", "linkedin_id"},
		[2]string{"client_secret", "a/b=c+d&e%f"},
		[2]string{"redirect_uri", "https://example.com/cb?x=1&y=2"},
	)
	// url.Values.Encode sorts alphabetically by key, so verify exact bytes.
	want := "client_id=linkedin_id" +
		"&client_secret=a%2Fb%3Dc%2Bd%26e%25f" +
		"&code=ABC%3D123%2Fxyz%2Babc" +
		"&grant_type=authorization_code" +
		"&redirect_uri=https%3A%2F%2Fexample.com%2Fcb%3Fx%3D1%26y%3D2"
	assert.Equal(t, want, got)
}

func TestNetURLEncodeForm_UTF8KeysAndValues(t *testing.T) {
	got := callURLEncodeForm(t, [2]string{"naïve", "café"})
	assert.Equal(t, "na%C3%AFve=caf%C3%A9", got)
}

func TestNetURLEncodeForm_EmptyValue(t *testing.T) {
	got := callURLEncodeForm(t, [2]string{"k", ""})
	assert.Equal(t, "k=", got)
}

// TestNetURLEncodeForm_Deterministic guards against Go map iteration order leaking
// into the output. url.Values.Encode sorts keys, but we want this checked under
// repeated runs (pure builtin contract). 20 iterations matches the project
// convention for verifying determinism (see milestone_checklist.md).
func TestNetURLEncodeForm_Deterministic(t *testing.T) {
	var first string
	for i := 0; i < 20; i++ {
		got := callURLEncodeForm(t,
			[2]string{"zeta", "1"},
			[2]string{"alpha", "2"},
			[2]string{"mu", "3"},
			[2]string{"beta", "4"},
		)
		if i == 0 {
			first = got
			continue
		}
		assert.Equal(t, first, got, "iteration %d diverged — non-deterministic output", i)
	}
	// Sanity: alphabetic sort applied.
	assert.Equal(t, "alpha=2&beta=4&mu=3&zeta=1", first)
}

func TestNetURLEncodeForm_RejectsNonRecord(t *testing.T) {
	_, err := netURLEncodeFormImpl(nil, []eval.Value{
		&eval.ListValue{Elements: []eval.Value{&eval.StringValue{Value: "not-a-record"}}},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "index 0")
	assert.Contains(t, err.Error(), "record")
}

func TestNetURLEncodeForm_RejectsMissingNameField(t *testing.T) {
	_, err := netURLEncodeFormImpl(nil, []eval.Value{
		&eval.ListValue{Elements: []eval.Value{
			&eval.RecordValue{Fields: map[string]eval.Value{
				"value": &eval.StringValue{Value: "v"},
			}},
		}},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing 'name'")
}

func TestNetURLEncodeForm_RejectsNonStringValue(t *testing.T) {
	_, err := netURLEncodeFormImpl(nil, []eval.Value{
		&eval.ListValue{Elements: []eval.Value{
			&eval.RecordValue{Fields: map[string]eval.Value{
				"name":  &eval.StringValue{Value: "k"},
				"value": &eval.IntValue{Value: 42},
			}},
		}},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "'value' must be String")
}

// TestNetURLEncode_RegisteredCorrectly checks both new builtins surface through
// the registry with the right type shapes.
func TestNetURLEncode_RegisteredCorrectly(t *testing.T) {
	for _, name := range []string{"_net_url_encode", "_net_url_encode_form"} {
		t.Run(name, func(t *testing.T) {
			spec, ok := GetSpec(name)
			require.True(t, ok, "%s should be registered", name)
			assert.Equal(t, "std/net", spec.Module)
			assert.Equal(t, 1, spec.NumArgs)
			assert.True(t, spec.IsPure, "%s must be pure", name)
			assert.Equal(t, "", spec.Effect, "%s must have no effect row", name)
			require.NotNil(t, spec.Type)
			require.NotNil(t, spec.Impl)
		})
	}
}
