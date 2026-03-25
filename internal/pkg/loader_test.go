package pkg

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupTestPackage(t *testing.T, root, name string, exports []string, files map[string]string) string {
	t.Helper()
	dir := filepath.Join(root, strings.ReplaceAll(name, "/", "_"))
	os.MkdirAll(filepath.Join(dir, "src"), 0755)

	// Write manifest
	exportsStr := ""
	if len(exports) > 0 {
		quoted := make([]string, len(exports))
		for i, e := range exports {
			quoted[i] = `"` + e + `"`
		}
		exportsStr = strings.Join(quoted, ", ")
	}

	manifest := `[package]
name = "` + name + `"
version = "0.1.0"
edition = "1"

[exports]
modules = [` + exportsStr + `]
`
	os.WriteFile(filepath.Join(dir, ManifestFile), []byte(manifest), 0644)

	// Write source files
	for path, content := range files {
		fullPath := filepath.Join(dir, path)
		os.MkdirAll(filepath.Dir(fullPath), 0755)
		os.WriteFile(fullPath, []byte(content), 0644)
	}

	return dir
}

func TestPackageLoader_ResolveExportedModule(t *testing.T) {
	root := t.TempDir()
	libDir := setupTestPackage(t, root, "sunholo/json", []string{"sunholo/json/parser"}, map[string]string{
		"src/parser.ail": "module sunholo/json/parser\n",
	})

	lf := &LockFile{
		Schema:  LockFileSchema,
		Version: "1.0.0",
		Packages: []LockedPackage{
			{Name: "sunholo/json", Version: "0.3.1", ContentHash: "sha256:abc", Source: "path", Path: libDir},
		},
	}

	loader := NewPackageLoader(lf, root)

	path, err := loader.ResolveImport("sunholo/json/parser")
	if err != nil {
		t.Fatalf("ResolveImport failed: %v", err)
	}
	wantSuffix := filepath.Join("src", "parser.ail")
	if !strings.HasSuffix(path, wantSuffix) {
		t.Errorf("resolved path = %q, want ...%s", path, wantSuffix)
	}
}

func TestPackageLoader_RejectNonExportedModule(t *testing.T) {
	root := t.TempDir()
	libDir := setupTestPackage(t, root, "sunholo/json",
		[]string{"sunholo/json/parser"}, // only parser is exported
		map[string]string{
			"src/parser.ail":   "module sunholo/json/parser\n",
			"src/internal.ail": "module sunholo/json/internal\n",
		},
	)

	lf := &LockFile{
		Schema:  LockFileSchema,
		Version: "1.0.0",
		Packages: []LockedPackage{
			{Name: "sunholo/json", Version: "0.3.1", ContentHash: "sha256:abc", Source: "path", Path: libDir},
		},
	}

	loader := NewPackageLoader(lf, root)

	_, err := loader.ResolveImport("sunholo/json/internal")
	if err == nil {
		t.Fatal("expected error for non-exported module")
	}
	if !strings.Contains(err.Error(), "not exported") {
		t.Errorf("error should mention 'not exported', got: %v", err)
	}
	// Error should suggest available exports
	if !strings.Contains(err.Error(), "sunholo/json/parser") {
		t.Errorf("error should list available exports, got: %v", err)
	}
}

func TestPackageLoader_UndeclaredDependency(t *testing.T) {
	lf := &LockFile{
		Schema:   LockFileSchema,
		Version:  "1.0.0",
		Packages: []LockedPackage{},
	}

	loader := NewPackageLoader(lf, "/tmp")

	_, err := loader.ResolveImport("unknown/pkg/module")
	if err == nil {
		t.Fatal("expected error for undeclared dependency")
	}
	if !strings.Contains(err.Error(), "not found in ailang.lock") {
		t.Errorf("error should mention ailang.lock, got: %v", err)
	}
}

func TestPackageLoader_NoExportsList_AllAccessible(t *testing.T) {
	root := t.TempDir()
	// Package with empty exports = everything accessible
	libDir := setupTestPackage(t, root, "sunholo/json",
		[]string{}, // no exports declared
		map[string]string{
			"src/parser.ail": "module sunholo/json/parser\n",
		},
	)

	lf := &LockFile{
		Schema:  LockFileSchema,
		Version: "1.0.0",
		Packages: []LockedPackage{
			{Name: "sunholo/json", Version: "0.1.0", ContentHash: "sha256:abc", Source: "path", Path: libDir},
		},
	}

	loader := NewPackageLoader(lf, root)

	// Should succeed — no exports means all modules accessible
	_, err := loader.ResolveImport("sunholo/json/parser")
	if err != nil {
		t.Fatalf("should succeed with no exports list: %v", err)
	}
}

func TestPackageLoader_RootFileResolution(t *testing.T) {
	root := t.TempDir()
	// Package with .ail files at root (no src/ directory)
	libDir := setupTestPackage(t, root, "sunholo/json", []string{}, map[string]string{
		"parser.ail": "module sunholo/json/parser\n",
	})

	lf := &LockFile{
		Schema:  LockFileSchema,
		Version: "1.0.0",
		Packages: []LockedPackage{
			{Name: "sunholo/json", Version: "0.1.0", ContentHash: "sha256:abc", Source: "path", Path: libDir},
		},
	}

	loader := NewPackageLoader(lf, root)

	path, err := loader.ResolveImport("sunholo/json/parser")
	if err != nil {
		t.Fatalf("should resolve root-level .ail files: %v", err)
	}
	if !strings.HasSuffix(path, "parser.ail") {
		t.Errorf("resolved path = %q, want .../parser.ail", path)
	}
}

