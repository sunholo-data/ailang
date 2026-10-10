//go:build !js

package effects

import (
	"os/exec"
	"sync"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/eval"
)

func managedSleepFixture(t *testing.T) (*EffContext, eval.Value, *managedProcess) {
	t.Helper()
	sleep, err := exec.LookPath("sleep")
	if err != nil {
		t.Skip("sleep unavailable")
	}
	ctx := NewEffContext(nil)
	ctx.Grant(NewCapability("Process"))
	ctx.Process = NewProcessContext()
	handle, err := ProcessSpawn(ctx, []eval.Value{&eval.StringValue{Value: sleep}, &eval.ListValue{Elements: []eval.Value{&eval.StringValue{Value: "60"}}}})
	if err != nil {
		t.Fatal(err)
	}
	id, _ := extractProcessHandleID(handle)
	mp, ok := ctx.Process.GetManagedProcess(id)
	if !ok {
		t.Fatal("spawn was not registered")
	}
	t.Cleanup(mp.Close)
	return ctx, handle, mp
}

func TestManagedOwnership_CloseStdinRetainsWorker(t *testing.T) {
	ctx, handle, mp := managedSleepFixture(t)
	if _, err := ProcessCloseStdin(ctx, []eval.Value{handle}); err != nil {
		t.Fatal(err)
	}
	id, _ := extractProcessHandleID(handle)
	got, ok := ctx.Process.GetManagedProcess(id)
	if !ok || got != mp {
		t.Fatal("closing stdin released a still-running owned worker")
	}
}

func TestManagedOwnership_HandlesDoNotCollideAcrossOwners(t *testing.T) {
	first, handle, _ := managedSleepFixture(t)
	second, otherHandle, _ := managedSleepFixture(t)
	firstID, _ := extractProcessHandleID(handle)
	secondID, _ := extractProcessHandleID(otherHandle)
	if firstID == secondID {
		t.Fatal("independent owners reused an opaque handle ID")
	}
	if _, ok := second.Process.GetManagedProcess(firstID); ok {
		t.Fatal("foreign handle resolved in another owner")
	}
	if _, ok := first.Process.GetManagedProcess(secondID); ok {
		t.Fatal("reverse foreign handle resolved")
	}
}

func TestManagedCancellation_StopsAndReaps(t *testing.T) {
	ctx, handle, mp := managedSleepFixture(t)
	result, err := Call(ctx, "Process", "cancelProcess", []eval.Value{handle})
	if err != nil {
		t.Fatal(err)
	}
	tagged := result.(*eval.TaggedValue)
	if !WorkerCancellationSupported() {
		if tagged.CtorName != "Err" || tagged.Fields[0].(*eval.TaggedValue).CtorName != "WorkerCancelUnsupported" {
			t.Fatalf("unexpected unsupported-platform result: %v", result)
		}
		return
	}
	if tagged.CtorName != "Ok" {
		t.Fatalf("cancellation failed: %v", result)
	}
	select {
	case <-mp.done:
	default:
		t.Fatal("success returned before Wait completed")
	}
	result, err = Call(ctx, "Process", "cancelProcess", []eval.Value{handle})
	if err != nil {
		t.Fatal(err)
	}
	if result.(*eval.TaggedValue).CtorName != "Err" {
		t.Fatal("released handle was accepted")
	}
}

func TestManagedCancellation_InvalidAndMissingAuthority(t *testing.T) {
	ctx := NewEffContext(nil)
	ctx.Grant(NewCapability("Process"))
	ctx.Process = NewProcessContext()
	result, err := Call(ctx, "Process", "cancelProcess", []eval.Value{makeProcessHandle(-1)})
	if err != nil {
		t.Fatal(err)
	}
	inner := result.(*eval.TaggedValue).Fields[0].(*eval.TaggedValue)
	if !WorkerCancellationSupported() {
		if inner.CtorName != "WorkerCancelUnsupported" {
			t.Fatal(result)
		}
	} else if inner.CtorName != "WorkerHandleInvalid" {
		t.Fatal(result)
	}
	denied := NewEffContext(nil)
	if _, err := Call(denied, "Process", "cancelProcess", []eval.Value{makeProcessHandle(-1)}); err == nil {
		t.Fatal("missing Process authority accepted")
	}
}

func TestManagedCancellation_ConcurrentWriteClose(t *testing.T) {
	cat, err := exec.LookPath("cat")
	if err != nil {
		t.Skip("cat unavailable")
	}
	mp, err := NewManagedProcess(t.Context(), cat, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer mp.Close()
	var group sync.WaitGroup
	for range 20 {
		group.Go(func() {
			for range 50 {
				_ = mp.Write([]byte("hello\n"))
			}
		})
	}
	group.Go(mp.CloseStdin)
	group.Wait()
	select {
	case <-mp.done:
	case <-time.After(3 * time.Second):
		t.Fatal("stdin close failed to finish")
	}
}

func TestManagedCancellation_ExhaustedBudgetDoesNotPreventHostCleanup(t *testing.T) {
	ctx, handle, mp := managedSleepFixture(t)
	limit := 0
	ctx.SetBudget(NewBudgetContext(map[string]*int{"Process": &limit}))
	if _, err := Call(ctx, "Process", "cancelProcess", []eval.Value{handle}); err == nil {
		t.Fatal("cancellation bypassed exhausted Process budget")
	}
	select {
	case <-mp.Done():
		t.Fatal("denied cancellation stopped worker")
	default:
	}
	if err := ctx.CloseWorkers(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-mp.Done():
	default:
		t.Fatal("mandatory cleanup was blocked by user budget")
	}
	if ctx.Budget.Used("Process") != 0 {
		t.Fatal("mandatory cleanup consumed user budget")
	}
}
