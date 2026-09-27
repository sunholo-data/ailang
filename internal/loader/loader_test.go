package loader

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestIsTempPath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		// Unix temp paths
		{
			name:     "Unix /tmp/ prefix",
			path:     "/tmp/foo.ail",
			expected: runtime.GOOS != "windows",
		},
		{
			name:     "Unix /tmp/ nested",
			path:     "/tmp/test/nested/bar.ail",
			expected: runtime.GOOS != "windows",
		},
		{
			name:     "macOS /var/folders/ prefix",
			path:     "/var/folders/xyz/abc123/T/foo.ail",
			expected: runtime.GOOS != "windows",
		},

		// Canonical paths (after CanonicalModuleID strips leading /)
		{
			name:     "Canonical tmp/ prefix",
			path:     "tmp/test_relax",
			expected: true, // Cross-platform: canonical path detection
		},
		{
			name:     "Canonical tmp/ nested",
			path:     "tmp/nested/deep/foo",
			expected: true,
		},
		{
			name:     "Canonical var/folders/ prefix",
			path:     "var/folders/xyz/abc123/foo",
			expected: true,
		},

		// Non-temp paths
		{
			name:     "Regular project path",
			path:     "./src/foo.ail",
			expected: false,
		},
		{
			name:     "Absolute project path",
			path:     "/home/user/project/src/foo.ail",
			expected: false,
		},
		{
			name:     "Examples directory",
			path:     "examples/hello.ail",
			expected: false,
		},
		{
			name:     "Path containing tmp but not temp dir",
			path:     "/home/user/tmp_backup/foo.ail",
			expected: false,
		},
		{
			name:     "Path starting with tmp_prefix (not tmp/)",
			path:     "tmp_backup/foo.ail",
			expected: false,
		},

		// Edge cases
		{
			name:     "Empty path",
			path:     "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsTempPath(tt.path)
			if result != tt.expected {
				t.Errorf("IsTempPath(%q) = %v, expected %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestIsTempPath_OsTempDir(t *testing.T) {
	// Test with actual os.TempDir() - should always match
	tempDir := os.TempDir()
	testFile := filepath.Join(tempDir, "test_ailang.ail")

	if !IsTempPath(testFile) {
		t.Errorf("IsTempPath(%q) = false, expected true (should match os.TempDir())", testFile)
	}
}

func TestIsTempPath_NestedTempDir(t *testing.T) {
	// Test with nested directory inside os.TempDir()
	tempDir := os.TempDir()
	nestedPath := filepath.Join(tempDir, "nested", "deep", "test.ail")

	if !IsTempPath(nestedPath) {
		t.Errorf("IsTempPath(%q) = false, expected true (nested in temp dir)", nestedPath)
	}
}

func TestCacheSource_ExactSnapshot(t *testing.T) {
	t.Run("disk bytes", func(t *testing.T) {
		assertDiskSourceExactSnapshot(t)
	})
}

// assertDiskSourceExactSnapshot asserts the loader retains the exact bytes it
// read from disk as the immutable source snapshot.
func assertDiskSourceExactSnapshot(t *testing.T) {
	root := t.TempDir()
	content := []byte("module exact\n\nexport pure func answer() -> int = 42\n")
	if err := os.WriteFile(filepath.Join(root, "exact.ail"), content, 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	loaded, err := NewModuleLoader(root).Load("exact")
	if err != nil {
		t.Fatalf("load disk source: %v", err)
	}
	if loaded.SourceContent == nil {
		t.Fatal("disk source snapshot is unavailable")
	}
	if got := *loaded.SourceContent; got != string(content) {
		t.Fatalf("disk source snapshot = %q, want exact bytes %q", got, content)
	}
}

func TestCanonicalModuleID(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Basic cases
		{"foo/bar", "foo/bar"},
		{"./foo/bar", "foo/bar"},
		{"foo/bar.ail", "foo/bar"},

		// Absolute paths (should strip leading /)
		{"/foo/bar", "foo/bar"},
		{"/foo/bar.ail", "foo/bar"},

		// Windows paths
		{"foo\\bar", "foo/bar"},
		{".\\foo\\bar", "foo/bar"},

		// Edge cases
		{".", "."},
		{"", "."},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := CanonicalModuleID(tt.input)
			if result != tt.expected {
				t.Errorf("CanonicalModuleID(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

// mockPkgResolver implements PackageResolver for testing prefix resolution.
type mockPkgResolver struct {
	files map[string]string // canonical import path → absolute file path
}

func (m *mockPkgResolver) ResolveImport(importPath string) (string, error) {
	if path, ok := m.files[importPath]; ok {
		return path, nil
	}
	return "", fmt.Errorf("not found: %s", importPath)
}

func TestLoad_ModulePrefixBareImport(t *testing.T) {
	// Simulate: package sunholo/ailang_parse with module_prefix="docparse"
	// A file inside the package has "import docparse/types/document"
	// The loader should resolve this via the prefix map.
	tmpDir := t.TempDir()

	// Create the target file that the bare import should resolve to
	docFile := filepath.Join(tmpDir, "docparse", "types", "document.ail")
	os.MkdirAll(filepath.Dir(docFile), 0755)
	os.WriteFile(docFile, []byte("module docparse/types/document\nexport func newDoc() = \"doc\"\n"), 0644)

	ml := NewModuleLoader(t.TempDir()) // consumer's basePath (different dir)

	resolver := &mockPkgResolver{
		files: map[string]string{
			"sunholo/ailang_parse/types/document": docFile,
		},
	}
	ml.SetPackageResolver(resolver)
	ml.SetModulePrefixMap(map[string]string{
		"sunholo/ailang_parse": "docparse",
	})

	// Load bare import "docparse/types/document" — should resolve via prefix
	loaded, err := ml.Load("docparse/types/document")
	if err != nil {
		t.Fatalf("expected prefix resolution to succeed, got: %v", err)
	}
	if loaded == nil {
		t.Fatal("expected non-nil LoadedModule")
	}
	if _, ok := loaded.Exports["newDoc"]; !ok {
		t.Error("expected 'newDoc' in exports")
	}
}

func TestLoad_ModulePrefixFallsBackToProjectRelative(t *testing.T) {
	// When prefix map is set but the import doesn't match any prefix,
	// it should fall through to project-relative resolution.
	tmpDir := t.TempDir()

	// Create a project-relative file
	localFile := filepath.Join(tmpDir, "utils", "helpers.ail")
	os.MkdirAll(filepath.Dir(localFile), 0755)
	os.WriteFile(localFile, []byte("module utils/helpers\nexport func help() = \"ok\"\n"), 0644)

	ml := NewModuleLoader(tmpDir)
	ml.SetPackageResolver(&mockPkgResolver{files: map[string]string{}})
	ml.SetModulePrefixMap(map[string]string{
		"sunholo/ailang_parse": "docparse",
	})

	// "utils/helpers" doesn't match "docparse" prefix — should fall through
	loaded, err := ml.Load("utils/helpers")
	if err != nil {
		t.Fatalf("expected project-relative fallback, got: %v", err)
	}
	if loaded == nil {
		t.Fatal("expected non-nil LoadedModule")
	}
}

func TestNormalizeRelativeImport(t *testing.T) {
	tests := []struct {
		name       string
		modulePath string
		relative   string
		expected   string
	}{
		{
			name:       "sibling module",
			modulePath: "sunholo/billing_entitlements/entitlement",
			relative:   "plan",
			expected:   "sunholo/billing_entitlements/plan",
		},
		{
			name:       "child directory module",
			modulePath: "sunholo/docparse/services/api_server",
			relative:   "utils/validation",
			expected:   "sunholo/docparse/services/utils/validation",
		},
		{
			name:       "sibling in flat package",
			modulePath: "sunholo/firestore/client",
			relative:   "fields",
			expected:   "sunholo/firestore/fields",
		},
		{
			name:       "two-segment module path",
			modulePath: "sunholo/firestore",
			relative:   "client",
			expected:   "sunholo/client",
		},
		{
			name:       "single-segment module path",
			modulePath: "main",
			relative:   "helpers",
			expected:   "helpers",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeRelativeImport(tt.modulePath, tt.relative)
			if result != tt.expected {
				t.Errorf("NormalizeRelativeImport(%q, %q) = %q, expected %q",
					tt.modulePath, tt.relative, result, tt.expected)
			}
		})
	}
}
