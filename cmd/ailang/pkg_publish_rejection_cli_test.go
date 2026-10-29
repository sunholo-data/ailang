package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/pkg"
)

func TestPublishAuthCLI_ZeroUploadAndUnusedFile(t *testing.T) {
	binary := buildAilang(t)
	isolatePublishCLI(t)
	root, home := t.TempDir(), t.TempDir()
	writePublishProbePackage(t, root, "unused-probe")
	if err := os.Remove(filepath.Join(root, pkg.SmokeFile)); err != nil {
		t.Fatal(err)
	}
	std, err := filepath.Abs("../../std")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("AILANG_STDLIB_PATH", std)
	t.Setenv("AILANG_Z3_PATH", filepath.Join(home, "absent-z3.exe"))
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		_, _ = w.Write([]byte(`{"packages":[]}`))
	}))
	defer server.Close()
	t.Setenv(config.EnvRegistry, server.URL)
	t.Setenv(config.EnvRegistryValidator, server.URL)
	unsafe := filepath.Join(home, "unsafe-key")
	if err := os.WriteFile(unsafe, []byte("unsafe-dummy-key"), 0644); err != nil {
		t.Fatal(err)
	}
	absent := filepath.Join(home, "absent-key")
	before, err := pkg.ContentHash(root)
	if err != nil {
		t.Fatal(err)
	}
	runPublishRejections(t, binary, root, home, absent, unsafe, &requests)
	t.Setenv(config.EnvRegistryAPIKey, "unused-dummy-env")
	result := runCLIIn(t, binary, home, root, "pkg", "publish", "--key-file", absent, "--dry-run")
	if result.exitCode != 0 || !strings.Contains(result.stdout, "credentials were not loaded or verified") || strings.Contains(result.stderr, "is inherited") {
		t.Fatalf("unused credential accessed: %+v", result)
	}
	after, err := pkg.ContentHash(root)
	if err != nil || before != after {
		t.Fatal("author content changed")
	}
}

func runPublishRejections(t *testing.T, binary, root, home, absent, unsafe string, requests *atomic.Int32) {
	t.Helper()
	cases := []struct{ name, path, env, want string }{
		{"missing", absent, "", "publish --key-file"},
		{"conflict", absent, "dummy-conflict", "both --key-file"},
		{"unsafe", unsafe, "", "0400 or 0600"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(config.EnvRegistryAPIKey, tt.env)
			before := requests.Load()
			result := runCLIIn(t, binary, home, root, "publish", "--key-file", tt.path)
			want := tt.want
			if tt.env == "" && runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
				want = "unsupported"
			}
			if result.exitCode == 0 || !strings.Contains(result.stderr, want) || requests.Load() != before {
				t.Fatalf("credential failure reached package/HTTP work: %+v requests=%d", result, requests.Load())
			}
		})
	}
}
