package pkg

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverPackageSources_MultiModule(t *testing.T) {
	dir := filepath.Join("testdata", "multi_module")

	sources, err := DiscoverPackageSources(dir)
	if err != nil {
		t.Fatalf("DiscoverPackageSources failed: %v", err)
	}

	// Check manifest loaded
	if sources.Manifest.Package.Name != "test/multi" {
		t.Errorf("expected package name test/multi, got %s", sources.Manifest.Package.Name)
	}

	// Check source files discovered
	if len(sources.SourceFiles) != 2 {
		t.Errorf("expected 2 source files, got %d: %v", len(sources.SourceFiles), sources.SourceFiles)
	}

	// Check specific modules mapped
	if _, ok := sources.SourceFiles["test/multi/core"]; !ok {
		t.Error("expected test/multi/core in source files")
	}
	if _, ok := sources.SourceFiles["test/multi/types"]; !ok {
		t.Error("expected test/multi/types in source files")
	}

	// Check test files discovered
	if len(sources.TestFiles) != 1 {
		t.Errorf("expected 1 test file, got %d", len(sources.TestFiles))
	}
	if len(sources.TestFiles) > 0 {
		base := filepath.Base(sources.TestFiles[0])
		if base != "core_test.ail" {
			t.Errorf("expected core_test.ail, got %s", base)
		}
	}

	// Check no missing modules
	if len(sources.MissingModules) != 0 {
		t.Errorf("expected no missing modules, got %v", sources.MissingModules)
	}
}

func TestDiscoverPackageSources_MissingModule(t *testing.T) {
	// Create a temporary package with a module in exports that has no source file
	dir := t.TempDir()

	// Write minimal ailang.toml with a module that doesn't exist
	tomlContent := `[package]
name = "test/missing"
version = "0.1.0"
edition = "1"

[exports]
modules = ["test/missing/core", "test/missing/nonexistent"]

[effects]
max = []

[stability]
level = "experimental"
`
	writeTestFile(t, dir, "ailang.toml", tomlContent)
	writeTestFile(t, dir, "src/core.ail", "module test/missing/core\n")

	sources, err := DiscoverPackageSources(dir)
	if err != nil {
		t.Fatalf("DiscoverPackageSources failed: %v", err)
	}

	if len(sources.MissingModules) != 1 {
		t.Errorf("expected 1 missing module, got %d: %v", len(sources.MissingModules), sources.MissingModules)
	}
	if len(sources.MissingModules) > 0 && sources.MissingModules[0] != "test/missing/nonexistent" {
		t.Errorf("expected test/missing/nonexistent, got %s", sources.MissingModules[0])
	}
}

func TestDiscoverPackageSources_NoManifest(t *testing.T) {
	dir := t.TempDir()

	_, err := DiscoverPackageSources(dir)
	if err == nil {
		t.Error("expected error for directory without ailang.toml")
	}
}

func TestDiscoverPackageSources_AllSourcePaths(t *testing.T) {
	dir := filepath.Join("testdata", "multi_module")

	sources, err := DiscoverPackageSources(dir)
	if err != nil {
		t.Fatalf("DiscoverPackageSources failed: %v", err)
	}

	paths := sources.AllSourcePaths()
	if len(paths) != 2 {
		t.Errorf("expected 2 source paths, got %d", len(paths))
	}
}

func writeTestFile(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("failed to create dir for %s: %v", name, err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write %s: %v", name, err)
	}
}
