//go:build !js

package effects

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/eval"
)

// This handler enters the real AI effect dispatch, acknowledges that boundary,
// then blocks without making a provider request. Cancellation kills its OS owner.
type workerBlockingAI struct {
	AIHandler
	ready string
}

func (h *workerBlockingAI) Call(string) (string, error) {
	// Publish only a complete PID: another process may observe file creation
	// before WriteFile has written the contents, especially on Linux CI.
	pending := h.ready + ".pending"
	if err := os.WriteFile(pending, []byte(strconv.Itoa(os.Getpid())), 0600); err != nil {
		return "", err
	}
	if err := os.Rename(pending, h.ready); err != nil {
		return "", err
	}
	time.Sleep(time.Minute)
	return "completed", nil
}

func TestWorkerBlockedAIChild(t *testing.T) {
	for i, arg := range os.Args {
		if arg == "--worker-blocked-ai" {
			ctx := NewEffContext(nil)
			ctx.Grant(NewCapability("AI"))
			ctx.AI = NewAIContext(&workerBlockingAI{AIHandler: NewStubAIHandler(), ready: os.Args[i+1]})
			if _, err := Call(ctx, "AI", "call", []eval.Value{&eval.StringValue{Value: "no-provider"}}); err != nil {
				os.Exit(3)
			}
			os.Exit(0)
		}
	}
}

func spawnBlockedAI(t *testing.T, ctx *EffContext, async bool, n int) (eval.Value, OwnedWorker, int) {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ready := filepath.Join(t.TempDir(), fmt.Sprintf("ready-%d", n))
	argv := &eval.ListValue{Elements: []eval.Value{
		&eval.StringValue{Value: "-test.run=^TestWorkerBlockedAIChild$"},
		&eval.StringValue{Value: "--"}, &eval.StringValue{Value: "--worker-blocked-ai"}, &eval.StringValue{Value: ready},
	}}
	var h eval.Value
	var w OwnedWorker
	if async {
		h, err = Call(ctx, "Stream", "asyncExecProcess", []eval.Value{&eval.StringValue{Value: executable}, argv, &eval.StringValue{Value: "AI"}, &eval.IntValue{Value: 1}, &eval.IntValue{Value: 1}})
		if err == nil {
			id, _ := extractSourceID(h)
			source, _ := ctx.Stream.GetSource(id)
			w = source.(*processSource)
		}
	} else {
		h, err = Call(ctx, "Process", "spawnProcess", []eval.Value{&eval.StringValue{Value: executable}, argv})
		if err == nil {
			id, _ := extractProcessHandleID(h)
			w, _ = ctx.Process.GetManagedProcess(id)
		}
	}
	if err != nil {
		t.Fatal(err)
	}
	pidText := managedAwaitFile(t, ready)
	pid, err := strconv.Atoi(string(pidText))
	if err != nil {
		t.Fatal(err)
	}
	return h, w, pid
}

func workerAIContext(t *testing.T) *EffContext {
	t.Helper()
	ctx := NewEffContext(nil)
	ctx.Grant(NewCapability("Process"))
	ctx.Grant(NewCapability("Stream"))
	ctx.Process = NewProcessContext()
	ctx.Stream = NewStreamContext()
	t.Cleanup(func() {
		if err := ctx.CloseWorkers(); err != nil {
			t.Errorf("cleanup: %v", err)
		}
	})
	return ctx
}

func TestWorkerCancellationDuringBlockedAI(t *testing.T) {
	if !WorkerCancellationSupported() {
		t.Skip("POSIX group contract")
	}
	for _, async := range []bool{false, true} {
		t.Run(fmt.Sprint("async=", async), func(t *testing.T) {
			ctx := workerAIContext(t)
			h, w, _ := spawnBlockedAI(t, ctx, async, 0)
			_, other, _ := spawnBlockedAI(t, ctx, async, 1)
			effect, op := "Process", "cancelProcess"
			if async {
				effect, op = "Stream", "cancelProcessSource"
			}
			result, err := Call(ctx, effect, op, []eval.Value{h})
			if err != nil || result.(*eval.TaggedValue).CtorName != "Ok" {
				t.Fatalf("cancel %v %v", result, err)
			}
			select {
			case <-w.Done():
			default:
				t.Fatal("cancel returned before worker join")
			}
			select {
			case <-other.Done():
				t.Fatal("cancellation stopped second worker")
			default:
			}
			if err := w.Join(context.Background()); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestWorkerTwentyBlockedAIOneShutdownDeadline(t *testing.T) {
	if !WorkerCancellationSupported() {
		t.Skip("POSIX group contract")
	}
	ctx := workerAIContext(t)
	workers := make([]OwnedWorker, 20)
	for i := range workers {
		_, workers[i], _ = spawnBlockedAI(t, ctx, i%2 == 0, i)
	}
	start := time.Now()
	if err := ctx.CloseWorkers(); err != nil {
		t.Fatal(err)
	}
	if elapsed := time.Since(start); elapsed > WorkerShutdownTimeout+500*time.Millisecond {
		t.Fatalf("shutdown exceeded total deadline: %s", elapsed)
	}
	for _, w := range workers {
		select {
		case <-w.Done():
		default:
			t.Fatal("owner returned before tasks joined")
		}
		switch worker := w.(type) {
		case *managedProcess:
			if worker.cmd.ProcessState == nil {
				t.Fatal("managed direct child not reaped")
			}
		case *processSource:
			if worker.cmd.ProcessState == nil {
				t.Fatal("async direct child not reaped")
			}
		}
	}
}

func TestWorkerRepeatedRunsDoNotAccumulateOwnedResources(t *testing.T) {
	if !WorkerCancellationSupported() {
		t.Skip("POSIX group contract")
	}
	fdDir := "/dev/fd"
	if runtime.GOOS == "linux" {
		fdDir = "/proc/self/fd"
	}
	before, err := workerOpenDescriptors(fdDir)
	if err != nil {
		t.Fatal(err)
	}
	initialGoroutines := runtime.NumGoroutine()
	for run := 0; run < 10; run++ {
		ctx := workerAIContext(t)
		_, managed, _ := spawnBlockedAI(t, ctx, false, run)
		_, async, _ := spawnBlockedAI(t, ctx, true, run)
		if err := ctx.CloseWorkers(); err != nil {
			t.Fatal(err)
		}
		for _, worker := range []OwnedWorker{managed, async} {
			select {
			case <-worker.Done():
			default:
				t.Fatal("joined receipt left runtime tasks")
			}
		}
	}
	deadline := time.Now().Add(time.Second)
	for {
		after, err := workerOpenDescriptors(fdDir)
		if err != nil {
			t.Fatal(err)
		}
		goroutines := runtime.NumGoroutine()
		if len(after) <= len(before)+2 && goroutines <= initialGoroutines+2 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("resources grew: fds %d -> %d; goroutines %d -> %d", len(before), len(after), initialGoroutines, goroutines)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// Darwin's /dev/fd entries disappear as enumeration proceeds; reading names avoids
// statting ephemeral descriptors and counts the enumeration descriptor equally.
func workerOpenDescriptors(dir string) ([]string, error) {
	file, err := os.Open(dir)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return file.Readdirnames(-1)
}
