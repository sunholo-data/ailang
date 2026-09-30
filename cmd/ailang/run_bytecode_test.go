package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCLI_RunBytecode_Arithmetic verifies that `ailang run --bytecode`
// compiles a simple .ail file through the bytecode pipeline and prints
// the VM result. This is the M2 acceptance test for M-BYTECODE-2D.
func TestCLI_RunBytecode_Arithmetic(t *testing.T) {
	tmpDir := t.TempDir()
	src := filepath.Join(tmpDir, "bcrun_arith.ail")
	if err := os.WriteFile(src, []byte("module test/bcrun_arith\n\nexport func main() -> int = 1 + 2 * 3\n"), 0644); err != nil {
		t.Fatalf("write src: %v", err)
	}

	stdout, stderr, exitCode := runCLI(t, "run", "--verbose", "--bytecode", "--relax-modules", src)
	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d\nstderr=%s", exitCode, stderr)
	}
	if !strings.Contains(stdout, "7") {
		t.Errorf("expected stdout to contain VM result 7, got %q\nstderr=%s", stdout, stderr)
	}
	// The "Running ... via bytecode VM" status line goes to stderr.
	if !strings.Contains(stderr, "via bytecode VM") {
		t.Errorf("expected stderr to mention bytecode VM run, got %q", stderr)
	}
}

// TestCLI_RunBytecode_TransparentBridge_OnEffect verifies that an effectful
// program runs under --bytecode by transparently bridging the EvalOnly entry
// function back to the tree-walking evaluator. This is the M-BYTECODE-2D M3
// acceptance test for the value bridge: there should be NO fallback warning
// — the VM dispatches into the evaluator via Interop on the call boundary.
func TestCLI_RunBytecode_TransparentBridge_OnEffect(t *testing.T) {
	tmpDir := t.TempDir()
	src := filepath.Join(tmpDir, "bcrun_bridge.ail")
	srcContent := `module test/bcrun_bridge

import std/io (println)

export func main() -> () ! {IO} = println("hello from bridge")
`
	if err := os.WriteFile(src, []byte(srcContent), 0644); err != nil {
		t.Fatalf("write src: %v", err)
	}

	stdout, stderr, exitCode := runCLI(t, "run", "--verbose", "--bytecode", "--caps", "IO", "--relax-modules", src)
	if exitCode != 0 {
		t.Fatalf("expected exit 0 (transparent bridge), got %d\nstderr=%s", exitCode, stderr)
	}
	if !strings.Contains(stdout, "hello from bridge") {
		t.Errorf("expected program output via bridge, got stdout=%q\nstderr=%s", stdout, stderr)
	}
	if !strings.Contains(stderr, "via bytecode VM") {
		t.Errorf("expected stderr to mention bytecode VM run, got %q", stderr)
	}
	// Crucially: NO "falling back to evaluator" message — the bridge runs
	// transparently inside the VM dispatch path, not as a top-level fallback.
	if strings.Contains(stderr, "falling back to evaluator") {
		t.Errorf("expected no fallback message under M3 bridge dispatch; got %q", stderr)
	}
}

// TestCLI_RunBytecode_DivByZero_ReportsLine verifies that a runtime error in
// the bytecode VM surfaces the source file:line of the offending statement
// (M-BYTECODE-2D milestone M4_LINEINFO).
func TestCLI_RunBytecode_DivByZero_ReportsLine(t *testing.T) {
	tmpDir := t.TempDir()
	src := filepath.Join(tmpDir, "divzero.ail")
	// Line 1: module
	// Line 2: blank
	// Line 3: export func main() -> int = 10 / 0
	srcContent := "module test/divzero\n\nexport func main() -> int = 10 / 0\n"
	if err := os.WriteFile(src, []byte(srcContent), 0644); err != nil {
		t.Fatalf("write src: %v", err)
	}

	_, stderr, exitCode := runCLI(t, "run", "--verbose", "--bytecode", "--strict-bytecode", "--relax-modules", src)
	if exitCode == 0 {
		t.Fatalf("expected non-zero exit (divide by zero), got 0\nstderr=%s", stderr)
	}
	if !strings.Contains(stderr, "division by zero") {
		t.Errorf("expected stderr to mention divide by zero, got %q", stderr)
	}
	// The runtime error should carry a file:line marker. Don't pin the exact
	// line — the lower pass may attribute it to the func decl rather than the
	// expression — but require *some* file:line shaped token from this file.
	if !strings.Contains(stderr, "divzero.ail:") {
		t.Errorf("expected stderr to contain source file:line, got %q", stderr)
	}
}

