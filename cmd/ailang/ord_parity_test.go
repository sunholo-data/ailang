package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestOrdEvaluatorVMParity pins float ordered comparisons (< <= > >=) to IEEE
// 754 on both backends and on every lowering path (M-FLOAT-ORD-ONE-SEMANTICS,
// #1419): every ordered comparison with a NaN operand is false. Before it, the
// evaluator's Ord[Float] dictionary used a total order with NaN greatest, so
// `nan > 1.0` was true on `ailang run` and false on `--bytecode`, and the
// evaluator disagreed with itself through a generic helper.
//
// The program is PURE on purpose (an effectful main is bridged back to the
// evaluator under --bytecode). Every row must agree across the backends; a
// future divergence should be fixed, not recorded as an expected difference.
func TestOrdEvaluatorVMParity(t *testing.T) {
	rows := []struct{ expr, want string }{
		{"n < 1.0", "false"},
		{"n <= 1.0", "false"},
		{"n > 1.0", "false"},
		{"n >= 1.0", "false"},
		{"1.0 < n", "false"},
		{"1.0 <= n", "false"},
		{"1.0 > n", "false"},
		{"1.0 >= n", "false"},
		{"n < n", "false"},
		{"n >= n", "false"},
		{"pinf > n", "false"},
		{"n > ninf", "false"},
		{"not (n <= 1.0)", "true"},
		{"gt2(n, 1.0)", "false"},
		{"ge2(n, 1.0)", "false"},
		{"(\\p. \\q. p > q)(n)(1.0)", "false"},
		{"(\\p. \\q. p >= q)(n)(1.0)", "false"},
		{"above(n, 1.0)", "false"},
		{"atLeast(n, 1.0)", "false"},
		{"isNaN(clampHigh(n, 1.0))", "true"},
		{"optShow(maximumFloat([n, 1.0]))", "1.0"},
		{"optShow(maximumFloat([1.0, n]))", "NaN"},
		{"optShow(minimumFloat([n, 1.0]))", "1.0"},
		{"optShow(minimumFloat([1.0, n]))", "NaN"},
		{"show(sortBy(cmp, [2.0, n, 1.0]))", "[2.0, NaN, 1.0]"},
		{"show(sortBy(cmp, [3.0, 1.0, 2.0]))", "[1.0, 2.0, 3.0]"},
		// Non-NaN regression rows.
		{"1.0 < 2.0", "true"},
		{"2.0 >= 2.0", "true"},
		{"ninf < pinf", "true"},
		{"0.0 < (0.0 * -1.0)", "false"},
		{"(0.0 * -1.0) <= 0.0", "true"},
		{"above(2.0, 1.0)", "true"},
		{"1 < 2", "true"},
		{"\"a\" < \"b\"", "true"},
	}
	parts := make([]string, len(rows))
	for i, r := range rows {
		e := r.expr
		if !strings.HasPrefix(e, "show(") && !strings.HasPrefix(e, "optShow(") {
			e = "show(" + e + ")"
		}
		parts[i] = e
	}
	src := filepath.Join(t.TempDir(), "ordparity.ail")
	prog := fmt.Sprintf(`module test/ordparity

import std/math (isNaN)
import std/list (maximumFloat, minimumFloat, sortBy)
import std/string (join)

func gt2(p: float, q: float) -> bool = p > q
func ge2(p: float, q: float) -> bool = p >= q
func above[a](x: a, hi: a) -> bool = x > hi
func atLeast[a](x: a, lo: a) -> bool = x >= lo
func clampHigh(x: float, hi: float) -> float = if x > hi then hi else x
func cmp(a: float, b: float) -> int = if a < b then -1 else if a > b then 1 else 0
func optShow(o: Option[float]) -> string = match o {
  Some(x) => show(x),
  None => "none"
}

export func main() -> string {
  let n = 0.0 / 0.0;
  let pinf = 1.0 / 0.0;
  let ninf = -1.0 / 0.0;
  join("|", [%s])
}
`, strings.Join(parts, ", "))
	if err := os.WriteFile(src, []byte(prog), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AILANG_NO_CACHE", "1")
	for _, backend := range []struct {
		name string
		args []string
	}{
		{"evaluator", []string{"run", "--relax-modules", src}},
		{"vm", []string{"run", "--verbose", "--bytecode", "--relax-modules", src}},
	} {
		stdout, stderr, code := runCLI(t, backend.args...)
		if code != 0 {
			t.Fatalf("%s: exit %d\nstderr=%s", backend.name, code, stderr)
		}
		if backend.name == "vm" && !strings.Contains(stderr, "via bytecode VM") {
			t.Fatalf("vm: program did not run on the VM\nstderr=%s", stderr)
		}
		lines := strings.Split(strings.TrimSpace(stdout), "\n")
		got := strings.Split(lines[len(lines)-1], "|")
		if len(got) != len(rows) {
			t.Fatalf("%s: got %d results, want %d: %q", backend.name, len(got), len(rows), lines[len(lines)-1])
		}
		for i, r := range rows {
			if got[i] != r.want {
				t.Errorf("%s: %s = %q, want %q", backend.name, r.expr, got[i], r.want)
			}
		}
	}
}
