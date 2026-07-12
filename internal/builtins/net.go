package builtins

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/types"
)

// Network effect builtins for AILANG
// These provide HTTP and network operations

func init() {
	registerNetHTTPRequest()
	registerNetHTTPRequestBytes()
	registerNetURLEncode()
	registerNetURLEncodeForm()
	registerNetURLParse()
	registerNetURLParseQuery()
}

// registerNetHTTPRequest registers the _net_httpRequest builtin
// Old location: internal/effects/net.go
func registerNetHTTPRequest() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/net",
		Name:    "_net_httpRequest",
		NumArgs: 4,
		IsPure:  false,
		Effect:  "Net",
		Type:    makeHTTPRequestType,
		Impl:    effects.NetHTTPRequest,

		// Enhanced metadata (M-DX1.11)
		Metadata: &BuiltinMetadata{
			Description: "Make an HTTP request with custom headers and body",
			LongDesc: `Performs an HTTP request to the specified URL with the given method,
headers, and request body. Returns a Result type containing either the HTTP
response (status, headers, body) or a NetError if the request fails.

The Net capability must be granted to use this function.`,

			Params: []ParamDoc{
				{Name: "method", Description: "HTTP method (GET, POST, PUT, DELETE, etc.)"},
				{Name: "url", Description: "Target URL (must start with http:// or https://)"},
				{Name: "headers", Description: "List of {name: string, value: string} header records"},
				{Name: "body", Description: "Request body as a string"},
			},

			Returns: "Result[HttpResponse, NetError] where HttpResponse contains status, headers, body, and ok flag",

			Examples: []Example{
				{
					Code: `let response = _net_httpRequest(
  "GET",
  "https://api.example.com/users",
  [{name: "Accept", value: "application/json"}],
  ""
)`,
					Description: "Simple GET request with JSON accept header",
				},
			},

			Since:     "v0.2.0",
			Stability: StabilityStable,
			Tags:      []string{"http", "network", "request", "api", "web"},
			Category:  "network",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _net_httpRequest: %v", err))
	}
}

// makeHTTPRequestType builds the type signature for _net_httpRequest
// Type: (String, String, List<{name: String, value: String}>, String)
//
//	-> Result<{status: Int, headers: List<Header>, body: String, bodyBytes: Bytes, ok: Bool}, NetError>
//	! {Net}
func makeHTTPRequestType() types.Type {
	T := types.NewBuilder()
	return T.Func(
		T.String(),                // method
		T.String(),                // url
		T.List(httpHeaderType(T)), // headers
		T.String(),                // body
	).Returns(
		T.App("Result", httpResponseType(T), T.Con("NetError")),
	).Effects("Net")
}

// registerNetHTTPRequestBytes registers the _net_httpRequestBytes builtin.
// Same shape as _net_httpRequest, but the body parameter is Bytes instead of String.
// Defaults Content-Type to application/octet-stream and sets explicit Content-Length.
func registerNetHTTPRequestBytes() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/net",
		Name:    "_net_httpRequestBytes",
		NumArgs: 4,
		IsPure:  false,
		Effect:  "Net",
		Type:    makeHTTPRequestBytesType,
		Impl:    effects.NetHTTPRequestBytes,

		Metadata: &BuiltinMetadata{
			Description: "Make an HTTP request with a raw bytes body (image upload, binary protocols, etc.)",
			LongDesc: `Performs an HTTP request to the specified URL with a raw bytes request
body. Use this for binary uploads (images, PDFs, audio) or any protocol that
requires exact byte-level control over the request body.

Defaults Content-Type to "application/octet-stream" if the caller doesn't set
one. Sets an explicit Content-Length to suppress chunked transfer encoding —
many binary-upload servers (S3, LinkedIn images, etc.) require a fixed length.

The Net capability must be granted to use this function.`,

			Params: []ParamDoc{
				{Name: "method", Description: "HTTP method (GET, POST, PUT, DELETE, etc.)"},
				{Name: "url", Description: "Target URL (must start with http:// or https://)"},
				{Name: "headers", Description: "List of {name: string, value: string} header records"},
				{Name: "body", Description: "Request body as raw bytes"},
			},

			Returns: "Result[HttpResponse, NetError] — HttpResponse has status, headers, body (UTF-8 view), bodyBytes (raw), ok",

			Examples: []Example{
				{
					Code: `import std/fs (readFileBytes)
import std/bytes (fromBase64)

let Ok(b64) = readFileBytes("photo.png")
let Some(bytes) = fromBase64(b64)
_net_httpRequestBytes("PUT", uploadUrl, [], bytes)`,
					Description: "PUT raw image bytes to an upload URL (e.g. LinkedIn image attach API)",
				},
			},

			Since:     "v0.19.0",
			Stability: StabilityStable,
			Tags:      []string{"http", "network", "request", "api", "binary", "upload"},
			Category:  "network",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _net_httpRequestBytes: %v", err))
	}
}

