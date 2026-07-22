package pipeline

// M-EFFECT-ROW-SHOW-INTERP (#386): effect rows must be preserved across pure
// nested calls inside effectful applications, and each use of an imported
// row-polymorphic combinator (mapE/filterE/foldlE/flatMapE/forEachE) must get
// FRESH row variables.
//
// These are the non-vacuity anchor tests. In M1 the "must accept" matrix and the
// unannotated-println(show(x)) "must reject" case are RED (the bug is live). M2
// (application-local solver + REPLACE-NOT-DELETE) and M3 (row-var generalization)
// make them GREEN. The controls (println("literal") missing IO, explicit `! {}`
// body doing IO via a nested pure call, and genuine incompatible closed rows)
// must REMAIN rejected the whole time — a run where the controls stop rejecting
// is a soundness REGRESSION, not a pass.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// check386 writes src (whose module name must equal `name`) to a temp file and
// runs the full module pipeline (resolving `import std/list` etc.), returning the
// first blocking error or nil.
func check386(t *testing.T, name, src string) error {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, name+".ail")
	if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
		t.Fatalf("write %s: %v", p, err)
	}
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read %s: %v", p, err)
	}
	cfg := Config{RelaxModules: true, NoCache: true}
	res, runErr := Run(cfg, Source{Code: string(b), Filename: p})
	if runErr != nil {
		return runErr
	}
	if len(res.Errors) > 0 {
		return res.Errors[0]
	}
	return nil
}

// mustAccept386 lists the programs that are valid and MUST type/effect-check
// clean once #386 is fixed. Each is derived directly from the design doc's
// live-verified matrix (§Root-Cause Analysis, §Testing Strategy).
var mustAccept386 = []struct {
	name string
	src  string
}{
	{
		// 1. Minimal reproducer: interpolation/show inside effectful mapE, then IO foldlE.
		name: "minimal_repro",
		src: `module minimal_repro
import std/io (println)
import std/list (mapE, filterE, foldlE, flatMap)

export func main() -> () ! {IO} {
  let doubled = mapE(\x. { println("mapping ${show(x)}"); x * 2 }, [1,2,3]);
  let total = foldlE(func(acc: int, x: int) -> int ! {IO} { println("acc"); acc + x }, 0, [10,20,30]);
  println("done")
}`,
	},
	{
		// 2. Same program with an UNANNOTATED foldlE lambda (effect inferred).
		name: "unannotated_foldlE_lambda",
		src: `module unannotated_foldlE_lambda
import std/io (println)
import std/list (mapE, foldlE)

export func main() -> () ! {IO} {
  let doubled = mapE(\x. { println("mapping ${show(x)}"); x * 2 }, [1,2,3]);
  let total = foldlE(\acc x. { println("acc"); acc + x }, 0, [10,20,30]);
  println("done")
}`,
	},
	{
		// 3. Direct println(show(x)) inside an effectful mapE callback.
		name: "direct_show_in_callback",
		src: `module direct_show_in_callback
import std/io (println)
import std/list (mapE, foldlE)

export func main() -> () ! {IO} {
  let doubled = mapE(\x. { println(show(x)); x * 2 }, [1,2,3]);
  let total = foldlE(func(acc: int, x: int) -> int ! {IO} { println("acc"); acc + x }, 0, [10,20,30]);
  println("done")
}`,
	},
	{
		// 4. println(intToStr(x)) — non-`show` guard proving no show special-case.
		name: "intToStr_in_callback",
		src: `module intToStr_in_callback
import std/io (println)
import std/string (intToStr)
import std/list (mapE, foldlE)

export func main() -> () ! {IO} {
  let doubled = mapE(\x. { println(intToStr(x)); x * 2 }, [1,2,3]);
  let total = foldlE(func(acc: int, x: int) -> int ! {IO} { println("acc"); acc + x }, 0, [10,20,30]);
  println("done")
}`,
	},
	{
		// 5. Pure mapE callback followed by {IO} foldlE (fresh-instantiation guard,
		//    independent of show).
		name: "pure_mapE_then_io_foldlE",
		src: `module pure_mapE_then_io_foldlE
import std/io (println)
import std/list (mapE, foldlE)

export func main() -> () ! {IO} {
  let doubled = mapE(\x. x * 2, [1,2,3]);
  let total = foldlE(func(acc: int, x: int) -> int ! {IO} { println("acc"); acc + x }, 0, [10,20,30]);
  println("done")
}`,
	},
	{
		// 6. Effectful mapE followed by effectful filterE (second effectful combinator).
		name: "effectful_mapE_then_effectful_filterE",
		src: `module effectful_mapE_then_effectful_filterE
import std/io (println)
import std/list (mapE, filterE)

export func main() -> () ! {IO} {
  let doubled = mapE(\x. { println(show(x)); x * 2 }, [1,2,3]);
  let kept = filterE(func(x: int) -> bool ! {IO} { println("keep"); x > 2 }, [10,20,30]);
  println("done")
}`,
	},
	{
		// 7. Reverse source order: foldlE first, then effectful mapE with show.
		name: "reverse_order_foldlE_then_mapE",
		src: `module reverse_order_foldlE_then_mapE
import std/io (println)
import std/list (mapE, foldlE)

export func main() -> () ! {IO} {
  let total = foldlE(func(acc: int, x: int) -> int ! {IO} { println("acc"); acc + x }, 0, [10,20,30]);
  let doubled = mapE(\x. { println(show(x)); x * 2 }, [1,2,3]);
  println("done")
}`,
	},
	{
		// 8. Two uses of the SAME combinator with different effect rows (one effectful,
		//    one effectful) — each must get a fresh row, no cross-use contamination.
		name: "two_uses_same_combinator",
		src: `module two_uses_same_combinator
import std/io (println)
import std/list (mapE)

export func main() -> () ! {IO} {
  let a = mapE(\x. { println(show(x)); x * 2 }, [1,2,3]);
  let b = mapE(\y. { println("y"); y + 1 }, [4,5,6]);
  println("done")
}`,
	},
	{
		// 9. Pure flatMap + top-level show + effectful foldlE control (flatMap has no
		//    row-polymorphic callback effect to poison).
		name: "pure_flatMap_topshow_effectful_foldlE",
		src: `module pure_flatMap_topshow_effectful_foldlE
import std/io (println)
import std/list (flatMap, foldlE)

export func main() -> () ! {IO} {
  let expanded = flatMap(\x. [x, x], [1,2,3]);
  let s = show(42);
  let total = foldlE(func(acc: int, x: int) -> int ! {IO} { println("acc"); acc + x }, 0, [10,20,30]);
  println("done")
}`,
	},
}

