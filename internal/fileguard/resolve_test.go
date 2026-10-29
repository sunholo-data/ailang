package fileguard

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// AILANG World 2026-10-17: `**` is zero or more directories, not `*`.
func TestMatchDenyWrite_Doublestar(t *testing.T) {
	deny := []string{"**/*.test.ts", "tests/**", "docs/**/gen/*.md", "**/fixtures/**"}
	protected := map[string]string{
		"a.test.ts":               "**/*.test.ts",
		"src/foo.test.ts":         "**/*.test.ts",
		"src/deep/x.test.ts":      "**/*.test.ts",
		"tests":                   "tests/**",
		"tests/x.py":              "tests/**",
		"tests/a/b/c.py":          "tests/**",
		"docs/gen/a.md":           "docs/**/gen/*.md",
		"docs/x/y/gen/a.md":       "docs/**/gen/*.md",
		"fixtures/x":              "**/fixtures/**",
		"pkg/a/fixtures/b/c.json": "**/fixtures/**",
	}
	for rel, want := range protected {
		if got := MatchDenyWrite(deny, rel); got != want {
			t.Errorf("MatchDenyWrite(%q) = %q, want %q", rel, got, want)
		}
	}
	// A slash anchors the pattern at the root: tests/** is not cli/tests.
	for _, rel := range []string{"cli/tests/x.py", "src/foo.test.tsx", "docs/gen/a.txt", "docs/gen/sub/a.md", "testsuite/x"} {
		if got := MatchDenyWrite(deny, rel); got != "" {
			t.Errorf("MatchDenyWrite(%q) = %q, want no match", rel, got)
		}
	}
	if got := MatchDenyWrite([]string{"src/[bad"}, "src/x"); got == "" {
		t.Error("a malformed glob must fail closed")
	}
}

type resolveFixture struct {
	dir  string
	root *Root
}

