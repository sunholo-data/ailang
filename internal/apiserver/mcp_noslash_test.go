package apiserver

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

// claude.ai's connector check POSTs the listed URL without its trailing slash
// and does not follow redirects (2026-10-07, prod logs: python-httpx POST
// /mcp/connect -> 307, "Couldn't determine how this server signs in"). Both
// forms must answer directly, and a client that used the no-slash form must
// be pointed at metadata whose resource is the URL it used (RFC 9728 §3.3).

func noRedirectClient() *http.Client {
	return &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
}

func postJSONRPC(t *testing.T, c *http.Client, url, body string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	resp, err := c.Do(req)
	if err != nil {
		t.Fatalf("POST %s: %v", url, err)
	}
	t.Cleanup(func() { resp.Body.Close() })
	return resp
}

const initBody = `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"t","version":"1"}}}`
const gatedCall = `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"parseDoc","arguments":{"doc":"x"}}}`

func TestListedSurface_NoTrailingSlash(t *testing.T) {
	as := authServer(t, true)
	hs := readyServer(t, as.URL)
	c := noRedirectClient()

	for _, path := range []string{"/mcp/connect", "/mcp/connect/", "/mcp", "/mcp/"} {
		if resp := postJSONRPC(t, c, hs.URL+path, initBody); resp.StatusCode != http.StatusOK {
			t.Errorf("initialize on %s = %d (location %q), want 200", path, resp.StatusCode, resp.Header.Get("Location"))
		}
	}

	for _, tc := range []struct{ path, metaSuffix, resource string }{
		{"/mcp/connect", "/.well-known/oauth-protected-resource/mcp/connect", "/mcp/connect"},
		{"/mcp/connect/", "/.well-known/oauth-protected-resource/mcp/connect/", "/mcp/connect/"},
	} {
		resp := postJSONRPC(t, c, hs.URL+tc.path, gatedCall)
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("gated call on %s without a token = %d, want 401", tc.path, resp.StatusCode)
		}
		wa := resp.Header.Get("WWW-Authenticate")
		wantMeta := hs.URL + tc.metaSuffix
		if !strings.Contains(wa, `resource_metadata="`+wantMeta+`"`) {
			t.Errorf("401 on %s: WWW-Authenticate = %q, want resource_metadata=%q", tc.path, wa, wantMeta)
		}
		mresp, err := c.Get(wantMeta)
		if err != nil {
			t.Fatal(err)
		}
		raw, _ := io.ReadAll(mresp.Body)
		mresp.Body.Close()
		if mresp.StatusCode != http.StatusOK {
			t.Fatalf("GET %s = %d, want 200", wantMeta, mresp.StatusCode)
		}
		var prm struct {
			Resource string `json:"resource"`
		}
		if err := json.Unmarshal(raw, &prm); err != nil {
			t.Fatalf("metadata %s: %v (%s)", wantMeta, err, raw)
		}
		if prm.Resource != hs.URL+tc.resource {
			t.Errorf("metadata at %s: resource = %q, want %q (the URL the client used)", wantMeta, prm.Resource, hs.URL+tc.resource)
		}
	}
}

func TestMCPCheck_ListedSurfaceReadyWithoutSlash(t *testing.T) {
	as := authServer(t, true)
	hs := readyServer(t, as.URL)
	fs := runCheck(t, hs.URL+"/mcp/connect", "anthropic")
	for _, c := range []string{"annotations", "credentials", "zero-arg", "lazy-auth", "authorization-server"} {
		if got := statusOf(fs, c); got != "PASS" {
			t.Errorf("%s = %s on /mcp/connect, want PASS; findings: %+v", c, got, fs)
		}
	}
}