// makeHTTPRequestBytesType builds the type signature for _net_httpRequestBytes.
// Same as makeHTTPRequestType but with Bytes body instead of String.
func makeHTTPRequestBytesType() types.Type {
	T := types.NewBuilder()
	return T.Func(
		T.String(),                // method
		T.String(),                // url
		T.List(httpHeaderType(T)), // headers
		T.Bytes(),                 // body
	).Returns(
		T.App("Result", httpResponseType(T), T.Con("NetError")),
	).Effects("Net")
}

// httpHeaderType returns the {name: String, value: String} record type.
// Shared between _net_httpRequest and _net_httpRequestBytes type specs.
func httpHeaderType(T *types.Builder) types.Type {
	return T.Record(
		types.Field("name", T.String()),
		types.Field("value", T.String()),
	)
}

// registerNetURLEncode registers the _net_url_encode builtin.
// Percent-encodes a single value per RFC 3986 (space -> %20). Safe for both
// path/query positions and form bodies that need RFC-3986 semantics.
func registerNetURLEncode() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/net",
		Name:    "_net_url_encode",
		NumArgs: 1,
		IsPure:  true,
		Effect:  "",
		Type:    makeNetURLEncodeType,
		Impl:    netURLEncodeImpl,

		Metadata: &BuiltinMetadata{
			Description: "Percent-encode a string per RFC 3986 (space encoded as %20)",
			LongDesc: `Percent-encodes a single value for use in a URL path, query string, or
form body. Follows RFC 3986: space is encoded as %20 (NOT +). Use this for
auth tokens, signed-URL parameters, or individual form values.

For building a full application/x-www-form-urlencoded body from key/value
pairs, use _net_url_encode_form instead — it encodes space as + per the
WHATWG form spec, which is what HTTP servers expect for form bodies.`,
			Params: []ParamDoc{
				{Name: "s", Description: "The string to percent-encode"},
			},
			Returns: "Percent-encoded string",
			Examples: []Example{
				{Code: `_net_url_encode("hello world")`, Description: `Returns "hello%20world"`},
				{Code: `_net_url_encode("a=b&c/d")`, Description: `Returns "a%3Db%26c%2Fd"`},
			},
			SeeAlso:   []string{"_net_url_encode_form", "_bytes_to_base64"},
			Since:     "v0.20.0",
			Stability: StabilityStable,
			Tags:      []string{"url", "encoding", "oauth", "http", "form"},
			Category:  "network",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _net_url_encode: %v", err))
	}
}

// makeNetURLEncodeType: string -> string
func makeNetURLEncodeType() types.Type {
	T := types.NewBuilder()
	return T.Func(T.String()).Returns(T.String()).Build()
}

// netURLEncodeImpl percent-encodes its argument with RFC 3986 semantics
// suitable for any URL position (path, query, fragment, form value).
//
// url.PathEscape leaves reserved query/form chars (=, &, +) untouched because
// they're valid inside a path segment — but those are exactly the chars OAuth
// secrets and query parameters need encoded. url.QueryEscape encodes them,
// at the cost of using + for space (form-spec). Swap + back to %20 so the
// output is safe in path, query, AND fragment positions.
func netURLEncodeImpl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
	s, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("_net_url_encode: expected String, got %T", args[0])
	}
	encoded := strings.ReplaceAll(url.QueryEscape(s.Value), "+", "%20")
	return &eval.StringValue{Value: encoded}, nil
}

