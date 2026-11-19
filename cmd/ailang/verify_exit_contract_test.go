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
	trailing    []string // flags placed AFTER the file
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
	// Review fix 1: obligations exist and NONE was verified — nothing proved,
	// so it fails as "unproved" (it passed as "partial" in the first cut), with
	// or without --strict.
	{name: "all skipped is unproved", fixture: "effectful.ail", solver: "unsat", solverOut: "unsat", wantCode: 1, wantOutcome: "unproved",
		wantStderr: "1 contract(s) skipped and none verified (verify.outcome: unproved): g (Function \"g\" has effects)"},
	{name: "all skipped is unproved, strict", fixture: "effectful.ail", solver: "unsat", solverOut: "unsat", extra: []string{"--strict"},
		wantCode: 1, wantOutcome: "unproved"},
	// The reviewer's probe: a FALSE contract the verifier skips. origin/dev
	// exited 1 (as a verifier error), the first cut exited 0.
	{name: "false contract, skipped", fixture: "skiplist.ail", solver: "unsat", solverOut: "unsat", wantCode: 1, wantOutcome: "unproved",
		wantStderr: "len (unsupported construct"},
	// A literal `ensures { true }` proves nothing: it does not count as the
	// "at least one verified" that would make this file partial (exit 0).
	// Red mutation: drop the isTriviallyTrue test in hasObligation.
	{name: "ensures true does not rescue a skip", fixture: "trueplusskip.ail", solver: "unsat", solverOut: "unsat", wantCode: 1, wantOutcome: "unproved"},
	// At least one verified + at least one skipped passes, loudly.
	{name: "partial passes loudly", fixture: "mixed.ail", solver: "unsat", solverOut: "unsat", wantCode: 0, wantOutcome: "partial",
		wantStderr: "1 contract(s) not verified (pass --strict to fail on these): g (Function \"g\" has effects)"},
	{name: "partial fails under --strict", fixture: "mixed.ail", solver: "unsat", solverOut: "unsat", extra: []string{"--strict"},
		wantCode: 1, wantOutcome: "partial", wantStderr: "--strict fails this run"},
	// Flags after the file are honoured (they used to be dropped silently).
	{name: "trailing --strict", fixture: "mixed.ail", solver: "unsat", solverOut: "unsat", trailing: []string{"--strict"},
		wantCode: 1, wantOutcome: "partial", wantStderr: "--strict fails this run"},
	// Review fix 2: a requires-only function is not an obligation — the same
	// verdict with and without a solver, and it passes --strict.
	{name: "requires-only", fixture: "requiresonly.ail", solver: "unsat", solverOut: "unsat", wantCode: 0, wantOutcome: "no_obligations"},
	{name: "requires-only, strict", fixture: "requiresonly.ail", solver: "unsat", solverOut: "unsat", extra: []string{"--strict"},
		wantCode: 0, wantOutcome: "no_obligations"},
	{name: "requires-only, no solver", fixture: "requiresonly.ail", solver: "missing", wantCode: 0, wantOutcome: "no_obligations"},
	// THE bug: a missing solver with contracts in the file used to exit 0 on
	// `check --verify` / `ai-check`, reporting `available: false` and nothing
	// else. The message names where it looked.
	{name: "solver unavailable", fixture: "half.ail", solver: "missing", wantCode: 1, wantOutcome: "unavailable",
		wantStderr: "Z3 solver not found (AILANG_Z3_PATH="},
	{name: "no obligations, no solver", fixture: "nocontracts.ail", solver: "missing", wantCode: 0, wantOutcome: "no_obligations"},
	{name: "no obligations", fixture: "nocontracts.ail", solver: "unsat", solverOut: "unsat", wantCode: 0, wantOutcome: "no_obligations"},
}

func runVerifySpelling(t *testing.T, bin string, sp verifySpelling, c verifyExitCase) (code int, outcome string, stdout, stderr string) {
	t.Helper()
	t.Setenv("AILANG_Z3_PATH", fakeSolver(t, c.solver, c.solverOut, c.solverRC))
	stdout, stderr, code = runAilangBin(t, bin, append(sp.args(c.extra, verifyExitFixtures+c.fixture), c.trailing...)...)
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
	// verify --json printed NO JSON on a type error; it now prints not_run.
	// Red mutation: drop the printVerifyJSONNotRun call.
	stdout, stderr, code := runAilangBin(t, bin, "verify", "--json", verifyExitFixtures+"typeerror.ail")
	v, err := flatVerify(stdout)
	if err != nil || code != 1 || v["outcome"] != "not_run" || v["compile_errors"] == nil {
		t.Fatalf("verify --json on a type error: exit %d, err %v, json %v\nstderr:\n%s", code, err, v, stderr)
	}
}

