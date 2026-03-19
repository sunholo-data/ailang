package pkg

import (
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
