package builtins

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/sunholo-data/ailang/internal/eval"
)

// callURLParse drives netURLParseImpl and returns the resulting TaggedValue
// (Ok(record) or Err(string)).
func callURLParse(t *testing.T, s string) *eval.TaggedValue {
	t.Helper()
	v, err := netURLParseImpl(nil, []eval.Value{&eval.StringValue{Value: s}})
	require.NoError(t, err, "_net_url_parse must not return a Go error")
	tv, ok := v.(*eval.TaggedValue)
	require.True(t, ok, "_net_url_parse should return a Result TaggedValue, got %T", v)
	return tv
}

// unwrapOkURL asserts the parse succeeded and returns the Url field map.
func unwrapOkURL(t *testing.T, tv *eval.TaggedValue) map[string]string {
	t.Helper()
	require.Equal(t, "Ok", tv.CtorName, "expected Ok, got %s", tv.CtorName)
	require.Len(t, tv.Fields, 1)
	rec, ok := tv.Fields[0].(*eval.RecordValue)
	require.True(t, ok, "Ok field should be a RecordValue, got %T", tv.Fields[0])
	out := map[string]string{}
	for _, f := range []string{"scheme", "host", "port", "path", "query", "fragment"} {
		sv, ok := rec.Fields[f].(*eval.StringValue)
		require.True(t, ok, "field %q should be a StringValue, got %T", f, rec.Fields[f])
		out[f] = sv.Value
	}
	require.Len(t, rec.Fields, 6, "Url record must have exactly 6 fields")
	return out
}

// callParseQuery drives netURLParseQueryImpl and returns ordered (name,value) pairs.
func callParseQuery(t *testing.T, s string) [][2]string {
	t.Helper()
	v, err := netURLParseQueryImpl(nil, []eval.Value{&eval.StringValue{Value: s}})
	require.NoError(t, err, "_net_url_parse_query must not return a Go error")
	list, ok := v.(*eval.ListValue)
	require.True(t, ok, "_net_url_parse_query should return a ListValue, got %T", v)
	out := make([][2]string, 0, len(list.Elements))
	for i, e := range list.Elements {
		rec, ok := e.(*eval.RecordValue)
		require.True(t, ok, "element %d should be a RecordValue, got %T", i, e)
		name, ok := rec.Fields["name"].(*eval.StringValue)
		require.True(t, ok, "element %d 'name' should be a StringValue", i)
		value, ok := rec.Fields["value"].(*eval.StringValue)
		require.True(t, ok, "element %d 'value' should be a StringValue", i)
		out = append(out, [2]string{name.Value, value.Value})
	}
	return out
}

func TestNetURLParse_FullURL(t *testing.T) {
	got := unwrapOkURL(t, callURLParse(t, "https://user@host:8443/a/b?q=1#frag"))
	assert.Equal(t, "https", got["scheme"])
	assert.Equal(t, "host", got["host"])
	assert.Equal(t, "8443", got["port"])
	assert.Equal(t, "/a/b", got["path"])
	assert.Equal(t, "q=1", got["query"])
	assert.Equal(t, "frag", got["fragment"])
}

func TestNetURLParse_SchemeRelative(t *testing.T) {
	// "//host/path" has no scheme.
	got := unwrapOkURL(t, callURLParse(t, "//host/path"))
	assert.Equal(t, "", got["scheme"])
	assert.Equal(t, "host", got["host"])
	assert.Equal(t, "", got["port"])
	assert.Equal(t, "/path", got["path"])
}

func TestNetURLParse_NoPort(t *testing.T) {
	got := unwrapOkURL(t, callURLParse(t, "https://example.com/x"))
	assert.Equal(t, "example.com", got["host"])
	assert.Equal(t, "", got["port"], "absent port must be empty string, not a sentinel")
}

func TestNetURLParse_IPv6Host(t *testing.T) {
	// Go url.Hostname() strips the [] brackets from an IPv6 literal.
	got := unwrapOkURL(t, callURLParse(t, "http://[::1]:80/"))
	assert.Equal(t, "::1", got["host"], "IPv6 host must have no brackets")
	assert.Equal(t, "80", got["port"])
	assert.Equal(t, "/", got["path"])
}

func TestNetURLParse_Userinfo(t *testing.T) {
	// Userinfo is not exposed as a field; host must not include it.
	got := unwrapOkURL(t, callURLParse(t, "https://user@host/"))
	assert.Equal(t, "host", got["host"])
	assert.Equal(t, "/", got["path"])
}

func TestNetURLParse_PathOnly(t *testing.T) {
	got := unwrapOkURL(t, callURLParse(t, "/just/a/path"))
	assert.Equal(t, "", got["scheme"])
	assert.Equal(t, "", got["host"])
	assert.Equal(t, "/just/a/path", got["path"])
}

func TestNetURLParse_EmptyQueryAndFragment(t *testing.T) {
	got := unwrapOkURL(t, callURLParse(t, "https://example.com/x"))
	assert.Equal(t, "", got["query"])
	assert.Equal(t, "", got["fragment"])
}

func TestNetURLParse_PathDecoded(t *testing.T) {
	// u.Path is the decoded path — %20 becomes a space.
	got := unwrapOkURL(t, callURLParse(t, "https://example.com/a%20b"))
	assert.Equal(t, "/a b", got["path"])
}

func TestNetURLParse_QueryRawEncoded(t *testing.T) {
	// query is RawQuery — still percent-encoded (feed to parseQuery to decode).
	got := unwrapOkURL(t, callURLParse(t, "https://example.com/?q=hello%20world"))
	assert.Equal(t, "q=hello%20world", got["query"])
}

