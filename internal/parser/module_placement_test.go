package parser

import (
	"strings"
	"testing"
)

// Module-placement diagnostics (M-PROMPT-FOOTGUNS).
//
// A `module` token at declaration level (i.e. anywhere other than the file's
// first declaration) used to fall through to expression parsing and produce the
// opaque PAR_NO_PREFIX_PARSE + PAR_UNEXPECTED_TOKEN cascade — which never stated
// the one-module-per-file rule. It now produces a coded, fix-carrying diagnostic:
//   - a valid leading module already seen → MOD002 (genuine duplicate)
//   - module-less file with a late module → PAR_MODULE_PLACEMENT (misplaced)
//
// These tests mirror import_placement_test.go and lock in gemini's
// error-recovery state-isolation fix (two late modules → both
// PAR_MODULE_PLACEMENT, the second is NEVER a false MOD002).

// TestModuleDuplicateEmitsMOD002 verifies a second module declaration after a
// valid leading module produces MOD002 stating the rule and naming both paths +
// the first module's position, and NOT the old PAR_NO_PREFIX_PARSE cascade.
func TestModuleDuplicateEmitsMOD002(t *testing.T) {
	input := `module benchmark/math_utils

export func add(a: int, b: int) -> int { a + b }

module benchmark/string_utils

export func greet(name: string) -> string { name }
`
	errs := mustParseError(t, input)

	if !hasErrCode(errs, "MOD002") {
		t.Fatalf("expected MOD002 for duplicate module, got: %v", errs)
	}
	// The coded diagnostic must REPLACE the cascade, not sit alongside it.
	if hasErrCode(errs, "PAR_NO_PREFIX_PARSE") {
		t.Errorf("PAR_NO_PREFIX_PARSE must not appear for a duplicate module; got: %v", errs)
	}
	if hasErrCode(errs, "PAR_UNEXPECTED_TOKEN") {
		t.Errorf("PAR_UNEXPECTED_TOKEN cascade must not appear for a duplicate module; got: %v", errs)
	}

	msg := errs[0].Error()
	if !strings.Contains(msg, "exactly one module declaration per file") {
		t.Errorf("expected the one-module-per-file rule in the message, got: %s", msg)
	}
	if !strings.Contains(msg, "benchmark/string_utils") {
		t.Errorf("expected the duplicate module path named in the message, got: %s", msg)
	}
	if !strings.Contains(msg, "benchmark/math_utils") {
		t.Errorf("expected the first module path named in the message, got: %s", msg)
	}
	// First-declaration position must be reported (line 1, col 1).
	if !strings.Contains(msg, "1:1") {
		t.Errorf("expected the first module's position (1:1) in the message, got: %s", msg)
	}
}

// TestModuleMisplacedEmitsPlacement verifies a module declaration that is not
// first (in a module-less file — here after an import) produces
// PAR_MODULE_PLACEMENT, NOT MOD002 (nothing was duplicated) and NOT the cascade.
func TestModuleMisplacedEmitsPlacement(t *testing.T) {
	input := `import std/io (println)

module test/late

export func main() -> () ! {IO} { println("hi") }
`
	errs := mustParseError(t, input)

	if !hasErrCode(errs, "PAR_MODULE_PLACEMENT") {
		t.Fatalf("expected PAR_MODULE_PLACEMENT for misplaced module, got: %v", errs)
	}
	if hasErrCode(errs, "MOD002") {
		t.Errorf("MOD002 must not fire for a misplaced (non-duplicate) module; got: %v", errs)
	}
	if hasErrCode(errs, "PAR_NO_PREFIX_PARSE") {
		t.Errorf("PAR_NO_PREFIX_PARSE must not appear for a misplaced module; got: %v", errs)
	}

	msg := errs[0].Error()
	if !strings.Contains(msg, "must be the first declaration in the file") {
		t.Errorf("expected the placement rule in the message, got: %s", msg)
	}
	if !strings.Contains(msg, "test/late") {
		t.Errorf("expected the misplaced module path in the suggestion, got: %s", msg)
	}
}

