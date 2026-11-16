package types

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/ast"
)

// #2308/#2329: a type held as a DAG denotes an exponentially larger tree. The
// traversals below must visit a shared node once; each test builds a 40-level
// doubling (2^40 paths) that a per-path walk could not finish.

// doublingTuple is ((…(leaf, leaf)…), (…)) n levels deep: n+1 nodes, 2^n paths.
func doublingTuple(leaf Type, n int) Type {
	t := leaf
	for i := 0; i < n; i++ {
		t = &TTuple{Elements: []Type{t, t}}
	}
	return t
}

// doublingRecord is {a: R, b: R} n levels deep over leaf.
func doublingRecord(leaf Type, n int) Type {
	t := leaf
	for i := 0; i < n; i++ {
		t = &TRecord{Fields: map[string]Type{"a": t, "b": t}}
	}
	return t
}

func within(t *testing.T, limit time.Duration, f func()) {
	t.Helper()
	done := make(chan struct{})
	go func() { defer close(done); f() }()
	select {
	case <-done:
	case <-time.After(limit):
		t.Fatalf("did not finish within %s: a shared node is walked once per path", limit)
	}
}

func TestDataArrows_SharedNodeVisitedOnce(t *testing.T) {
	fn := &TFunc2{Params: []Type{TInt}, Return: TInt, EffectRow: &Row{Kind: EffectRow, Labels: map[string]Type{}, Tail: &RowVar{Name: "e", Kind: EffectRow}}}
	var arrows []*TFunc2
	within(t, 5*time.Second, func() { arrows = dataArrowsWith(doublingTuple(fn, 40), nil) })
	if len(arrows) != 1 || arrows[0] != fn {
		t.Fatalf("want the one arrow once, got %d", len(arrows))
	}
}

func TestTypeNamePropagation_SharedRecordVisitedOnce(t *testing.T) {
	tc := NewCoreTypeChecker()
	tc.aliasEnv = map[string]Type{"Pair": &TRecord{Fields: map[string]Type{"x": TInt, "y": TInt}}}
	leaf := &TRecord{Fields: map[string]Type{"x": TInt, "y": TInt}}
	other := &TRecord{Fields: map[string]Type{"x": TInt, "z": TInt}} // same arity, other names
	tc.CoreTI = CoreTypeInfo{1: doublingRecord(leaf, 40), 2: other}
	within(t, 5*time.Second, tc.propagateTypeNameToCoreTI)
	if leaf.TypeName != "Pair" {
		t.Fatalf("leaf record matching alias Pair got TypeName %q", leaf.TypeName)
	}
	if other.TypeName != "" {
		t.Fatalf("record with other field names got TypeName %q", other.TypeName)
	}
}

func TestTypeSizer_CountsTheTreeInDAGTime(t *testing.T) {
	s := newTypeSizer()
	// (int, [string]) is four nodes; a record counts its fields once (#2308).
	if n := s.size(&TTuple{Elements: []Type{TInt, &TList{Element: TString}}}); n != 4 {
		t.Fatalf("size((int, [string])) = %d, want 4", n)
	}
	if n := s.size(&TRecord{Fields: map[string]Type{"a": TInt, "b": TInt}}); n != 3 {
		t.Fatalf("size({a: int, b: int}) = %d, want 3", n)
	}
	// 15 doublings: 2^16-1 nodes, exactly at the bound; one more is over it.
	if n := s.size(doublingTuple(TInt, 15)); n != 1<<16-1 {
		t.Fatalf("size(15 doublings) = %d, want %d", n, 1<<16-1)
	}
	var n int64
	within(t, 5*time.Second, func() { n = newTypeSizer().size(doublingTuple(TInt, 40)) })
	if n != 1<<41-1 {
		t.Fatalf("size(40 doublings) = %d, want %d counted in DAG time", n, int64(1<<41-1))
	}
}

// The unification step budget turns a walk that never holds a large type into
// the coded error at the declaration boundary (a small limit stands in for
// maxUnifySteps; a 10-level tuple tree with a distinct variable at each of
// its 1024 leaves takes over 2000 steps to unify with a ground doubling).
func TestUnifyWork_BudgetIsADiagnostic(t *testing.T) {
	u := NewUnifier()
	u.work = &unifyWork{limit: 1000}
	leaves := 0
	var tree func(int) Type
	tree = func(n int) Type {
		if n == 0 {
			leaves++
			return &TVar2{Name: fmt.Sprintf("v%d", leaves), Kind: Star}
		}
		return &TTuple{Elements: []Type{tree(n - 1), tree(n - 1)}}
	}
	a, b := doublingTuple(TInt, 10), tree(10)
	err := func() (err error) {
		defer recoverLatentDataDepth(&err)
		_, err = u.Unify(a, b, Substitution{})
		return err
	}()
	if err == nil || !strings.Contains(err.Error(), TC_TYPE_SIZE_001) || !strings.Contains(err.Error(), "more than 1000 steps") {
		t.Fatalf("want %s on the step budget, got %v", TC_TYPE_SIZE_001, err)
	}
	if newUnifyWork(ast.Pos{}).limit != maxUnifySteps {
		t.Fatal("a declaration's unifier is not bounded by maxUnifySteps")
	}
}

// Two ground nullary aliases unify once per pair (#2329); a pair with a
// variable in its expansion is never shared.
func TestUnify_GroundAliasPairsShared(t *testing.T) {
	aliases := map[string]Type{}
	for _, side := range []string{"D", "E"} {
		aliases[side+"0"] = TInt
		for i := 1; i < 60; i++ {
			var elems []Type
			for j := max(0, i-5); j < i; j++ {
				elems = append(elems, &TCon{Name: fmt.Sprintf("%s%d", side, j)})
			}
			aliases[fmt.Sprintf("%s%d", side, i)] = &TTuple{Elements: append(elems, TInt)}
		}
	}
	aliases["V"] = &TList{Element: &TVar2{Name: "a", Kind: Star}}
	u := NewUnifierWithAliases(aliases)
	within(t, 5*time.Second, func() {
		if _, err := u.Unify(&TCon{Name: "D59"}, &TCon{Name: "E59"}, Substitution{}); err != nil {
			t.Errorf("isomorphic alias DAGs did not unify: %v", err)
		}
	})
	if _, ok := u.groundAliasPair(&TCon{Name: "V"}, &TCon{Name: "V"}); ok {
		t.Fatal("an alias with a variable in its expansion was treated as ground")
	}
}

func TestTypeSizeCheck_ReportsTheCode(t *testing.T) {
	c := NewTypeSizeCheck()
	if err := c.Check(doublingTuple(TInt, 15), "exported f"); err != nil {
		t.Fatalf("a type at the bound was rejected: %v", err)
	}
	err := c.Check(doublingTuple(TInt, 16), "exported f")
	// The author is told the measured size against the limit, and what to do.
	if err == nil || !strings.Contains(err.Error(), "131,071 nodes") || !strings.Contains(err.Error(), "limit of 65,536") || !strings.Contains(err.Error(), "wrap the repeated part") {
		t.Fatalf("diagnostic does not name the measured size, the limit and the remedy: %v", err)
	}
	if err == nil || !strings.Contains(err.Error(), TC_TYPE_SIZE_001) || !strings.Contains(err.Error(), "exported f") {
		t.Fatalf("want %s naming exported f, got %v", TC_TYPE_SIZE_001, err)
	}
}
