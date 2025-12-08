package loader

import (
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