// TestVerifyExit_SecondFileIsAUsageError: `check --verify ok.ail bad.ail`
// verified ok.ail, exited 0 and never looked at bad.ail. A second file is now
// a usage error (exit 2) on every spelling. Red mutation: make
// verificationFileArg ignore extra arguments.
func TestVerifyExit_SecondFileIsAUsageError(t *testing.T) {
	requireShForFakeSolver(t)
	bin := buildAilang(t)
	t.Setenv("AILANG_Z3_PATH", fakeSolver(t, "unsat", "unsat", 0))
	for _, sp := range verifySpellings {
		args := append(sp.args(nil, verifyExitFixtures+"nocontracts.ail"), verifyExitFixtures+"skiplist.ail")
		_, stderr, code := runAilangBin(t, bin, args...)
		if code != 2 || !strings.Contains(stderr, "one file at a time") || !strings.Contains(stderr, "skiplist.ail") {
			t.Fatalf("%s with two files: exit %d, stderr %q; want exit 2 naming the unchecked file", sp.name, code, stderr)
		}
	}
}

// TestVerifyExit_PartialEnumeratesSkipsInJSON: a partial run lists each
// skipped obligation BY NAME in the JSON results (not only on stderr), so a
// machine consumer can see what was not proved.
func TestVerifyExit_PartialEnumeratesSkipsInJSON(t *testing.T) {
	requireShForFakeSolver(t)
	bin := buildAilang(t)
	t.Setenv("AILANG_Z3_PATH", fakeSolver(t, "unsat", "unsat", 0))
	for _, sp := range verifySpellings {
		stdout, stderr, code := runAilangBin(t, bin, sp.args(nil, verifyExitFixtures+"mixed.ail")...)
		v, err := sp.jsonVerify(stdout)
		if err != nil || code != 0 {
			t.Fatalf("%s: exit %d, err %v\n%s", sp.name, code, err, stderr)
		}
		status := map[string]string{}
		rows, _ := v["results"].([]any)
		for _, r := range rows {
			m, _ := r.(map[string]any)
			fn, _ := m["function"].(string)
			st, _ := m["status"].(string)
			status[fn] = st
		}
		if status["g"] != "skipped" || status["dbl"] != "verified" || v["obligations"] != float64(2) {
			t.Fatalf("%s: results %v, obligations %v; want g skipped and dbl verified by name, 2 obligations", sp.name, status, v["obligations"])
		}
	}
}

