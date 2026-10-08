package apiserver

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// An omitted record param binds its DECLARED shape at field zeros on every
// surface — REST (@route named / empty-body / positional / multipart) and an
// omitted @optional MCP param — never the empty record {} the type name
// "record" gives (which crashed the function: "record has no field: file_id").

const zeroModule = `module test/api/zeros

type OpenAIFile = {download_url: string, file_id: string, mime_type: string, file_name: string}

type Opts = {limit: int, ratio: float, verbose: bool, tags: [string], inner: {label: string}}

@route("POST", "/parse")
@mcp_file("file")
@optional("file")
export func parse(file: OpenAIFile, mode: string) -> string = "[${file.file_id}|${file.download_url}|${file.mime_type}|${file.file_name}|${mode}]"

@route("POST", "/inline")
@mcp_file("doc")
@optional("doc")
export func inlineDoc(doc: {download_url: string, file_id: string, mime_type: string, file_name: string}, mode: string) -> string = "[${doc.file_id}|${doc.file_name}|${mode}]"

@route("POST", "/opts")
@optional("opts")
export func withOpts(opts: Opts, q: string) -> string = "[${show(opts.limit + 1)}|${show(opts.ratio + 0.5)}|${show(opts.verbose)}|${match opts.tags { [] => "empty", _ => "some" }}|${opts.inner.label}|${q}]"
`

func zeroServer(t *testing.T) *httptest.Server {
	t.Helper()
	tmpDir, modPath := writeModule(t, "zeros", zeroModule)
	srv := New(tmpDir, Config{Port: "0", MCP: true, NoFeedbackTool: true})
	t.Cleanup(func() { srv.Close() })
	if err := srv.LoadModules([]string{modPath}); err != nil {
		t.Fatalf("LoadModules: %v", err)
	}
	hs := httptest.NewServer(srv.buildRoutes())
	t.Cleanup(hs.Close)
	return hs
}

func postREST(t *testing.T, url, contentType, body string) (int, string) {
	t.Helper()
	resp, err := http.Post(url, contentType, bytes.NewBufferString(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(b)
}

func TestParamZero_RESTOmittedRecordBindsDeclaredShape(t *testing.T) {
	hs := zeroServer(t)
	cases := []struct{ name, path, body, want string }{
		{"file alias omitted, named", "/parse", `{"mode":"md"}`, `[||||md]`},
		{"file alias omitted, empty body", "/parse", ``, `[||||]`},
		{"file alias omitted, positional", "/parse", `{"args":[]}`, `[||||]`},
		{"file alias given", "/parse", `{"file":{"download_url":"u","file_id":"f","mime_type":"m","file_name":"n"},"mode":"md"}`, `[f|u|m|n|md]`},
		{"inline file record omitted", "/inline", `{"mode":"x"}`, `[||x]`},
		{"non-file record omitted", "/opts", `{"q":"z"}`, `[1|0.5|false|empty||z]`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			st, body := postREST(t, hs.URL+c.path, "application/json", c.body)
			if st != 200 {
				t.Fatalf("status %d: %s", st, body)
			}
			if !strings.Contains(body, c.want) {
				t.Fatalf("want %q in %s", c.want, body)
			}
		})
	}
}

func TestParamZero_RESTMultipartOmittedRecord(t *testing.T) {
	hs := zeroServer(t)
	var buf bytes.Buffer
	ct := "multipart/form-data; boundary=XX"
	buf.WriteString("--XX\r\nContent-Disposition: form-data; name=\"mode\"\r\n\r\nmd\r\n--XX--\r\n")
	st, body := postREST(t, hs.URL+"/parse", ct, buf.String())
	if st != 200 || !strings.Contains(body, `[||||md]`) {
		t.Fatalf("status %d: %s", st, body)
	}
}

// MCP behaviour is unchanged for the file param (all four "") and the
// non-file record param now binds its declared shape too.
func TestParamZero_MCPOmittedOptionalRecord(t *testing.T) {
	hs := zeroServer(t)
	cases := []struct {
		tool string
		args map[string]any
		want string
	}{
		{"parse", map[string]any{"mode": "md"}, `[||||md]`},
		{"parse", map[string]any{"file": map[string]any{"download_url": "u", "file_id": "f"}, "mode": "md"}, `[f|u|||md]`},
		{"inlineDoc", map[string]any{"mode": "x"}, `[||x]`},
		{"withOpts", map[string]any{"q": "z"}, `[1|0.5|false|empty||z]`},
	}
	for _, c := range cases {
		st, _, body := rpc(t, hs.URL+"/mcp/", "", "tools/call", call(c.tool, c.args))
		if st != 200 || !strings.Contains(body, c.want) {
			t.Fatalf("%s: status %d, want %q in %s", c.tool, st, c.want, body)
		}
	}
}

// Each binding gets a fresh zero: a shared map would let one call's record
// leak into the next.
func TestParamZero_FreshCopyPerCall(t *testing.T) {
	zeros := []any{map[string]any{"a": "", "inner": map[string]any{"b": ""}}}
	z1 := paramZero([]string{"record"}, zeros, 0).(map[string]any)
	z1["a"] = "mutated"
	z1["inner"].(map[string]any)["b"] = "mutated"
	z2 := paramZero([]string{"record"}, zeros, 0).(map[string]any)
	if z2["a"] != "" || z2["inner"].(map[string]any)["b"] != "" {
		t.Fatalf("zero shared across calls: %v", z2)
	}
	// Non-record params keep the type-name zero.
	if got := paramZero([]string{"string"}, nil, 0); got != "" {
		t.Fatalf("string zero = %v", got)
	}
}
