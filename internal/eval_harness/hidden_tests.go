package eval_harness

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// M-EVAL-HIDDEN-TESTS: held-out tests. A benchmark's visible stdin/expected_stdout
// is rendered into the agent's README and task prompt (agent_prompt.go), so it is a
// sample, not an oracle: in the 2026-10-28 competitive-programming spike, 14 AILANG
// harness passes became 3 once re-graded against the problems' own hidden tests.
// Hidden tests are graded after the visible test passes and are never shown to the
// model. See design_docs/planned/v0_55_0/m-eval-hidden-tests.md.

// HiddenTest is one held-out stdin/expected-stdout pair, loaded from
// <hidden_tests_dir>/<Name>.in and <Name>.ans.
type HiddenTest struct {
	Name     string
	Stdin    string
	Expected string
}

// HiddenResult is the outcome of grading a candidate against a spec's hidden tests.
// FirstFail is "<name>:wrong_output|timeout|runtime_error", empty when all passed.
type HiddenResult struct {
	Total     int
	Passed    int
	FirstFail string
}

// AllPassed reports whether every hidden test passed. A spec with no hidden tests
// passes vacuously; callers only consult this when Total > 0.
func (h HiddenResult) AllPassed() bool { return h.Passed == h.Total }

// HiddenRun is what one execution of the candidate against a hidden test produced.
type HiddenRun struct {
	Stdout    string
	RuntimeOk bool
	TimedOut  bool
}

// loadHiddenTests reads spec.HiddenTestsDir (relative to the spec file's directory)
// into spec.HiddenTests. Every .in needs a .ans of the same stem and vice versa, and
// the directory must hold at least one pair: a hidden-test benchmark that silently
// loaded zero tests would grade exactly like one without them.
func loadHiddenTests(spec *BenchmarkSpec, specDir string) error {
	if spec.HiddenTestsDir == "" {
		return nil
	}
	dir := spec.HiddenTestsDir
	if !filepath.IsAbs(dir) {
		dir = filepath.Join(specDir, dir)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("hidden_tests_dir: %w", err)
	}
	ins, outs := map[string]bool{}, map[string]bool{}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		switch ext := filepath.Ext(e.Name()); ext {
		case ".in":
			ins[strings.TrimSuffix(e.Name(), ext)] = true
		case ".ans":
			outs[strings.TrimSuffix(e.Name(), ext)] = true
		}
	}
	for stem := range outs {
		if !ins[stem] {
			return fmt.Errorf("hidden_tests_dir %s: %s.ans has no %s.in", dir, stem, stem)
		}
	}
	tests := make([]HiddenTest, 0, len(ins))
	for stem := range ins {
		if !outs[stem] {
			return fmt.Errorf("hidden_tests_dir %s: %s.in has no %s.ans", dir, stem, stem)
		}
		in, err := os.ReadFile(filepath.Join(dir, stem+".in"))
		if err != nil {
			return fmt.Errorf("hidden_tests_dir: %w", err)
		}
		out, err := os.ReadFile(filepath.Join(dir, stem+".ans"))
		if err != nil {
			return fmt.Errorf("hidden_tests_dir: %w", err)
		}
		tests = append(tests, HiddenTest{Name: stem, Stdin: string(in), Expected: string(out)})
	}
	if len(tests) == 0 {
		return fmt.Errorf("hidden_tests_dir %s: no .in/.ans pairs", dir)
	}
	// Smallest input first, so a too-slow candidate fails fast on the cheapest
	// timeout; name breaks ties so the order is deterministic.
	sort.Slice(tests, func(i, j int) bool {
		if len(tests[i].Stdin) != len(tests[j].Stdin) {
			return len(tests[i].Stdin) < len(tests[j].Stdin)
		}
		return tests[i].Name < tests[j].Name
	})
	spec.HiddenTests = tests
	return nil
}

// ExecTimeoutOr returns the benchmark's declared exec_timeout, or def when it
// declares none. Every lane that executes candidate code calls this, so one
// declared budget applies to every language and mode.
func (s *BenchmarkSpec) ExecTimeoutOr(def time.Duration) time.Duration {
	if s != nil && s.ExecTimeout > 0 {
		return time.Duration(s.ExecTimeout) * time.Second
	}
	return def
}

// withHiddenTest returns a copy of the spec whose visible test is the hidden one,
// so every runner (which reads Stdin/ExpectedOut from the spec it holds) can run it
// unchanged.
func (s *BenchmarkSpec) withHiddenTest(t HiddenTest) *BenchmarkSpec {
	cp := *s
	cp.Stdin = t.Stdin
	cp.ExpectedOut = t.Expected
	cp.HiddenTests = nil
	return &cp
}

// GradeHidden runs the candidate against the hidden tests in order (smallest input
// first) and stops at the first failure of any kind, so grading costs at most one
// failing run beyond the passing ones. Passed is therefore the length of the
// passing prefix — a monotone progress signal, since later tests are no smaller.
//
// Nothing about a hidden test (its stdin, expected output, or the candidate's
// stdout/stderr on it) is returned: callers bank only the counts and FirstFail,
// and must never feed a hidden run's output into a repair prompt or the agent.
func GradeHidden(spec *BenchmarkSpec, run func(*BenchmarkSpec) (HiddenRun, error)) (HiddenResult, error) {
	res := HiddenResult{Total: len(spec.HiddenTests)}
	for _, t := range spec.HiddenTests {
		cp := spec.withHiddenTest(t)
		out, err := run(cp)
		if err != nil {
			return res, fmt.Errorf("hidden test %s: %w", t.Name, err)
		}
		switch {
		case out.TimedOut:
			res.FirstFail = t.Name + ":timeout"
		case !out.RuntimeOk:
			res.FirstFail = t.Name + ":runtime_error"
		case !GradeStdout(cp, out.Stdout, ""):
			res.FirstFail = t.Name + ":wrong_output"
		default:
			res.Passed++
			continue
		}
		return res, nil
	}
	return res, nil
}

// hiddenFailureCategory maps a hidden failure onto the categories a visible failure
// of the same kind banks: a crash or timeout is a runtime error, a wrong answer a
// logic error.
func hiddenFailureCategory(firstFail string) string {
	if strings.HasSuffix(firstFail, ":wrong_output") {
		return ErrorCategoryLogic
	}
	return ErrorCategoryRuntime
}
