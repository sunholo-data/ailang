package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestEqContainersExample runs examples/eq_containers.ail and requires every
// comparison to evaluate to its expected boolean. The reverted 08-29 attempt
// claimed its probes were "verified live" while its equalities never ran;
// this pins the output, not just a clean exit (M-EQ-DERIVE-CONTAINERS).
func TestEqContainersExample(t *testing.T) {
	t.Setenv("AILANG_NO_CACHE", "1")
	example, err := filepath.Abs(filepath.Join("..", "..", "examples", "eq_containers.ail"))
	if err != nil {
		t.Fatal(err)
	}
	stdout, stderr, code := runCLI(t, "run", "--caps", "IO", "--entry", "main", "--relax-modules", example)
	if code != 0 {
		t.Fatalf("exit %d\nstdout=%s\nstderr=%s", code, stdout, stderr)
	}
	if strings.Contains(stdout, "WRONG") {
		t.Errorf("a comparison evaluated to the wrong boolean:\n%s", stdout)
	}
	if n := strings.Count(stdout, "\nok "); n < 28 {
		t.Errorf("want 28 evaluated comparisons, got %d:\n%s", n, stdout)
	}
}

// TestEqContainersNegatives: each file in examples/eq_containers_negative/ must
// FAIL with the error its "-- Expected:" header names.
func TestEqContainersNegatives(t *testing.T) {
	t.Setenv("AILANG_NO_CACHE", "1")
	files, err := filepath.Glob(filepath.Join("..", "..", "examples", "eq_containers_negative", "*.ail"))
	if err != nil || len(files) == 0 {
		t.Fatalf("no negative fixtures found: %v", err)
	}
	expected := regexp.MustCompile(`(?m)^-- Expected: (.+)$`)
	for _, f := range files {
		f, _ = filepath.Abs(f)
		t.Run(filepath.Base(f), func(t *testing.T) {
			src, err := os.ReadFile(f)
			if err != nil {
				t.Fatal(err)
			}
			m := expected.FindSubmatch(src)
			if m == nil {
				t.Fatalf("%s has no '-- Expected:' header", f)
			}
			want := strings.TrimSpace(string(m[1]))
			// A short, stable stem of the header: the part before any "{" or "...".
			if i := strings.IndexAny(want, "{…"); i > 0 {
				want = strings.TrimSpace(want[:i])
			}
			stdout, stderr, code := runCLI(t, "run", "--caps", "IO", "--entry", "main", "--relax-modules", f)
			if code == 0 {
				t.Fatalf("must fail, but ran:\n%s", stdout)
			}
			if out := stdout + stderr; !strings.Contains(out, want) {
				t.Errorf("error should contain %q:\n%s", want, out)
			}
		})
	}
}

// TestFloatNaNExample pins examples/float_nan.ail: float == is IEEE on every
// path and std/math.isNaN detects NaN (M-FLOAT-EQ-ONE-SEMANTICS, #1274).
func TestFloatNaNExample(t *testing.T) {
	t.Setenv("AILANG_NO_CACHE", "1")
	example, err := filepath.Abs(filepath.Join("..", "..", "examples", "float_nan.ail"))
	if err != nil {
		t.Fatal(err)
	}
	stdout, stderr, code := runCLI(t, "run", "--caps", "IO", "--entry", "main", "--relax-modules", example)
	if code != 0 {
		t.Fatalf("exit %d\nstdout=%s\nstderr=%s", code, stdout, stderr)
	}
	if strings.Contains(stdout, "WRONG") {
		t.Errorf("a NaN comparison evaluated to the wrong boolean:\n%s", stdout)
	}
	if n := strings.Count(stdout, "\nok "); n < 12 {
		t.Errorf("want 12 evaluated comparisons, got %d:\n%s", n, stdout)
	}
}
