package main

import (
	"encoding/json"
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
// maps through with its real exit code. Availability is decided by probing the
// ailang binary on a fixed contract-free program and cached in the work root,
// so an image that gains z3 starts verifying with no code change.
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
	bin := buildAilang(t)
	work := t.TempDir()
	t.Setenv("AILANG_Z3_PATH", z3)
	t.Setenv("EXECUTOR_AILANG_BIN", bin)
	t.Setenv("EXECUTOR_WORK_DIR", work)
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

// TestCheckTool_SolverProbeIsCached: availability is decided once per host
// configuration and cached in the work root ("yes"/"no"), keyed by the binary
// and the variables the CLI's Z3 resolution reads; it is not re-derived from
// a failed verification. Red mutation: skip the cache write — no file.
func TestCheckTool_SolverProbeIsCached(t *testing.T) {
	for _, tc := range []struct{ solver, out, want string }{{"unsat", "unsat", "yes"}, {"missing", "", "no"}} {
		run := runCheckTool(t, fakeSolver(t, tc.solver, tc.out, 0), toolNoContracts)
		matches, _ := filepath.Glob(filepath.Join(run.workDir, ".solver-available-*"))
		if len(matches) != 1 {
			t.Fatalf("solver=%s: want one cache file in the work root, got %v", tc.solver, matches)
		}
		data, err := os.ReadFile(matches[0])
		if err != nil || strings.TrimSpace(string(data)) != tc.want {
			t.Fatalf("solver=%s: cache = %q (%v), want %q", tc.solver, data, err, tc.want)
		}
	}
}
