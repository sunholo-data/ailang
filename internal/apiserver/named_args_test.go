package apiserver

import (
	"net/http/httptest"
	"testing"

	"github.com/sunholo/ailang/internal/ast"
)

func TestExtractParamNames(t *testing.T) {
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
					{Name: "path"},
					{Name: "outputFormat"},
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
					{Name: "input"},
					{Name: "maxSize"},
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

	extractParamNames(modInfo, file)

	// parseFile should have param names
	if len(modInfo.Exports[0].ParamNames) != 2 {
		t.Fatalf("expected 2 param names for parseFile, got %d", len(modInfo.Exports[0].ParamNames))
	}
	if modInfo.Exports[0].ParamNames[0] != "path" {
		t.Errorf("expected first param 'path', got %q", modInfo.Exports[0].ParamNames[0])
	}
	if modInfo.Exports[0].ParamNames[1] != "outputFormat" {
		t.Errorf("expected second param 'outputFormat', got %q", modInfo.Exports[0].ParamNames[1])
	}

	// health should have empty param names
	if len(modInfo.Exports[1].ParamNames) != 0 {
		t.Errorf("expected 0 param names for health, got %d", len(modInfo.Exports[1].ParamNames))
	}

	// convert should have param names
	if len(modInfo.Exports[2].ParamNames) != 2 {
		t.Fatalf("expected 2 param names for convert, got %d", len(modInfo.Exports[2].ParamNames))
	}
	if modInfo.Exports[2].ParamNames[0] != "input" {
		t.Errorf("expected first param 'input', got %q", modInfo.Exports[2].ParamNames[0])
	}
	if modInfo.Exports[2].ParamNames[1] != "maxSize" {
		t.Errorf("expected second param 'maxSize', got %q", modInfo.Exports[2].ParamNames[1])
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

	t.Run("exact match", func(t *testing.T) {
		body := map[string]interface{}{"path": "file.docx", "outputFormat": "blocks", "maxSize": float64(100)}
		args := parseNamedArgs(body, paramNames)
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
		args := parseNamedArgs(body, paramNames)
		if args == nil {
			t.Fatal("expected args, got nil")
		}
		if args[1] != "blocks" {
			t.Errorf("args[1] = %v, want 'blocks' (via snake_case)", args[1])
		}
	})

	t.Run("no match returns nil", func(t *testing.T) {
		body := map[string]interface{}{"unknown": "val", "other": "val2"}
		args := parseNamedArgs(body, paramNames)
		if args != nil {
			t.Errorf("expected nil for no matching keys, got %v", args)
		}
	})

	t.Run("partial match", func(t *testing.T) {
		body := map[string]interface{}{"path": "file.docx"}
		args := parseNamedArgs(body, paramNames)
		if args == nil {
			t.Fatal("expected args for partial match, got nil")
		}
		if args[0] != "file.docx" {
			t.Errorf("args[0] = %v, want 'file.docx'", args[0])
		}
		if args[1] != nil {
			t.Errorf("args[1] = %v, want nil for unmatched param", args[1])
		}
	})

	t.Run("empty param names", func(t *testing.T) {
		body := map[string]interface{}{"path": "file.docx"}
		args := parseNamedArgs(body, nil)
		if args != nil {
			t.Errorf("expected nil for empty param names, got %v", args)
		}
	})

	t.Run("extra keys ignored", func(t *testing.T) {
		body := map[string]interface{}{"path": "file.docx", "extra_field": "ignored"}
		args := parseNamedArgs(body, paramNames)
		if args == nil {
			t.Fatal("expected args, got nil")
		}
		if args[0] != "file.docx" {
			t.Errorf("args[0] = %v, want 'file.docx'", args[0])
		}
	})
}

func TestParseArgsWithNames(t *testing.T) {
	paramNames := []string{"path", "outputFormat"}

	t.Run("positional args take precedence", func(t *testing.T) {
		body := []byte(`{"args": ["file.docx", "blocks"]}`)
		args, err := parseArgsWithNames(body, paramNames)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(args) != 2 || args[0] != "file.docx" {
			t.Errorf("expected positional args, got %v", args)
		}
	})

	t.Run("named binding", func(t *testing.T) {
		body := []byte(`{"path": "data/sample.docx", "output_format": "blocks"}`)
		args, err := parseArgsWithNames(body, paramNames)
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
		args, err := parseArgsWithNames(body, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Should fall back to single-arg (the whole object)
		if len(args) != 1 {
			t.Fatalf("expected 1 arg (single object), got %d", len(args))
		}
	})

	t.Run("empty body", func(t *testing.T) {
		args, err := parseArgsWithNames(nil, paramNames)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(args) != 0 {
			t.Errorf("expected 0 args for empty body, got %d", len(args))
		}
	})

	t.Run("non-object JSON falls back", func(t *testing.T) {
		body := []byte(`"just a string"`)
		args, err := parseArgsWithNames(body, paramNames)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(args) != 1 || args[0] != "just a string" {
			t.Errorf("expected single string arg, got %v", args)
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
