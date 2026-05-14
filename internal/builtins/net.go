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
