package pkg

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/testutil"
)

func TestRegistryConfinement(t *testing.T) {
	home := t.TempDir()
	testutil.SetHomeDir(t, home)
	t.Setenv(config.EnvAgentPolicy, "policy.toml")
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls++; w.Write([]byte(`{}`)) }))
	defer server.Close()
	client := &RegistryClient{BaseURL: server.URL, httpClient: server.Client(), indexCache: &RegistryIndex{}}
	for _, fetch := range []func() error{
		func() error { _, err := client.FetchIndex(); return err },
		func() error { _, err := client.FetchPackage("test/lib", "0.1.0"); return err },
		func() error { _, err := client.FetchMetadata("test/lib", "0.1.0"); return err },
	} {
		if err := fetch(); err == nil || !strings.Contains(err.Error(), "confined by AILANG_AGENT_POLICY") {
			t.Fatalf("expected confinement refusal, got %v", err)
		}
	}
	if calls != 0 {
		t.Fatalf("network requests: %d", calls)
	}
	if _, err := RegistryCacheDir(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(home, ".ailang")); !os.IsNotExist(err) {
		t.Fatalf("HOME changed: %v", err)
	}
}

func TestRegistryCacheConfinementWrites(t *testing.T) {
	home := t.TempDir()
	testutil.SetHomeDir(t, home)
	t.Setenv(config.EnvAgentPolicy, "policy.toml")
	if _, err := EnsureRegistryCacheDir(); err == nil {
		t.Fatal("confined cache creation allowed")
	}
	if _, err := os.Stat(filepath.Join(home, ".ailang")); !os.IsNotExist(err) {
		t.Fatal("cache created")
	}
	expected := "confined by AILANG_AGENT_POLICY: writing the registry cache is operator authority — provision AILANG_PACKAGE_ROOT or run 'ailang install' outside the sandbox"
	if err := RefuseIfConfined("writing the registry cache"); err == nil || err.Error() != expected {
		t.Fatalf("refusal: %v", err)
	}
	t.Setenv(config.EnvAgentPolicy, "")
	dir, err := EnsureRegistryCacheDir()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dir); err != nil {
		t.Fatal(err)
	}
}

func TestRegistryConfinementResolverMiss(t *testing.T) {
	home := t.TempDir()
	testutil.SetHomeDir(t, home)
	t.Setenv(config.EnvAgentPolicy, "policy.toml")
	root := t.TempDir()
	writeManifest(t, root, `[package]
name = "test/app"
version = "0.1.0"
edition = "1"
[dependencies]
"test/lib" = "0.1.0"
`)
	m, err := LoadManifest(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ResolveDependencies(m, root); err == nil || !strings.Contains(err.Error(), "confined by AILANG_AGENT_POLICY") {
		t.Fatalf("expected refusal: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, ".ailang")); !os.IsNotExist(err) {
		t.Fatal("HOME changed")
	}
}
