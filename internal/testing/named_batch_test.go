package testing

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/lexer"
	"github.com/sunholo-data/ailang/internal/parser"
)

// runCounted runs every test in the file at path and also returns how many
// named-test pipeline compiles the run took (M-TEST-RUNNER-COMPILE-ONCE).
func runCounted(t *testing.T, path string, bytecode, strict bool) (*SuiteResult, int) {
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
	cfg := TestConfig{WorkspaceRoot: filepath.Dir(path), SeedMode: SeedModeMaster, MasterSeed: 1,
		Bytecode: bytecode, StrictBytecode: strict}
	identity, err := ResolveModuleIdentity(cfg.WorkspaceRoot, path, file.Module.Path)
	if err != nil {
		t.Fatal(err)
	}
	r := NewRunnerWithConfig(path, cfg, identity)
	r.executor.SetSourceFile(file)
	res := r.RunSuite(NewCollector(path).Collect(file))
	res.Engine = r.executor.engine // as RunTestsFromFileWithConfig does
	return res, r.executor.pipelineRuns
}

func byName(res *SuiteResult) map[string]TestResult {
	m := make(map[string]TestResult, len(res.Tests))
	for _, tr := range res.Tests {
		m[tr.Name] = tr
	}
	return m
}

// All named tests of a file share one compile, on both engines.
func TestNamedBatch_OneCompilePerFile(t *testing.T) {
	path := writeEngineSource(t, "three.ail", `module three

export pure func double(x: int) -> int = x * 2

test "a" { double(1) == 2 }
test "b" { double(2) == 4 }
test "c" { assert double(3) == 6; true }
`)
	for _, bc := range []bool{false, true} {
		res, runs := runCounted(t, path, bc, false)
		if runs != 1 {
			t.Errorf("bytecode=%v: %d named-test compiles, want 1", bc, runs)
		}
		if res.PassedTests != 3 || len(res.NamedBatchFailures) != 0 {
			t.Errorf("bytecode=%v: passed=%d batchFailures=%v", bc, res.PassedTests, res.NamedBatchFailures)
		}
		if bc && res.Engine.VMBodies != 3 {
			t.Errorf("VM bodies = %d, want 3 (%s)", res.Engine.VMBodies, res.Engine.Summary())
		}
	}
}

// Outcomes and messages match the per-body path: pass, false, a failing
// assert (sentinel decoded per entry), a runtime error in a module function,
// and a polymorphic let. testdata/named_batch/mixed.ail is the V15 fixture.
func TestNamedBatch_OutcomesMatchPerBody(t *testing.T) {
	path := engineFixtureFrom(t, filepath.Join("testdata", "named_batch", "mixed.ail"))
	for _, bc := range []bool{false, true} {
		res, runs := runCounted(t, path, bc, false)
		if runs != 1 {
			t.Errorf("bytecode=%v: %d compiles, want 1", bc, runs)
		}
		got := byName(res)
		want := map[string]string{
			"plain pass":          "",
			"plain false":         "expected true, got false",
			"assert pass":         "",
			"assert second fails": "assertion 2 failed: `assert (double(3) == 7)`",
			"runtime error":       "evaluation error: RT001: integer division by zero",
			"poly let":            "",
			"body runtime error":  "evaluation error: RT001: integer division by zero",
		}
		for name, msg := range want {
			tr, ok := got[name]
			if !ok {
				t.Fatalf("bytecode=%v: no result for %q", bc, name)
			}
			if msg == "" {
				if tr.Status != StatusPass {
					t.Errorf("bytecode=%v %q: %s %s, want pass", bc, name, tr.Status, tr.Error)
				}
				continue
			}
			if tr.Status != StatusFail || !strings.Contains(tr.Error, msg) {
				t.Errorf("bytecode=%v %q: %s %q, want fail containing %q", bc, name, tr.Status, tr.Error, msg)
			}
		}
	}
}

