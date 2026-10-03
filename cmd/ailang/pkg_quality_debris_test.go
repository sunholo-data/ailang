package main

// #1502: a package that already carries a `_namedtest_body_<n>.ail` from an
// interrupted `ailang test` of an older binary is refused by `pkg quality`
// (PUB024, exit 2), its compile count is the clean tree's, and `ailang check`
// on the directory skips the file. `pkg init` scaffolds the .gitignore entry.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/pkg"
	"github.com/sunholo-data/ailang/internal/testutil"
)

// copyFlatFixture copies the flat quality fixture's files into a
// temp dir the test may dirty.
func copyFlatFixture(t *testing.T) string {
	t.Helper()
	src := fixtureDir(t, "flat_self_import")
	dst := t.TempDir()
	entries, err := os.ReadDir(src)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.IsDir() { // e.g. a .ailang/ cache other tests leave behind
			continue
		}
		data, err := os.ReadFile(filepath.Join(src, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dst, e.Name()), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dst
}

func TestPkgQuality_PUB024RefusesStaleNamedTestBody(t *testing.T) {
	bin := buildAilang(t)
	dir := copyFlatFixture(t)

	quality := func() (pkg.QualityReport, int, string) {
		t.Helper()
		out, stderr, exit := runAilangBin(t, bin, "pkg", "quality", "--json", "--no-run", dir)
		var r pkg.QualityReport
		if err := json.Unmarshal([]byte(out), &r); err != nil {
			t.Fatalf("decode: %v\nstdout:\n%s\nstderr:\n%s", err, out, stderr)
		}
		return r, exit, out
	}
	clean, _, _ := quality()

	settle, err := os.ReadFile(filepath.Join(dir, "settle.ail"))
	if err != nil {
		t.Fatal(err)
	}
	const debris = "_namedtest_body_3176816492.ail"
	if err := os.WriteFile(filepath.Join(dir, debris), settle, 0o600); err != nil {
		t.Fatal(err)
	}

	dirty, exit, out := quality()
	if exit != 2 {
		t.Errorf("pkg quality exit = %d, want 2 (gate)\n%s", exit, out)
	}
	var msg string
	for _, g := range dirty.Gates {
		if g.Code == "PUB024" {
			msg = g.Msg
		}
	}
	if !strings.Contains(msg, debris) || !strings.Contains(msg, "delete") {
		t.Errorf("want a PUB024 gate naming %s and the fix, got gates %v", debris, dirty.Gates)
	}
	if dirty.Compile.Files != clean.Compile.Files {
		t.Errorf("compile counted %d files with the stale body, %d without", dirty.Compile.Files, clean.Compile.Files)
	}

	// `ailang check <dir>` skips it and says so.
	_, stderr, _ := runAilangBin(t, bin, "check", "--relax-modules", dir)
	if !strings.Contains(stderr, debris) {
		t.Errorf("check <dir> did not name the skipped stale body:\n%s", stderr)
	}
}

func TestPkgInit_ScaffoldsNamedTestBodyGitignore(t *testing.T) {
	bin := buildAilang(t)
	dir := t.TempDir()
	out, stderr, exit := testutil.RunBounded(t, dir, 30*time.Second, bin, "init", "package", "--name", "test/scaffold")
	if exit != 0 {
		t.Fatalf("init package exit %d\n%s\n%s", exit, out, stderr)
	}
	data, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil {
		t.Fatalf("no .gitignore scaffolded: %v", err)
	}
	if !strings.Contains(string(data), pkg.NamedTestBodyGitignore) {
		t.Errorf(".gitignore lacks %q:\n%s", pkg.NamedTestBodyGitignore, data)
	}
}
