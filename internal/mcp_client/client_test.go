package mcp_client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// fakeServer is a minimal MCP endpoint whose transport behaviour is set per
// test: whether it issues and requires an Mcp-Session-Id, whether it answers
// requests as SSE or as plain JSON, and what it reports as served_for.
type fakeServer struct {
	t           *testing.T
	stateful    bool   // issue Mcp-Session-Id on initialize and require it after
	jsonReplies bool   // answer requests with application/json instead of SSE
	servedFor   string // served_for in the prompt_get payload
	initStatus  int    // non-zero: fail initialize with this status
	gotVersion  string // forVersion the client sent to prompt_get
}

const fakeSessionID = "sess-123"

func (f *fakeServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	var msg struct {
		ID     json.RawMessage `json:"id"`
		Method string          `json:"method"`
		Params struct {
			ProtocolVersion string         `json:"protocolVersion"`
			Arguments       map[string]any `json:"arguments"`
		} `json:"params"`
	}
	if err := json.Unmarshal(body, &msg); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	if msg.Method == "initialize" {
		if f.initStatus != 0 {
			http.Error(w, "upstream exploded", f.initStatus)
			return
		}
		if f.stateful {
			w.Header().Set("Mcp-Session-Id", fakeSessionID)
		}
		f.reply(w, msg.ID, map[string]any{"protocolVersion": msg.Params.ProtocolVersion, "capabilities": map[string]any{}})
		return
	}
	// Every request after initialize must carry the negotiated version header;
	// a stateless server has nothing else to know it by.
	if got := r.Header.Get("MCP-Protocol-Version"); got != ProtocolVersion {
		http.Error(w, fmt.Sprintf("missing/unsupported MCP-Protocol-Version %q", got), http.StatusBadRequest)
		return
	}
	if f.stateful && r.Header.Get("Mcp-Session-Id") != fakeSessionID {
		http.Error(w, "missing session", http.StatusBadRequest)
		return
	}
	if !f.stateful && r.Header.Get("Mcp-Session-Id") != "" {
		f.t.Errorf("client sent Mcp-Session-Id %q to a server that never issued one", r.Header.Get("Mcp-Session-Id"))
	}
	switch msg.Method {
	case "notifications/initialized":
		w.WriteHeader(http.StatusAccepted)
	case "tools/call":
		f.gotVersion, _ = msg.Params.Arguments["forVersion"].(string)
		payload, _ := json.Marshal(map[string]any{"served_for": f.servedFor, "data": map[string]any{"markdown": "# prompt"}})
		f.reply(w, msg.ID, map[string]any{"content": []map[string]any{{"type": "text", "text": string(payload)}}})
	default:
		http.Error(w, "unexpected method "+msg.Method, http.StatusBadRequest)
	}
}

func (f *fakeServer) reply(w http.ResponseWriter, id json.RawMessage, result any) {
	data, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": id, "result": result})
	if f.jsonReplies {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(data)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	_, _ = fmt.Fprintf(w, "event: message\ndata: %s\n\n", data)
}

func call(t *testing.T, f *fakeServer, cliVersion string) (map[string]any, error) {
	t.Helper()
	f.t = t
	server := httptest.NewServer(f)
	defer server.Close()
	c := New(Options{BaseURL: server.URL + "/mcp/", AILangVersion: cliVersion})
	return c.CallTool(context.Background(), "prompt_get", map[string]any{"forVersion": cliVersion, "kind": "agent"})
}

// The prod regression (2026-09-28): the server runs stateless and never issues
// Mcp-Session-Id, and the client refused to continue without one.
func TestStatelessServerNoSessionID(t *testing.T) {
	f := &fakeServer{servedFor: "0.47.1"}
	body, err := call(t, f, "v0.47.1")
	if err != nil {
		t.Fatalf("CallTool against a stateless server: %v", err)
	}
	if body["served_for"] != "0.47.1" {
		t.Fatalf("body = %v", body)
	}
}

func TestStatefulServerSessionIDIsEchoed(t *testing.T) {
	if _, err := call(t, &fakeServer{stateful: true, servedFor: "0.47.1"}, "v0.47.1"); err != nil {
		t.Fatalf("CallTool against a stateful server: %v", err)
	}
}

// The transport lets a server answer with one JSON object instead of SSE; a
// client MUST accept both.
func TestJSONResponsesAreAccepted(t *testing.T) {
	if _, err := call(t, &fakeServer{jsonReplies: true, servedFor: "0.47.1"}, "v0.47.1"); err != nil {
		t.Fatalf("CallTool with application/json replies: %v", err)
	}
}

// Snapshots are keyed "0.47.1"; the CLI's version is "v0.47.1".
func TestForVersionIsSentInWireForm(t *testing.T) {
	f := &fakeServer{servedFor: "0.47.1"}
	if _, err := call(t, f, "v0.47.1"); err != nil {
		t.Fatal(err)
	}
	if f.gotVersion != "0.47.1" {
		t.Fatalf("server received forVersion %q, want the snapshot key 0.47.1", f.gotVersion)
	}
}

func TestVersionMismatchStillDetected(t *testing.T) {
	_, err := call(t, &fakeServer{servedFor: "0.46.0"}, "v0.47.1")
	if !errors.Is(err, ErrVersionMismatch) {
		t.Fatalf("err = %v, want ErrVersionMismatch", err)
	}
}

// A server failure must be reported as the failure, not as a missing header
// (the old client checked Mcp-Session-Id before the status code).
func TestInitializeServerErrorIsReportedAsStatus(t *testing.T) {
	_, err := call(t, &fakeServer{initStatus: http.StatusInternalServerError}, "v0.47.1")
	if err == nil || !strings.Contains(err.Error(), "HTTP 500") || strings.Contains(err.Error(), "Mcp-Session-Id") {
		t.Fatalf("err = %v, want an HTTP 500 diagnosis", err)
	}
}

func TestWireVersion(t *testing.T) {
	for in, want := range map[string]string{
		"v0.47.1":             "0.47.1",
		"0.47.1":              "0.47.1",
		"v0.47.1-21-g441efa4": "0.47.1-21-g441efa4", // dev builds stay distinct
		"dev":                 "dev",
		"":                    "",
	} {
		if got := WireVersion(in); got != want {
			t.Errorf("WireVersion(%q) = %q, want %q", in, got, want)
		}
	}
}
