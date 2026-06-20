package projecteval

import (
	"os"
	"os/exec"
	"testing"
)

func runnerReturning(out string) CheckRunner {
	return func(string) (string, error) { return out, nil }
}

func TestParseCheck(t *testing.T) {
	cases := []struct {
		out          string
		wantP, wantF int
		wantOK       bool
	}{
		{"✗ 2 files checked: 1 passed, 1 failed", 1, 1, true},
		{"✓ 3 files checked: 3 passed, 0 failed", 3, 0, true},
		{"✓ 2 files checked, all passed!", 2, 0, true}, // REAL clean multi-file format
		{"✓ No errors found!", 0, 0, false},
	}
	for _, c := range cases {
		p, f, ok := parseCheck(c.out)
		if p != c.wantP || f != c.wantF || ok != c.wantOK {
			t.Errorf("parseCheck(%q) = (%d,%d,%v), want (%d,%d,%v)", c.out, p, f, ok, c.wantP, c.wantF, c.wantOK)
		}
	}
}

func TestGradeBuild(t *testing.T) {
	// Clean multi-module build (zero failures) → BuildOk.
	if r := GradeBuild("x", runnerReturning("✓ 3 files checked: 3 passed, 0 failed")); !r.BuildOk || r.Passed != 3 {
		t.Errorf("clean build: %+v, want BuildOk && Passed=3", r)
	}
	// One module failing → not BuildOk (don't trust exit code — parse the summary).
	if r := GradeBuild("x", runnerReturning("✗ 2 files checked: 1 passed, 1 failed")); r.BuildOk {
		t.Errorf("failing build graded BuildOk: %+v", r)
	}
	// Single-file "No errors" path (no summary line) → BuildOk.
	if r := GradeBuild("x", runnerReturning("→ Type checking...\n✓ No errors found!")); !r.BuildOk {
		t.Errorf("single-file clean build not BuildOk: %+v", r)
	}
}

// TestGradeBuild_RealPackage validates the grader end-to-end against the installed compiler on a
// real locked multi-module package — catching format drift (the clean summary is "all passed!", not
// "0 failed"). Skips if ailang isn't on PATH or the example/lock is absent.
func TestGradeBuild_RealPackage(t *testing.T) {
	if _, err := exec.LookPath("ailang"); err != nil {
		t.Skip("ailang not on PATH")
	}
	dir := "../../examples/intra_package_imports"
	if _, err := os.Stat(dir + "/ailang.lock"); err != nil {
		t.Skip("example package not locked (run `ailang lock` in it)")
	}
	r := GradeBuild(dir, AilangCheckRunner("", 0))
	if !r.BuildOk || r.Passed < 2 {
		t.Errorf("real locked package: %+v, want BuildOk && Passed>=2", r)
	}
}

// TestGradeProject_RealFixture grades the calc_bugfix fixture end-to-end: it BUILDS (the bug is a
// logic error, not a type error) but FAILS acceptance (sub=a+b prints 13, expected 7). Validates
// the full build+behaviour pipeline on a real multi-module project. Skips if ailang/fixture absent.
func TestGradeProject_RealFixture(t *testing.T) {
	if _, err := exec.LookPath("ailang"); err != nil {
		t.Skip("ailang not on PATH")
	}
	dir := "../../eval_projects/calc_bugfix"
	if _, err := os.Stat(dir + "/ailang.lock"); err != nil {
		t.Skip("fixture not locked")
	}
	accept := StdoutAcceptance(AcceptanceSpec{EntryFile: "main.ail", Expected: "7"})
	r := GradeProject(dir, AilangCheckRunner("", 0), accept)
	if !r.BuildOk {
		t.Errorf("fixture should BUILD (logic bug, not type error): %+v", r)
	}
	if r.AcceptOk {
		t.Errorf("buggy baseline should FAIL acceptance (prints 13, expected 7): %+v", r)
	}
}

func TestGradeProject_AcceptanceGatedOnBuild(t *testing.T) {
	// Build fails → acceptance must NOT run (AcceptOk stays false).
	called := false
	r := GradeProject("x", runnerReturning("✗ 1 passed, 1 failed"), func(string) bool { called = true; return true })
	if r.BuildOk || r.AcceptOk || called {
		t.Errorf("acceptance ran despite failing build: %+v called=%v", r, called)
	}
	// Build ok → acceptance runs and is recorded.
	r = GradeProject("x", runnerReturning("✓ 2 passed, 0 failed"), func(string) bool { return true })
	if !r.BuildOk || !r.AcceptOk {
		t.Errorf("build-ok project: %+v, want BuildOk && AcceptOk", r)
	}
}
