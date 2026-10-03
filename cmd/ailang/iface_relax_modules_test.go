package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/loader"
	"github.com/sunholo-data/ailang/internal/pipeline"
	"github.com/sunholo-data/ailang/internal/testutil"
)

// mod010Fixture writes a module whose declaration does not match its path into
// a directory that is NOT a temp directory (MOD010 auto-relaxes there), and
// returns its absolute path. The compile cache is disabled so a relaxed run
// cannot serve a later strict one.
func mod010Fixture(t *testing.T) string {
	t.Helper()
	t.Setenv("AILANG_NO_CACHE", "1")
	dir, err := os.MkdirTemp(".", "iface-mod010-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	abs, err := filepath.Abs(dir)
	if err != nil {
		t.Fatal(err)
	}
	if loader.IsTempPath(abs) {
		t.Skipf("checkout lives in a temp directory (%s); MOD010 auto-relaxes there", abs)
	}
	file := filepath.Join(abs, "mismatch.ail")
	src := "module some/other_name\n\nexport pure func answer() -> int = 42\n"
	if err := os.WriteFile(file, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	return file
}

// #575: the MOD010 error `iface` prints says "use --relax-modules flag or set
// AILANG_RELAX_MODULES=1"; iface rejected the flag as undefined and ignored
// the variable.
func TestIface_RelaxModulesFlag(t *testing.T) {
	file := mod010Fixture(t)
	t.Setenv("AILANG_RELAX_MODULES", "")
	stdout, stderr, code := runCLI(t, "iface", "--relax-modules", file)
	if code != 0 || !strings.Contains(stdout, `"answer"`) {
		t.Fatalf("iface --relax-modules: exit %d, want 0 with the interface\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}
}

func TestIface_RelaxModulesEnv(t *testing.T) {
	file := mod010Fixture(t)
	t.Setenv("AILANG_RELAX_MODULES", "1")
	stdout, stderr, code := runCLI(t, "iface", file)
	if code != 0 || !strings.Contains(stdout, `"answer"`) {
		t.Fatalf("AILANG_RELAX_MODULES=1 iface: exit %d, want 0 with the interface\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}
}

func TestIface_StrictWithoutRelax(t *testing.T) {
	file := mod010Fixture(t)
	t.Setenv("AILANG_RELAX_MODULES", "")
	_, stderr, code := runCLI(t, "iface", file)
	if code == 0 || !strings.Contains(stderr, "MOD010") {
		t.Fatalf("iface without relax: exit %d, want MOD010 failure\nstderr: %s", code, stderr)
	}
}

// #574 part 3: an absolute path's canonical module ID has its leading "/"
// stripped, and the temp-path check resolved that against the cwd — so the
// same non-temp file auto-relaxed when iface (or check) ran from a temp
// directory and failed MOD010 from anywhere else. The outcome must not
// depend on the cwd.
func TestMOD010_AbsolutePathIndependentOfCwd(t *testing.T) {
	file := mod010Fixture(t)
	t.Setenv("AILANG_RELAX_MODULES", "")
	bin := buildAilang(t)
	for _, cmd := range []string{"iface", "check"} {
		_, rootErr, rootCode := runAilangBin(t, bin, cmd, file)
		_, tmpErr, tmpCode := testutil.RunBounded(t, os.TempDir(), 60*time.Second, bin, cmd, file)
		if rootCode == 0 || tmpCode == 0 {
			t.Errorf("%s %s: exit %d from the repo, %d from %s; want MOD010 failure from both\nrepo stderr: %s\ntemp stderr: %s",
				cmd, file, rootCode, tmpCode, os.TempDir(), rootErr, tmpErr)
		}
	}
}

// #1374: the JSON signatures of these stdlib modules carried Go type names
// ("<*types.TRecord2>", "<*types.TTuple>", ...) in place of the types.
func TestOutputInterface_StdlibNoGoTypeNames(t *testing.T) {
	t.Chdir(repoRoot(t))
	for _, mod := range []string{"std/ai", "std/array", "std/map", "std/sem", "std/secret", "std/list"} {
		got, err := pipeline.BuildCanonicalJSON(context.Background(), outputInterfacePackageDir, mod)
		if err != nil {
			t.Fatalf("%s: %v", mod, err)
		}
		if strings.Contains(string(got), "types.T") {
			t.Errorf("%s interface leaks a Go type name:\n%s", mod, got)
		}
	}
}
