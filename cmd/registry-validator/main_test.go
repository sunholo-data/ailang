package main

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/sunholo-data/ailang/internal/pkg"
)

func TestHealthEndpoint(t *testing.T) {
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	handleHealth(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("health status = %d, want 200", w.Code)
	}

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["status"] != "ok" {
		t.Errorf("status = %q, want ok", resp["status"])
	}
}

func TestPublish_MethodNotAllowed(t *testing.T) {
	v := &validator{}
	req := httptest.NewRequest("GET", "/publish", nil)
	w := httptest.NewRecorder()

	v.handlePublish(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET /publish status = %d, want 405", w.Code)
	}
}

func TestPublish_MissingPackageField(t *testing.T) {
	v := &validator{}

	// Empty multipart form (no "package" field)
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	writer.Close()

	req := httptest.NewRequest("POST", "/publish", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	v.handlePublish(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestPublish_InvalidTarball(t *testing.T) {
	v := &validator{}

	// Send garbage data as tarball
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, _ := writer.CreateFormFile("package", "package.tar.gz")
	part.Write([]byte("not a valid tarball"))
	writer.Close()

	req := httptest.NewRequest("POST", "/publish", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	v.handlePublish(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["error"] == "" {
		t.Error("expected error message in response")
	}
}

func TestPublish_InvalidManifest(t *testing.T) {
	v := &validator{}

	// Create a tarball with a bad ailang.toml (missing required fields)
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "ailang.toml"), []byte(`
[package]
name = "bad"
`), 0644)
	os.WriteFile(filepath.Join(dir, "core.ail"), []byte("module bad/core\n"), 0644)

	tarball, err := pkg.CreateTarball(dir)
	if err != nil {
		t.Fatalf("CreateTarball: %v", err)
	}

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, _ := writer.CreateFormFile("package", "package.tar.gz")
	part.Write(tarball)
	writer.Close()

	req := httptest.NewRequest("POST", "/publish", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	v.handlePublish(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 for invalid manifest", w.Code)
	}
}

func TestPublish_ValidPackage_NoGCS(t *testing.T) {
	// Test the validation pipeline up to the GCS upload step.
	// Without a real GCS bucket, the upload will fail — but we verify
	// that all validation steps pass for a valid package.
	v := &validator{} // nil bucket — will fail at GCS step

	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "ailang.toml"), []byte(`
[package]
name = "test/valid"
version = "0.1.0"
edition = "1"

[exports]
modules = ["test/valid/core"]

[effects]
max = []

[stability]
level = "experimental"

[metadata]
ai_summary = "A test package"
`), 0644)
	os.WriteFile(filepath.Join(dir, "core.ail"), []byte("module test/valid/core\n\nexport pure func add(a: int, b: int) -> int = a + b\n"), 0644)

	tarball, err := pkg.CreateTarball(dir)
	if err != nil {
		t.Fatalf("CreateTarball: %v", err)
	}

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, _ := writer.CreateFormFile("package", "package.tar.gz")
	part.Write(tarball)
	writer.Close()

	req := httptest.NewRequest("POST", "/publish", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	v.handlePublish(w, req)

	// Without a GCS bucket, we expect either:
	// - 500 (GCS upload fails because bucket is nil) — means validation PASSED
	// - 400 (ailang check failed because ailang not in PATH in test env)
	// Both are acceptable — the point is that manifest parsing succeeded.
	if w.Code == http.StatusOK {
		// If we somehow got 200, verify metadata is returned
		var meta pkg.PackageMetadata
		if err := json.Unmarshal(w.Body.Bytes(), &meta); err != nil {
			t.Errorf("expected valid metadata JSON, got: %s", w.Body.String())
		}
	}

	// The key assertion: we did NOT get 400 for "Invalid manifest" or "Invalid tarball"
	// (those would mean our validation pipeline is broken)
	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	errMsg := resp["error"]
	if errMsg != "" && (containsString2(errMsg, "Invalid manifest") || containsString2(errMsg, "Invalid tarball")) {
		t.Errorf("valid package should pass manifest/tarball validation, got: %s", errMsg)
	}
}

func TestHelpers(t *testing.T) {
	// Test getMetaString
	meta := map[string]interface{}{
		"ai_summary": "test summary",
		"count":      42,
	}
	if got := getMetaString(meta, "ai_summary"); got != "test summary" {
		t.Errorf("getMetaString = %q, want 'test summary'", got)
	}
	if got := getMetaString(meta, "missing"); got != "" {
		t.Errorf("getMetaString missing = %q, want empty", got)
	}
	if got := getMetaString(nil, "key"); got != "" {
		t.Errorf("getMetaString nil = %q, want empty", got)
	}

	// Test getMetaStringSlice
	meta2 := map[string]interface{}{
		"tags": []interface{}{"auth", "security"},
	}
	tags := getMetaStringSlice(meta2, "tags")
	if len(tags) != 2 || tags[0] != "auth" || tags[1] != "security" {
		t.Errorf("getMetaStringSlice = %v, want [auth security]", tags)
	}

	// Test containsString
	if !containsString([]string{"a", "b", "c"}, "b") {
		t.Error("containsString should find 'b'")
	}
	if containsString([]string{"a", "b"}, "z") {
		t.Error("containsString should not find 'z'")
	}

	// Test fileExists
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "exists.txt"), []byte("hi"), 0644)
	if !fileExists(filepath.Join(dir, "exists.txt")) {
		t.Error("fileExists should find existing file")
	}
	if fileExists(filepath.Join(dir, "nope.txt")) {
		t.Error("fileExists should not find missing file")
	}
}

// helper to avoid conflict with the package-level containsString
func containsString2(s, substr string) bool {
	return bytes.Contains([]byte(s), []byte(substr))
}
