package pipeline

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestIntraPackageImports_NoLockFile verifies that a fresh multi-module
// package can be type-checked without first running `ailang lock`.
//
// Before the M-PKG-INTRA-PACKAGE-IMPORTS fix, every intra-package import
// form failed in this scenario:
//   - `pkg/<self>/sibling` errored with "requires ailang.toml and ailang.lock"
//   - `./sibling` was normalized to `pkg/<self>/sibling` and hit the same error
//   - bare `<self>/sibling` fell through to project-relative resolution and
//     errored with LDR001
//
// The fix wires a self-only PackageLoader (no lock file required) as a
// fallback when `tryLoadPackageResolver` returns nil. This test exercises
// the full pipeline path through that fallback.
func TestIntraPackageImports_NoLockFile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "ailang-intra-pkg-test")
	if err != nil {
		t.Fatalf("mkdir temp: %v", err)
	}
	defer os.RemoveAll(tempDir)

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer func() { _ = os.Chdir(originalDir) }()

	manifest := `[package]
name = "sunholo/intra_pkg_test"
version = "0.1.0"
edition = "1"

[exports]
modules = ["sunholo/intra_pkg_test/types", "sunholo/intra_pkg_test/service"]
`
	typesSrc := `module sunholo/intra_pkg_test/types

export pure func name() -> string = "Foo"
`
	serviceSrc := `module sunholo/intra_pkg_test/service

import pkg/sunholo/intra_pkg_test/types (name)

export pure func greet() -> string = "Hello ${name()}"
`
	mustWrite(t, tempDir, "ailang.toml", manifest)
	mustWrite(t, tempDir, "types.ail", typesSrc)
	mustWrite(t, tempDir, "service.ail", serviceSrc)

	// Entry module imports the sibling — this is the path that exercised
	// the bug. We don't need to run main(), just type-check.
	src := Source{Filename: "service.ail"}
	cfg := Config{Mode: ModeCheck, RelaxModules: true} // mirrors `ailang check --package .`

	if _, err := Run(cfg, src); err != nil {
		t.Fatalf("pkg/<self>/sibling import must resolve without ailang.lock; got: %v", err)
	}
}

func TestIntraPackageImports_RelativeFormWorks(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "ailang-intra-pkg-rel-test")
	if err != nil {
		t.Fatalf("mkdir temp: %v", err)
	}
	defer os.RemoveAll(tempDir)

	originalDir, _ := os.Getwd()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer func() { _ = os.Chdir(originalDir) }()

	manifest := `[package]
name = "sunholo/intra_pkg_rel"
version = "0.1.0"
edition = "1"

[exports]
modules = ["sunholo/intra_pkg_rel/types", "sunholo/intra_pkg_rel/service"]
`
	typesSrc := `module sunholo/intra_pkg_rel/types

export pure func name() -> string = "Bar"
`
	serviceSrc := `module sunholo/intra_pkg_rel/service

import ./types (name)

export pure func greet() -> string = "Hello ${name()}"
`
	mustWrite(t, tempDir, "ailang.toml", manifest)
	mustWrite(t, tempDir, "types.ail", typesSrc)
	mustWrite(t, tempDir, "service.ail", serviceSrc)

	src := Source{Filename: "service.ail"}
	cfg := Config{Mode: ModeCheck, RelaxModules: true} // mirrors `ailang check --package .`

	if _, err := Run(cfg, src); err != nil {
		t.Fatalf("./sibling import (M-DX-RELIMPORT form) must resolve without ailang.lock; got: %v", err)
	}
}

// TestIntraPackageImports_BareCanonicalForm covers the LinkedIn user's actual
// form: `import sunholo/linkedin/types` — no pkg/, no ./. Until M3 routed
// this through the self-reference path, it fell through to project-relative
// resolution and surfaced as LDR001 even though types.ail was a sibling.
func TestIntraPackageImports_BareCanonicalForm(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "ailang-intra-pkg-bare-test")
	if err != nil {
		t.Fatalf("mkdir temp: %v", err)
	}
	defer os.RemoveAll(tempDir)

	originalDir, _ := os.Getwd()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer func() { _ = os.Chdir(originalDir) }()

	manifest := `[package]
name = "sunholo/intra_pkg_bare"
version = "0.1.0"
edition = "1"

[exports]
modules = ["sunholo/intra_pkg_bare/types", "sunholo/intra_pkg_bare/service"]
`
	typesSrc := `module sunholo/intra_pkg_bare/types

export pure func name() -> string = "Baz"
`
	serviceSrc := `module sunholo/intra_pkg_bare/service

import sunholo/intra_pkg_bare/types (name)

export pure func greet() -> string = "Hello ${name()}"
`
	mustWrite(t, tempDir, "ailang.toml", manifest)
	mustWrite(t, tempDir, "types.ail", typesSrc)
	mustWrite(t, tempDir, "service.ail", serviceSrc)

	src := Source{Filename: "service.ail"}
	cfg := Config{Mode: ModeCheck, RelaxModules: true}

	if _, err := Run(cfg, src); err != nil {
		t.Fatalf("bare-canonical self-import (LinkedIn's form) must resolve, got: %v", err)
	}
}

func TestIntraPackageImports_ExternalImportStillRequiresLock(t *testing.T) {
	// Regression guard: the self-only fallback must NOT silently swallow
	// missing-lock errors for genuinely external dependencies. Authors should
	// still see a clear "run ailang lock" message.
	tempDir, err := os.MkdirTemp("", "ailang-intra-pkg-extern-test")
	if err != nil {
		t.Fatalf("mkdir temp: %v", err)
	}
	defer os.RemoveAll(tempDir)

	originalDir, _ := os.Getwd()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer func() { _ = os.Chdir(originalDir) }()

	manifest := `[package]
name = "sunholo/intra_pkg_extern"
version = "0.1.0"
edition = "1"
`
	serviceSrc := `module sunholo/intra_pkg_extern/service

import pkg/sunholo/firestore/client (getDoc)

export pure func use() -> string = "stub"
`
	mustWrite(t, tempDir, "ailang.toml", manifest)
	mustWrite(t, tempDir, "service.ail", serviceSrc)

	src := Source{Filename: "service.ail"}
	cfg := Config{Mode: ModeCheck, RelaxModules: true} // mirrors `ailang check --package .`

	_, err = Run(cfg, src)
	if err == nil {
		t.Fatal("external pkg/<other>/... import must error when no lock file exists")
	}
	if !strings.Contains(err.Error(), "ailang.lock") {
		t.Errorf("error %q should mention ailang.lock to tell the author what to run", err.Error())
	}
}

func mustWrite(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}