// registerNetURLEncodeForm registers the _net_url_encode_form builtin.
// Builds an application/x-www-form-urlencoded body from a list of
// {name, value} records. Space is encoded as + per the WHATWG form spec.
func registerNetURLEncodeForm() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/net",
		Name:    "_net_url_encode_form",
		NumArgs: 1,
		IsPure:  true,
		Effect:  "",
		Type:    makeNetURLEncodeFormType,
		Impl:    netURLEncodeFormImpl,

		Metadata: &BuiltinMetadata{
			Description: "Build an application/x-www-form-urlencoded body from {name,value} records",
			LongDesc: `Builds a form-encoded request body (key=value&key=value...) from a list
of {name: String, value: String} records. Encodes space as + and percent-
encodes other unsafe characters per the WHATWG URL form spec — this is the
shape HTTP servers expect for application/x-www-form-urlencoded bodies
(OAuth2 token exchange, webhooks, traditional HTML form POSTs).

Keys are sorted alphabetically (Go url.Values.Encode behavior). If you need
order-preserving encoding for signing schemes (AWS SigV4, OAuth1.0a), build
the body manually using _net_url_encode on each value.

For single-value encoding (auth tokens, query-string params), use
_net_url_encode — it follows RFC 3986 and encodes space as %20.`,
			Params: []ParamDoc{
				{Name: "params", Description: "List of {name: String, value: String} records"},
			},
			Returns: "URL-encoded form body string (empty string for empty list)",
			Examples: []Example{
				{
					Code:        `_net_url_encode_form([{name: "client_id", value: "x"}, {name: "client_secret", value: "a/b=c"}])`,
					Description: `Returns "client_id=x&client_secret=a%2Fb%3Dc"`,
				},
			},
			SeeAlso:   []string{"_net_url_encode", "_net_httpRequest"},
			Since:     "v0.20.0",
			Stability: StabilityStable,
			Tags:      []string{"url", "encoding", "oauth", "http", "form"},
			Category:  "network",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _net_url_encode_form: %v", err))
	}
}

// makeNetURLEncodeFormType: List[{name: String, value: String}] -> String
func makeNetURLEncodeFormType() types.Type {
	T := types.NewBuilder()
	return T.Func(T.List(httpHeaderType(T))).Returns(T.String()).Build()
}

// netURLEncodeFormImpl walks the list of {name,value} records and returns
// the url.Values.Encode() output. Same field-extraction shape as parseHeaders
// in internal/effects/net.go.
func netURLEncodeFormImpl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
	list, ok := args[0].(*eval.ListValue)
	if !ok {
		return nil, fmt.Errorf("_net_url_encode_form: expected List, got %T", args[0])
	}
	values := url.Values{}
	for i, elem := range list.Elements {
		rec, ok := elem.(*eval.RecordValue)
		if !ok {
			return nil, fmt.Errorf("_net_url_encode_form: param at index %d must be a record, got %T", i, elem)
		}
		nameVal, ok := rec.Fields["name"]
		if !ok {
			return nil, fmt.Errorf("_net_url_encode_form: param at index %d missing 'name' field", i)
		}
		nameStr, ok := nameVal.(*eval.StringValue)
		if !ok {
			return nil, fmt.Errorf("_net_url_encode_form: param at index %d 'name' must be String, got %T", i, nameVal)
		}
		valueVal, ok := rec.Fields["value"]
		if !ok {
			return nil, fmt.Errorf("_net_url_encode_form: param at index %d missing 'value' field", i)
		}
		valueStr, ok := valueVal.(*eval.StringValue)
		if !ok {
			return nil, fmt.Errorf("_net_url_encode_form: param at index %d 'value' must be String, got %T", i, valueVal)
		}
		values.Add(nameStr.Value, valueStr.Value)
	}
	return &eval.StringValue{Value: values.Encode()}, nil
}

