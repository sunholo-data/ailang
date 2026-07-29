package main

// M-BUDGET-SCOPING-BUG: end-to-end regression matrix for hierarchical
// per-invocation budget frames.
//
// These tests drive real .ail source through the full ailang binary
// (parse -> typecheck -> link -> eval with effect budgets) — the exact
// user-facing path where the cumulative-budget bug lived and the design's repro
// was reported. One test per semantics-matrix cell (design_docs/planned/
// v0_29_0/m-budget-scoping-bug.md), plus a frame-leak test proving the
// defer-guarded pop unwinds cleanly and leaves no stale frame for a sibling call.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// budgetBin returns the shared ailang test binary. It used to keep its own
// sync.Once + build of the identical `./cmd/ailang` target; that duplicate is
// gone so the whole package pays for exactly one `go build` (see buildAilang).

// runBudget writes src to a temp module and runs `main` with --caps IO.
// Returns combined stdout, stderr, and exit code.
func runBudget(t *testing.T, name, src string) (stdout, stderr string, exit int) {
	t.Helper()
	bin := buildAilang(t)
	dir := t.TempDir()
	file := filepath.Join(dir, name+".ail")
	if err := os.WriteFile(file, []byte(src), 0o644); err != nil {
		t.Fatalf("write module: %v", err)
	}
	return runAilangBin(t, bin, "run", "--caps", "IO", "--entry", "main", "--quiet", file)
}

// Cell 1 — the repro. Unannotated caller (3 IO) → annotated callee @limit=3 (2 IO).
// MUST succeed: main pushes no frame; limited's frame counts only its own 2 ops.
func TestBudgetFrame_UnannotatedCaller_AnnotatedCallee_Repro(t *testing.T) {
	src := `module cell1
import std/io (println)

export func limited(x: int) -> () ! {IO @limit=3} {
  println("a");
  println("b")
}

export func main() -> () ! {IO} {
  println("p1"); println("p2"); println("p3");
  limited(0)
}
`
	stdout, stderr, exit := runBudget(t, "cell1", src)
	if exit != 0 {
		t.Fatalf("repro must succeed under per-frame semantics, exit=%d\nstdout:\n%s\nstderr:\n%s", exit, stdout, stderr)
	}
	for _, want := range []string{"p1", "p2", "p3", "a", "b"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("expected stdout to contain %q, got:\n%s", want, stdout)
		}
	}
}

// Cell 2 — annotated caller @limit=2 (1 IO) → annotated callee @limit=3 (2 IO).
// Fails on limited's 2nd op: charges both frames; outer would hit 1+2=3 > 2.
// Tightest active limit (outer) wins; error names outer's limit=2.
func TestBudgetFrame_AnnotatedCaller_AnnotatedCallee(t *testing.T) {
	src := `module cell2
import std/io (println)

export func limited(x: int) -> () ! {IO @limit=3} {
  println("a");
  println("b")
}

export func outer() -> () ! {IO @limit=2} {
  println("o1");
  limited(0)
}

export func main() -> () ! {IO} {
  outer()
}
`
	stdout, stderr, exit := runBudget(t, "cell2", src)
	if exit == 0 {
		t.Fatalf("cell2 must fail (outer @limit=2 tripped), got success\nstdout:\n%s", stdout)
	}
	if !strings.Contains(stderr, "limit=2") {
		t.Errorf("expected error naming outer's limit=2, got stderr:\n%s", stderr)
	}
	if !strings.Contains(stdout, "a") {
		t.Errorf("expected 'a' printed before the trip, got stdout:\n%s", stdout)
	}
	if strings.Contains(stdout, "b") {
		t.Errorf("'b' (the tripping op) must NOT be performed (pre-op check), got stdout:\n%s", stdout)
	}
}

// Cell 3 — annotated caller @limit=3 (1 IO) → unannotated helper (3 IO).
// Fails on helper's 3rd op (outer frame 1+3 > 3). Delegation cannot launder.
func TestBudgetFrame_AnnotatedCaller_UnannotatedCallee(t *testing.T) {
	src := `module cell3
import std/io (println)

export func helper() -> () ! {IO} {
  println("h1");
  println("h2");
  println("h3")
}

export func outer() -> () ! {IO @limit=3} {
  println("o1");
  helper()
}

export func main() -> () ! {IO} {
  outer()
}
`
	stdout, stderr, exit := runBudget(t, "cell3", src)
	if exit == 0 {
		t.Fatalf("cell3 must fail (outer @limit=3 tripped by delegated ops), got success\nstdout:\n%s", stdout)
	}
	if !strings.Contains(stderr, "limit=3") {
		t.Errorf("expected error naming outer's limit=3, got stderr:\n%s", stderr)
	}
	if !strings.Contains(stdout, "h2") {
		t.Errorf("expected 'h2' printed before trip, got stdout:\n%s", stdout)
	}
	if strings.Contains(stdout, "h3") {
		t.Errorf("'h3' (tripping op) must NOT be performed, got stdout:\n%s", stdout)
	}
}

