package pipeline

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/types"
)

// #2308 and #2329: small programs drove `check` (and so the LSP and the hosted
// check tool) to exponential time, and one shape to tens of GiB. Each shape
// below runs at a size that took well over 30 s before the fix and must now
// finish in a fraction of a second with a bounded allocation. The in-test time
// limit fails the test instead of hanging CI; the allocation bound is on
// TotalAlloc (allocation, not RSS — RSS-based tests have been flaky here).

// blowupLit is the #2308 record literal: n levels, each with one extra field,
// read back through n field accesses.
func blowupLit(n int) string {
	lit := `{v: 1, w: "s"}`
	for i := 0; i < n; i++ {
		lit = fmt.Sprintf("{n: %s, k%d: %d}", lit, i, i)
	}
	return lit
}

func blowupAccess(n int) string { return strings.Repeat(".n", n) + ".v" }

// blowupAliasChain is #2329 shape 1: aliases each doubling their argument.
func blowupAliasChain(n int, use string) string {
	var b strings.Builder
	b.WriteString("module m\n")
	for i := 0; i < n; i++ {
		fmt.Fprintf(&b, "type P%d[a] = P%d[(a, a)]\n", i, i+1)
	}
	fmt.Fprintf(&b, "type P%d[a] = Option[a]\n%s\nexport func main() -> () ! {IO} = println(\"p\")\n", n, use)
	return b.String()
}

// blowupAliasDAG is #2329 shape 2: n aliases, each a tuple of the previous five.
func blowupAliasDAG(n int, use string) string {
	var b strings.Builder
	b.WriteString("module m\ntype D0 = int\n")
	for i := 1; i < n; i++ {
		var parts []string
		for j := max(0, i-5); j < i; j++ {
			parts = append(parts, fmt.Sprintf("D%d", j))
		}
		if len(parts) < 2 {
			parts = append(parts, "int")
		}
		fmt.Fprintf(&b, "type D%d = (%s)\n", i, strings.Join(parts, ", "))
	}
	fmt.Fprintf(&b, "%s\nexport func main() -> () ! {IO} = println(\"dag\")\n", strings.ReplaceAll(use, "DLAST", fmt.Sprintf("D%d", n-1)))
	return b.String()
}

// blowupLetDoubling binds n lets, each a pair of the previous one: a type of
// 2^n leaves held in n nodes.
func blowupLetDoubling(n int) string {
	var b strings.Builder
	b.WriteString("module m\nexport func main() -> () ! {IO} {\n  let t0 = 1;\n")
	for i := 1; i <= n; i++ {
		fmt.Fprintf(&b, "  let t%d = (t%d, t%d);\n", i, i-1, i-1)
	}
	b.WriteString("  println(\"t\")\n}\n")
	return b.String()
}

// checkBounded type-checks src (cache off) within limit, failing the test on
// timeout or when the check allocated more than allocBudget bytes.
func checkBounded(t *testing.T, src string, limit time.Duration, allocBudget uint64) error {
	t.Helper()
	t.Setenv("AILANG_NO_CACHE", "1")
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "m.ail"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	type outcome struct {
		err   error
		alloc uint64
	}
	done := make(chan outcome, 1)
	go func() {
		var before, after runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&before)
		_, err := Run(Config{DryLink: true}, Source{Code: src, Filename: "m.ail"})
		runtime.ReadMemStats(&after)
		done <- outcome{err, after.TotalAlloc - before.TotalAlloc}
	}()
	// A regression grows the live heap by GiB a second (the #2308 shape reached
	// 6 GB in 15 s): stop the whole test binary before it exhausts the machine,
	// since a timed-out check goroutine cannot be cancelled.
	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()
	deadline := time.After(limit)
	for {
		select {
		case o := <-done:
			if o.alloc > allocBudget {
				t.Fatalf("check allocated %d MB (budget %d MB)", o.alloc>>20, allocBudget>>20)
			}
			return o.err
		case <-deadline:
			t.Fatalf("check did not finish within %s", limit)
			return nil
		case <-tick.C:
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			if m.HeapAlloc > 8*allocBudget {
				panic(fmt.Sprintf("check holds %d MB of live heap (allocation budget %d MB): stopping before it exhausts memory", m.HeapAlloc>>20, allocBudget>>20))
			}
		}
	}
}

