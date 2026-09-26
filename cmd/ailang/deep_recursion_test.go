package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// #1317: `--max-recursion-depth N` counted AILANG calls, but the evaluator
// recursed on one goroutine, whose stack Go caps at 512 MB (1 GB at most). A raised
// ceiling was therefore a promise the runtime could not keep: these programs died
// with `fatal error: stack overflow` and a Go runtime dump, exit 2, well below N.
// Deep evaluation now continues on fresh goroutines, so each row must complete,
// and exceeding N must still be the ordinary RT_REC_003.
//
// The callback row recurses THROUGH a builtin (map re-enters the evaluator via
// FnCaller), so it also pins that a continuation goroutine is re-registered as
// the request's fork: without that, builtins there resolve the shared evaluator.
func TestDeepRecursionHonoursMaxRecursionDepth(t *testing.T) {
	cases := []struct {
		name, body string
		n, flag    int
		want       string
	}{
		{"plain", `pure func down(n: int) -> int = if n == 0 then 0 else 1 + down(n - 1)`, 300000, 400000, "300000"},
		{"through map callback", `pure func down(n: int) -> int =
  if n == 0 then 0 else match map(\x. down(n - 1) + x, [1]) { [v] => v, _ => 0 }`, 150000, 400000, "150000"},
		{"over the flag", `pure func down(n: int) -> int = if n == 0 then 0 else 1 + down(n - 1)`, 400001, 400000, "RT_REC_003"},
		// Two AILANG calls per level (down and the lambda), so 60,000 levels is
		// 120,000 calls against a 100,000 ceiling, past the first hop (~14k levels
		// here). If the continuation goroutine were not re-registered as the fork,
		// the callbacks would run on the shared evaluator, whose call counter starts
		// at zero, and the ceiling would silently stop being enforced.
		{"over the flag through map callback", `pure func down(n: int) -> int =
  if n == 0 then 0 else match map(\x. down(n - 1) + x, [1]) { [v] => v, _ => 0 }`, 60000, 100000, "RT_REC_003"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			src := filepath.Join(t.TempDir(), "deep.ail")
			prog := fmt.Sprintf(`module test/deep

import std/list (map)

%s

export func main() -> string = show(down(%d))
`, c.body, c.n)
			if err := os.WriteFile(src, []byte(prog), 0644); err != nil {
				t.Fatal(err)
			}
			t.Setenv("AILANG_NO_CACHE", "1")
			stdout, stderr, code := runCLI(t, "run", "--relax-modules", "--max-recursion-depth", fmt.Sprint(c.flag), src)
			if strings.Contains(stderr, "fatal error") {
				t.Fatalf("Go runtime fatal instead of an AILANG result (exit %d):\n%.600s", code, stderr)
			}
			if c.want == "RT_REC_003" {
				if code == 0 || !strings.Contains(stderr, "RT_REC_003") {
					t.Fatalf("exceeding the flag: exit %d, want RT_REC_003\nstderr=%.600s", code, stderr)
				}
				// One line, not one "callback error at index 0:" prefix per level:
				// that wrapping made the through-callback row 1.9 MB of stderr.
				if len(stderr) > 4096 {
					t.Fatalf("RT_REC_003 buried in %d bytes of stderr:\n%.300s", len(stderr), stderr)
				}
				return
			}
			if code != 0 {
				t.Fatalf("exit %d\nstderr=%.600s", code, stderr)
			}
			lines := strings.Split(strings.TrimSpace(stdout), "\n")
			if got := lines[len(lines)-1]; got != c.want {
				t.Fatalf("down(%d) = %q, want %q", c.n, got, c.want)
			}
		})
	}
}