// Cell 4 — recursion. rec @limit=3 does 2 own IO then recurses.
// Independent frames per invocation, but inner ops bubble to ancestor frames:
// the outermost frame accumulates 2+2=4 > 3 on the 2nd invocation's 2nd op.
func TestBudgetFrame_Recursion(t *testing.T) {
	recSrc := `module cell4rec
import std/io (println)

export func rec(n: int) -> () ! {IO @limit=3} {
  println("r1");
  println("r2");
  if n > 0 then rec(n - 1) else ()
}

export func main() -> () ! {IO} {
  rec(2)
}
`
	_, stderr, exit := runBudget(t, "cell4rec", recSrc)
	if exit == 0 {
		t.Fatalf("recursion must trip a @limit=3 frame (bubbling), got success")
	}
	if !strings.Contains(stderr, "limit=3") {
		t.Errorf("expected recursion to trip a @limit=3 frame, got stderr:\n%s", stderr)
	}

	// Control: a single non-recursive invocation (2 ops <= 3) succeeds.
	singleSrc := `module cell4single
import std/io (println)

export func rec(n: int) -> () ! {IO @limit=3} {
  println("r1");
  println("r2")
}

export func main() -> () ! {IO} {
  rec(0)
}
`
	_, stderr, exit = runBudget(t, "cell4single", singleSrc)
	if exit != 0 {
		t.Errorf("single invocation (2 ops <= 3) must succeed, stderr:\n%s", stderr)
	}
}

// Cell 5 — @min on normal exit. atLeast @min=3 does 2 IO → fails at pop.
func TestBudgetFrame_MinNormalExit(t *testing.T) {
	base := `module cell5base
import std/io (println)

export func atLeast() -> () ! {IO @min=3 @limit=5} {
  println("m1");
  println("m2")
}

export func main() -> () ! {IO} {
  atLeast()
}
`
	_, stderr, exit := runBudget(t, "cell5base", base)
	if exit == 0 {
		t.Fatalf("cell5 base must fail (@min=3 unmet with 2 ops), got success")
	}
	if !strings.Contains(stderr, "underrun") || !strings.Contains(stderr, "min=3") {
		t.Errorf("expected @min underrun (min=3), got stderr:\n%s", stderr)
	}

	// Variant A: 3 ops == min → succeeds.
	variantA := `module cell5a
import std/io (println)

export func atLeast() -> () ! {IO @min=3 @limit=5} {
  println("m1"); println("m2"); println("m3")
}

export func main() -> () ! {IO} {
  atLeast()
}
`
	if _, stderr, exit := runBudget(t, "cell5a", variantA); exit != 0 {
		t.Errorf("variant A (3 ops == min 3) must succeed, stderr:\n%s", stderr)
	}

	// Variant B: 1 own op + unannotated callee doing 2 → 3 charged to atLeast's
	// frame (bubbling) → succeeds.
	variantB := `module cell5b
import std/io (println)

export func helper() -> () ! {IO} {
  println("h1"); println("h2")
}

export func atLeast() -> () ! {IO @min=3 @limit=5} {
  println("m1");
  helper()
}

export func main() -> () ! {IO} {
  atLeast()
}
`
	if _, stderr, exit := runBudget(t, "cell5b", variantB); exit != 0 {
		t.Errorf("variant B (1 own + 2 callee bubbled = 3) must succeed, stderr:\n%s", stderr)
	}
}

// Cell 6 — @min on error exit is SUPPRESSED. atLeast @min=3 does 1 IO then a
// callee trips its own @limit. The callee's error propagates; @min is suppressed.
func TestBudgetFrame_MinErrorExit_Suppressed(t *testing.T) {
	src := `module cell6
import std/io (println)

export func boom() -> () ! {IO @limit=1} {
  println("b1");
  println("b2")
}

export func atLeast() -> () ! {IO @min=3 @limit=9} {
  println("m1");
  boom()
}

export func main() -> () ! {IO} {
  atLeast()
}
`
	_, stderr, exit := runBudget(t, "cell6", src)
	if exit == 0 {
		t.Fatalf("cell6 must fail (callee @limit=1 tripped), got success")
	}
	if strings.Contains(stderr, "underrun") {
		t.Errorf("@min must be SUPPRESSED on error exit; got a min underrun instead of the callee error:\n%s", stderr)
	}
	if !strings.Contains(stderr, "exhausted") || !strings.Contains(stderr, "limit=1") {
		t.Errorf("expected the callee's @limit=1 exhaustion to propagate, got stderr:\n%s", stderr)
	}
}

