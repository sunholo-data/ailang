package testing

import "testing"

// #1328: inline tests compiled the module twice per case (binding + cluster),
// so a 41-case file paid 82 compiles. Both compiles read the same file, so
// each now runs once per file and every case reuses it.
func TestInlineTests_CompileOncePerFile(t *testing.T) {
	path := writeEngineSource(t, "gcd.ail", `module gcd

export pure func gcd(a: int, b: int) -> int
  tests [
    ((12, 8), 4),
    ((54, 24), 6),
    ((7, 0), 7)
  ]
{
  if b == 0 then a else gcd(b, a % b)
}

export pure func lcm(a: int, b: int) -> int
  tests [
    ((4, 6), 12),
    ((3, 5), 99)
  ]
{
  (a * b) / gcd(a, b)
}
`)
	suite, exec := runWithExecutor(t, path, false, false)
	if exec.inlineRuns != 2 {
		t.Errorf("inline compiles = %d, want 2 (one binding, one cluster) for 5 cases", exec.inlineRuns)
	}
	// Outcomes are per case: gcd (no deps, binding harness) passes, lcm
	// (depends on gcd, cluster harness) has one deliberate failure.
	if suite.PassedTests != 4 || suite.FailedTests != 1 {
		t.Errorf("passed=%d failed=%d, want 4 and 1: %+v", suite.PassedTests, suite.FailedTests, suite.Tests)
	}
}

// A module-less file strips per function (the function under test is kept
// even when effectful), so its binding compile is memoized per function.
func TestInlineTests_ModulelessMemoPerFunction(t *testing.T) {
	path := writeEngineSource(t, "loose.ail", `pure func inc(x: int) -> int
  tests [ (1, 2), (2, 3) ]
{ x + 1 }

pure func dec(x: int) -> int
  tests [ (2, 1), (3, 2) ]
{ x - 1 }
`)
	suite, exec := runWithExecutor(t, path, false, false)
	// two functions' bindings + one shared cluster attempt
	if exec.inlineRuns != 3 {
		t.Errorf("inline compiles = %d, want 3", exec.inlineRuns)
	}
	if suite.PassedTests != 4 {
		t.Errorf("passed=%d, want 4: %+v", suite.PassedTests, suite.Tests)
	}
}
