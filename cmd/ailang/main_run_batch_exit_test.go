package main

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/testutil"
)

// batchExitProgram branches on its single batch input: BOOM calls exit(1),
// ZERO calls exit(0), anything else just prints. One compiled module can then
// play the failing item, the exit(0) item and the surviving item.
const batchExitProgram = `module main

import std/io (println, exit)
import std/env (getArgs)

export func main() -> () ! {IO, Env} =
  match getArgs() {
    [a] => if a == "BOOM" then let _ = println("BOOM: calling exit(1)") in exit(1)
           else if a == "ZERO" then let _ = println("ZERO: calling exit(0)") in exit(0)
           else println("item ran: ${a}")
    _ => println("no args")
  }
`

// runBatchExitFixture runs the fixture above in batch mode over the given
// inputs and returns (stdout+stderr, exit code).
func runBatchExitFixture(t *testing.T, inputs ...string) (string, int) {
	t.Helper()

	ailangBin := testutil.FindAilangBinary(t)

	tmpDir := t.TempDir()
	src := filepath.Join(tmpDir, "main.ail")
	if err := os.WriteFile(src, []byte(batchExitProgram), 0o644); err != nil {
		t.Fatalf("write program: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	args := append([]string{"run", "--caps", "IO,Env", "--entry", "main", "--batch", src}, inputs...)
	cmd := exec.CommandContext(ctx, ailangBin, args...)
	out, err := cmd.CombinedOutput()

	code := 0
	if err != nil {
		exitErr, ok := err.(*exec.ExitError)
		if !ok {
			t.Fatalf("run %v: %v (output: %s)", args, err, out)
		}
		code = exitErr.ExitCode()
	}
	if ctx.Err() != nil {
		t.Fatalf("batch run timed out; output: %s", out)
	}
	return string(out), code
}

// TestBatchMode_ExitInOneItemDoesNotKillRun pins #607.
//
// Before the fix, exit() inside a batch item raised the *eval.EvalExitCode
// sentinel through executeBatchItem — which had no recover — so the process
// died with rc=2 and a raw Go panic stack, and every later input was skipped.
// The reporter hit this on a 2,500-file PDF batch: one bad file aborted the
// whole job.
//
// This asserts the per-item-isolation contract that the "Batch complete: X/Y
// succeeded" summary already implies. Removing the recover in
// runBatchItemEntrypoint reds every arm below.
func TestBatchMode_ExitInOneItemDoesNotKillRun(t *testing.T) {
	out, code := runBatchExitFixture(t, "BOOM", "SECOND")

	// The defect's signature: a raw Go panic reaching the user.
	if strings.Contains(out, "panic:") || strings.Contains(out, "goroutine 1 [running]") {
		t.Errorf("batch run leaked a Go panic to the user (#607):\n%s", out)
	}
	if strings.Contains(out, "EvalExitCode") {
		t.Errorf("exit() sentinel escaped as a panic value (#607):\n%s", out)
	}

	// The item that called exit(1) must be reported as failed, not silently
	// dropped: the caller prints the error and counts it.
	if !strings.Contains(out, "exit(1)") {
		t.Errorf("failing item did not report its exit code; output:\n%s", out)
	}

	// The load-bearing assertion: the SECOND item still ran.
	if !strings.Contains(out, "[2/2]") {
		t.Errorf("batch stopped after the failing item — [2/2] never started:\n%s", out)
	}
	if !strings.Contains(out, "item ran: SECOND") {
		t.Errorf("second batch item produced no output — it never executed:\n%s", out)
	}
	if !strings.Contains(out, "Batch complete: 1/2 succeeded") {
		t.Errorf("batch summary missing or miscounted; want 1/2 succeeded:\n%s", out)
	}

	// A batch with a failed item exits non-zero, but cleanly (1, not the 2 a
	// Go panic produces).
	if code != 1 {
		t.Errorf("exit code = %d, want 1 (clean failure, not a panic's 2):\n%s", code, out)
	}
}

// TestBatchMode_ExitCodeZeroCountsAsSuccess pins the other half of the
// contract: exit(0) is a successful item, not a failed one, and likewise does
// not abort the run. The first input really does call exit(0) — asserted via
// its own stdout line, so this cannot pass on a program that never exits.
func TestBatchMode_ExitCodeZeroCountsAsSuccess(t *testing.T) {
	out, code := runBatchExitFixture(t, "ZERO", "SECOND")

	if strings.Contains(out, "panic:") {
		t.Errorf("exit(0) batch leaked a panic:\n%s", out)
	}
	// Proof the exit(0) arm was actually taken, not the println arm.
	if !strings.Contains(out, "ZERO: calling exit(0)") {
		t.Fatalf("fixture never reached the exit(0) arm — test is vacuous:\n%s", out)
	}
	if !strings.Contains(out, "item ran: SECOND") {
		t.Errorf("exit(0) aborted the batch — second item never ran:\n%s", out)
	}
	if !strings.Contains(out, "Batch complete: 2/2 succeeded") {
		t.Errorf("exit(0) item was not counted as a success; want 2/2:\n%s", out)
	}
	if code != 0 {
		t.Errorf("exit(0) batch exit code = %d, want 0:\n%s", code, out)
	}
}

// TestBatchMode_CleanRunUnaffected is the negative control for the recover:
// a batch where no item exits must behave exactly as before the fix.
func TestBatchMode_CleanRunUnaffected(t *testing.T) {
	out, code := runBatchExitFixture(t, "FIRST", "SECOND")

	if strings.Contains(out, "panic:") {
		t.Errorf("clean batch leaked a panic:\n%s", out)
	}
	if !strings.Contains(out, "Batch complete: 2/2 succeeded") {
		t.Errorf("clean batch did not report 2/2; output:\n%s", out)
	}
	if code != 0 {
		t.Errorf("clean batch exit code = %d, want 0:\n%s", code, out)
	}
}

// TestRecoverBatchItemExit_Branches pins every branch of the recover directly,
// including the re-panic arm, which no .ail fixture can reach deterministically
// (it needs a genuine Go crash inside the evaluator). Without this, that branch
// would ship unguarded — a batch item that really crashes must stay loud rather
// than being silently downgraded to "this item failed".
func TestRecoverBatchItemExit_Branches(t *testing.T) {
	t.Run("no panic passes the error through unchanged", func(t *testing.T) {
		want := errors.New("ordinary failure")
		if got := recoverBatchItemExit(func() error { return want }); !errors.Is(got, want) {
			t.Errorf("err = %v, want %v", got, want)
		}
		if got := recoverBatchItemExit(func() error { return nil }); got != nil {
			t.Errorf("err = %v, want nil", got)
		}
	})

	t.Run("non-zero exit becomes a per-item error", func(t *testing.T) {
		got := recoverBatchItemExit(func() error {
			panic(&eval.EvalExitCode{Code: 3})
		})
		if got == nil {
			t.Fatal("exit(3) produced no error — the item would count as a success")
		}
		if !strings.Contains(got.Error(), "exit(3)") {
			t.Errorf("err = %q, want it to name exit(3)", got.Error())
		}
	})

	t.Run("exit zero is a success", func(t *testing.T) {
		if got := recoverBatchItemExit(func() error {
			panic(&eval.EvalExitCode{Code: 0})
		}); got != nil {
			t.Errorf("exit(0) err = %v, want nil", got)
		}
	})

	t.Run("a real crash is re-panicked, not swallowed", func(t *testing.T) {
		defer func() {
			r := recover()
			if r == nil {
				t.Fatal("a non-exit panic was swallowed — genuine crashes would be " +
					"silently reported as a failed batch item")
			}
			if s, ok := r.(string); !ok || s != "genuine crash" {
				t.Errorf("re-panicked value = %v, want the original panic", r)
			}
		}()
		_ = recoverBatchItemExit(func() error { panic("genuine crash") })
	})
}

// TestSingleFileRun_ExitStaysClean is the path-specificity control: the
// single-file path already recovered the sentinel, and must keep doing so.
// If this ever reds alongside the batch tests, the regression is in the shared
// exit() mechanism rather than in batch mode.
func TestSingleFileRun_ExitStaysClean(t *testing.T) {
	ailangBin := testutil.FindAilangBinary(t)

	tmpDir := t.TempDir()
	src := filepath.Join(tmpDir, "main.ail")
	if err := os.WriteFile(src, []byte(batchExitProgram), 0o644); err != nil {
		t.Fatalf("write program: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, ailangBin, "run", "--caps", "IO,Env", "--entry", "main", src, "BOOM")
	out, err := cmd.CombinedOutput()

	code := 0
	if err != nil {
		exitErr, ok := err.(*exec.ExitError)
		if !ok {
			t.Fatalf("run: %v (output: %s)", err, out)
		}
		code = exitErr.ExitCode()
	}
	if strings.Contains(string(out), "panic:") {
		t.Errorf("single-file exit() leaked a panic:\n%s", out)
	}
	if code != 1 {
		t.Errorf("single-file exit(1) code = %d, want 1:\n%s", code, out)
	}
}
