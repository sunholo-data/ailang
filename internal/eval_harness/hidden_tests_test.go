package eval_harness

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/ai"
	"github.com/sunholo-data/ailang/internal/executor"
)

// M-EVAL-HIDDEN-TESTS. See design_docs/planned/v0_55_0/m-eval-hidden-tests.md.

func writeHiddenFixture(t *testing.T, files map[string]string, specExtra string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "data", "h"), 0o755); err != nil {
		t.Fatal(err)
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, "data", "h", name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	spec := `id: hidden_fixture
description: "hidden-test fixture"
languages: ["python"]
entrypoint: "main"
caps: ["IO"]
task_prompt: "Echo stdin doubled in <LANG>"
stdin: "1\n"
expected_stdout: "2"
hidden_tests_dir: data/h
` + specExtra
	path := filepath.Join(dir, "bench.yml")
	if err := os.WriteFile(path, []byte(spec), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadSpec_HiddenTests_OrderedBySizeThenName(t *testing.T) {
	path := writeHiddenFixture(t, map[string]string{
		"b.in": "300\n", "b.ans": "600\n",
		"a.in": "4000\n", "a.ans": "8000\n",
		"c.in": "7\n", "c.ans": "14\n",
	}, "exec_timeout: 7\n")
	spec, err := LoadSpec(path)
	if err != nil {
		t.Fatalf("LoadSpec: %v", err)
	}
	var names []string
	for _, h := range spec.HiddenTests {
		names = append(names, h.Name)
	}
	if got := strings.Join(names, ","); got != "c,b,a" {
		t.Errorf("order = %s, want c,b,a (smallest input first)", got)
	}
	if spec.HiddenTests[0].Stdin != "7\n" || spec.HiddenTests[0].Expected != "14\n" {
		t.Errorf("pair contents not loaded: %+v", spec.HiddenTests[0])
	}
	if got := spec.ExecTimeoutOr(30 * time.Second); got != 7*time.Second {
		t.Errorf("ExecTimeoutOr = %v, want 7s", got)
	}
}

func TestLoadSpec_HiddenTests_LoadErrors(t *testing.T) {
	cases := map[string]struct {
		files map[string]string
		extra string
		want  string
	}{
		"in without out": {map[string]string{"a.in": "1", "b.in": "2", "b.ans": "4"}, "", "a.in has no a.ans"},
		"out without in": {map[string]string{"a.ans": "1", "b.in": "2", "b.ans": "4"}, "", "a.ans has no a.in"},
		"empty dir":      {map[string]string{"README": "x"}, "", "no .in/.ans pairs"},
		"with quine":     {map[string]string{"a.in": "1", "a.ans": "2"}, "grading: quine\n", "cannot be combined"},
		"negative exec":  {map[string]string{"a.in": "1", "a.ans": "2"}, "exec_timeout: -1\n", "exec_timeout must be >= 0"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := LoadSpec(writeHiddenFixture(t, c.files, c.extra))
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("err = %v, want it to contain %q", err, c.want)
			}
		})
	}
}

func TestExecTimeoutOr_DefaultWhenUnset(t *testing.T) {
	if got := (&BenchmarkSpec{}).ExecTimeoutOr(10 * time.Second); got != 10*time.Second {
		t.Errorf("ExecTimeoutOr = %v, want the lane default 10s", got)
	}
}

func hiddenSpec(names ...string) *BenchmarkSpec {
	s := &BenchmarkSpec{ID: "h", Stdin: "visible", ExpectedOut: "visible-out"}
	for _, n := range names {
		s.HiddenTests = append(s.HiddenTests, HiddenTest{Name: n, Stdin: n + "-in", Expected: n + "-out"})
	}
	return s
}

func TestGradeHidden_AllPass(t *testing.T) {
	res, err := GradeHidden(hiddenSpec("a", "b", "c"), func(cp *BenchmarkSpec) (HiddenRun, error) {
		if len(cp.HiddenTests) != 0 {
			t.Error("spec copy must not carry the hidden tests on")
		}
		return HiddenRun{Stdout: strings.TrimSuffix(cp.Stdin, "-in") + "-out", RuntimeOk: true}, nil
	})
	if err != nil || !res.AllPassed() || res.Passed != 3 || res.FirstFail != "" {
		t.Fatalf("res = %+v, err = %v; want 3/3 with no failure", res, err)
	}
}

