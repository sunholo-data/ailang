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
package apiserver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
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
			if strings.Contains(recorder.Body.String(), htmlPayload) {
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

// TestEmbeddedMCPReplayDefaultsContentTypeWhenTransportOmitsIt covers the OTHER half of
// the #603 guard, which the battery above cannot reach.
//
// The SDK sets a Content-Type on every response path it has today, so the default branch
// in serveTransport never fires through the public HTTP path — deleting that branch reds
// no test at all (measured). A guard whose removal breaks nothing is not a guard. The only
// way to exercise it is to inject a transport that writes an unlabelled body, which is
// precisely the future the branch exists to survive: a dependency bump that stops setting
// Content-Type would otherwise hand the browser a sniffable, reflecting body.
func TestEmbeddedMCPReplayDefaultsContentTypeWhenTransportOmitsIt(t *testing.T) {
	handler := &embeddedMCPHandler{
		transport: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			// No Header().Set("Content-Type", ...) — deliberately unlabelled.
			_, _ = w.Write([]byte("<html>" + htmlPayload + "</html>"))
		}),
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/mcp/", strings.NewReader(`{}`))
	request = request.WithContext(context.WithValue(request.Context(), embeddedMCPContextKey{},
		embeddedMCPContext{failure: &embeddedCallbackFailure{}}))

	handler.serveTransport(recorder, request, json.RawMessage("1"))

	// Control 1: the body really is the hostile payload we wrote, so this test is not
	// asserting over an empty response.
	if !strings.Contains(recorder.Body.String(), htmlPayload) {
		t.Fatalf("control failed: fake transport body did not reach the replay: %q", recorder.Body.String())
	}
	// Control 2: absent the guard this body WOULD be sniffed as HTML, so the assertion
	// below is protecting against a real rendering path rather than a hypothetical one.
	if sniffed := http.DetectContentType(recorder.Body.Bytes()); !strings.Contains(sniffed, "text/html") {
		t.Fatalf("control failed: body should sniff to text/html, got %q", sniffed)
	}

	if got := recorder.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("unlabelled replayed body must be defaulted to application/json, got %q", got)
	}
	if got := recorder.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("replayed response must carry nosniff, got %q", got)
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
	writeMCPEnvelope(recorder, json.RawMessage(`"`+htmlPayload+`"`), "invalid MCP request body")

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
