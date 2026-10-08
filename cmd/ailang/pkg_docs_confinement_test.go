package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/pkg"
)

func TestPkgDocsConfinement(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv(config.EnvAgentPolicy, "policy.toml")
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls++; http.Error(w, "unexpected fetch", 500) }))
	defer server.Close()
	t.Setenv("AILANG_REGISTRY", server.URL)
	err := pkgDocsCommand([]string{"test/missing"})
	if err == nil || !strings.Contains(err.Error(), "confined by AILANG_AGENT_POLICY") {
		t.Fatalf("expected refusal, got %v", err)
	}
	if calls != 0 {
		t.Fatalf("network requests: %d", calls)
	}
	if _, err := os.Stat(filepath.Join(home, ".ailang")); !os.IsNotExist(err) {
		t.Fatalf("HOME changed: %v", err)
	}
}

func TestPkgRegistryUnconfinedWorkflows(t *testing.T) {
	t.Setenv(config.EnvAgentPolicy, "")
	for _, op := range []string{"docs", "install", "lock"} {
		t.Run(op, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			t.Setenv("USERPROFILE", home)
			src := t.TempDir()
			for name, data := range map[string]string{"ailang.toml": "[package]\nname = \"test/lib\"\nversion = \"0.1.0\"\nedition = \"1\"\n", "AGENT.md": "Test package guide"} {
				if err := os.WriteFile(filepath.Join(src, name), []byte(data), 0644); err != nil {
					t.Fatal(err)
				}
			}
			tarball, err := pkg.CreateTarball(src)
			if err != nil {
				t.Fatal(err)
			}
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				switch r.URL.Path {
				case "/index.json":
					w.Write([]byte(`{"packages":[{"name":"test/lib","latest":"0.1.0"}]}`))
				case "/packages/test/lib/0.1.0/metadata.json":
					w.Write([]byte(`{"manifest":{}}`))
				case "/packages/test/lib/0.1.0/package.tar.gz":
					w.Write(tarball)
				default:
					http.NotFound(w, r)
				}
			}))
			defer server.Close()
			t.Setenv("AILANG_REGISTRY", server.URL)
			root := t.TempDir()
			t.Chdir(root)
			switch op {
			case "docs":
				err = pkgDocsCommand([]string{"test/lib"})
			case "install":
				err = pkgInstallCommand([]string{"--no-bin", "test/lib@0.1.0"})
			case "lock":
				if err := os.WriteFile(filepath.Join(root, "ailang.toml"), []byte("[package]\nname = \"test/app\"\nversion = \"0.1.0\"\nedition = \"1\"\n[dependencies]\n\"test/lib\" = \"0.1.0\"\n"), 0644); err != nil {
					t.Fatal(err)
				}
				err = pkgLockCommand(nil)
			}
			if err != nil {
				t.Fatal(err)
			}
			if calls == 0 {
				t.Fatal("workflow made no requests")
			}
			cached, err := pkg.CachedPackagePath("test/lib", "0.1.0")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(filepath.Join(cached, "AGENT.md")); err != nil {
				t.Fatal(err)
			}
		})
	}
}
