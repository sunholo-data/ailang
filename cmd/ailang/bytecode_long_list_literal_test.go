package main

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

// #1736: a list literal longer than the 256-register frame made its function
// evaluator-only, so --strict-bytecode failed at run time. Long literals now
// compile as chunked MAKE_LISTs joined with _concat_List; order and values
// must match the interpreter, including elements computed from parameters.
func TestLongListLiteralRunsOnStrictVM(t *testing.T) {
	bin := buildAilang(t)
	for _, n := range []int{64, 65, 300, 601} {
		elems := make([]string, n)
		for i := range elems {
			elems[i] = fmt.Sprint(i)
		}
		xs := strings.Join(elems, ", ")
		src := filepath.Join(t.TempDir(), "biglist.ail")
		writeFile(t, src, fmt.Sprintf(`module biglist
pure func xs() -> [int] = [%s]
pure func wide(a: int, b: int) -> [int] = [a, %s, a + b]
export pure func main() -> [int] = wide(7, 8) ++ xs()
`, xs, xs))
		var outs []string
		for _, args := range [][]string{
			{"run", "--relax-modules", "--quiet", "--entry", "main", src},
			{"run", "--relax-modules", "--quiet", "--bytecode", "--strict-bytecode", "--entry", "main", src},
		} {
			out, stderr, code := runWithStdin(t, bin, "", args...)
			if code != 0 {
				t.Fatalf("n=%d %v: exit=%d stderr=%s", n, args, code, stderr)
			}
			outs = append(outs, out)
		}
		if outs[0] != outs[1] {
			t.Errorf("n=%d: interpreter and strict VM differ:\n%s\n%s", n, outs[0], outs[1])
		}
		if !strings.Contains(outs[1], fmt.Sprintf("%d, 15", n-1)) {
			t.Errorf("n=%d: wide() tail missing from strict VM output: %.120s", n, outs[1])
		}
	}
}

// #1736: a long if/else-if chain over a wide record pinned one or two
// compiler temps per level, so the 16-register record rebuild ran out of
// contiguous registers after ~15 arms. Tail and value position chains of 80
// arms must now run on the strict VM and match the interpreter.
func TestLongElseIfChainRunsOnStrictVM(t *testing.T) {
	bin := buildAilang(t)
	fields, init, arms := make([]string, 16), make([]string, 16), make([]string, 80)
	for i := range fields {
		fields[i] = fmt.Sprintf("f%d: int", i)
		init[i] = fmt.Sprintf("f%d: 0", i)
	}
	for k := range arms {
		arms[k] = fmt.Sprintf(`if key == "k%d" then {r | f%d: v + %d}`, k, k%16, k)
	}
	chain := strings.Join(arms, " else ") + " else r"
	src := filepath.Join(t.TempDir(), "chain.ail")
	writeFile(t, src, fmt.Sprintf(`module chain
type R = {%s}
pure func tailChain(r: R, key: string, v: int) -> R = %s
pure func valueChain(r: R, key: string, v: int) -> int {
  let out = %s;
  out.f3 + out.f7 + 1
}
pure func zero() -> R = {%s}
export pure func main() -> [int] {
  let a = tailChain(tailChain(zero(), "k3", 1), "k79", 2);
  [a.f3, a.f15, valueChain(zero(), "k19", 5), valueChain(zero(), "k7", 1), valueChain(zero(), "none", 9)]
}
`, strings.Join(fields, ", "), chain, chain, strings.Join(init, ", ")))
	var outs []string
	for _, args := range [][]string{
		{"run", "--relax-modules", "--quiet", "--entry", "main", src},
		{"run", "--relax-modules", "--quiet", "--bytecode", "--strict-bytecode", "--entry", "main", src},
	} {
		out, stderr, code := runWithStdin(t, bin, "", args...)
		if code != 0 {
			t.Fatalf("%v: exit=%d stderr=%s", args, code, stderr)
		}
		outs = append(outs, strings.TrimSpace(out))
	}
	if want := "[4, 81, 25, 9, 1]"; outs[0] != want || outs[1] != want {
		t.Errorf("interpreter=%s strict VM=%s, want %s", outs[0], outs[1], want)
	}
}
