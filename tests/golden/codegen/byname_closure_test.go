package codegen_golden_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/testutil"
)

// writeFiles writes files (path -> source) under a fresh directory.
func writeFiles(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for p, src := range files {
		full := filepath.Join(dir, filepath.FromSlash(p))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// runEveryEngine runs entry's summary on the evaluator, --bytecode and
// --strict-bytecode and requires want from each (no fallback).
func runEveryEngine(t *testing.T, dir, entry, want string) {
	t.Helper()
	for _, flags := range [][]string{nil, {"--bytecode"}, {"--strict-bytecode"}} {
		args := append(append([]string{"run", "--quiet"}, flags...), "--entry", "summary", entry)
		stdout, stderr, code := testutil.RunBounded(t, dir, 90*time.Second, ailangBin, args...)
		if got := strings.TrimSpace(stdout); code != 0 || got != want || strings.Contains(stderr, "falling back") {
			t.Errorf("run %v: exit %d, got %q, want %q\n%s", flags, code, got, want, stderr)
		}
	}
}

// Review shape hop3 (#2380 follow-up): a by-name match on a value whose type
// is three imports away — probe → lib/w (imports lib/m only as a namespace)
// → lib/m (`type Sh = Shape`) → lib/s. The import closure the candidates come
// from is transitive; a direct-imports-only closure rejected this program.
func TestByNameMatchThreeImportsAwayRunsOnEveryEngine(t *testing.T) {
	t.Setenv("AILANG_CACHE_DIR", t.TempDir())
	dir := writeFiles(t, map[string]string{
		"lib/s.ail": "module lib/s\nexport type Shape = Circle(int) | Rect(int, int) | Dot\n" +
			"export pure func mks(i: int) -> Shape = if i == 0 then Circle(5) else if i == 1 then Rect(2, 3) else Dot\n",
		"lib/m.ail": "module lib/m\nimport lib/s (Shape, mks)\nexport type Sh = Shape\nexport pure func mm(i: int) -> Sh = mks(i)\n",
		"lib/w.ail": "module lib/w\nimport lib/m as M\nexport pure func ww(i: int) = M.mm(i)\n",
		"probe.ail": "module probe\nimport lib/w (ww)\npure func f(i: int) -> int = match ww(i) { Circle(r) => r, Rect(a, b) => a * b, _ => -1 }\n" +
			"export pure func summary() -> string = show([f(0), f(1), f(2)])\n",
	})
	runEveryEngine(t, dir, "probe.ail", "[5, 6, -1]")
}

// #2371's accepted→rejected case, fixed by the import its error names: a
// library matching its own (unimported) Status.Ok / Maybe.None by name now
// gets the prelude's Ok / None, and `import lib/st (Status, Err, Ok,
// Pending)` / `import lib/st (Maybe, Just, None)` restores the old answers
// on every engine.
func TestPreludeShadowedCtorFixedByTheHintedImport(t *testing.T) {
	t.Setenv("AILANG_CACHE_DIR", t.TempDir())
	libs := map[string]string{
		"lib/st.ail": "module lib/st\nexport type Status = Ok | Err(string) | Pending\nexport type Maybe = Just(int) | None\n" +
			"export pure func stat(i: int) -> Status = if i == 0 then Ok else if i == 1 then Err(\"bad\") else Pending\n" +
			"export pure func mb(i: int) -> Maybe = if i > 0 then Just(i) else None\n",
		"lib/w.ail": "module lib/w\nimport lib/st (stat, mb)\nexport pure func ws(i: int) = stat(i)\nexport pure func wm(i: int) = mb(i)\n",
	}
	for _, c := range []struct{ name, fix, body, want string }{
		{"status", "import lib/st (Status, Err, Ok, Pending)", "match ws(i) { Ok => 1, Err(_) => 2, _ => 3 }", "[1, 2, 3]"},
		{"maybe", "import lib/st (Maybe, Just, None)", "match wm(i) { Just(v) => v, None => 0 }", "[0, 1, 2]"},
	} {
		t.Run(c.name, func(t *testing.T) {
			files := map[string]string{"probe.ail": "module probe\nimport lib/w (ws, wm)\n" + c.fix + "\npure func f(i: int) -> int = " + c.body +
				"\nexport pure func summary() -> string = show([f(0), f(1), f(2)])\n"}
			for p, s := range libs {
				files[p] = s
			}
			runEveryEngine(t, writeFiles(t, files), "probe.ail", c.want)
		})
	}
}
