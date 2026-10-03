package testing

// Engine parity for `ailang test --bytecode` (#1487): a suite must report the
// same outcomes, error text, property counterexamples and seeds on the
// evaluator and on the bytecode VM.

import (
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/lexer"
	"github.com/sunholo-data/ailang/internal/parser"
)

// engineFixture copies testdata/engine_parity/<name> into a temp dir, once,
// so that both engines run the same path (locations are part of the report).
func engineFixture(t *testing.T, name string) string {
	t.Helper()
	src, err := os.ReadFile(filepath.Join("testdata", "engine_parity", name))
	if err != nil {
		t.Fatal(err)
	}
	return writeEngineSource(t, name, string(src))
}

func writeEngineSource(t *testing.T, name, src string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// runEngine runs the file at path with the given engine flags, a fixed
// master seed and the given recursion limit (0 = default).
func runEngine(t *testing.T, path string, bytecode, strict bool, maxDepth int) *SuiteResult {
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
	cfg := TestConfig{
		WorkspaceRoot:     filepath.Dir(path),
		SeedMode:          SeedModeMaster,
		MasterSeed:        1487,
		MaxRecursionDepth: maxDepth,
		Bytecode:          bytecode,
		StrictBytecode:    strict,
	}
	res, err := RunTestsFromFileWithConfig(path, file, cfg)
	if err != nil {
		t.Fatalf("run %s: %v", path, err)
	}
	return res
}

// tempBodyDir matches the private temp dir a body is compiled in; error text
// that quotes it differs between any two runs, engine or not.
var tempBodyDir = regexp.MustCompile(`ailang-namedtest-\d+`)

// outcomes strips what legitimately differs between two runs (durations, the
// temp dir name, engine stats) and keeps everything the report shows.
func outcomes(r *SuiteResult) SuiteResult {
	out := *r
	out.TotalDuration = 0
	out.Engine = EngineStats{}
	out.Tests = make([]TestResult, len(r.Tests))
	for i, tr := range r.Tests {
		tr.Duration = 0
		tr.Error = tempBodyDir.ReplaceAllString(tr.Error, "ailang-namedtest-N")
		out.Tests[i] = tr
	}
	out.Properties = make([]PropertyResult, len(r.Properties))
	for i, pr := range r.Properties {
		pr.Duration = 0
		out.Properties[i] = pr
	}
	return out
}

func TestEngineParity_SameReportOnBothEngines(t *testing.T) {
	for _, fixture := range []string{"mixed.ail", "sweep.ail"} {
		t.Run(fixture, func(t *testing.T) {
			path := engineFixture(t, fixture)
			ev := runEngine(t, path, false, false, 0)
			bc := runEngine(t, path, true, false, 0)

			if ev.FailedTests == 0 {
				t.Fatalf("fixture has no failing test; parity over passes only proves little")
			}
			if bc.Engine.VMBodies == 0 {
				t.Fatalf("no body ran on the VM (%s); the comparison would be evaluator vs evaluator", bc.Engine.Summary())
			}
			if ev.Engine != (EngineStats{}) {
				t.Errorf("evaluator run recorded engine stats %+v", ev.Engine)
			}
			if got, want := outcomes(bc), outcomes(ev); !reflect.DeepEqual(got, want) {
				t.Errorf("bytecode report differs from evaluator report\nbytecode:  %+v\nevaluator: %+v", got, want)
			}
		})
	}
}

// The mixed fixture covers each route: bodies the VM runs (including one that
// calls an evaluator-only helper through the bridge, and an ADT match), and
// bodies that fall back (a runtime error, a non-bool body).
func TestEngineParity_MixedFixtureRoutes(t *testing.T) {
	bc := runEngine(t, engineFixture(t, "mixed.ail"), true, false, 0)
	want := EngineStats{VMBodies: 8, FallbackBodies: 2}
	got := bc.Engine
	got.FirstFallback = ""
	if got != want {
		t.Errorf("engine stats = %+v, want %+v (%s)", bc.Engine, want, bc.Engine.Summary())
	}
	if len(bc.Properties) != 2 {
		t.Fatalf("want 2 ensures properties, got %d", len(bc.Properties))
	}
	var failed bool
	for _, p := range bc.Properties {
		if p.Status == StatusFail && p.Seed != 0 {
			failed = true
		}
	}
	if !failed {
		t.Errorf("want a failing seeded property in the fixture, got %+v", bc.Properties)
	}
}

// --strict-bytecode fails exactly the bodies the VM cannot run, names the
// flag, and leaves every other outcome as the evaluator reports it.
func TestEngineParity_StrictFailsOnlyNonVMBodies(t *testing.T) {
	path := engineFixture(t, "mixed.ail")
	ev := runEngine(t, path, false, false, 0)
	st := runEngine(t, path, true, true, 0)

	strictOnly := map[string]bool{"runtime error": true, "non-bool body": true, "bridged helper": true}
	evByName := map[string]TestResult{}
	for _, tr := range outcomes(ev).Tests {
		evByName[tr.Name] = tr
	}
	for _, tr := range outcomes(st).Tests {
		if strictOnly[tr.Name] {
			if tr.Status != StatusFail || !strings.Contains(tr.Error, "--strict-bytecode") {
				t.Errorf("%q: status %s error %q; want a --strict-bytecode failure", tr.Name, tr.Status, tr.Error)
			}
			continue
		}
		if want := evByName[tr.Name]; !reflect.DeepEqual(tr, want) {
			t.Errorf("%q under strict: %+v, want the evaluator's %+v", tr.Name, tr, want)
		}
	}
	if st.Engine.StrictFailures != 3 || st.Engine.FallbackBodies != 0 {
		t.Errorf("strict engine stats = %+v, want 3 strict failures and no fallback", st.Engine)
	}
	if !reflect.DeepEqual(outcomes(st).Properties, outcomes(ev).Properties) {
		t.Errorf("properties differ under strict mode")
	}
}

// --max-recursion-depth bounds the VM's frame stack as it bounds the
// evaluator, so a deep non-tail recursion fails at the same configured depth
// on both engines.
func TestEngineParity_RecursionLimitReachesVM(t *testing.T) {
	path := writeEngineSource(t, "deep.ail", `module deep

pure func sumTo(i: int) -> int =
  if i <= 0 then 0 else i + sumTo(i - 1)

test "500 deep" {
  sumTo(500) == 125250
}
`)
	if r := runEngine(t, path, true, true, 100); r.FailedTests != 1 || !strings.Contains(r.Tests[0].Error, "stack overflow") {
		t.Errorf("strict VM with limit 100: failed=%d error=%q; want a stack overflow", r.FailedTests, r.Tests[0].Error)
	}
	if r := runEngine(t, path, true, true, 1000); r.PassedTests != 1 {
		t.Errorf("strict VM with limit 1000: %+v; want a pass", r.Tests)
	}
	ev := runEngine(t, path, false, false, 100)
	bc := runEngine(t, path, true, false, 100)
	if ev.FailedTests != 1 || !reflect.DeepEqual(outcomes(bc), outcomes(ev)) {
		t.Errorf("limit 100: evaluator %+v, bytecode %+v; want the same failure", ev.Tests, bc.Tests)
	}
}
