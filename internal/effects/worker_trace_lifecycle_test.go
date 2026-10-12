package effects

import (
	"github.com/sunholo-data/ailang/internal/trace"
	"sync"
	"testing"
)

func TestWorkerShutdownReceiptConcurrentWithEvaluationTrace(t *testing.T) {
	ctx := NewEffContext(nil)
	ctx.Trace = trace.NewCollector()
	ctx.BindWorkerScope()
	a, _ := ctx.BeginWorker()
	w := &ownedTestWorker{done: make(chan struct{})}
	if err := a.Complete(w); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 1000; i++ {
			ctx.Trace.RecordFunctionEnter("active", nil)
			ctx.Trace.RecordEffect("IO", "print", nil, "value")
			ctx.Trace.RecordFunctionExit("active", "()")
		}
	}()
	if err := ctx.CloseWorkers(); err != nil {
		t.Fatal(err)
	}
	if err := ctx.CloseWorkers(); err != nil {
		t.Fatal(err)
	}
	wg.Wait()
	receipts := 0
	for _, event := range ctx.Trace.Events() {
		if event.Effect != nil && event.Effect.OpName == "shutdownWorkers" {
			receipts++
			if event.Effect.Result != "joined" || len(event.Effect.Args) != 3 {
				t.Fatalf("incomplete receipt: %+v", event.Effect)
			}
		}
	}
	if receipts != 1 {
		t.Fatalf("repeated shutdown emitted %d receipts", receipts)
	}
}
