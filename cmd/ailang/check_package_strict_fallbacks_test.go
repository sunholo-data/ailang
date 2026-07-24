package main

// M-CHECK-STRICT-FALLBACKS M4: package-channel HARD ERROR (exit 1).
//
// `ailang check --package` is the publish boundary: an unannotated empty/default
// `Ok(...)` in a Result-returning function must FAIL the check (exit 1) with a
// STRICT_FALLBACK_001 message. The same file under plain `ailang check` stays
// exit-0-with-warning (regression guard). @allow_empty_ok flips it back to
// exit 0.
//
// Drives the real binary end-to-end (exec) because the check --package path
// calls os.Exit directly and the exit code IS the contract under test.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeStrictFallbackPackage writes an ailang.toml + a src module into dir.
// bodyOk is the offending (or annotated) function source. Returns the file path
// of the module.
func writeStrictFallbackPackage(t *testing.T, dir, moduleSrc string) {
	t.Helper()
	manifest := `[package]
name = "sfvendor/sfpkg"
version = "0.1.0"
edition = "1"

[exports]
modules = ["sfvendor/sfpkg/client"]
`
	if err := os.WriteFile(filepath.Join(dir, "ailang.toml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write ailang.toml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "client.ail"), []byte(moduleSrc), 0o644); err != nil {
		t.Fatalf("write client.ail: %v", err)
	}
}

const strictFallbackViolationSrc = `module sfvendor/sfpkg/client
import std/result (Result, Ok, Err)
import std/json (Json, jo, get)

export func getDoc(json: Json) -> Result[Json, string] =
  match get(json, "fields") {
    Some(fields) => Ok(fields),
    None => Ok(jo([]))
  }
`

const strictFallbackAnnotatedSrc = `module sfvendor/sfpkg/client
import std/result (Result, Ok, Err)
import std/json (Json, jo, get)

@allow_empty_ok("missing 'fields' legitimately means an empty object here")
export func getDoc(json: Json) -> Result[Json, string] =
  match get(json, "fields") {
    Some(fields) => Ok(fields),
    None => Ok(jo([]))
  }
`

// TestCheckPackage_StrictFallback_ViolationExits1 is the core publish-gate test:
// an unannotated Ok(jo([])) in a Result function fails check --package (exit 1).
func TestCheckPackage_StrictFallback_ViolationExits1(t *testing.T) {
	bin := buildAilang(t)
	dir := t.TempDir()
	writeStrictFallbackPackage(t, dir, strictFallbackViolationSrc)

	stdout, stderr, exit := runAilangBin(t, bin, "check", "--package", dir)
	combined := stdout + stderr
	if exit != 1 {
		t.Fatalf("expected exit 1 for unannotated empty-Ok, got %d\noutput:\n%s", exit, combined)
	}
	if !strings.Contains(combined, "STRICT_FALLBACK_001") {
		t.Errorf("expected STRICT_FALLBACK_001 in output, got:\n%s", combined)
	}
}

// TestCheckPackage_StrictFallback_AnnotatedExits0 proves @allow_empty_ok flips
// the publish gate back to passing (exit 0).
func TestCheckPackage_StrictFallback_AnnotatedExits0(t *testing.T) {
	bin := buildAilang(t)
	dir := t.TempDir()
	writeStrictFallbackPackage(t, dir, strictFallbackAnnotatedSrc)

	stdout, stderr, exit := runAilangBin(t, bin, "check", "--package", dir)
	combined := stdout + stderr
	if exit != 0 {
		t.Fatalf("expected exit 0 for @allow_empty_ok-annotated fn, got %d\noutput:\n%s", exit, combined)
	}
	if strings.Contains(combined, "STRICT_FALLBACK_001") {
		t.Errorf("annotated fn must not emit STRICT_FALLBACK_001, got:\n%s", combined)
	}
}

// TestCheckPackage_StrictFallback_PlainCheckStaysExit0 is the regression guard:
// the SAME violating file under plain `ailang check` (not --package) warns but
// exits 0 (dev is advisory, publish is strict).
func TestCheckPackage_StrictFallback_PlainCheckStaysExit0(t *testing.T) {
	bin := buildAilang(t)
	dir := t.TempDir()
	writeStrictFallbackPackage(t, dir, strictFallbackViolationSrc)

	file := filepath.Join(dir, "client.ail")
	stdout, stderr, exit := runAilangBin(t, bin, "check", file)
	combined := stdout + stderr
	if exit != 0 {
		t.Fatalf("plain check must stay exit 0 (advisory), got %d\noutput:\n%s", exit, combined)
	}
	if !strings.Contains(combined, "STRICT_FALLBACK_001") {
		t.Errorf("plain check should still emit the STRICT_FALLBACK_001 warning, got:\n%s", combined)
	}
}
