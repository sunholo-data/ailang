package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/sunholo-data/ailang/internal/pkg"
)

func TestPkgSmokeStaging_QualityAndDryRun(t *testing.T) {
	bin := buildAilang(t)
	root := t.TempDir()
	fixture, err := filepath.Abs("../../internal/pkg/testdata/smoke_staged_lock")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.CopyFS(root, os.DirFS(fixture)); err != nil {
		t.Fatal(err)
	}
	var posts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			posts.Add(1)
			http.Error(w, "no publish authorized", http.StatusMethodNotAllowed)
			return
		}
		if r.URL.Path == "/index.json" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"packages":[]}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()
	t.Setenv("AILANG_REGISTRY", server.URL)
	t.Setenv("AILANG_PACKAGE_ROOT", filepath.Join(root, "registry"))
	t.Setenv("AILANG_NO_CACHE", "1")
	std, err := filepath.Abs("../../std")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("AILANG_STDLIB_PATH", std)
	author := filepath.Join(root, "a")
	home := t.TempDir()
	lock := runCLIIn(t, bin, home, author, "lock")
	if lock.exitCode != 0 {
		t.Fatalf("lock: %+v", lock)
	}
	original := make(map[string]string)
	for _, name := range []string{pkg.ManifestFile, pkg.LockFileName} {
		data, err := os.ReadFile(filepath.Join(author, name))
		if err != nil {
			t.Fatal(err)
		}
		original[name] = string(data)
	}
	originalContent, err := pkg.ContentHash(author)
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"quality", "dry-run"} {
		t.Run(mode, func(t *testing.T) {
			args := []string{"publish", "--dry-run"}
			if mode == "quality" {
				args = []string{"pkg", "quality", "--json", "."}
			}
			result := runCLIIn(t, bin, home, author, args...)
			if result.exitCode != 0 {
				t.Fatalf("%s failed: code=%d stdout=%s stderr=%s", mode, result.exitCode, result.stdout, result.stderr)
			}
			for name, want := range original {
				after, err := os.ReadFile(filepath.Join(author, name))
				if err != nil {
					t.Fatal(err)
				}
				if string(after) != want {
					t.Fatalf("%s changed author %s", mode, name)
				}
			}
			afterContent, err := pkg.ContentHash(author)
			if err != nil {
				t.Fatal(err)
			}
			if afterContent != originalContent {
				t.Fatalf("%s changed author content", mode)
			}
			if mode == "quality" {
				var report pkg.QualityReport
				if err := json.Unmarshal([]byte(result.stdout), &report); err != nil {
					t.Fatalf("quality stdout must be only JSON: %v\n%s", err, result.stdout)
				}
				if report.Smoke == nil || !report.Smoke.Passed {
					t.Fatalf("smoke not passed: %+v", report.Smoke)
				}
			}
		})
	}
	if posts.Load() != 0 {
		t.Fatalf("unexpected publish requests: %d", posts.Load())
	}
}
