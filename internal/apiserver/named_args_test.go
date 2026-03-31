package apiserver

import (
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/sunholo/ailang/internal/ast"
)

func TestExtractParamInfo(t *testing.T) {
	modInfo := &ModuleInfo{
		Path: "test/api",
		Exports: []ExportInfo{
			{Name: "parseFile", Type: "string -> string -> string", Arity: 2},
			{Name: "health", Type: "string", Arity: 0},
			{Name: "convert", Type: "string -> int -> string", Arity: 2},
		},
	}

	file := &ast.File{
		Funcs: []*ast.FuncDecl{
			{
				Name:     "parseFile",
				IsExport: true,
				Params: []*ast.Param{
					{Name: "path", Type: &ast.SimpleType{Name: "string"}},
					{Name: "outputFormat", Type: &ast.SimpleType{Name: "string"}},
				},
			},
			{
				Name:     "health",
				IsExport: true,
				Params:   []*ast.Param{}, // no params
			},
			{
				Name:     "convert",
				IsExport: true,
				Params: []*ast.Param{
					{Name: "input", Type: &ast.SimpleType{Name: "string"}},
					{Name: "maxSize", Type: &ast.SimpleType{Name: "int"}},
				},
			},
			{
				Name:     "internalHelper",
				IsExport: false, // not exported
				Params: []*ast.Param{
					{Name: "x"},
				},
			},
		},
	}

	extractParamInfo(modInfo, file)

	// parseFile should have param names and types
	if len(modInfo.Exports[0].ParamNames) != 2 {
		t.Fatalf("expected 2 param names for parseFile, got %d", len(modInfo.Exports[0].ParamNames))
	}
	if modInfo.Exports[0].ParamNames[0] != "path" {
		t.Errorf("expected first param 'path', got %q", modInfo.Exports[0].ParamNames[0])
	}
	if modInfo.Exports[0].ParamNames[1] != "outputFormat" {
		t.Errorf("expected second param 'outputFormat', got %q", modInfo.Exports[0].ParamNames[1])
	}
	if len(modInfo.Exports[0].ParamTypes) != 2 {
		t.Fatalf("expected 2 param types for parseFile, got %d", len(modInfo.Exports[0].ParamTypes))
	}
	if modInfo.Exports[0].ParamTypes[0] != "string" {
		t.Errorf("expected first param type 'string', got %q", modInfo.Exports[0].ParamTypes[0])
	}
	if modInfo.Exports[0].ParamTypes[1] != "string" {
		t.Errorf("expected second param type 'string', got %q", modInfo.Exports[0].ParamTypes[1])
	}

	// health should have empty param names and types
	if len(modInfo.Exports[1].ParamNames) != 0 {
		t.Errorf("expected 0 param names for health, got %d", len(modInfo.Exports[1].ParamNames))
	}
	if len(modInfo.Exports[1].ParamTypes) != 0 {
		t.Errorf("expected 0 param types for health, got %d", len(modInfo.Exports[1].ParamTypes))
	}

	// convert should have param names and types
	if len(modInfo.Exports[2].ParamNames) != 2 {
		t.Fatalf("expected 2 param names for convert, got %d", len(modInfo.Exports[2].ParamNames))
	}
	if modInfo.Exports[2].ParamNames[0] != "input" {
		t.Errorf("expected first param 'input', got %q", modInfo.Exports[2].ParamNames[0])
	}
	if modInfo.Exports[2].ParamNames[1] != "maxSize" {
		t.Errorf("expected second param 'maxSize', got %q", modInfo.Exports[2].ParamNames[1])
	}
	if modInfo.Exports[2].ParamTypes[0] != "string" {
		t.Errorf("expected first param type 'string', got %q", modInfo.Exports[2].ParamTypes[0])
	}
	if modInfo.Exports[2].ParamTypes[1] != "int" {
		t.Errorf("expected second param type 'int', got %q", modInfo.Exports[2].ParamTypes[1])
	}
}

func TestParamTypeToString(t *testing.T) {
	tests := []struct {
		name string
		typ  ast.Type
		want string
	}{
		{"nil type", nil, "unknown"},
		{"string", &ast.SimpleType{Name: "string"}, "string"},
		{"int", &ast.SimpleType{Name: "int"}, "int"},
		{"float", &ast.SimpleType{Name: "float"}, "float"},
		{"bool", &ast.SimpleType{Name: "bool"}, "bool"},
		{"list", &ast.ListType{}, "list"},
		{"array", &ast.ArrayType{}, "array"},
		{"record", &ast.RecordType{}, "record"},
		{"func type", &ast.FuncType{}, "unknown"},
		{"type var", &ast.TypeVar{Name: "a"}, "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := paramTypeToString(tt.typ)
			if got != tt.want {
				t.Errorf("paramTypeToString(%v) = %q, want %q", tt.typ, got, tt.want)
			}
		})
	}
}

