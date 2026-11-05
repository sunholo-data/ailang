package apiserver

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

// Named regressions for the two reports the shared binder closes, plus a
// per-kind matrix over every surface that binds arguments: MCP (named and
// positional), REST (named, positional and multipart) and A2A.

// kindsModule declares one export per bindable kind. zeros has every kind
// @optional, so one call shows each omitted param's declared zero.
const kindsModule = `module test/api/kinds
import std/list (length)

-- Double a number.
@mcp_title("Twice")
@mcp_hints("readOnly")
export func twice(n: int) -> int = n * 2

@optional("x")
export func half(x: float) -> float = x / 2.0

@optional("n")
export func optInt(n: int) -> int = n

export func flip(b: bool) -> bool = !b
export func greet(s: string) -> string = "hi ${s}"
export func area(r: {w: int, h: float, label: string, on: bool}) -> float = if r.on then intToFloat(r.w) * r.h else 0.0
export func total(xs: [int]) -> int = match xs { [] => 0, x :: rest => x + total(rest) }
export func ftotal(xs: [float]) -> float = match xs { [] => 0.0, x :: rest => x + ftotal(rest) }

@optional("n", "x", "b", "s", "r", "xs")
export func zeros(n: int, x: float, b: bool, s: string, r: {w: int, h: float}, xs: [int]) -> string =
  "${show(n + 1)}|${show(x + 0.5)}|${show(b)}|[${s}]|${show(r.w + 1)}|${show(r.h + 0.5)}|${show(length(xs))}"
`

const kindsModPath = "test/api/kinds"

func kindsServer(t *testing.T) (*Server, map[string]ExportInfo) {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("AILANG_STDLIB_PATH", filepath.Join(root, "std"))
	dir, path := writeModule(t, "kinds", kindsModule)
	srv := New(dir, Config{Port: "0", MCP: true, NoFeedbackTool: true})
	t.Cleanup(func() { srv.Close() })
	if err := srv.LoadModules([]string{path}); err != nil {
		t.Fatal(err)
	}
	exports := map[string]ExportInfo{}
	for _, mod := range srv.modules {
		for _, e := range mod.Exports {
			exports[e.Name] = e
		}
	}
	return srv, exports
}

// callMCP runs tools/call through the registered tool handler.
func callMCP(t *testing.T, srv *Server, exp ExportInfo, args string) (string, bool) {
	t.Helper()
	if exp.Name == "" {
		t.Fatal("missing export")
	}
	res, err := NewMCPServer(srv).makeToolHandler(kindsModPath, exp)(context.Background(), mcpCallReq(exp.Name, args))
	if err != nil {
		t.Fatal(err)
	}
	return mcpResultText(t, res), res.IsError
}

