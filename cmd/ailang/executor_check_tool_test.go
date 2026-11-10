package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// M-V1-VERIFICATION-HONESTY — the hosted `ailang_check` tool
// (executor_tools/executor.ail), with and without a solver.
// Any host may lack z3 (the executor image had none until it moved to
// distroless/cc-debian12), so both solver states are tested.
//
// Mark's ruling on #2222: when the tool knows it has no solver, it runs the
// type check WITHOUT requesting verification (`ai-check --no-verify`), so a
// program that type-checks genuinely exits 0; ok is exit_code == 0 as it
// always was; `verification` is "unavailable" with a message written for a
// remote caller. With a solver, verification is requested and every outcome
// maps through with its real exit code.
//
// Second-round review: availability is probed on EVERY call (one tiny
// `ai-check --no-verify` on a fixed contract-free program) and never stored —
// a cached answer could silently switch verification off (a stale or corrupt
// "no", a probe failure cached as "no", a file inside the program sandbox).
// The quiet type-check-only path is taken only when the probe ran and
// positively reported "no solver"; a failed probe requests verification, so a
// missing solver is reported loudly.
//
// These tests run the tool function itself (`ailang run --entry ailangCheck`
// on executor_tools/executor.ail) against the built binary, with
// AILANG_Z3_PATH pointing at a fake solver ("present") or a missing file
// ("absent").

type checkToolRun struct {
	result  string         // show() of the returned Json
	log     map[string]any // the executor_call log line
	workDir string
}

func runCheckTool(t *testing.T, z3 string, code string) checkToolRun {
	t.Helper()
	requireShForFakeSolver(t) // the fake solver and the tool's process exec
	return runCheckToolWith(t, z3, code, t.TempDir(), buildAilang(t))
}

// runCheckToolWith runs the tool with an explicit work root and an explicit
// EXECUTOR_AILANG_BIN (the binary the tool execs; the tool itself always runs
// on the built binary).
func runCheckToolWith(t *testing.T, z3, code, work, execBin string) checkToolRun {
	t.Helper()
	bin := buildAilang(t)
	t.Setenv("AILANG_Z3_PATH", z3)
	t.Setenv("EXECUTOR_AILANG_BIN", execBin)
	t.Setenv("EXECUTOR_WORK_DIR", work)
	t.Setenv("AILANG_NO_CACHE", "1") // as docker/Dockerfile.executor-mcp runs it
	arg, _ := json.Marshal(code)
	stdout, stderr, rc := runAilangBin(t, bin, "run", "--quiet", "--caps", "Clock,Debug,Env,FS,Process,Rand",
		"--entry", "ailangCheck", "--args-json", string(arg), "executor_tools/executor.ail")
	if rc != 0 {
		t.Fatalf("tool run failed (exit %d)\nstdout:\n%s\nstderr:\n%s", rc, stdout, stderr)
	}
	run := checkToolRun{result: stdout, workDir: work}
	for _, line := range strings.Split(stderr, "\n") {
		if strings.Contains(line, `"event":"executor_call"`) {
			if err := json.Unmarshal([]byte(strings.TrimSpace(line)), &run.log); err != nil {
				t.Fatalf("log line is not JSON: %v\n%s", err, line)
			}
		}
	}
	if run.log == nil {
		t.Fatalf("no executor_call log line\nstderr:\n%s", stderr)
	}
	return run
}

func field(name, value string) string { return "{key: " + name + ", value: " + value + "}" }

const (
	toolTypeError    = "module main\nexport pure func f(x: int) -> int = x + \"a\"\n"
	toolNoContracts  = "module main\nexport pure func f(x: int) -> int = x + 1\n"
	toolContract     = "module main\nexport pure func half(x: int) -> int\n  ensures { result >= 0 }\n{ x - 100 }\n"
	toolRequiresOnly = "module main\nexport pure func safeDiv(a: int, b: int) -> int\n  requires { b != 0 }\n{ a / b }\n"
	noSolverMsg      = "contract verification is not available on this hosted tool; type checking passed"
)

