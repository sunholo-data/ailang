package module

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestNewResolver(t *testing.T) {
	r := NewResolver()
	
	if r.projectRoot == "" {
		t.Error("projectRoot should not be empty")
	}
	
	if r.stdlibPath == "" {
		t.Error("stdlibPath should not be empty")
	}
	
	if r.searchPaths == nil {
		t.Error("searchPaths should not be nil")
	}
}

func TestNormalizePath(t *testing.T) {
	r := NewResolver()
	
	// Test home directory expansion
	home, _ := os.UserHomeDir()
	path, err := r.NormalizePath("~/test.ail")
	if err != nil {
		t.Errorf("NormalizePath failed: %v", err)
	}
	if !strings.HasPrefix(path, home) {
		t.Errorf("Path should start with home directory: %s", path)
	}
	
	// Test relative path
	path, err = r.NormalizePath("./test.ail")
	if err != nil {
		t.Errorf("NormalizePath failed: %v", err)
	}
	if !filepath.IsAbs(path) {
		t.Errorf("Path should be absolute: %s", path)
	}
	
	// Test .. resolution
	path, err = r.NormalizePath("../test.ail")
	if err != nil {
		t.Errorf("NormalizePath failed: %v", err)
	}
	if strings.Contains(path, "..") {
		t.Errorf("Path should not contain ..: %s", path)
	}
}

func TestResolveImportTypes(t *testing.T) {
	r := NewResolver()
	currentFile := "/project/src/main.ail"
	
	tests := []struct {
		name        string
		importPath  string
		currentFile string
		shouldError bool
		pathType    string
	}{
		{
			name:        "relative import",
			importPath:  "./utils",
			currentFile: currentFile,
			shouldError: true, // Will fail unless file exists
			pathType:    "relative",
		},
		{
			name:        "parent relative import",
			importPath:  "../lib/helper",
			currentFile: currentFile,
			shouldError: true, // Will fail unless file exists
			pathType:    "relative",
		},
		{
			name:        "stdlib import",
			importPath:  "std/list",
			currentFile: "",
			shouldError: true, // Will fail unless stdlib exists
			pathType:    "stdlib",
		},
		{
			name:        "project import with slash",
			importPath:  "data/structures",
			currentFile: "",
			shouldError: true, // Will fail unless module exists
			pathType:    "project",
		},
		{
			name:        "local import",
			importPath:  "utils",
			currentFile: currentFile,
			shouldError: true, // Will fail unless file exists
			pathType:    "local",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := r.ResolveImport(tt.importPath, tt.currentFile)
			// We expect errors since files don't exist, but we're testing the logic
			if err == nil && tt.shouldError {
				t.Errorf("Expected error for %s import", tt.pathType)
			}
		})
	}
}

func TestGetModuleIdentity(t *testing.T) {
	r := NewResolver()
	
	// Test basic file
	identity, err := r.GetModuleIdentity("/project/utils.ail")
	if err != nil {
		t.Errorf("GetModuleIdentity failed: %v", err)
	}
	if identity != "utils" {
		t.Errorf("Identity = %s, want utils", identity)
	}
	
	// Test nested path
	identity, err = r.GetModuleIdentity("/project/data/structures.ail")
	if err != nil {
		t.Errorf("GetModuleIdentity failed: %v", err)
	}
	// Should be relative to project root or just base name
	if !strings.HasSuffix(identity, "structures") {
		t.Errorf("Identity should end with 'structures': %s", identity)
	}
}

func TestValidateModuleName(t *testing.T) {
	r := NewResolver()
	
	tests := []struct {
		name         string
		declaredName string
		filePath     string
		shouldError  bool
	}{
		{
			name:         "matching base name",
			declaredName: "utils",
			filePath:     "/project/utils.ail",
			shouldError:  false,
		},
		{
			name:         "mismatched name",
			declaredName: "wrong",
			filePath:     "/project/utils.ail",
			shouldError:  true,
		},
		{
			name:         "stdlib flexibility",
			declaredName: "list",
			filePath:     "/stdlib/std/list.ail",
			shouldError:  false, // Base name matches
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := r.ValidateModuleName(tt.declaredName, tt.filePath)
			if (err != nil) != tt.shouldError {
				t.Errorf("ValidateModuleName(%s, %s) error = %v, shouldError = %v",
					tt.declaredName, tt.filePath, err, tt.shouldError)
			}
		})
	}
}

func TestIsFileSystemCaseSensitive(t *testing.T) {
	result := isFileSystemCaseSensitive()
	
	switch runtime.GOOS {
	case "windows", "darwin":
		if result {
			t.Errorf("Expected case-insensitive on %s", runtime.GOOS)
		}
	case "linux":
		if !result {
			t.Errorf("Expected case-sensitive on %s", runtime.GOOS)
		}
	}
}

