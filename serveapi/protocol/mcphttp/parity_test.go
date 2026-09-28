package mcphttp

// The differential against the MCP SDK (design doc AC4). The SDK is imported
// ONLY here, in a _test file: test imports are outside the build closure that
// `make check-protocol-closure` measures, so consumers never link it. Every SDK
// bump reruns this file, which is the drift alarm for our hand-written subset.
//
// Normalisation, applied to both sides before comparing a `parity` row:
//   - only the JSON-RPC payload is compared (the SSE `data:` line, or the body);
//   - `ttlMs` / `cacheScope` are dropped: 2026-era fields the SDK emits even
//     under older negotiated versions (design doc D-E);
//   - `outputSchema: null` is dropped: an SDK artefact of an absent schema (V30);
//   - `initialize` capabilities are dropped: we deliberately advertise
//     `listChanged:false` and no `logging` (W7).

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/sunholo-data/ailang/serveapi/protocol"
	"github.com/sunholo-data/ailang/serveapi/protocol/hostcall"
)

// sdkReference rebuilds the handler this package replaced: a stateless SDK
// streamable handler over the same descriptors and the same invoker.
func sdkReference(t *testing.T) http.Handler {
	t.Helper()
	host := fakeHost{}
	server := mcp.NewServer(&mcp.Implementation{Name: "t", Version: "1"}, nil)
	for _, name := range []string{"bad", "echo", "empty"} {
		name := name
		server.AddTool(&mcp.Tool{Name: name, Description: name, InputSchema: json.RawMessage(`{"type":"object"}`)},
			func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				result, _ := host.Invoke(ctx, "s", protocol.Invocation{Name: name, Arguments: req.Params.Arguments})
				return &mcp.CallToolResult{
					Content:           []mcp.Content{&mcp.TextContent{Text: string(result.Value)}},
					StructuredContent: result.Value,
				}, nil
			})
	}
	return mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server },
		&mcp.StreamableHTTPOptions{Stateless: true})
}

func parityHandler(t *testing.T) http.Handler {
	t.Helper()
	runner, err := hostcall.New(time.Second, 4)
	if err != nil {
		t.Fatal(err)
	}
	host := fakeHost{tools: []protocol.ToolDescriptor{tool("bad"), tool("echo"), tool("empty")}}
	h, err := NewHandler(Config{Agent: protocol.AgentInfo{Name: "t", Version: "1"}, Resolver: host, Tools: host, Invoker: host, Runner: runner})
	if err != nil {
		t.Fatal(err)
	}
	return h
}

type parityCase struct {
	name, method, accept, version, body string
	mode                                string // "parity" or "diff"
	// check asserts OUR response on a "diff" row (status, decoded payload).
	check func(t *testing.T, status int, payload any)
}

func payloadOf(t *testing.T, rec *httptest.ResponseRecorder) any {
	t.Helper()
	body := rec.Body.String()
	for _, line := range strings.Split(body, "\n") {
		if strings.HasPrefix(line, "data: ") {
			body = strings.TrimPrefix(line, "data: ")
			break
		}
	}
	body = strings.TrimSpace(body)
	if body == "" {
		return nil
	}
	var v any
	if err := json.Unmarshal([]byte(body), &v); err != nil {
		return "non-json:" + body
	}
	return normalise(v)
}

func normalise(v any) any {
	switch x := v.(type) {
	case map[string]any:
		out := map[string]any{}
		for k, val := range x {
			if k == "ttlMs" || k == "cacheScope" || (k == "outputSchema" && val == nil) {
				continue
			}
			out[k] = normalise(val)
		}
		if result, ok := out["result"].(map[string]any); ok {
			if _, isInit := result["serverInfo"]; isInit {
				delete(result, "capabilities")
			}
		}
		return out
	case []any:
		for i := range x {
			x[i] = normalise(x[i])
		}
		return x
	}
	return v
}

