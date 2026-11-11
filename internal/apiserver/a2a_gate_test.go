package apiserver

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// M-EXECUTOR-GE-A2A M2: under --mcp-connect-only, /a2a/ sits behind the same
// Bearer gate as /mcp/connect/, and the agent card stays open.

const a2aGatedModule = `module test/api/agent

import std/option (Option, Some, None)
import std/string (startsWith, substring, length, toUpper)
` + identVerifier + `
-- Answer in capitals.
@noexpose
@nomcp
export func ask(text: string) -> string ! {IO} = toUpper(text)

@mcp_auth("oauth2")
export func whoami(note: string, _auth: McpIdentity) -> string ! {IO} = "${_auth.subject}|${note}"
`

const a2aUngatedHandlerModule = `module test/api/agent

import std/option (Option, Some, None)
import std/string (startsWith, substring, length, toUpper)
` + identVerifier + `
-- An ordinary export: HTTP route and MCP tool.
export func ask(text: string) -> string ! {IO} = toUpper(text)

@mcp_auth("oauth2")
export func whoami(note: string, _auth: McpIdentity) -> string ! {IO} = "${_auth.subject}|${note}"
`

func a2aGatedHTTP(t *testing.T, cfg Config) *httptest.Server {
	t.Helper()
	cfg.MCP, cfg.MCPConnectOnly, cfg.A2A, cfg.A2AMessage = true, true, true, "ask"
	srv, err := identityServer(t, "agent", a2aGatedModule, cfg)
	if err != nil {
		t.Fatalf("LoadModules: %v", err)
	}
	for _, v := range []func() error{srv.ValidateMCPConnectOnly, srv.ValidateMCPIdentity, srv.ValidateA2A} {
		if err := v(); err != nil {
			t.Fatalf("startup validation: %v", err)
		}
	}
	prev := log.Writer()
	log.SetOutput(io.Discard)
	mux := srv.buildRoutes()
	log.SetOutput(prev)
	hs := httptest.NewServer(mux)
	t.Cleanup(hs.Close)
	return hs
}

func a2aSendAs(t *testing.T, url, token, text string) (int, http.Header, string) {
	t.Helper()
	return rpcWith(t, url+"/a2a/", token, nil, "message/send", a2aUserMessage(text))
}

func TestA2AGatedUnderConnectOnly(t *testing.T) {
	hs := a2aGatedHTTP(t, Config{})

	st, h, body := a2aSendAs(t, hs.URL, "", "hi")
	if st != http.StatusUnauthorized || !strings.Contains(h.Get("WWW-Authenticate"), "resource_metadata=") {
		t.Fatalf("no token: %d %q %s, want 401 naming the resource metadata", st, h.Get("WWW-Authenticate"), body)
	}
	if strings.Contains(body, "HI") {
		t.Fatal("no token: the handler ran")
	}
	if st, _, body := a2aSendAs(t, hs.URL, "wrong", "hi"); st != http.StatusUnauthorized || strings.Contains(body, "HI") {
		t.Fatalf("bad token: %d %s, want 401", st, body)
	}
	if st, _, body := a2aSendAs(t, hs.URL, "tok-alice", "hi"); st != http.StatusOK || !strings.Contains(body, `"text":"HI"`) || !strings.Contains(body, `"completed"`) {
		t.Fatalf("signed in: %d %s, want the completed answer", st, body)
	}
	// The resource metadata the 401 points at is served.
	md := strings.TrimSuffix(strings.SplitN(h.Get("WWW-Authenticate"), `resource_metadata="`, 2)[1], `"`)
	md = strings.SplitN(md, `"`, 2)[0]
	req, _ := http.NewRequest(http.MethodGet, hs.URL+strings.TrimPrefix(md, "http://"+strings.TrimPrefix(hs.URL, "http://")), nil)
	if resp, err := http.DefaultClient.Do(req); err != nil || resp.StatusCode != 200 {
		t.Fatalf("resource metadata %s: %v %v", md, err, resp)
	}
}

