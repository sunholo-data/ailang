package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// M-V1-VERIFICATION-HONESTY M1 — the shared verification exit contract, end to
// end, on every command spelling.
//
// Before this, `AILANG_Z3_PATH=/nonexistent ailang check --verify` exited 0 on
// a function that violates its own ensures (it had proved nothing), ai-check
// and verify passed an inconclusive obligation, and the three spellings
// disagreed on a missing solver and on an unsupported construct. Every
// spelling now exits through verification.ExitCode.
//
// HERMETIC: the solver is a fake shell script selected with AILANG_Z3_PATH, so
// these tests run (and must PASS) on a machine with no Z3. Only the answer the
// solver prints matters to the lane: unsat → verified, sat → counterexample
// (inconclusive on a bounded-unrolled recursive function), unknown → error,
// "timeout" → error, a crash → error, no binary → unavailable.

const verifyExitFixtures = "cmd/ailang/testdata/verify_exit/"

// fakeSolver writes an executable sh script that prints out (and exits rc)
// and returns its path. "" returns a path that does not exist.
func fakeSolver(t *testing.T, name, out string, rc int) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if name == "missing" {
		return path
	}
	script := "#!/bin/sh\n"
	if out != "" {
		script += fmt.Sprintf("printf '%%s\\n' '%s'\n", out)
	}
	script += fmt.Sprintf("exit %d\n", rc)
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func requireShForFakeSolver(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the fake solvers are sh scripts; Windows cannot exec them")
	}
}

// verifySpelling is one way to ask for contract verification.
type verifySpelling struct {
	name string
	args func(extra []string, file string) []string
	// jsonVerify extracts the verify object from stdout.
	jsonVerify func(stdout string) (map[string]any, error)
}

func nestedVerify(stdout string) (map[string]any, error) {
	var p struct {
		Verify map[string]any `json:"verify"`
	}
	if err := json.Unmarshal([]byte(stdout), &p); err != nil {
		return nil, err
	}
	return p.Verify, nil
}

func flatVerify(stdout string) (map[string]any, error) {
	var m map[string]any
	err := json.Unmarshal([]byte(stdout), &m)
	return m, err
}

var verifySpellings = []verifySpelling{
	{"check --verify", func(extra []string, f string) []string {
		return append(append([]string{"check", "--verify"}, extra...), f)
	}, nestedVerify},
	{"ai-check", func(extra []string, f string) []string {
		return append(append([]string{"ai-check"}, extra...), f)
	}, nestedVerify},
	{"verify --json", func(extra []string, f string) []string {
		return append(append([]string{"verify", "--json"}, extra...), f)
	}, flatVerify},
}

type verifyExitCase struct {
	name        string
	fixture     string
	solver      string // fakeSolver name
	solverOut   string
	solverRC    int
	extra       []string
	wantCode    int
	wantOutcome string // "" when no JSON verify object is expected (check failed under `verify`)
	wantStderr  string // substring every spelling must print on stderr ("" = no requirement)
}

var verifyExitCases = []verifyExitCase{
	{name: "verified", fixture: "half.ail", solver: "unsat", solverOut: "unsat", wantCode: 0, wantOutcome: "verified"},
	{name: "counterexample", fixture: "half.ail", solver: "sat", solverOut: "sat", wantCode: 1, wantOutcome: "counterexample"},
	{name: "solver error", fixture: "half.ail", solver: "crash", solverRC: 1, wantCode: 1, wantOutcome: "error"},
	{name: "solver unknown", fixture: "half.ail", solver: "unknown", solverOut: "unknown", wantCode: 1, wantOutcome: "error"},
	{name: "solver timeout", fixture: "half.ail", solver: "timeout", solverOut: "timeout", wantCode: 1, wantOutcome: "error"},
	{name: "inconclusive", fixture: "sumto.ail", solver: "sat", solverOut: "sat", wantCode: 1, wantOutcome: "inconclusive",
		wantStderr: "1 contract(s) inconclusive (not proved; fails this run): sumTo"},
	{name: "inconclusive allowed", fixture: "sumto.ail", solver: "sat", solverOut: "sat", extra: []string{"--allow-inconclusive"},
		wantCode: 0, wantOutcome: "inconclusive", wantStderr: "accepted by --allow-inconclusive"},
	{name: "skipped passes loudly", fixture: "effectful.ail", solver: "unsat", solverOut: "unsat", wantCode: 0, wantOutcome: "partial",
		wantStderr: "1 contract(s) not verified (pass --strict to fail on these): g (Function \"g\" has effects)"},
	{name: "skipped fails under --strict", fixture: "effectful.ail", solver: "unsat", solverOut: "unsat", extra: []string{"--strict"},
		wantCode: 1, wantOutcome: "partial", wantStderr: "--strict fails this run"},
	// THE bug: a missing solver with contracts in the file used to exit 0 on
	// `check --verify` / `ai-check`, reporting `available: false` and nothing else.
	{name: "solver unavailable", fixture: "half.ail", solver: "missing", wantCode: 1, wantOutcome: "unavailable",
		wantStderr: "Z3 solver not found: 1 contract obligation(s) were not verified"},
	{name: "no obligations, no solver", fixture: "nocontracts.ail", solver: "missing", wantCode: 0, wantOutcome: "no_obligations"},
	{name: "no obligations", fixture: "nocontracts.ail", solver: "unsat", solverOut: "unsat", wantCode: 0, wantOutcome: "no_obligations"},
}

