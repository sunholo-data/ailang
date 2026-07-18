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

func TestFormatOne_RejectsComments(t *testing.T) {
	dir := t.TempDir()
	path := writeTemp(t, dir, "c.ail", "module t/c\n-- hello\nfunc f() = 1\n")
	if _, _, err := formatOne(path); err == nil {
		t.Fatal("expected error for commented file, got nil")
	}
}

func TestFormatOne_RejectsParseError(t *testing.T) {
	dir := t.TempDir()
	path := writeTemp(t, dir, "p.ail", "module t/p\nfunc f( = \n")
	if _, _, err := formatOne(path); err == nil {
		t.Fatal("expected parse error, got nil")
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
	// One valid non-canonical file and one commented (invalid) file. The write
	// must touch NEITHER, because validation of all inputs must pass first.
	good := writeTemp(t, dir, "good.ail", "module t/g\nfunc f() = let a = 1; a\n")
	bad := writeTemp(t, dir, "bad.ail", "module t/b\n-- comment\nfunc f() = 1\n")

	goodBefore, _ := os.ReadFile(good)
	badBefore, _ := os.ReadFile(bad)

	code := fmtWrite([]string{good, bad})
	if code != 2 {
		t.Errorf("expected exit code 2 for commented input, got %d", code)
	}

	goodAfter, _ := os.ReadFile(good)
	badAfter, _ := os.ReadFile(bad)
	if string(goodBefore) != string(goodAfter) {
		t.Error("valid file was modified despite a sibling failure (atomicity violated)")
	}
	if string(badBefore) != string(badAfter) {
		t.Error("commented file was modified (must stay byte-identical)")
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
	commented := writeTemp(t, dir, "x.ail", "module t/x\n-- c\nfunc f() = 1\n")

	if code := fmtCheck([]string{canonical}); code != 0 {
		t.Errorf("canonical: expected 0, got %d", code)
	}
	if code := fmtCheck([]string{drifted}); code != 1 {
		t.Errorf("drifted: expected 1, got %d", code)
	}
	if code := fmtCheck([]string{commented}); code != 2 {
		t.Errorf("commented: expected 2, got %d", code)
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

// TestFmtCommentedExitsAndUnchanged proves the Phase-1 comment partition across
// all three modes: a commented file yields exit 2 and is never modified.
func TestFmtCommentedExitsAndUnchanged(t *testing.T) {
	dir := t.TempDir()
	src := "module t/c\n-- a real comment\nfunc f() = 1\n"

	t.Run("stdout", func(t *testing.T) {
		path := writeTemp(t, dir, "s.ail", src)
		before, _ := os.ReadFile(path)
		if code := fmtStdout(path); code != 2 {
			t.Errorf("expected exit 2, got %d", code)
		}
		after, _ := os.ReadFile(path)
		if string(before) != string(after) {
			t.Error("commented file was modified in stdout mode")
		}
	})

	t.Run("check", func(t *testing.T) {
		path := writeTemp(t, dir, "c.ail", src)
		before, _ := os.ReadFile(path)
		if code := fmtCheck([]string{path}); code != 2 {
			t.Errorf("expected exit 2, got %d", code)
		}
		after, _ := os.ReadFile(path)
		if string(before) != string(after) {
			t.Error("commented file was modified in check mode")
		}
	})

	t.Run("write", func(t *testing.T) {
		path := writeTemp(t, dir, "w.ail", src)
		before, _ := os.ReadFile(path)
		if code := fmtWrite([]string{path}); code != 2 {
			t.Errorf("expected exit 2, got %d", code)
		}
		after, _ := os.ReadFile(path)
		if string(before) != string(after) {
			t.Error("commented file was modified in write mode")
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
