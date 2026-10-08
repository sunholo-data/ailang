package testing

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/lexer"
	"github.com/sunholo-data/ailang/internal/parser"
)

const forallFixture = `module forallfix

export pure func double(x: int) -> int = x * 2

export pure func rev(xs: [int]) -> [int] = match xs { [] => [], [h, ...t] => rev(t) ++ [h] }

test "a named test shares the compile" { double(2) == 4 }

property "increment grows" {
  forall(n: int) => n + 1 > n
}

property "calls a module function" {
  forall(n: int) => double(n) == n + n
}

property "float binder" {
  forall(x: float) => x + 1.0 > x - 1.0
}

property "two binders" {
  forall(a: int, b: string) => a == a && b == b
}

property "list binder" {
  forall(xs: [int]) => rev(rev(xs)) == xs
}

property "deliberately false" {
  forall(n: int) => n < 5
}
`

func propsByName(res *SuiteResult) map[string]PropertyResult {
	m := make(map[string]PropertyResult, len(res.Properties))
	for _, p := range res.Properties {
		m[p.Name] = p
	}
	return m
}

// #624: forall properties never evaluated ("empty program" on the first case,
// a parse error once the body called a module function). They now run every
// generated case as a call to one compiled function, in the file's batch.
func TestForall_RunsEveryCase(t *testing.T) {
	path := writeEngineSource(t, "forallfix.ail", forallFixture)
	res, exec := runWithExecutor(t, path, false, false)
	if exec.pipelineRuns != 1 {
		t.Errorf("compiles = %d, want 1 (named test and properties share the batch)", exec.pipelineRuns)
	}
	props := propsByName(res)
	for _, name := range []string{"increment grows", "calls a module function", "float binder", "two binders", "list binder"} {
		p := props[name]
		if p.Status != StatusPass || p.TestsRun != 100 {
			t.Errorf("%q: %s after %d cases (%s), want pass after 100", name, p.Status, p.TestsRun, p.Error)
		}
	}
	// Shrinking runs to a fixpoint: n < 5 fails first at some large n and
	// shrinks to exactly 5.
	if p := props["deliberately false"]; p.Status != StatusFail || p.Error != "property failed on input: [5]" {
		t.Errorf("deliberately false: %s %q, want fail with input [5]", p.Status, p.Error)
	}
}

// The generated stream is the derived seed's: two runs agree exactly, and the
// failing case number moves with the master seed.
func TestForall_SeedDrivesTheStream(t *testing.T) {
	path := writeEngineSource(t, "seedfix.ail", `module seedfix

property "small" {
  forall(n: int) => n < 50
}
`)
	run := func(seed int64) PropertyResult {
		t.Helper()
		res := runEngineSeed(t, path, seed)
		return propsByName(res)["small"]
	}
	a, b := run(7), run(7)
	if a.TestsRun != b.TestsRun || a.Error != b.Error {
		t.Errorf("same seed, different runs: %d %q vs %d %q", a.TestsRun, a.Error, b.TestsRun, b.Error)
	}
	if a.Status != StatusFail || a.Error != "property failed on input: [50]" {
		t.Errorf("seed 7: %s %q, want fail shrunk to [50]", a.Status, a.Error)
	}
	differs := false
	for seed := int64(8); seed < 20 && !differs; seed++ {
		differs = run(seed).TestsRun != a.TestsRun
	}
	if !differs {
		t.Errorf("the failing case number never moved across 12 seeds: the stream ignores the seed")
	}
}

// A property that does not type-check fails by itself (D1): the batch falls
// back, the property is compiled alone and reports the compile error, and
// the other tests in the file still run.
func TestForall_IllTypedPropertyFailsAlone(t *testing.T) {
	path := writeEngineSource(t, "badprop.ail", `module badprop

test "fine" { 1 + 1 == 2 }

property "ill typed" {
  forall(n: int) => n == "a"
}

property "fine too" {
  forall(n: int) => n == n
}
`)
	res, _ := runWithExecutor(t, path, false, false)
	props := propsByName(res)
	if p := props["ill typed"]; p.Status != StatusFail || !strings.Contains(p.Error, "property does not compile") ||
		!strings.Contains(p.Error, path+":6 (property)") || strings.Contains(p.Error, "ailang-namedtest-") {
		t.Errorf("ill typed: %s %q, want a compile error at %s:6 (property), the forall line", p.Status, p.Error, path)
	}
	if p := props["fine too"]; p.Status != StatusPass || p.TestsRun != 100 {
		t.Errorf("fine too: %s after %d (%s)", p.Status, p.TestsRun, p.Error)
	}
	if tr := byName(res)["fine"]; tr.Status != StatusPass {
		t.Errorf("named test must still pass: %+v", tr)
	}
	if len(res.NamedBatchFailures) != 1 || res.NamedBatchFailures[0].HarnessBug {
		t.Errorf("want one batch failure, not a harness bug: %+v", res.NamedBatchFailures)
	}
}

// runEngineSeed runs the file on the evaluator under a master seed.
func runEngineSeed(t *testing.T, path string, seed int64) *SuiteResult {
	t.Helper()
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	p := parser.New(lexer.New(string(src), path))
	file := p.ParseFile()
	if errs := p.Errors(); len(errs) != 0 {
		t.Fatalf("parse %s: %v", path, errs)
	}
	res, err := RunTestsFromFileWithConfig(path, file, TestConfig{
		WorkspaceRoot: filepath.Dir(path), SeedMode: SeedModeMaster, MasterSeed: seed,
	})
	if err != nil {
		t.Fatal(err)
	}
	return res
}
