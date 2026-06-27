package importhint

import (
	"strings"
	"testing"
)

// M-AGENT-STUCK-FIXES M2: IMP010 must steer the agent off the two import mistakes that loop it.

func TestIMP010_AutoImportedBuiltin(t *testing.T) {
	got := IMP010("show", "std/string")
	if !strings.Contains(got, "builtin") || !strings.Contains(got, "remove it") {
		t.Errorf("show should get the auto-import hint, got %q", got)
	}
}

func TestIMP010_WrongModule(t *testing.T) {
	old := Locator
	defer func() { Locator = old }()
	Locator = func(name string) []string {
		if name == "println" {
			return []string{"std/io"}
		}
		return nil
	}
	got := IMP010("println", "std/string")
	if !strings.Contains(got, "std/io") || !strings.Contains(got, "not 'std/string'") {
		t.Errorf("println should point at std/io, got %q", got)
	}
}

func TestIMP010_UnknownNoHint(t *testing.T) {
	old := Locator
	defer func() { Locator = old }()
	Locator = func(string) []string { return nil }
	if got := IMP010("nope", "std/string"); got != "" {
		t.Errorf("unknown symbol should get no hint, got %q", got)
	}
}

// If the only module that exports the symbol is the one already imported, suppress the hint
// (it would otherwise say "import it from there, not <the same module>").
func TestIMP010_SelfModuleFiltered(t *testing.T) {
	old := Locator
	defer func() { Locator = old }()
	Locator = func(string) []string { return []string{"std/string"} }
	if got := IMP010("foo", "std/string"); got != "" {
		t.Errorf("self-only locator should produce no hint, got %q", got)
	}
}