const (
	blowupLimit = 20 * time.Second // ~0.2 s after the fix; minutes or more before it
	blowupAlloc = 256 << 20
)

// #2308: depth 24 (~4.5x per two levels from 2.9 s at depth 14 on dev: ~30 min).
func TestCheckBlowup_2308_NestedRecordLiteral(t *testing.T) {
	src := fmt.Sprintf("module m\nexport func main() -> () ! {IO} {\n  let r = %s;\n  println(show(r%s))\n}\n", blowupLit(24), blowupAccess(24))
	if err := checkBounded(t, src, blowupLimit, blowupAlloc); err != nil {
		t.Fatalf("valid program rejected: %v", err)
	}
}

// #2308, the 50 GiB form: a getter lambda over 40 levels applied to the literal.
func TestCheckBlowup_2308_GetterOverNestedLiteral(t *testing.T) {
	src := fmt.Sprintf("module m\nexport func main() -> () ! {IO} {\n  let get = \\q. q%s;\n  println(show(get(%s)))\n}\n", blowupAccess(40), blowupLit(40))
	if err := checkBounded(t, src, blowupLimit, blowupAlloc); err != nil {
		t.Fatalf("valid program rejected: %v", err)
	}
}

// #2329 shape 1: 30 doubling aliases in a signature (n=24 took 1.5 s, x2 per alias).
func TestCheckBlowup_2329_DoublingAliasChain(t *testing.T) {
	src := blowupAliasChain(30, "pure func f(x: P0[int]) -> int = 0")
	if err := checkBounded(t, src, blowupLimit, blowupAlloc); err != nil {
		t.Fatalf("valid program rejected: %v", err)
	}
}

// #2329 shape 2: 120 aliases, each a tuple of the previous five.
func TestCheckBlowup_2329_DenseAliasDAG(t *testing.T) {
	src := blowupAliasDAG(120, "pure func f(x: DLAST) -> int = 0")
	if err := checkBounded(t, src, blowupLimit, blowupAlloc); err != nil {
		t.Fatalf("valid program rejected: %v", err)
	}
}

// #2329 shape 2, unified with a separately declared twin (n=25 took 18 s
// with the IFC fix alone; each pair of aliases is now unified once).
func TestCheckBlowup_2329_TwinAliasDAGsUnified(t *testing.T) {
	if err := checkBounded(t, blowupTwinDAGs(120), blowupLimit, blowupAlloc); err != nil {
		t.Fatalf("valid program rejected: %v", err)
	}
}

// paramChain declares X0[a] = a, Xi[a] = (X{i-1}[a], X{i-1}[a]): a doubling
// chain that passes its parameter through.
func paramChain(x string, n int) string {
	var b strings.Builder
	fmt.Fprintf(&b, "type %s0[a] = a\n", x)
	for i := 1; i <= n; i++ {
		fmt.Fprintf(&b, "type %s%d[a] = (%s%d[a], %s%d[a])\n", x, i, x, i-1, x, i-1)
	}
	return b.String()
}

// paramDAG declares X0[a] = a, Xi[a] = a tuple of the previous five, each
// applied to a.
func paramDAG(x string, n int) string {
	var b strings.Builder
	fmt.Fprintf(&b, "type %s0[a] = a\n", x)
	for i := 1; i < n; i++ {
		var parts []string
		for j := max(0, i-5); j < i; j++ {
			parts = append(parts, fmt.Sprintf("%s%d[a]", x, j))
		}
		if len(parts) < 2 {
			parts = append(parts, "a")
		}
		fmt.Fprintf(&b, "type %s%d[a] = (%s)\n", x, i, strings.Join(parts, ", "))
	}
	return b.String()
}