func TestExtractParamInfoNilType(t *testing.T) {
	modInfo := &ModuleInfo{
		Path:    "test/api",
		Exports: []ExportInfo{{Name: "fn", Type: "a -> string", Arity: 1}},
	}
	file := &ast.File{
		Funcs: []*ast.FuncDecl{{
			Name:     "fn",
			IsExport: true,
			Params:   []*ast.Param{{Name: "x"}}, // no Type annotation
		}},
	}
	extractParamInfo(modInfo, file)
	if modInfo.Exports[0].ParamTypes[0] != "unknown" {
		t.Errorf("expected 'unknown' for nil type, got %q", modInfo.Exports[0].ParamTypes[0])
	}
}

func TestCamelToSnake(t *testing.T) {
	tests := []struct{ input, want string }{
		{"outputFormat", "output_format"},
		{"path", "path"},
		{"maxUploadSize", "max_upload_size"},
		{"x", "x"},
		{"URL", "u_r_l"}, // edge case: all-caps (no leading underscore)
		{"getHTTPResponse", "get_h_t_t_p_response"},
		{"", ""},
	}
	for _, tt := range tests {
		got := camelToSnake(tt.input)
		if got != tt.want {
			t.Errorf("camelToSnake(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestParseNamedArgs(t *testing.T) {
	paramNames := []string{"path", "outputFormat", "maxSize"}
	paramTypes := []string{"string", "string", "int"}

	t.Run("exact match", func(t *testing.T) {
		body := map[string]interface{}{"path": "file.docx", "outputFormat": "blocks", "maxSize": float64(100)}
		args := parseNamedArgs(body, paramNames, paramTypes)
		if len(args) != 3 {
			t.Fatalf("expected 3 args, got %d", len(args))
		}
		if args[0] != "file.docx" {
			t.Errorf("args[0] = %v, want 'file.docx'", args[0])
		}
		if args[1] != "blocks" {
			t.Errorf("args[1] = %v, want 'blocks'", args[1])
		}
	})

	t.Run("snake_case match", func(t *testing.T) {
		body := map[string]interface{}{"path": "file.docx", "output_format": "blocks"}
		args := parseNamedArgs(body, paramNames, paramTypes)
		if args == nil {
			t.Fatal("expected args, got nil")
		}
		if args[1] != "blocks" {
			t.Errorf("args[1] = %v, want 'blocks' (via snake_case)", args[1])
		}
	})

	t.Run("no match returns nil", func(t *testing.T) {
		body := map[string]interface{}{"unknown": "val", "other": "val2"}
		args := parseNamedArgs(body, paramNames, paramTypes)
		if args != nil {
			t.Errorf("expected nil for no matching keys, got %v", args)
		}
	})

	t.Run("partial match pads zero-values", func(t *testing.T) {
		body := map[string]interface{}{"path": "file.docx"}
		args := parseNamedArgs(body, paramNames, paramTypes)
		if args == nil {
			t.Fatal("expected args for partial match, got nil")
		}
		if args[0] != "file.docx" {
			t.Errorf("args[0] = %v, want 'file.docx'", args[0])
		}
		if args[1] != "" {
			t.Errorf("args[1] = %v, want empty string for missing string param", args[1])
		}
		if args[2] != float64(0) {
			t.Errorf("args[2] = %v, want 0 for missing int param", args[2])
		}
	})

	t.Run("partial match without types falls back to nil", func(t *testing.T) {
		body := map[string]interface{}{"path": "file.docx"}
		args := parseNamedArgs(body, paramNames, nil)
		if args == nil {
			t.Fatal("expected args for partial match, got nil")
		}
		if args[1] != nil {
			t.Errorf("args[1] = %v, want nil when no type info", args[1])
		}
	})

	t.Run("empty param names", func(t *testing.T) {
		body := map[string]interface{}{"path": "file.docx"}
		args := parseNamedArgs(body, nil, nil)
		if args != nil {
			t.Errorf("expected nil for empty param names, got %v", args)
		}
	})

	t.Run("extra keys ignored", func(t *testing.T) {
		body := map[string]interface{}{"path": "file.docx", "extra_field": "ignored"}
		args := parseNamedArgs(body, paramNames, paramTypes)
		if args == nil {
			t.Fatal("expected args, got nil")
		}
		if args[0] != "file.docx" {
			t.Errorf("args[0] = %v, want 'file.docx'", args[0])
		}
	})
}

func TestZeroValueForType(t *testing.T) {
	tests := []struct {
		typeName string
		want     interface{}
	}{
		{"string", ""},
		{"int", float64(0)},
		{"float", float64(0)},
		{"bool", false},
		{"list", []interface{}{}},
		{"array", []interface{}{}},
		{"record", map[string]interface{}{}},
		{"unknown", nil},
		{"CustomType", nil},
	}
	for _, tt := range tests {
		t.Run(tt.typeName, func(t *testing.T) {
			got := zeroValueForType(tt.typeName)
			if fmt.Sprintf("%v", got) != fmt.Sprintf("%v", tt.want) {
				t.Errorf("zeroValueForType(%q) = %v (%T), want %v (%T)", tt.typeName, got, got, tt.want, tt.want)
			}
		})
	}
}

func TestParseArgsWithNames(t *testing.T) {
	paramNames := []string{"path", "outputFormat"}
	paramTypes := []string{"string", "string"}

	t.Run("positional args take precedence", func(t *testing.T) {
		body := []byte(`{"args": ["file.docx", "blocks"]}`)
		args, err := parseArgsWithNames(body, paramNames, paramTypes)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(args) != 2 || args[0] != "file.docx" {
			t.Errorf("expected positional args, got %v", args)
		}
	})

	t.Run("positional args padded with zero-values", func(t *testing.T) {
		paramNames3 := []string{"a", "b", "c"}
		paramTypes3 := []string{"string", "int", "bool"}
		body := []byte(`{"args": ["hello"]}`)
		args, err := parseArgsWithNames(body, paramNames3, paramTypes3)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(args) != 3 {
			t.Fatalf("expected 3 args (padded), got %d", len(args))
		}
		if args[0] != "hello" {
			t.Errorf("args[0] = %v, want 'hello'", args[0])
		}
		if args[1] != float64(0) {
			t.Errorf("args[1] = %v, want 0 for missing int param", args[1])
		}
		if args[2] != false {
			t.Errorf("args[2] = %v, want false for missing bool param", args[2])
		}
	})

	t.Run("named binding", func(t *testing.T) {
		body := []byte(`{"path": "data/sample.docx", "output_format": "blocks"}`)
		args, err := parseArgsWithNames(body, paramNames, paramTypes)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(args) != 2 {
			t.Fatalf("expected 2 args, got %d", len(args))
		}
		if args[0] != "data/sample.docx" {
			t.Errorf("args[0] = %v, want 'data/sample.docx'", args[0])
		}
		if args[1] != "blocks" {
			t.Errorf("args[1] = %v, want 'blocks'", args[1])
		}
	})

	t.Run("no param names falls back to parseArgs", func(t *testing.T) {
		body := []byte(`{"path": "file.docx"}`)
		args, err := parseArgsWithNames(body, nil, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Should fall back to single-arg (the whole object)
		if len(args) != 1 {
			t.Fatalf("expected 1 arg (single object), got %d", len(args))
		}
	})

	t.Run("empty body", func(t *testing.T) {
		args, err := parseArgsWithNames(nil, paramNames, paramTypes)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(args) != 0 {
			t.Errorf("expected 0 args for empty body, got %d", len(args))
		}
	})

	t.Run("non-object JSON falls back to single arg", func(t *testing.T) {
		// A raw string body is passed through as a single argument
		// (zero-value padding only applies to JSON object bodies)
		body := []byte(`"just a string"`)
		args, err := parseArgsWithNames(body, paramNames, paramTypes)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(args) != 1 || args[0] != "just a string" {
			t.Errorf("expected single string arg, got %v", args)
		}
	})
}

func TestParseArgsWithNames_UnmatchedKeysZeroValuePadding(t *testing.T) {
	t.Run("single string param with empty body object", func(t *testing.T) {
		// POST {} to func foo(apiKey: string) should get "" not a Record
		body := []byte(`{}`)
		args, err := parseArgsWithNames(body, []string{"apiKey"}, []string{"string"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(args) != 1 {
			t.Fatalf("expected 1 arg, got %d", len(args))
		}
		if args[0] != "" {
			t.Errorf("args[0] = %v (%T), want empty string", args[0], args[0])
		}
	})

	t.Run("single string param with non-matching keys", func(t *testing.T) {
		// POST {"foo":"bar"} to func foo(apiKey: string) should get ""
		body := []byte(`{"foo":"bar"}`)
		args, err := parseArgsWithNames(body, []string{"apiKey"}, []string{"string"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(args) != 1 {
			t.Fatalf("expected 1 arg, got %d", len(args))
		}
		if args[0] != "" {
			t.Errorf("args[0] = %v (%T), want empty string", args[0], args[0])
		}
	})

	t.Run("multi param with non-matching keys", func(t *testing.T) {
		// POST {"x":"y"} to func bar(name: string, count: int) should get ["", 0]
		body := []byte(`{"x":"y"}`)
		args, err := parseArgsWithNames(body, []string{"name", "count"}, []string{"string", "int"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(args) != 2 {
			t.Fatalf("expected 2 args, got %d", len(args))
		}
		if args[0] != "" {
			t.Errorf("args[0] = %v, want empty string", args[0])
		}
		if args[1] != float64(0) {
			t.Errorf("args[1] = %v, want 0", args[1])
		}
	})

	t.Run("no params — raw passthrough preserved", func(t *testing.T) {
		// func noParams() with POST {"key":"val"} — should get raw object
		body := []byte(`{"key":"val"}`)
		args, err := parseArgsWithNames(body, nil, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(args) != 1 {
			t.Fatalf("expected 1 arg (raw passthrough), got %d", len(args))
		}
		m, ok := args[0].(map[string]interface{})
		if !ok {
			t.Fatalf("args[0] type = %T, want map[string]interface{}", args[0])
		}
		if m["key"] != "val" {
			t.Errorf("args[0][key] = %v, want val", m["key"])
		}
	})

	t.Run("matching key still works", func(t *testing.T) {
		// POST {"apiKey":"secret"} to func foo(apiKey: string) — should get "secret"
		body := []byte(`{"apiKey":"secret"}`)
		args, err := parseArgsWithNames(body, []string{"apiKey"}, []string{"string"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(args) != 1 {
			t.Fatalf("expected 1 arg, got %d", len(args))
		}
		if args[0] != "secret" {
			t.Errorf("args[0] = %v, want 'secret'", args[0])
		}
	})
}

func TestNowrapHeaders_ExtractsFromGoMap(t *testing.T) {
	goResult := map[string]interface{}{
		"data":  "parsed content",
		"count": float64(15),
		"_headers": map[string]interface{}{
			"X-Request-Id":          "req_abc123",
			"X-RateLimit-Remaining": "99",
		},
	}

	w := httptest.NewRecorder()
	w.Header().Set("Content-Type", "application/json")

	if headersVal, ok := goResult["_headers"]; ok {
		if headers, ok := headersVal.(map[string]interface{}); ok {
			for k, v := range headers {
				if sv, ok := v.(string); ok {
					w.Header().Set(k, sv)
				}
			}
		}
		delete(goResult, "_headers")
	}

	if w.Header().Get("X-Request-Id") != "req_abc123" {
		t.Errorf("expected X-Request-Id header 'req_abc123', got %q", w.Header().Get("X-Request-Id"))
	}
	if w.Header().Get("X-RateLimit-Remaining") != "99" {
		t.Errorf("expected X-RateLimit-Remaining header '99', got %q", w.Header().Get("X-RateLimit-Remaining"))
	}
	if _, ok := goResult["_headers"]; ok {
		t.Error("expected _headers to be removed from result map")
	}
	if goResult["data"] != "parsed content" {
		t.Errorf("expected data field preserved, got %v", goResult["data"])
	}
}

func TestNowrapHeaders_NoHeaders(t *testing.T) {
	goResult := map[string]interface{}{
		"data": "content",
	}

	w := httptest.NewRecorder()
	if headersVal, ok := goResult["_headers"]; ok {
		if headers, ok := headersVal.(map[string]interface{}); ok {
			for k, v := range headers {
				if sv, ok := v.(string); ok {
					w.Header().Set(k, sv)
				}
			}
		}
		delete(goResult, "_headers")
	}

	if len(w.Header()) != 0 {
		t.Errorf("expected no headers for result without _headers, got %v", w.Header())
	}
	if goResult["data"] != "content" {
		t.Error("result should be unchanged")
	}
}