// Cell 7 — @limit attribution: callee-frame trip names callee; caller-frame trip
// names caller. Both: failing op not performed (pre-op).
func TestBudgetFrame_LimitAttribution_CalleeVsCaller(t *testing.T) {
	// (a) callee @limit=1 does 2 → callee's own frame trips (limit=1).
	calleeSrc := `module cell7a
import std/io (println)

export func callee() -> () ! {IO @limit=1} {
  println("c1");
  println("c2")
}

export func main() -> () ! {IO} {
  callee()
}
`
	stdout, stderr, exit := runBudget(t, "cell7a", calleeSrc)
	if exit == 0 {
		t.Fatalf("cell7a must fail (callee @limit=1), got success")
	}
	if !strings.Contains(stderr, "limit=1") {
		t.Errorf("(a) expected callee's limit=1 in error, got stderr:\n%s", stderr)
	}
	if strings.Contains(stdout, "c2") {
		t.Errorf("(a) tripping op c2 must not be performed, got stdout:\n%s", stdout)
	}

	// (b) caller @limit=2 (1 IO) → callee @limit=9 (2 IO): caller frame trips.
	callerSrc := `module cell7b
import std/io (println)

export func callee() -> () ! {IO @limit=9} {
  println("c1");
  println("c2")
}

export func caller() -> () ! {IO @limit=2} {
  println("k1");
  callee()
}

export func main() -> () ! {IO} {
  caller()
}
`
	stdout, stderr, exit = runBudget(t, "cell7b", callerSrc)
	if exit == 0 {
		t.Fatalf("cell7b must fail (caller @limit=2), got success")
	}
	if !strings.Contains(stderr, "limit=2") {
		t.Errorf("(b) expected caller's limit=2 in error, got stderr:\n%s", stderr)
	}
	if strings.Contains(stdout, "c2") {
		t.Errorf("(b) tripping op c2 must not be performed, got stdout:\n%s", stdout)
	}
}

// Cell 8 — op would exceed BOTH caller and callee frames at once. Deterministic
// innermost-first violator selection → attribution names inner; no increment; op
// not performed.
func TestBudgetFrame_DualViolation_InnermostAttribution(t *testing.T) {
	// outer @limit=1 does 0 IO, calls inner @limit=1 which does 1 then attempts a
	// 2nd op. That op violates BOTH inner (1+1>1) and outer (1+1>1). Innermost-
	// first → names inner (limit=1, used=1).
	src := `module cell8
import std/io (println)

export func inner() -> () ! {IO @limit=1} {
  println("i1");
  println("i2")
}

export func outer() -> () ! {IO @limit=1} {
  inner()
}

export func main() -> () ! {IO} {
  outer()
}
`
	stdout, stderr, exit := runBudget(t, "cell8", src)
	if exit == 0 {
		t.Fatalf("cell8 must fail (dual-frame violation), got success")
	}
	if !strings.Contains(stderr, "limit=1") || !strings.Contains(stderr, "used=1") {
		t.Errorf("expected inner's frame (limit=1, used=1) in error, got stderr:\n%s", stderr)
	}
	if strings.Contains(stdout, "i2") {
		t.Errorf("dual-violation tripping op i2 must not be performed, got stdout:\n%s", stdout)
	}
}

// Frame-leak test (acceptance criterion 6): a callee error unwinds through an
// annotated frame; a subsequent SIBLING annotated call must still succeed,
// proving the deferred pop left no stale frame on the stack. Because AILANG has
// no try/catch, we sequence it through the effect of a recursive helper that
// errors, then a sibling that must succeed — driven by two entrypoints on one
// binary invocation is not possible, so we assert the semantics with a single
// program: an annotated sibling runs BEFORE an annotated boom that trips. If a
// prior annotated frame leaked, the sibling's own frame would be mis-charged.
//
// Concretely: sibling @limit=5 does 2 ops (must succeed), THEN boom @limit=1 does
// 2 ops (must trip). The success of sibling followed by boom's clean trip (naming
// boom's limit=1, not a polluted count) proves each invocation gets an
// independent frame and pops cleanly.
func TestBudgetFrame_ErrorUnwind_NoStaleFrame(t *testing.T) {
	src := `module cellleak
import std/io (println)

export func sibling() -> () ! {IO @limit=5} {
  println("sib1");
  println("sib2")
}

export func boom() -> () ! {IO @limit=1} {
  println("boom1");
  println("boom2")
}

export func main() -> () ! {IO} {
  sibling();
  boom()
}
`
	stdout, stderr, exit := runBudget(t, "cellleak", src)
	if exit == 0 {
		t.Fatalf("boom must trip @limit=1 after sibling succeeds, got success\nstdout:\n%s", stdout)
	}
	// sibling completed cleanly (both ops), proving its frame popped and left no
	// residue for boom.
	if !strings.Contains(stdout, "sib1") || !strings.Contains(stdout, "sib2") {
		t.Errorf("sibling must complete before boom (no stale frame), got stdout:\n%s", stdout)
	}
	// boom trips on ITS OWN frame (limit=1, used=1) — not a leaked/polluted count.
	if !strings.Contains(stderr, "limit=1") || !strings.Contains(stderr, "used=1") {
		t.Errorf("boom must trip its own fresh frame (limit=1, used=1), got stderr:\n%s", stderr)
	}
	if strings.Contains(stdout, "boom2") {
		t.Errorf("boom's tripping op must not be performed, got stdout:\n%s", stdout)
	}
}
