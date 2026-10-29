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

// #1865: elaboration binds every non-atomic list element (a call, a negated
// literal, a record) to a $tmp local first, so only constant lists were
// covered by the chunked MAKE_LIST: 128 calls pinned 128 locals and the
// function went evaluator-only again. Those temps are now compiled straight
// into their slots. Values must match the interpreter, and so must the order
// effects in elements run in.
func TestLongComputedListLiteralRunsOnStrictVM(t *testing.T) {
	bin := buildAilang(t)
	n := 300
	calls, negs, says := make([]string, n), make([]string, n), make([]string, n)
	for i := 0; i < n; i++ {
		calls[i] = fmt.Sprintf("g(%d)", i)
		negs[i] = fmt.Sprintf("-%d.5", i)
		says[i] = fmt.Sprintf("say(%d)", i)
	}
	recs := make([]string, 16)
	for i := range recs {
		strs := make([]string, 16)
		for j := range strs {
			strs[j] = fmt.Sprintf(`"s%d_%d"`, i, j)
		}
		recs[i] = fmt.Sprintf(`{ r: "x%d", g: [%s], n: g(%d) }`, i, strings.Join(strs, ", "), i)
	}
	src := filepath.Join(t.TempDir(), "computed.ail")
	writeFile(t, src, fmt.Sprintf(`module computed
import std/io (println)
import std/list (length)
pure func g(i: int) -> int = i * 3 + 1
func say(i: int) -> int ! {IO} {
  println("e${show(i)}");
  i
}
pure func calls() -> [int] = [%s]
pure func negs() -> [float] = [%s]
pure func regions() -> [{ r: string, g: [string], n: int }] = [%s]
export func main() -> () ! {IO} {
  let s = [%s];
  println(show(calls()));
  println(show(negs()));
  println(show(regions()));
  println(show(length(s)))
}
`, strings.Join(calls, ", "), strings.Join(negs, ", "), strings.Join(recs, ", "), strings.Join(says, ", ")))
	var outs []string
	for _, args := range [][]string{
		{"run", "--relax-modules", "--quiet", "--caps", "IO", "--entry", "main", src},
		{"run", "--relax-modules", "--quiet", "--bytecode", "--strict-bytecode", "--caps", "IO", "--entry", "main", src},
	} {
		out, stderr, code := runWithStdin(t, bin, "", args...)
		if code != 0 {
			t.Fatalf("%v: exit=%d stderr=%s", args, code, stderr)
		}
		outs = append(outs, out)
	}
	if outs[0] != outs[1] {
		t.Errorf("interpreter and strict VM differ:\n%.400s\n---\n%.400s", outs[0], outs[1])
	}
	if !strings.HasPrefix(outs[1], "e0\ne1\ne2\n") || !strings.Contains(outs[1], "e299\n[1, 4, 7") {
		t.Errorf("effects out of order or values missing: %.200s", outs[1])
	}
}