// TestCheckTool_Matrix is the ruled matrix: {solver present, absent} ×
// {type error, no contracts, contract that verifies, contract with a
// counterexample, requires-only}. "present" is a fake solver answering unsat
// (the contract verifies) or sat (a counterexample).
//
// Red mutations, each measured: make solverAvailable return true always (the
// absent rows exit 1 as "unavailable"); return false always (the present
// rows report "unavailable" instead of verified / counterexample); render with
// the generic outputJson (no verification fields).
func TestCheckTool_Matrix(t *testing.T) {
	type row struct {
		name, solver, solverOut, code string
		ok                            bool
		exitCode, verification, log   string
		message                       bool
	}
	rows := []row{
		{"present/type error", "unsat", "unsat", toolTypeError, false, "1.0", "not_run", "check_failed", false},
		{"present/no contracts", "unsat", "unsat", toolNoContracts, true, "0.0", "no_obligations", "ok", false},
		{"present/verifies", "unsat", "unsat", toolContract, true, "0.0", "verified", "ok", false},
		{"present/counterexample", "sat", "sat", toolContract, false, "1.0", "counterexample", "verify_failed", false},
		{"present/requires-only", "unsat", "unsat", toolRequiresOnly, true, "0.0", "no_obligations", "ok", false},
		{"absent/type error", "missing", "", toolTypeError, false, "1.0", "not_run", "check_failed", false},
		{"absent/no contracts", "missing", "", toolNoContracts, true, "0.0", "no_obligations", "ok", false},
		{"absent/would verify", "missing", "", toolContract, true, "0.0", "unavailable", "ok_unverified", true},
		{"absent/would be a counterexample", "missing", "", toolContract, true, "0.0", "unavailable", "ok_unverified", true},
		{"absent/requires-only", "missing", "", toolRequiresOnly, true, "0.0", "no_obligations", "ok", false},
	}
	for _, r := range rows {
		t.Run(r.name, func(t *testing.T) {
			run := runCheckTool(t, fakeSolver(t, r.solver, r.solverOut, 0), r.code)
			okVal := "JBool(false)"
			if r.ok {
				okVal = "JBool(true)"
			}
			for _, want := range []string{field("ok", okVal), field("exit_code", "JNumber("+r.exitCode+")"), field("verification", "JString("+r.verification+")")} {
				if !strings.Contains(run.result, want) {
					t.Fatalf("result missing %s:\n%s", want, run.result)
				}
			}
			hasMsg := strings.Contains(run.result, field("verification_message", "JString("+noSolverMsg+")"))
			if hasMsg != r.message || (!r.message && strings.Contains(run.result, "verification_message")) {
				t.Fatalf("verification_message present=%v, want %v:\n%s", hasMsg, r.message, run.result)
			}
			// (third-round review) The tool checks a self-contained `module
			// main` under -relax-modules; it must never report its own file as
			// an unverified import.
			if strings.Contains(run.result, "imported_contracts_unverified") || strings.Contains(run.result, "imported modules were not verified") {
				t.Fatalf("a self-contained program reports unverified imports:\n%s", run.result)
			}
			for _, local := range []string{"Install Z3", "brew"} {
				if strings.Contains(run.result, local) {
					t.Fatalf("result tells a remote caller about local setup (%q):\n%s", local, run.result)
				}
			}
			if run.log["outcome"] != r.log {
				t.Fatalf("log outcome %v, want %s (log %v)", run.log["outcome"], r.log, run.log)
			}
		})
	}
}

// TestCheckTool_NothingPersisted: the tool stores no availability answer in
// its work root (the program sandbox root) or anywhere else: every call
// probes, and the request directories are removed. Red mutation: restore a
// cache file.
func TestCheckTool_NothingPersisted(t *testing.T) {
	for _, solver := range []string{"unsat", "missing"} {
		run := runCheckTool(t, fakeSolver(t, solver, solver, 0), toolContract)
		entries, err := os.ReadDir(run.workDir)
		if err != nil {
			t.Fatal(err)
		}
		if len(entries) != 0 {
			var names []string
			for _, e := range entries {
				names = append(names, e.Name())
			}
			t.Fatalf("solver=%s: the tool left %v in its work root", solver, names)
		}
	}
}

