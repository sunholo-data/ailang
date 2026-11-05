package apiserver

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/effects"
)

// M-MCP-OAUTH-CLIENT-MODELS M2 (D5, D6, T17): an identity-returning
// @mcp_token_verifier, the _auth param bound only from it, and the
// per-subject limit, end to end through buildRoutes.

const identVerifier = `
type McpIdentity = { subject: string, client_id: string, scopes: [string] }

-- "tok-<s>" signs in subject <s>; "empty" returns Some with an empty subject;
-- anything else is None.
@mcp_token_verifier
export func verifyToken(token: string) -> Option[McpIdentity] ! {IO} =
  if token == "empty" then Some({ subject: "", client_id: "c", scopes: [] })
  else if startsWith(token, "tok-") then Some({ subject: substring(token, 4, length(token)), client_id: "https://client.example/cimd", scopes: ["read"] })
  else None
`

const identModule = `module test/api/ident

import std/json (Json, encode)
import std/option (Option, Some, None)
import std/string (startsWith, substring, length)
` + identVerifier + `
@mcp_auth("oauth2")
export func whoami(note: string, _auth: McpIdentity) -> string ! {IO} = "${_auth.subject}|${_auth.client_id}|${note}"

@mcp_auth("oauth2")
export func hdrs(_headers: Json) -> string ! {IO} = encode(_headers)

export func openTool(x: string) -> string ! {IO} = "open:${x}"
`

func writeIdentModule(t *testing.T, name, src string) (string, string) {
	t.Helper()
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
	modPath := filepath.Join(apiDir, name+".ail")
	if err := os.WriteFile(modPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	return tmpDir, modPath
}

// identityServer loads src as test/api/<name> and returns the server (not
// started) after LoadModules; loadErr is LoadModules' error.
func identityServer(t *testing.T, name, src string, cfg Config) (*Server, error) {
	t.Helper()
	tmpDir, modPath := writeIdentModule(t, name, src)
	eff := effects.NewEffContext(nil)
	eff.Grant(effects.NewCapability("IO"))
	cfg.Port, cfg.EffCtx = "0", eff
	if cfg.OAuthIssuer == "" {
		cfg.OAuthIssuer = gateIssuer
	}
	srv := New(tmpDir, cfg)
	t.Cleanup(func() { srv.Close() })
	return srv, srv.LoadModules([]string{modPath})
}

// identityHTTP is identityServer plus Start's identity validation and the
// real route table, logs captured.
func identityHTTP(t *testing.T, cfg Config) (*httptest.Server, string) {
	t.Helper()
	cfg.MCP = true
	srv, err := identityServer(t, "ident", identModule, cfg)
	if err != nil {
		t.Fatalf("LoadModules: %v", err)
	}
	if err := srv.ValidateMCPIdentity(); err != nil {
		t.Fatalf("ValidateMCPIdentity: %v", err)
	}
	var logs bytes.Buffer
	prev := log.Writer()
	log.SetOutput(&logs)
	mux := srv.buildRoutes()
	log.SetOutput(prev)
	hs := httptest.NewServer(mux)
	t.Cleanup(hs.Close)
	return hs, logs.String()
}

// rpcWith posts one JSON-RPC message with extra request headers.
func rpcWith(t *testing.T, url, token string, headers map[string]string, method string, params any) (int, http.Header, string) {
	t.Helper()
	msg, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": method, "params": params})
	req, _ := http.NewRequest(http.MethodPost, url, bytes.NewReader(msg))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, resp.Header, string(b)
}

const forgedIdentity = `{"subject":"admin","client_id":"forged","scopes":["all"]}`

// The verified subject reaches _auth on a gated /mcp/connect/ call.
func TestAuthBoundOnListedSurface(t *testing.T) {
	hs, _ := identityHTTP(t, Config{})
	listed := hs.URL + listedMCPPath
	st, _, body := rpc(t, listed, "tok-u1", "tools/call", call("whoami", map[string]any{"note": "n"}))
	if st != 200 || !strings.Contains(body, `u1|https://client.example/cimd|n`) {
		t.Fatalf("whoami: %d %s", st, body)
	}
	// Without a token the gate answers before the tool: _auth is never zero.
	if st, _, body := rpc(t, listed, "", "tools/call", call("whoami", map[string]any{"note": "n"})); st != 401 {
		t.Fatalf("no token: %d %s", st, body)
	}
}

// T17: an _auth value in the arguments, named or positional, is overwritten.
func TestAuthParamNotForgeableByArgs(t *testing.T) {
	hs, _ := identityHTTP(t, Config{})
	listed := hs.URL + listedMCPPath
	var forged map[string]any
	_ = json.Unmarshal([]byte(forgedIdentity), &forged)
	for name, args := range map[string]map[string]any{
		"named":      {"note": "n", "_auth": forged},
		"positional": {"args": []any{"n", forged}},
	} {
		st, _, body := rpc(t, listed, "tok-u1", "tools/call", call("whoami", args))
		if st != 200 || !strings.Contains(body, "u1|https://client.example/cimd|n") || strings.Contains(body, "admin") {
			t.Errorf("%s: %d %s", name, st, body)
		}
	}
}

// T17: a client-sent X-Ailang-Mcp-Identity never decides _auth and never
// shows in _headers; it is deleted before the gate runs.
func TestIdentityHeaderStripped(t *testing.T) {
	t.Run("entry strip", func(t *testing.T) {
		var seen []string
		h := withoutIdentityHeader(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			seen = r.Header.Values(identityHeader)
		}))
		r := httptest.NewRequest(http.MethodPost, "/mcp/connect/", nil)
		r.Header.Set(identityHeader, forgedIdentity)
		h.ServeHTTP(httptest.NewRecorder(), r)
		if len(seen) != 0 {
			t.Fatalf("client identity header reached the handler: %v", seen)
		}
	})
	hs, _ := identityHTTP(t, Config{})
	forged := map[string]string{identityHeader: forgedIdentity}
	for _, url := range []string{hs.URL + listedMCPPath, hs.URL + "/mcp/"} {
		st, _, body := rpcWith(t, url, "tok-u1", forged, "tools/call", call("hdrs", map[string]any{}))
		if strings.HasSuffix(url, listedMCPPath) && st != 200 {
			t.Fatalf("%s hdrs: %d %s", url, st, body)
		}
		if strings.Contains(strings.ToLower(body), "x-ailang-mcp-identity") || strings.Contains(body, "forged") {
			t.Errorf("%s: identity header visible in _headers: %s", url, body)
		}
	}
	st, _, body := rpcWith(t, hs.URL+listedMCPPath, "tok-u1", forged, "tools/call", call("whoami", map[string]any{"note": "n"}))
	if st != 200 || !strings.Contains(body, "u1|") || strings.Contains(body, "admin") {
		t.Fatalf("whoami with forged header: %d %s", st, body)
	}
}

