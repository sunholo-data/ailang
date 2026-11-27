package apiserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/eval"
)

// TST-RC1-05
func TestParamSchemaAndBindArg_StdJsonADT(t *testing.T) {
	s := schemaFromAST(&ast.SimpleType{Name: "Json"}, nil, 0)
	if s.Kind != "json" {
		t.Fatalf("schemaFromAST(Json).Kind = %q, want \"json\"", s.Kind)
	}
	nested := schemaFromAST(&ast.RecordType{
		Fields: []*ast.RecordField{{Name: "payload", Type: &ast.SimpleType{Name: "Json"}}},
	}, nil, 0)
	if nested.Kind != "record" || len(nested.Fields) != 1 || nested.Fields[0].Schema.Kind != "json" {
		t.Fatalf("nested Json record schema = %+v", nested)
	}

	// Null -> JNull (NOT UnitValue!)
	nullVal, err := bindArg(ParamSchema{Kind: "json"}, nil, "payload")
	if err != nil {
		t.Fatalf("bindArg(json, nil): %v", err)
	}
	tvNull, ok := nullVal.(*eval.TaggedValue)
	if !ok || tvNull.TypeName != "Json" || tvNull.CtorName != "JNull" || tvNull.ModulePath != "std/json" {
		t.Fatalf("nil bound to %#v, want std/json.Json JNull", nullVal)
	}

	// Complex nested object -> JObject([{key, value}]) with deterministic sorted keys
	input := map[string]any{
		"z_str":  "hello",
		"a_bool": true,
		"b_int":  float64(42),
		"c_num":  float64(3.5),
		"d_null": nil,
		"e_arr":  []any{"x", json.Number("9007199254740993")},
	}
	bound, err := bindArg(ParamSchema{Kind: "json"}, input, "payload")
	if err != nil {
		t.Fatalf("bindArg(json, object): %v", err)
	}
	obj, ok := bound.(*eval.TaggedValue)
	if !ok || obj.TypeName != "Json" || obj.CtorName != "JObject" || len(obj.Fields) != 1 {
		t.Fatalf("object bound to %#v, want JObject", bound)
	}
	pairs := obj.Fields[0].(*eval.ListValue).Elements
	if len(pairs) != 6 {
		t.Fatalf("JObject pair count = %d, want 6", len(pairs))
	}
	// Verify alphabetical key ordering: a_bool, b_int, c_num, d_null, e_arr, z_str
	wantKeys := []string{"a_bool", "b_int", "c_num", "d_null", "e_arr", "z_str"}
	wantCtors := []string{"JBool", "JInt", "JNumber", "JNull", "JArray", "JString"}
	for i, p := range pairs {
		rec := p.(*eval.RecordValue)
		gotKey := rec.Fields["key"].(*eval.StringValue).Value
		if gotKey != wantKeys[i] {
			t.Fatalf("pair[%d].key = %q, want %q", i, gotKey, wantKeys[i])
		}
		gotVal := rec.Fields["value"].(*eval.TaggedValue)
		if gotVal.CtorName != wantCtors[i] || gotVal.ModulePath != "std/json" || gotVal.TypeName != "Json" {
			t.Fatalf("pair[%d].value = %#v, want %s", i, gotVal, wantCtors[i])
		}
	}
	// Check 64-bit json.Number inside JArray became JInt(9007199254740993)
	arrElems := pairs[4].(*eval.RecordValue).Fields["value"].(*eval.TaggedValue).Fields[0].(*eval.ListValue).Elements
	bigElem := arrElems[1].(*eval.TaggedValue)
	if bigElem.CtorName != "JInt" || bigElem.Fields[0].(*eval.IntValue).Value != 9007199254740993 {
		t.Fatalf("64-bit json.Number bound to %#v, want JInt(9007199254740993)", bigElem)
	}
}

// TST-RC1-05
func TestServeAPI_StdJsonEndpointReturnsHTTP200(t *testing.T) {
	tmpDir := t.TempDir()
	apiDir := filepath.Join(tmpDir, "api")
	if err := os.MkdirAll(apiDir, 0o755); err != nil {
		t.Fatal(err)
	}
	modContent := `module api/webhook
import std/json (Json, getString, getInt, encode)
import std/option (getOrElse)

export pure func summarize(payload: Json) -> string {
  let event = getOrElse(getString(payload, "event"), "none");
  let count = getOrElse(getInt(payload, "count"), -1);
  "${event}:${show(count)}:${encode(payload)}"
}
`
	modPath := filepath.Join(apiDir, "webhook.ail")
	if err := os.WriteFile(modPath, []byte(modContent), 0o644); err != nil {
		t.Fatal(err)
	}

	cwd, _ := os.Getwd()
	for dir := cwd; dir != filepath.Dir(dir); dir = filepath.Dir(dir) {
		if _, err := os.Stat(filepath.Join(dir, "std", "json.ail")); err == nil {
			t.Setenv("AILANG_STDLIB_PATH", filepath.Join(dir, "std"))
			break
		}
	}

	srv := New(tmpDir, Config{Port: "0", CORS: true})
	defer srv.Close()
	if err := srv.LoadModules([]string{modPath}); err != nil {
		t.Fatalf("LoadModules: %v", err)
	}

	mux := srv.buildRoutes()

	// Named parameter call: {"payload": {"count": 7, "event": "push"}}
	reqBody := `{"payload": {"count": 7, "event": "push"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/api/webhook/summarize", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("POST /api/api/webhook/summarize status = %d, want 200; body = %s", rr.Code, rr.Body.String())
	}
	var resp FunctionCallResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	got, _ := resp.Result.(string)
	want := `push:7:{"count":7,"event":"push"}`
	if got != want {
		t.Fatalf("result = %q, want %q", got, want)
	}

	// Positional null call: {"args": [null]} -> JNull
	reqNull := httptest.NewRequest(http.MethodPost, "/api/api/webhook/summarize", strings.NewReader(`{"args": [null]}`))
	reqNull.Header.Set("Content-Type", "application/json")
	rrNull := httptest.NewRecorder()
	mux.ServeHTTP(rrNull, reqNull)
	if rrNull.Code != http.StatusOK {
		t.Fatalf("POST null Json status = %d, want 200; body = %s", rrNull.Code, rrNull.Body.String())
	}
	var respNull FunctionCallResponse
	if err := json.Unmarshal(rrNull.Body.Bytes(), &respNull); err != nil {
		t.Fatal(err)
	}
	if respNull.Result != "none:-1:null" {
		t.Fatalf("null Json result = %v, want %q", respNull.Result, "none:-1:null")
	}
}