func runVerifySpelling(t *testing.T, bin string, sp verifySpelling, c verifyExitCase) (code int, outcome string, stdout, stderr string) {
	t.Helper()
	t.Setenv("AILANG_Z3_PATH", fakeSolver(t, c.solver, c.solverOut, c.solverRC))
	stdout, stderr, code = runAilangBin(t, bin, sp.args(c.extra, verifyExitFixtures+c.fixture)...)
	if c.wantOutcome == "" {
		return code, "", stdout, stderr
	}
	v, err := sp.jsonVerify(stdout)
	if err != nil || v == nil {
		t.Fatalf("%s %s: no verify JSON (exit %d): %v\nstdout:\n%s\nstderr:\n%s", sp.name, c.fixture, code, err, stdout, stderr)
	}
	outcome, _ = v["outcome"].(string)
	return code, outcome, stdout, stderr
}

// TestVerifyExit_EveryOutcomeEverySpelling is the M1 acceptance table: each
// outcome × each spelling → exit code, JSON outcome, stderr notice.
//
// Red mutations, each measured: drop the `unavailable` arm of
// verification.Classify (the "solver unavailable" row exits 0 on every
// spelling); restore the old `aiCheckExitCode` in runAICheckReport (the
// inconclusive and unavailable rows fail on check --verify / ai-check).
func TestVerifyExit_EveryOutcomeEverySpelling(t *testing.T) {
	requireShForFakeSolver(t)
	bin := buildAilang(t)
	for _, c := range verifyExitCases {
		for _, sp := range verifySpellings {
			t.Run(c.name+"/"+sp.name, func(t *testing.T) {
				code, outcome, stdout, stderr := runVerifySpelling(t, bin, sp, c)
				if code != c.wantCode {
					t.Fatalf("exit = %d, want %d\nstdout:\n%s\nstderr:\n%s", code, c.wantCode, stdout, stderr)
				}
				if outcome != c.wantOutcome {
					t.Fatalf("outcome = %q, want %q\nstdout:\n%s", outcome, c.wantOutcome, stdout)
				}
				if c.wantStderr != "" && !strings.Contains(stderr, c.wantStderr) {
					t.Fatalf("stderr missing %q:\n%s", c.wantStderr, stderr)
				}
			})
		}
	}
}

// TestVerifyExit_SpellingsAgree is the cross-command parity guard: for every
// case, all spellings return the same exit code and outcome. It is the test
// that fails if one command grows a private exit rule again (red mutation:
// give `verify` back its old `Counterexample > 0 || Errors > 0` exit — the
// inconclusive row disagrees).
func TestVerifyExit_SpellingsAgree(t *testing.T) {
	requireShForFakeSolver(t)
	bin := buildAilang(t)
	for _, c := range verifyExitCases {
		t.Run(c.name, func(t *testing.T) {
			type got struct {
				code    int
				outcome string
			}
			seen := map[string]got{}
			for _, sp := range verifySpellings {
				code, outcome, _, _ := runVerifySpelling(t, bin, sp, c)
				seen[sp.name] = got{code, outcome}
			}
			first := seen[verifySpellings[0].name]
			for name, g := range seen {
				if g != first {
					t.Fatalf("spellings disagree: %v (first %s=%+v, %s=%+v)", seen, verifySpellings[0].name, first, name, g)
				}
			}
		})
	}
}

