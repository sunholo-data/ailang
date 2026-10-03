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

// #1449: an integer division by zero inside a served function answers the
// request with a structured 500 — RT001 in `error` and in error_detail.code —
// instead of panicking the request goroutine (which net/http turns into a
// dropped connection).
func TestDivZero_StructuredError(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("AILANG_STDLIB_PATH", filepath.Join(root, "std"))
	tmpDir := t.TempDir()
	apiDir := filepath.Join(tmpDir, "test", "api")
	if err := os.MkdirAll(apiDir, 0o755); err != nil {
		t.Fatal(err)
	}
	modPath := filepath.Join(apiDir, "divz.ail")
	src := "module test/api/divz\n\nexport pure func boom(n: int) -> bool = 10 / n > 0\n"
	if err := os.WriteFile(modPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	srv := New(tmpDir, Config{Port: "0"})
	t.Cleanup(func() { srv.Close() })
	if err := srv.LoadModules([]string{modPath}); err != nil {
		t.Fatalf("LoadModules: %v", err)
	}

	req := httptest.NewRequest("POST", "/api/test/api/divz/boom", strings.NewReader(`{"args": [0]}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.callFunction(w, req, "test/api/divz", "boom")

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status %d, want 500: %s", w.Code, w.Body.String())
	}
	var resp FunctionCallResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode %s: %v", w.Body.String(), err)
	}
	if !strings.Contains(resp.Error, "RT001: integer division by zero at") || !strings.Contains(resp.Error, "divz.ail:3:") {
		t.Errorf("error %q, want RT001 with the divz.ail:3 position", resp.Error)
	}
	if resp.ErrorDetail == nil || resp.ErrorDetail.Code != "RT001" || resp.ErrorDetail.Retryable {
		t.Errorf("error_detail = %+v, want code RT001, not retryable", resp.ErrorDetail)
	}
}
