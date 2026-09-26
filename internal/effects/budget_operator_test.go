package effects

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"

	"github.com/sunholo-data/ailang/internal/eval"
)

// M-EXECUTOR-POLICY-HARDENING M4 (AC8): the operator budget is one shared
// ceiling for the whole run.

func budgetCtx(t *testing.T, limits map[string]int) *EffContext {
	t.Helper()
	ctx := NewEffContext(nil)
	ctx.Env.Sandbox = t.TempDir()
	ctx.Grant(NewCapability("FS"))
	ctx.Grant(NewCapability("IO"))
	ctx.SetOperatorBudget(NewOperatorBudget(limits))
	t.Cleanup(func() { _ = ctx.CloseFSRoot() })
	return ctx
}

func writeN(ctx *EffContext, name string) error {
	_, err := Call(ctx, "FS", "writeFile", []eval.Value{&eval.StringValue{Value: name}, &eval.StringValue{Value: "x"}})
	return err
}

// Exactly N operations, then denial BEFORE the side effect.
func TestOperatorBudget_ExactlyN(t *testing.T) {
	ctx := budgetCtx(t, map[string]int{"FS": 3})
	for i := 0; i < 3; i++ {
		if err := writeN(ctx, "f"+strconv.Itoa(i)+".txt"); err != nil {
			t.Fatalf("op %d within budget: %v", i, err)
		}
	}
	err := writeN(ctx, "f3.txt")
	var obe *OperatorBudgetError
	if !errors.As(err, &obe) || obe.Effect != "FS" || obe.Limit != 3 {
		t.Fatalf("4th op must be the operator denial, got %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(ctx.Env.Sandbox, "f3.txt")); statErr == nil {
		t.Fatal("the denied op had a side effect")
	}
	// Another effect with no ceiling is unlimited.
	for i := 0; i < 10; i++ {
		if _, err := Call(ctx, "IO", "print", []eval.Value{&eval.StringValue{Value: ""}}); err != nil {
			t.Fatalf("IO has no ceiling: %v", err)
		}
	}
}

// The ceiling survives WithBudget scopes (nested calls / imports get a
// derived context) and is never reset by them.
func TestOperatorBudget_SharedAcrossScopes(t *testing.T) {
	ctx := budgetCtx(t, map[string]int{"FS": 2})
	child := ctx.WithBudget(nil)
	hundred := 100
	grandchild := child.WithBudget(NewBudgetContext(map[string]*int{"FS": &hundred}))
	if err := writeN(child, "a.txt"); err != nil {
		t.Fatal(err)
	}
	if err := writeN(grandchild, "b.txt"); err != nil {
		t.Fatal(err)
	}
	if err := writeN(ctx, "c.txt"); err == nil {
		t.Fatal("the third op across three scopes must be denied")
	}
	clone := ctx.Clone().(*EffContext)
	if err := writeN(clone, "d.txt"); err == nil {
		t.Fatal("a Clone shares the ceiling")
	}
}

// A source @limit larger than the ceiling cannot raise it; a smaller one
// still applies underneath it. --no-budgets does not touch it.
func TestOperatorBudget_SourceLimitsAndNoBudgetsCannotRaise(t *testing.T) {
	ctx := budgetCtx(t, map[string]int{"FS": 1})
	ctx.PushBudgetFrame("f", map[string]int{"FS": 50}, nil)
	if err := writeN(ctx, "a.txt"); err != nil {
		t.Fatal(err)
	}
	if err := writeN(ctx, "b.txt"); err == nil {
		t.Fatal("@limit=50 must not raise the ceiling of 1")
	}
	_ = ctx.PopBudgetFrame("f", nil)

	ctx = budgetCtx(t, map[string]int{"FS": 1})
	ctx.DisableBudgets = true
	if err := writeN(ctx, "a.txt"); err != nil {
		t.Fatal(err)
	}
	if err := writeN(ctx, "b.txt"); err == nil {
		t.Fatal("--no-budgets must not lift the operator ceiling")
	}
}

// Explicit zero permits nothing; absent capability still denies.
func TestOperatorBudget_ZeroAndAbsentCap(t *testing.T) {
	ctx := budgetCtx(t, map[string]int{"FS": 0})
	if err := writeN(ctx, "a.txt"); err == nil {
		t.Fatal("FS = 0 must deny the first op")
	}
	ctx = budgetCtx(t, map[string]int{"Net": 5})
	if _, err := Call(ctx, "Net", "httpGet", []eval.Value{&eval.StringValue{Value: "https://x/"}}); err == nil {
		t.Fatal("a budget does not grant the capability")
	}
}

// The wrapper's charge scope dedups the wrapper and the impl: an op that
// passes through both charges once.
func TestOperatorBudget_ChargedOncePerLogicalOp(t *testing.T) {
	ctx := budgetCtx(t, map[string]int{"FS": 1})
	// Simulate the runtime builtin wrapper: charge, open a scope, then the
	// impl's own Call (which charges again on the baseline path).
	if err := ctx.RequireCapWithBudget("FS", ""); err != nil {
		t.Fatal(err)
	}
	ctx.BeginBudgetChargeScope()
	if err := writeN(ctx, "a.txt"); err != nil {
		t.Fatalf("nested charge inside the scope must not count: %v", err)
	}
	ctx.EndBudgetChargeScope()
	if err := writeN(ctx, "b.txt"); err == nil {
		t.Fatal("the second logical op must be denied")
	}
	if got := ctx.OperatorBudgetUsage()["FS"]; got != 1 {
		t.Fatalf("usage = %d, want 1", got)
	}
}

// Concurrent chargers (async Stream handlers) never exceed the ceiling.
// Run with -race.
func TestOperatorBudget_ConcurrentChargeIsExact(t *testing.T) {
	b := NewOperatorBudget(map[string]int{"IO": 100})
	var wg sync.WaitGroup
	var mu sync.Mutex
	ok := 0
	for g := 0; g < 8; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 50; i++ {
				if err := b.Charge("IO", ""); err == nil {
					mu.Lock()
					ok++
					mu.Unlock()
				}
			}
		}()
	}
	wg.Wait()
	if ok != 100 || b.Usage()["IO"] != 100 {
		t.Fatalf("granted %d, used %d; want exactly 100", ok, b.Usage()["IO"])
	}
}
