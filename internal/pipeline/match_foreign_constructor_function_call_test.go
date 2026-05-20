package pipeline

// M-SCHEME-IMPORT-PRESERVE-ADT-HEAD (v0.22.0) regression suite.
//
// These tests cover the function-call-scrutinee case that
// M-MATCH-ADT-XCHECK (v0.18.10) could not catch. The pattern-check
// logic in internal/types/typechecker_patterns.go has a fast-path
// that fires when the scrutinee resolves to a concrete TApp. For
// let-bound, type-annotated scrutinees it worked; for function-call
// scrutinees the type used to come through as a fresh TVar because
// inferLet / inferLetRec discarded the substitution returned by
// SolveConstraints, so generalization quantified the return position
// even when unification had bound it to a concrete ADT.
//
// After the M1 fix (apply sub before generalize at the three
// typechecker_functions.go discard sites), function-call scrutinees
// import with concrete ADT heads (e.g. Option[string]) and the
// existing fast-path xcheck fires correctly.
//
// Bug report: msg_20260520_111521_44c38751 (sunholo-demos/cognitive_commons)

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeOptionResultStdlib adds a helper module that returns Option[T] from a
// function — mimicking the std/json.getNumber shape that triggered the
// original bug report.
func writeOptionResultStdlib(t *testing.T, stdDir string) {
	t.Helper()
	writeMiniStdlib(t, stdDir)
	// Add a helper module exporting a function with a concrete Option return.
	// This is the import surface that previously got corrupted on import.
	helper := `module std/json
import std/option (Option, Some, None)

export type Json = JNum(float) | JStr(string) | JNull

-- getNumber: declared to return Option[float]. Pre-M1 fix, the imported
-- scheme was stored as forall α. (Json, string) -> α, silently masking
-- pattern-match bugs at every call site. Post-fix, the scheme MUST be
-- (Json, string) -> Option[float] for the foreign-ctor xcheck to fire.
export pure func getNumber(j: Json, k: string) -> Option[float] =
  match j {
    JNum(n) => Some(n),
    _       => None
  }
`
	if err := os.WriteFile(filepath.Join(stdDir, "json.ail"), []byte(helper), 0644); err != nil {
		t.Fatalf("failed to write std/json.ail: %v", err)
	}
}

// TestSchemeImport_FunctionCallScrutinee_ForeignCtorRejected is the canonical
// reproducer from msg_20260520_111521_44c38751. Match a function-call result
// (Option[float]) using Result's Ok/Err constructors. Must produce a
// MatchForeignConstructorError naming both ADTs.
//
// Pre-M1: passes ailang check silently (bug).
// Post-M1: typecheck error.
func TestSchemeImport_FunctionCallScrutinee_ForeignCtorRejected(t *testing.T) {
	tempDir := t.TempDir()
	writeOptionResultStdlib(t, filepath.Join(tempDir, "std"))

	testContent := `module test
import std/json (Json, getNumber, JNum)
import std/result (Ok, Err)

export pure func main() -> float =
  match getNumber(JNum(3.14), "x") {
    Ok(v)  => v,
    Err(_) => 0.0
  }
`
	err := runCheck(t, tempDir, testContent)
	if err == nil {
		t.Fatal("expected typecheck error for match getNumber(...) { Ok ... Err ... } — getNumber returns Option, arms use Result")
	}
	msg := err.Error()
	if !strings.Contains(msg, "Ok") && !strings.Contains(msg, "Err") {
		t.Errorf("error message should name an offending Result ctor, got: %s", msg)
	}
	if !strings.Contains(msg, "Option") || !strings.Contains(msg, "Result") {
		t.Errorf("error message should name both 'Option' and 'Result', got: %s", msg)
	}
}

