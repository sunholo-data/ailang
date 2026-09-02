package main

import (
	"strings"
	"testing"
)

// These tests exercise the SHIPPED CLI SURFACE through a real binary, not the
// library function behind it.
//
// The milestone's first round of tests called pipeline.BuildCanonicalJSON
// directly and hand-appended the trailing newline outputInterface was assumed
// to print. That verifies the arithmetic, never the artifact: the sprint
// evaluator confirmed by mutation that the entire `internal-dump-iface`
// dispatch arm could be deleted, that `iface --compact` could be neutered, and
// that the trailing newline could be dropped — each with zero test failures
// across the whole ./cmd/ailang/... suite.

// ifaceFixture is a module reachable both as <packageDir>/<modulePath>.ail and
// as a single file path, so the hidden subcommand and the public command can be
// pointed at the SAME module by their two different argument conventions.
type ifaceFixture struct {
	name       string
	packageDir string
	modulePath string
	filePath   string
}

var ifaceFixtures = []ifaceFixture{
	{
		// No imports: agrees under any packageDir, so it cannot discriminate.
		name:       "standalone_module",
		packageDir: "examples/docs",
		modulePath: "module_greet",
		filePath:   "examples/docs/module_greet.ail",
	},
	{
		// Imports a sibling, so its interface can only be built when the
		// ailang.toml search is anchored at the package root. This is the arm
		// that discriminates a correct packageDir from a wrong one (ailang#671).
		name:       "intra_package_imports",
		packageDir: "examples/intra_package_imports",
		modulePath: "service",
		filePath:   "examples/intra_package_imports/service.ail",
	},
}

// TestInternalDumpIface_ByteIdenticalToPublicIface is the milestone's M2-A5
// acceptance criterion as a PERSISTENT gate rather than a one-shot command.
//
// It runs both commands from the repository root — i.e. from outside the
// fixture's own package — which is the CWD that makes the intra-package arm
// meaningful.
//
// Kills: deleting the `case "internal-dump-iface":` dispatch arm (the
// subcommand then exits non-zero with "unknown command").
func TestInternalDumpIface_ByteIdenticalToPublicIface(t *testing.T) {
	bin := buildAilang(t)
	for _, f := range ifaceFixtures {
		t.Run(f.name, func(t *testing.T) {
			hidden, hiddenErr, hiddenExit := runAilangBin(t, bin, "internal-dump-iface", f.packageDir, f.modulePath)
			if hiddenExit != 0 {
				t.Fatalf("internal-dump-iface %s %s exited %d\nstderr:\n%s", f.packageDir, f.modulePath, hiddenExit, hiddenErr)
			}
			public, publicErr, publicExit := runAilangBin(t, bin, "iface", f.filePath)
			if publicExit != 0 {
				t.Fatalf("iface %s exited %d\nstderr:\n%s", f.filePath, publicExit, publicErr)
			}
			if hidden != public {
				t.Fatalf("hidden subcommand and public command disagree for %s\nhidden:\n%q\npublic:\n%q", f.name, hidden, public)
			}
			if strings.TrimSpace(hidden) == "" {
				t.Fatalf("expected non-empty interface JSON for %s", f.name)
			}
		})
	}
}

// TestInternalDumpIface_WrongPackageDirFailsLoudly pins the subcommand's
// argument contract: packageDir must be the module's PACKAGE ROOT, not merely
// some ancestor directory.
//
// This is documentation with teeth. The plan's M3 will invoke this subcommand
// from a subprocess wrapper; if it passes a directory that is not the package
// root (".", or a repo root), intra-package imports do not resolve. Asserting
// the failure here means M3 cannot quietly inherit ailang#671 — the behaviour
// is pinned rather than merely commented.
func TestInternalDumpIface_WrongPackageDirFailsLoudly(t *testing.T) {
	bin := buildAilang(t)
	_, stderr, exit := runAilangBin(t, bin, "internal-dump-iface", ".", "examples/intra_package_imports/service")
	if exit == 0 {
		t.Fatal("expected a non-zero exit when packageDir is not the module's package root")
	}
	if !strings.Contains(stderr, "ailang.toml") {
		t.Errorf("expected the error to name the missing manifest, got:\n%s", stderr)
	}
}

// TestIface_EmitsExactlyOneTrailingNewline pins outputInterface's stdout shape.
//
// Kills: replacing fmt.Println with fmt.Print in the non-compact branch.
func TestIface_EmitsExactlyOneTrailingNewline(t *testing.T) {
	bin := buildAilang(t)
	stdout, stderr, exit := runAilangBin(t, bin, "iface", "examples/docs/module_greet.ail")
	if exit != 0 {
		t.Fatalf("iface exited %d\nstderr:\n%s", exit, stderr)
	}
	if !strings.HasSuffix(stdout, "}\n") {
		t.Fatalf("expected stdout to end with exactly one newline after the closing brace, got %q", lastBytes(stdout, 12))
	}
	if strings.HasSuffix(stdout, "\n\n") {
		t.Fatalf("expected exactly one trailing newline, got %q", lastBytes(stdout, 12))
	}
}

// TestIface_CompactFlagRendersCompactView pins the --compact branch of
// outputInterface, which the library-level tests bypass entirely.
//
// Kills: neutering the `if compact { printCompactInterface(...); return }` arm,
// which otherwise falls through to the JSON view with no test noticing.
func TestIface_CompactFlagRendersCompactView(t *testing.T) {
	bin := buildAilang(t)
	compact, stderr, exit := runAilangBin(t, bin, "iface", "--compact", "examples/docs/module_greet.ail")
	if exit != 0 {
		t.Fatalf("iface --compact exited %d\nstderr:\n%s", exit, stderr)
	}
	full, _, fullExit := runAilangBin(t, bin, "iface", "examples/docs/module_greet.ail")
	if fullExit != 0 {
		t.Fatalf("iface exited %d", fullExit)
	}
	if compact == full {
		t.Fatal("--compact produced byte-identical output to the JSON view; the compact branch is not being taken")
	}
	// The compact view is signatures, not JSON: it must not open with a brace.
	if strings.HasPrefix(strings.TrimSpace(compact), "{") {
		t.Fatalf("--compact rendered JSON rather than a compact view:\n%s", compact)
	}
	if strings.TrimSpace(compact) == "" {
		t.Fatal("--compact produced no output")
	}
}

func lastBytes(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[len(s)-n:]
}