// #2363 review: aliases that pass a type parameter through. Each body writes
// its own `a`, so a walk keyed on AST identity never shared them (the IFC
// bound walk reached 3 GB at n=22), and unifying two such twins was 2^n pairs
// (n=22: 23 s; n=24: rejected by the step budget only after a minute).
func TestCheckBlowup_ParameterPassingAliases(t *testing.T) {
	main := "export func main() -> () ! {IO} = println(\"x\")\n"
	cases := map[string]string{
		"doubling chain in a signature (qbin)": "module m\n" + paramChain("Q", 30) + "pure func f(x: Q30[int]) -> int = 0\n" + main,
		"dense DAG at int (pdag1)":             "module m\n" + paramDAG("D", 60) + "pure func f(x: D59[int]) -> int = 0\n" + main,
		"dense DAG at a variable (pdag1v)":     "module m\n" + paramDAG("D", 60) + "pure func f(x: D59[b]) -> int = 0\n" + main,
		"twin dense DAGs unified (pdag2)":      "module m\n" + paramDAG("D", 60) + paramDAG("E", 60) + "pure func f(x: D59[b]) -> E59[b] = x\n" + main,
		"twin chains unified at int (ubin)": "module m\n" + paramChain("P", 30) + paramChain("Q", 30) +
			"pure func g(u: int) -> P30[int] = unreachable(\"x\")\npure func f(u: int) -> Q30[int] = g(u)\n" + main,
	}
	for name, src := range cases {
		t.Run(name, func(t *testing.T) {
			if err := checkBounded(t, src, blowupLimit, blowupAlloc); err != nil {
				t.Fatalf("valid program rejected: %v", err)
			}
		})
	}
}

// #2363 review: a type error through a doubling alias rendered the expanded
// type in full while the message was built (~900 MB at n=24, over 3 GB at
// n=30). Diagnostics render a large type in part.
func TestCheckBlowup_TypeErrorThroughDoublingAliasIsBounded(t *testing.T) {
	src := blowupAliasChain(30, "pure func f(x: P0[int]) -> int = x")
	err := checkBounded(t, src, blowupLimit, blowupAlloc)
	if err == nil {
		t.Fatal("a mismatch through the alias was accepted")
	}
	if len(err.Error()) > 16<<10 {
		t.Fatalf("diagnostic is %d bytes: the expanded type was written out", len(err.Error()))
	}
	if !strings.Contains(err.Error(), "shown in part") {
		t.Fatalf("diagnostic does not say the type is shown in part: %.300s", err.Error())
	}
}

// blowupTwinDAGs declares the dense alias DAG twice (D and E) and unifies
// the two: both sides stay single alias names while unification walks their
// expansions pairwise.
func blowupTwinDAGs(n int) string {
	src := blowupAliasDAG(n, "")
	twin := strings.NewReplacer("type D", "type E", "D", "E").Replace(src[strings.Index(src, "type D0"):strings.Index(src, "\nexport func main")])
	return strings.Replace(src, "\nexport func main", "\n"+twin+fmt.Sprintf("\npure func f(x: D%d) -> E%d = x\nexport func main", n-1, n-1), 1)
}

// blowupRecordDoubling binds n lets, each a record holding the previous twice.
func blowupRecordDoubling(n int) string {
	var b strings.Builder
	b.WriteString("module m\nexport func main() -> () ! {IO} {\n  let r0 = {v: 1};\n")
	for i := 1; i <= n; i++ {
		fmt.Fprintf(&b, "  let r%d = {a: r%d, b: r%d};\n", i, i-1, i-1)
	}
	fmt.Fprintf(&b, "  println(show(r%d%s.v))\n}\n", n, strings.Repeat(".a", n))
	return b.String()
}