// TestSchemeImport_NestedFunctionCallMatch covers the cognitive_commons shape:
// outer match on a Result, inner match on a function call returning Option.
// The OUTER arms are correctly typed; only the INNER arms should be flagged.
func TestSchemeImport_NestedFunctionCallMatch(t *testing.T) {
	tempDir := t.TempDir()
	writeOptionResultStdlib(t, filepath.Join(tempDir, "std"))

	testContent := `module test
import std/json (Json, getNumber, JNum)
import std/result (Result, Ok, Err)

export pure func decode(s: string) -> Result[Json, string] = Ok(JNum(3.14))

export pure func main() -> float =
  match decode("{\"x\":3.14}") {
    Ok(j) => match getNumber(j, "x") {
      Ok(v)  => v,
      Err(_) => 0.0
    },
    Err(_) => 0.0
  }
`
	err := runCheck(t, tempDir, testContent)
	if err == nil {
		t.Fatal("expected typecheck error: inner match on getNumber(...) uses Result ctors against an Option scrutinee")
	}
	msg := err.Error()
	if !strings.Contains(msg, "Option") || !strings.Contains(msg, "Result") {
		t.Errorf("error message should name both 'Option' and 'Result', got: %s", msg)
	}
}

// TestSchemeImport_CrossModule_OptionScrutinee_ResultArms verifies that the
// fix also works when the scrutinee comes from a deeper module dependency
// chain (not just direct import) — the most common shape in real codebases.
func TestSchemeImport_CrossModule_OptionScrutinee_ResultArms(t *testing.T) {
	tempDir := t.TempDir()
	writeOptionResultStdlib(t, filepath.Join(tempDir, "std"))

	// Add a third module that re-exports a function that returns Option.
	indirection := `module std/helper
import std/option (Option)
import std/json (Json, getNumber)

export pure func getX(j: Json) -> Option[float] = getNumber(j, "x")
`
	if err := os.WriteFile(filepath.Join(tempDir, "std", "helper.ail"), []byte(indirection), 0644); err != nil {
		t.Fatalf("failed to write helper.ail: %v", err)
	}

	testContent := `module test
import std/helper (getX)
import std/json (Json, JNum)
import std/result (Ok, Err)

export pure func main() -> float =
  match getX(JNum(3.14)) {
    Ok(v)  => v,
    Err(_) => 0.0
  }
`
	err := runCheck(t, tempDir, testContent)
	if err == nil {
		t.Fatal("expected typecheck error: cross-module Option-returning function with Result-ctor arms")
	}
	msg := err.Error()
	if !strings.Contains(msg, "Option") {
		t.Errorf("error message should name 'Option' (scrutinee ADT), got: %s", msg)
	}
}

// TestSchemeImport_LegitimatelyPolymorphic_StillWorks is the negative test:
// a function that genuinely returns a polymorphic type (forall a. ... -> a)
// must still type-check cleanly when its result is matched. Without this,
// the fix could over-constrain legitimately generic code.
//
// Note: this works because `unwrap` here has a constrained polymorphic
// return (passed through Option), so the scrutinee resolves to the
// specific ADT instantiated at the call site, not a free TVar.
func TestSchemeImport_LegitimatelyPolymorphic_StillWorks(t *testing.T) {
	tempDir := t.TempDir()
	writeMiniStdlib(t, filepath.Join(tempDir, "std"))

	testContent := `module test
import std/option (Option, Some, None)

-- A generic helper that returns Option[a] — its scheme genuinely IS polymorphic.
-- After the M1 fix, the Option head is preserved but the element type stays
-- a quantified variable. The match below uses Some/None (Option's OWN ctors)
-- so the xcheck must NOT fire.
export pure func wrap[a](x: a) -> Option[a] = Some(x)

export pure func main() -> int =
  match wrap(42) {
    Some(v) => v,
    None    => 0
  }
`
	err := runCheck(t, tempDir, testContent)
	if err != nil {
		t.Fatalf("expected clean typecheck for legitimate polymorphic match, got: %v", err)
	}
}
