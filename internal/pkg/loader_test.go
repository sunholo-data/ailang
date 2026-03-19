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
	if !strings.HasSuffix(path, "src/parser.ail") {
		t.Errorf("resolved path = %q, want .../src/parser.ail", path)
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
