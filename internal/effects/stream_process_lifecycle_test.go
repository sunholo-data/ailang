//go:build !js

package effects

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

// TestAsyncWorkerHelper is a provider-free subprocess fixture. Its ready byte
// gives tests a startup handshake rather than relying on a scheduling sleep.
func TestAsyncWorkerHelper(t *testing.T) {
	if len(os.Args) < 3 || os.Args[len(os.Args)-2] != "--async-worker" {
		return
	}
	switch os.Args[len(os.Args)-1] {
	case "partial":
		fmt.Print("0123456789")
		os.Exit(0)
	case "blocked":
		fmt.Print("R")
		time.Sleep(time.Minute)
	case "descendant":
		time.Sleep(time.Minute)
	case "tree", "leader_exit":
		child := exec.Command(os.Args[0], "-test.run=^TestAsyncWorkerHelper$", "--", "--async-worker", "descendant")
		child.Stdout = os.Stdout
		if err := child.Start(); err != nil {
			os.Exit(2)
		}
		fmt.Printf("%d\n", child.Process.Pid)
		if os.Args[len(os.Args)-1] == "leader_exit" {
			os.Exit(0)
		}
		time.Sleep(time.Minute)
	case "backpressure":
		fmt.Print(strings.Repeat("x", 10000))
		time.Sleep(time.Minute)
	}
	os.Exit(0)
}

func lifecycleProcessSource(t *testing.T, mode string, chunk int) *processSource {
	t.Helper()
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("POSIX worker cancellation contract")
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	source, err := NewProcessSource(context.Background(), exe, []string{"-test.run=^TestAsyncWorkerHelper$", "--", "--async-worker", mode}, "owned", 3, chunk)
	if err != nil {
		t.Fatal(err)
	}
	ps := source.(*processSource)
	t.Cleanup(func() {
		ps.Close()
		bounded, cancel := context.WithTimeout(context.Background(), WorkerShutdownTimeout)
		defer cancel()
		if err := ps.Join(bounded); err != nil {
			t.Errorf("fixture cleanup: %v", err)
		}
	})
	return ps
}

func TestProcessSourceLifecycle_NaturalWaitPreservesPartialOutput(t *testing.T) {
	ps := lifecycleProcessSource(t, "partial", 4)
	var out strings.Builder
	for event := range ps.Events() {
		out.Write(event.data)
	}
	if out.String() != "0123456789" {
		t.Fatalf("output = %q", out.String())
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := ps.Join(ctx); err != nil {
		t.Fatalf("natural exit did not reap and join: %v", err)
	}
	if ps.cmd.ProcessState == nil {
		t.Fatal("event closure without Wait/reaping")
	}
	select {
	case <-ps.Done():
	default:
		t.Fatal("worker completion not published")
	}
}

func TestProcessSourceLifecycle_CancelJoinsBlockedReader(t *testing.T) {
	ps := lifecycleProcessSource(t, "blocked", 1)
	select {
	case event := <-ps.Events():
		if string(event.data) != "R" {
			t.Fatal("missing startup handshake")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("startup timeout")
	}
	if err := ps.RequestStop(); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := ps.Join(ctx); err != nil {
		t.Fatal(err)
	}
	if ps.cmd.ProcessState == nil {
		t.Fatal("cancellation returned without Wait")
	}
	if err := ps.RequestStop(); err != nil {
		t.Fatalf("repeated cancellation: %v", err)
	}
}

func TestProcessSourceLifecycle_CancelUnconsumedOutput(t *testing.T) {
	ps := lifecycleProcessSource(t, "backpressure", 1)
	deadline := time.Now().Add(5 * time.Second)
	for len(ps.ch) < cap(ps.ch) && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if len(ps.ch) != cap(ps.ch) {
		t.Fatal("fixture did not fill delivery queue")
	}
	if err := ps.RequestStop(); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := ps.Join(ctx); err != nil {
		t.Fatalf("full queue prevented cancellation: %v", err)
	}
	if ps.cmd.ProcessState == nil {
		t.Fatal("cancellation returned without Wait")
	}
}

func TestProcessSourceLifecycle_StopsInheritedDescendants(t *testing.T) {
	for _, mode := range []string{"tree", "leader_exit"} {
		t.Run(mode, func(t *testing.T) {
			ps := lifecycleProcessSource(t, mode, 1)
			var line strings.Builder
			deadline := time.NewTimer(5 * time.Second)
			defer deadline.Stop()
		reading:
			for {
				select {
				case event, ok := <-ps.Events():
					if !ok {
						t.Fatal("source ended before descendant PID")
					}
					line.Write(event.data)
					if strings.Contains(line.String(), "\n") {
						break reading
					}
				case <-deadline.C:
					t.Fatal("descendant startup timeout")
				}
			}
			pid, err := strconv.Atoi(strings.TrimSpace(line.String()))
			if err != nil {
				t.Fatal(err)
			}
			if mode == "tree" {
				if err := ps.RequestStop(); err != nil {
					t.Fatal(err)
				}
			}
			// Drain queued stdout after a natural leader exit before joining its reader.
			if mode == "leader_exit" {
				for range ps.Events() {
				}
			}
			bounded, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			if err := ps.Join(bounded); err != nil {
				t.Fatal(err)
			}
			if ps.cmd.ProcessState == nil {
				t.Fatal("direct child was not reaped")
			}
			process, err := os.FindProcess(pid)
			if err != nil {
				t.Fatal(err)
			}
			defer process.Release()
			// A non-child can briefly remain a zombie awaiting its system reaper. That
			// process is dead; the direct child above must already have completed Wait.
			until := time.Now().Add(time.Second)
			for time.Now().Before(until) {
				if err := process.Signal(syscall.Signal(0)); err != nil {
					return
				}
				stat, err := exec.Command("ps", "-o", "stat=", "-p", strconv.Itoa(pid)).Output()
				if err != nil || strings.HasPrefix(strings.TrimSpace(string(stat)), "Z") {
					return
				}
				time.Sleep(time.Millisecond)
			}
			t.Fatalf("owned descendant %d remains live", pid)
		})
	}
}

func TestProcessSourceLifecycle_ConcurrentStopAndJoin(t *testing.T) {
	ps := lifecycleProcessSource(t, "blocked", 1)
	select {
	case <-ps.Events():
	case <-time.After(5 * time.Second):
		t.Fatal("startup timeout")
	}
	var calls sync.WaitGroup
	failures := make(chan error, 12)
	for i := 0; i < 12; i++ {
		calls.Add(1)
		go func() {
			defer calls.Done()
			if err := ps.RequestStop(); err != nil {
				failures <- err
				return
			}
			bounded, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			if err := ps.Join(bounded); err != nil {
				failures <- err
			}
		}()
	}
	calls.Wait()
	close(failures)
	for err := range failures {
		t.Error(err)
	}
}
