package main

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// #1518 / #1501: std/list any, findIndex, foldr, the six extremes and the five
// effectful combinators recursed once per element. On the interpreter the
// non-tail ones failed RT_REC_003 past 10,000 elements; on the strict VM they
// overflowed its stack at 50,001; and every `[x, ...rest]` walk copied the tail
// on the VM, so any/findIndex took 117 s at 320,000 elements. Each row here runs
// at 200,000 elements on all three engines, under a 30 s wall-clock ceiling: the
// iterative forms need about a second, the quadratic VM walks took ~45 s.

// scaleN is the list length each engine is held to in the at-scale test.
//
// The interpreter runs a smaller list. Its effectful helpers are linear but
// cost about 50us per element (measured 3.8s / 5.5s / 10.9s at 50k / 100k /
// 200k on an M-series Mac), and CI runners are about 3x slower: at 200k the
// test took 33-42s against its 30s ceiling on every OS, and on Windows it
// pushed the cmd/ailang package past its -timeout budget. 50k keeps the
// regression it guards against visible: a per-element copy is quadratic, about
// 1.25e9 steps at 50k, far beyond the ceiling on any runner.
var listIterEngines = []struct {
	name   string
	args   []string
	scaleN int
}{
	{"interpreter", nil, 50000},
	{"vm", []string{"--bytecode"}, 200000},
	{"strict-vm", []string{"--bytecode", "--strict-bytecode"}, 200000},
}

const listIterSrc = "cmd/ailang/testdata/listiter/helpers.ail"

func runListIter(t *testing.T, bin string, engine []string, entry string, n int, caps ...string) (string, time.Duration) {
	t.Helper()
	args := append([]string{"run", "--quiet"}, engine...)
	args = append(args, caps...)
	args = append(args, "--entry", entry, "--args-json", strconv.Itoa(n), filepath.FromSlash(listIterSrc))
	start := time.Now()
	stdout, stderr, code := runWithStdin(t, bin, "", args...)
	elapsed := time.Since(start)
	if code != 0 {
		t.Fatalf("%s %v: exit %d\nstdout=%s\nstderr=%s", entry, engine, code, stdout, stderr)
	}
	return strings.TrimSpace(stdout), elapsed
}

// expected values, computed independently of AILANG
func listIterWant(entry string, n int) string {
	switch entry {
	case "anyAt":
		return "true,false"
	case "findIndexAt":
		return fmt.Sprintf("%d,none", n-1)
	case "foldrAt":
		acc := 0
		for x := n - 1; x >= 0; x-- {
			acc = (acc*31 + x) % 1000000007
		}
		return strconv.Itoa(acc)
	case "extremesAt":
		maxI, minI := -1, n
		maxS, minS := "", ""
		for i := 0; i < n; i++ {
			v := (i * 7919) % n
			s := strconv.Itoa(v)
			if v > maxI {
				maxI = v
			}
			if v < minI {
				minI = v
			}
			if i == 0 || s > maxS {
				maxS = s
			}
			if i == 0 || s < minS {
				minS = s
			}
		}
		return fmt.Sprintf("%d,%d,%d.0,%d.0,%s,%s", maxI, minI, maxI, minI, maxS, minS)
	case "effectfulAt":
		acc := 0
		for x := 0; x < n; x++ {
			acc = (acc*31 + x) % 1000000007
		}
		lastEven := n - 1
		if lastEven%2 != 0 {
			lastEven--
		}
		return fmt.Sprintf("%d,%d:%d,%d:%d,%d:%d", acc, n, 2*(n-1), (n+1)/2, lastEven, 2*n, n-1)
	}
	panic("no expectation for " + entry)
}

func TestStdListHelpersIterativeAtScale(t *testing.T) {
	bin := buildAilang(t)
	const ceiling = 30 * time.Second
	for _, entry := range []string{"anyAt", "findIndexAt", "foldrAt", "extremesAt", "effectfulAt"} {
		for _, eng := range listIterEngines {
			n := eng.scaleN
			want := listIterWant(entry, n)
			t.Run(entry+"/"+eng.name, func(t *testing.T) {
				got, took := runListIter(t, bin, eng.args, entry, n)
				if got != want {
					t.Errorf("%s on %s = %q, want %q", entry, eng.name, got, want)
				}
				if took > ceiling {
					t.Errorf("%s on %s took %v at n=%d (ceiling %v): a per-element copy or recursion is back", entry, eng.name, took, n, ceiling)
				}
			})
		}
	}
}

// Answers on small inputs must be exactly what the recursive definitions gave
// (captured from them before the rewrite). The float NaN cases follow IEEE
// ordering, identical on every engine since #1419 made the interpreter's `>`/`<`
// on NaN agree with the VM's.
func TestStdListHelpersSemanticsUnchanged(t *testing.T) {
	bin := buildAilang(t)
	const common1 = "none 5 9 1 | "
	const common2 = " -0.0 0.0 | b  none | true 1 | 1 none false | 123. ."
	want := map[string]string{
		"interpreter": common1 + "2.0 NaN 1.0 NaN" + common2,
		"vm":          common1 + "2.0 NaN 1.0 NaN" + common2,
		"strict-vm":   common1 + "2.0 NaN 1.0 NaN" + common2,
	}
	for _, eng := range listIterEngines {
		got, _ := runListIter(t, bin, eng.args, "semantics", 5)
		if got != want[eng.name] {
			t.Errorf("semantics on %s:\n got %q\nwant %q", eng.name, got, want[eng.name])
		}
	}
}

func TestStdListEffectfulCombinatorsOrder(t *testing.T) {
	bin := buildAilang(t)
	const n = 37 // odd and not a power of two: exercises uneven range splits
	var want []string
	for _, p := range []string{"m", "f", "g", "l", "e"} {
		for i := 0; i < n; i++ {
			want = append(want, fmt.Sprintf("%s%d", p, i))
		}
	}
	want = append(want, fmt.Sprintf("%d %d %d %d", n, (n+1)/2, n, n*(n-1)/2))
	// the strict VM has no IO, so effect order is checked on the other two
	for _, eng := range listIterEngines[:2] {
		got, _ := runListIter(t, bin, eng.args, "effectOrder", n, "--caps", "IO")
		if got != strings.Join(want, "\n") {
			t.Errorf("effectOrder on %s:\n got %q\nwant %q", eng.name, got, strings.Join(want, "\n"))
		}
	}
}