// TestVerifyExit_UnsupportedConstructIsASkipEverywhere: a contracted function
// matching on list patterns is outside the decidable fragment. `verify` always
// reported it skipped (#757); ai-check reported it as a verifier ERROR. Under
// the shared contract it is a skip on every spelling, and because it is the
// file's only obligation nothing was proved: "unproved", exit 1, strict or
// not, with the function listed on stderr.
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
			if strict {
				extra = append(extra, "--strict")
			}
			stdout, stderr, code := runAilangBin(t, bin, sp.args(extra, f)...)
			v, err := sp.jsonVerify(stdout)
			if err != nil {
				t.Fatalf("%s: invalid JSON (exit %d): %v\n%s", sp.name, code, err, stderr)
			}
			if code != 1 || v["outcome"] != "unproved" || v["skipped"] != float64(1) || v["errors"] != float64(0) {
				t.Fatalf("%s strict=%v: exit %d, verify %v; want exit 1, unproved, skipped 1, errors 0\nstderr:\n%s", sp.name, strict, code, v, stderr)
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
// path it looked at (not an assumed installer) and the obligation count.
func TestVerifyExit_HumanVerifyWithoutSolver(t *testing.T) {
	requireShForFakeSolver(t)
	bin := buildAilang(t)
	missing := fakeSolver(t, "missing", "", 0)
	t.Setenv("AILANG_Z3_PATH", missing)
	_, stderr, code := runAilangBin(t, bin, "verify", verifyExitFixtures+"nocontracts.ail")
	if code != 0 || !strings.Contains(stderr, "nothing needed it") {
		t.Fatalf("contract-free file without Z3: exit %d, stderr %q; want 0", code, stderr)
	}
	_, stderr, code = runAilangBin(t, bin, "verify", verifyExitFixtures+"half.ail")
	if code != 1 || !strings.Contains(stderr, "1 contract obligation(s) were not verified") ||
		!strings.Contains(stderr, "AILANG_Z3_PATH="+missing+" (no such file)") || strings.Contains(stderr, "brew") {
		t.Fatalf("contracted file without Z3: exit %d, stderr %q; want 1 naming AILANG_Z3_PATH and its value, no platform installer", code, stderr)
	}
}

// TestVerifyExit_AICheckNoVerify: `ai-check --no-verify` is the type check
// alone (the hosted ailang_check tool uses it when its host has no solver). A
// program that type-checks exits 0 whatever its contracts and whether or not
// a solver exists; the report says not_requested and still counts the
// obligations; a type error still fails; --strict / --allow-inconclusive with
// it would do nothing and are refused. Red mutation: ignore the flag — the
// contracted file exits 1 as "unavailable".
func TestVerifyExit_AICheckNoVerify(t *testing.T) {
	requireShForFakeSolver(t)
	bin := buildAilang(t)
	for _, solver := range []string{"missing", "unsat"} {
		t.Setenv("AILANG_Z3_PATH", fakeSolver(t, solver, solver, 0))
		for _, c := range []struct {
			fixture, outcome string
			code             int
			obligations      float64
		}{
			{"half.ail", "not_requested", 0, 1},
			{"nocontracts.ail", "no_obligations", 0, 0},
			{"requiresonly.ail", "no_obligations", 0, 0},
			{"typeerror.ail", "not_run", 1, 0},
		} {
			stdout, stderr, code := runAilangBin(t, bin, "ai-check", "--no-verify", verifyExitFixtures+c.fixture)
			v, err := nestedVerify(stdout)
			if err != nil || code != c.code || v["outcome"] != c.outcome || v["obligations"] != c.obligations {
				t.Fatalf("solver=%s %s: exit %d, verify %v (err %v); want exit %d, %s, %v obligations\nstderr:\n%s",
					solver, c.fixture, code, v, err, c.code, c.outcome, c.obligations, stderr)
			}
			if v["available"] != (solver != "missing") {
				t.Fatalf("solver=%s: available = %v; it must still say whether a solver was found", solver, v["available"])
			}
		}
	}
	_, stderr, code := runAilangBin(t, bin, "ai-check", "--no-verify", "--strict", verifyExitFixtures+"half.ail")
	if code != 2 || !strings.Contains(stderr, "--no-verify runs no verifier") {
		t.Fatalf("--no-verify --strict: exit %d, stderr %q; want a refusal", code, stderr)
	}
}

// TestVerifyExit_ContradictoryFlagsAreUsageErrors: flag combinations that
// contradict each other or would silently do nothing exit 2 with a message,
// never a silent precedence: `check --strict` without --verify, and
// --strict with --allow-inconclusive on every spelling (the second-round
// review). `ai-check --no-verify --strict` is pinned in
// TestVerifyExit_AICheckNoVerify. Red mutation: drop the
// refuseContradictoryPolicy call — --strict wins silently, exit 1.
func TestVerifyExit_ContradictoryFlagsAreUsageErrors(t *testing.T) {
	requireShForFakeSolver(t)
	bin := buildAilang(t)
	t.Setenv("AILANG_Z3_PATH", fakeSolver(t, "sat", "sat", 0))
	_, stderr, code := runAilangBin(t, bin, "check", "--strict", verifyExitFixtures+"nocontracts.ail")
	if code != 2 || !strings.Contains(stderr, "add --verify") {
		t.Fatalf("check --strict without --verify: exit %d, stderr %q; want exit 2", code, stderr)
	}
	for _, sp := range verifySpellings {
		_, stderr, code := runAilangBin(t, bin, sp.args([]string{"--strict", "--allow-inconclusive"}, verifyExitFixtures+"sumto.ail")...)
		if code != 2 || !strings.Contains(stderr, "contradict each other") {
			t.Fatalf("%s --strict --allow-inconclusive: exit %d, stderr %q; want exit 2", sp.name, code, stderr)
		}
	}
}

// TestVerifyExit_ImportedContractsAreNamed: contracts in imported modules are
// not verified (unchanged behaviour: no_obligations, exit 0), and every
// spelling now says so on stderr and in imported_contracts_unverified. A file
// with no such import carries neither. Red mutation: make
// importedContractModules return nil.
func TestVerifyExit_ImportedContractsAreNamed(t *testing.T) {
	requireShForFakeSolver(t)
	bin := buildAilang(t)
	t.Setenv("AILANG_Z3_PATH", fakeSolver(t, "unsat", "unsat", 0))
	const lib = "cmd/ailang/testdata/verify_exit/half"
	for _, sp := range verifySpellings {
		stdout, stderr, code := runAilangBin(t, bin, sp.args(nil, verifyExitFixtures+"importer.ail")...)
		v, err := sp.jsonVerify(stdout)
		if err != nil || code != 0 || v["outcome"] != "no_obligations" {
			t.Fatalf("%s: exit %d, verify %v (err %v)\nstderr:\n%s", sp.name, code, v, err, stderr)
		}
		mods, _ := v["imported_contracts_unverified"].([]any)
		if len(mods) != 1 || mods[0] != lib {
			t.Fatalf("%s: imported_contracts_unverified = %v, want [%s]", sp.name, v["imported_contracts_unverified"], lib)
		}
		if !strings.Contains(stderr, "contracts in imported modules were not verified") || !strings.Contains(stderr, lib) {
			t.Fatalf("%s: stderr does not name the unverified import:\n%s", sp.name, stderr)
		}
		stdout, stderr, _ = runAilangBin(t, bin, sp.args(nil, verifyExitFixtures+"half.ail")...)
		if strings.Contains(stdout, "imported_contracts_unverified") || strings.Contains(stderr, "imported modules") {
			t.Fatalf("%s: a file without such imports mentions them:\n%s\n%s", sp.name, stdout, stderr)
		}
	}
}

// importedList runs one spelling and returns imported_contracts_unverified
// and whether stderr carried the imports note.
func importedList(t *testing.T, bin string, sp verifySpelling, extra []string, file string) ([]string, bool) {
	t.Helper()
	stdout, stderr, code := runAilangBin(t, bin, sp.args(extra, file)...)
	v, err := sp.jsonVerify(stdout)
	if err != nil {
		t.Fatalf("%s %s: invalid JSON (exit %d): %v\n%s", sp.name, file, code, err, stderr)
	}
	var mods []string
	raw, _ := v["imported_contracts_unverified"].([]any)
	for _, m := range raw {
		s, _ := m.(string)
		mods = append(mods, s)
	}
	return mods, strings.Contains(stderr, "contracts in imported modules were not verified")
}

// TestVerifyExit_ImportedContractsIdentity (third-round review): the checked
// file is never listed as its own import. Under -relax-modules a file
// `r/main.ail` declaring `module main` used to report
// imported_contracts_unverified: ["r/main"] — the declared module name and
// the pipeline's key for the entry module differ, and the hosted tool runs
// exactly that shape on every call. The entry module is now excluded by the
// key the pipeline used for it, and "imported" means reached through
// imports, transitively. Red mutation: compare against the declared module
// name again.
func TestVerifyExit_ImportedContractsIdentity(t *testing.T) {
	requireShForFakeSolver(t)
	bin := buildAilang(t)
	t.Setenv("AILANG_Z3_PATH", fakeSolver(t, "unsat", "unsat", 0))
	const half = "cmd/ailang/testdata/verify_exit/half"
	dir := filepath.Join(t.TempDir(), "r")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	selfOnly := filepath.Join(dir, "main.ail")
	if err := os.WriteFile(selfOnly, []byte("module main\n\nexport pure func half(x: int) -> int\n  requires { x >= 0 }\n  ensures { result >= 0 }\n{ x / 2 }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	relaxImporter := filepath.Join(t.TempDir(), "main.ail")
	if err := os.WriteFile(relaxImporter, []byte("module main\n\nimport "+half+" (half)\n\nexport pure func quarter(x: int) -> int = half(half(x))\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	relax := []string{"--relax-modules"}
	for _, sp := range verifySpellings {
		for _, c := range []struct {
			name  string
			extra []string
			file  string
			want  []string
		}{
			{"(a) relax-modules self-contained file with contracts", relax, selfOnly, nil},
			{"(b) strict-module self-contained file", nil, verifyExitFixtures + "half.ail", nil},
			{"(c) real import, strict", nil, verifyExitFixtures + "importer.ail", []string{half}},
			{"(c) real import, relax-modules", relax, relaxImporter, []string{half}},
			{"(d) transitive import A -> B -> C", nil, verifyExitFixtures + "transitive.ail", []string{half}},
		} {
			got, note := importedList(t, bin, sp, c.extra, c.file)
			if strings.Join(got, ",") != strings.Join(c.want, ",") || note != (len(c.want) > 0) {
				t.Fatalf("%s %s: imported_contracts_unverified = %v (note %v), want %v", sp.name, c.name, got, note, c.want)
			}
		}
	}
}

// TestVerifyExit_AllowInconclusiveAlone: a file whose ONLY obligation is
// inconclusive exits 0 under --allow-inconclusive on every spelling — an
// explicit opt-in that may mean nothing was proved — and the outcome says
// so: "inconclusive", never "verified" or "partial".
func TestVerifyExit_AllowInconclusiveAlone(t *testing.T) {
	requireShForFakeSolver(t)
	bin := buildAilang(t)
	t.Setenv("AILANG_Z3_PATH", fakeSolver(t, "sat", "sat", 0))
	for _, sp := range verifySpellings {
		stdout, stderr, code := runAilangBin(t, bin, sp.args([]string{"--allow-inconclusive"}, verifyExitFixtures+"sumto.ail")...)
		v, err := sp.jsonVerify(stdout)
		if err != nil || code != 0 || v["outcome"] != "inconclusive" || v["verified"] != float64(0) {
			t.Fatalf("%s: exit %d, verify %v (err %v); want exit 0, outcome inconclusive, 0 verified\nstderr:\n%s", sp.name, code, v, err, stderr)
		}
	}
}
