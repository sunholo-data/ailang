//go:build !js

package effects

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/eval"
)

func asyncCancellationContext() *EffContext {
	ctx := NewEffContext(nil)
	ctx.Grant(NewCapability("Stream"))
	ctx.Grant(NewCapability("Process"))
	ctx.Stream = NewStreamContext()
	ctx.Process = NewProcessContext()
	return ctx
}

func workerErrorConstructor(t *testing.T, result eval.Value) string {
	t.Helper()
	outer, ok := result.(*eval.TaggedValue)
	if !ok || outer.CtorName != "Err" || len(outer.Fields) != 1 {
		t.Fatalf("expected typed Err, got %v", result)
	}
	return outer.Fields[0].(*eval.TaggedValue).CtorName
}

func TestStreamCancelProcessSource_AuthorityAndBudget(t *testing.T) {
	for _, cap := range []string{"Stream", "Process"} {
		t.Run("missing_"+cap, func(t *testing.T) {
			ctx := asyncCancellationContext()
			delete(ctx.Caps, cap)
			_, err := Call(ctx, "Stream", "cancelProcessSource", []eval.Value{makeStreamSource(987654321)})
			if err == nil {
				t.Fatalf("missing %s accepted", cap)
			}
		})
	}
	for _, effect := range []string{"Stream", "Process"} {
		t.Run("budget_"+effect, func(t *testing.T) {
			ctx := asyncCancellationContext()
			limit := 0
			ctx.SetBudget(NewBudgetContext(map[string]*int{effect: &limit}))
			// The evaluator has already charged the primary Stream effect and opens a
			// duplicate-charge suppression scope around the builtin implementation.
			if effect == "Process" {
				ctx.BeginBudgetChargeScope()
				defer ctx.EndBudgetChargeScope()
			}
			_, err := Call(ctx, "Stream", "cancelProcessSource", []eval.Value{makeStreamSource(987654321)})
			if err == nil {
				t.Fatalf("%s budget exhaustion ignored", effect)
			}
		})
	}
	t.Run("one_secondary_charge", func(t *testing.T) {
		ctx := asyncCancellationContext()
		limit := 1
		ctx.SetBudget(NewBudgetContext(map[string]*int{"Process": &limit}))
		ctx.BeginBudgetChargeScope()
		defer ctx.EndBudgetChargeScope()
		_, err := Call(ctx, "Stream", "cancelProcessSource", []eval.Value{makeStreamSource(987654321)})
		if err != nil {
			t.Fatal(err)
		}
		if used := ctx.Budget.Used("Process"); used != 1 {
			t.Fatalf("Process charged %d times, expected once", used)
		}
	})
}

func TestStreamCancelProcessSource_RejectsBorrowedAndForeignSources(t *testing.T) {
	if !WorkerCancellationSupported() {
		t.Skip("POSIX cancellation")
	}
	ctx := asyncCancellationContext()
	borrowed := NewStdinSource(strings.NewReader("keep\n"), "borrowed", 0)
	id := ctx.Stream.AcquireSource(borrowed)
	result, err := Call(ctx, "Stream", "cancelProcessSource", []eval.Value{makeStreamSource(id)})
	if err != nil {
		t.Fatal(err)
	}
	if got := workerErrorConstructor(t, result); got != "WorkerCancelUnsupported" {
		t.Fatal(got)
	}
	select {
	case event := <-borrowed.Events():
		if event.text != "keep" {
			t.Fatal("borrowed reader changed")
		}
	case <-time.After(time.Second):
		t.Fatal("borrowed source lost")
	}
	other := asyncCancellationContext()
	otherID := other.Stream.AcquireSource(NewStdinSource(strings.NewReader("other\n"), "other", 0))
	if otherID == id {
		t.Fatal("cross-owner IDs collide")
	}
	for _, invalid := range []eval.Value{makeStreamSource(otherID), makeStreamSource(987654321), &eval.StringValue{Value: "not a source"}} {
		result, err := Call(ctx, "Stream", "cancelProcessSource", []eval.Value{invalid})
		if err != nil {
			t.Fatal(err)
		}
		if got := workerErrorConstructor(t, result); got != "WorkerHandleInvalid" {
			t.Fatal(got)
		}
	}
}