// Every failure kind stops the loop: the later, larger tests are never run.
func TestGradeHidden_StopsAtFirstFailure(t *testing.T) {
	for kind, fail := range map[string]HiddenRun{
		"wrong_output":  {Stdout: "nope", RuntimeOk: true},
		"runtime_error": {RuntimeOk: false},
		"timeout":       {TimedOut: true},
	} {
		t.Run(kind, func(t *testing.T) {
			var ran []string
			res, err := GradeHidden(hiddenSpec("a", "b", "c"), func(cp *BenchmarkSpec) (HiddenRun, error) {
				name := strings.TrimSuffix(cp.Stdin, "-in")
				ran = append(ran, name)
				if name == "b" {
					return fail, nil
				}
				return HiddenRun{Stdout: name + "-out", RuntimeOk: true}, nil
			})
			if err != nil {
				t.Fatal(err)
			}
			if res.Passed != 1 || res.Total != 3 || res.FirstFail != "b:"+kind {
				t.Errorf("res = %+v, want 1/3 with first failure b:%s", res, kind)
			}
			if strings.Join(ran, ",") != "a,b" {
				t.Errorf("ran %v, want a,b (c must not run)", ran)
			}
		})
	}
}

func TestHiddenFailureCategory(t *testing.T) {
	if got := hiddenFailureCategory("x:wrong_output"); got != ErrorCategoryLogic {
		t.Errorf("wrong_output -> %s, want %s", got, ErrorCategoryLogic)
	}
	for _, f := range []string{"x:timeout", "x:runtime_error"} {
		if got := hiddenFailureCategory(f); got != ErrorCategoryRuntime {
			t.Errorf("%s -> %s, want %s", f, got, ErrorCategoryRuntime)
		}
	}
}

// End to end through the agent-mode lane with a real Python run: a solution that
// passes the visible test but hard-codes it fails the hidden tests, and nothing a
// hidden test holds leaks into the banked stdout/stderr.
func TestValidateSolution_HiddenTests_Python(t *testing.T) {
	if _, err := resolveUv(); err != nil {
		t.Skipf("uv not available on this machine: %v", err)
	}
	spec, err := LoadSpec(writeHiddenFixture(t, map[string]string{
		"01.in": "21\n", "01.ans": "42\n",
		"02.in": "1000\n", "02.ans": "2000\n",
	}, ""))
	if err != nil {
		t.Fatal(err)
	}
	ok := &executor.Result{Success: true}

	for name, c := range map[string]struct {
		src       string
		passed    int
		firstFail string
	}{
		"correct":    {"print(int(input()) * 2)\n", 2, ""},
		"hard-coded": {"input()\nprint(2)\n", 0, "01:wrong_output"},
	} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			sol := filepath.Join(dir, "solution.py")
			if err := os.WriteFile(sol, []byte(c.src), 0o644); err != nil {
				t.Fatal(err)
			}
			v := validateSolution(ok, spec, dir, "python", sol)
			if v.Hidden.Total != 2 || v.Hidden.Passed != c.passed || v.Hidden.FirstFail != c.firstFail {
				t.Errorf("hidden = %+v, want %d/2 first fail %q", v.Hidden, c.passed, c.firstFail)
			}
			if want := c.firstFail == ""; v.StdoutOk != want {
				t.Errorf("StdoutOk = %v, want %v", v.StdoutOk, want)
			}
			if !v.RuntimeOk {
				t.Errorf("a wrong answer is a logic error, not a runtime error")
			}
			for _, secret := range []string{"21", "42", "1000", "2000"} {
				if strings.Contains(v.Stdout+v.Stderr, secret) {
					t.Errorf("hidden test content %q leaked into banked stdout/stderr", secret)
				}
			}
		})
	}
}

