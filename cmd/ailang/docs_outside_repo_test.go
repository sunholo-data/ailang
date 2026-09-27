package main

import (
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/stdlibroot"
	"github.com/sunholo-data/ailang/internal/testutil"
)

// M-STDLIB-ROOT-RESOLUTION: `ailang docs` failed with "stdlib directory not
// found" from any directory without a ./std — including `docs prelude`, which
// reads no stdlib file at all. Daneel's agents are told to run
// `ailang docs std/<module>` first, and they work in scratch directories.
// Every test here runs from an empty temp dir with AILANG_STDLIB_PATH cleared.

func outsideRepoDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Chdir(dir)
	t.Setenv("AILANG_STDLIB_PATH", "")
	testutil.SetHomeDir(t, filepath.Join(dir, "home"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(dir, "xdg"))
	t.Setenv("APPDATA", filepath.Join(dir, "appdata"))
	stdlibroot.Configure(stdlibroot.Options{})
	t.Cleanup(func() { stdlibroot.Configure(stdlibroot.Options{}) })
	return dir
}

// runDocs drives docsCommand in-process exactly as main dispatches it
// (flag.Args() == ["docs", args...]) and returns what it printed.
func runDocs(t *testing.T, args ...string) string {
	t.Helper()
	orig := flag.CommandLine
	t.Cleanup(func() { flag.CommandLine = orig })
	flag.CommandLine = flag.NewFlagSet("ailang", flag.ContinueOnError)
	if err := flag.CommandLine.Parse(append([]string{"docs"}, args...)); err != nil {
		t.Fatal(err)
	}
	return captureStdout(t, docsCommand)
}

func TestDocsOutsideRepo_Module(t *testing.T) {
	outsideRepoDir(t)
	out := runDocs(t, "std/stream")
	for _, want := range []string{"# std/stream", "## Exports", "connect(", "transmit("} {
		if !strings.Contains(out, want) {
			t.Errorf("docs std/stream from a temp dir: missing %q in:\n%s", want, out)
		}
	}
}

func TestDocsOutsideRepo_List(t *testing.T) {
	outsideRepoDir(t)
	out := runDocs(t, "--list")
	for _, want := range []string{"std/io", "std/stream", "std/clock"} {
		if !strings.Contains(out, want) {
			t.Errorf("docs --list from a temp dir: missing %q", want)
		}
	}
}

func TestDocsOutsideRepo_AllFunctions(t *testing.T) {
	outsideRepoDir(t)
	out := runDocs(t, "--all-functions")
	for _, want := range []string{"std/clock.now:", "std/list.length:", "prelude.println:"} {
		if !strings.Contains(out, want) {
			t.Errorf("docs --all-functions from a temp dir: missing %q", want)
		}
	}
}

// `docs prelude` must not need a stdlib root at all: it still works when an
// explicit AILANG_STDLIB_PATH names no stdlib (which fails every other docs view).
func TestDocsOutsideRepo_PreludeNeedsNoStdlib(t *testing.T) {
	dir := outsideRepoDir(t)
	t.Setenv("AILANG_STDLIB_PATH", filepath.Join(dir, "not-a-stdlib"))
	out := runDocs(t, "prelude")
	if !strings.Contains(out, "println") {
		t.Errorf("docs prelude with a bogus AILANG_STDLIB_PATH: expected the prelude page, got:\n%s", out)
	}
}

// The ticket's acceptance criterion against the real binary:
// `cd $(mktemp -d) && ailang docs std/stream` prints the module's exports.
func TestDocsOutsideRepo_Binary(t *testing.T) {
	bin := testutil.FindAilangBinary(t)
	dir := t.TempDir()
	cmd := exec.Command(bin, "docs", "std/stream")
	cmd.Dir = dir
	cmd.Env = append(envWithout(os.Environ(), "AILANG_STDLIB_PATH"), "HOME="+dir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("ailang docs std/stream from %s: %v\n%s", dir, err, out)
	}
	if !strings.Contains(string(out), "# std/stream") || !strings.Contains(string(out), "connect(") {
		t.Fatalf("expected std/stream exports, got:\n%s", out)
	}
}

func envWithout(env []string, key string) []string {
	out := make([]string, 0, len(env))
	for _, kv := range env {
		if !strings.HasPrefix(kv, key+"=") {
			out = append(out, kv)
		}
	}
	return out
}
