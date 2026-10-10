//go:build unix

package effects

import (
	"context"
	"errors"
	"github.com/sunholo-data/ailang/internal/eval"
	"os"
	"path/filepath"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/proctree"
)

// Force the Darwin zombie-group outcome independently of kernel scheduling:
// SIGKILL reaches the owned group, but its result remains raw EPERM until Join.
func TestWorkerPermissionResolvedOnlyAfterJoin(t *testing.T) {
	for _, kind := range []string{"managed", "source"} {
		for _, via := range []string{"explicit", "context"} {
			t.Run(kind+"/"+via, func(t *testing.T) {
				parent, cancel := context.WithCancel(context.Background())
				defer cancel()
				var calls atomic.Int32
				var group atomic.Int64
				kill := func(pid int) error {
					calls.Add(1)
					group.Store(int64(pid))
					if err := proctree.KillGroup(pid); err != nil {
						return err
					}
					return syscall.EPERM
				}
				worker, stopErr, ioDone := permissionWorker(t, parent, kind, kill)
				if via == "context" {
					cancel()
				} else if err := worker.RequestStop(); err != nil {
					t.Fatal(err)
				}
				ctx, c := context.WithTimeout(context.Background(), WorkerShutdownTimeout)
				defer c()
				if err := worker.Join(ctx); err != nil {
					t.Fatalf("reaped group still failed checked Join: %v", err)
				}
				if stopErr() != syscall.EPERM {
					t.Fatalf("raw stop error lost: %v", stopErr())
				}
				select {
				case <-ioDone:
				default:
					t.Fatal("Join returned before owned I/O task finished")
				}
				if calls.Load() != 1 || group.Load() <= 0 {
					t.Fatalf("termination calls=%d group=%d", calls.Load(), group.Load())
				}
			})
		}
	}
}

func TestWorkerPermissionDoesNotCompleteLiveWorker(t *testing.T) {
	for _, kind := range []string{"managed", "source"} {
		t.Run(kind, func(t *testing.T) {
			var pid atomic.Int64
			var calls atomic.Int32
			worker, stopErr, _ := permissionWorker(t, context.Background(), kind, func(p int) error { pid.Store(int64(p)); calls.Add(1); return syscall.EPERM })
			if err := worker.RequestStop(); err != nil {
				t.Fatal(err)
			}
			ctx, c := context.WithTimeout(context.Background(), 20*time.Millisecond)
			err := worker.Join(ctx)
			c()
			if !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("live denied worker reported completion: %v", err)
			}
			if stopErr() != syscall.EPERM || calls.Load() != 1 {
				t.Fatalf("permission error lost or kill retried: %v %d", stopErr(), calls.Load())
			}
			select {
			case <-worker.Done():
				t.Fatal("live worker published completion")
			default:
			}
		})
	}
}

func permissionWorker(t *testing.T, ctx context.Context, kind string, kill func(int) error) (OwnedWorker, func() error, <-chan struct{}) {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	if kind == "managed" {
		path := filepath.Join(t.TempDir(), "ready")
		mp, err := newManagedProcess(ctx, exe, []string{"-test.run=^TestManagedHelperProcess$", "--", "--managed-worker-helper", "block", path}, kill)
		if err != nil {
			t.Fatal(err)
		}
		permissionCleanup(t, mp, mp.cmd.Process.Pid)
		managedAwaitFile(t, path+".ready")
		return mp, func() error { mp.stopMu.Lock(); defer mp.stopMu.Unlock(); return mp.stopErr }, mp.writerDone
	}
	source, err := newProcessSource(ctx, exe, []string{"-test.run=^TestAsyncWorkerHelper$", "--", "--async-worker", "blocked"}, "permission", 0, 1, kill)
	if err != nil {
		t.Fatal(err)
	}
	ps := source.(*processSource)
	permissionCleanup(t, ps, ps.cmd.Process.Pid)
	select {
	case <-ps.Events():
	case <-time.After(3 * time.Second):
		t.Fatal("worker startup timeout")
	}
	return ps, func() error { ps.mu.Lock(); defer ps.mu.Unlock(); return ps.stopErr }, ps.readerDone
}

