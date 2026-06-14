package eval_harness

import (
	"strings"
	"testing"
	"time"
)

// skipIfNoAver skips the test unless the `aver` binary can be resolved (PATH,
// $AILANG_AVER, or the $CARGO_HOME/bin fallback). CI typically lacks Aver.
func skipIfNoAver(t *testing.T) {
	t.Helper()
	if _, err := resolveAver(); err != nil {
		t.Skip("aver not available, skipping (install with `cargo install aver-lang`)")
	}
}

// TestNewAverCheckCommand verifies the command shape: `aver check <file>`.
// This is a contract test and does not execute Aver.
func TestNewAverCheckCommand(t *testing.T) {
	skipIfNoAver(t)
	cmd, err := newAverCheckCommand("solution.av")
	if err != nil {
		t.Fatalf("newAverCheckCommand: %v", err)
	}
	// argv is [<resolved aver path>, "check", "solution.av"].
	if len(cmd.Args) != 3 {
		t.Fatalf("expected 3 args, got %d: %v", len(cmd.Args), cmd.Args)
	}
	if cmd.Args[1] != "check" || cmd.Args[2] != "solution.av" {
		t.Errorf("expected [_, check, solution.av], got %v", cmd.Args)
	}
}

// TestAverRunner_Success confirms a runnable program passes and that pass/fail
// tracks `aver run` (not `aver check`). The program below deliberately OMITS a
// `module` declaration — `aver run` accepts it, `aver check` would reject it —
// so this guards against any regression to gating pass/fail on `aver check`.
func TestAverRunner_Success(t *testing.T) {
	skipIfNoAver(t)

	code := "fn main() -> Unit\n" +
		"    ! [Console.print]\n" +
		"    Console.print(\"hello\")\n"

	r := NewAverRunner()
	result, err := r.Run(code, 30*time.Second)
	if err != nil {
		t.Fatalf("Run should not return error: %v", err)
	}
	if !result.RuntimeOk || result.ExitCode != 0 {
		t.Fatalf("expected success (module-less program still runs); got exit=%d stderr=%q",
			result.ExitCode, result.Stderr)
	}
	if !strings.Contains(result.Stdout, "hello") {
		t.Errorf("expected stdout to contain 'hello', got %q", result.Stdout)
	}
}

// TestAverRunner_CompileErrorEnrichedWithCheck confirms that on a compile-class
// failure the runner surfaces `aver check`'s rich diagnostics (named category +
// `repair:` hint + source excerpt) rather than `aver run`'s terse one-liner.
func TestAverRunner_CompileErrorEnrichedWithCheck(t *testing.T) {
	skipIfNoAver(t)

	// Type error: adding a String and an Int. Compiles-class failure for both
	// `aver run` (terse) and `aver check` (rich).
	code := "module Solution\n" +
		"    intent = \"type error\"\n" +
		"    exposes [main]\n" +
		"    effects [Console.print]\n" +
		"\n" +
		"fn main() -> Unit\n" +
		"    ! [Console.print]\n" +
		"    x = \"hello\" + 1\n" +
		"    Console.print(\"{x}\")\n"

	r := NewAverRunner()
	result, err := r.Run(code, 30*time.Second)
	if err != nil {
		t.Fatalf("Run should not return error: %v", err)
	}
	if result.ExitCode == 0 {
		t.Fatalf("expected non-zero exit for a type error, got 0; stderr=%q", result.Stderr)
	}
	if result.CompileOk {
		t.Errorf("expected CompileOk=false for a compile-class error")
	}
	// `aver check` enrichment markers — absent from `aver run`'s one-line output.
	if !strings.Contains(result.Stderr, "error[") {
		t.Errorf("expected a named error category (error[...]) in diagnostics, got %q", result.Stderr)
	}
	if !strings.Contains(result.Stderr, "Check:") {
		t.Errorf("expected `aver check` header (Check:) in enriched diagnostics, got %q", result.Stderr)
	}
}
