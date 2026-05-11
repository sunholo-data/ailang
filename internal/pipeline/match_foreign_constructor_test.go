package pipeline

// M-MATCH-ADT-XCHECK (v0.18.10) regression suite.
//
// The bug class this catches: writing
//   match getString(json, "model") { Err(_) => ..., Ok(_) => ... }
// when getString actually returns Option[string]. Today the AILANG
// typechecker accepts this (constructor patterns are type-checked in
// isolation against their OWN ADT, not cross-checked against the
// scrutinee's type). At runtime, the match panics with
// "no pattern matched in match expression" since neither Err nor Ok
// can ever bind to a Some/None value.
//
// This sprint adds a typechecker cross-check: when a constructor
// pattern's ADT differs from the scrutinee's ADT (and BOTH are
// concretely resolved), emit a MatchForeignConstructorError naming
// both ADTs and their constructor lists.
//
// Tests below cover:
//   - Option scrutinee with Result-ctor arms (the canonical bug)
//   - Result scrutinee with Option-ctor arms (symmetric)
//   - list scrutinee with Option-ctor arms (no list-vs-Option confusion)
//   - Valid matches still type-check (regression)
//   - Wildcard / variable arms alongside foreign ctors (mixed-arm case)

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeMiniStdlib creates a minimal Option + Result + list-importing stdlib
// in tempDir/std. Returns nothing — fails the test on write errors.
//
// Note: we use Option from std/option, NOT redefining it inline, because
// the cross-check needs the constructor → ADT name mapping to be
// populated from imports (the production path) — inline ADTs in the test
// module would land in a different namespace and might miss the bug.
func writeMiniStdlib(t *testing.T, stdDir string) {
	t.Helper()
	if err := os.MkdirAll(stdDir, 0755); err != nil {
		t.Fatalf("failed to create std dir: %v", err)
	}
	files := map[string]string{
		"option.ail": `module std/option
export type Option[a] = Some(a) | None
`,
		"result.ail": `module std/result
export type Result[a, e] = Ok(a) | Err(e)
`,
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(stdDir, name), []byte(content), 0644); err != nil {
			t.Fatalf("failed to write %s: %v", name, err)
		}
	}
}

// runCheck runs the pipeline in check mode against the given source code,
// returning the error (if any). Caller asserts on whether the error
// indicates the expected foreign-constructor rejection.
func runCheck(t *testing.T, tempDir, testContent string) error {
	t.Helper()
	testFile := filepath.Join(tempDir, "test.ail")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("failed to write test.ail: %v", err)
	}
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change to temp dir: %v", err)
	}
	defer os.Chdir(originalDir)

	src := Source{Filename: "test.ail"}
	cfg := Config{Mode: ModeCheck}
	_, err = Run(cfg, src)
	return err
}

// TestMatchForeignConstructor_OptionScrutinee_ResultArms is the canonical
// bug shape: match an Option value using Result's Err/Ok constructors.
// This is exactly what crashed motoko_ext_compaction_ai 0.1.1.
func TestMatchForeignConstructor_OptionScrutinee_ResultArms(t *testing.T) {
	tempDir := t.TempDir()
	writeMiniStdlib(t, filepath.Join(tempDir, "std"))

	testContent := `module test
import std/option (Option, Some, None)
import std/result (Ok, Err)

export pure func main() -> string =
  let x: Option[string] = Some("hi") in
  match x {
    Err(_) => "err",
    Ok(m) => m
  }
`
	err := runCheck(t, tempDir, testContent)
	if err == nil {
		t.Fatal("expected typecheck error for match Option { Err(_) ... Ok(_) ... }, got nil")
	}
	msg := err.Error()
	// Error should name BOTH ADTs (Option and Result) and the offending constructor.
	if !strings.Contains(msg, "Err") {
		t.Errorf("error message should name the foreign constructor 'Err', got: %s", msg)
	}
	if !strings.Contains(msg, "Option") || !strings.Contains(msg, "Result") {
		t.Errorf("error message should name both 'Option' and 'Result' ADTs, got: %s", msg)
	}
}

