package apiserver

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

const typedArgsModule = `module test/api/typed
import std/array as A
export func twice(n: int) -> int = n * 2
export func half(x: float) -> float = x / 2.0
export func count(r: {count: int}) -> int = r.count + 1
export func scale(r: {factor: float}) -> float = r.factor * 3.0
@optional("n")
export func optional(n: int) -> int = n + 1
export func nested(r: {items: [{n: int, f: float}]}) -> int = match r.items { [] => 0, x :: _ => x.n + 1 }
export func arraySize(xs: Array[int]) -> int = A.length(xs)
@optional("xs")
export func optionalArray(xs: Array[int]) -> int = A.length(xs)
export func truth(b: bool) -> bool = b
export func text(s: string) -> string = s
`

func typedArgsServer(t *testing.T) *Server {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("AILANG_STDLIB_PATH", filepath.Join(root, "std"))
	dir, path := writeModule(t, "typed", typedArgsModule)
	srv := New(dir, Config{Port: "0", MCP: true, NoFeedbackTool: true})
	t.Cleanup(func() { srv.Close() })
	if err := srv.LoadModules([]string{path}); err != nil {
		t.Fatal(err)
	}
	return srv
}

func TestTypedArgsSurfaceParity(t *testing.T) {
	srv := typedArgsServer(t)
	ms := NewMCPServer(srv)
	var exports []ExportInfo
	for _, mod := range srv.modules {
		exports = append(exports, mod.Exports...)
	}
	for _, tc := range []struct {
		name, fn, args, want string
		invalid              bool
	}{
		{"int", "twice", `{"n":21}`, `42`, false},
		{"float", "half", `{"x":2}`, `1`, false},
		{"int record", "count", `{"r":{"count":5}}`, `6`, false},
		{"float record", "scale", `{"r":{"factor":2}}`, `6`, false},
		{"optional int", "optional", `{}`, `1`, false},
		{"fraction", "twice", `{"n":2.5}`, `n`, true},
		{"bool mismatch", "truth", `{"b":1}`, `b`, true},
		{"string mismatch", "text", `{"s":2}`, `s`, true},
		{"nested", "nested", `{"r":{"items":[{"n":5,"f":2}]}}`, `6`, false},
		{"nested error", "nested", `{"r":{"items":[{"n":2.5,"f":2}]}}`, `r.items[0].n`, true},
		{"array", "arraySize", `{"xs":[1,2]}`, `2`, false},
		{"optional array", "optionalArray", `{}`, `0`, false},
		{"positional", "twice", `{"args":[21]}`, `42`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var exp ExportInfo
			for _, e := range exports {
				if e.Name == tc.fn {
					exp = e
				}
			}
			if exp.Name == "" {
				t.Fatal("missing export")
			}
			res, err := ms.makeToolHandler("test/api/typed", exp)(context.Background(), mcpCallReq(tc.fn, tc.args))
			if err != nil {
				t.Fatal(err)
			}
			txt := mcpResultText(t, res)
			if res.IsError != tc.invalid || !strings.Contains(txt, tc.want) {
				t.Fatalf("MCP error=%v text=%s", res.IsError, txt)
			}
			req := httptest.NewRequest("POST", "/api/test/api/typed/"+tc.fn, strings.NewReader(tc.args))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			srv.buildRoutes().ServeHTTP(w, req)
			if tc.invalid {
				if w.Code != 400 || !strings.Contains(w.Body.String(), tc.want) {
					t.Fatalf("REST %d: %s", w.Code, w.Body.String())
				}
			} else {
				var response FunctionCallResponse
				if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
					t.Fatal(err)
				}
				want := tc.want
				result, _ := json.Marshal(response.Result)
				if w.Code != 200 || string(result) != want {
					t.Fatalf("REST %d: %s", w.Code, w.Body.String())
				}
			}
		})
	}
}

func TestTypedArgsA2A(t *testing.T) {
	srv := typedArgsServer(t)
	for _, part := range []string{`{"type":"data","data":{"args":[21]}}`, `{"type":"text","text":"[21]"}`, `{"type":"data","data":{"args":[2.5]}}`} {
		body := `{"jsonrpc":"2.0","id":1,"method":"tasks/send","params":{"metadata":{"skill_id":"test.api.typed.twice"},"message":{"parts":[` + part + `]}}}`
		w := httptest.NewRecorder()
		srv.handleA2ATask(w, httptest.NewRequest("POST", "/a2a/", strings.NewReader(body)))
		if strings.Contains(part, "2.5") {
			if !strings.Contains(w.Body.String(), `-32602`) || !strings.Contains(w.Body.String(), `n: expected int`) {
				t.Fatal(w.Body.String())
			}
		} else if !strings.Contains(w.Body.String(), `42`) {
			t.Fatal(w.Body.String())
		}
	}
}
