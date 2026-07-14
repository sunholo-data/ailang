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

// When a symbol exists in NO module, suggest the closest export of the module the
// user aimed at — so `flushStdout` points at `flush` (keeping one canonical name)
// without shipping a duplicate alias. A genuinely unknown symbol still gets nothing.
func TestIMP010_ClosestExportSuggestion(t *testing.T) {
	oldL, oldS := Locator, SymbolsOf
	defer func() { Locator, SymbolsOf = oldL, oldS }()
	Locator = func(string) []string { return nil } // symbol exists nowhere
	SymbolsOf = func(mod string) []string {
		if mod == "std/io" {
			return []string{"print", "println", "flush", "printErr", "eprintln", "readLine", "writeBytes", "exit"}
		}
		return nil
	}
	// Prefix relationship: flushStdout -> flush.
	if got := IMP010("flushStdout", "std/io"); !strings.Contains(got, "'flush'") || !strings.Contains(got, "did you mean") {
		t.Errorf("flushStdout should suggest flush, got %q", got)
	}
	// Typo distance: writeByte -> writeBytes (edit distance 1).
	if got := IMP010("writeByte", "std/io"); !strings.Contains(got, "'writeBytes'") {
		t.Errorf("writeByte should suggest writeBytes, got %q", got)
	}
	// Genuinely unknown -> no misleading suggestion.
	if got := IMP010("zzqqxx", "std/io"); got != "" {
		t.Errorf("unknown symbol should get no suggestion, got %q", got)
	}
}

// M-DX-AI-DISCOVERY M3: ModuleSuggestion recovers a mistyped stdlib MODULE name.

func modLocatorFixture() []string {
	// A representative sorted slice of the real stdlib module set.
	return []string{"std/clock", "std/datetime", "std/io", "std/list", "std/net", "std/regex", "std/string"}
}

// Alias-table hit: time->clock is edit-distance 5, so ONLY the alias table catches
// it (the highest-value trap). Alias hits do not need the locator.
func TestModuleSuggestion_AliasHit(t *testing.T) {
	old := ModuleLocator
	defer func() { ModuleLocator = old }()
	ModuleLocator = nil // prove the alias works without a module list
	if got := ModuleSuggestion("time"); got != "std/clock" {
		t.Errorf("time should alias to std/clock, got %q", got)
	}
}

// Distance hit: `lst` -> `std/list` (edit distance 1) via Levenshtein.
func TestModuleSuggestion_DistanceHit(t *testing.T) {
	old := ModuleLocator
	defer func() { ModuleLocator = old }()
	ModuleLocator = modLocatorFixture
	if got := ModuleSuggestion("lst"); got != "std/list" {
		t.Errorf("lst should suggest std/list, got %q", got)
	}
}

// Empty / no-match -> "".
func TestModuleSuggestion_NoMatch(t *testing.T) {
	old := ModuleLocator
	defer func() { ModuleLocator = old }()
	ModuleLocator = modLocatorFixture
	if got := ModuleSuggestion(""); got != "" {
		t.Errorf("empty name should give no suggestion, got %q", got)
	}
	if got := ModuleSuggestion("zzqqxxww"); got != "" {
		t.Errorf("far-off name should give no suggestion, got %q", got)
	}
}

// Alias ALWAYS outranks a distance hit: `date` aliases to std/datetime even though
// it is also within edit distance of other modules.
func TestModuleSuggestion_AliasOutranksDistance(t *testing.T) {
	old := ModuleLocator
	defer func() { ModuleLocator = old }()
	// Locator offers a close distance match, but the alias must win.
	ModuleLocator = func() []string { return []string{"std/data", "std/datetime"} }
	if got := ModuleSuggestion("date"); got != "std/datetime" {
		t.Errorf("date should alias to std/datetime (alias outranks distance), got %q", got)
	}
}

// Tie-break: among equal Levenshtein distances, the lexicographically-smallest
// module wins (the locator list is sorted, so a first-min scan is lexicographic).
func TestModuleSuggestion_TieBreakLexicographic(t *testing.T) {
	old := ModuleLocator
	defer func() { ModuleLocator = old }()
	// "aa" and "ab" are both edit-distance 1 from "ax"; sorted list => std/aa wins.
	ModuleLocator = func() []string { return []string{"std/aa", "std/ab"} }
	if got := ModuleSuggestion("ax"); got != "std/aa" {
		t.Errorf("tie should break to lexicographically-smallest std/aa, got %q", got)
	}
}

// Exact module name is NOT a "did you mean" (no self-suggestion).
func TestModuleSuggestion_ExactNotSuggested(t *testing.T) {
	old := ModuleLocator
	defer func() { ModuleLocator = old }()
	ModuleLocator = modLocatorFixture
	if got := ModuleSuggestion("list"); got != "" {
		t.Errorf("exact module name should not self-suggest, got %q", got)
	}
}

// No locator wired + no alias -> "" (no panic, no misleading line).
func TestModuleSuggestion_NoLocatorNoAlias(t *testing.T) {
	old := ModuleLocator
	defer func() { ModuleLocator = old }()
	ModuleLocator = nil
	if got := ModuleSuggestion("wibble"); got != "" {
		t.Errorf("no locator + no alias should give no suggestion, got %q", got)
	}
}
