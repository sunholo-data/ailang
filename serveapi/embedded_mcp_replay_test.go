// TestEmbeddedMCPReplayIsNeverSniffable guards issue #603 (CodeQL go/reflected-xss).
//
// This wrapper replays the MCP SDK's buffered headers, status and body onto the real
// ResponseWriter. Request-controlled bytes genuinely reach the body (the reflection is
// real), but every response the SDK produces today carries a Content-Type and
// X-Content-Type-Options: nosniff, so a browser never content-sniffs the reflecting body
// into a rendered document — i.e. it is non-renderable, not exploitable. This test pins
// the LOCAL assertions we now make on the replay (M1) instead of trusting the SDK's
// internal behaviour to keep doing this.
//
// The anti-vacuity control exists because assertions over a battery that stopped
// reflecting anywhere would pass while measuring nothing. It forces at least one case to
// genuinely contain the payload before the (a)/(b)/(c) checks are trusted.
package serveapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/serveapi/protocol"
)

const htmlPayload = "<script>alert(1)</script>"

func TestEmbeddedMCPReplayIsNeverSniffable(t *testing.T) {
	host := embeddedTestHost{
		resolve: func(context.Context, *http.Request) (any, error) { return embeddedTestSession{name: "s"}, nil },
		tools: func(context.Context, any) ([]ToolDescriptor, error) {
			return []ToolDescriptor{objectTool("echo")}, nil
		},
		invoke: func(_ context.Context, _ any, _ string, args json.RawMessage) (json.RawMessage, error) {
			return args, nil
		},
	}
	handler := embeddedHandler(t, host, 2*time.Second, 4)

	cases := []struct {
		name        string
		body        string
		contentType string // "" means default application/json
		accept      string // "" means default application/json, text/event-stream
		extra       func(*http.Request)
	}{
		{
			name: "normal tools/list",
			body: `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`,
		},
		{
			name: "html in tool args (json-escaped on echo, so it must NOT reflect literally)",
			body: `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"echo","arguments":{"x":"<script>alert(1)</script>"}}}`,
		},
		{
			name:        "bad request Content-Type",
			body:        `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`,
			contentType: "text/html",
		},
		{
			name:   "bad Accept",
			body:   `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`,
			accept: "text/html",
		},
		{
			name: "malformed payload",
			body: htmlPayload,
		},
		{
			name: "empty body",
			body: "",
		},
		{
			name: "hostile Host header",
			body: `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`,
			extra: func(r *http.Request) {
				r.Host = htmlPayload
			},
		},
		{
			name: "hostile Last-Event-ID",
			body: `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`,
			extra: func(r *http.Request) {
				r.Header.Set("Last-Event-ID", htmlPayload)
			},
		},
		{
			name: "hostile MCP-Protocol-Version",
			body: `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`,
			extra: func(r *http.Request) {
				r.Header.Set("MCP-Protocol-Version", htmlPayload)
			},
		},
		{
			name: "unknown method",
			body: `{"jsonrpc":"2.0","id":1,"method":"<script>alert(1)</script>"}`,
		},
		{
			name: "batch rejected",
			body: `[{"jsonrpc":"2.0","id":1,"method":"tools/list"}]`,
		},
		{
			name: "hostile Mcp-Session-Id",
			body: `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`,
			extra: func(r *http.Request) {
				r.Header.Set("Mcp-Session-Id", htmlPayload)
			},
		},
	}

	reflected := 0
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, "/mcp/", strings.NewReader(tc.body))
			ct := tc.contentType
			if ct == "" {
				ct = "application/json"
			}
			accept := tc.accept
			if accept == "" {
				accept = "application/json, text/event-stream"
			}
			request.Header.Set("Content-Type", ct)
			request.Header.Set("Accept", accept)
			if tc.extra != nil {
				tc.extra(request)
			}
			handler.ServeHTTP(recorder, request)

			contentType := recorder.Header().Get("Content-Type")
			if contentType == "" {
				t.Fatalf("empty Content-Type lets the browser sniff a reflecting body: %q", recorder.Body.String())
			}
			if recorder.Header().Get("X-Content-Type-Options") != "nosniff" {
				t.Fatalf("Content-Type=%q missing nosniff; body=%q", contentType, recorder.Body.String())
			}
			if strings.Contains(contentType, "text/html") {
				t.Fatalf("Content-Type=%q is text/html; body=%q", contentType, recorder.Body.String())
			}
			// The SDK this handler used before #885 reflected the payload literally in
			// text/plain errors. The stdlib dispatcher JSON-encodes every body, so the
			// payload now reaches the response only escaped. Count that as reflection
			// (the request data still reaches the body, so the checks above measure
			// something), and require that the literal form never appears.
			if strings.Contains(recorder.Body.String(), htmlPayload) {
				t.Fatalf("request data reflected UNESCAPED: Content-Type=%q body=%q", contentType, recorder.Body.String())
			}
			if strings.Contains(recorder.Body.String(), `\u003cscript\u003e`) {
				reflected++
			}
		})
	}

	// Anti-vacuity control: if nothing reflected the payload, the assertions above
	// passed while measuring nothing.
	if reflected == 0 {
		t.Fatal("battery no longer reflects any request data into the response; assertions are vacuous and prove nothing — repair this test, do not delete it")
	}
}

