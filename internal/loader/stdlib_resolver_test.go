package loader

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestValidateModuleName(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		shouldErr bool
		errMsg    string
	}{
		// Valid names
		{
			name:      "simple module name",
			input:     "io",
			shouldErr: false,
		},
		{
			name:      "module with slash",
			input:     "std/io",
			shouldErr: false,
		},
		{
			name:      "nested module",
			input:     "std/internal/utils",
			shouldErr: false,
		},
		{
			name:      "module with underscore",
			input:     "string_utils",
			shouldErr: false,
		},
		{
			name:      "module with hyphen",
			input:     "http-client",
			shouldErr: false,
		},
		{
			name:      "module with numbers",
			input:     "v2/api",
			shouldErr: false,
		},

		// Invalid names - security
		{
			name:      "directory traversal with ../",
			input:     "../etc/passwd",
			shouldErr: true,
			errMsg:    "cannot contain '..'",
		},
		{
			name:      "directory traversal in middle",
			input:     "foo/../bar",
			shouldErr: true,
			errMsg:    "cannot contain '..'",
		},
		{
			name:      "absolute path unix",
			input:     "/etc/passwd",
			shouldErr: true,
			errMsg:    "cannot be an absolute path",
		},
		{
			name:      "null byte injection",
			input:     "foo\x00bar",
			shouldErr: true,
			errMsg:    "cannot contain null bytes",
		},
		{
			name:      "suspicious /etc/ pattern",
			input:     "test/etc/passwd",
			shouldErr: true,
			errMsg:    "contains suspicious pattern",
		},
		{
			name:      "suspicious /usr/ pattern",
			input:     "test/usr/bin",
			shouldErr: true,
			errMsg:    "contains suspicious pattern",
		},

		// Invalid names - syntax
		{
			name:      "empty string",
			input:     "",
			shouldErr: true,
			errMsg:    "cannot be empty",
		},
		{
			name:      "only std/ prefix",
			input:     "std/",
			shouldErr: true,
			errMsg:    "cannot be empty",
		},
		{
			name:      "special characters - space",
			input:     "foo bar",
			shouldErr: true,
			errMsg:    "contains invalid characters",
		},
		{
			name:      "special characters - asterisk",
			input:     "foo*bar",
			shouldErr: true,
			errMsg:    "contains invalid characters",
		},
		{
			name:      "special characters - semicolon",
			input:     "foo;bar",
			shouldErr: true,
			errMsg:    "contains invalid characters",
		},
		{
			name:      "backslash (Windows path)",
			input:     "foo\\bar",
			shouldErr: true,
			errMsg:    "contains invalid characters",
		},
		{
			name:      "Windows drive letter C:",
			input:     "c:/windows",
			shouldErr: true,
			errMsg:    "contains invalid characters", // Caught by regex check (colon not allowed)
		},
		{
			name:      "UNC path",
			input:     "\\\\server\\share",
			shouldErr: true,
			errMsg:    "contains invalid characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateModuleName(tt.input)

			if tt.shouldErr {
				if err == nil {
					t.Errorf("expected error for input %q, got nil", tt.input)
					return
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("expected error containing %q, got %q", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("expected no error for input %q, got: %v", tt.input, err)
				}
			}
		})
	}
}