// TestVerifyExit_CheckFailed: a file that does not type-check never reaches
// the verifier. check --verify / ai-check report outcome not_run and keep
// `available` meaning "solver found" (it used to read false, which looked like
// "no solver"); verify prints the compile error. All exit 1.
func TestVerifyExit_CheckFailed(t *testing.T) {
	requireShForFakeSolver(t)
	bin := buildAilang(t)
	t.Setenv("AILANG_Z3_PATH", fakeSolver(t, "unsat", "unsat", 0))
	for _, sp := range verifySpellings[:2] {
		stdout, stderr, code := runAilangBin(t, bin, sp.args(nil, verifyExitFixtures+"typeerror.ail")...)
		v, err := nestedVerify(stdout)
		if err != nil {
			t.Fatalf("%s: invalid JSON (exit %d): %v\n%s", sp.name, code, err, stderr)
		}
		if code != 1 || v["outcome"] != "not_run" || v["available"] != true {
			t.Fatalf("%s: exit %d, verify %v; want exit 1, outcome not_run, available true", sp.name, code, v)
		}
	}
	_, stderr, code := runAilangBin(t, bin, "verify", verifyExitFixtures+"typeerror.ail")
	if code != 1 || !strings.Contains(stderr, "compilation failed") {
		t.Fatalf("verify on a type error: exit %d, stderr %q", code, stderr)
	}
}

// TestVerifyExit_UnsupportedConstructIsASkipEverywhere: a contracted function
// matching on list patterns is outside the decidable fragment. `verify` always
// reported it skipped (#757); ai-check reported it as a verifier ERROR and
// exited 1. Under the shared contract it is a skip on every spelling: exit 0,
// listed on stderr, and --strict fails it.
func TestVerifyExit_UnsupportedConstructIsASkipEverywhere(t *testing.T) {
	requireShForFakeSolver(t)
	bin := buildAilang(t)
	src := `module m

export func sumList(ls: [int]) -> int ! {}
ensures { result >= 0 }
{
  match ls {
    [] => 0,
    x :: rest => x + sumList(rest)
  }
}
`
	f := filepath.Join(t.TempDir(), "m.ail")
	if err := os.WriteFile(f, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AILANG_Z3_PATH", fakeSolver(t, "unsat", "unsat", 0))
	for _, sp := range verifySpellings {
		for _, strict := range []bool{false, true} {
			extra := []string{"--relax-modules"}
			want := 0
			if strict {
				extra = append(extra, "--strict")
				want = 1
			}
			stdout, stderr, code := runAilangBin(t, bin, sp.args(extra, f)...)
			v, err := sp.jsonVerify(stdout)
			if err != nil {
				t.Fatalf("%s: invalid JSON (exit %d): %v\n%s", sp.name, code, err, stderr)
			}
			if code != want || v["outcome"] != "partial" || v["skipped"] != float64(1) || v["errors"] != float64(0) {
				t.Fatalf("%s strict=%v: exit %d, verify %v; want exit %d, partial, skipped 1, errors 0\nstderr:\n%s", sp.name, strict, code, v, want, stderr)
			}
			if strings.Contains(stdout+stderr, "*core.ListPattern") {
				t.Fatalf("%s leaked the internal pattern type name:\n%s", sp.name, stdout+stderr)
			}
			if !strings.Contains(stderr, "sumList (unsupported construct") {
				t.Fatalf("%s: stderr does not list the skipped function:\n%s", sp.name, stderr)
			}
		}
	}
}

// TestVerifyExit_HumanVerifyWithoutSolver: the human `verify` path (no
// --json) compiles BEFORE looking for the solver, so a contract-free file
// passes without Z3 (it used to exit 1) and a contracted one fails naming the
// install command and the obligation count.
func TestVerifyExit_HumanVerifyWithoutSolver(t *testing.T) {
	requireShForFakeSolver(t)
	bin := buildAilang(t)
	t.Setenv("AILANG_Z3_PATH", fakeSolver(t, "missing", "", 0))
	_, stderr, code := runAilangBin(t, bin, "verify", verifyExitFixtures+"nocontracts.ail")
	if code != 0 || !strings.Contains(stderr, "nothing needed it") {
		t.Fatalf("contract-free file without Z3: exit %d, stderr %q; want 0", code, stderr)
	}
	_, stderr, code = runAilangBin(t, bin, "verify", verifyExitFixtures+"half.ail")
	if code != 1 || !strings.Contains(stderr, "1 contract obligation(s) were not verified") || !strings.Contains(stderr, "brew install z3") {
		t.Fatalf("contracted file without Z3: exit %d, stderr %q; want 1 with the install hint", code, stderr)
	}
}

// TestVerifyExit_StrictNeedsVerify: `check --strict` alone would silently do
// nothing, so it is refused.
func TestVerifyExit_StrictNeedsVerify(t *testing.T) {
	bin := buildAilang(t)
	_, stderr, code := runAilangBin(t, bin, "check", "--strict", verifyExitFixtures+"nocontracts.ail")
	if code != 1 || !strings.Contains(stderr, "add --verify") {
		t.Fatalf("check --strict without --verify: exit %d, stderr %q; want a refusal", code, stderr)
	}
}
