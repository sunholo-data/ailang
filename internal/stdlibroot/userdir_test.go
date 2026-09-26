package stdlibroot

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestUserDataDir(t *testing.T) {

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
			t.Setenv("XDG_DATA_HOME", tt.xdgDataHome)
			t.Setenv("HOME", tt.home)
			t.Setenv("APPDATA", tt.appdata)

			result := UserDataDir()

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