// TestCLI_RunBytecode_EvalParity_Effectful is the M-BYTECODE-2D M3 acceptance
// test for end-to-end parity: an effectful program (one the bytecode compiler
// can't lower) must produce identical stdout under --bytecode (via the eval
// bridge) and under the regular evaluator path. Without M3, the --bytecode
// run would either fall back at the top level (no VM involvement) or fail.
func TestCLI_RunBytecode_EvalParity_Effectful(t *testing.T) {
	tmpDir := t.TempDir()
	src := filepath.Join(tmpDir, "parity.ail")
	srcContent := `module test/parity

import std/io (println)

export func main() -> () ! {IO} = {
  println("line one");
  println("line two");
  println("line three")
}
`
	if err := os.WriteFile(src, []byte(srcContent), 0644); err != nil {
		t.Fatalf("write src: %v", err)
	}

	// Run via the evaluator path (baseline).
	evalStdout, evalStderr, evalExit := runCLI(t, "run", "--caps", "IO", "--relax-modules", src)
	if evalExit != 0 {
		t.Fatalf("evaluator path exited %d\nstderr=%s", evalExit, evalStderr)
	}

	// Run via the bytecode VM path. Per M3, the entry function will be
	// EvalOnly and dispatched through the bridge — but the user-visible
	// stdout must be identical.
	vmStdout, vmStderr, vmExit := runCLI(t, "run", "--verbose", "--bytecode", "--caps", "IO", "--relax-modules", src)
	if vmExit != 0 {
		t.Fatalf("bytecode path exited %d\nstderr=%s", vmExit, vmStderr)
	}

	if vmStdout != evalStdout {
		t.Errorf("stdout parity mismatch:\nevaluator:\n%s\nbytecode:\n%s", evalStdout, vmStdout)
	}
	// Sanity: the VM run actually went through the bytecode path.
	if !strings.Contains(vmStderr, "via bytecode VM") {
		t.Errorf("expected --bytecode run to mention VM dispatch, got %q", vmStderr)
	}
}

// TestCLI_RunBytecode_StrictFails verifies --strict-bytecode exits non-zero
// instead of falling back when the program can't be compiled.
func TestCLI_RunBytecode_StrictFails(t *testing.T) {
	tmpDir := t.TempDir()
	src := filepath.Join(tmpDir, "bcrun_strict.ail")
	srcContent := `module test/bcrun_strict

import std/io (println)

export func main() -> () ! {IO} = println("nope")
`
	if err := os.WriteFile(src, []byte(srcContent), 0644); err != nil {
		t.Fatalf("write src: %v", err)
	}

	_, stderr, exitCode := runCLI(t, "run", "--verbose", "--bytecode", "--strict-bytecode", "--caps", "IO", "--relax-modules", src)
	if exitCode == 0 {
		t.Fatalf("expected non-zero exit, got 0\nstderr=%s", stderr)
	}
	if !strings.Contains(stderr, "bytecode execution failed") {
		t.Errorf("expected strict-bytecode failure message, got %q", stderr)
	}
}

func TestCLI_ListPatternArity_Bytecode(t *testing.T) {
	src := filepath.Join("tests", "golden", "bytecode", "pattern_arity.ail")
	evalStdout, evalStderr, evalExit := runCLI(t, "run", "--caps", "IO", src)
	if evalExit != 0 {
		t.Fatalf("evaluator path exited %d\nstderr=%s", evalExit, evalStderr)
	}

	vmStdout, vmStderr, vmExit := runCLI(t, "run", "--verbose", "--bytecode", "--caps", "IO", src)
	if vmExit != 0 {
		t.Fatalf("bytecode path exited %d\nstderr=%s", vmExit, vmStderr)
	}
	if !strings.Contains(vmStderr, "via bytecode VM") {
		t.Errorf("expected --bytecode run to mention VM dispatch, got %q", vmStderr)
	}
	evalStdout = listPatternProgramOutput(t, evalStdout)
	vmStdout = listPatternProgramOutput(t, vmStdout)

	want := strings.Join([]string{
		"arity1 []      = other",
		"arity1 [7]     = n1",
		"arity1 [7,8]   = other",
		"arity2 [7]     = other",
		"arity2 [7,8]   = n2",
		"arity2 [7,8,9] = other",
		"arity3 [7,8]   = other",
		"arity3 [7,8,9] = n3",
		"arity3 [7,8,9,10] = other",
		"tail2  [7]     = other",
		"tail2  [7,8]   = tail2",
		"tail2  [7,8,9] = tail2",
		"cons   []      = other",
		"cons   [7]     = cons",
		"cons   [7,8]   = cons",
	}, "\n") + "\n"
	if vmStdout != evalStdout {
		t.Errorf("stdout parity mismatch:\nevaluator:\n%s\nbytecode:\n%s", evalStdout, vmStdout)
	}
	if vmStdout != want {
		t.Errorf("bytecode stdout mismatch:\ngot:\n%swant:\n%s", vmStdout, want)
	}
}