func serve(h http.Handler, c parityCase) *httptest.ResponseRecorder {
	req := httptest.NewRequest(c.method, "/mcp/", strings.NewReader(c.body))
	req.Header.Set("Content-Type", "application/json")
	if c.accept != "" {
		req.Header.Set("Accept", c.accept)
	}
	if c.version != "" {
		req.Header.Set("MCP-Protocol-Version", c.version)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func errCode(payload any) float64 {
	m, _ := payload.(map[string]any)
	e, _ := m["error"].(map[string]any)
	code, _ := e["code"].(float64)
	return code
}

func wantError(status int, code float64) func(*testing.T, int, any) {
	return func(t *testing.T, gotStatus int, payload any) {
		t.Helper()
		if gotStatus != status || errCode(payload) != code {
			t.Fatalf("got status=%d code=%v payload=%v, want status=%d code=%v", gotStatus, errCode(payload), payload, status, code)
		}
	}
}

func parityCorpus() []parityCase {
	const both = "application/json, text/event-stream"
	var cases []parityCase
	add := func(mode, name, method, accept, version, body string, check func(*testing.T, int, any)) {
		cases = append(cases, parityCase{name: name, method: method, accept: accept, version: version, body: body, mode: mode, check: check})
	}
	versions := []string{"", "2025-03-26", "2025-06-18", "2025-11-25"}
	for _, v := range versions {
		label := v
		if label == "" {
			label = "no-header"
		}
		add("parity", "ping/"+label, "POST", both, v, `{"jsonrpc":"2.0","id":1,"method":"ping"}`, nil)
		add("parity", "tools-list/"+label, "POST", both, v, `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`, nil)
		add("parity", "call-echo/"+label, "POST", both, v, `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"echo","arguments":{"x":1}}}`, nil)
		add("parity", "call-unknown/"+label, "POST", both, v, `{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"nope","arguments":{}}}`, nil)
		add("parity", "notification/"+label, "POST", both, v, `{"jsonrpc":"2.0","method":"notifications/initialized"}`, nil)
		add("parity", "string-id/"+label, "POST", both, v, `{"jsonrpc":"2.0","id":"abc","method":"ping"}`, nil)
	}
	for _, cv := range []string{"2025-03-26", "2025-06-18", "2025-11-25"} {
		add("parity", "initialize/"+cv, "POST", both, "", `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"`+cv+`","capabilities":{},"clientInfo":{"name":"c","version":"1"}}}`, nil)
	}
	add("parity", "initialize/unknown-version", "POST", both, "", `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"1999-01-01","capabilities":{},"clientInfo":{"name":"c","version":"1"}}}`, nil)
	add("parity", "GET", "GET", "text/event-stream", "2025-06-18", "", nil)
	add("parity", "DELETE", "DELETE", "", "2025-06-18", "", nil)
	add("parity", "batch-under-2025-03-26-requests-only", "POST", both, "", `[{"jsonrpc":"2.0","method":"notifications/initialized"}]`, nil)

	// Intentional differences (design doc W3, W4, W6, W15). Each asserts our shape.
	add("diff", "W3 accept json only", "POST", "application/json", "2025-06-18", `{"jsonrpc":"2.0","id":7,"method":"tools/list"}`, wantError(400, codeInvalidRequest))
	add("diff", "W4 version 2026-07-28", "POST", both, "2026-07-28", `{"jsonrpc":"2.0","id":8,"method":"tools/list"}`, wantError(400, codeInvalidRequest))
	add("diff", "W4 version bogus", "POST", both, "1999-01-01", `{"jsonrpc":"2.0","id":9,"method":"tools/list"}`, wantError(400, codeInvalidRequest))
	add("diff", "W6 invalid json", "POST", both, "2025-06-18", `{not json`, wantError(400, codeParseError))
	add("diff", "W6 jsonrpc 1.0", "POST", both, "2025-06-18", `{"jsonrpc":"1.0","id":1,"method":"ping"}`, wantError(400, codeInvalidRequest))
	add("diff", "W6 batch under 2025-06-18", "POST", both, "2025-06-18", `[{"jsonrpc":"2.0","id":1,"method":"ping"}]`, wantError(400, codeInvalidRequest))
	add("diff", "W15 unknown method", "POST", both, "2025-06-18", `{"jsonrpc":"2.0","id":6,"method":"bogus/method"}`, wantError(200, codeMethodNotFound))
	add("diff", "W6 one-element batch under 2025-03-26 answers an array", "POST", both, "", `[{"jsonrpc":"2.0","id":1,"method":"ping"}]`,
		func(t *testing.T, status int, payload any) {
			arr, ok := payload.([]any)
			if status != 200 || !ok || len(arr) != 1 {
				t.Fatalf("status=%d payload=%v, want 200 + one-element array", status, payload)
			}
		})
	return cases
}

func TestParityWithSDK(t *testing.T) {
	ours, ref := parityHandler(t), sdkReference(t)
	corpus := parityCorpus()
	if len(corpus) < 30 {
		t.Fatalf("corpus has %d requests, design doc AC4 requires >= 30", len(corpus))
	}
	parity := 0
	for _, c := range corpus {
		t.Run(c.name, func(t *testing.T) {
			got := serve(ours, c)
			if c.mode == "diff" {
				c.check(t, got.Code, payloadOf(t, got))
				return
			}
			want := serve(ref, c)
			if got.Code != want.Code {
				t.Fatalf("status: ours=%d sdk=%d\n ours=%q\n  sdk=%q", got.Code, want.Code, got.Body.String(), want.Body.String())
			}
			gotSSE := strings.HasPrefix(got.Header().Get("Content-Type"), "text/event-stream")
			wantSSE := strings.HasPrefix(want.Header().Get("Content-Type"), "text/event-stream")
			if gotSSE != wantSSE {
				t.Fatalf("framing: ours=%q sdk=%q", got.Header().Get("Content-Type"), want.Header().Get("Content-Type"))
			}
			if g, w := payloadOf(t, got), payloadOf(t, want); !reflect.DeepEqual(g, w) {
				t.Fatalf("payload differs\n ours=%v\n  sdk=%v", g, w)
			}
			parity++
		})
	}
	if parity == 0 {
		t.Fatal("control failed: no parity row was compared")
	}
}

// TestSDKClientInterop drives the handler with the SDK's own client end to end:
// connect (initialize + initialized), list tools, call one.
func TestSDKClientInterop(t *testing.T) {
	server := httptest.NewServer(testHandler(t))
	defer server.Close()
	ctx := context.Background()
	client := mcp.NewClient(&mcp.Implementation{Name: "interop", Version: "1"}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: server.URL}, nil)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer session.Close()
	tools, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("tools/list: %v", err)
	}
	if len(tools.Tools) != 5 {
		t.Fatalf("tools/list returned %d tools, want 5", len(tools.Tools))
	}
	result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "echo", Arguments: map[string]any{}})
	if err != nil {
		t.Fatalf("tools/call: %v", err)
	}
	text, ok := result.Content[0].(*mcp.TextContent)
	if !ok || text.Text != `{"ok":true}` {
		t.Fatalf("tools/call content = %#v", result.Content)
	}
}
