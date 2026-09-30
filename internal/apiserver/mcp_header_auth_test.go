package apiserver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// MCP header auth: a tool can take its API key from the tools/call HTTP
// request headers instead of a tool argument, so an agent never holds the key
// in model context (MCP registries inject secrets only as headers).
//
//   - a declared `_headers: Json` param binds from the request headers on the
//     MCP path (same contract as REST @route), is hidden from inputSchema, and
//     cannot be forged through the arguments;
//   - @optional("p") drops p from `required` and binds an absent/null p to its
//     type's zero value, so the handler can fall back to the header;
//   - an @optional name that is not a zero-valuable param is refused at
//     registration.
//
// The E2E tests go through the real streamable-HTTP handler, because that is
// the seam that was broken: the tool handler never read req.Extra.Header.

const headerAuthModule = `module test/api/keyed

import std/json (Json, getString)
import std/option (Option, Some, None)

@optional("apiKey")
export pure func whoami(apiKey: string, _headers: Json) -> string =
  if apiKey != "" then "arg:${apiKey}"
  else match getString(_headers, "X-Api-Key") {
    Some(k) => "header:${k}",
    None => match getString(_headers, "Authorization") {
      Some(a) => "auth:${a}",
      None => "none"
    }
  }

@optional("nope")
export pure func badOptional(x: string) -> string = x
`

func headerAuthServer(t *testing.T) *Server {
	t.Helper()
	tmpDir := t.TempDir()
	apiDir := filepath.Join(tmpDir, "test", "api")
	if err := os.MkdirAll(apiDir, 0755); err != nil {
		t.Fatal(err)
	}
	modPath := filepath.Join(apiDir, "keyed.ail")
	if err := os.WriteFile(modPath, []byte(headerAuthModule), 0644); err != nil {
		t.Fatal(err)
	}
	repoRoot, _ := filepath.Abs(filepath.Join("..", ".."))
	t.Setenv("AILANG_STDLIB_PATH", filepath.Join(repoRoot, "std"))
	srv := New(tmpDir, Config{Port: "0"})
	if err := srv.LoadModules([]string{modPath}); err != nil {
		t.Fatalf("LoadModules: %v", err)
	}
	return srv
}

type headerRoundTripper struct{ headers map[string]string }

func (h headerRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	for k, v := range h.headers {
		r.Header.Set(k, v)
	}
	return http.DefaultTransport.RoundTrip(r)
}

// mcpHTTPSession connects a go-sdk client over streamable HTTP, sending the
// given headers on every request.
func mcpHTTPSession(t *testing.T, srv *Server, headers map[string]string) *mcp.ClientSession {
	t.Helper()
	hs := httptest.NewServer(NewMCPServer(srv).HTTPHandler())
	t.Cleanup(hs.Close)
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "0.0.1"}, nil)
	cs, err := client.Connect(context.Background(), &mcp.StreamableClientTransport{
		Endpoint:             hs.URL,
		HTTPClient:           &http.Client{Transport: headerRoundTripper{headers}},
		DisableStandaloneSSE: true,
		MaxRetries:           -1,
	}, nil)
	if err != nil {
		t.Fatalf("client Connect: %v", err)
	}
	t.Cleanup(func() { cs.Close() })
	return cs
}

func callText(t *testing.T, cs *mcp.ClientSession, name string, args map[string]any) (string, bool) {
	t.Helper()
	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("CallTool %s: %v", name, err)
	}
	return mcpResultText(t, res), res.IsError
}

func TestMCPHeaderAuth_OptionalAnnotationLoaded(t *testing.T) {
	srv := headerAuthServer(t)
	defer srv.Close()
	exp := findExport(t, srv, "whoami")
	if len(exp.Optional) != 1 || exp.Optional[0] != "apiKey" {
		t.Fatalf("Optional = %v, want [apiKey]", exp.Optional)
	}
}

