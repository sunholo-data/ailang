package protocol

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// M-MCP-FILE-HANDOFF F1b (O5): a refused tools/call keeps the 401 status and
// WWW-Authenticate header, and its body is a JSON-RPC tool error carrying the
// same challenge in _meta["mcp/www_authenticate"].
func TestBearerGate_AdmitCallCarriesChallengeMeta(t *testing.T) {
	g := NewBearerGate(func(_ context.Context, tok string) (bool, error) { return tok == "good", nil }, metadataFor, time.Second, 4)
	for _, token := range []string{"", "bad"} {
		w := httptest.NewRecorder()
		if g.AdmitCall(w, gateRequest(token), json.RawMessage(`"req-7"`)) {
			t.Fatalf("token %q admitted", token)
		}
		challenge := w.Header().Get("WWW-Authenticate")
		if w.Code != 401 || !strings.HasPrefix(challenge, "Bearer resource_metadata=") {
			t.Fatalf("token %q: %d %q", token, w.Code, challenge)
		}
		var msg struct {
			JSONRPC string          `json:"jsonrpc"`
			ID      json.RawMessage `json:"id"`
			Result  struct {
				IsError bool                `json:"isError"`
				Meta    map[string][]string `json:"_meta"`
				Content []map[string]any    `json:"content"`
			} `json:"result"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &msg); err != nil {
			t.Fatalf("token %q: body %s: %v", token, w.Body, err)
		}
		got := msg.Result.Meta[WWWAuthenticateMetaKey]
		if msg.JSONRPC != "2.0" || string(msg.ID) != `"req-7"` || !msg.Result.IsError || len(msg.Result.Content) != 1 ||
			len(got) != 1 || got[0] != challenge {
			t.Errorf("token %q: body %s", token, w.Body)
		}
	}
	// Admit (no call id) keeps the plain JSON error body.
	w := httptest.NewRecorder()
	g.Admit(w, gateRequest(""))
	if !strings.Contains(w.Body.String(), `"error":"unauthorized"`) {
		t.Errorf("Admit body changed: %s", w.Body)
	}
}

func TestToolCalls(t *testing.T) {
	batch := `[{"jsonrpc":"2.0","id":1,"method":"tools/list"},{"jsonrpc":"2.0","id":"a","method":"tools/call","params":{"name":"x"}}]`
	calls := ToolCalls([]byte(batch))
	if len(calls) != 1 || calls[0].Name != "x" || string(calls[0].ID) != `"a"` {
		t.Fatalf("ToolCalls(batch) = %+v", calls)
	}
	if names := ToolCallNames([]byte(`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"y"}}`)); len(names) != 1 || names[0] != "y" {
		t.Fatalf("ToolCallNames = %v", names)
	}
}
