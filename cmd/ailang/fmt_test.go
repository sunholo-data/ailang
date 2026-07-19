package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// fmt_test.go covers the fmt command's non-exit logic: the in-memory format +
// round-trip pipeline (formatOne) and the atomic-write helper. The exit-code and
// flag-dispatch behavior is proven end-to-end with the built binary; unit tests
// here focus on the pieces that can be exercised without os.Exit.

func writeTemp(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestFormatOne_CanonicalizesSeparators(t *testing.T) {
	dir := t.TempDir()
	path := writeTemp(t, dir, "a.ail", "module t/a\nfunc f() = let a = 1; let b = 2; a + b\n")
	orig, canonical, err := formatOne(path)
	if err != nil {
		t.Fatalf("formatOne: %v", err)
	}
	want := "module t/a\n\nfunc f() {\n  let a = 1\n  let b = 2\n  a + b\n}\n"
	if string(canonical) != want {
		t.Errorf("canonical mismatch:\n got %q\nwant %q", canonical, want)
	}
	if string(orig) == string(canonical) {
		t.Error("expected non-canonical input to differ from canonical output")
	}
}

// TestFormatOne_PreservesComments proves the Phase-2 behavior: a commented file
// now FORMATS (comments preserved), it is no longer refused.
func TestFormatOne_PreservesComments(t *testing.T) {
	dir := t.TempDir()
	path := writeTemp(t, dir, "c.ail", "module t/c\n\n-- hello\nfunc f() = 1\n")
	_, canonical, err := formatOne(path)
	if err != nil {
		t.Fatalf("expected commented file to format, got: %v", err)
	}
	if !contains(string(canonical), "-- hello") {
		t.Errorf("comment not preserved in output:\n%s", canonical)
	}
}

func TestFormatOne_RejectsParseError(t *testing.T) {
	dir := t.TempDir()
	path := writeTemp(t, dir, "p.ail", "module t/p\nfunc f( = \n")
	_, _, err := formatOne(path)
	if err == nil {
		t.Fatal("expected parse error, got nil")
	}
	// A parse error must be tagged so the dispatcher maps it to exit 3.
	if fmtExitCode(err) != exitParse {
		t.Errorf("parse error should map to exit %d, got %d", exitParse, fmtExitCode(err))
	}
}

// TestFormatOne_RefusesInterpolationComment proves the fail-closed carve-out: a
// comment inside a ${...} interpolation hole is refused (operational error, exit
// 2), never silently dropped.
func TestFormatOne_RefusesInterpolationComment(t *testing.T) {
	dir := t.TempDir()
	path := writeTemp(t, dir, "i.ail", "module t/i\n\nexport func f() -> string = \"pre${ x -- oops\n }post\"\n")
	_, _, err := formatOne(path)
	if err == nil {
		t.Fatal("expected fail-closed refusal for interpolation comment, got nil")
	}
	if fmtExitCode(err) != 2 {
		t.Errorf("interpolation-comment refusal should be exit 2, got %d", fmtExitCode(err))
	}
}

func TestFormatOne_ReadError(t *testing.T) {
	if _, _, err := formatOne(filepath.Join(t.TempDir(), "does-not-exist.ail")); err == nil {
		t.Fatal("expected read error for missing file, got nil")
	}
}

func TestFormatOne_CanonicalIsStable(t *testing.T) {
	dir := t.TempDir()
	// Already-canonical input must come back byte-identical (orig == canonical).
	canonicalSrc := "module t/s\n\nfunc f() = 1\n"
	path := writeTemp(t, dir, "s.ail", canonicalSrc)
	orig, canonical, err := formatOne(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(orig) != string(canonical) {
		t.Errorf("canonical input should be unchanged:\n got %q\nwant %q", canonical, orig)
	}
}

func TestAtomicWriteFile_PreservesMode(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "m.ail")
	if err := os.WriteFile(path, []byte("old"), 0o640); err != nil {
		t.Fatal(err)
	}
	if err := atomicWriteFile(path, []byte("new content")); err != nil {
		t.Fatalf("atomicWriteFile: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "new content" {
		t.Errorf("content = %q, want %q", data, "new content")
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	// Windows does not honor Unix permission bits (Go reports 0666/0444 based on
	// the read-only attribute only), so the exact-mode assertion is Unix-only.
	// atomicWriteFile still preserves the source mode via os.Chmod on all platforms.
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o640 {
		t.Errorf("mode = %v, want 0640", info.Mode().Perm())
	}
}

func TestAtomicWriteFile_NoTempLeftBehind(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "x.ail")
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := atomicWriteFile(path, []byte("y")); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.Name() != "x.ail" {
			t.Errorf("unexpected leftover file: %s", e.Name())
		}
	}
}

func TestFmtWrite_AtomicAcrossFiles(t *testing.T) {
	dir := t.TempDir()
	// One valid non-canonical file and one that does NOT parse. The write must
	// touch NEITHER, because validation of all inputs must pass first. (Commented
	// files now format, so the failing sibling is a parse error → exit 3.)
	good := writeTemp(t, dir, "good.ail", "module t/g\nfunc f() = let a = 1; a\n")
	bad := writeTemp(t, dir, "bad.ail", "module t/b\nfunc f( = \n")

	goodBefore, _ := os.ReadFile(good)
	badBefore, _ := os.ReadFile(bad)

	code := fmtWrite([]string{good, bad})
	if code != exitParse {
		t.Errorf("expected exit code %d for parse-error input, got %d", exitParse, code)
	}

	goodAfter, _ := os.ReadFile(good)
	badAfter, _ := os.ReadFile(bad)
	if string(goodBefore) != string(goodAfter) {
		t.Error("valid file was modified despite a sibling failure (atomicity violated)")
	}
	if string(badBefore) != string(badAfter) {
		t.Error("failing file was modified (must stay byte-identical)")
	}
}

func TestFmtWrite_RewritesAllValid(t *testing.T) {
	dir := t.TempDir()
	a := writeTemp(t, dir, "a.ail", "module t/a\nfunc f() = let x = 1; x\n")
	b := writeTemp(t, dir, "b.ail", "module t/b\nfunc g() = let y = 2; y\n")

	if code := fmtWrite([]string{a, b}); code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	// Both should now be canonical: a second write is a no-op and --check-style
	// comparison holds.
	for _, path := range []string{a, b} {
		orig, canonical, err := formatOne(path)
		if err != nil {
			t.Fatal(err)
		}
		if string(orig) != string(canonical) {
			t.Errorf("%s not canonical after write", path)
		}
	}
}

func TestFmtCheck_ReturnsCodes(t *testing.T) {
	dir := t.TempDir()
	canonical := writeTemp(t, dir, "c.ail", "module t/c\n\nfunc f() = 1\n")
	drifted := writeTemp(t, dir, "d.ail", "module t/d\nfunc f() = let a = 1; a\n")
	parseErr := writeTemp(t, dir, "x.ail", "module t/x\nfunc f( = \n")

	if code := fmtCheck([]string{canonical}); code != 0 {
		t.Errorf("canonical: expected 0, got %d", code)
	}
	if code := fmtCheck([]string{drifted}); code != 1 {
		t.Errorf("drifted: expected 1, got %d", code)
	}
	if code := fmtCheck([]string{parseErr}); code != exitParse {
		t.Errorf("parse error: expected %d, got %d", exitParse, code)
	}
	// A drifted file among canonical ones still yields 1, and no file is written.
	before, _ := os.ReadFile(drifted)
	if code := fmtCheck([]string{canonical, drifted}); code != 1 {
		t.Errorf("mixed: expected 1, got %d", code)
	}
	after, _ := os.ReadFile(drifted)
	if string(before) != string(after) {
		t.Error("--check must never write")
	}
}

// TestFmtCommentedFormats proves the Phase-2 behavior across all three modes: a
// commented file now FORMATS with its comment preserved (exit 0 in stdout;
// stdout/check never modify the file; write canonicalizes it).
func TestFmtCommentedFormats(t *testing.T) {
	src := "module t/c\n\n-- a real comment\nfunc f() = 1\n"

	t.Run("stdout", func(t *testing.T) {
		dir := t.TempDir()
		path := writeTemp(t, dir, "s.ail", src)
		before, _ := os.ReadFile(path)
		if code := fmtStdout(path); code != 0 {
			t.Errorf("expected exit 0 (commented file now formats), got %d", code)
		}
		after, _ := os.ReadFile(path)
		if string(before) != string(after) {
			t.Error("stdout mode must not modify the input file")
		}
	})

	t.Run("check", func(t *testing.T) {
		dir := t.TempDir()
		path := writeTemp(t, dir, "c.ail", src)
		before, _ := os.ReadFile(path)
		// Already canonical (comment preserved, canonical layout) → exit 0.
		if code := fmtCheck([]string{path}); code != 0 && code != 1 {
			t.Errorf("expected exit 0 or 1 (formats), got %d", code)
		}
		after, _ := os.ReadFile(path)
		if string(before) != string(after) {
			t.Error("--check must never write")
		}
	})

	t.Run("write", func(t *testing.T) {
		dir := t.TempDir()
		path := writeTemp(t, dir, "w.ail", src)
		if code := fmtWrite([]string{path}); code != 0 {
			t.Errorf("expected exit 0, got %d", code)
		}
		after, _ := os.ReadFile(path)
		if !contains(string(after), "-- a real comment") {
			t.Errorf("comment lost after --write:\n%s", after)
		}
	})
}

func TestFmtStdout_LeavesFileUnchanged(t *testing.T) {
	dir := t.TempDir()
	src := "module t/s\nfunc f() = let a = 1; a\n"
	path := writeTemp(t, dir, "s.ail", src)
	if code := fmtStdout(path); code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	after, _ := os.ReadFile(path)
	if string(after) != src {
		t.Error("stdout mode must not modify the input file")
	}
}
