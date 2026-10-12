package mcphttp

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/serveapi/protocol"
	"github.com/sunholo-data/ailang/serveapi/protocol/hostcall"
)

type typedHostError struct {
	code    int
	message string
}

func (e typedHostError) Error() string               { return "private host details" }
func (e typedHostError) JSONRPCError() (int, string) { return e.code, e.message }

type errorHost struct {
	fakeHost
	err error
}

func (h errorHost) Invoke(ctx context.Context, s protocol.Session, call protocol.Invocation) (protocol.InvocationResult, error) {
	if call.Name == "fail" {
		return protocol.InvocationResult{}, h.err
	}
	return h.fakeHost.Invoke(ctx, s, call)
}

func typedErrorHandler(t *testing.T, err error) http.Handler {
	t.Helper()
	runner, e := hostcall.New(time.Second, 4)
	if e != nil {
		t.Fatal(e)
	}
	host := errorHost{fakeHost{[]protocol.ToolDescriptor{tool("echo"), tool("fail")}}, err}
	handler, e := NewHandler(Config{Agent: protocol.AgentInfo{Name: "t", Version: "1"}, Resolver: host, Tools: host, Invoker: host, Runner: runner})
	if e != nil {
		t.Fatal(e)
	}
	return handler
}

func assertErrorWire(t *testing.T, handler http.Handler, version, body, contentType, want string) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/mcp/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("MCP-Protocol-Version", version)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != contentType || rec.Body.String() != want {
		t.Fatalf("status=%d type=%q body=%q; want 200 %q %q", rec.Code, rec.Header().Get("Content-Type"), rec.Body.String(), contentType, want)
	}
}

func TestTypedHostError(t *testing.T) {
	typed := typedHostError{-32002, `effects 100% "unrecorded"; refs=世界`}
	plain := errors.New("private database failure")
	frozen := "{\"jsonrpc\":\"2.0\",\"id\":\"call-41\",\"error\":{\"code\":-32603,\"message\":\"host callback failed\"}}\n"
	typedWire := sse(`{"jsonrpc":"2.0","id":"call-41","error":{"code":-32002,"message":"effects 100% \"unrecorded\"; refs=世界"}}`)
	cases := []struct {
		name              string
		err               error
		contentType, wire string
	}{
		{"typed", typed, "text/event-stream", typedWire},
		{"wrapped typed", fmt.Errorf("invoke: %w", typed), "text/event-stream", typedWire},
		{"zero code", typedHostError{0, "invalid"}, "application/json", frozen},
		{"empty message", typedHostError{-32002, ""}, "application/json", frozen},
		{"plain", plain, "application/json", frozen},
		{"wrapped plain", fmt.Errorf("invoke: %w", plain), "application/json", frozen},
	}
	for _, version := range SupportedVersions {
		for _, tc := range cases {
			t.Run(version+"/"+tc.name, func(t *testing.T) {
				assertErrorWire(t, typedErrorHandler(t, tc.err), version,
					`{"jsonrpc":"2.0","id":"call-41","method":"tools/call","params":{"name":"fail"}}`, tc.contentType, tc.wire)
			})
		}
	}
}

func TestTypedHostErrorBatch(t *testing.T) {
	body := `[{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"echo"}},{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"fail"}},{"jsonrpc":"2.0","id":3,"method":"ping"}]`
	const success = `{"jsonrpc":"2.0","id":1,"result":{"content":[{"type":"text","text":"{\"ok\":true}"}],"structuredContent":{"ok":true}}}`
	const failure = `{"jsonrpc":"2.0","id":2,"error":{"code":-32002,"message":"effects unrecorded; refs=[er-1 er-2]"}}`
	const ping = `{"jsonrpc":"2.0","id":3,"result":{}}`
	assertErrorWire(t, typedErrorHandler(t, typedHostError{-32002, "effects unrecorded; refs=[er-1 er-2]"}), "2025-03-26", body, "text/event-stream", sse("["+success+","+failure+","+ping+"]"))
	// Untyped errors still discard sibling results and retain the frozen POST-level null id.
	assertErrorWire(t, typedErrorHandler(t, errors.New("db down")), "2025-03-26", body, "application/json", "{\"jsonrpc\":\"2.0\",\"id\":null,\"error\":{\"code\":-32603,\"message\":\"host callback failed\"}}\n")
}
