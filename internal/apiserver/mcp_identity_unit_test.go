package apiserver

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// D5: only (string) -> bool and (string) -> Option[identity record] are
// verifiers. Any other Option payload is a bad signature, so no gated tool
// registers behind it.
func TestIdentityVerifierSignature(t *testing.T) {
	head := "module test/api/sig\n\nimport std/option (Option, Some, None)\n"
	cases := []struct {
		name, ret         string
		wantOK, wantIdent bool
	}{
		{"bool", "bool", true, false},
		{"inline identity", "Option[{ subject: string, client_id: string, scopes: [string] }]", true, true},
		{"option string", "Option[string]", false, false},
		{"missing scopes", "Option[{ subject: string, client_id: string }]", false, false},
		{"open record", "Option[{ subject: string, client_id: string, scopes: [string] | r }]", false, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			body := "false"
			if c.ret != "bool" {
				body = "None"
			}
			src := head + "@mcp_token_verifier\nexport func verifyToken(token: string) -> " + c.ret + " ! {IO} = " + body + "\n"
			srv, err := identityServer(t, "sig", src, Config{MCP: true})
			if err != nil {
				t.Skipf("does not type-check, so never a verifier: %v", err)
			}
			v := findExport(t, srv, "verifyToken")
			if v.VerifierSigOK != c.wantOK || v.VerifierIdentity != c.wantIdent {
				t.Fatalf("sigOK=%v identity=%v, want %v %v", v.VerifierSigOK, v.VerifierIdentity, c.wantOK, c.wantIdent)
			}
			_, err = resolveTokenVerifier(srv.GetModules())
			if c.wantOK != (err == nil) {
				t.Fatalf("resolveTokenVerifier err = %v", err)
			}
			if err != nil && !strings.Contains(err.Error(), "Option[{subject: string, client_id: string, scopes: [string]}]") {
				t.Errorf("error does not name the identity shape: %v", err)
			}
		})
	}
}

// The tool handler never binds an empty identity: no stamped header, a
// malformed one, or an empty subject fails the call.
func TestIdentityArgFailsClosed(t *testing.T) {
	for name, h := range map[string]http.Header{
		"no extra":      nil,
		"no header":     {},
		"malformed":     {identityHeader: {"{not json"}},
		"empty subject": {identityHeader: {`{"subject":"","client_id":"c","scopes":[]}`}},
	} {
		req := &mcp.CallToolRequest{}
		if h != nil {
			req.Extra = &mcp.RequestExtra{Header: h}
		}
		if id, err := identityArg(req); err == nil {
			t.Errorf("%s: bound %v", name, id)
		}
	}
	req := &mcp.CallToolRequest{Extra: &mcp.RequestExtra{Header: http.Header{identityHeader: {`{"subject":"u1","client_id":"c","scopes":["a"]}`}}}}
	id, err := identityArg(req)
	if err != nil || id["subject"] != "u1" || id["client_id"] != "c" {
		t.Fatalf("valid header: %v %v", id, err)
	}
}

// --routes-only (the executor's flag set) still serves an _auth tool on the
// gated listed surface: declaring _auth is as explicit as @route. It stays
// off HTTP, and undeclared helpers stay off everything.
func TestAuthServedUnderRoutesOnly(t *testing.T) {
	hs, _ := identityHTTP(t, Config{RoutesOnly: true, MCPConnectOnly: true})
	listed := hs.URL + listedMCPPath
	if st, _, body := rpc(t, listed, "tok-u1", "tools/call", call("whoami", map[string]any{"note": "n"})); st != 200 || !strings.Contains(body, "u1|") {
		t.Fatalf("whoami under --routes-only: %d %s", st, body)
	}
	if _, _, list := rpc(t, listed, "", "tools/list", map[string]any{}); strings.Contains(list, `"openTool"`) || !strings.Contains(list, `"whoami"`) {
		t.Fatalf("--routes-only listing: %s", list)
	}
	if st, _ := httpDo(t, http.MethodPost, hs.URL+"/api/test/api/ident/whoami", `{"note":"n"}`); st == 200 {
		t.Fatalf("REST whoami under --routes-only: %d", st)
	}
}

// A JSON-RPC batch spends one token per gated call, so it cannot multiply a
// subject's allowance: three calls against a burst of two are refused.
func TestSubjectRateLimitBatch(t *testing.T) {
	hs, _ := identityHTTP(t, Config{MCPSubjectRPM: 1, MCPSubjectBurst: 2})
	one := `{"jsonrpc":"2.0","id":%d,"method":"tools/call","params":{"name":"whoami","arguments":{"note":"n"}}}`
	batch := "[" + fmt.Sprintf(one, 1) + "," + fmt.Sprintf(one, 2) + "," + fmt.Sprintf(one, 3) + "]"
	req, _ := http.NewRequest(http.MethodPost, hs.URL+listedMCPPath, strings.NewReader(batch))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("Authorization", "Bearer tok-u1")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("batch of 3 against burst 2: %d, want 429", resp.StatusCode)
	}
}
