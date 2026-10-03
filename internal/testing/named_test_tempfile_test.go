package testing

// Regression tests for #1502 (design m-namedtest-tempfile-leak): the
// named-test executor used to write its `_namedtest_body_*.ail` copy of the
// test module INTO the package directory, removed only by a defer. An
// interrupted `ailang test` (SIGTERM from a CI timeout, Ctrl-C, SIGKILL) left
// the copy behind, and `pkg quality`, `publish` and the next `ailang test`
// picked it up as a real module. The copy now lives in a private temp dir, so
// no kill of any kind can leave it in the package.

import (
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// The package directory is made read-only for the duration of the run: a
// harness that writes ANYTHING there fails, deterministically, instead of
// depending on catching a file in the window before its defer removes it.
func TestNamedTestBodyIsNeverWrittenIntoThePackageDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("directory write permission bits are not enforced on Windows")
	}
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory write permission bits")
	}
	pkgDir := t.TempDir()
	write := func(name, body string) {
		if err := os.WriteFile(filepath.Join(pkgDir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("ailang.toml", "[package]\nname = \"x/leak\"\nversion = \"0.1.0\"\nedition = \"1\"\n")
	write("lib.ail", "module x/leak/lib\n\nexport pure func inc(x: int) -> int = x + 1\n")
	src := "module x/leak/t_test\n\nimport x/leak/lib (inc)\n\ntest \"sibling import\" { inc(1) == 2 }\n"
	path := filepath.Join(pkgDir, "t_test.ail")
	write("t_test.ail", src)

	if err := os.Chmod(pkgDir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(pkgDir, 0o755) })

	stderr := captureStderr(t, func() {
		result := runTestsOnFile(t, path, src)
		if result.FailedTests > 0 || result.PassedTests != 1 {
			t.Errorf("expected 1 pass and 0 failures with a read-only package dir, got %d passed / %d failed; first error: %s",
				result.PassedTests, result.FailedTests, firstFailureError(result))
		}
	})

	entries, err := os.ReadDir(pkgDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "_namedtest_body_") {
			t.Errorf("package dir contains harness debris %s", e.Name())
		}
	}
	// The temp copy is an implementation detail: no diagnostic may name it.
	if strings.Contains(stderr, "_namedtest_body_") || strings.Contains(stderr, "MOD010") {
		t.Errorf("named-test run leaked its temp copy into diagnostics:\n%s", stderr)
	}
}

// captureStderr runs fn with os.Stderr redirected and returns what it wrote.
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	orig := os.Stderr
	os.Stderr = w
	done := make(chan string)
	go func() {
		b, _ := io.ReadAll(r)
		done <- string(b)
	}()
	defer func() { os.Stderr = orig }()
	fn()
	os.Stderr = orig
	_ = w.Close()
	return <-done
}