// The prompts and workspace the model sees must not contain any hidden test.
func TestHiddenTests_NeverReachPromptOrWorkspace(t *testing.T) {
	spec, err := LoadSpec(writeHiddenFixture(t, map[string]string{
		"01.in": "HIDDEN_STDIN_7781\n", "01.ans": "HIDDEN_EXPECTED_9934\n",
	}, ""))
	if err != nil {
		t.Fatal(err)
	}
	secrets := []string{"HIDDEN_STDIN_7781", "HIDDEN_EXPECTED_9934"}
	check := func(where, text string) {
		for _, s := range secrets {
			if strings.Contains(text, s) {
				t.Errorf("%s contains hidden test content %q", where, s)
			}
		}
	}

	for _, lang := range []string{"python", "ailang"} {
		check("standard prompt ("+lang+")", spec.PromptForLanguage(lang))
		check("agent prompt ("+lang+")", GenerateAgentPrompt(spec, DefaultAgentConfig(), "", lang))
	}

	ws := t.TempDir()
	if err := PrepareWorkspaceWithSyntax(ws, spec, ""); err != nil {
		t.Fatal(err)
	}
	_ = filepath.WalkDir(ws, func(p string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			b, _ := os.ReadFile(p)
			check("workspace file "+filepath.Base(p), string(b))
		}
		return nil
	})
}

// stdinRunner answers each run from the stdin of the spec it was built for, so a
// hidden test's runner can be told apart from the visible one's.
type stdinRunner struct {
	spec   *BenchmarkSpec
	answer func(stdin string) RunResult
}

func (r *stdinRunner) Run(string, time.Duration) (*RunResult, error) {
	out := r.answer(r.spec.Stdin)
	return &out, nil
}
func (r *stdinRunner) Language() string { return "python" }

// Standard mode: the hidden verdict lands on the attempt and the banked row, and
// the visible RunResult — which the repair prompt is built from — is untouched.
func TestStandardMode_HiddenTests(t *testing.T) {
	agent, _ := newTestAgent(&recordingProvider{}, ai.ProviderOpenAI)
	spec := hiddenSpec("a", "b")
	spec.Languages = []string{"python"}

	for name, c := range map[string]struct {
		answer    func(stdin string) RunResult
		stdoutOk  bool
		runtimeOk bool
		passed    int
		firstFail string
		category  string
	}{
		"all pass": {func(in string) RunResult {
			return RunResult{CompileOk: true, RuntimeOk: true, Stdout: strings.TrimSuffix(in, "-in") + "-out"}
		}, true, true, 2, "", ErrorCategoryNone},
		"hidden wrong": {func(in string) RunResult {
			return RunResult{CompileOk: true, RuntimeOk: true, Stdout: "visible-out"}
		}, false, true, 0, "a:wrong_output", ErrorCategoryLogic},
		"hidden timeout": {func(in string) RunResult {
			if in == "visible" {
				return RunResult{CompileOk: true, RuntimeOk: true, Stdout: "visible-out"}
			}
			return RunResult{CompileOk: true, RuntimeOk: false, TimedOut: true, Stderr: "secret " + in}
		}, false, false, 0, "a:timeout", ErrorCategoryRuntime},
	} {
		t.Run(name, func(t *testing.T) {
			visible := &stdinRunner{spec: spec, answer: func(in string) RunResult {
				if in == "visible" {
					return RunResult{CompileOk: true, RuntimeOk: true, Stdout: "visible-out"}
				}
				return c.answer(in)
			}}
			rr := NewRepairRunner(agent, visible, spec, time.Second, false)
			rr.newRunner = func(_ context.Context, _ string, cp *BenchmarkSpec, _ string) (LanguageRunner, error) {
				return &stdinRunner{spec: cp, answer: c.answer}, nil
			}
			attempt, err := rr.runSingleAttempt(context.Background(), "x")
			if err != nil {
				t.Fatal(err)
			}
			var m RunMetrics
			rr.populateMetrics(&m, attempt)
			if m.StdoutOk != c.stdoutOk || m.RuntimeOk != c.runtimeOk || m.ErrorCategory != c.category {
				t.Errorf("banked stdout_ok=%v runtime_ok=%v category=%s, want %v %v %s", m.StdoutOk, m.RuntimeOk, m.ErrorCategory, c.stdoutOk, c.runtimeOk, c.category)
			}
			if m.HiddenTotal != 2 || m.HiddenPassed != c.passed || m.HiddenFirstFail != c.firstFail {
				t.Errorf("banked hidden %d/%d %q, want %d/2 %q", m.HiddenPassed, m.HiddenTotal, m.HiddenFirstFail, c.passed, c.firstFail)
			}
			if strings.Contains(attempt.RunResult.Stderr+attempt.RunResult.Stdout, "-in") {
				t.Errorf("hidden run output reached the visible RunResult (repair prompt source): %+v", attempt.RunResult)
			}
		})
	}
}
