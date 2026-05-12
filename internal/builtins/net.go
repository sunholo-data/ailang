package builtins

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/types"
)

// Network effect builtins for AILANG
// These provide HTTP and network operations

func init() {
	registerNetHTTPRequest()
	registerNetHTTPRequestBytes()
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
