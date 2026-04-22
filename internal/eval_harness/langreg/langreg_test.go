package langreg_test

import (
	"context"
	"testing"

	"github.com/sunholo-data/ailang/internal/eval_harness/langreg"
)

// TestRegistry_BasicLookup verifies python and ailang are registered by init().
func TestRegistry_BasicLookup(t *testing.T) {
	for _, name := range []string{"python", "ailang"} {
		lang, err := langreg.Get(name)
		if err != nil {
			t.Fatalf("Get(%q) error: %v", name, err)
		}
		if lang == nil {
			t.Fatalf("Get(%q) returned nil", name)
		}
		if lang.Name() != name {
			t.Errorf("Get(%q).Name() = %q, want %q", name, lang.Name(), name)
		}
	}
}

// TestRegistry_UnknownLanguage verifies a helpful error for unregistered languages.
func TestRegistry_UnknownLanguage(t *testing.T) {
	_, err := langreg.Get("javascript")
	if err == nil {
		t.Fatal("Get(\"javascript\") should return error, got nil")
	}
}

// TestRegistry_Idempotent verifies calling Register twice is a no-op, not a panic.
func TestRegistry_Idempotent(t *testing.T) {
	lang, _ := langreg.Get("python")
	langreg.Register(lang) // second registration — must not panic or duplicate
	names := langreg.Names()
	count := 0
	for _, n := range names {
		if n == "python" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("python registered %d times after double-Register, want 1", count)
	}
}

// TestRegistry_Names verifies Names() returns sorted language keys.
func TestRegistry_Names(t *testing.T) {
	names := langreg.Names()
	if len(names) < 2 {
		t.Fatalf("Names() returned %d entries, want >= 2", len(names))
	}
	// Must include python and ailang.
	has := map[string]bool{}
	for _, n := range names {
		has[n] = true
	}
	for _, want := range []string{"python", "ailang"} {
		if !has[want] {
			t.Errorf("Names() missing %q", want)
		}
	}
	// Must be sorted.
	for i := 1; i < len(names); i++ {
		if names[i] < names[i-1] {
			t.Errorf("Names() not sorted: %v", names)
		}
	}
}

// TestLanguageContract verifies every registered language implements all methods
// without panicking.
func TestLanguageContract(t *testing.T) {
	for _, name := range langreg.Names() {
		lang := langreg.MustGet(name)
		t.Run(name, func(t *testing.T) {
			if lang.Name() == "" {
				t.Error("Name() returned empty string")
			}
			if lang.DisplayName() == "" {
				t.Error("DisplayName() returned empty string")
			}
			if lang.FileExt() == "" {
				t.Error("FileExt() returned empty string")
			}
			if lang.SolutionFilename() == "" {
				t.Error("SolutionFilename() returned empty string")
			}
			// TaskTemplatePath must be non-empty.
			if lang.TaskTemplatePath() == "" {
				t.Error("TaskTemplatePath() returned empty string")
			}
			// DefaultPrompt must always return something usable.
			if lang.DefaultPrompt() == "" {
				t.Error("DefaultPrompt() returned empty string")
			}
			// NewRunner with no factory set should return a clear error, not panic.
			_, err := lang.NewRunner(context.Background(), nil, "")
			if err == nil {
				// Factory was set — that's also fine.
				t.Logf("NewRunner: factory is wired for %q", name)
			}
		})
	}
}

// TestPython_Fields verifies Python descriptor values match the harness constants.
func TestPython_Fields(t *testing.T) {
	py := langreg.MustGet("python")
	if got := py.FileExt(); got != ".py" {
		t.Errorf("FileExt() = %q, want .py", got)
	}
	if got := py.SolutionFilename(); got != "solution.py" {
		t.Errorf("SolutionFilename() = %q, want solution.py", got)
	}
	if got := py.DisplayName(); got != "Python 3" {
		t.Errorf("DisplayName() = %q, want 'Python 3'", got)
	}
	if got := py.PromptTemplatePath(); got == "" {
		t.Error("PromptTemplatePath() returned empty string for python")
	}
}

// TestAILANG_Fields verifies AILANG descriptor values match the harness constants.
func TestAILANG_Fields(t *testing.T) {
	al := langreg.MustGet("ailang")
	if got := al.FileExt(); got != ".ail" {
		t.Errorf("FileExt() = %q, want .ail", got)
	}
	if got := al.SolutionFilename(); got != "solution.ail" {
		t.Errorf("SolutionFilename() = %q, want solution.ail", got)
	}
	if got := al.DisplayName(); got != "AILANG" {
		t.Errorf("DisplayName() = %q, want AILANG", got)
	}
	// AILANG uses empty string for PromptTemplatePath (falls back to default template).
	// TaskTemplatePath must be non-empty.
	if got := al.TaskTemplatePath(); got == "" {
		t.Error("TaskTemplatePath() returned empty string for ailang")
	}
}

// TestLoadSyntaxRef_DoesNotPanic verifies LoadSyntaxRef never panics even when
// prompt files are missing (falls back to DefaultPrompt).
func TestLoadSyntaxRef_DoesNotPanic(t *testing.T) {
	for _, name := range langreg.Names() {
		lang := langreg.MustGet(name)
		t.Run(name, func(t *testing.T) {
			content, version, err := lang.LoadSyntaxRef("")
			// Must not panic. Error is acceptable (missing prompt files in test env).
			if err != nil {
				t.Logf("LoadSyntaxRef(%q): err=%v (acceptable in test env)", name, err)
			}
			if content == "" {
				t.Errorf("LoadSyntaxRef(%q) returned empty content", name)
			}
			if version == "" {
				t.Errorf("LoadSyntaxRef(%q) returned empty version", name)
			}
		})
	}
}

// TestRegister_NilPanics verifies Register panics on nil input.
func TestRegister_NilPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Register(nil) should panic")
		}
	}()
	langreg.Register(nil)
}
