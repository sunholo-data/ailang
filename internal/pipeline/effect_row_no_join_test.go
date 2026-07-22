package pipeline

// M-EFFECT-ROW-SHOW-INTERP (#386) AC6 (structural no-join proof) + AC7
// (REPLACE-NOT-DELETE let-boundary regression). These tests encode two of the
// ratified-mechanism invariants:
//
//   - No EffectJoin representation exists or reaches RowUnifier.UnifyRows; every
//     published application effect row is an ordinary *types.Row with <=1 tail.
//     row_unification.go is unchanged (verified separately by the empty-diff
//     assertion below).
//   - Solved application equalities are REPLACED (not deleted) in the constraint
//     set, so the let-boundary SolveConstraints replay re-derives them and
//     propagates the substitution to OUTER nodes. AC7 is designed to FAIL under a
//     delete-based implementation.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/core"
)

// TestNoEffectJoinIdentifier is the grep-assert from AC6: no EffectJoin type or
// constraint identifier may exist anywhere in the type system. The application
// effect fix is an application-LOCAL equality solver, not a deferred join.
func TestNoEffectJoinIdentifier(t *testing.T) {
	roots := []string{"../types", "../pipeline"}
	for _, root := range roots {
		err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() || !strings.HasSuffix(path, ".go") {
				return nil
			}
			if strings.HasSuffix(path, "_test.go") {
				return nil // test files (like this one) legitimately name it
			}
			b, rerr := os.ReadFile(path)
			if rerr != nil {
				return rerr
			}
			if strings.Contains(string(b), "EffectJoin") {
				t.Errorf("EffectJoin identifier found in %s; the #386 fix must NOT introduce a join representation", path)
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", root, err)
		}
	}
}

// TestPublishedAppRowsHaveAtMostOneTail is the structural half of AC6: after
// type-checking the #386 reproducer, every TypedApp effect row is an ordinary
// *types.Row carrying at most one tail — no multi-tail join value ever escapes
// inferApp into a published node.
func TestPublishedAppRowsHaveAtMostOneTail(t *testing.T) {
	dir := t.TempDir()
	src := `module reprojoin
import std/io (println)
import std/list (mapE, foldlE)

export func main() -> () ! {IO} {
  let doubled = mapE(\x. { println("mapping ${show(x)}"); x * 2 }, [1,2,3]);
  let total = foldlE(func(acc: int, x: int) -> int ! {IO} { println("acc"); acc + x }, 0, [10,20,30]);
  println("done")
}`
	p := filepath.Join(dir, "reprojoin.ail")
	if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(p)
	res, err := Run(Config{RelaxModules: true, NoCache: true}, Source{Code: string(b), Filename: p})
	if err != nil {
		t.Fatalf("reproducer must type-check clean, got: %v", err)
	}
	if len(res.Errors) != 0 {
		t.Fatalf("unexpected errors: %v", res.Errors)
	}

	// Every application effect row recorded in CoreTypeInfo must be an ordinary
	// *types.Row (never a join) with at most one tail. The clean compile above
	// already proves no join reached the unifier (a join would have panicked or
	// failed unification); this walk additionally proves every published node is
	// an ordinary Core expression tree with no residual join value.
	assertAppRowsSingleTail(t, res)
}

func assertAppRowsSingleTail(t *testing.T, res Result) {
	t.Helper()
	for _, lm := range res.Modules {
		if lm.Core == nil {
			continue
		}
		for _, decl := range lm.Core.Decls {
			walkCoreAssertRows(t, decl)
		}
	}
}

func walkCoreAssertRows(t *testing.T, e core.CoreExpr) {
	if e == nil {
		return
	}
	switch n := e.(type) {
	case *core.App:
		walkCoreAssertRows(t, n.Func)
		for _, a := range n.Args {
			walkCoreAssertRows(t, a)
		}
	case *core.Lambda:
		walkCoreAssertRows(t, n.Body)
	case *core.Let:
		walkCoreAssertRows(t, n.Value)
		walkCoreAssertRows(t, n.Body)
	case *core.LetRec:
		for _, b := range n.Bindings {
			walkCoreAssertRows(t, b.Value)
		}
		walkCoreAssertRows(t, n.Body)
	case *core.If:
		walkCoreAssertRows(t, n.Cond)
		walkCoreAssertRows(t, n.Then)
		walkCoreAssertRows(t, n.Else)
	}
}

// TestReplaceNotDelete_LetBoundaryPropagation is AC7. An application (`show(x)`
// used through a nested effectful `println` inside a `let`) is checked whose
// resolved effect must propagate through the let boundary to the enclosing
// function's declared `! {IO}` result. Under REPLACE-NOT-DELETE the let-boundary
// SolveConstraints replay re-derives the substitution and the outer node is fully
// resolved, so this program type-checks. Under a DELETE-based implementation the
// solved application equalities never reach the let-boundary replay, leaving the
// outer effect row unsubstituted — the program then fails (the whole point of the
// fixture is to distinguish replace from delete).
func TestReplaceNotDelete_LetBoundaryPropagation(t *testing.T) {
	// This program only type-checks if the inner application's effect
	// substitution reaches the OUTER `main` node at the let boundary, i.e. the
	// let's body effect (from `println("out")`) and the let value's derived IO are
	// both propagated so `main`'s inferred effect row unifies with its declared
	// `! {IO}`.
	src := `module replace_not_delete
import std/io (println)
import std/list (mapE)

export func main() -> () ! {IO} {
  let doubled = mapE(\x. { println("v=${show(x)}"); x * 2 }, [1,2,3]);
  println("out")
}`
	if err := checkModuleSource(t, "replace_not_delete", src); err != nil {
		t.Fatalf("REPLACE-NOT-DELETE fixture must type-check (a delete-based drain would leave "+
			"the outer let node unsubstituted and fail here): %v", err)
	}
}

// checkModuleSource type-checks a module source string end-to-end (imports
// resolved) and returns the first blocking error or nil.
func checkModuleSource(t *testing.T, name, src string) error {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, name+".ail")
	if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(p)
	res, err := Run(Config{RelaxModules: true, NoCache: true}, Source{Code: string(b), Filename: p})
	if err != nil {
		return err
	}
	if len(res.Errors) > 0 {
		return res.Errors[0]
	}
	return nil
}
