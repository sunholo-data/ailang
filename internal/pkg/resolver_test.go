package pkg

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeManifest(t *testing.T, dir, content string) {
	t.Helper()
	os.MkdirAll(dir, 0755)
	os.WriteFile(filepath.Join(dir, ManifestFile), []byte(content), 0644)
}

func TestResolveDependencies_SinglePathDep(t *testing.T) {
	root := t.TempDir()

	// Create dependency package
	libDir := filepath.Join(root, "lib")
	writeManifest(t, libDir, `
[package]
name = "test/lib"
version = "0.2.0"
edition = "1"
`)
	os.WriteFile(filepath.Join(libDir, "core.ail"), []byte("module test/lib/core\n"), 0644)

	// Create root package
	appDir := filepath.Join(root, "app")
	writeManifest(t, appDir, `
[package]
name = "test/app"
version = "0.1.0"
edition = "1"

[dependencies]
"test/lib" = { path = "../lib" }
`)

	manifest, err := LoadManifest(appDir)
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}

	resolved, err := ResolveDependencies(manifest, appDir)
	if err != nil {
		t.Fatalf("ResolveDependencies: %v", err)
	}

	if len(resolved) != 1 {
		t.Fatalf("expected 1 resolved, got %d", len(resolved))
	}
	if resolved[0].Name != "test/lib" {
		t.Errorf("name = %q, want test/lib", resolved[0].Name)
	}
	if resolved[0].Source != "path" {
		t.Errorf("source = %q, want path", resolved[0].Source)
	}
	if !strings.HasPrefix(resolved[0].ContentHash, "sha256:") {
		t.Errorf("content_hash should start with sha256:, got %s", resolved[0].ContentHash)
	}
}

func TestResolveDependencies_TransitiveDeps(t *testing.T) {
	root := t.TempDir()

	// C has no deps
	cDir := filepath.Join(root, "c")
	writeManifest(t, cDir, `
[package]
name = "test/c"
version = "0.1.0"
edition = "1"
`)
	os.WriteFile(filepath.Join(cDir, "c.ail"), []byte("module test/c/core\n"), 0644)

	// B depends on C
	bDir := filepath.Join(root, "b")
	writeManifest(t, bDir, `
[package]
name = "test/b"
version = "0.1.0"
edition = "1"

[dependencies]
"test/c" = { path = "../c" }
`)
	os.WriteFile(filepath.Join(bDir, "b.ail"), []byte("module test/b/core\n"), 0644)

	// A depends on B
	aDir := filepath.Join(root, "a")
	writeManifest(t, aDir, `
[package]
name = "test/a"
version = "0.1.0"
edition = "1"

[dependencies]
"test/b" = { path = "../b" }
`)

	manifest, err := LoadManifest(aDir)
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}

	resolved, err := ResolveDependencies(manifest, aDir)
	if err != nil {
		t.Fatalf("ResolveDependencies: %v", err)
	}

	// Should resolve both B and C (C first as transitive dep of B)
	if len(resolved) != 2 {
		t.Fatalf("expected 2 resolved, got %d", len(resolved))
	}

	// C should be resolved first (transitive, deeper in the graph)
	if resolved[0].Name != "test/c" {
		t.Errorf("first resolved = %q, want test/c", resolved[0].Name)
	}
	if resolved[1].Name != "test/b" {
		t.Errorf("second resolved = %q, want test/b", resolved[1].Name)
	}
}

func TestResolveDependencies_CycleDetection(t *testing.T) {
	root := t.TempDir()

	// A depends on B, B depends on A
	aDir := filepath.Join(root, "a")
	writeManifest(t, aDir, `
[package]
name = "test/a"
version = "0.1.0"
edition = "1"

[dependencies]
"test/b" = { path = "../b" }
`)

	bDir := filepath.Join(root, "b")
	writeManifest(t, bDir, `
[package]
name = "test/b"
version = "0.1.0"
edition = "1"

[dependencies]
"test/a" = { path = "../a" }
`)

	manifest, err := LoadManifest(aDir)
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}

	_, err = ResolveDependencies(manifest, aDir)
	if err == nil {
		t.Fatal("expected cycle detection error")
	}
	if !strings.Contains(err.Error(), "circular dependency") {
		t.Errorf("error should mention circular dependency, got: %v", err)
	}
}

func TestResolveDependencies_MissingDir(t *testing.T) {
	dir := t.TempDir()
	writeManifest(t, dir, `
[package]
name = "test/app"
version = "0.1.0"
edition = "1"

[dependencies]
"test/missing" = { path = "../nonexistent" }
`)

	manifest, err := LoadManifest(dir)
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}

	_, err = ResolveDependencies(manifest, dir)
	if err == nil {
		t.Fatal("expected error for missing dependency directory")
	}
}