// An _auth export is served only on the gated listed surface: absent from
// /mcp/ (and stdio, which is the same agent server), with a startup line.
// Tools without _auth list exactly as before, and _auth is never in a schema.
func TestAuthExportAbsentFromAgentSurface(t *testing.T) {
	hs, logs := identityHTTP(t, Config{})
	agent := hs.URL + "/mcp/"
	_, _, list := rpc(t, agent, "", "tools/list", map[string]any{})
	if strings.Contains(list, `"whoami"`) || !strings.Contains(list, `"openTool"`) {
		t.Fatalf("/mcp/ tools/list: %s", list)
	}
	if _, _, body := rpc(t, agent, "tok-u1", "tools/call", call("whoami", map[string]any{"note": "n"})); strings.Contains(body, "u1|") {
		t.Fatalf("/mcp/ ran whoami: %s", body)
	}
	if !strings.Contains(logs, "whoami takes _auth and is served only on the gated /mcp/connect/") {
		t.Errorf("no startup line for the omitted export; logs:\n%s", logs)
	}

	tools, _, _ := listToolsFromSource(t, identModule, Config{OAuthIssuer: gateIssuer})
	if _, ok := tools["whoami"]; ok {
		t.Error("stdio/agent server lists whoami")
	}

	_, _, listed := rpc(t, hs.URL+listedMCPPath, "", "tools/list", map[string]any{})
	var resp struct {
		Result struct {
			Tools []struct {
				Name        string         `json:"name"`
				InputSchema map[string]any `json:"inputSchema"`
			} `json:"tools"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(sseData(listed)), &resp); err != nil {
		t.Fatalf("listed tools/list: %v: %s", err, listed)
	}
	found := false
	for _, tl := range resp.Result.Tools {
		if tl.Name != "whoami" {
			continue
		}
		found = true
		props, _ := tl.InputSchema["properties"].(map[string]any)
		if _, has := props["_auth"]; has || len(props) != 1 || props["note"] == nil {
			t.Errorf("whoami inputSchema: %v", tl.InputSchema)
		}
		if req, _ := tl.InputSchema["required"].([]any); len(req) != 1 || req[0] != "note" {
			t.Errorf("whoami required: %v", tl.InputSchema["required"])
		}
	}
	if !found {
		t.Fatalf("listed surface lacks whoami: %s", listed)
	}
}

// sseData returns the JSON of a streamable-HTTP response (plain or SSE).
func sseData(body string) string {
	for _, line := range strings.Split(body, "\n") {
		if strings.HasPrefix(line, "data: ") {
			return strings.TrimPrefix(line, "data: ")
		}
	}
	return body
}

// No HTTP path binds _auth: no /api/ endpoint, no OpenAPI operation, no A2A
// skill.
func TestAuthExportNoRESTRoute(t *testing.T) {
	hs, _ := identityHTTP(t, Config{A2A: true})
	if st, body := httpDo(t, http.MethodPost, hs.URL+"/api/test/api/ident/whoami", `{"note":"n"}`); st == 200 || strings.Contains(body, "|n") {
		t.Fatalf("REST whoami: %d %s", st, body)
	}
	if st, body := httpDo(t, http.MethodPost, hs.URL+"/api/test/api/ident/openTool", `{"x":"a"}`); st != 200 || !strings.Contains(body, "open:a") {
		t.Fatalf("control: REST openTool: %d %s", st, body)
	}
	for _, path := range []string{"/api/_meta/openapi.json", "/.well-known/agent.json"} {
		st, body := httpDo(t, http.MethodGet, hs.URL+path, "")
		if st != 200 || strings.Contains(body, "whoami") || !strings.Contains(body, "openTool") {
			t.Errorf("%s: %d, whoami present or control missing: %s", path, st, body)
		}
	}
}

// Every load-time rule refuses the module with an error naming it.
func TestAuthParamLoadErrors(t *testing.T) {
	head := "module test/api/bad\n\nimport std/option (Option, Some, None)\nimport std/string (startsWith, substring, length)\n" + identVerifier
	cases := []struct{ name, fn, want string }{
		{"noauth", `
@mcp_auth("noauth")
export func t1(_auth: McpIdentity) -> string ! {IO} = _auth.subject`, `_auth with @mcp_auth("noauth")`},
		{"not a record", `
@mcp_auth("oauth2")
export func t1(_auth: string) -> string ! {IO} = _auth`, "_auth: the type is not a record"},
		{"missing field", `
@mcp_auth("oauth2")
export func t1(_auth: { subject: string, client_id: string }) -> string ! {IO} = _auth.subject`, "the record lacks scopes"},
		{"extra field", `
@mcp_auth("oauth2")
export func t1(_auth: { subject: string, client_id: string, scopes: [string], admin: bool }) -> string ! {IO} = _auth.subject`, "unexpected field admin"},
		{"scopes type", `
@mcp_auth("oauth2")
export func t1(_auth: { subject: string, client_id: string, scopes: string }) -> string ! {IO} = _auth.subject`, "field scopes must be [string]"},
		{"optional", `
@mcp_auth("oauth2")
@optional("_auth")
export func t1(_auth: McpIdentity) -> string ! {IO} = _auth.subject`, `@optional("_auth")`},
		{"secret", `
@mcp_auth("oauth2")
@mcp_secret("_auth")
export func t1(_auth: McpIdentity) -> string ! {IO} = _auth.subject`, `@mcp_secret("_auth")`},
		{"route", `
@mcp_auth("oauth2")
@route("POST", "/api/who")
export func t1(_auth: McpIdentity) -> string ! {IO} = _auth.subject`, "_auth with @route"},
		{"nomcp", `
@nomcp
export func t1(_auth: McpIdentity) -> string ! {IO} = _auth.subject`, "_auth with @nomcp"},
		{"agent only", `
@mcp_auth("oauth2")
@mcp_agent_only
export func t1(_auth: McpIdentity) -> string ! {IO} = _auth.subject`, "_auth with @mcp_agent_only"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := identityServer(t, "bad", head+c.fn+"\n", Config{MCP: true})
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("LoadModules error = %v, want it to contain %q", err, c.want)
			}
		})
	}
}

// TestAuthWithNoauthIsError is the plan's named case of the table above.
func TestAuthWithNoauthIsError(t *testing.T) {
	src := "module test/api/bad\n\nimport std/option (Option, Some, None)\nimport std/string (startsWith, substring, length)\n" + identVerifier + `
@mcp_auth("noauth")
export func t1(_auth: McpIdentity) -> string ! {IO} = _auth.subject
`
	if _, err := identityServer(t, "bad", src, Config{MCP: true}); err == nil || !strings.Contains(err.Error(), "served only as a gated tool") {
		t.Fatalf("LoadModules error = %v", err)
	}
}

// Rules that depend on flags or on the verifier: refused at Start, and the
// tool is not registered if a surface is built anyway. Under
// --mcp-connect-only an unannotated _auth tool is gated, so it is valid.
func TestAuthStartupRules(t *testing.T) {
	boolVerifier := `
type McpIdentity = { subject: string, client_id: string, scopes: [string] }
@mcp_token_verifier
export func verifyToken(token: string) -> bool ! {IO} = token == "good"
`
	cases := []struct {
		name, src string
		cfg       Config
		want      string // "" = valid
	}{
		{"bool verifier", boolVerifier + `
@mcp_auth("oauth2")
export func t1(_auth: McpIdentity) -> string ! {IO} = _auth.subject`, Config{}, "returns bool, which names no subject"},
		{"not gated", identVerifier + `
export func t1(_auth: McpIdentity) -> string ! {IO} = _auth.subject`, Config{}, "_auth needs a gated tool"},
		{"no verifier", `
type McpIdentity = { subject: string, client_id: string, scopes: [string] }
@mcp_auth("oauth2")
export func t1(_auth: McpIdentity) -> string ! {IO} = _auth.subject`, Config{}, "_auth needs one @mcp_token_verifier"},
		{"connect-only gates it", identVerifier + `
export func t1(_auth: McpIdentity) -> string ! {IO} = _auth.subject`, Config{MCPConnectOnly: true}, ""},
	}
	head := "module test/api/authmod\n\nimport std/option (Option, Some, None)\nimport std/string (startsWith, substring, length)\n"
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			c.cfg.MCP = true
			srv, err := identityServer(t, "authmod", head+c.src+"\n", c.cfg)
			if err != nil {
				t.Fatalf("LoadModules: %v", err)
			}
			err = srv.ValidateMCPIdentity()
			if c.want == "" {
				if err != nil {
					t.Fatalf("valid setup refused: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("ValidateMCPIdentity = %v, want %q", err, c.want)
			}
			var logs bytes.Buffer
			prev := log.Writer()
			log.SetOutput(&logs)
			listed := NewListedMCPServer(srv)
			log.SetOutput(prev)
			// The registration ERROR may name an earlier rule (validateMCPAuth's
			// "needs one @mcp_token_verifier"); either way t1 is not served.
			if listed.gated["t1"] || !strings.Contains(logs.String(), "ERROR: skipping MCP tool registration for test/api/authmod/t1") {
				t.Errorf("listed surface registered t1 or logged no ERROR; logs:\n%s", logs.String())
			}
		})
	}
}

// D5: Some with an empty subject is a rejected token (401), not an identity.
func TestEmptySubjectRejected(t *testing.T) {
	hs, _ := identityHTTP(t, Config{})
	for _, tool := range []string{"whoami", "hdrs"} {
		st, h, body := rpc(t, hs.URL+listedMCPPath, "empty", "tools/call", call(tool, map[string]any{"note": "n"}))
		if st != 401 || !strings.Contains(h.Get("WWW-Authenticate"), `error="invalid_token"`) {
			t.Errorf("%s with empty subject: %d %q %s", tool, st, h.Get("WWW-Authenticate"), body)
		}
	}
	if st, _, _ := rpc(t, hs.URL+listedMCPPath, "nobody", "tools/call", call("whoami", map[string]any{"note": "n"})); st != 401 {
		t.Errorf("None: %d, want 401", st)
	}
}

// A bool verifier keeps its contract: its gated tool runs with a good token,
// and no identity header is stamped for it.
func TestBoolVerifierUnchanged(t *testing.T) {
	src := `module test/api/boolv

import std/json (Json, encode)

@mcp_token_verifier
export func verifyToken(token: string) -> bool ! {IO} = token == "good"

@mcp_auth("oauth2")
export func hdrs(_headers: Json) -> string ! {IO} = encode(_headers)
`
	srv, err := identityServer(t, "boolv", src, Config{MCP: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := srv.ValidateMCPIdentity(); err != nil {
		t.Fatalf("bool verifier refused: %v", err)
	}
	v := findExport(t, srv, "verifyToken")
	if !v.VerifierSigOK || v.VerifierIdentity {
		t.Fatalf("bool verifier flags: sigOK=%v identity=%v", v.VerifierSigOK, v.VerifierIdentity)
	}
	hs := httptest.NewServer(srv.buildRoutes())
	t.Cleanup(hs.Close)
	if st, _, body := rpc(t, hs.URL+listedMCPPath, "good", "tools/call", call("hdrs", map[string]any{})); st != 200 || strings.Contains(strings.ToLower(body), "x-ailang-mcp-identity") {
		t.Fatalf("bool-gated tool: %d %s", st, body)
	}
	if st, _, _ := rpc(t, hs.URL+listedMCPPath, "bad", "tools/call", call("hdrs", map[string]any{})); st != 401 {
		t.Fatalf("bad token: %d", st)
	}
}

// D6: the limit is per verified subject. u1's third call inside the burst is
// a 429 with Retry-After and the tool does not run; u2 is untouched.
func TestSubjectRateLimit429(t *testing.T) {
	hs, _ := identityHTTP(t, Config{MCPSubjectRPM: 1, MCPSubjectBurst: 2})
	listed := hs.URL + listedMCPPath
	who := func(token string) (int, http.Header, string) {
		return rpc(t, listed, token, "tools/call", call("whoami", map[string]any{"note": "n"}))
	}
	for i := 0; i < 2; i++ {
		if st, _, body := who("tok-u1"); st != 200 || !strings.Contains(body, "u1|") {
			t.Fatalf("u1 call %d: %d %s", i+1, st, body)
		}
	}
	st, h, body := who("tok-u1")
	if st != 429 || h.Get("Retry-After") == "" || h.Get("Retry-After") == "0" || strings.Contains(body, "u1|") {
		t.Fatalf("u1 over the limit: %d Retry-After=%q %s", st, h.Get("Retry-After"), body)
	}
	for i := 0; i < 2; i++ {
		if st, _, body := who("tok-u2"); st != 200 || !strings.Contains(body, "u2|") {
			t.Fatalf("u2 call %d (independent subject): %d %s", i+1, st, body)
		}
	}
	// Open tools on the listed surface are not limited.
	if st, _, _ := rpc(t, listed, "", "tools/call", call("openTool", map[string]any{"x": "a"})); st != 200 {
		t.Errorf("open tool: %d", st)
	}
}

// The limit needs a subject: with a bool verifier, or without --mcp-http, the
// server refuses to start rather than run with the limit silently off.
func TestSubjectRateLimitNeedsIdentity(t *testing.T) {
	boolSrc := `module test/api/boolv

@mcp_token_verifier
export func verifyToken(token: string) -> bool ! {IO} = token == "good"

@mcp_auth("oauth2")
export func g(x: string) -> string ! {IO} = x
`
	srv, err := identityServer(t, "boolv", boolSrc, Config{MCP: true, MCPSubjectRPM: 10})
	if err != nil {
		t.Fatal(err)
	}
	if err := srv.ValidateMCPIdentity(); err == nil || !strings.Contains(err.Error(), "a bool verifier names no subject") {
		t.Fatalf("bool verifier + subject limit: %v", err)
	}
	srv, err = identityServer(t, "ident", identModule, Config{MCPSubjectRPM: 10})
	if err != nil {
		t.Fatal(err)
	}
	if err := srv.ValidateMCPIdentity(); err == nil || !strings.Contains(err.Error(), "needs --mcp-http") {
		t.Fatalf("no --mcp-http: %v", err)
	}
	srv, err = identityServer(t, "ident", identModule, Config{MCP: true, MCPSubjectBurst: 3})
	if err != nil {
		t.Fatal(err)
	}
	if err := srv.ValidateMCPIdentity(); err == nil || !strings.Contains(err.Error(), "--mcp-subject-burst needs --mcp-subject-rpm") {
		t.Fatalf("burst alone: %v", err)
	}
}

// The verifier's return is read strictly: None, or Some(identity record).
func TestDecodeIdentityOption(t *testing.T) {
	some := func(rec any) map[string]any {
		return map[string]any{"__type": "Option", "__tag": "Some", "fields": []any{rec}}
	}
	id, err := decodeIdentityOption(some(map[string]any{"subject": "u", "client_id": "c", "scopes": []any{"a"}}))
	if err != nil || id == nil || id.Subject != "u" || id.ClientID != "c" || len(id.Scopes) != 1 {
		t.Fatalf("Some: %+v %v", id, err)
	}
	if id, err := decodeIdentityOption(map[string]any{"__type": "Option", "__tag": "None"}); id != nil || err != nil {
		t.Fatalf("None: %+v %v", id, err)
	}
	for name, bad := range map[string]any{
		"bool":          true,
		"result":        map[string]any{"__type": "Result", "__tag": "Ok"},
		"subject int":   some(map[string]any{"subject": 1, "client_id": "c", "scopes": []any{}}),
		"scopes string": some(map[string]any{"subject": "u", "client_id": "c", "scopes": "a"}),
		"scope int":     some(map[string]any{"subject": "u", "client_id": "c", "scopes": []any{1}}),
	} {
		if _, err := decodeIdentityOption(bad); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}