func TestGetResolutionOrder(t *testing.T) {
	r := NewResolver()
	
	// Test relative import
	order := r.GetResolutionOrder("./utils", "/project/src/main.ail")
	if len(order) == 0 {
		t.Error("Resolution order should not be empty")
	}
	// Should try with and without .ail extension
	hasAilVariant := false
	for _, path := range order {
		if strings.HasSuffix(path, ".ail") {
			hasAilVariant = true
			break
		}
	}
	if !hasAilVariant {
		t.Error("Resolution order should include .ail variants")
	}
	
	// Test stdlib import
	order = r.GetResolutionOrder("std/list", "")
	if len(order) == 0 {
		t.Error("Resolution order should not be empty for stdlib")
	}
	foundStdlib := false
	for _, path := range order {
		if strings.Contains(path, "list") {
			foundStdlib = true
			break
		}
	}
	if !foundStdlib {
		t.Error("Resolution order should include stdlib path")
	}
	
	// Test project import
	order = r.GetResolutionOrder("data/structures", "")
	if len(order) == 0 {
		t.Error("Resolution order should not be empty for project import")
	}
}

func TestFindProjectRoot(t *testing.T) {
	// This test depends on the actual project structure
	root := findProjectRoot()
	
	// Should find a valid directory
	if root == "" {
		t.Error("Project root should not be empty")
	}
	
	// Should be an absolute path
	if !filepath.IsAbs(root) {
		t.Errorf("Project root should be absolute: %s", root)
	}
	
	// Should exist
	if _, err := os.Stat(root); err != nil {
		t.Errorf("Project root should exist: %s", root)
	}
}

func TestFindStdlibPath(t *testing.T) {
	path := findStdlibPath()
	
	// Should return a path (even if it doesn't exist)
	if path == "" {
		t.Error("Stdlib path should not be empty")
	}
	
	// Test environment variable override
	testPath := "/test/stdlib"
	os.Setenv("AILANG_STDLIB", testPath)
	defer os.Unsetenv("AILANG_STDLIB")
	
	path = findStdlibPath()
	if path != testPath {
		t.Errorf("Stdlib path = %s, want %s", path, testPath)
	}
}

func TestGetSearchPaths(t *testing.T) {
	// Test with environment variable
	testPaths := "/path1" + string(os.PathListSeparator) + "/path2"
	os.Setenv("AILANG_PATH", testPaths)
	defer os.Unsetenv("AILANG_PATH")
	
	paths := getSearchPaths()
	
	// Should include the paths from environment
	found1 := false
	found2 := false
	for _, p := range paths {
		if p == "/path1" {
			found1 = true
		}
		if p == "/path2" {
			found2 = true
		}
	}
	
	if !found1 || !found2 {
		t.Errorf("Search paths should include environment paths: %v", paths)
	}
	
	// Should include project root
	projectRoot := findProjectRoot()
	foundRoot := false
	for _, p := range paths {
		if p == projectRoot {
			foundRoot = true
			break
		}
	}
	if !foundRoot {
		t.Error("Search paths should include project root")
	}
}

func TestResolveRelativeImport(t *testing.T) {
	r := NewResolver()
	
	// Create a temporary directory and file
	tmpDir, err := os.MkdirTemp("", "resolver_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	
	// Create test files
	mainFile := filepath.Join(tmpDir, "main.ail")
	utilsFile := filepath.Join(tmpDir, "utils.ail")
	
	if err := os.WriteFile(mainFile, []byte("module main"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(utilsFile, []byte("module utils"), 0644); err != nil {
		t.Fatal(err)
	}
	
	// Test resolving ./utils from main.ail
	resolved, err := r.resolveRelativeImport("./utils", mainFile)
	if err != nil {
		t.Errorf("Failed to resolve relative import: %v", err)
	}
	
	// Should resolve to utils.ail
	if !strings.HasSuffix(resolved, "utils.ail") {
		t.Errorf("Resolved path should end with utils.ail: %s", resolved)
	}
	
	// Should be absolute
	if !filepath.IsAbs(resolved) {
		t.Errorf("Resolved path should be absolute: %s", resolved)
	}
	
	// Test error case - no current file
	_, err = r.resolveRelativeImport("./utils", "")
	if err == nil {
		t.Error("Should error when no current file provided for relative import")
	}
}

func TestNormalizeCaseInsensitive(t *testing.T) {
	r := NewResolver()
	
	// This is currently a no-op, but test it doesn't panic
	result := r.normalizeCaseInsensitive("/Test/Path/File.AIL")
	if result != "/Test/Path/File.AIL" {
		t.Errorf("normalizeCaseInsensitive modified path unexpectedly: %s", result)
	}
}