func TestStreamCancelProcessSource_StopsAndReleasesOwnedWorker(t *testing.T) {
	ps := lifecycleProcessSource(t, "blocked", 1)
	ctx := asyncCancellationContext()
	admission, err := ctx.BeginWorker()
	if err != nil {
		t.Fatal(err)
	}
	if err := admission.Complete(ps); err != nil {
		t.Fatal(err)
	}
	id := ctx.Stream.AcquireSource(ps)
	select {
	case <-ps.Events():
	case <-time.After(5 * time.Second):
		t.Fatal("startup timeout")
	}
	result, err := Call(ctx, "Stream", "cancelProcessSource", []eval.Value{makeStreamSource(id)})
	if err != nil {
		t.Fatal(err)
	}
	if result.(*eval.TaggedValue).CtorName != "Ok" {
		t.Fatal(result)
	}
	if _, found := ctx.Stream.GetSource(id); found {
		t.Fatal("cancelled source still registered")
	}
	if ps.cmd.ProcessState == nil {
		t.Fatal("success before reap")
	}
	result, err = Call(ctx, "Stream", "cancelProcessSource", []eval.Value{makeStreamSource(id)})
	if err != nil {
		t.Fatal(err)
	}
	if workerErrorConstructor(t, result) != "WorkerHandleInvalid" {
		t.Fatal(result)
	}
	if err := ctx.CloseWorkers(); err != nil {
		t.Fatal(err)
	}
	bounded, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := ps.Join(bounded); err != nil {
		t.Fatal(err)
	}
}

func TestStreamAsyncExecProcess_CompletedOutputRemainsSelectable(t *testing.T) {
	if !WorkerCancellationSupported() {
		t.Skip("POSIX helper fixture")
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ctx := asyncCancellationContext()
	defer ctx.CloseWorkers()
	handle, err := StreamAsyncExecProcess(ctx, []eval.Value{
		&eval.StringValue{Value: exe},
		&eval.ListValue{Elements: []eval.Value{&eval.StringValue{Value: "-test.run=^TestAsyncWorkerHelper$"}, &eval.StringValue{Value: "--"}, &eval.StringValue{Value: "--async-worker"}, &eval.StringValue{Value: "partial"}}},
		&eval.StringValue{Value: "fast"}, &eval.IntValue{Value: 1}, &eval.IntValue{Value: 4},
	})
	if err != nil {
		t.Fatal(err)
	}
	id, err := extractSourceID(handle)
	if err != nil {
		t.Fatal(err)
	}
	source, ok := ctx.Stream.GetSource(id)
	if !ok {
		t.Fatal("fast source released before consumption")
	}
	worker := source.(*processSource)
	bounded, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := worker.Join(bounded); err != nil {
		t.Fatal(err)
	}
	// Wait complete, output still buffered: future selectors must resolve it.
	source, ok = ctx.Stream.GetSource(id)
	if !ok {
		t.Fatal("completed child's queued output became unreachable")
	}
	var output strings.Builder
	for event := range source.Events() {
		output.Write(event.data)
	}
	if output.String() != "0123456789" {
		t.Fatalf("tail lost: %q", output.String())
	}
	result, err := Call(ctx, "Stream", "cancelProcessSource", []eval.Value{handle})
	if err != nil || result.(*eval.TaggedValue).CtorName != "Ok" {
		t.Fatalf("completed cancellation: %v, %v", result, err)
	}
}

func TestStreamCancelProcessSource_IsolatesWorkersAndOwners(t *testing.T) {
	first := lifecycleProcessSource(t, "blocked", 1)
	second := lifecycleProcessSource(t, "blocked", 1)
	bystander := lifecycleProcessSource(t, "blocked", 1)
	contexts := []*EffContext{asyncCancellationContext(), asyncCancellationContext()}
	ids := make([]int, 2)
	for i, worker := range []*processSource{first, second} {
		admission, err := contexts[i].BeginWorker()
		if err != nil {
			t.Fatal(err)
		}
		if err := admission.Complete(worker); err != nil {
			t.Fatal(err)
		}
		ids[i] = contexts[i].Stream.AcquireSource(worker)
	}
	for _, worker := range []*processSource{first, second, bystander} {
		select {
		case <-worker.Events():
		case <-time.After(5 * time.Second):
			t.Fatal("startup timeout")
		}
	}
	result, err := Call(contexts[0], "Stream", "cancelProcessSource", []eval.Value{makeStreamSource(ids[1])})
	if err != nil {
		t.Fatal(err)
	}
	if workerErrorConstructor(t, result) != "WorkerHandleInvalid" {
		t.Fatal(result)
	}
	result, err = Call(contexts[0], "Stream", "cancelProcessSource", []eval.Value{makeStreamSource(ids[0])})
	if err != nil || result.(*eval.TaggedValue).CtorName != "Ok" {
		t.Fatalf("own cancellation: %v, %v", result, err)
	}
	for _, worker := range []*processSource{second, bystander} {
		select {
		case <-worker.Done():
			t.Fatal("cancelled unrelated worker")
		default:
		}
	}
	for _, ctx := range contexts {
		if err := ctx.CloseWorkers(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestStreamCancelProcessSource_UnsupportedNativePlatform(t *testing.T) {
	if WorkerCancellationSupported() {
		t.Skip("non-POSIX platform assertion")
	}
	ctx := asyncCancellationContext()
	result, err := Call(ctx, "Stream", "cancelProcessSource", []eval.Value{makeStreamSource(1)})
	if err != nil {
		t.Fatal(err)
	}
	if workerErrorConstructor(t, result) != "WorkerCancelUnsupported" {
		t.Fatal(result)
	}
}
