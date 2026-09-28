package mcphttp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/serveapi/protocol"
	"github.com/sunholo-data/ailang/serveapi/protocol/hostcall"
)

type fakeHost struct{ tools []protocol.ToolDescriptor }

func (fakeHost) ResolveSession(context.Context, *http.Request) (protocol.Session, error) {
	return "s", nil
}
func (h fakeHost) Tools(context.Context, protocol.Session) ([]protocol.ToolDescriptor, error) {
	return h.tools, nil
}
func (fakeHost) Invoke(ctx context.Context, _ protocol.Session, call protocol.Invocation) (protocol.InvocationResult, error) {
	switch call.Name {
	case "hang":
		<-ctx.Done()
		return protocol.InvocationResult{}, ctx.Err()
	case "empty":
		return protocol.InvocationResult{}, nil
	case "bad":
		return protocol.InvocationResult{Value: json.RawMessage(`{`)}, nil
	}
	return protocol.InvocationResult{Value: json.RawMessage(`{"ok":true}`)}, nil
}

func tool(name string) protocol.ToolDescriptor {
	return protocol.ToolDescriptor{Name: name, Description: name, InputSchema: json.RawMessage(`{"type":"object"}`)}
}

func testHandler(t *testing.T) http.Handler {
	t.Helper()
	runner, err := hostcall.New(50*time.Millisecond, 4)
	if err != nil {
		t.Fatal(err)
	}
	host := fakeHost{tools: []protocol.ToolDescriptor{tool("echo"), tool("hang"), tool("empty"), tool("bad"),
		{Name: "typed", Description: "typed", InputSchema: json.RawMessage(`{"type":"object"}`), OutputSchema: json.RawMessage(`{"type":"object"}`)}}}
	h, err := NewHandler(Config{Agent: protocol.AgentInfo{Name: "t", Version: "1"}, Resolver: host, Tools: host, Invoker: host, Runner: runner})
	if err != nil {
		t.Fatal(err)
	}
	return h
}

type wireCase struct {
	name, method, accept, version, body string
	wantStatus                          int
	wantType                            string
	wantBody                            string // exact body
}

func sse(data string) string { return "event: message\ndata: " + data + "\n\n" }