// TestModuleThreeModulesTwoErrors verifies the per-offending-declaration design
// choice: three module declarations (one valid leading + two late) produce
// exactly TWO placement/duplicate errors (one per offending decl) and no
// cascade.
func TestModuleThreeModulesTwoErrors(t *testing.T) {
	input := `module benchmark/a

export func fa() -> int { 1 }

module benchmark/b

export func fb() -> int { 2 }

module benchmark/c

export func fc() -> int { 3 }
`
	errs := mustParseError(t, input)

	// Two offending declarations (b and c), both after a valid leading module →
	// both are genuine duplicates → two MOD002.
	dup := countErrCode(errs, "MOD002")
	if dup != 2 {
		t.Errorf("expected 2 MOD002 errors (one per offending duplicate module), got %d: %v", dup, errs)
	}
	if noprefix := countErrCode(errs, "PAR_NO_PREFIX_PARSE"); noprefix != 0 {
		t.Errorf("expected 0 PAR_NO_PREFIX_PARSE for duplicate modules, got %d: %v", noprefix, errs)
	}
}

// TestModuleTwoLateModulesStateIsolation is the gemini error-recovery fix
// regression: a MODULE-LESS file (no valid leading module) with TWO late module
// declarations must emit PAR_MODULE_PLACEMENT for BOTH — the second must NEVER be
// a false MOD002 referencing a non-existent "first module". This locks in that
// the recovery path does not mutate seenModule/firstModulePos/firstModulePath.
func TestModuleTwoLateModulesStateIsolation(t *testing.T) {
	input := `import std/io (println)

module test/first_late

module test/second_late

export func main() -> () ! {IO} { println("hi") }
`
	errs := mustParseError(t, input)

	placement := countErrCode(errs, "PAR_MODULE_PLACEMENT")
	if placement != 2 {
		t.Errorf("expected 2 PAR_MODULE_PLACEMENT (both late modules), got %d: %v", placement, errs)
	}
	if dup := countErrCode(errs, "MOD002"); dup != 0 {
		t.Errorf("expected 0 MOD002 in a module-less file (state-isolation rule), got %d: %v", dup, errs)
	}
	if noprefix := countErrCode(errs, "PAR_NO_PREFIX_PARSE"); noprefix != 0 {
		t.Errorf("expected 0 PAR_NO_PREFIX_PARSE, got %d: %v", noprefix, errs)
	}
}

// TestModuleLeadingUnaffected verifies a valid single-module file parses cleanly
// — the new diagnostics must fire ONLY on the error path.
func TestModuleLeadingUnaffected(t *testing.T) {
	input := `module benchmark/solution

import std/list (map)

export func main() -> int { 42 }
`
	// Should parse with no errors at all.
	_ = mustParse(t, input)
}

// TestModuleMalformedDuplicateOneError is the cascade-suppression guard: a
// duplicate module whose own declaration is malformed (a hyphen in the path,
// which parseModuleDecl rejects) must still produce exactly ONE MOD002 — the
// module-internal error is truncated (mirroring the errCountBefore pattern), so
// it does not cascade.
func TestModuleMalformedDuplicateOneError(t *testing.T) {
	input := `module benchmark/ok

export func main() -> int { 42 }

module benchmark/bad-name

export func other() -> int { 1 }
`
	errs := mustParseError(t, input)

	dup := countErrCode(errs, "MOD002")
	if dup != 1 {
		t.Errorf("expected exactly 1 MOD002 for a malformed duplicate module, got %d: %v", dup, errs)
	}
	// The module-internal hyphen error must be truncated (not cascaded).
	if hasErrCode(errs, "PAR_HYPHEN_IN_MODULE") {
		t.Errorf("PAR_HYPHEN_IN_MODULE must be truncated inside a duplicate-module recovery; got: %v", errs)
	}
	if hasErrCode(errs, "PAR_NO_PREFIX_PARSE") {
		t.Errorf("PAR_NO_PREFIX_PARSE must not appear; got: %v", errs)
	}
}