// registerNetURLParse registers the _net_url_parse builtin.
// Parses an RFC-3986 URL into a Url record (scheme/host/port/path/query/fragment).
// Backed by Go net/url.Parse — pure, no Net capability. Fallible: Err on malformed input.
func registerNetURLParse() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/net",
		Name:    "_net_url_parse",
		NumArgs: 1,
		IsPure:  true,
		Effect:  "",
		Type:    makeNetURLParseType,
		Impl:    netURLParseImpl,

		Metadata: &BuiltinMetadata{
			Description: "Parse an RFC-3986 URL into a {scheme,host,port,path,query,fragment} record",
			LongDesc: `Parses a URL string into its structured components using Go's net/url
(the reference RFC-3986 parser). Returns Result[Url, string]: Ok(record) with
scheme/host/port/path/query/fragment fields, or Err(message) for malformed
input (control chars, invalid %-escape, bad port) — never a silent fallback.

Fields are all strings. port is "" when absent (Go url.Port() semantics); host
is the hostname only (no port, no IPv6 brackets); path and fragment are percent-
DECODED; query is the raw substring after ? (still encoded — feed to parseQuery).

Pure function: no Net capability needed. It reaches no network; it only takes a
URL apart. The inverse of urlEncode/urlEncodeForm.`,
			Params: []ParamDoc{
				{Name: "s", Description: "The URL string to parse"},
			},
			Returns: "Result[Url, string] where Url = {scheme, host, port, path, query, fragment : string}",
			Examples: []Example{
				{
					Code:        `_net_url_parse("https://user@host:8443/a/b?q=1#frag")`,
					Description: `Ok({scheme:"https", host:"host", port:"8443", path:"/a/b", query:"q=1", fragment:"frag"})`,
				},
			},
			SeeAlso:   []string{"_net_url_parse_query", "_net_url_encode", "_net_url_encode_form"},
			Since:     "v0.30.0",
			Stability: StabilityExperimental,
			Tags:      []string{"url", "parse", "http", "web", "orchestration"},
			Category:  "network",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _net_url_parse: %v", err))
	}
}

// makeNetURLParseType: string -> Result[Url, string]
func makeNetURLParseType() types.Type {
	T := types.NewBuilder()
	return T.Func(T.String()).Returns(
		T.App("Result", urlRecordType(T), T.String()),
	).Build()
}

// urlRecordType returns the Url record type:
// {scheme, host, port, path, query, fragment : string}. All fields are strings.
func urlRecordType(T *types.Builder) types.Type {
	return T.Record(
		types.Field("scheme", T.String()),
		types.Field("host", T.String()),
		types.Field("port", T.String()),
		types.Field("path", T.String()),
		types.Field("query", T.String()),
		types.Field("fragment", T.String()),
	)
}

// netURLParseImpl parses its argument with net/url.Parse and marshals the
// *url.URL into a Url RecordValue wrapped in Ok(...). On parse error it returns
// Err(err.Error()) — no silent fallback on the structural fields (CP2).
//
// Field mapping (all decoded string fields — no byte offsets exposed):
//
//	scheme   = u.Scheme      ("" if scheme-relative)
//	host     = u.Hostname()  (no port, no IPv6 brackets)
//	port     = u.Port()      ("" when absent)
//	path     = u.Path        (percent-decoded)
//	query    = u.RawQuery    (raw, still percent-encoded — feed to parseQuery)
//	fragment = u.Fragment    (decoded; "" when absent)
func netURLParseImpl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
	s, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("_net_url_parse: expected String, got %T", args[0])
	}
	u, err := url.Parse(s.Value)
	if err != nil {
		return wrapErr(err.Error()), nil
	}
	rec := &eval.RecordValue{
		Fields: map[string]eval.Value{
			"scheme":   &eval.StringValue{Value: u.Scheme},
			"host":     &eval.StringValue{Value: u.Hostname()},
			"port":     &eval.StringValue{Value: u.Port()},
			"path":     &eval.StringValue{Value: u.Path},
			"query":    &eval.StringValue{Value: u.RawQuery},
			"fragment": &eval.StringValue{Value: u.Fragment},
		},
	}
	return wrapOk(rec), nil
}