// TestWireContract pins every row of the design doc's wire contract (W1–W16) to
// exact bytes. The design doc (m-serveapi-sdk-free-mcp-dispatch.md §Wire contract)
// is the source of truth; a change here is a wire change and needs a changelog line.
func TestWireContract(t *testing.T) {
	const both = "application/json, text/event-stream"
	cases := []wireCase{
		{"W1 GET refused", http.MethodGet, both, "", "", 405, "text/plain; charset=utf-8", "Method Not Allowed\n"},
		{"W3 Accept missing SSE", http.MethodPost, "application/json", "", `{"jsonrpc":"2.0","id":1,"method":"ping"}`, 400, "application/json",
			`{"jsonrpc":"2.0","id":1,"error":{"code":-32600,"message":"Accept must list both application/json and text/event-stream"}}` + "\n"},
		{"W4 unsupported version", http.MethodPost, both, "2026-07-28", `{"jsonrpc":"2.0","id":1,"method":"ping"}`, 400, "application/json",
			`{"jsonrpc":"2.0","id":1,"error":{"code":-32600,"message":"unsupported MCP-Protocol-Version \"2026-07-28\" (supported: 2025-11-25, 2025-06-18, 2025-03-26)"}}` + "\n"},
		{"W5 absent header served", http.MethodPost, both, "", `{"jsonrpc":"2.0","id":1,"method":"ping"}`, 200, "text/event-stream",
			sse(`{"jsonrpc":"2.0","id":1,"result":{}}`)},
		{"W6 invalid JSON", http.MethodPost, both, "2025-06-18", `{not json`, 400, "application/json", ""},
		{"W6 jsonrpc 1.0", http.MethodPost, both, "2025-06-18", `{"jsonrpc":"1.0","id":1,"method":"ping"}`, 400, "application/json",
			`{"jsonrpc":"2.0","id":1,"error":{"code":-32600,"message":"expected jsonrpc: \"2.0\""}}` + "\n"},
		{"W6 batch refused under 2025-06-18", http.MethodPost, both, "2025-06-18", `[{"jsonrpc":"2.0","id":1,"method":"ping"}]`, 400, "application/json",
			`{"jsonrpc":"2.0","id":null,"error":{"code":-32600,"message":"batch requests are not supported under protocol version 2025-06-18"}}` + "\n"},
		{"W6 batch served under 2025-03-26", http.MethodPost, both, "2025-03-26", `[{"jsonrpc":"2.0","id":1,"method":"ping"},{"jsonrpc":"2.0","method":"notifications/initialized"},{"jsonrpc":"2.0","id":2,"method":"ping"}]`, 200, "text/event-stream",
			sse(`[{"jsonrpc":"2.0","id":1,"result":{}},{"jsonrpc":"2.0","id":2,"result":{}}]`)},
		{"W6 notification-only batch", http.MethodPost, both, "", `[{"jsonrpc":"2.0","method":"notifications/initialized"}]`, 202, "application/json", ""},
		{"W7 initialize supported", http.MethodPost, both, "", `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18"}}`, 200, "text/event-stream",
			sse(`{"jsonrpc":"2.0","id":1,"result":{"capabilities":{"tools":{"listChanged":false}},"protocolVersion":"2025-06-18","serverInfo":{"name":"t","version":"1"}}}`)},
		{"W7 initialize unknown version gets latest", http.MethodPost, both, "", `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"1999-01-01"}}`, 200, "text/event-stream",
			sse(`{"jsonrpc":"2.0","id":1,"result":{"capabilities":{"tools":{"listChanged":false}},"protocolVersion":"2025-11-25","serverInfo":{"name":"t","version":"1"}}}`)},
		{"W8 notification", http.MethodPost, both, "2025-06-18", `{"jsonrpc":"2.0","method":"notifications/initialized"}`, 202, "application/json", ""},
		{"W9 ping", http.MethodPost, both, "2025-11-25", `{"jsonrpc":"2.0","id":"p","method":"ping"}`, 200, "text/event-stream",
			sse(`{"jsonrpc":"2.0","id":"p","result":{}}`)},
		{"W10 tools/list sorted, outputSchema only when declared", http.MethodPost, both, "2025-06-18", `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`, 200, "text/event-stream",
			sse(`{"jsonrpc":"2.0","id":2,"result":{"tools":[` +
				`{"name":"bad","description":"bad","inputSchema":{"type":"object"}},` +
				`{"name":"echo","description":"echo","inputSchema":{"type":"object"}},` +
				`{"name":"empty","description":"empty","inputSchema":{"type":"object"}},` +
				`{"name":"hang","description":"hang","inputSchema":{"type":"object"}},` +
				`{"name":"typed","description":"typed","inputSchema":{"type":"object"},"outputSchema":{"type":"object"}}]}}`)},
		{"W11 tools/call", http.MethodPost, both, "2025-06-18", `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"echo","arguments":{}}}`, 200, "text/event-stream",
			sse(`{"jsonrpc":"2.0","id":3,"result":{"content":[{"type":"text","text":"{\"ok\":true}"}],"structuredContent":{"ok":true}}}`)},
		{"W11 empty host result", http.MethodPost, both, "2025-06-18", `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"empty"}}`, 200, "text/event-stream",
			sse(`{"jsonrpc":"2.0","id":3,"error":{"code":-32603,"message":"host callback returned no result"}}`)},
		{"W11 invalid host JSON", http.MethodPost, both, "2025-06-18", `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"bad"}}`, 200, "text/event-stream",
			sse(`{"jsonrpc":"2.0","id":3,"error":{"code":-32603,"message":"host callback returned invalid JSON"}}`)},
		{"W12 unknown tool", http.MethodPost, both, "2025-06-18", `{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"nope","arguments":{}}}`, 200, "text/event-stream",
			sse(`{"jsonrpc":"2.0","id":4,"error":{"code":-32602,"message":"unknown tool \"nope\""}}`)},
		{"W13 host timeout envelope", http.MethodPost, both, "2025-06-18", `{"jsonrpc":"2.0","id":"x","method":"tools/call","params":{"name":"hang","arguments":{}}}`, 200, "application/json",
			`{"jsonrpc":"2.0","id":"x","error":{"code":-32603,"message":"host callback timed out"}}` + "\n"},
		{"W15 unknown method", http.MethodPost, both, "2025-06-18", `{"jsonrpc":"2.0","id":6,"method":"bogus/method"}`, 200, "text/event-stream",
			sse(`{"jsonrpc":"2.0","id":6,"error":{"code":-32601,"message":"method not found: \"bogus/method\""}}`)},
	}
	handler := testHandler(t)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest(tc.method, "/mcp/", strings.NewReader(tc.body))
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("Accept", tc.accept)
			if tc.version != "" {
				request.Header.Set("MCP-Protocol-Version", tc.version)
			}
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, request)
			if recorder.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d; body=%q", recorder.Code, tc.wantStatus, recorder.Body.String())
			}
			if got := recorder.Header().Get("Content-Type"); got != tc.wantType {
				t.Fatalf("Content-Type = %q, want %q", got, tc.wantType)
			}
			// W16: every response is labelled.
			if recorder.Header().Get("X-Content-Type-Options") != "nosniff" {
				t.Fatal("missing X-Content-Type-Options: nosniff")
			}
			if tc.wantBody != "" && recorder.Body.String() != tc.wantBody {
				t.Fatalf("body mismatch\n got %q\nwant %q", recorder.Body.String(), tc.wantBody)
			}
		})
	}
}

