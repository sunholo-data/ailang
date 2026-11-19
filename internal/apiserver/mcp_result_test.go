package apiserver

import (
	"context"
	"strings"
	"testing"
)

const resultModule = `module results

import std/result (Result, Ok, Err)

-- Ok answers with the value.
export func good(n: int) -> Result[int, string] = Ok(n * 2)

-- Err with a string is a tool error with that text.
export func refuse(why: string) -> Result[int, string] = Err(why)

-- Err with a record is a tool error carrying the record as JSON.
export func refuseRec(code: string) -> Result[string, {code: string, hint: string}] =
  Err({code: code, hint: "see the docs"})

-- A plain value is untouched.
export func plain(s: string) -> string = s
`

// TestMCPResult_ErrIsToolError: a tool returning Result has the meaning a
// @route handler gives it. Ok answers with the inner value; Err is isError
// with the payload as text. Before, both crossed MCP as a successful
// {"__tag": ...} object.
func TestMCPResult_ErrIsToolError(t *testing.T) {
	srv, _ := newSurfaceServer(t, Config{MCP: true}, map[string]string{"results.ail": resultModule})
	exports := map[string]ExportInfo{}
	modPath := ""
	for path, mod := range srv.modules {
		for _, e := range mod.Exports {
			exports[e.Name] = e
			modPath = path
		}
	}
	call := func(fn, args string) (string, bool) {
		t.Helper()
		exp, ok := exports[fn]
		if !ok {
			t.Fatalf("export %s not loaded (have %v)", fn, exports)
		}
		res, err := NewMCPServer(srv).makeToolHandler(modPath, exp)(context.Background(), mcpCallReq(fn, args))
		if err != nil {
			t.Fatal(err)
		}
		return mcpResultText(t, res), res.IsError
	}

	if text, isErr := call("good", `{"n": 21}`); isErr || strings.TrimSpace(text) != "42" {
		t.Errorf("good: isError=%v text=%q, want false and 42", isErr, text)
	}
	if text, isErr := call("refuse", `{"why": "no such slug"}`); !isErr || text != "no such slug" {
		t.Errorf("refuse: isError=%v text=%q, want true and the Err string", isErr, text)
	}
	text, isErr := call("refuseRec", `{"code": "missing"}`)
	if !isErr || !strings.Contains(text, `"code":"missing"`) || !strings.Contains(text, `"hint":"see the docs"`) {
		t.Errorf("refuseRec: isError=%v text=%q, want true and the record as JSON", isErr, text)
	}
	if strings.Contains(text, "__tag") {
		t.Errorf("refuseRec leaks the constructor tag: %q", text)
	}
	if text, isErr := call("plain", `{"s": "as is"}`); isErr || !strings.Contains(text, "as is") {
		t.Errorf("plain: isError=%v text=%q, want the value unchanged", isErr, text)
	}
}
