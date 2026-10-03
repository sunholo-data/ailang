package apiserver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/sunholo-data/ailang/internal/testutil"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/effects"
)

// M-SERVEAPI-DIRECTORY-READY M3: the lazy-auth gate on /mcp/connect/, end to
// end through buildRoutes with an AILANG @mcp_token_verifier.

const gateIssuer = "https://auth.example.com"

func gatedModule(slowURL string) string {
	return fmt.Sprintf(`module test/api/gated
import std/net (httpGet)
import std/string (length)

-- "good" passes; "slow" hangs on a Net call (the gate's deadline must abort
-- it); "boom" divides by zero (an RT001 runtime error inside the verifier).
@mcp_token_verifier
export func verifyToken(token: string) -> bool ! {Net} =
  if token == "slow" then length(httpGet(%q)) >= 0
  else if token == "boom" then (10 / (length(token) - 4)) > 0
  else token == "good"

@mcp_auth("oauth2")
@mcp_secret("apiKey")
@optional("apiKey")
export func gated(doc: string, apiKey: string) -> string ! {IO} = "parsed:${doc}"

export func openTool(x: string) -> string ! {IO} = "open:${x}"
`, slowURL)
}

func gatedServer(t *testing.T, slowURL string) *httptest.Server {
	t.Helper()
	// The module imports std/net, so pin the stdlib per test: an earlier test
	// in this package os.Setenv's AILANG_STDLIB_PATH to a directory without it.
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("AILANG_STDLIB_PATH", filepath.Join(root, "std"))
	tmpDir := t.TempDir()
	apiDir := filepath.Join(tmpDir, "test", "api")
	if err := os.MkdirAll(apiDir, 0o755); err != nil {
		t.Fatal(err)
	}
	modPath := filepath.Join(apiDir, "gated.ail")
	if err := os.WriteFile(modPath, []byte(gatedModule(slowURL)), 0o644); err != nil {
		t.Fatal(err)
	}
	eff := effects.NewEffContext(nil)
	for _, c := range []string{"IO", "Net"} {
		eff.Grant(effects.NewCapability(c))
	}
	eff.Net.AllowHTTP, eff.Net.AllowLocalhost = true, true
	srv := New(tmpDir, Config{Port: "0", MCP: true, OAuthIssuer: gateIssuer, EffCtx: eff})
	t.Cleanup(func() { srv.Close() })
	if err := srv.LoadModules([]string{modPath}); err != nil {
		t.Fatalf("LoadModules: %v", err)
	}
	hs := httptest.NewServer(srv.buildRoutes())
	t.Cleanup(hs.Close)
	return hs
}

// rpc posts one JSON-RPC message and returns status, headers and body.
func rpc(t *testing.T, url, token, method string, params any) (int, http.Header, string) {
	t.Helper()
	msg, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": method, "params": params})
	req, _ := http.NewRequest(http.MethodPost, url, bytes.NewReader(msg))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, resp.Header, string(b)
}

func call(name string, args map[string]any) map[string]any {
	return map[string]any{"name": name, "arguments": args}
}

