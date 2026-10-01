package mcphttp

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/serveapi/protocol"
	"github.com/sunholo-data/ailang/serveapi/protocol/hostcall"
)

// M-SERVEAPI-DIRECTORY-READY M3 parity: the stdlib dispatcher refuses gated
// tools/calls exactly as serve-api's /mcp/connect/ does (same BearerGate), so
// an embedder's listed surface answers the same 401/503 contract. The cases
// mirror internal/apiserver/mcp_gate_test.go TestMCPGate_Outcomes.
func gatedTestHandler(t *testing.T, gate *protocol.BearerGate) http.Handler {
	t.Helper()
	runner, err := hostcall.New(time.Second, 4)
	if err != nil {
		t.Fatal(err)
	}
	gated := tool("gated")
	gated.Auth = protocol.ToolAuthOAuth2
	host := fakeHost{tools: []protocol.ToolDescriptor{tool("echo"), gated}}
	h, err := NewHandler(Config{Agent: protocol.AgentInfo{Name: "t", Version: "1"}, Resolver: host, Tools: host, Invoker: host, Runner: runner, Gate: gate})
	if err != nil {
		t.Fatal(err)
	}
	return h
}

func postCall(h http.Handler, name, token string) *httptest.ResponseRecorder {
	body, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{"name": name, "arguments": map[string]any{}}})
	r := httptest.NewRequest(http.MethodPost, "https://svc.example/mcp/connect/", strings.NewReader(string(body)))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Accept", "application/json, text/event-stream")
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func TestGateParity(t *testing.T) {
	verify := func(_ context.Context, tok string) (bool, error) {
		if tok == "boom" {
			return false, errors.New("backend down")
		}
		return tok == "good", nil
	}
	meta := func(r *http.Request) string {
		return protocol.PublicBaseURL(r) + "/.well-known/oauth-protected-resource/mcp/connect/"
	}
	h := gatedTestHandler(t, protocol.NewBearerGate(verify, meta, time.Second, 4))

	cases := []struct {
		tool, token string
		want        int
	}{
		{"echo", "", 200},  // open tool, no token
		{"gated", "", 401}, // challenge
		{"gated", "bad", 401},
		{"gated", "good", 200},
		{"gated", "boom", 503}, // verifier error fails closed
	}
	for _, c := range cases {
		w := postCall(h, c.tool, c.token)
		if w.Code != c.want {
			t.Errorf("%s token=%q: status %d, want %d (%s)", c.tool, c.token, w.Code, c.want, w.Body)
		}
		if c.want == 401 && !strings.Contains(w.Header().Get("WWW-Authenticate"), "resource_metadata=") {
			t.Errorf("%s token=%q: 401 without a resource_metadata challenge", c.tool, c.token)
		}
	}
}

// A gated descriptor on a handler with no Gate fails closed rather than open.
func TestGateMissingFailsClosed(t *testing.T) {
	h := gatedTestHandler(t, nil)
	if w := postCall(h, "gated", "anything"); w.Code != 503 {
		t.Fatalf("gated tool, no gate: %d, want 503 (%s)", w.Code, w.Body)
	}
	if w := postCall(h, "echo", ""); w.Code != 200 {
		t.Fatalf("open tool, no gate: %d", w.Code)
	}
}

// An unknown Auth value is a descriptor error, not an open tool.
func TestDescriptorUnknownAuthRejected(t *testing.T) {
	d := tool("x")
	d.Auth = "basic"
	if _, err := protocol.CallerSurface([]protocol.ToolDescriptor{d}); err == nil {
		t.Fatal("CallerSurface accepted Auth=basic")
	}
}