// callREST posts a JSON body and returns (status, result JSON, error text).
func callREST(t *testing.T, srv *Server, fn, body string) (int, string, string) {
	t.Helper()
	req := httptest.NewRequest("POST", "/api/"+kindsModPath+"/"+fn, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return serveREST(t, srv, req)
}

// callMultipart posts form text fields and returns (status, result JSON, error text).
func callMultipart(t *testing.T, srv *Server, fn string, fields map[string]string) (int, string, string) {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	for k, v := range fields {
		if err := mw.WriteField(k, v); err != nil {
			t.Fatal(err)
		}
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("POST", "/api/"+kindsModPath+"/"+fn, &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	return serveREST(t, srv, req)
}

func serveREST(t *testing.T, srv *Server, req *http.Request) (int, string, string) {
	t.Helper()
	w := httptest.NewRecorder()
	srv.buildRoutes().ServeHTTP(w, req)
	var resp FunctionCallResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("REST body %d: %s", w.Code, w.Body.String())
	}
	result, _ := json.Marshal(resp.Result)
	return w.Code, string(result), resp.Error
}

// callA2A sends positional args as an A2A data part; returns the raw body.
func callA2A(t *testing.T, srv *Server, fn, argsJSON string) string {
	t.Helper()
	body := `{"jsonrpc":"2.0","id":1,"method":"tasks/send","params":{"metadata":{"skill_id":"test.api.kinds.` + fn +
		`"},"message":{"parts":[{"type":"data","data":{"args":` + argsJSON + `}}]}}}`
	w := httptest.NewRecorder()
	srv.handleA2ATask(w, httptest.NewRequest("POST", "/a2a/", strings.NewReader(body)))
	return w.Body.String()
}

// #1631: every int param failed over MCP tools/call ("expected float arguments").
func TestIssue1631_MCPIntParam(t *testing.T) {
	srv, exp := kindsServer(t)
	if txt, isErr := callMCP(t, srv, exp["twice"], `{"n":21}`); isErr || txt != "42" {
		t.Fatalf("twice(21) over MCP = %q (isError=%v), want 42", txt, isErr)
	}
	if txt, isErr := callMCP(t, srv, exp["twice"], `{"args":[21]}`); isErr || txt != "42" {
		t.Fatalf("positional twice(21) over MCP = %q (isError=%v), want 42", txt, isErr)
	}
	txt, isErr := callMCP(t, srv, exp["twice"], `{"n":2.5}`)
	if !isErr || !strings.HasPrefix(txt, "invalid arguments: n: expected int, got number (2.5)") {
		t.Fatalf("twice(2.5) over MCP = %q (isError=%v), want invalid-argument error", txt, isErr)
	}
	if txt, isErr := callMCP(t, srv, exp["optInt"], `{}`); isErr || txt != "0" {
		t.Fatalf("omitted @optional int over MCP = %q (isError=%v), want 0", txt, isErr)
	}
}

// #1648: REST float params failed for whole numbers and for an omitted
// @optional float (the int-shaped zero reached a float division).
func TestIssue1648_RESTFloatParam(t *testing.T) {
	srv, _ := kindsServer(t)
	for _, tc := range []struct{ body, want string }{
		{`{}`, `0`},
		{`{"x":3}`, `1.5`},
		{`{"x":3.5}`, `1.75`},
		{`{"args":[3]}`, `1.5`},
	} {
		code, got, errText := callREST(t, srv, "half", tc.body)
		if code != 200 || got != tc.want {
			t.Errorf("REST half %s = %d %s %q, want 200 %s", tc.body, code, got, errText, tc.want)
		}
	}
	code, _, errText := callREST(t, srv, "twice", `{"n":2.5}`)
	if code != 400 || !strings.HasPrefix(errText, "invalid arguments: n: expected int") {
		t.Fatalf("REST twice 2.5 = %d %q, want 400 invalid arguments", code, errText)
	}
}

// TestTypedArgsEverySurface: each kind binds the same on every surface.
func TestTypedArgsEverySurface(t *testing.T) {
	srv, exp := kindsServer(t)
	rec := `{"w":2,"h":3,"label":"a","on":true}`
	for _, tc := range []struct {
		fn, param, value, want string
		form                   string // multipart text; "" = same as value
	}{
		{"twice", "n", `21`, `42`, ""},
		{"half", "x", `3`, `1.5`, ""},
		{"half", "x", `3.5`, `1.75`, ""},
		{"flip", "b", `true`, `false`, ""},
		{"greet", "s", `"bob"`, `"hi bob"`, "bob"},
		{"area", "r", rec, `6`, ""},
		{"total", "xs", `[1,2,3]`, `6`, ""},
		{"ftotal", "xs", `[1,2]`, `3`, ""},
	} {
		t.Run(tc.fn+"_"+tc.value, func(t *testing.T) {
			named := `{"` + tc.param + `":` + tc.value + `}`
			positional := `{"args":[` + tc.value + `]}`
			for _, args := range []string{named, positional} {
				if txt, isErr := callMCP(t, srv, exp[tc.fn], args); isErr || txt != tc.want {
					t.Errorf("MCP %s %s = %q (isError=%v), want %s", tc.fn, args, txt, isErr, tc.want)
				}
				if code, got, e := callREST(t, srv, tc.fn, args); code != 200 || got != tc.want {
					t.Errorf("REST %s %s = %d %s %q, want %s", tc.fn, args, code, got, e, tc.want)
				}
			}
			form := tc.form
			if form == "" {
				form = tc.value
			}
			if code, got, e := callMultipart(t, srv, tc.fn, map[string]string{tc.param: form}); code != 200 || got != tc.want {
				t.Errorf("multipart %s %s=%s = %d %s %q, want %s", tc.fn, tc.param, form, code, got, e, tc.want)
			}
			if body := callA2A(t, srv, tc.fn, `[`+tc.value+`]`); !strings.Contains(body, `"completed"`) || strings.Contains(body, `"error"`) {
				t.Errorf("A2A %s [%s] = %s", tc.fn, tc.value, body)
			}
		})
	}
}

// TestTypedArgsRejectedEverySurface: a value of the wrong kind is refused
// before evaluation, naming the param (and the field or index inside it).
func TestTypedArgsRejectedEverySurface(t *testing.T) {
	srv, exp := kindsServer(t)
	for _, tc := range []struct{ fn, param, value, form, path string }{
		{"twice", "n", `2.5`, `2.5`, "n: expected int"},
		{"twice", "n", `"21"`, `abc`, "n: expected int"},
		{"flip", "b", `1`, `maybe`, "b: expected bool"},
		{"greet", "s", `5`, ``, "s: expected string"},
		{"area", "r", `{"w":2.5,"h":3,"label":"a","on":true}`, `{"w":2.5,"h":3,"label":"a","on":true}`, "r.w: expected int"},
		{"total", "xs", `[1,2.5]`, `[1,2.5]`, "xs[1]: expected int"},
	} {
		t.Run(tc.fn+"_"+tc.value, func(t *testing.T) {
			named := `{"` + tc.param + `":` + tc.value + `}`
			if txt, isErr := callMCP(t, srv, exp[tc.fn], named); !isErr || !strings.Contains(txt, "invalid arguments: "+tc.path) {
				t.Errorf("MCP %s = %q (isError=%v), want %q", named, txt, isErr, tc.path)
			}
			if code, _, e := callREST(t, srv, tc.fn, named); code != 400 || !strings.Contains(e, "invalid arguments: "+tc.path) {
				t.Errorf("REST %s = %d %q, want 400 %q", named, code, e, tc.path)
			}
			if tc.form != "" {
				if code, _, e := callMultipart(t, srv, tc.fn, map[string]string{tc.param: tc.form}); code != 400 || !strings.Contains(e, tc.path) {
					t.Errorf("multipart %s=%s = %d %q, want 400 %q", tc.param, tc.form, code, e, tc.path)
				}
			}
			if body := callA2A(t, srv, tc.fn, `[`+tc.value+`]`); !strings.Contains(body, `-32602`) || !strings.Contains(body, tc.path) {
				t.Errorf("A2A %s [%s] = %s", tc.fn, tc.value, body)
			}
		})
	}
}

// TestTypedArgsOptionalZerosEverySurface: an omitted @optional param binds
// its declared zero (int 0, float 0.0, false, "", the record at field zeros,
// the empty list) on MCP, REST JSON and multipart alike.
func TestTypedArgsOptionalZerosEverySurface(t *testing.T) {
	srv, exp := kindsServer(t)
	if err := validateOptionalParams(exp["zeros"]); err != nil {
		t.Fatalf("@optional list/record/scalars must register on MCP: %v", err)
	}
	const want = "1|0.5|false|[]|1|0.5|0"
	if txt, isErr := callMCP(t, srv, exp["zeros"], `{}`); isErr || txt != `"`+want+`"` {
		t.Errorf("MCP zeros {} = %q (isError=%v)", txt, isErr)
	}
	if code, got, e := callREST(t, srv, "zeros", `{}`); code != 200 || got != `"`+want+`"` {
		t.Errorf("REST zeros {} = %d %s %q", code, got, e)
	}
	if code, got, e := callMultipart(t, srv, "zeros", map[string]string{"s": "q"}); code != 200 || got != `"1|0.5|false|[q]|1|0.5|0"` {
		t.Errorf("multipart zeros s=q = %d %s %q", code, got, e)
	}
	// A list param advertises an array schema, not the "string" an
	// unrecognised [T] used to fall back to.
	props := buildNamedInputSchema(exp["total"])["properties"].(map[string]any)
	if got := props["xs"].(map[string]any)["type"]; got != "array" {
		t.Errorf("MCP schema for xs: [int] = %v, want array", got)
	}
}

// TestFormTextValue: multipart text is parsed only into the declared kind,
// and text that does not parse stays text for the binder to refuse.
func TestFormTextValue(t *testing.T) {
	for _, tc := range []struct {
		kind, text string
		want       any
	}{
		{"int", "21", json.Number("21")},
		{"int", "1_000", "1_000"}, // strconv would read 1000; a form field must not
		{"float", "3.5", json.Number("3.5")},
		{"bool", "true", true},
		{"bool", "yes", "yes"},
		{"record", "{bad", "{bad"},
		{"string", "21", "21"},
		{"", "21", "21"}, // unknown leaf: text, as before
	} {
		if got := formTextValue(tc.kind, tc.text); got != tc.want {
			t.Errorf("formTextValue(%q, %q) = %#v, want %#v", tc.kind, tc.text, got, tc.want)
		}
	}
	if got, ok := formTextValue("list", "[1,2]").([]any); !ok || len(got) != 2 {
		t.Errorf("list text = %#v, want a 2-element list", got)
	}
}