// TestEmbeddedMCPEveryResponseClassIsLabelled covers the half of the #603 guard the
// battery above cannot reach on its own.
//
// Before ailang#885 this wrapper replayed the MCP SDK's buffered headers, and the test
// here injected a transport that wrote an unlabelled body to prove the replay defaulted
// Content-Type. There is no replay any more: every writer in serveapi/protocol/mcphttp
// labels its own response. The guarantee to pin is therefore "every response CLASS is
// labelled". The battery only produces some of them (SSE results and 400s), so this test
// drives the rest (405, 202, 401 and the host-failure envelope) and requires each
// class to be seen, so a class that silently stops being produced cannot pass vacuously.
func TestEmbeddedMCPEveryResponseClassIsLabelled(t *testing.T) {
	block := func(ctx context.Context) error { <-ctx.Done(); return ctx.Err() }
	host := embeddedTestHost{
		resolve: func(_ context.Context, r *http.Request) (any, error) {
			if r.Header.Get("X-Deny") != "" {
				return nil, testAuthorizationError{status: http.StatusUnauthorized}
			}
			return "s", nil
		},
		tools: func(context.Context, any) ([]ToolDescriptor, error) {
			return []ToolDescriptor{objectTool("echo"), objectTool("hang")}, nil
		},
		invoke: func(ctx context.Context, _ any, name string, args json.RawMessage) (json.RawMessage, error) {
			if name == "hang" {
				return nil, block(ctx)
			}
			return args, nil
		},
	}
	handler := embeddedHandler(t, host, 50*time.Millisecond, 4)
	cases := []struct {
		name, method, body string
		deny               bool
		wantStatus         int
	}{
		{"GET is refused", http.MethodGet, "", false, http.StatusMethodNotAllowed},
		{"notification", http.MethodPost, `{"jsonrpc":"2.0","method":"notifications/initialized"}`, false, http.StatusAccepted},
		{"unauthorized", http.MethodPost, `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`, true, http.StatusUnauthorized},
		{"host timeout envelope", http.MethodPost, `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"hang","arguments":{}}}`, false, http.StatusOK},
		{"sse result", http.MethodPost, `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"echo","arguments":{"x":"<b>"}}}`, false, http.StatusOK},
	}
	seen := map[int]int{}
	for _, tc := range cases {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(tc.method, "/mcp/", strings.NewReader(tc.body))
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Accept", "application/json, text/event-stream")
		if tc.deny {
			request.Header.Set("X-Deny", "1")
		}
		handler.ServeHTTP(recorder, request)
		if recorder.Code != tc.wantStatus {
			t.Fatalf("%s: status=%d want=%d body=%q", tc.name, recorder.Code, tc.wantStatus, recorder.Body.String())
		}
		seen[recorder.Code]++
		if recorder.Header().Get("Content-Type") == "" {
			t.Fatalf("%s: unlabelled response lets a browser sniff it: %q", tc.name, recorder.Body.String())
		}
		if recorder.Header().Get("X-Content-Type-Options") != "nosniff" {
			t.Fatalf("%s: missing nosniff (Content-Type=%q)", tc.name, recorder.Header().Get("Content-Type"))
		}
	}
	// Control: every class was actually produced, so the assertions above were not vacuous.
	for _, status := range []int{http.StatusMethodNotAllowed, http.StatusAccepted, http.StatusUnauthorized, http.StatusOK} {
		if seen[status] == 0 {
			t.Fatalf("control failed: no response with status %d was produced", status)
		}
	}
}

// TestWriteMCPEnvelopeIsLabelled covers the OTHER response path out of this file.
//
// serveTransport is not the only writer: writeMCPEnvelope answers oversized bodies,
// surface errors and panic recovery, and it echoes a request-controlled `id`. It set
// Content-Type but not nosniff, so the two paths disagreed — found by the iteration-153
// evaluator. Not exploitable (encoding/json escapes the id, asserted below as a control),
// but the point of #603 is that this wrapper asserts its own labelling instead of
// inheriting it, and "the encoder escapes by default" is exactly such an inheritance.
func TestWriteMCPEnvelopeIsLabelled(t *testing.T) {
	recorder := httptest.NewRecorder()
	protocol.WriteMCPEnvelope(recorder, json.RawMessage(`"`+htmlPayload+`"`), "invalid MCP request body")

	if got := recorder.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", got)
	}
	if got := recorder.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("envelope path must carry nosniff like the replay path does, got %q", got)
	}
	// Control: the hostile id really did reach the body (so this test is not asserting
	// over an empty response), but only in escaped form.
	body := recorder.Body.String()
	if !strings.Contains(body, `\u003cscript`) {
		t.Fatalf("control failed: escaped id not found in envelope body: %q", body)
	}
	if strings.Contains(body, htmlPayload) {
		t.Fatalf("request-controlled id reached the body UNESCAPED: %q", body)
	}
}