func TestMCPHeaderAuth_Schema(t *testing.T) {
	srv := headerAuthServer(t)
	defer srv.Close()
	cs := mcpHTTPSession(t, srv, nil)

	res, err := cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	var whoami *mcp.Tool
	for _, tool := range res.Tools {
		switch tool.Name {
		case "whoami":
			whoami = tool
		case "badOptional":
			t.Error("badOptional has @optional on a non-existent param and must not be registered")
		}
	}
	if whoami == nil {
		t.Fatalf("whoami missing from tools/list: %v", toolNames(res.Tools))
	}
	raw, _ := json.Marshal(whoami.InputSchema)
	var schema struct {
		Properties map[string]any `json:"properties"`
		Required   []string       `json:"required"`
	}
	if err := json.Unmarshal(raw, &schema); err != nil {
		t.Fatal(err)
	}
	if _, ok := schema.Properties["_headers"]; ok {
		t.Errorf("_headers must not be advertised: %s", raw)
	}
	if _, ok := schema.Properties["apiKey"]; !ok {
		t.Errorf("apiKey should still be advertised as a property: %s", raw)
	}
	for _, r := range schema.Required {
		if r == "apiKey" || r == "_headers" {
			t.Errorf("%s must not be required: %s", r, raw)
		}
	}
}

func TestMCPHeaderAuth_KeyFromHeaders(t *testing.T) {
	srv := headerAuthServer(t)
	defer srv.Close()

	cases := []struct {
		name    string
		headers map[string]string
		args    map[string]any
		want    string
	}{
		{"x-api-key header, no arg", map[string]string{"X-API-Key": "k1"}, map[string]any{}, `"header:k1"`},
		{"bearer header, no arg", map[string]string{"Authorization": "Bearer k2"}, map[string]any{}, `"auth:Bearer k2"`},
		{"null arg falls back to header", map[string]string{"X-API-Key": "k1"}, map[string]any{"apiKey": nil}, `"header:k1"`},
		{"explicit arg still wins", map[string]string{"X-API-Key": "k1"}, map[string]any{"apiKey": "k3"}, `"arg:k3"`},
		{"no key anywhere reaches the handler", nil, map[string]any{}, `"none"`},
		{"_headers in args cannot forge headers", nil, map[string]any{"_headers": map[string]any{"X-Api-Key": "forged"}}, `"none"`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cs := mcpHTTPSession(t, srv, tc.headers)
			text, isErr := callText(t, cs, "whoami", tc.args)
			if isErr {
				t.Fatalf("unexpected tool error: %s", text)
			}
			if strings.TrimSpace(text) != tc.want {
				t.Errorf("got %s, want %s", text, tc.want)
			}
		})
	}
}

// Without an HTTP request (stdio, or a direct call) _headers binds to an empty
// object rather than nil/Unit.
func TestMCPHeaderAuth_NoExtraBindsEmptyHeaders(t *testing.T) {
	srv := headerAuthServer(t)
	defer srv.Close()
	exp := findExport(t, srv, "whoami")
	handler := NewMCPServer(srv).makeToolHandler("test/api/keyed", exp)
	res, err := handler(context.Background(), mcpCallReq("whoami", `{}`))
	if err != nil {
		t.Fatal(err)
	}
	if text := mcpResultText(t, res); res.IsError || strings.TrimSpace(text) != `"none"` {
		t.Errorf("got %s (isError=%v), want \"none\"", text, res.IsError)
	}
}

func TestValidateOptionalParams(t *testing.T) {
	base := ExportInfo{Name: "f", ParamNames: []string{"s", "h", "j"}, ParamTypes: []string{"string", "Json", "Json"}}
	cases := []struct {
		optional []string
		wantErr  string
	}{
		{[]string{"s"}, ""},
		{[]string{"missing"}, "no such parameter"},
		{[]string{"j"}, "has no zero value"},
	}
	for _, tc := range cases {
		exp := base
		exp.Optional = tc.optional
		err := validateOptionalParams(exp)
		if tc.wantErr == "" && err != nil {
			t.Errorf("%v: unexpected error %v", tc.optional, err)
		}
		if tc.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tc.wantErr)) {
			t.Errorf("%v: want error containing %q, got %v", tc.optional, tc.wantErr, err)
		}
	}
	headers := ExportInfo{Name: "f", ParamNames: []string{"_headers"}, ParamTypes: []string{"Json"}, Optional: []string{"_headers"}}
	if err := validateOptionalParams(headers); err == nil || !strings.Contains(err.Error(), "bound from the request") {
		t.Errorf("@optional(\"_headers\") should be refused, got %v", err)
	}
}
