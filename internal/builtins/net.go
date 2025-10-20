package builtins

import (
	"fmt"

	"github.com/sunholo/ailang/internal/effects"
	"github.com/sunholo/ailang/internal/types"
)

// Network effect builtins for AILANG
// These provide HTTP and network operations

func init() {
	registerNetHTTPRequest()
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
//	-> Result<{status: Int, headers: List<{name: String, value: String}>, body: String, ok: Bool}, NetError>
//	! {Net}
func makeHTTPRequestType() types.Type {
	T := types.NewBuilder()

	// Header type: {name: String, value: String}
	headerType := T.Record(
		types.Field("name", T.String()),
		types.Field("value", T.String()),
	)

	// Response type: {status: Int, headers: List<Header>, body: String, ok: Bool}
	responseType := T.Record(
		types.Field("status", T.Int()),
		types.Field("headers", T.List(headerType)),
		types.Field("body", T.String()),
		types.Field("ok", T.Bool()),
	)

	// Function signature with effects
	return T.Func(
		T.String(),         // method
		T.String(),         // url
		T.List(headerType), // headers
		T.String(),         // body
	).Returns(
		T.App("Result", responseType, T.Con("NetError")),
	).Effects("Net")
}
