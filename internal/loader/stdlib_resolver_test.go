package loader

import (
	"runtime"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/importhint"
	"github.com/sunholo-data/ailang/internal/stdlibroot"
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
			errMsg:    "contains invalid characters", // Colon not allowed in [a-zA-Z0-9_/-]
		},
		{
			name:      "UNC path",
			input:     "\\\\server\\share",
			shouldErr: true,
			errMsg:    "contains invalid characters", // Backslashes not allowed in [a-zA-Z0-9_/-]
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
					// Platform-specific handling for absolute path test
					// On Windows, /etc/passwd isn't absolute, so it hits suspicious pattern check
					if tt.name == "absolute path unix" && runtime.GOOS == "windows" {
						if !strings.Contains(err.Error(), "contains suspicious pattern") {
							t.Errorf("expected error containing 'contains suspicious pattern' (Windows), got %q", err.Error())
						}
					} else {
						t.Errorf("expected error containing %q, got %q", tt.errMsg, err.Error())
					}
				}
			} else {
				if err != nil {
					t.Errorf("expected no error for input %q, got: %v", tt.input, err)
				}
			}
		})
	}
}

// M-DX-AI-DISCOVERY M3: errWithSearchTrace appends a "did you mean" line + the
// available-module list, sourced from internal/importhint (alias table +
// Levenshtein) and its injected ModuleLocator.

func TestErrWithSearchTrace_AliasSuggestionAndList(t *testing.T) {
	old := importhint.ModuleLocator
	defer func() { importhint.ModuleLocator = old }()
	importhint.ModuleLocator = func() []string {
		return []string{"std/clock", "std/io", "std/list"}
	}

	r := NewStdlibResolver("", false, false)
	err := r.errWithSearchTrace("time", stdlibroot.Root{Dir: "/x/std", Source: "env"})
	got := err.Error()

	// Alias-table catch: time -> clock (edit distance 5, only the alias catches it).
	if !strings.Contains(got, "did you mean: std/clock?") {
		t.Errorf("expected 'did you mean: std/clock?', got:\n%s", got)
	}
	// Available list with count.
	if !strings.Contains(got, "available: std/clock, std/io, std/list (3 modules)") {
		t.Errorf("expected sorted available list with count, got:\n%s", got)
	}
	// The original searched/tip block is preserved.
	if !strings.Contains(got, "stdlib module not found: std/time") || !strings.Contains(got, "tip:") {
		t.Errorf("original error block should be preserved, got:\n%s", got)
	}
}

func TestErrWithSearchTrace_NoMatchNoMisleadingLine(t *testing.T) {
	old := importhint.ModuleLocator
	defer func() { importhint.ModuleLocator = old }()
	importhint.ModuleLocator = func() []string { return []string{"std/clock", "std/io"} }

	r := NewStdlibResolver("", false, false)
	got := r.errWithSearchTrace("zzqqxxww", stdlibroot.Root{Dir: "/x/std", Source: "env"}).Error()

	if strings.Contains(got, "did you mean") {
		t.Errorf("a far-off unknown module must NOT get a did-you-mean line, got:\n%s", got)
	}
	// The available list is still shown (helpful, not misleading).
	if !strings.Contains(got, "available: std/clock, std/io (2 modules)") {
		t.Errorf("expected available list even with no suggestion, got:\n%s", got)
	}
}

func TestErrWithSearchTrace_ModuleListUnavailableNote(t *testing.T) {
	old := importhint.ModuleLocator
	defer func() { importhint.ModuleLocator = old }()
	importhint.ModuleLocator = func() []string { return nil } // stdlib root unresolved

	r := NewStdlibResolver("", false, false)
	got := r.errWithSearchTrace("time", stdlibroot.Root{Dir: "/x/std", Source: "env"}).Error()

	// Alias suggestion (static data) still prints even with no module list.
	if !strings.Contains(got, "did you mean: std/clock?") {
		t.Errorf("alias suggestion should print without a module list, got:\n%s", got)
	}
	// Explicit unavailable note, never a silent skip.
	if !strings.Contains(got, "available: (module list unavailable") {
		t.Errorf("expected explicit unavailable note, got:\n%s", got)
	}
}