// D1: a body that does not type-check fails alone, with the per-body message,
// and the batch failure is recorded (never silent) without blaming the harness.
func TestNamedBatch_IllTypedBodyFallsBackLoudly(t *testing.T) {
	path := writeEngineSource(t, "iso.ail", `module iso

export pure func double(x: int) -> int = x * 2

test "good one" { double(2) == 4 }
test "ill typed" { double("a") == 4 }
test "good two" { double(3) == 6 }
`)
	res, runs := runCounted(t, path, false, false)
	if runs != 4 { // the failed batch, then one per body
		t.Errorf("%d compiles, want 4", runs)
	}
	got := byName(res)
	if got["good one"].Status != StatusPass || got["good two"].Status != StatusPass {
		t.Errorf("good bodies must still pass: %+v", res.Tests)
	}
	if tr := got["ill typed"]; tr.Status != StatusFail || !strings.Contains(tr.Error, "type unification failed") {
		t.Errorf("ill-typed body: %s %q", tr.Status, tr.Error)
	}
	if len(res.NamedBatchFailures) != 1 {
		t.Fatalf("batch failures = %v, want 1", res.NamedBatchFailures)
	}
	f := res.NamedBatchFailures[0]
	if f.HarnessBug {
		t.Errorf("a user type error must not be reported as a harness bug: %s", f.Notice())
	}
	if !strings.Contains(f.Notice(), "compiled each test separately") {
		t.Errorf("notice = %q", f.Notice())
	}

	var buf bytes.Buffer
	if err := NewReporter(FormatJSON, &buf, false).Report(res); err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if _, ok := out["named_test_batch_failures"]; !ok {
		t.Errorf("--json lacks named_test_batch_failures: %s", buf.String())
	}
}

// A module that declares a reserved entry name is refused by the batch (it
// would collide) and runs per body, with the reason named.
func TestNamedBatch_ReservedNameFallsBack(t *testing.T) {
	path := writeEngineSource(t, "reserved.ail", `module reserved

pure func __namedtest_0() -> int = 1

test "uses it" { __namedtest_0() == 1 }
`)
	res, _ := runCounted(t, path, false, false)
	if res.PassedTests != 1 {
		t.Errorf("test must still run per body: %+v", res.Tests)
	}
	if len(res.NamedBatchFailures) != 1 || !strings.Contains(res.NamedBatchFailures[0].Reason, "reserved") ||
		res.NamedBatchFailures[0].HarnessBug {
		t.Errorf("batch failures = %+v", res.NamedBatchFailures)
	}
}

func engineFixtureFrom(t *testing.T, rel string) string {
	t.Helper()
	src, err := os.ReadFile(rel)
	if err != nil {
		t.Fatal(err)
	}
	return writeEngineSource(t, filepath.Base(rel), string(src))
}

// A body that does not TYPE as bool is a runtime failure of that test, not a
// compile error that sinks the batch (TestTestCommandBytecodeFlags found it).
func TestNamedBatch_NonBoolBodyStaysInBatch(t *testing.T) {
	path := writeEngineSource(t, "nonbool.ail", `module nonbool

test "passes" { true }
test "non-bool body" { 1.5 }
`)
	for _, bc := range []bool{false, true} {
		res, runs := runCounted(t, path, bc, false)
		if runs != 1 || len(res.NamedBatchFailures) != 0 {
			t.Errorf("bytecode=%v: compiles=%d batchFailures=%+v, want 1 and none", bc, runs, res.NamedBatchFailures)
		}
		if tr := byName(res)["non-bool body"]; tr.Status != StatusFail || !strings.Contains(tr.Error, "expected bool result") {
			t.Errorf("bytecode=%v: non-bool body: %s %q", bc, tr.Status, tr.Error)
		}
	}
}

// D5: runtime-error positions name the user's file, not the batch's temp
// file: a module-function line maps through the strip (an effectful function
// above it is stripped, shifting every later line), and a line inside a test
// body becomes the test's own line.
func TestNamedBatch_RuntimeErrorPositionsMapToSource(t *testing.T) {
	path := writeEngineSource(t, "pos.ail", `module pos

import std/io (println)

export func shout(s: string) -> () ! {IO} = {
  println(s);
  println(s)
}

export pure func safeDiv(a: int, b: int) -> int = a / b

test "in a function" { safeDiv(1, 0) == 0 }

test "in the body" { 1 / 0 == 0 }
`)
	for _, bc := range []bool{false, true} {
		res, _ := runCounted(t, path, bc, false)
		got := byName(res)
		for name, want := range map[string]string{
			"in a function": path + ":10:",
			"in the body":   path + ":14 (test body)",
		} {
			tr := got[name]
			if tr.Status != StatusFail || !strings.Contains(tr.Error, want) || strings.Contains(tr.Error, "ailang-namedtest-") {
				t.Errorf("bytecode=%v %q: %q, want a position %q and no temp path", bc, name, tr.Error, want)
			}
		}
	}
}