func newResolveFixture(t *testing.T) resolveFixture {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("symlinks need a privilege on Windows")
	}
	dir := t.TempDir()
	if real, err := filepath.EvalSymlinks(dir); err == nil {
		dir = real
	}
	for _, d := range []string{"tests", "pkg", "src/inner"} {
		if err := os.MkdirAll(filepath.Join(dir, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for _, f := range []string{"tests/test_x.py", "pkg/a_test.go", "src/ok.txt"} {
		if err := os.WriteFile(filepath.Join(dir, f), []byte("orig"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	links := map[string]string{
		"lnk":           "tests",
		"alias.txt":     "pkg/a_test.go",
		"dangling.py":   "tests/new.py",
		"src/inner/up":  "../../tests",
		"chain":         "lnk",
		"src/okalias":   "ok.txt",
		"abs":           filepath.Join(dir, "tests"),
		"loop":          "loop",
		"escape":        "../outside",
		"gitlink":       ".git",
		"src/dirlink":   "../pkg",
		"src/self":      ".",
		"tests_via_dot": "./tests/../tests",
	}
	for name, target := range links {
		if err := os.Symlink(target, filepath.Join(dir, name)); err != nil {
			t.Fatal(err)
		}
	}
	r, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = r.Close() })
	return resolveFixture{dir: dir, root: r}
}

func TestRoot_Resolve(t *testing.T) {
	f := newResolveFixture(t)
	for in, want := range map[string]string{
		"lnk/new5.py":           "tests/new5.py",
		"lnk/test_x.py":         "tests/test_x.py",
		"alias.txt":             "pkg/a_test.go",
		"dangling.py":           "tests/new.py",
		"src/inner/up/z.py":     "tests/z.py",
		"chain/a":               "tests/a",
		"src/okalias":           "src/ok.txt",
		"missing/dir/f":         "missing/dir/f",
		"src/self/self/ok.txt":  "src/ok.txt",
		"tests_via_dot/q":       "tests/q",
		"src/dirlink/a_test.go": "pkg/a_test.go",
		".":                     ".",
	} {
		if got, err := f.root.Resolve(in, true); err != nil || got != want {
			t.Errorf("Resolve(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	// Not following the final component: the link entry itself.
	if got, err := f.root.Resolve("lnk", false); err != nil || got != "lnk" {
		t.Errorf("Resolve(lnk, final=false) = %q, %v", got, err)
	}
	if got, err := f.root.Resolve("src/inner/up/x", false); err != nil || got != "tests/x" {
		t.Errorf("parents are still followed: %q, %v", got, err)
	}
	for _, bad := range []string{"abs/x", "escape", "../x", "loop"} {
		if got, err := f.root.Resolve(bad, true); err == nil {
			t.Errorf("Resolve(%q) = %q, want an error", bad, got)
		}
	}
}

func TestRoot_CheckPathFollowsSymlinks(t *testing.T) {
	f := newResolveFixture(t)
	p := Protection{GitDir: true, DenyWrite: []string{"tests/**", "*_test.go"}}
	for _, name := range []string{"lnk/new5.py", "lnk/test_x.py", "alias.txt", "dangling.py", "src/inner/up/z.py", "chain/q", "src/dirlink/a_test.go", "gitlink/config"} {
		if v := f.root.CheckPath(p, name, false); !v.Protected() {
			t.Errorf("CheckPath(%q) = %+v, want protected", name, v)
		}
	}
	if v := f.root.CheckPath(p, "lnk/new5.py", false); v.Via != "tests/new5.py" || v.Pattern != "tests/**" {
		t.Errorf("the refusal names the resolved path: %+v", v)
	}
	for _, name := range []string{"src/ok.txt", "src/okalias", "src/new.txt", "missing/new.txt"} {
		if v := f.root.CheckPath(p, name, false); v.Protected() {
			t.Errorf("CheckPath(%q) = %+v, want writable", name, v)
		}
	}
	// Removing or renaming a link entry touches the link, not its target.
	if v := f.root.CheckPath(p, "alias.txt", true); v.Protected() {
		t.Errorf("removing the alias entry is not a write to its target: %+v", v)
	}
	if v := f.root.CheckPath(p, "lnk/test_x.py", true); !v.Protected() {
		t.Errorf("removing through a linked parent reaches the protected file: %+v", v)
	}
	// The spelled check still runs first and is unchanged.
	if v := f.root.CheckPath(p, "tests/new.py", false); !v.Protected() || v.Via != "" {
		t.Errorf("direct: %+v", v)
	}
}

// Under a leading-`**` pattern every directory may hold a protected file,
// so directories stay unmovable; a plain file holds nothing and is judged
// by its own name.
func TestRoot_CheckPathMoveFileVsDir(t *testing.T) {
	f := newResolveFixture(t)
	p := Protection{GitDir: true, DenyWrite: []string{"**/*.test.ts"}}
	if v := f.root.CheckPath(p, "src/ok.txt", true); v.Protected() {
		t.Errorf("removing a plain file: %+v", v)
	}
	if v := f.root.CheckPath(p, "src", true); !v.Protected() || !v.Ancestor {
		t.Errorf("a directory may hold a protected file: %+v", v)
	}
	if v := f.root.CheckPath(p, "src/new-dest", true); !v.Protected() {
		t.Errorf("a rename destination that does not exist yet stays conservative: %+v", v)
	}
	if err := os.WriteFile(filepath.Join(f.dir, "src", "a.test.ts"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if v := f.root.CheckPath(p, "src/a.test.ts", true); !v.Protected() {
		t.Errorf("a protected file is still protected: %+v", v)
	}
}

func TestRoot_CheckPathHardLink(t *testing.T) {
	f := newResolveFixture(t)
	if err := os.Link(filepath.Join(f.dir, "pkg", "a_test.go"), filepath.Join(f.dir, "src", "hard.txt")); err != nil {
		t.Skipf("hard links unavailable: %v", err)
	}
	p := Protection{DenyWrite: []string{"*_test.go"}}
	if v := f.root.CheckPath(p, "src/hard.txt", false); !v.HardLink || !v.Protected() {
		t.Errorf("a hard-linked file must be refused: %+v", v)
	}
	if v := f.root.CheckPath(p, "src/hard.txt", true); v.Protected() {
		t.Errorf("removing one name of a hard link leaves the other intact: %+v", v)
	}
	if v := f.root.CheckPath(Protection{}, "src/hard.txt", false); v.Protected() {
		t.Errorf("with nothing protected, nothing is refused: %+v", v)
	}
}