func listPatternProgramOutput(t *testing.T, stdout string) string {
	t.Helper()
	const firstRow = "arity1 []      = "
	start := strings.Index(stdout, firstRow)
	if start < 0 {
		t.Fatalf("program output did not contain first arity row: %q", stdout)
	}
	return stdout[start:]
}

// TestCLI_RunBytecode_SharedFieldNames pins ailang#1354 and #1355: record
// types sharing a field name at different sorted positions. The compiler
// used to take a field's slot from whichever type it met first in Go map
// order, so each process compiled different bytecode — an out-of-range
// GET_FIELD in some runs, a silently wrong value in others. Map order is
// per process, so one run proves little: the strict VM must match the
// interpreter on every one of several runs, and the disassembly must be
// byte-identical across them.
func TestCLI_RunBytecode_SharedFieldNames(t *testing.T) {
	src := filepath.Join("tests", "golden", "bytecode", "shared_field_names.ail")
	const want = "[3.0, 10.0, 5.0, 6.0, 10.0, 14.0] [101.0, 104.0, 200.0, 300.0, 4.0, 3.0, 7.0] [42.0, 1.0, 2.0, 42.0, 20.0]"
	args := []string{"--quiet", "--relax-modules", "--entry", "main", "--args-json", "1", src}

	evalOut, evalErr, evalExit := runCLI(t, append([]string{"run"}, args...)...)
	if evalExit != 0 || strings.TrimSpace(evalOut) != want {
		t.Fatalf("interpreter: exit %d, got %q, want %q\nstderr=%s", evalExit, evalOut, want, evalErr)
	}

	const runs = 30
	var firstDisasm string
	for i := 0; i < runs; i++ {
		out, stderr, exit := runCLI(t, append([]string{"run", "--verbose", "--bytecode", "--strict-bytecode"}, args...)...)
		if exit != 0 || strings.TrimSpace(out) != want {
			t.Fatalf("strict VM run %d: exit %d, got %q, want %q\nstderr=%s", i, exit, out, want, stderr)
		}
		disasm, dErr, dExit := runCLI(t, "disasm", "--relax-modules", src)
		if dExit != 0 {
			t.Fatalf("disasm run %d: exit %d\nstderr=%s", i, dExit, dErr)
		}
		if i == 0 {
			firstDisasm = disasm
		} else if disasm != firstDisasm {
			t.Fatalf("disasm run %d differs from run 0: codegen is not deterministic", i)
		}
	}
}

func TestCLI_RunBytecode_QuicksortArity(t *testing.T) {
	src := filepath.Join("examples", "runnable", "recursion_quicksort.ail")
	stdout, stderr, exitCode := runCLI(t, "run", "--verbose", "--bytecode", "--caps", "IO", src)
	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d\nstdout=%s\nstderr=%s", exitCode, stdout, stderr)
	}
	for _, want := range []string{
		"Quicksort: [1, 1, 2, 3, 4, 5, 6, 9]",
		"sortBy:    [1, 1, 2, 3, 4, 5, 6, 9]",
	} {
		if !strings.Contains(stdout, want) {
			t.Errorf("expected stdout to contain %q, got %q", want, stdout)
		}
	}
	if strings.Contains(stderr, "falling back to evaluator") {
		t.Errorf("expected no fallback to evaluator, got stderr=%q", stderr)
	}
	if !strings.Contains(stderr, "via bytecode VM") {
		t.Errorf("expected stderr to mention bytecode VM run, got %q", stderr)
	}
}