func TestA2ACardOpenUnderConnectOnly(t *testing.T) {
	hs := a2aGatedHTTP(t, Config{})
	for _, path := range []string{"/.well-known/agent-card.json", "/.well-known/agent.json"} {
		var card map[string]any
		getJSON(t, hs.URL+path, &card)
		if card["protocolVersion"] != "0.3.0" || len(card["skills"].([]any)) != 1 {
			t.Fatalf("%s without a token: %#v", path, card)
		}
	}
}

func TestA2AGatedStreamAndTasksGet(t *testing.T) {
	hs := a2aGatedHTTP(t, Config{})
	for _, method := range []string{"message/stream", "tasks/get", "tasks/send"} {
		st, _, _ := rpcWith(t, hs.URL+"/a2a/", "", nil, method, map[string]any{"id": "x"})
		if st != http.StatusUnauthorized {
			t.Fatalf("%s without a token: %d, want 401", method, st)
		}
	}
	st, h, body := rpcWith(t, hs.URL+"/a2a/", "tok-bob", nil, "message/stream", a2aUserMessage("go"))
	if st != 200 || h.Get("Content-Type") != "text/event-stream" || !strings.Contains(body, `"text":"GO"`) {
		t.Fatalf("signed-in stream: %d %q %s", st, h.Get("Content-Type"), body)
	}
}

func TestA2AGatedSubjectRateLimit(t *testing.T) {
	hs := a2aGatedHTTP(t, Config{MCPSubjectRPM: 1, MCPSubjectBurst: 2})
	for i := 0; i < 2; i++ {
		if st, _, body := a2aSendAs(t, hs.URL, "tok-u1", "a"); st != 200 {
			t.Fatalf("u1 message %d: %d %s", i+1, st, body)
		}
	}
	if st, h, _ := a2aSendAs(t, hs.URL, "tok-u1", "a"); st != http.StatusTooManyRequests || h.Get("Retry-After") == "" {
		t.Fatalf("u1 over the limit: %d Retry-After=%q, want 429", st, h.Get("Retry-After"))
	}
	if st, _, _ := a2aSendAs(t, hs.URL, "tok-u2", "a"); st != 200 {
		t.Fatalf("u2 (independent subject): %d", st)
	}
}

func TestA2AHandlerMustBeHiddenUnderConnectOnly(t *testing.T) {
	cfg := Config{MCP: true, MCPConnectOnly: true, A2A: true, A2AMessage: "ask"}
	srv, err := identityServer(t, "agent", a2aUngatedHandlerModule, cfg)
	if err != nil {
		t.Fatalf("LoadModules: %v", err)
	}
	if err := srv.ValidateMCPConnectOnly(); err != nil {
		t.Fatalf("--mcp-connect-only with --a2a is allowed now that /a2a/ is gated: %v", err)
	}
	err = srv.ValidateA2A()
	if err == nil || !strings.Contains(err.Error(), "@noexpose @nomcp") {
		t.Fatalf("ValidateA2A = %v, want a refusal naming @noexpose @nomcp", err)
	}
}

// Without --mcp-connect-only nothing changes: /a2a/ is mounted as before.
func TestA2AUngatedWithoutConnectOnly(t *testing.T) {
	srv, err := identityServer(t, "agent", a2aGatedModule, Config{MCP: true, A2A: true, A2AMessage: "ask"})
	if err != nil {
		t.Fatalf("LoadModules: %v", err)
	}
	if err := srv.ValidateA2A(); err != nil {
		t.Fatal(err)
	}
	prev := log.Writer()
	log.SetOutput(io.Discard)
	hs := httptest.NewServer(srv.buildRoutes())
	log.SetOutput(prev)
	t.Cleanup(hs.Close)
	resp, err := http.Post(hs.URL+"/a2a/", "application/json", bytes.NewReader([]byte(`{"jsonrpc":"2.0","id":1,"method":"message/send","params":{"message":{"role":"user","parts":[{"kind":"text","text":"x"}]}}}`)))
	if err != nil || resp.StatusCode != 200 {
		t.Fatalf("ungated /a2a/: %v %v", err, resp)
	}
}