func TestNetURLParse_ErrorNeverPanics(t *testing.T) {
	// A control char / invalid escape makes url.Parse error. Must be Err, never panic.
	for _, bad := range []string{"http://%zz", "http://foo\x7f.com", "://noscheme"} {
		tv := callURLParse(t, bad)
		if tv.CtorName == "Err" {
			require.Len(t, tv.Fields, 1)
			_, ok := tv.Fields[0].(*eval.StringValue)
			assert.True(t, ok, "Err payload must be a StringValue")
		}
		// If Go leniently accepts it (Ok), that is also fine — the point is no panic.
	}
	// Assert at least one of these is a genuine Err (invalid %-escape is a hard error).
	assert.Equal(t, "Err", callURLParse(t, "http://%zz").CtorName)
}

func TestNetURLParse_RejectsNonString(t *testing.T) {
	_, err := netURLParseImpl(nil, []eval.Value{&eval.IntValue{Value: 42}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expected String")
}

func TestNetURLParseQuery_PercentDecoded(t *testing.T) {
	got := callParseQuery(t, "q=hello%20world&r=2")
	assert.Equal(t, [][2]string{{"q", "hello world"}, {"r", "2"}}, got)
}

func TestNetURLParseQuery_PlusIsSpace(t *testing.T) {
	// url.QueryUnescape decodes + as space (form semantics) — round-trips urlEncodeForm.
	got := callParseQuery(t, "q=hello+world")
	assert.Equal(t, [][2]string{{"q", "hello world"}}, got)
}

func TestNetURLParseQuery_OrderPreserved(t *testing.T) {
	// Deliberately NOT alphabetical — url.ParseQuery would sort; we must not.
	got := callParseQuery(t, "zeta=1&alpha=2&mu=3")
	assert.Equal(t, [][2]string{{"zeta", "1"}, {"alpha", "2"}, {"mu", "3"}}, got)
}

func TestNetURLParseQuery_DuplicateKeys(t *testing.T) {
	got := callParseQuery(t, "a=1&a=2")
	assert.Equal(t, [][2]string{{"a", "1"}, {"a", "2"}}, got, "duplicate keys must be kept as separate entries")
}

func TestNetURLParseQuery_BareKey(t *testing.T) {
	got := callParseQuery(t, "flag")
	assert.Equal(t, [][2]string{{"flag", ""}}, got)
}

func TestNetURLParseQuery_EmptyString(t *testing.T) {
	got := callParseQuery(t, "")
	assert.Equal(t, [][2]string{}, got, "empty query string must yield an empty list")
}

func TestNetURLParseQuery_EmptyValue(t *testing.T) {
	got := callParseQuery(t, "k=")
	assert.Equal(t, [][2]string{{"k", ""}}, got)
}

func TestNetURLParseQuery_EmptyKey(t *testing.T) {
	got := callParseQuery(t, "=v")
	assert.Equal(t, [][2]string{{"", "v"}}, got)
}

func TestNetURLParseQuery_ValueWithEquals(t *testing.T) {
	// Split on the FIRST '=' only; the rest is part of the value.
	got := callParseQuery(t, "expr=a=b")
	assert.Equal(t, [][2]string{{"expr", "a=b"}}, got)
}

func TestNetURLParseQuery_SkipsEmptySegments(t *testing.T) {
	got := callParseQuery(t, "a=1&&b=2")
	assert.Equal(t, [][2]string{{"a", "1"}, {"b", "2"}}, got)
}

func TestNetURLParseQuery_LenientOnBadEscape(t *testing.T) {
	// A malformed %-escape must not panic; the raw half is kept.
	got := callParseQuery(t, "q=%zz")
	require.Len(t, got, 1)
	assert.Equal(t, "q", got[0][0])
	assert.Equal(t, "%zz", got[0][1])
}

func TestNetURLParseQuery_RejectsNonString(t *testing.T) {
	_, err := netURLParseQueryImpl(nil, []eval.Value{&eval.IntValue{Value: 42}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expected String")
}

// TestNetURLParseRoundTrip proves parseQuery(urlEncodeForm(pairs)) recovers the
// pairs (modulo urlEncodeForm's alphabetical key sort).
func TestNetURLParseRoundTrip(t *testing.T) {
	encoded := callURLEncodeForm(t,
		[2]string{"q", "hello world"},
		[2]string{"n", "42"},
	)
	// urlEncodeForm sorts keys: "n=42&q=hello+world".
	got := callParseQuery(t, encoded)
	assert.Equal(t, [][2]string{{"n", "42"}, {"q", "hello world"}}, got)
}

// TestNetURLParse_RegisteredPureNoNetCap asserts both new builtins surface through
// the registry as PURE with no effect row — usable without the Net capability (CP2 /
// design success-criterion "no Net capability required").
func TestNetURLParse_RegisteredPureNoNetCap(t *testing.T) {
	for _, name := range []string{"_net_url_parse", "_net_url_parse_query"} {
		t.Run(name, func(t *testing.T) {
			spec, ok := GetSpec(name)
			require.True(t, ok, "%s should be registered", name)
			assert.Equal(t, "std/net", spec.Module)
			assert.Equal(t, 1, spec.NumArgs)
			assert.True(t, spec.IsPure, "%s must be pure", name)
			assert.Equal(t, "", spec.Effect, "%s must have no effect row (no Net capability)", name)
			require.NotNil(t, spec.Type)
			require.NotNil(t, spec.Impl)
		})
	}
}
