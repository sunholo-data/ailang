package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestRunAilangCheck_IntraPackageImports verifies that the registry
// validator's check step accepts a multi-module package whose modules
// import each other via the natural canonical form. This is the original
// failure mode reported by the demos/linkedin inbox message (913c0aa4):
//
//	packages/linkedin/
//	├── ailang.toml
//	├── types.ail        # module sunholo/linkedin/types
//	├── auth.ail         # imports sunholo/linkedin/types
//
// Before M-PKG-INTRA-PACKAGE-IMPORTS, validation failed with LDR001
// because each file was checked in isolation and the validator never
// taught the loader that `sunholo/linkedin/types` was a sibling.
//
// The test shells out to the installed `ailang` binary because that's
// exactly what the deployed validator does. If `ailang` is not in PATH
// (e.g. in a hermetic CI environment), the test is skipped — matching
// the existing test pattern in TestPublish_ValidPackage_NoGCS.
func TestRunAilangCheck_IntraPackageImports(t *testing.T) {
	if _, err := exec.LookPath("ailang"); err != nil {
		t.Skip("ailang not in PATH; skipping integration test")
	}

	type form struct {
		name        string
		importLine  string
		description string
	}

	forms := []form{
		{
			name:        "bare_canonical",
			importLine:  "import sunholo/intra_validator_pkg/types (name)",
			description: "LinkedIn's reported form — no pkg/ prefix",
		},
		{
			name:        "relative",
			importLine:  "import ./types (name)",
			description: "M-DX-RELIMPORT shorthand",
		},
		{
			name:        "canonical_pkg",
			importLine:  "import pkg/sunholo/intra_validator_pkg/types (name)",
			description: "fully canonical pkg/ prefix",
		},
	}

	for _, f := range forms {
		t.Run(f.name, func(t *testing.T) {
			dir := t.TempDir()
			writeFile(t, dir, "ailang.toml", `[package]
name = "sunholo/intra_validator_pkg"
version = "0.1.0"
edition = "1"

[exports]
modules = ["sunholo/intra_validator_pkg/types", "sunholo/intra_validator_pkg/service"]

[effects]
max = []

[stability]
level = "experimental"

[metadata]
ai_summary = "test fixture for intra-package imports"
`)
			writeFile(t, dir, "types.ail", `module sunholo/intra_validator_pkg/types

export pure func name() -> string = "Hi"
`)
			writeFile(t, dir, "service.ail", `module sunholo/intra_validator_pkg/service

`+f.importLine+`

export pure func greet() -> string = "Hello ${name()}"
`)

			ok, out := runAilangCheck(dir)
			if !ok {
				t.Fatalf("%s (%s) should pass validation; got:\n%s",
					f.name, f.description, out)
			}
			// Sanity: the check should not surface LDR001 nor a missing-lock
			// error in its output even when ok=true (false-positive guard).
			if strings.Contains(out, "LDR001") || strings.Contains(out, "ailang.lock") {
				t.Errorf("%s output unexpectedly mentions LDR001/ailang.lock: %s",
					f.name, out)
			}
		})
	}
}

// TestRunAilangCheck_ExternalImportStillErrors guards against the self-only
// fallback silently swallowing genuinely-missing dependencies. If a published
// package imports `pkg/sunholo/firestore/client` and never resolves it
// (no ailang.lock, no client at all), validation MUST fail. Without this
// guard the M-PKG fix would mask broken external imports.
func TestRunAilangCheck_ExternalImportStillErrors(t *testing.T) {
	if _, err := exec.LookPath("ailang"); err != nil {
		t.Skip("ailang not in PATH; skipping integration test")
	}

	dir := t.TempDir()
	writeFile(t, dir, "ailang.toml", `[package]
name = "sunholo/intra_validator_extern"
version = "0.1.0"
edition = "1"

[exports]
modules = ["sunholo/intra_validator_extern/service"]

[effects]
max = []

[stability]
level = "experimental"

[metadata]
ai_summary = "test fixture"
`)
	writeFile(t, dir, "service.ail", `module sunholo/intra_validator_extern/service

import pkg/sunholo/firestore/client (getDoc)

export pure func use() -> string = "stub"
`)

	ok, out := runAilangCheck(dir)
	if ok {
		t.Fatalf("external import to a non-existent package must error; output:\n%s", out)
	}
	// Error should at least mention the missing package so a publisher
	// can act on it.
	if !bytes.Contains([]byte(out), []byte("firestore")) {
		t.Errorf("error output should mention the missing package; got: %s", out)
	}
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}