// TestEffectRowShowInterp_MustAccept pins the 9 must-accept fixtures.
//
// M1 STATUS: RED — every fixture currently fails with
//
//	incompatible closed rows: r1 has extra labels [], r2 has extra labels [IO]
//
// because combineEffects drops the unresolved tail to closed {} before the
// pending {IO} constraint is solved, and imported combinators reuse the literal
// row-var name `e`. M2+M3 make these GREEN.
func TestEffectRowShowInterp_MustAccept(t *testing.T) {
	for _, tc := range mustAccept386 {
		t.Run(tc.name, func(t *testing.T) {
			if err := check386(t, tc.name, tc.src); err != nil {
				t.Fatalf("must-accept fixture %q rejected (want clean): %v", tc.name, err)
			}
		})
	}
}

// mustReject386 lists the controls that MUST stay rejected. These prove the fix
// does not "make everything compile" (non-vacuity).
var mustReject386 = []struct {
	name       string
	src        string
	wantSubstr string // substring the diagnostic must contain
}{
	{
		// 1. Unannotated pure function containing println(show(x)) — the soundness
		//    hole. Currently WRONGLY ACCEPTED (RED here in M1), flips to correctly
		//    REJECTED with missing IO in M2.
		name: "unannotated_show_missing_io",
		src: `module unannotated_show_missing_io
import std/io (println)

export func callback(x: int) -> int {
  println(show(x));
  x * 2
}`,
		wantSubstr: "IO",
	},
	{
		// 2. Explicit callback annotation whose body performs IO through a nested
		//    pure call, but whose declared effect row does NOT include IO — must
		//    reject (declared effects must cover body effects). The design names the
		//    `! {}` (explicit-empty) form; that exact spelling is erased by
		//    elaboration (an explicit `! {}` is indistinguishable from no annotation
		//    in the AST — there is no HasEffectAnnotation flag, and adding one is a
		//    parser change explicitly out of scope for this sprint). A non-empty
		//    wrong annotation `! {FS}` exercises the SAME soundness property (an
		//    inline combinator-argument lambda whose declared effects miss a body
		//    effect) and is the control the ratified mechanism enforces. See the
		//    sprint report for the explicit-`! {}` erasure gap.
		name: "inline_lambda_annotation_misses_io",
		src: `module inline_lambda_annotation_misses_io
import std/io (println)
import std/list (mapE)

export func main() -> [int] ! {IO} {
  mapE(func(x: int) -> int ! {FS} { println(show(x)); x * 2 }, [1,2,3])
}`,
		wantSubstr: "IO",
	},
	{
		// 3. Genuine incompatible closed rows unrelated to scheme reuse: main
		//    declares {} but calls println (IO).
		name: "genuine_missing_io_literal",
		src: `module genuine_missing_io_literal
import std/io (println)

export func callback(x: int) -> int {
  println("literal");
  x * 2
}`,
		wantSubstr: "IO",
	},
	{
		// 4. Existing IO/FS non-subsumption: declaring ! {FS} does not satisfy Env.
		name: "fs_does_not_absorb_env",
		src: `module fs_does_not_absorb_env

export func getHome() -> string ! {FS} {
  getEnvOr("HOME", "/tmp")
}`,
		wantSubstr: "Env",
	},
}

// TestEffectRowShowInterp_MustReject pins the 4 controls.
//
// M1 STATUS: unannotated_show_missing_io is RED (wrongly accepted today), proving
// the soundness hole. The other three are GREEN from the start and must STAY green
// through M2/M3 — they are the non-vacuity guard.
func TestEffectRowShowInterp_MustReject(t *testing.T) {
	for _, tc := range mustReject386 {
		t.Run(tc.name, func(t *testing.T) {
			err := check386(t, tc.name, tc.src)
			if err == nil {
				t.Fatalf("must-reject control %q was ACCEPTED (soundness regression): expected rejection mentioning %q", tc.name, tc.wantSubstr)
			}
			if !strings.Contains(err.Error(), tc.wantSubstr) {
				t.Errorf("control %q rejected but diagnostic missing %q: %v", tc.name, tc.wantSubstr, err)
			}
		})
	}
}