// TestCheckTool_SolverInstalledLaterIsUsed is the reviewer's stale.sh end to
// end: the solver path is empty on the first call (quiet type check), then a
// solver appears AT THE SAME PATH; a program with a counterexample must now
// fail with a real verification result, not a remembered "unavailable".
// Red mutation: cache a negative answer.
func TestCheckTool_SolverInstalledLaterIsUsed(t *testing.T) {
	requireShForFakeSolver(t)
	bin := buildAilang(t)
	work := t.TempDir()
	slot := filepath.Join(t.TempDir(), "z3")
	first := runCheckToolWith(t, slot, toolContract, work, bin)
	if !strings.Contains(first.result, field("verification", "JString(unavailable)")) || !strings.Contains(first.result, field("exit_code", "JNumber(0.0)")) {
		t.Fatalf("no solver yet: want the quiet unavailable lane:\n%s", first.result)
	}
	if err := os.WriteFile(slot, []byte("#!/bin/sh\nprintf '%s\\n' 'sat'\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	second := runCheckToolWith(t, slot, toolContract, work, bin)
	for _, want := range []string{field("ok", "JBool(false)"), field("exit_code", "JNumber(1.0)"), field("verification", "JString(counterexample)")} {
		if !strings.Contains(second.result, want) {
			t.Fatalf("solver installed at the same path: result missing %s:\n%s", want, second.result)
		}
	}
}

// probeFailingAilang writes an EXECUTOR_AILANG_BIN wrapper whose probe
// (`ai-check --no-verify`) prints probeOut and exits probeRC, while every
// other call runs the real binary.
func probeFailingAilang(t *testing.T, probeOut string, probeRC int) string {
	t.Helper()
	real := buildAilang(t)
	path := filepath.Join(t.TempDir(), "ailang-probe-fails")
	script := "#!/bin/sh\ncase \"$*\" in *--no-verify*) printf '%s\\n' '" + probeOut + "'; exit " + fmt.Sprint(probeRC) + ";; esac\nexec '" + real + "' \"$@\"\n"
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

// TestCheckTool_ProbeFailureIsNotNoSolver (review fix d; the surviving
// mutant M6): a probe that errors or prints garbage is not "no solver". The
// caller's check runs WITH verification requested: with a solver it
// verifies, and with none the caller gets the loud, truthful unavailable /
// exit 1 — never a quiet type-check-only pass. The loud lane names no host
// path, gives no install advice, and does not claim the check passed.
// Red mutation: treat a failed probe as "no solver".
func TestCheckTool_ProbeFailureIsNotNoSolver(t *testing.T) {
	requireShForFakeSolver(t)
	for _, probe := range []struct {
		name, out string
		rc        int
	}{
		{"garbage", "not json", 3},
		// Well-formed JSON that says "no solver", but from a probe that did
		// not run cleanly (exit 1, check failed): not a trustworthy answer.
		{"failed report", `{"check":{"passed":false},"verify":{"outcome":"not_run","available":false}}`, 1},
		// Each guard on its own: exit 0 but the probe's check failed; and a
		// clean-looking report from a probe that exited non-zero.
		{"exit 0, check failed", `{"check":{"passed":false},"verify":{"outcome":"not_run","available":false}}`, 0},
		{"clean report, exit 1", `{"check":{"passed":true},"verify":{"outcome":"no_obligations","available":false}}`, 1},
	} {
		wrapper := probeFailingAilang(t, probe.out, probe.rc)
		withSolver := runCheckToolWith(t, fakeSolver(t, "sat", "sat", 0), toolContract, t.TempDir(), wrapper)
		if !strings.Contains(withSolver.result, field("verification", "JString(counterexample)")) {
			t.Fatalf("%s probe, solver present: want verification to run:\n%s", probe.name, withSolver.result)
		}
		noSolver := runCheckToolWith(t, fakeSolver(t, "missing", "", 0), toolContract, t.TempDir(), wrapper)
		for _, want := range []string{field("ok", "JBool(false)"), field("exit_code", "JNumber(1.0)"), field("verification", "JString(unavailable)")} {
			if !strings.Contains(noSolver.result, want) {
				t.Fatalf("%s probe, no solver: result missing %s:\n%s", probe.name, want, noSolver.result)
			}
		}
		assertRemoteSafe(t, noSolver.result)
		if strings.Contains(noSolver.result, "type checking passed") {
			t.Fatalf("%s probe: verification_message claims success beside exit 1:\n%s", probe.name, noSolver.result)
		}
	}
}

// TestCheckTool_SolverThatCannotStartNamesNoHostPath (third-round review): a
// solver path that exists but cannot be started — a non-executable file or a
// directory — makes the CLI report `fork/exec <host path>: permission
// denied` in the result reason. The hosted tool keeps the outcome and exit
// code (error, exit 1) and replaces that reason with a fixed message, so no
// host path reaches a remote caller in any field. Red mutation: drop
// scrubSolverStart.
func TestCheckTool_SolverThatCannotStartNamesNoHostPath(t *testing.T) {
	requireShForFakeSolver(t)
	noExec := filepath.Join(t.TempDir(), "z3")
	if err := os.WriteFile(noExec, []byte("not a program"), 0o644); err != nil {
		t.Fatal(err)
	}
	dirSolver := filepath.Join(t.TempDir(), "z3dir")
	if err := os.MkdirAll(dirSolver, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, z3 := range []string{noExec, dirSolver} {
		run := runCheckTool(t, z3, toolContract)
		for _, want := range []string{field("ok", "JBool(false)"), field("exit_code", "JNumber(1.0)"), field("verification", "JString(error)"),
			"the verification solver could not be started"} {
			if !strings.Contains(run.result, want) {
				t.Fatalf("solver %s: result missing %q:\n%s", z3, want, run.result)
			}
		}
		for _, leak := range []string{z3, "fork/exec"} {
			if strings.Contains(run.result, leak) {
				t.Fatalf("solver %s: result leaks %q:\n%s", z3, leak, run.result)
			}
		}
		assertRemoteSafe(t, run.result)
	}
}

// assertRemoteSafe: nothing returned to a remote caller names a host path or
// gives local install advice.
func assertRemoteSafe(t *testing.T, result string) {
	t.Helper()
	for _, local := range []string{"Install Z3", "brew", "AILANG_Z3_PATH", "/opt/homebrew", os.TempDir()} {
		if strings.Contains(result, local) {
			t.Fatalf("result leaks local setup or a host path (%q):\n%s", local, result)
		}
	}
}