// The backstop: a type too large to check is a located TC_TYPE_SIZE_001,
// quickly, where it first appears.
func TestCheckBlowup_TypeTooLargeIsADiagnostic(t *testing.T) {
	// Unused, g costs nothing; the call g(None) binds None's argument to the
	// expanded P0[int], which is where it first materialises.
	matchP0 := "pure func g(x: P0[int]) -> bool = match x { Some(_) => true, None => false }\npure func h() -> bool = g(None)"
	cases := map[string]struct {
		src, where string
		located    bool // the message carries a source position
	}{
		"let doubling":             {blowupLetDoubling(40), "let binding t", true},
		"record doubling":          {blowupRecordDoubling(40), "let binding r", true},
		"alias chain materialised": {blowupAliasChain(40, matchP0), "this declaration", true},
		"alias DAG exported":       {blowupAliasDAG(120, "export pure func f(x: DLAST) -> DLAST = x"), "exported f", false},
		// #2327's panic ("unification depth exceeded 1000") is no longer
		// reachable through a self-referential alias (TC_TYPE_ALIAS_CYCLE_001),
		// but was through two written 1100-deep list types.
		"deeply nested written types": {"module m\nexport func main() -> () ! {IO} {\n  let x: " + strings.Repeat("[", 1100) + "int" + strings.Repeat("]", 1100) +
			" = [];\n  let y: " + strings.Repeat("[", 1100) + "string" + strings.Repeat("]", 1100) + " = x;\n  println(\"d\")\n}\n", "nest more than 1000 deep", true},
		"doubling by application": {"module m\nexport func main() -> () ! {IO} {\n  let d = \\x. (x, x);\n  let t = " + strings.Repeat("d(", 40) + "1" + strings.Repeat(")", 40) + ";\n  println(\"t\")\n}\n", "an intermediate expression", true},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			alloc := uint64(blowupAlloc)
			if strings.Contains(c.where, "deep") {
				alloc = 1 << 30 // unifying 1100 levels re-applies the substitution per level: quadratic, not a blowup
			}
			err := checkBounded(t, c.src, blowupLimit, alloc)
			if err == nil || !strings.Contains(err.Error(), types.TC_TYPE_SIZE_001) || !strings.Contains(err.Error(), c.where) {
				t.Fatalf("want %s naming %q, got: %v", types.TC_TYPE_SIZE_001, c.where, err)
			}
			if c.located && !strings.Contains(err.Error(), "m.ail:") {
				t.Fatalf("diagnostic has no source position: %v", err)
			}
		})
	}
}

// Controls: the same shapes at ordinary sizes still check.
func TestCheckBlowup_SmallInstancesStillCheck(t *testing.T) {
	cases := map[string]string{
		"let doubling":             blowupLetDoubling(12),
		"record doubling":          blowupRecordDoubling(8),
		"alias chain materialised": blowupAliasChain(12, "pure func g(x: P0[int]) -> bool = match x { Some(_) => true, None => false }"),
		"alias chain used":         blowupAliasChain(12, "export pure func f(x: P0[int]) -> P0[int] = x\npure func g(x: P0[int]) -> bool = match f(x) { Some(_) => true, None => false }"),
		"alias DAG used":           blowupAliasDAG(12, "export pure func f(x: DLAST) -> DLAST = x\npure func g(x: DLAST) -> DLAST = f(f(x))"),
		"two alias DAGs unified":   blowupTwinDAGs(12),
		// A long expression is not a deep type: 5,000 list elements are one
		// [int], however long the literal (the 1,000-deep bound is on types).
		"5,000-element list literal": "module m\nexport func main() -> () ! {IO} {\n  let xs = [" + strings.TrimSuffix(strings.Repeat("1, ", 5000), ", ") + "];\n  println(show(xs))\n}\n",
		"nested record pair":         "module m\nexport func main() -> () ! {IO} {\n  let r0 = {v: \"s\"};\n  let r1 = {a: r0, b: r0};\n  let r2 = {a: r1, b: r1};\n  println(r2.a.b.v)\n}\n",
	}
	for name, src := range cases {
		t.Run(name, func(t *testing.T) {
			if err := checkBounded(t, src, blowupLimit, blowupAlloc); err != nil {
				t.Fatalf("valid program rejected: %v", err)
			}
		})
	}
}
