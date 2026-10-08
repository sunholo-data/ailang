package apiserver

import (
	"bytes"
	"encoding/json"
	"flag"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// M-MCP-FILE-HANDOFF F1b: @mcp_file (openai/fileParams), securitySchemes and
// _meta["mcp/www_authenticate"] on refused calls, end to end through
// buildRoutes.

var updateMCPGolden = flag.Bool("update-mcp-golden", false, "rewrite testdata/mcp_file_tools_list.golden.json")

const fileModule = `module test/api/files

type OpenAIFile = {download_url: string, file_id: string, mime_type: string, file_name: string}

@mcp_token_verifier
export func verifyToken(token: string) -> bool ! {IO} = token == "good"

-- Parse a file the host hands over.
@mcp_title("Parse file")
@mcp_hints("readOnly", "openWorld")
@mcp_auth("oauth2")
@mcp_file("file")
export func parseFile(file: OpenAIFile, mode: string) -> string ! {IO} = "${file.file_id}|${file.download_url}|${file.mime_type}|${file.file_name}|${mode}"

-- Echo a file, if one was given.
@mcp_title("Echo file")
@mcp_hints("readOnly")
@mcp_file("doc")
@optional("doc")
export func echoFile(doc: {download_url: string, file_id: string, mime_type: string, file_name: string}) -> string = "${doc.file_id}|${doc.download_url}|${doc.mime_type}|${doc.file_name}"

-- No file param, no auth: unchanged on /mcp/.
@mcp_title("Plain")
@mcp_hints("readOnly")
export func plain(x: string) -> string = x
`

// writeModule writes src as test/api/<name>.ail under a fresh base path.
func writeModule(t *testing.T, name, src string) (string, string) {
	t.Helper()
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

func fileServer(t *testing.T) *httptest.Server {
	t.Helper()
	return fileServerFor(t, gateIssuer)
}

func fileServerFor(t *testing.T, issuer string) *httptest.Server {
	t.Helper()
	tmpDir, modPath := writeModule(t, "files", fileModule)
	srv := New(tmpDir, Config{Port: "0", MCP: true, OAuthIssuer: issuer, NoFeedbackTool: true})
	t.Cleanup(func() { srv.Close() })
	if err := srv.LoadModules([]string{modPath}); err != nil {
		t.Fatalf("LoadModules: %v", err)
	}
	hs := httptest.NewServer(srv.buildRoutes())
	t.Cleanup(hs.Close)
	return hs
}

// rpcMessage decodes the one JSON-RPC message in a plain-JSON or SSE body.
func rpcMessage(t *testing.T, body string) map[string]any {
	t.Helper()
	payload := strings.TrimSpace(body)
	for _, line := range strings.Split(body, "\n") {
		if strings.HasPrefix(line, "data: ") {
			payload = strings.TrimPrefix(line, "data: ")
		}
	}
	var msg map[string]any
	if err := json.Unmarshal([]byte(payload), &msg); err != nil {
		t.Fatalf("not a JSON-RPC message: %v\n%s", err, body)
	}
	return msg
}

func listedTools(t *testing.T, url string) []any {
	t.Helper()
	st, _, body := rpc(t, url, "", "tools/list", map[string]any{})
	if st != 200 {
		t.Fatalf("tools/list: %d %s", st, body)
	}
	res, _ := rpcMessage(t, body)["result"].(map[string]any)
	tools, _ := res["tools"].([]any)
	sort.Slice(tools, func(i, j int) bool {
		return tools[i].(map[string]any)["name"].(string) < tools[j].(map[string]any)["name"].(string)
	})
	return tools
}

func toolNamed(tools []any, name string) map[string]any {
	for _, tl := range tools {
		if m := tl.(map[string]any); m["name"] == name {
			return m
		}
	}
	return nil
}

// TestMCPFile_ListedGolden pins the listed surface's tools/list: the file
// param's schema is OpenAI's documented four-property object, the tool's
// _meta names it, and every tool declares securitySchemes (top level and the
// _meta mirror). Regenerate with -update-mcp-golden.
func TestMCPFile_ListedGolden(t *testing.T) {
	hs := fileServer(t)
	got, err := json.MarshalIndent(listedTools(t, hs.URL+listedMCPPath), "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	got = append(got, '\n')
	golden := filepath.Join("testdata", "mcp_file_tools_list.golden.json")
	if *updateMCPGolden {
		if err := os.WriteFile(golden, got, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("tools/list drifted from %s:\n%s", golden, got)
	}
}

// TestMCPFile_SchemaMatchesOpenAI checks the rules OpenAI's Scan Tools
// enforces, independently of the golden file.
func TestMCPFile_SchemaMatchesOpenAI(t *testing.T) {
	hs := fileServer(t)
	for _, surface := range []string{"/mcp/", listedMCPPath} {
		tools := listedTools(t, hs.URL+surface)
		pf := toolNamed(tools, "parseFile")
		if pf == nil {
			t.Fatalf("%s: parseFile not listed", surface)
		}
		meta, _ := pf["_meta"].(map[string]any)
		if fp, _ := json.Marshal(meta["openai/fileParams"]); string(fp) != `["file"]` {
			t.Errorf("%s: _meta openai/fileParams = %s", surface, fp)
		}
		schema := pf["inputSchema"].(map[string]any)
		file, _ := json.Marshal(schema["properties"].(map[string]any)["file"])
		const want = `{"additionalProperties":false,"properties":{"download_url":{"type":"string"},"file_id":{"type":"string"},"file_name":{"type":"string"},"mime_type":{"type":"string"}},"required":["download_url","file_id"],"type":"object"}`
		if string(file) != want {
			t.Errorf("%s: file schema\n got %s\nwant %s", surface, file, want)
		}
		req, _ := json.Marshal(schema["required"])
		if string(req) != `["file","mode"]` {
			t.Errorf("%s: required = %s", surface, req)
		}
		// @optional file param: advertised, not required.
		ef := toolNamed(tools, "echoFile")
		if req, _ := json.Marshal(ef["inputSchema"].(map[string]any)["required"]); string(req) != `[]` {
			t.Errorf("%s: echoFile required = %s", surface, req)
		}
	}
	// The agent surface declares no auth schemes, and a tool with neither a
	// file param nor auth carries no _meta at all there.
	agent := listedTools(t, hs.URL+"/mcp/")
	if _, has := toolNamed(agent, "parseFile")["securitySchemes"]; has {
		t.Error("/mcp/ parseFile has securitySchemes")
	}
	if _, has := toolNamed(agent, "plain")["_meta"]; has {
		t.Error("/mcp/ plain tool gained _meta")
	}
}

// TestMCPFile_BindsFileObject: a tools/call with a file object reaches the
// AILANG function as the record; the optional fields default to "", extra
// fields are dropped, and a bad object is refused before the call.
func TestMCPFile_BindsFileObject(t *testing.T) {
	hs := fileServer(t)
	listed := hs.URL + listedMCPPath
	cases := []struct {
		name, url, token, tool string
		args                   map[string]any
		want                   string
	}{
		{"full object, gated, signed in", listed, "good", "parseFile", map[string]any{
			"file": map[string]any{"download_url": "https://files.example/x", "file_id": "file_1", "mime_type": "application/pdf", "file_name": "a.pdf", "extra": 1},
			"mode": "md",
		}, `file_1|https://files.example/x|application/pdf|a.pdf|md`},
		{"required fields only", hs.URL + "/mcp/", "", "parseFile", map[string]any{
			"file": map[string]any{"download_url": "https://files.example/y", "file_id": "file_2"},
			"mode": "md",
		}, `file_2|https://files.example/y|||md`},
		{"@optional file omitted", listed, "", "echoFile", map[string]any{}, `|||`},
		{"@optional file given", listed, "", "echoFile", map[string]any{
			"doc": map[string]any{"download_url": "u", "file_id": "f", "file_name": "n"},
		}, `f|u||n`},
		{"no file_id", listed, "", "echoFile", map[string]any{"doc": map[string]any{"download_url": "u"}}, `the file object has no file_id`},
		{"not an object", listed, "", "echoFile", map[string]any{"doc": "chat_upload://image_0"}, `got a string`},
		{"non-string name", listed, "", "echoFile", map[string]any{"doc": map[string]any{"download_url": "u", "file_id": "f", "file_name": 3}}, `file_name must be a string`},
		{"required file missing", listed, "good", "parseFile", map[string]any{"mode": "md"}, `missing required parameter(s): file`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			st, _, body := rpc(t, c.url, c.token, "tools/call", call(c.tool, c.args))
			if st != 200 {
				t.Fatalf("status %d: %s", st, body)
			}
			if !strings.Contains(body, c.want) {
				t.Fatalf("want %q in %s", c.want, body)
			}
		})
	}
}

// TestMCPGate_RefusalCarriesWWWAuthenticateMeta: a refused gated call keeps
// HTTP 401 + WWW-Authenticate (Claude) and its body is the JSON-RPC tool
// error ChatGPT reads, with the same challenge in _meta.
func TestMCPGate_RefusalCarriesWWWAuthenticateMeta(t *testing.T) {
	hs := fileServer(t)
	file := map[string]any{"download_url": "u", "file_id": "f"}
	for _, token := range []string{"", "bad"} {
		st, h, body := rpc(t, hs.URL+listedMCPPath, token, "tools/call", call("parseFile", map[string]any{"file": file, "mode": "m"}))
		challenge := h.Get("WWW-Authenticate")
		if st != 401 || !strings.HasPrefix(challenge, "Bearer resource_metadata=") {
			t.Fatalf("token %q: %d %q", token, st, challenge)
		}
		msg := rpcMessage(t, body)
		if id, _ := json.Marshal(msg["id"]); string(id) != "1" || msg["jsonrpc"] != "2.0" {
			t.Errorf("token %q: not a response to id 1: %s", token, body)
		}
		res, _ := msg["result"].(map[string]any)
		meta, _ := res["_meta"].(map[string]any)
		got, _ := json.Marshal(meta["mcp/www_authenticate"])
		want, _ := json.Marshal([]string{challenge})
		if res["isError"] != true || string(got) != string(want) {
			t.Errorf("token %q: result %s, want _meta challenge %s", token, body, want)
		}
	}
}

// TestMCPFile_LoadErrors: a bad @mcp_file is a load error naming the problem.
func TestMCPFile_LoadErrors(t *testing.T) {
	const rec = "{download_url: string, file_id: string, mime_type: string, file_name: string}"
	cases := []struct{ name, decl, want string }{
		{"no such param", `@mcp_file("fiel")
export func f(file: ` + rec + `) -> string = file.file_id`, `@mcp_file("fiel"): no such parameter (params: file)`},
		{"not a record", `@mcp_file("file")
export func f(file: string) -> string = file`, `@mcp_file("file"): the param is not a record`},
		{"missing field", `@mcp_file("file")
export func f(file: {download_url: string, file_id: string}) -> string = file.file_id`, `the record lacks mime_type, file_name`},
		{"extra field", `@mcp_file("file")
export func f(file: {download_url: string, file_id: string, mime_type: string, file_name: string, size: string}) -> string = file.file_id`, `fields OpenAI does not send: size`},
		{"non-string field", `@mcp_file("file")
export func f(file: {download_url: string, file_id: int, mime_type: string, file_name: string}) -> string = file.download_url`, `field file_id must be string`},
		{"secret", `@mcp_file("file")
@mcp_secret("file")
@optional("file")
export func f(file: ` + rec + `) -> string = file.file_id`, `cannot be both @mcp_file and @mcp_secret`},
		{"named twice", `@mcp_file("file")
@mcp_file("file")
export func f(file: ` + rec + `) -> string = file.file_id`, `named twice`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tmpDir, modPath := writeModule(t, "bad", "module test/api/bad\n\n"+c.decl+"\n")
			srv := New(tmpDir, Config{Port: "0", MCP: true})
			defer srv.Close()
			err := srv.LoadModules([]string{modPath})
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("LoadModules error = %v, want it to contain %q", err, c.want)
			}
		})
	}
}

// TestMCPCheck_OpenAITargetFileParams: `ailang mcp check --target openai`
// passes the listed surface of a module using @mcp_file + @mcp_auth.
func TestMCPCheck_OpenAITargetFileParams(t *testing.T) {
	as := authServer(t, true)
	hs := fileServerFor(t, as.URL)
	fs := runCheck(t, hs.URL+listedMCPPath, "openai")
	for _, c := range []string{"annotations", "lazy-auth", "file-params", "security-schemes"} {
		if got := statusOf(fs, c); got != "PASS" {
			t.Errorf("%s = %s, want PASS; findings: %+v", c, got, fs)
		}
	}
}