// TestMatchForeignConstructor_ResultScrutinee_OptionArms is the symmetric
// case: matching a Result value using Option's Some/None constructors.
func TestMatchForeignConstructor_ResultScrutinee_OptionArms(t *testing.T) {
	tempDir := t.TempDir()
	writeMiniStdlib(t, filepath.Join(tempDir, "std"))

	testContent := `module test
import std/result (Result, Ok, Err)
import std/option (Some, None)

export pure func main() -> string =
  let x: Result[string, string] = Ok("hi") in
  match x {
    Some(s) => s,
    None => "none"
  }
`
	err := runCheck(t, tempDir, testContent)
	if err == nil {
		t.Fatal("expected typecheck error for match Result { Some(_) ... None ... }, got nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, "Some") && !strings.Contains(msg, "None") {
		t.Errorf("error message should name an offending constructor (Some or None), got: %s", msg)
	}
	if !strings.Contains(msg, "Result") || !strings.Contains(msg, "Option") {
		t.Errorf("error message should name both 'Result' and 'Option' ADTs, got: %s", msg)
	}
}

// TestMatchForeignConstructor_ValidOptionMatch is the negative test: a
// valid match against an Option scrutinee using its OWN constructors
// must still type-check. Without this, the cross-check would be useless.
func TestMatchForeignConstructor_ValidOptionMatch(t *testing.T) {
	tempDir := t.TempDir()
	writeMiniStdlib(t, filepath.Join(tempDir, "std"))

	testContent := `module test
import std/option (Option, Some, None)

export pure func main() -> string =
  let x: Option[string] = Some("hi") in
  match x {
    Some(m) => m,
    None => "none"
  }
`
	err := runCheck(t, tempDir, testContent)
	if err != nil {
		t.Fatalf("valid Option match should type-check cleanly, got error: %s", err)
	}
}

// TestMatchForeignConstructor_ValidResultMatch is the negative test for
// Result: matching with Ok/Err must still type-check.
func TestMatchForeignConstructor_ValidResultMatch(t *testing.T) {
	tempDir := t.TempDir()
	writeMiniStdlib(t, filepath.Join(tempDir, "std"))

	testContent := `module test
import std/result (Result, Ok, Err)

export pure func main() -> string =
  let x: Result[string, string] = Ok("hi") in
  match x {
    Ok(m) => m,
    Err(e) => e
  }
`
	err := runCheck(t, tempDir, testContent)
	if err != nil {
		t.Fatalf("valid Result match should type-check cleanly, got error: %s", err)
	}
}

// TestMatchForeignConstructor_ListScrutinee_OptionArms verifies that
// matching a list value with Option's Some/None constructors is
// rejected. Common confusion: AILANG's list operations (e.g.
// std/list.head) historically returned Option, so users may try to
// pattern-match a list directly with Some/None instead of using ::
// and [] list patterns.
func TestMatchForeignConstructor_ListScrutinee_OptionArms(t *testing.T) {
	tempDir := t.TempDir()
	writeMiniStdlib(t, filepath.Join(tempDir, "std"))

	testContent := `module test
import std/option (Some, None)

export pure func main() -> string =
  let xs: [int] = [1, 2, 3] in
  match xs {
    Some(x) => "some",
    None => "none"
  }
`
	err := runCheck(t, tempDir, testContent)
	if err == nil {
		t.Fatal("expected typecheck error: list scrutinee can't be matched with Option ctors")
	}
}

// TestMatchForeignConstructor_WildcardArmAllowed ensures wildcard arms
// mixed with foreign-constructor arms don't suppress the error — the
// wildcard arm is allowed, but the foreign-constructor arm still fails.
func TestMatchForeignConstructor_WildcardArmAllowed(t *testing.T) {
	tempDir := t.TempDir()
	writeMiniStdlib(t, filepath.Join(tempDir, "std"))

	testContent := `module test
import std/option (Option, Some, None)
import std/result (Err)

export pure func main() -> string =
  let x: Option[string] = Some("hi") in
  match x {
    _ => "any",
    Err(_) => "bad"
  }
`
	err := runCheck(t, tempDir, testContent)
	if err == nil {
		t.Fatal("expected typecheck error: wildcard arm shouldn't suppress foreign-ctor error")
	}
}
