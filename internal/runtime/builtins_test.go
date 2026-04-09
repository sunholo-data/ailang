package runtime

import (
	"sync"
	"testing"

	"github.com/petermattis/goid"
	"github.com/sunholo/ailang/internal/eval"
)

func TestGetEffContext_FastPath(t *testing.T) {
	// With no forked evaluators, getEffContext should use the shared evaluator
	// without calling goid.Get() or doing a sync.Map lookup.
	br := &BuiltinRegistry{
		builtins:  make(map[string]eval.Value),
		evaluator: eval.NewCoreEvaluator(),
	}

	// goroutineEvalCount starts at 0 → fast path
	if br.goroutineEvalCount.Load() != 0 {
		t.Fatal("expected goroutineEvalCount to be 0 initially")
	}

	ctx := br.getEffContext()
	if ctx == nil {
		t.Fatal("expected non-nil EffContext from shared evaluator")
	}
}

func TestGetEffContext_ForkedEvaluator(t *testing.T) {
	sharedEval := eval.NewCoreEvaluator()
	forkedEval := eval.NewCoreEvaluator()

	br := &BuiltinRegistry{
		builtins:  make(map[string]eval.Value),
		evaluator: sharedEval,
	}

	// Register forked evaluator for this goroutine
	br.SetGoroutineEvaluator(forkedEval)
	defer br.ClearGoroutineEvaluator()

	if br.goroutineEvalCount.Load() != 1 {
		t.Fatalf("expected goroutineEvalCount=1, got %d", br.goroutineEvalCount.Load())
	}

	// getEffContext should return the forked evaluator's context
	ctx := br.getEffContext()
	if ctx == nil {
		t.Fatal("expected non-nil EffContext from forked evaluator")
	}
}

func TestGetEffContext_ClearRestoresFastPath(t *testing.T) {
	br := &BuiltinRegistry{
		builtins:  make(map[string]eval.Value),
		evaluator: eval.NewCoreEvaluator(),
	}

	br.SetGoroutineEvaluator(eval.NewCoreEvaluator())
	if br.goroutineEvalCount.Load() != 1 {
		t.Fatal("expected count=1 after Set")
	}

	br.ClearGoroutineEvaluator()
	if br.goroutineEvalCount.Load() != 0 {
		t.Fatal("expected count=0 after Clear — fast path should reactivate")
	}

	// Should still work via fast path
	ctx := br.getEffContext()
	if ctx == nil {
		t.Fatal("expected non-nil EffContext after clearing fork")
	}
}

func TestGetEffContext_ConcurrentForks(t *testing.T) {
	br := &BuiltinRegistry{
		builtins:  make(map[string]eval.Value),
		evaluator: eval.NewCoreEvaluator(),
	}

	const numGoroutines = 10
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			forked := eval.NewCoreEvaluator()
			br.SetGoroutineEvaluator(forked)
			defer br.ClearGoroutineEvaluator()

			// Each goroutine should get a valid context
			ctx := br.getEffContext()
			if ctx == nil {
				t.Error("expected non-nil EffContext in concurrent goroutine")
			}
		}()
	}

	wg.Wait()

	// After all goroutines complete, count should be back to 0
	if count := br.goroutineEvalCount.Load(); count != 0 {
		t.Fatalf("expected goroutineEvalCount=0 after all clears, got %d", count)
	}
}

func TestGoidGet_Consistent(t *testing.T) {
	// Verify goid.Get() returns a consistent value within the same goroutine
	id1 := goid.Get()
	id2 := goid.Get()
	if id1 != id2 {
		t.Fatalf("goid.Get() returned different values: %d vs %d", id1, id2)
	}
	if id1 <= 0 {
		t.Fatalf("expected positive goroutine ID, got %d", id1)
	}
}

func TestGoidGet_DifferentGoroutines(t *testing.T) {
	mainID := goid.Get()
	ch := make(chan int64, 1)
	go func() {
		ch <- goid.Get()
	}()
	childID := <-ch
	if mainID == childID {
		t.Fatalf("expected different goroutine IDs, both got %d", mainID)
	}
}