// registerNetURLParseQuery registers the _net_url_parse_query builtin.
// Parses a query string ("a=1&b=2") into order-preserving {name,value} pairs.
// Values are percent-DECODED (inverse of urlEncodeForm). Total — never panics.
func registerNetURLParseQuery() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/net",
		Name:    "_net_url_parse_query",
		NumArgs: 1,
		IsPure:  true,
		Effect:  "",
		Type:    makeNetURLParseQueryType,
		Impl:    netURLParseQueryImpl,

		Metadata: &BuiltinMetadata{
			Description: "Parse a query string into order-preserving, percent-decoded {name,value} pairs",
			LongDesc: `Parses a query string ("a=1&b=2") into a list of {name: String, value: String}
records — the inverse of urlEncodeForm. Values are percent-DECODED.

Unlike Go's url.ParseQuery (which returns a sorted map, lossy for order and
duplicate keys), this parses the raw string itself: it splits on &, each pair on
the FIRST =, url.QueryUnescape's both halves, and PRESERVES source order and
duplicate keys (?a=1&a=2 -> two entries). A bare key (?flag) yields
{name:"flag", value:""}. An empty string yields []. Total: never panics.

Pure function: no Net capability needed.`,
			Params: []ParamDoc{
				{Name: "s", Description: "The query string to parse (the raw substring after ?, no leading ?)"},
			},
			Returns: "List[{name: String, value: String}] in source order, values percent-decoded",
			Examples: []Example{
				{
					Code:        `_net_url_parse_query("q=hello%20world&r=2")`,
					Description: `[{name:"q", value:"hello world"}, {name:"r", value:"2"}]`,
				},
			},
			SeeAlso:   []string{"_net_url_parse", "_net_url_encode_form"},
			Since:     "v0.30.0",
			Stability: StabilityExperimental,
			Tags:      []string{"url", "parse", "query", "http", "web"},
			Category:  "network",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _net_url_parse_query: %v", err))
	}
}

// makeNetURLParseQueryType: string -> List[{name: String, value: String}]
func makeNetURLParseQueryType() types.Type {
	T := types.NewBuilder()
	return T.Func(T.String()).Returns(T.List(httpHeaderType(T))).Build()
}

// netURLParseQueryImpl parses a raw query string into order-preserving
// {name,value} records with percent-decoded values. It deliberately does NOT
// use url.ParseQuery (which sorts and dedups); it splits the raw string to
// preserve source order and duplicate keys — the inverse of netURLEncodeFormImpl.
//
// Rules: empty string -> []; split on "&" (empty segments skipped); each segment
// on the FIRST "="; both halves url.QueryUnescape'd (best-effort: on decode error
// the raw half is kept, matching url.ParseQuery leniency, never panics); a bare
// key with no "=" -> {name:key, value:""}.
func netURLParseQueryImpl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
	s, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("_net_url_parse_query: expected String, got %T", args[0])
	}
	elems := []eval.Value{}
	if s.Value != "" {
		for _, pair := range strings.Split(s.Value, "&") {
			if pair == "" {
				continue
			}
			var rawKey, rawVal string
			if idx := strings.IndexByte(pair, '='); idx >= 0 {
				rawKey, rawVal = pair[:idx], pair[idx+1:]
			} else {
				rawKey, rawVal = pair, ""
			}
			name := queryUnescapeLenient(rawKey)
			value := queryUnescapeLenient(rawVal)
			elems = append(elems, &eval.RecordValue{
				Fields: map[string]eval.Value{
					"name":  &eval.StringValue{Value: name},
					"value": &eval.StringValue{Value: value},
				},
			})
		}
	}
	return &eval.ListValue{Elements: elems}, nil
}

// queryUnescapeLenient percent-decodes a query half, falling back to the raw
// input if decoding fails so the parser stays total (never panics, never errors).
func queryUnescapeLenient(raw string) string {
	if decoded, err := url.QueryUnescape(raw); err == nil {
		return decoded
	}
	return raw
}

// httpResponseType returns the HttpResponse record type with body (string view)
// and bodyBytes (raw bytes). Shared between _net_httpRequest and _net_httpRequestBytes
// — both populate both fields so callers can pick whichever view they need.
func httpResponseType(T *types.Builder) types.Type {
	return T.Record(
		types.Field("status", T.Int()),
		types.Field("headers", T.List(httpHeaderType(T))),
		types.Field("body", T.String()),
		types.Field("bodyBytes", T.Bytes()),
		types.Field("ok", T.Bool()),
	)
}