func TestResolveDependencies_NameMismatch(t *testing.T) {
	root := t.TempDir()

	libDir := filepath.Join(root, "lib")
	writeManifest(t, libDir, `
[package]
name = "actual/name"
version = "0.1.0"
edition = "1"
`)

	appDir := filepath.Join(root, "app")
	writeManifest(t, appDir, `
[package]
name = "test/app"
version = "0.1.0"
edition = "1"

[dependencies]
"wrong/name" = { path = "../lib" }
`)

	manifest, err := LoadManifest(appDir)
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}

	_, err = ResolveDependencies(manifest, appDir)
	if err == nil {
		t.Fatal("expected error for name mismatch")
	}
	if !strings.Contains(err.Error(), "name mismatch") {
		t.Errorf("error should mention name mismatch, got: %v", err)
	}
}

func TestResolveDependencies_TransitiveRegistryWithPathDeps(t *testing.T) {
	// Simulates: app → B@registry (B has path dep on C) → C@registry
	// B was published before path-to-registry rewrite, so its manifest
	// still contains { path = "../c" }. The resolver should detect this
	// and convert it to a registry lookup.

	root := t.TempDir()

	// Create package B source (with path dep on C — as it would appear in a pre-rewrite tarball)
	bSrcDir := filepath.Join(root, "b_src")
	writeManifest(t, bSrcDir, `
[package]
name = "test/b"
version = "0.1.0"
edition = "1"

[dependencies]
"test/c" = { path = "../c" }
`)
	os.WriteFile(filepath.Join(bSrcDir, "b.ail"), []byte("module test/b/core\n"), 0644)
	bTarball, err := CreateTarball(bSrcDir)
	if err != nil {
		t.Fatalf("CreateTarball(b): %v", err)
	}

	// Create package C source (no deps)
	cSrcDir := filepath.Join(root, "c_src")
	writeManifest(t, cSrcDir, `
[package]
name = "test/c"
version = "0.2.0"
edition = "1"
`)
	os.WriteFile(filepath.Join(cSrcDir, "c.ail"), []byte("module test/c/core\n"), 0644)
	cTarball, err := CreateTarball(cSrcDir)
	if err != nil {
		t.Fatalf("CreateTarball(c): %v", err)
	}

	// Mock registry server: serves index.json and package tarballs
	index := RegistryIndex{
		Schema: "ailang.registry/v1",
		Packages: []IndexEntry{
			{Name: "test/b", Latest: "0.1.0", Versions: []string{"0.1.0"}},
			{Name: "test/c", Latest: "0.2.0", Versions: []string{"0.2.0"}},
		},
	}
	indexJSON, _ := json.Marshal(index)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/index.json":
			w.Header().Set("Content-Type", "application/json")
			w.Write(indexJSON)
		case r.URL.Path == "/packages/test/b/0.1.0/package.tar.gz":
			w.Header().Set("Content-Type", "application/gzip")
			w.Write(bTarball)
		case r.URL.Path == "/packages/test/c/0.2.0/package.tar.gz":
			w.Header().Set("Content-Type", "application/gzip")
			w.Write(cTarball)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	// Point AILANG_REGISTRY at mock server
	t.Setenv("AILANG_REGISTRY", server.URL)

	// Override HOME so cache goes to temp dir
	fakeHome := filepath.Join(root, "fakehome")
	os.MkdirAll(fakeHome, 0755)
	t.Setenv("HOME", fakeHome)

	// Create app that depends on B@registry
	appDir := filepath.Join(root, "app")
	writeManifest(t, appDir, `
[package]
name = "test/app"
version = "1.0.0"
edition = "1"

[dependencies]
"test/b" = "0.1.0"
`)

	manifest, err := LoadManifest(appDir)
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}

	resolved, err := ResolveDependencies(manifest, appDir)
	if err != nil {
		t.Fatalf("ResolveDependencies: %v", err)
	}

	// Should resolve both B and C
	if len(resolved) != 2 {
		t.Fatalf("expected 2 resolved packages, got %d: %+v", len(resolved), resolved)
	}

	// C should be first (transitive dep, resolved before B)
	names := []string{resolved[0].Name, resolved[1].Name}
	if names[0] != "test/c" || names[1] != "test/b" {
		t.Errorf("expected [test/c, test/b], got %v", names)
	}

	// Both should be registry source
	for _, r := range resolved {
		if r.Source != "registry" {
			t.Errorf("package %s: source = %q, want registry", r.Name, r.Source)
		}
	}
}

func TestBuildDependencyTree(t *testing.T) {
	root := t.TempDir()

	libDir := filepath.Join(root, "lib")
	writeManifest(t, libDir, `
[package]
name = "test/lib"
version = "0.2.0"
edition = "1"
`)

	appDir := filepath.Join(root, "app")
	writeManifest(t, appDir, `
[package]
name = "test/app"
version = "0.1.0"
edition = "1"

[dependencies]
"test/lib" = { path = "../lib" }
`)

	manifest, err := LoadManifest(appDir)
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}

	tree, err := BuildDependencyTree(manifest, appDir)
	if err != nil {
		t.Fatalf("BuildDependencyTree: %v", err)
	}

	if !strings.Contains(tree, "test/app@0.1.0") {
		t.Errorf("tree should contain root package, got:\n%s", tree)
	}
	if !strings.Contains(tree, "test/lib") {
		t.Errorf("tree should contain dependency, got:\n%s", tree)
	}
}
