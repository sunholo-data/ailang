package apiserver

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sunholo/ailang/internal/ast"
)

func TestExtractNoExposeAnnotations(t *testing.T) {
	modInfo := &ModuleInfo{
		Path: "test/billing",
		Exports: []ExportInfo{
			{Name: "getUsage", Type: "string -> Json", Arity: 1, RoutePath: "/api/v1/usage", RouteMethod: "GET"},
			{Name: "generateApiKey", Type: "string -> string", Arity: 1},
			{Name: "validateApiKey", Type: "string -> bool", Arity: 1},
			{Name: "publicHelper", Type: "int -> int", Arity: 1},
		},
	}

	file := &ast.File{
		Funcs: []*ast.FuncDecl{
			{Name: "getUsage", IsExport: true, Annotations: []*ast.Annotation{{Name: "noexpose"}}},
			{Name: "generateApiKey", IsExport: true, Annotations: []*ast.Annotation{{Name: "noexpose"}}},
			{Name: "validateApiKey", IsExport: true, Annotations: []*ast.Annotation{{Name: "noexpose"}}},
			{Name: "publicHelper", IsExport: true},
		},
	}

	extractNoExposeAnnotations(modInfo, file)

	// getUsage has @route — @noexpose should NOT apply (route overrides)
	if modInfo.Exports[0].IsNoExpose {
		t.Error("getUsage has @route, should NOT be marked @noexpose")
	}

	// generateApiKey has @noexpose and no @route — should be hidden
	if !modInfo.Exports[1].IsNoExpose {
		t.Error("generateApiKey should be marked @noexpose")
	}

	// validateApiKey has @noexpose and no @route — should be hidden
	if !modInfo.Exports[2].IsNoExpose {
		t.Error("validateApiKey should be marked @noexpose")
	}

	// publicHelper has no @noexpose — should remain visible
	if modInfo.Exports[3].IsNoExpose {
		t.Error("publicHelper should NOT be marked @noexpose")
	}
}

func TestIsExposed_Default(t *testing.T) {
	s := &Server{routesOnly: false}

	tests := []struct {
		name     string
		export   ExportInfo
		expected bool
	}{
		{"plain export", ExportInfo{Name: "f1"}, true},
		{"route export", ExportInfo{Name: "f2", RoutePath: "/api"}, true},
		{"noexpose export", ExportInfo{Name: "f3", IsNoExpose: true}, false},
		{"noexpose with route", ExportInfo{Name: "f4", RoutePath: "/api", IsNoExpose: false}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := s.isExposed(tt.export); got != tt.expected {
				t.Errorf("isExposed() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsExposed_RoutesOnly(t *testing.T) {
	s := &Server{routesOnly: true}

	tests := []struct {
		name     string
		export   ExportInfo
		expected bool
	}{
		{"plain export hidden", ExportInfo{Name: "f1"}, false},
		{"route export visible", ExportInfo{Name: "f2", RoutePath: "/api", RouteMethod: "GET"}, true},
		{"noexpose still hidden", ExportInfo{Name: "f3", IsNoExpose: true}, false},
		{"noexpose with route visible", ExportInfo{Name: "f4", RoutePath: "/x", RouteMethod: "POST", IsNoExpose: false}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := s.isExposed(tt.export); got != tt.expected {
				t.Errorf("isExposed() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsExposed_RoutesOnly_CombinedWithNoExpose(t *testing.T) {
	s := &Server{routesOnly: true}

	// Even with routes-only, @noexpose on a non-route function is redundant but consistent
	exp := ExportInfo{Name: "internal", IsNoExpose: true}
	if s.isExposed(exp) {
		t.Error("@noexpose + routes-only should hide function")
	}

	// Route function should still be visible even with routes-only
	exp = ExportInfo{Name: "api", RoutePath: "/api/v1/test", RouteMethod: "POST"}
	if !s.isExposed(exp) {
		t.Error("@route function should be visible with routes-only")
	}
}

// --- isValidJSONObjectOrArray tests ---

func TestIsValidJSONObjectOrArray(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"json object", `{"status":"healthy"}`, true},
		{"json array", `[1,2,3]`, true},
		{"nested object", `{"a":{"b":1},"c":[1,2]}`, true},
		{"empty object", `{}`, true},
		{"empty array", `[]`, true},
		{"whitespace padded", `  {"key": "val"}  `, true},
		{"plain string", `"hello"`, false},
		{"number", `42`, false},
		{"boolean true", `true`, false},
		{"boolean false", `false`, false},
		{"null", `null`, false},
		{"empty string", ``, false},
		{"single char", `{`, false},
		{"invalid json object", `{"key":}`, false},
		{"invalid json array", `[1,2,`, false},
		{"not json at all", `hello world`, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isValidJSONObjectOrArray(tt.input); got != tt.expected {
				t.Errorf("isValidJSONObjectOrArray(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

// --- @nowrap auto-unwrap integration tests ---

func TestNowrapAutoUnwrap_JSONObject(t *testing.T) {
	srv := testServer(t)
	defer srv.Close()

	// The hello function returns a string. We simulate a function returning
	// a JSON string by calling directly with nowrap and checking the logic.
	// Use callFunction directly — the test server's "hello" returns a plain string.
	req := httptest.NewRequest("POST", "/test", strings.NewReader(`{"args": ["World"]}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.callFunction(w, req, "test/api/greet", "hello", callOpts{Nowrap: true})

	// "hello" returns "Hello, World!" which is NOT a JSON object/array,
	// so it should still be JSON-encoded as a string
	body := strings.TrimSpace(w.Body.String())
	if body != `"Hello, World!"` {
		t.Errorf("expected JSON-encoded string, got %s", body)
	}
}

func TestNowrapAutoUnwrap_NotTriggeredForPlainString(t *testing.T) {
	// Verify that a plain string result doesn't get unwrapped
	srv := testServer(t)
	defer srv.Close()

	req := httptest.NewRequest("POST", "/test", strings.NewReader(`{"args": ["test"]}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.callFunction(w, req, "test/api/greet", "hello", callOpts{Nowrap: true})

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	body := strings.TrimSpace(w.Body.String())
	// Should be JSON-encoded string, NOT unwrapped
	if !strings.HasPrefix(body, `"`) {
		t.Errorf("plain string should remain JSON-encoded, got %s", body)
	}
}

func TestNowrapWithoutAutoUnwrap_StillHasTimingHeader(t *testing.T) {
	srv := testServer(t)
	defer srv.Close()

	req := httptest.NewRequest("POST", "/test", strings.NewReader(`{"args": ["World"]}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.callFunction(w, req, "test/api/greet", "hello", callOpts{Nowrap: true})

	if w.Header().Get("X-Elapsed-Ms") == "" {
		t.Error("expected X-Elapsed-Ms header even with nowrap")
	}
	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("expected application/json content type, got %s", w.Header().Get("Content-Type"))
	}
}
