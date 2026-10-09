package apiserver

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/eval"
)

func headerSourceServer(t *testing.T, src string) *Server {
	t.Helper()
	setHeaderTestStdlib(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "headers.ail")
	if err := os.WriteFile(path, []byte("module headers\nimport std/json (Json, jo, kv, js, jnum)\n"+src), 0600); err != nil {
		t.Fatal(err)
	}
	srv := New(dir, Config{CORS: true})
	t.Cleanup(func() { srv.Close() })
	if err := srv.LoadModules([]string{path}); err != nil {
		t.Fatal(err)
	}
	return srv
}

func TestResponseHeaderFailuresBothHandlers(t *testing.T) {
	for _, tc := range []struct{ name, typ, value, field string }{
		{"scalar", "int", "42", "IntValue"},
		{"record-value", "{x_bad: int}", "{x_bad: 42}", "x_bad"},
		{"json-value", "Json", "jo([kv(\"X-Bad\", jnum(42.0))])", "X-Bad"},
		{"json-scalar", "Json", "js(\"bad\")", "JString"},
		{"json-name", "Json", "jo([kv(\"Bad Name\", js(\"x\"))])", "Bad Name"},
		{"newline", "{x_bad: string}", "{x_bad: \"a\\nb\"}", "x-bad"},
		{"tab", "{x_bad: string}", "{x_bad: \"a\\tb\"}", "x-bad"},
		{"reserved-timing", "{x_elapsed_ms: string}", "{x_elapsed_ms: \"999\"}", "x-elapsed-ms"},
		{"reserved-cors", "Json", "jo([kv(\"aCcEsS-CoNtRoL-Allow-Origin\", js(\"evil\"))])", "aCcEsS-CoNtRoL-Allow-Origin"},
		{"reserved-vary", "{vary: string}", "{vary: \"evil\"}", "vary"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, nowrap := range []bool{false, true} {
				bodyField := "_body"
				if nowrap {
					bodyField = "data"
				}
				src := "export pure func bad() -> {" + bodyField + ": string, _headers: " + tc.typ + "} = {" + bodyField + ": \"ok\", _headers: " + tc.value + "}"
				srv := headerSourceServer(t, src)
				var logs bytes.Buffer
				old := log.Writer()
				log.SetOutput(&logs)
				w := httptest.NewRecorder()
				w.Header().Set("Access-Control-Allow-Origin", "trusted")
				srv.callFunction(w, httptest.NewRequest("GET", "/", nil), "headers", "bad", callOpts{Nowrap: nowrap})
				log.SetOutput(old)
				if w.Code != 500 || !strings.Contains(w.Body.String(), "_headers must be") || !strings.Contains(w.Body.String(), tc.field) {
					t.Fatalf("nowrap=%v: %d %s", nowrap, w.Code, w.Body.String())
				}
				if !strings.Contains(logs.String(), "ERROR:") || !strings.Contains(logs.String(), tc.field) {
					t.Fatalf("missing ERROR diagnostic: %s", logs.String())
				}
				if w.Header().Get("Access-Control-Allow-Origin") != "trusted" {
					t.Error("overwrote CORS")
				}
			}
		})
	}
}

func TestResponseHeadersAtomicAndImmutable(t *testing.T) {
	rec := &eval.RecordValue{Fields: map[string]eval.Value{"_headers": &eval.RecordValue{Fields: map[string]eval.Value{"a_valid": &eval.StringValue{Value: "yes"}, "z_invalid": &eval.IntValue{Value: 1}}}}}
	w := httptest.NewRecorder()
	rec.Fields["_body"] = &eval.StringValue{Value: "ok"}
	writeRawResponse(w, rec, 0)
	if w.Code != 500 || w.Result().Header.Get("A-Valid") != "" {
		t.Fatal("partially applied invalid header set")
	}
	clean := withoutResponseHeaders(rec).(*eval.RecordValue)
	if _, ok := clean.Fields["_headers"]; ok {
		t.Error("metadata retained")
	}
	if _, ok := rec.Fields["_headers"]; !ok {
		t.Error("mutated original record")
	}
}

func TestRawBodyDefaultsAndOverride(t *testing.T) {
	for _, tc := range []struct {
		body eval.Value
		want string
	}{
		{&eval.StringValue{Value: "x"}, "text/plain; charset=utf-8"},
		{&eval.BytesValue{Value: []byte{1, 2}}, "application/octet-stream"},
		{&eval.IntValue{Value: 1}, "application/json"},
	} {
		for _, override := range []bool{false, true} {
			rec := &eval.RecordValue{Fields: map[string]eval.Value{"_body": tc.body, "_status": &eval.IntValue{Value: 201}}}
			want := tc.want
			if override {
				want = "text/html"
				rec.Fields["_headers"] = &eval.RecordValue{Fields: map[string]eval.Value{"content_type": &eval.StringValue{Value: want}}}
			}
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { writeRawResponse(w, rec, 5) }))
			resp, err := ts.Client().Get(ts.URL)
			if err != nil {
				ts.Close()
				t.Fatal(err)
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			ts.Close()
			if resp.StatusCode != 201 || resp.Header.Get("Content-Type") != want {
				t.Fatalf("status=%d headers=%v want %q", resp.StatusCode, resp.Header, want)
			}
		}
	}
}

func TestJsonExactNamesAndLastWins(t *testing.T) {
	srv := headerSourceServer(t, `@nowrap
@route("GET", "/exact")
export pure func exact() -> {data: string, _headers: Json} =
 {data: "ok", _headers: jo([kv("X_Exact", js("one")), kv("x_exact", js("two")), kv("Content-Type", js("text/custom"))])}`)
	w := httptest.NewRecorder()
	srv.buildRoutes().ServeHTTP(w, httptest.NewRequest("GET", "/exact", nil))
	if w.Result().Header.Get("X_Exact") != "two" || w.Result().Header.Get("X-Exact") != "" || w.Result().Header.Get("Content-Type") != "text/custom" {
		t.Fatalf("%v", w.Result().Header)
	}
}

func TestDeclaredResponseHeaderRegistration(t *testing.T) {
	setHeaderTestStdlib(t)
	for _, tc := range []struct {
		decl  string
		value string
		bad   bool
	}{
		{"int", "42", true}, {"{x_bad: int}", "{x_bad: 1}", true},
		{"{x_good: string}", "{x_good: \"ok\"}", false}, {"Json", "jo([])", false},
	} {
		t.Run(tc.decl, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "headers.ail")
			src := "module headers\nimport std/json (Json, jo)\n@route(\"GET\", \"/test\")\nexport pure func f() -> {_body: string, _headers: " + tc.decl + "} = {_body: \"ok\", _headers: " + tc.value + "}"
			if err := os.WriteFile(path, []byte(src), 0600); err != nil {
				t.Fatal(err)
			}
			srv := New(dir, Config{})
			defer srv.Close()
			err := srv.LoadModules([]string{path})
			if (err != nil) != tc.bad {
				t.Fatalf("LoadModules=%v bad=%v", err, tc.bad)
			}
			if tc.bad && !strings.Contains(err.Error(), responseHeadersShape) {
				t.Fatalf("missing actionable diagnostic: %v", err)
			}
		})
	}
}

func setHeaderTestStdlib(t *testing.T) {
	t.Helper()
	path, err := filepath.Abs("../../std")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("AILANG_STDLIB_PATH", path)
}