func TestGetUserDataDir(t *testing.T) {
	// Save original env vars and restore after test
	origXDG := os.Getenv("XDG_DATA_HOME")
	origHome := os.Getenv("HOME")
	origAppData := os.Getenv("APPDATA")
	defer func() {
		os.Setenv("XDG_DATA_HOME", origXDG)
		os.Setenv("HOME", origHome)
		os.Setenv("APPDATA", origAppData)
	}()

	tests := []struct {
		name        string
		goos        string
		xdgDataHome string
		home        string
		appdata     string
		wantContain string // Expected substring in result
		wantEmpty   bool
	}{
		{
			name:        "Linux with XDG_DATA_HOME",
			goos:        "linux",
			xdgDataHome: "/custom/data",
			home:        "/home/user",
			wantContain: filepath.Join("/custom/data", "ailang", "std"),
		},
		{
			name:        "Linux without XDG_DATA_HOME",
			goos:        "linux",
			xdgDataHome: "",
			home:        "/home/user",
			wantContain: filepath.Join("/home/user", ".local", "share", "ailang", "std"),
		},
		{
			name:        "Linux with no env vars",
			goos:        "linux",
			xdgDataHome: "",
			home:        "",
			wantEmpty:   true,
		},
		{
			name:        "macOS with HOME",
			goos:        "darwin",
			home:        "/Users/alice",
			wantContain: filepath.Join("/Users/alice", "Library", "Application Support", "ailang", "std"),
		},
		{
			name:      "macOS without HOME",
			goos:      "darwin",
			home:      "",
			wantEmpty: true,
		},
		{
			name:        "Windows with APPDATA",
			goos:        "windows",
			appdata:     "C:\\Users\\alice\\AppData\\Roaming",
			wantContain: filepath.Join("C:\\Users\\alice\\AppData\\Roaming", "ailang", "std"),
		},
		{
			name:      "Windows without APPDATA",
			goos:      "windows",
			appdata:   "",
			wantEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip if test doesn't match current OS
			if tt.goos != runtime.GOOS {
				t.Skipf("Skipping %s test on %s", tt.goos, runtime.GOOS)
			}

			// Set up environment
			os.Setenv("XDG_DATA_HOME", tt.xdgDataHome)
			os.Setenv("HOME", tt.home)
			os.Setenv("APPDATA", tt.appdata)

			result := getUserDataDir()

			if tt.wantEmpty {
				if result != "" {
					t.Errorf("expected empty result, got %q", result)
				}
				return
			}

			if !strings.Contains(result, tt.wantContain) {
				t.Errorf("expected result to contain %q, got %q", tt.wantContain, result)
			}

			// Verify path ends with ailang/std
			expectedSuffix := filepath.Join("ailang", "std")
			if !strings.HasSuffix(result, expectedSuffix) {
				t.Errorf("expected result to end with %q, got %q", expectedSuffix, result)
			}
		})
	}
}

