package loader

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/stdlibroot"
	"github.com/sunholo-data/ailang/internal/testutil"
	"github.com/sunholo-data/ailang/std"
)

// M-STDLIB-ROOT-RESOLUTION: the loader reads every std module from the ONE root
// internal/stdlibroot picks. These tests run from an empty temp dir with
// AILANG_STDLIB_PATH cleared, so none of them depends on the repo being the cwd.

// realStdDir is the repo's std/, computed before any test changes directory.
var realStdDir = func() string {
	abs, err := filepath.Abs(filepath.Join("..", "..", "std"))
	if err != nil {
		panic(err)
	}
	return abs
}()

func isolateStdlib(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Chdir(dir)
	t.Setenv("AILANG_STDLIB_PATH", "")
	testutil.SetHomeDir(t, filepath.Join(dir, "home"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(dir, "xdg"))
	t.Setenv("APPDATA", filepath.Join(dir, "appdata"))
	stdlibroot.Configure(stdlibroot.Options{})
	t.Cleanup(func() { stdlibroot.Configure(stdlibroot.Options{}) })
	// Compare against the cwd as the process sees it (macOS /var vs /private/var,
	// Windows short names), not the string t.TempDir returned.
	if wd, err := os.Getwd(); err == nil {
		dir = wd
	}
	return dir
}

func writeStdFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// Outside any repo the embedded copy is the root, and the loaded source is its
// exact bytes under the synthetic <embedded> path.
func TestLoad_EmbeddedRootOutsideRepo(t *testing.T) {
	isolateStdlib(t)
	want, err := std.FS.ReadFile("option.ail")
	if err != nil {
		t.Fatal(err)
	}
	ml := NewModuleLoader(t.TempDir())
	loaded, err := ml.Load("std/option")
	if err != nil {
		t.Fatalf("Load(std/option) from an empty dir: %v", err)
	}
	if filepath.ToSlash(loaded.File.Path) != "<embedded>/std/option.ail" {
		t.Fatalf("path = %q, want the embedded copy", loaded.File.Path)
	}
	if loaded.SourceContent == nil || *loaded.SourceContent != string(want) {
		t.Fatal("embedded source snapshot differs from std.FS bytes")
	}
	again, err := ml.Load("std/option")
	if err != nil || again != loaded {
		t.Fatalf("second Load not served from cache: %v", err)
	}
	if _, err := ml.Load("std/nonexistent_module_xyz"); err == nil {
		t.Fatal("a module the embedded root lacks must be an error")
	}
}

// The Audit's V5 scenario: a shadow ./std/list.ail used to beat --stdlib-path.
func TestLoad_StdlibPathBeatsShadowCwdStd(t *testing.T) {
	dir := isolateStdlib(t)
	shadow := filepath.Join(dir, "std")
	writeStdFile(t, shadow, "io.ail", "module std/io\n")
	writeStdFile(t, shadow, "list.ail", "module std/list\nexport pure func map(f: int -> int, xs: [int]) -> [int] = [999]\n")

	// Without an override the shadow ./std is the root (development layout).
	ml := NewModuleLoader(dir)
	got, err := ml.resolvePath("std/list")
	if err != nil || got != filepath.Join(shadow, "list.ail") {
		t.Fatalf("no override: resolvePath = (%q, %v), want the cwd std", got, err)
	}

	// With --stdlib-path (as the runner configures it) the real stdlib wins.
	stdlibroot.Configure(stdlibroot.Options{Override: realStdDir})
	ml = NewModuleLoader(dir)
	loaded, err := ml.Load("std/list")
	if err != nil {
		t.Fatalf("Load(std/list) with --stdlib-path: %v", err)
	}
	if loaded.File.Path != filepath.Join(realStdDir, "list.ail") {
		t.Fatalf("std/list came from %q, want the --stdlib-path root %q", loaded.File.Path, realStdDir)
	}
	if strings.Contains(*loaded.SourceContent, "[999]") {
		t.Fatal("the shadow map was loaded despite --stdlib-path")
	}
}

// One root per run: a partial ./std (has io.ail, lacks list.ail) is an error that
// names the root, never a silent per-module fallback to another stdlib.
func TestLoad_PartialCwdStdErrorsInsteadOfMixing(t *testing.T) {
	dir := isolateStdlib(t)
	partial := filepath.Join(dir, "std")
	writeStdFile(t, partial, "io.ail", "module std/io\n")

	ml := NewModuleLoader(dir)
	loaded, err := ml.Load("std/list")
	if err == nil {
		t.Fatalf("Load(std/list) from a partial ./std succeeded from %q; want an error", loaded.File.Path)
	}
	msg := err.Error()
	if !strings.Contains(msg, "stdlib module not found: std/list") || !strings.Contains(msg, partial) {
		t.Fatalf("error must name the module and the chosen root %s, got:\n%s", partial, msg)
	}
	if _, err := ml.resolvePath("std/list"); err == nil {
		t.Fatal("resolvePath must agree with Load: no per-module embedded fallback")
	}
}

// An explicit AILANG_STDLIB_PATH that is not a stdlib is a loud error.
func TestLoad_BogusEnvStdlibPathIsAnError(t *testing.T) {
	dir := isolateStdlib(t)
	t.Setenv("AILANG_STDLIB_PATH", dir) // a project root, not <root>/std
	_, err := NewModuleLoader(dir).Load("std/io")
	var nse *stdlibroot.NotStdlibError
	if !errors.As(err, &nse) {
		t.Fatalf("err = %v, want a NotStdlibError", err)
	}
}

// --strict turns an on-disk VERSION mismatch into an error.
func TestLoad_StrictVersionMismatchIsFatal(t *testing.T) {
	dir := isolateStdlib(t)
	root := filepath.Join(dir, "std")
	writeStdFile(t, root, "io.ail", "module std/io\n")
	writeStdFile(t, root, "VERSION", "v0.0.1\n")

	old := BinaryVersion
	BinaryVersion = "v9.9.9"
	t.Cleanup(func() { BinaryVersion = old })

	if _, err := NewStdlibResolver("", false, false).ResolveStdlib("io"); err != nil {
		t.Fatalf("non-strict mismatch must only warn, got %v", err)
	}
	stdlibroot.Configure(stdlibroot.Options{StrictVersion: true})
	_, err := NewStdlibResolver("", false, false).ResolveStdlib("io")
	if err == nil || !strings.Contains(err.Error(), "version mismatch") {
		t.Fatalf("strict mismatch: err = %v, want a version mismatch error", err)
	}
}
