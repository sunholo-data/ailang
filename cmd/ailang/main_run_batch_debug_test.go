package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/testutil"
)

const batchDebugProgram = `module main

import std/debug as Debug
import std/env (getArgs)
import std/io (exit)

export func main() -> () ! {IO, Env} =
  match getArgs() {
    [a] => let _ = Debug.log("DEBUG-${a}") in
           if a == "FIRST_FAIL" then exit(1) else ()
    _ => ()
  }
`

// batchSeverityProgram emits one message BELOW the --log-level threshold and one
// ABOVE it. Both halves are load-bearing: without the high-severity line this
// fixture could only assert an ABSENCE, which a batch path that flushes nothing
// at all satisfies just as well as a working filter.
const batchSeverityProgram = `module main

import std/debug as Debug

export func main() -> () =
  let _ = Debug.log("{\"severity\":\"DEBUG\",\"message\":\"LOW-SEVERITY-BATCH\"}") in
  Debug.log("{\"severity\":\"ERROR\",\"message\":\"HIGH-SEVERITY-BATCH\"}")
`

func runDebugFixture(t *testing.T, program string, flags []string, programArgs ...string) (string, int) {
	t.Helper()

	ailangBin := testutil.FindAilangBinary(t)
	tmpDir := t.TempDir()
	src := filepath.Join(tmpDir, "debug.ail")
	if err := os.WriteFile(src, []byte(program), 0o644); err != nil {
		t.Fatalf("write debug fixture: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cmdArgs := append([]string{"run"}, flags...)
	cmdArgs = append(cmdArgs, src)
	cmdArgs = append(cmdArgs, programArgs...)
	cmd := exec.CommandContext(ctx, ailangBin, cmdArgs...)
	out, err := cmd.CombinedOutput()
	if ctx.Err() != nil {
		t.Fatalf("debug fixture timed out; output: %s", out)
	}
	if err == nil {
		return string(out), 0
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("run %v: %v (output: %s)", cmdArgs, err, out)
	}
	return string(out), exitErr.ExitCode()
}

func TestBatchDebugOutput_TwoItemsAttributedWhenQuiet(t *testing.T) {
	out, code := runDebugFixture(t, batchDebugProgram,
		[]string{"--quiet", "--caps", "IO,Env", "--entry", "main", "--batch"}, "FIRST", "SECOND")
	if code != 0 {
		t.Fatalf("batch exit code = %d, want 0; output:\n%s", code, out)
	}
	for _, want := range []string{"[FIRST] DEBUG-FIRST", "[SECOND] DEBUG-SECOND"} {
		if !strings.Contains(out, want) {
			t.Errorf("quiet batch output missing attributable debug line %q:\n%s", want, out)
		}
	}
}

func TestSingleFileDebugOutput_RemainsUnlabelled(t *testing.T) {
	out, code := runDebugFixture(t, batchDebugProgram,
		[]string{"--quiet", "--caps", "IO,Env", "--entry", "main"}, "SINGLE")
	if code != 0 {
		t.Fatalf("single-file exit code = %d, want 0; output:\n%s", code, out)
	}
	foundExact := false
	for _, line := range strings.Split(strings.TrimSuffix(out, "\n"), "\n") {
		if line == "DEBUG-SINGLE" {
			foundExact = true
		} else if strings.Contains(line, "DEBUG-SINGLE") {
			t.Errorf("single-file debug output gained a prefix or suffix: %q\nfull output:\n%s", line, out)
		}
	}
	if !foundExact {
		t.Fatalf("single-file output has no exact unlabelled DEBUG-SINGLE line; output:\n%s", out)
	}
}

func TestBatchDebugOutput_FailingItemFlushesAndBatchContinues(t *testing.T) {
	out, code := runDebugFixture(t, batchDebugProgram,
		[]string{"--quiet", "--caps", "IO,Env", "--entry", "main", "--batch"}, "FIRST_FAIL", "SECOND")
	if code != 1 {
		t.Fatalf("batch exit code = %d, want 1; output:\n%s", code, out)
	}
	if !strings.Contains(out, "[FIRST_FAIL] DEBUG-FIRST_FAIL") {
		t.Errorf("failing first item's debug output was not flushed with attribution:\n%s", out)
	}
	if !strings.Contains(out, "[SECOND] DEBUG-SECOND") {
		t.Errorf("second item did not run and flush after first item failed:\n%s", out)
	}
}

// TestBatchDebugOutput_SeverityFilterApplies pins that a batch item's flush goes
// THROUGH the shared debugLogLevel filter rather than around it.
//
// The known-positive control is not optional here. As first written this test
// asserted only that the DEBUG-severity line was absent, and it SURVIVED the
// mutation that neuters the per-item flush entirely (measured: rc=0 with
// `if false { flushDebugOutput(effCtx, input) }`) — an absence is satisfied
// just as well by "the filter worked" as by "nothing was ever emitted". The
// high-severity assertion below is what makes the absence mean something: it
// fails if the flush is removed, so the two arms together can only pass when
// the flush ran AND the filter suppressed the low-severity line.
func TestBatchDebugOutput_SeverityFilterApplies(t *testing.T) {
	out, code := runDebugFixture(t, batchSeverityProgram,
		[]string{"--quiet", "--log-level", "warn", "--entry", "main", "--batch"}, "ONLY")
	if code != 0 {
		t.Fatalf("batch exit code = %d, want 0; output:\n%s", code, out)
	}
	// Known-positive control: ERROR (3) >= warn (2), so this line MUST survive
	// the filter — and it can only appear if the batch item flushed at all.
	if !strings.Contains(out, "[ONLY] ") || !strings.Contains(out, "HIGH-SEVERITY-BATCH") {
		t.Fatalf("instrument failure: the above-threshold line never reached stderr, "+
			"so the absence assertion below proves nothing:\n%s", out)
	}
	if strings.Contains(out, "LOW-SEVERITY-BATCH") {
		t.Errorf("DEBUG-severity message escaped --log-level warn filter:\n%s", out)
	}
}