// Register immediately after construction, before any startup barrier can fail.
// Cleanup targets only a valid group created by this fixture, never PID zero.
func permissionCleanup(t *testing.T, worker OwnedWorker, pid int) {
	t.Helper()
	if pid <= 0 {
		t.Fatal("fixture has no owned process group")
	}
	t.Cleanup(func() {
		_ = worker.RequestStop()
		bounded, c := context.WithTimeout(context.Background(), 20*time.Millisecond)
		err := worker.Join(bounded)
		c()
		if err != nil {
			_ = proctree.KillGroup(pid)
			bounded, c = context.WithTimeout(context.Background(), WorkerShutdownTimeout)
			defer c()
			_ = worker.Join(bounded)
		}
	})
}

// A completed leader with a live descendant must still fail checked public
// cancellation, and natural release must retain a sticky owner failure.
func TestWorkerPermissionRejectsCompletedWorkerWithLiveDescendant(t *testing.T) {
	for _, kind := range []string{"managed", "source"} {
		t.Run(kind, func(t *testing.T) {
			exe, err := os.Executable()
			if err != nil {
				t.Fatal(err)
			}
			ctx := asyncCancellationContext()
			admission, err := ctx.BeginWorker()
			if err != nil {
				t.Fatal(err)
			}
			var pid atomic.Int64
			kill := func(p int) error { pid.Store(int64(p)); return syscall.EPERM }
			var worker OwnedWorker
			var handle eval.Value
			var op, effect string
			if kind == "managed" {
				path := filepath.Join(t.TempDir(), "descendant")
				mp, err := newManagedProcess(admission.Context(), exe, []string{"-test.run=^TestManagedHelperProcess$", "--", "--managed-worker-helper", "tree_exit", path}, kill)
				if err != nil {
					t.Fatal(err)
				}
				permissionCleanup(t, mp, mp.cmd.Process.Pid)
				worker = mp
				handle = makeProcessHandle(ctx.Process.AcquireManagedProcess(mp))
				op = "cancelProcess"
				effect = "Process"
			} else {
				source, err := newProcessSource(admission.Context(), exe, []string{"-test.run=^TestAsyncWorkerHelper$", "--", "--async-worker", "leader_exit"}, "descendant", 0, 1, kill)
				if err != nil {
					t.Fatal(err)
				}
				ps := source.(*processSource)
				permissionCleanup(t, ps, ps.cmd.Process.Pid)
				worker = ps
				handle = makeStreamSource(ctx.Stream.AcquireSource(ps))
				op = "cancelProcessSource"
				effect = "Stream"
				deadline := time.Now().Add(3 * time.Second)
				for pid.Load() == 0 && time.Now().Before(deadline) {
					time.Sleep(time.Millisecond)
				}
				if pid.Load() == 0 {
					t.Fatal("leader did not finish")
				}
				if err := worker.RequestStop(); err != nil {
					t.Fatal(err)
				}
			}
			if err := admission.Complete(worker); err != nil {
				t.Fatal(err)
			}
			bounded, c := context.WithTimeout(context.Background(), WorkerShutdownTimeout)
			defer c()
			if err := worker.Join(bounded); !errors.Is(err, syscall.EPERM) {
				t.Fatalf("live group error discarded after completion: %v", err)
			}
			result, err := Call(ctx, effect, op, []eval.Value{handle})
			if err != nil {
				t.Fatal(err)
			}
			if got := workerErrorConstructor(t, result); got != "WorkerCancelFailed" {
				t.Fatalf("public cancellation = %s", got)
			}
			ctx.ReleaseWorker(worker)
			if err := ctx.CloseWorkers(); !errors.Is(err, syscall.EPERM) {
				t.Fatalf("natural release lost sticky permission failure: %v", err)
			}
		})
	}
}