func TestPackageLoader_HasPackage(t *testing.T) {
	lf := &LockFile{
		Schema:  LockFileSchema,
		Version: "1.0.0",
		Packages: []LockedPackage{
			{Name: "sunholo/json", ContentHash: "sha256:abc", Source: "path"},
		},
	}

	loader := NewPackageLoader(lf, "/tmp")

	if !loader.HasPackage("sunholo/json") {
		t.Error("should have sunholo/json")
	}
	if loader.HasPackage("sunholo/xml") {
		t.Error("should not have sunholo/xml")
	}
}

func TestCheckEffectCeiling_WithinBounds(t *testing.T) {
	err := CheckEffectCeiling("test/pkg", []string{"IO"}, []string{"IO", "Net"})
	if err != nil {
		t.Errorf("should pass when effects within ceiling: %v", err)
	}
}

func TestCheckEffectCeiling_Violation(t *testing.T) {
	err := CheckEffectCeiling("test/pkg", []string{"IO", "FS"}, []string{"IO"})
	if err == nil {
		t.Fatal("should fail when effects exceed ceiling")
	}
	if !strings.Contains(err.Error(), "FS") {
		t.Errorf("error should mention violating effect FS, got: %v", err)
	}
	if !strings.Contains(err.Error(), "ailang.toml") {
		t.Errorf("error should suggest ailang.toml fix, got: %v", err)
	}
}

func TestCheckEffectCeiling_PurePackage(t *testing.T) {
	err := CheckEffectCeiling("test/pkg", []string{"IO"}, []string{})
	if err == nil {
		t.Fatal("should fail when pure package uses effects")
	}
}

func TestCheckEffectCeiling_NoCeiling(t *testing.T) {
	err := CheckEffectCeiling("test/pkg", []string{"IO", "Net", "FS"}, nil)
	if err != nil {
		t.Errorf("should pass when no ceiling declared: %v", err)
	}
}

func TestCheckEffectCeiling_EmptyEffects(t *testing.T) {
	err := CheckEffectCeiling("test/pkg", []string{}, []string{})
	if err != nil {
		t.Errorf("should pass when function has no effects: %v", err)
	}
}

func TestCheckEffectCeiling_GhostEffectBypass(t *testing.T) {
	// Debug is a ghost effect — it should pass ceiling check even when not in max list
	err := CheckEffectCeiling("test/pkg", []string{"IO", "Debug"}, []string{"IO"})
	if err != nil {
		t.Errorf("ghost effect Debug should bypass ceiling check: %v", err)
	}

	// Debug-only function with no max effects should also pass
	err = CheckEffectCeiling("test/pkg", []string{"Debug"}, []string{})
	if err != nil {
		t.Errorf("ghost effect Debug should bypass empty ceiling: %v", err)
	}

	// Non-ghost effects should still be checked
	err = CheckEffectCeiling("test/pkg", []string{"IO", "Debug"}, []string{"Debug"})
	if err == nil {
		t.Error("non-ghost effect IO should still be caught by ceiling check")
	}
}

func TestPackageDir_RegistryRuntimeResolution(t *testing.T) {
	// Setup: create a registry package in the cache (simulates ailang install)
	cachePath, err := CachedPackagePath("sunholo/testpkg", "0.1.0")
	if err != nil {
		t.Fatalf("CachedPackagePath: %v", err)
	}
	os.MkdirAll(cachePath, 0755)
	defer os.RemoveAll(cachePath)

	// Write a minimal manifest so loader can work
	manifest := `[package]
name = "sunholo/testpkg"
version = "0.1.0"
edition = "1"

[exports]
modules = ["sunholo/testpkg/hello"]
`
	os.WriteFile(filepath.Join(cachePath, ManifestFile), []byte(manifest), 0644)
	os.WriteFile(filepath.Join(cachePath, "hello.ail"), []byte("module sunholo/testpkg/hello\n"), 0644)

	// Lock entry has NO Path — must resolve at runtime
	lf := &LockFile{
		Schema:  LockFileSchema,
		Version: "1.0.0",
		Packages: []LockedPackage{
			{Name: "sunholo/testpkg", Version: "0.1.0", ContentHash: "sha256:abc", Source: "registry"},
			// Note: Path is empty
		},
	}

	loader := NewPackageLoader(lf, t.TempDir())

	// Should resolve via CachedPackagePath at runtime
	resolved, err := loader.ResolveImport("sunholo/testpkg/hello")
	if err != nil {
		t.Fatalf("ResolveImport should work without stored Path: %v", err)
	}
	if !strings.HasSuffix(resolved, "hello.ail") {
		t.Errorf("expected .../hello.ail, got %s", resolved)
	}
}

func TestPackageDir_RegistryBackwardCompat(t *testing.T) {
	// Old lock file format has stored Path — should still work
	root := t.TempDir()
	libDir := setupTestPackage(t, root, "sunholo/oldpkg", []string{"sunholo/oldpkg/util"}, map[string]string{
		"util.ail": "module sunholo/oldpkg/util\n",
	})

	lf := &LockFile{
		Schema:  LockFileSchema,
		Version: "1.0.0",
		Packages: []LockedPackage{
			{Name: "sunholo/oldpkg", Version: "0.1.0", ContentHash: "sha256:abc", Source: "registry", Path: libDir},
		},
	}

	loader := NewPackageLoader(lf, root)

	resolved, err := loader.ResolveImport("sunholo/oldpkg/util")
	if err != nil {
		t.Fatalf("backward compat failed — old lock file with Path should work: %v", err)
	}
	if !strings.HasSuffix(resolved, "util.ail") {
		t.Errorf("expected .../util.ail, got %s", resolved)
	}
}
