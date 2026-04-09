package apiserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// testServer creates an API server loaded with a test module.
func testServer(t *testing.T) *Server {
	t.Helper()

	// Create a temporary directory with a test AILANG module
	tmpDir := t.TempDir()

	// Create module directory structure
	apiDir := filepath.Join(tmpDir, "test", "api")
	if err := os.MkdirAll(apiDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Write a simple test module
	modContent := `module test/api/greet

export pure func hello(name: string) -> string =
  "Hello, " ++ name ++ "!"

export pure func add(x: int, y: int) -> int =
  x + y
`
	modPath := filepath.Join(apiDir, "greet.ail")
	if err := os.WriteFile(modPath, []byte(modContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Need stdlib available
	stdlibPath := os.Getenv("AILANG_STDLIB_PATH")
	if stdlibPath == "" {
		// Try to find stdlib relative to test
		cwd, _ := os.Getwd()
		// Walk up to find project root
		for dir := cwd; dir != filepath.Dir(dir); dir = filepath.Dir(dir) {
			if _, err := os.Stat(filepath.Join(dir, "stdlib")); err == nil {
				stdlibPath = dir
				break
			}
		}
	}
	if stdlibPath != "" {
		os.Setenv("AILANG_STDLIB_PATH", stdlibPath)
	}

	srv := New(tmpDir, Config{Port: "0", CORS: true})

	if err := srv.LoadModules([]string{modPath}); err != nil {
		t.Fatalf("LoadModules: %v", err)
	}

	return srv
}

func TestHealthEndpoint(t *testing.T) {
	srv := testServer(t)
	defer srv.Close()

	mux := srv.buildRoutes()
	req := httptest.NewRequest("GET", "/api/_health", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp HealthResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}

	if resp.Status != "ok" {
		t.Errorf("expected status 'ok', got %q", resp.Status)
	}
	if resp.ModulesCount != 1 {
		t.Errorf("expected 1 module, got %d", resp.ModulesCount)
	}
	if resp.ExportsCount < 2 {
		t.Errorf("expected at least 2 exports, got %d", resp.ExportsCount)
	}
}

func TestListModules(t *testing.T) {
	srv := testServer(t)
	defer srv.Close()

	mux := srv.buildRoutes()
	req := httptest.NewRequest("GET", "/api/_meta/modules", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp ModulesListResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}

	if resp.Count != 1 {
		t.Fatalf("expected 1 module, got %d", resp.Count)
	}

	mod := resp.Modules[0]
	if mod.Path != "test/api/greet" {
		t.Errorf("expected module path 'test/api/greet', got %q", mod.Path)
	}

	// Check exports
	exportNames := make(map[string]bool)
	for _, e := range mod.Exports {
		exportNames[e.Name] = true
	}

	if !exportNames["hello"] {
		t.Error("expected 'hello' export")
	}
	if !exportNames["add"] {
		t.Error("expected 'add' export")
	}
}

func TestFunctionCall(t *testing.T) {
	srv := testServer(t)
	defer srv.Close()

	mux := srv.buildRoutes()

	t.Run("call hello with args array", func(t *testing.T) {
		body := `{"args": ["World"]}`
		req := httptest.NewRequest("POST", "/api/test/api/greet/hello", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var resp FunctionCallResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatal(err)
		}

		if resp.Error != "" {
			t.Fatalf("unexpected error: %s", resp.Error)
		}
		if resp.Module != "test/api/greet" {
			t.Errorf("expected module 'test/api/greet', got %q", resp.Module)
		}
		if resp.Func != "hello" {
			t.Errorf("expected func 'hello', got %q", resp.Func)
		}

		resultStr, ok := resp.Result.(string)
		if !ok {
			t.Fatalf("expected string result, got %T: %v", resp.Result, resp.Result)
		}
		if resultStr != "Hello, World!" {
			t.Errorf("expected 'Hello, World!', got %q", resultStr)
		}
	})

	t.Run("call add with args array", func(t *testing.T) {
		body := `{"args": [3, 4]}`
		req := httptest.NewRequest("POST", "/api/test/api/greet/add", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var resp FunctionCallResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatal(err)
		}

		if resp.Error != "" {
			t.Fatalf("unexpected error: %s", resp.Error)
		}

		// JSON numbers are float64 but our converter detects whole numbers → int
		resultNum, ok := resp.Result.(float64)
		if !ok {
			t.Fatalf("expected number result, got %T: %v", resp.Result, resp.Result)
		}
		if resultNum != 7 {
			t.Errorf("expected 7, got %v", resultNum)
		}
	})

	t.Run("call with single value body", func(t *testing.T) {
		// Passing just a string should work as single argument
		body := `"Alice"`
		req := httptest.NewRequest("POST", "/api/test/api/greet/hello", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var resp FunctionCallResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatal(err)
		}

		if resp.Error != "" {
			t.Fatalf("unexpected error: %s", resp.Error)
		}
		if resp.Result != "Hello, Alice!" {
			t.Errorf("expected 'Hello, Alice!', got %q", resp.Result)
		}
	})
}

func TestFunctionCallErrors(t *testing.T) {
	srv := testServer(t)
	defer srv.Close()

	mux := srv.buildRoutes()

	t.Run("unknown module", func(t *testing.T) {
		body := `{"args": ["test"]}`
		req := httptest.NewRequest("POST", "/api/nonexistent/module/func", strings.NewReader(body))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("unknown function", func(t *testing.T) {
		body := `{"args": ["test"]}`
		req := httptest.NewRequest("POST", "/api/test/api/greet/nonexistent", strings.NewReader(body))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("GET allowed for function calls", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/test/api/greet/hello", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		// GET is now allowed — should attempt the call (not 405)
		if w.Code == http.StatusMethodNotAllowed {
			t.Fatalf("GET should be allowed for function calls, got 405")
		}
	})

	t.Run("DELETE not allowed for function calls", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/test/api/greet/hello", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Fatalf("expected 405, got %d", w.Code)
		}
	})
}

func TestCORSHeaders(t *testing.T) {
	srv := testServer(t)
	defer srv.Close()

	mux := srv.buildRoutes()

	t.Run("OPTIONS preflight", func(t *testing.T) {
		req := httptest.NewRequest("OPTIONS", "/api/_health", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusNoContent {
			t.Fatalf("expected 204, got %d", w.Code)
		}

		if w.Header().Get("Access-Control-Allow-Origin") != "*" {
			t.Error("expected CORS origin header")
		}
	})

	t.Run("CORS on regular request", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/_health", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Header().Get("Access-Control-Allow-Origin") != "*" {
			t.Error("expected CORS origin header on regular request")
		}
	})
}

func TestModuleDetail(t *testing.T) {
	srv := testServer(t)
	defer srv.Close()

	mux := srv.buildRoutes()

	t.Run("existing module", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/_meta/modules/test/api/greet", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var mod ModuleInfo
		if err := json.NewDecoder(w.Body).Decode(&mod); err != nil {
			t.Fatal(err)
		}

		if mod.Path != "test/api/greet" {
			t.Errorf("expected path 'test/api/greet', got %q", mod.Path)
		}
	})

	t.Run("nonexistent module", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/_meta/modules/no/such/module", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})
}

func TestParseArgs(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		wantLen int
		wantErr bool
	}{
		{"empty body", "", 0, false},
		{"args array", `{"args": [1, "hello", true]}`, 3, false},
		{"single string", `"hello"`, 1, false},
		{"single number", `42`, 1, false},
		{"single object", `{"key": "value"}`, 1, false},
		{"invalid json", `{bad`, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args, err := parseArgs([]byte(tt.body))
			if tt.wantErr {
				if err == nil {
					t.Error("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(args) != tt.wantLen {
				t.Errorf("expected %d args, got %d", tt.wantLen, len(args))
			}
		})
	}
}

func TestCountFunctionArity(t *testing.T) {
	tests := []struct {
		typeStr string
		want    int
	}{
		{"string -> string", 1},
		{"string -> string -> int", 2},
		{"int", -1},
		{"(int, string) -> Result[ApiResponse, Error]", 1},
	}

	for _, tt := range tests {
		t.Run(tt.typeStr, func(t *testing.T) {
			got := countFunctionArity(tt.typeStr)
			if got != tt.want {
				t.Errorf("countFunctionArity(%q) = %d, want %d", tt.typeStr, got, tt.want)
			}
		})
	}
}