func TestMCPGate_ResourceMetadata(t *testing.T) {
	hs := gatedServer(t, "http://127.0.0.1:1/")
	for _, path := range []string{protectedResourceRoot, protectedResourcePath} {
		resp, err := http.Get(hs.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		var doc struct {
			Resource string   `json:"resource"`
			Servers  []string `json:"authorization_servers"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&doc)
		resp.Body.Close()
		if doc.Resource != hs.URL+listedMCPPath || len(doc.Servers) != 1 || doc.Servers[0] != gateIssuer {
			t.Errorf("%s: %+v", path, doc)
		}
	}
}

func TestMCPGate_Outcomes(t *testing.T) {
	hs := gatedServer(t, "http://127.0.0.1:1/")
	listed := hs.URL + listedMCPPath

	// Discovery and open tools need no token.
	if st, _, body := rpc(t, listed, "", "tools/list", map[string]any{}); st != 200 || !strings.Contains(body, `"gated"`) {
		t.Fatalf("tools/list without token: %d %s", st, body)
	}
	if st, _, body := rpc(t, listed, "", "tools/call", call("openTool", map[string]any{"x": "a"})); st != 200 || !strings.Contains(body, "open:a") {
		t.Fatalf("open tool without token: %d %s", st, body)
	}

	// Gated tool: no token → 401 whose challenge names the metadata document.
	st, h, _ := rpc(t, listed, "", "tools/call", call("gated", map[string]any{"doc": "d"}))
	if st != 401 || !strings.Contains(h.Get("WWW-Authenticate"), `resource_metadata="`+hs.URL+protectedResourcePath+`"`) {
		t.Fatalf("no token: %d, WWW-Authenticate %q", st, h.Get("WWW-Authenticate"))
	}
	if st, h, _ := rpc(t, listed, "bad", "tools/call", call("gated", map[string]any{"doc": "d"})); st != 401 || !strings.Contains(h.Get("WWW-Authenticate"), `error="invalid_token"`) {
		t.Fatalf("bad token: %d %q", st, h.Get("WWW-Authenticate"))
	}
	if st, _, body := rpc(t, listed, "good", "tools/call", call("gated", map[string]any{"doc": "d"})); st != 200 || !strings.Contains(body, "parsed:d") {
		t.Fatalf("good token: %d %s", st, body)
	}
	// A runtime error inside the AILANG verifier fails closed. (Integer
	// division by zero was a Go panic until #1449; the gate's panic arm is
	// pinned by serveapi/protocol TestBearerGate_Outcomes "verifier panic".)
	if st, h, body := rpc(t, listed, "boom", "tools/call", call("gated", map[string]any{"doc": "d"})); st != 503 || h.Get("Retry-After") == "" || strings.Contains(body, "parsed:") {
		t.Fatalf("failing verifier: %d %s", st, body)
	}
}

// The agent surface is not gated: existing integrations that pass the key as
// an argument keep working with no Bearer token.
func TestMCPGate_AgentSurfaceUngated(t *testing.T) {
	hs := gatedServer(t, "http://127.0.0.1:1/")
	st, _, body := rpc(t, hs.URL+"/mcp/", "", "tools/call", call("gated", map[string]any{"doc": "d", "apiKey": "dp_x"}))
	if st != 200 || !strings.Contains(body, "parsed:d") {
		t.Fatalf("/mcp/ gated tool with argument key: %d %s", st, body)
	}
}

// D7 end to end: a verifier hanging on Net is refused with 503 at the 5 s
// deadline, and the deadline reaches the AILANG Net call.
func TestMCPGate_SlowVerifierTimesOut(t *testing.T) {
	testutil.SkipInFastLoop(t, "waits for the 5 s verification deadline")
	cancelled := make(chan struct{}, 1)
	hang := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done(): // the verifier's request was cancelled
			cancelled <- struct{}{}
		case <-time.After(30 * time.Second):
		}
	}))
	defer hang.Close()
	hs := gatedServer(t, hang.URL+"/")
	start := time.Now()
	st, _, body := rpc(t, hs.URL+listedMCPPath, "slow", "tools/call", call("gated", map[string]any{"doc": "d"}))
	elapsed := time.Since(start)
	if st != 503 || strings.Contains(body, "parsed:") {
		t.Fatalf("slow verifier: %d %s", st, body)
	}
	if elapsed < 4*time.Second || elapsed > 7*time.Second {
		t.Fatalf("refused after %v, want about the 5 s deadline", elapsed)
	}
	// The deadline reached the AILANG Net call through EffContext.GoCtx: the
	// hung request is cancelled, not left running.
	select {
	case <-cancelled:
	case <-time.After(2 * time.Second):
		t.Fatal("the verifier's Net request was not cancelled at the deadline")
	}
}
