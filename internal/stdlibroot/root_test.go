package stdlibroot

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/testutil"
)

// isolate runs the test from an empty temp dir with no override, no env root and
// a home without an installed stdlib, so only what the test creates can win.
func isolate(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Chdir(dir)
	t.Setenv("AILANG_STDLIB_PATH", "")
	testutil.SetHomeDir(t, filepath.Join(dir, "home"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(dir, "xdg"))
	t.Setenv("APPDATA", filepath.Join(dir, "appdata"))
	Configure(Options{})
	t.Cleanup(func() { Configure(Options{}) })
	// Compare against the cwd as the process sees it (macOS /var vs /private/var,
	// Windows short names), not the string t.TempDir returned.
	if wd, err := os.Getwd(); err == nil {
		dir = wd
	}
	return dir
}

func makeStd(t *testing.T, dir string, files ...string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		if err := os.WriteFile(filepath.Join(dir, f), []byte("module std/"+strings.TrimSuffix(f, ".ail")+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	abs, _ := filepath.Abs(dir)
	return abs
}

func TestResolve_EmptyDirIsEmbedded(t *testing.T) {
	isolate(t)
	r, err := Resolve("")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if r.Source != "embedded" || !r.Embedded() {
		t.Fatalf("Source = %q Dir = %q, want embedded", r.Source, r.Dir)
	}
	if _, err := fs.ReadFile(r.FS, "stream.ail"); err != nil {
		t.Fatalf("embedded root cannot read stream.ail: %v", err)
	}
	if got := r.DisplayPath("stream.ail"); got != "<embedded>/std/stream.ail" {
		t.Fatalf("DisplayPath = %q", got)
	}
}

func TestResolve_Order(t *testing.T) {
	tests := []struct {
		name       string
		cwdStd     bool
		env        bool
		flag       bool
		wantSource string
	}{
		{"cwd std wins over embedded", true, false, false, "cwd"},
		{"env beats cwd std", true, true, false, "env"},
		{"flag beats cwd std", true, false, true, "flag"},
		{"flag beats env", true, true, true, "flag"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := isolate(t)
			want := map[string]string{}
			if tt.cwdStd {
				want["cwd"] = makeStd(t, filepath.Join(dir, "std"), "io.ail")
			}
			if tt.env {
				want["env"] = makeStd(t, filepath.Join(dir, "envstd"), "io.ail")
				t.Setenv("AILANG_STDLIB_PATH", filepath.Join(dir, "missing")+string(os.PathListSeparator)+want["env"])
			}
			override := ""
			if tt.flag {
				want["flag"] = makeStd(t, filepath.Join(dir, "flagstd"), "io.ail")
				override = want["flag"]
			}
			r, err := Resolve(override)
			if err != nil {
				t.Fatalf("Resolve: %v", err)
			}
			if r.Source != tt.wantSource || r.Dir != want[tt.wantSource] {
				t.Fatalf("got %s %q, want %s %q", r.Source, r.Dir, tt.wantSource, want[tt.wantSource])
			}
		})
	}
}

func TestResolve_ConfiguredOverrideWins(t *testing.T) {
	dir := isolate(t)
	makeStd(t, filepath.Join(dir, "std"), "io.ail")
	flag := makeStd(t, filepath.Join(dir, "flagstd"), "io.ail")
	Configure(Options{Override: flag})
	r, err := Resolve("")
	if err != nil || r.Source != "flag" || r.Dir != flag {
		t.Fatalf("Resolve = %+v, %v; want the configured override", r, err)
	}
}

func TestResolve_StdWithoutMarkerIsSkipped(t *testing.T) {
	dir := isolate(t)
	makeStd(t, filepath.Join(dir, "std"), "list.ail") // a project std/, not the stdlib
	r, err := Resolve("")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if r.Source != "embedded" {
		t.Fatalf("Source = %q, want embedded (a std/ without io.ail is not a stdlib)", r.Source)
	}
	if len(r.Tried) == 0 || r.Tried[0] != filepath.Join(dir, "std") {
		t.Fatalf("Tried = %v, want ./std first", r.Tried)
	}
}

func TestResolve_BogusFlagIsAnError(t *testing.T) {
	dir := isolate(t)
	makeStd(t, filepath.Join(dir, "std"), "io.ail") // must NOT be fallen back to
	bogus := filepath.Join(dir, "nope")
	_, err := Resolve(bogus)
	var nse *NotStdlibError
	if !errors.As(err, &nse) {
		t.Fatalf("err = %v, want *NotStdlibError", err)
	}
	if nse.What != "--stdlib-path" || len(nse.Tried) != 1 || nse.Tried[0] != bogus {
		t.Fatalf("error = %+v", nse)
	}
	if !strings.Contains(err.Error(), bogus) {
		t.Fatalf("error text does not name the tried path: %v", err)
	}
}

func TestResolve_BogusEnvIsAnError(t *testing.T) {
	dir := isolate(t)
	t.Setenv("AILANG_STDLIB_PATH", dir) // a project root, not <root>/std
	_, err := Resolve("")
	var nse *NotStdlibError
	if !errors.As(err, &nse) || nse.What != "AILANG_STDLIB_PATH" {
		t.Fatalf("err = %v, want NotStdlibError for AILANG_STDLIB_PATH", err)
	}
}

func TestResolve_MemoisedPerInputs(t *testing.T) {
	dir := isolate(t)
	first, _ := Resolve("")
	// A std/ that appears after the first resolution does not change the root of
	// this process: one root per run.
	makeStd(t, filepath.Join(dir, "std"), "io.ail")
	second, _ := Resolve("")
	if first.Source != second.Source || second.Source != "embedded" {
		t.Fatalf("root changed within a process: %s then %s", first.Source, second.Source)
	}
}