func TestStdlibResolver_ResolveStdlib(t *testing.T) {
	// Create temporary test stdlib directory
	tmpDir := t.TempDir()
	stdDir := filepath.Join(tmpDir, "std")
	if err := os.MkdirAll(stdDir, 0755); err != nil {
		t.Fatalf("failed to create test stdlib dir: %v", err)
	}

	// Create test module file
	ioPath := filepath.Join(stdDir, "io.ail")
	if err := os.WriteFile(ioPath, []byte("export func println() { }"), 0644); err != nil {
		t.Fatalf("failed to create test module: %v", err)
	}

	// Create VERSION file
	versionPath := filepath.Join(stdDir, "VERSION")
	if err := os.WriteFile(versionPath, []byte("v0.4.4\n"), 0644); err != nil {
		t.Fatalf("failed to create VERSION file: %v", err)
	}

	t.Run("resolve existing module", func(t *testing.T) {
		resolver := NewStdlibResolver(stdDir, false, false)
		path, err := resolver.ResolveStdlib("io")
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if path != ioPath {
			t.Errorf("expected path %q, got %q", ioPath, path)
		}
	})

	t.Run("resolve with std/ prefix", func(t *testing.T) {
		resolver := NewStdlibResolver(stdDir, false, false)
		path, err := resolver.ResolveStdlib("std/io")
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if path != ioPath {
			t.Errorf("expected path %q, got %q", ioPath, path)
		}
	})

	t.Run("module not found", func(t *testing.T) {
		resolver := NewStdlibResolver(stdDir, false, false)
		_, err := resolver.ResolveStdlib("nonexistent")
		if err == nil {
			t.Fatal("expected error for nonexistent module")
		}
		if !strings.Contains(err.Error(), "stdlib module not found") {
			t.Errorf("expected 'not found' error, got: %v", err)
		}
		if !strings.Contains(err.Error(), "searched:") {
			t.Errorf("expected search trace in error, got: %v", err)
		}
	})

	t.Run("negative caching", func(t *testing.T) {
		resolver := NewStdlibResolver(stdDir, false, false)
		// First lookup - miss
		_, err1 := resolver.ResolveStdlib("missing")
		if err1 == nil {
			t.Fatal("expected error for missing module")
		}
		// Second lookup - should use cache
		_, err2 := resolver.ResolveStdlib("missing")
		if err2 == nil {
			t.Fatal("expected error for missing module (cached)")
		}
		// Check that it was cached
		if _, found := resolver.checkNegativeCache("missing"); !found {
			t.Error("expected module to be in negative cache")
		}
	})

	t.Run("security validation", func(t *testing.T) {
		resolver := NewStdlibResolver(stdDir, false, false)
		_, err := resolver.ResolveStdlib("../etc/passwd")
		if err == nil {
			t.Fatal("expected error for directory traversal")
		}
		if !strings.Contains(err.Error(), "..") {
			t.Errorf("expected '..' error, got: %v", err)
		}
	})

	t.Run("version mismatch - non-strict", func(t *testing.T) {
		// Create mismatched version
		mismatchDir := filepath.Join(tmpDir, "mismatch")
		if err := os.MkdirAll(mismatchDir, 0755); err != nil {
			t.Fatalf("failed to create mismatch dir: %v", err)
		}
		ioMismatch := filepath.Join(mismatchDir, "io.ail")
		if err := os.WriteFile(ioMismatch, []byte("export func println() { }"), 0644); err != nil {
			t.Fatalf("failed to create mismatch module: %v", err)
		}
		versionMismatch := filepath.Join(mismatchDir, "VERSION")
		if err := os.WriteFile(versionMismatch, []byte("v0.0.1\n"), 0644); err != nil {
			t.Fatalf("failed to create mismatch VERSION: %v", err)
		}

		resolver := NewStdlibResolver(mismatchDir, false, false) // non-strict
		path, err := resolver.ResolveStdlib("io")
		// Should succeed with warning
		if err != nil {
			t.Fatalf("expected no error in non-strict mode, got: %v", err)
		}
		if path != ioMismatch {
			t.Errorf("expected path %q, got %q", ioMismatch, path)
		}
	})

	t.Run("version mismatch - strict", func(t *testing.T) {
		// Create mismatched version
		mismatchDir := filepath.Join(tmpDir, "strict_mismatch")
		if err := os.MkdirAll(mismatchDir, 0755); err != nil {
			t.Fatalf("failed to create strict mismatch dir: %v", err)
		}
		ioMismatch := filepath.Join(mismatchDir, "io.ail")
		if err := os.WriteFile(ioMismatch, []byte("export func println() { }"), 0644); err != nil {
			t.Fatalf("failed to create strict mismatch module: %v", err)
		}
		versionMismatch := filepath.Join(mismatchDir, "VERSION")
		if err := os.WriteFile(versionMismatch, []byte("v0.0.1\n"), 0644); err != nil {
			t.Fatalf("failed to create strict mismatch VERSION: %v", err)
		}

		resolver := NewStdlibResolver(mismatchDir, false, true) // strict mode
		_, err := resolver.ResolveStdlib("io")
		// Should fail in strict mode
		if err == nil {
			t.Fatal("expected error in strict mode for version mismatch")
		}
		if !strings.Contains(err.Error(), "version mismatch") {
			t.Errorf("expected 'version mismatch' error, got: %v", err)
		}
	})
}

func TestStdlibResolver_SearchPathOrder(t *testing.T) {
	// Create multiple stdlib directories
	tmpDir := t.TempDir()

	// Create two stdlib dirs with different versions of io.ail
	stdDir1 := filepath.Join(tmpDir, "std1")
	stdDir2 := filepath.Join(tmpDir, "std2")

	for _, dir := range []string{stdDir1, stdDir2} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("failed to create dir %s: %v", dir, err)
		}
	}

	ioPath1 := filepath.Join(stdDir1, "io.ail")
	ioPath2 := filepath.Join(stdDir2, "io.ail")

	if err := os.WriteFile(ioPath1, []byte("// version 1"), 0644); err != nil {
		t.Fatalf("failed to write io1: %v", err)
	}
	if err := os.WriteFile(ioPath2, []byte("// version 2"), 0644); err != nil {
		t.Fatalf("failed to write io2: %v", err)
	}

	t.Run("CLI path has highest priority", func(t *testing.T) {
		// Set env var to stdDir2, but use CLI flag for stdDir1
		os.Setenv("AILANG_STDLIB_PATH", stdDir2)
		defer os.Unsetenv("AILANG_STDLIB_PATH")

		resolver := NewStdlibResolver(stdDir1, false, false)
		path, err := resolver.ResolveStdlib("io")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Should use CLI path (stdDir1), not env path (stdDir2)
		if path != ioPath1 {
			t.Errorf("expected CLI path %q, got %q", ioPath1, path)
		}
	})
}
