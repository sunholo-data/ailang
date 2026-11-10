package main

import (
	"encoding/json"
	"strings"
	"testing"
)

// M-V1-VERIFICATION-HONESTY review fix 3 — the hosted `ailang_check` tool
// (executor_tools/executor.ail) on a host with NO solver, which is what
// docker/Dockerfile.executor-mcp builds (distroless/static, no z3).
//
// Once ai-check fails the `unavailable` lane (exit 1), the tool's generic
// envelope would have turned every contracted program into ok:false with
// stderr telling a REMOTE caller to install z3, while its log still said
// outcome "ok". The tool now reports the type-check verdict (ok:true), keeps
// exit_code truthful, says verification did not run in two additive fields,
// replaces the local install advice with a remote message, and logs
// "ok_unverified".
//
// These tests run the tool function itself (`ailang run --entry ailangCheck`
// on executor_tools/executor.ail) against the built binary, with
// AILANG_Z3_PATH pointing at a missing file or a fake solver.

type checkToolRun struct {
	result string         // show() of the returned Json
	log    map[string]any // the executor_call log line
}

func runCheckTool(t *testing.T, z3 string, code string) checkToolRun {
	t.Helper()
	requireShForFakeSolver(t) // the fake solver and the tool's process exec
	bin := buildAilang(t)
	t.Setenv("AILANG_Z3_PATH", z3)
	t.Setenv("EXECUTOR_AILANG_BIN", bin)
	t.Setenv("EXECUTOR_WORK_DIR", t.TempDir())
	arg, _ := json.Marshal(code)
	stdout, stderr, rc := runAilangBin(t, bin, "run", "--quiet", "--caps", "Clock,Debug,Env,FS,Process,Rand",
		"--entry", "ailangCheck", "--args-json", string(arg), "executor_tools/executor.ail")
	if rc != 0 {
		t.Fatalf("tool run failed (exit %d)\nstdout:\n%s\nstderr:\n%s", rc, stdout, stderr)
	}
	var run checkToolRun
	run.result = stdout
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

const contractedProgram = "module main\nexport pure func half(x: int) -> int\n  ensures { result >= 0 }\n{ x - 100 }\n"

// TestCheckTool_NoSolverReportsTypeCheckAndNotRunVerification. Red
// mutation: render ailang_check with the generic outputJson — ok becomes false
// and the verification fields vanish.
func TestCheckTool_NoSolverReportsTypeCheckAndNotRunVerification(t *testing.T) {
	run := runCheckTool(t, fakeSolver(t, "missing", "", 0), contractedProgram)
	for _, want := range []string{
		field("ok", "JBool(true)"),
		field("exit_code", "JNumber(1.0)"),
		field("verification", "JString(unavailable)"),
		field("verification_message", "JString(contract verification is not available on this hosted tool; type checking passed)"),
	} {
		if !strings.Contains(run.result, want) {
			t.Fatalf("result missing %s:\n%s", want, run.result)
		}
	}
	for _, local := range []string{"Install Z3", "brew", "AILANG_Z3_PATH"} {
		if strings.Contains(run.result, local) {
			t.Fatalf("result tells a remote caller about local setup (%q):\n%s", local, run.result)
		}
	}
	if run.log["outcome"] != "ok_unverified" || run.log["verify_outcome"] != "unavailable" {
		t.Fatalf("log outcome %v / verify_outcome %v; want ok_unverified / unavailable", run.log["outcome"], run.log["verify_outcome"])
	}
}

// TestCheckTool_TypeErrorIsStillAFailure: the no-solver lane must not excuse
// a program that does not type-check.
func TestCheckTool_TypeErrorIsStillAFailure(t *testing.T) {
	run := runCheckTool(t, fakeSolver(t, "missing", "", 0), "module main\nexport pure func f(x: int) -> int = x + \"a\"\n")
	if !strings.Contains(run.result, field("ok", "JBool(false)")) || !strings.Contains(run.result, field("verification", "JString(not_run)")) ||
		strings.Contains(run.result, "verification_message") {
		t.Fatalf("type error under no solver:\n%s", run.result)
	}
	if run.log["outcome"] != "check_failed" {
		t.Fatalf("log outcome %v, want check_failed", run.log["outcome"])
	}
}

// TestCheckTool_CounterexampleIsLoggedAsVerifyFailed: with a solver that
// answers sat, the program type-checks but verification fails; the log used
// to say "ok".
func TestCheckTool_CounterexampleIsLoggedAsVerifyFailed(t *testing.T) {
	run := runCheckTool(t, fakeSolver(t, "sat", "sat", 0), contractedProgram)
	if !strings.Contains(run.result, field("ok", "JBool(false)")) || !strings.Contains(run.result, field("verification", "JString(counterexample)")) {
		t.Fatalf("counterexample result:\n%s", run.result)
	}
	if run.log["outcome"] != "verify_failed" {
		t.Fatalf("log outcome %v, want verify_failed", run.log["outcome"])
	}
}

// TestCheckTool_NoContractsNoSolverIsOK: nothing to verify, nothing missing.
func TestCheckTool_NoContractsNoSolverIsOK(t *testing.T) {
	run := runCheckTool(t, fakeSolver(t, "missing", "", 0), "module main\nexport pure func f(x: int) -> int = x + 1\n")
	if !strings.Contains(run.result, field("ok", "JBool(true)")) || !strings.Contains(run.result, field("exit_code", "JNumber(0.0)")) ||
		!strings.Contains(run.result, field("verification", "JString(no_obligations)")) {
		t.Fatalf("no-contract result:\n%s", run.result)
	}
	if run.log["outcome"] != "ok" {
		t.Fatalf("log outcome %v, want ok", run.log["outcome"])
	}
}