func TestInvalidJSONIsAParseError(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/mcp/", strings.NewReader(`{not json`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json, text/event-stream")
	recorder := httptest.NewRecorder()
	testHandler(t).ServeHTTP(recorder, request)
	var resp struct {
		ID    json.RawMessage `json:"id"`
		Error struct {
			Code int `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("body is not JSON: %q", recorder.Body.String())
	}
	if resp.Error.Code != -32700 || string(resp.ID) != "null" {
		t.Fatalf("got code=%d id=%s, want -32700/null", resp.Error.Code, resp.ID)
	}
}

func TestNewHandlerRefusesMissingFields(t *testing.T) {
	runner, _ := hostcall.New(time.Second, 1)
	host := fakeHost{}
	full := Config{Agent: protocol.AgentInfo{Name: "t", Version: "1"}, Resolver: host, Tools: host, Invoker: host, Runner: runner}
	if _, err := NewHandler(full); err != nil {
		t.Fatalf("control: full config refused: %v", err)
	}
	for name, mutate := range map[string]func(*Config){
		"resolver": func(c *Config) { c.Resolver = nil },
		"tools":    func(c *Config) { c.Tools = nil },
		"invoker":  func(c *Config) { c.Invoker = nil },
		"runner":   func(c *Config) { c.Runner = nil },
		"agent":    func(c *Config) { c.Agent.Name = " " },
	} {
		config := full
		mutate(&config)
		if _, err := NewHandler(config); err == nil {
			t.Fatalf("missing %s was accepted", name)
		}
	}
}

func TestOversizedBodyIsEnvelope(t *testing.T) {
	body := `{"jsonrpc":"2.0","id":1,"method":"ping","params":{"pad":"` + strings.Repeat("a", MaxRequestBodyBytes) + `"}}`
	request := httptest.NewRequest(http.MethodPost, "/mcp/", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json, text/event-stream")
	recorder := httptest.NewRecorder()
	testHandler(t).ServeHTTP(recorder, request)
	if !strings.Contains(recorder.Body.String(), "invalid MCP request body") || recorder.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("W2: status=%d body=%q", recorder.Code, recorder.Body.String()[:min(200, recorder.Body.Len())])
	}
}
