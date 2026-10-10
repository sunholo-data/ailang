//go:build !js

package effects

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/eval"
)

// The test executable supplies deterministic barriers, without a paid provider.
func TestManagedHelperProcess(t *testing.T) {
	marker := -1
	for i, arg := range os.Args {
		if arg == "--managed-worker-helper" {
			marker = i
			break
		}
	}
	if marker < 0 {
		return
	}
	mode, path := os.Args[marker+1], os.Args[marker+2]
	if mode == "stdin" {
		data, err := io.ReadAll(os.Stdin)
		if err == nil {
			err = os.WriteFile(path, data, 0600)
		}
		if err != nil {
			os.Exit(3)
		}
		os.Exit(0)
	}
	if mode == "tree" || mode == "tree_exit" {
		executable, err := os.Executable()
		if err != nil {
			os.Exit(3)
		}
		child := exec.Command(executable, "-test.run=^TestManagedHelperProcess$", "--", "--managed-worker-helper", "block", path+".child")
		if err := child.Start(); err != nil {
			os.Exit(3)
		}
		if err := os.WriteFile(path, []byte(strconv.Itoa(child.Process.Pid)), 0600); err != nil {
			os.Exit(3)
		}
		if mode == "tree_exit" {
			deadline := time.Now().Add(3 * time.Second)
			for {
				if _, err := os.Stat(path + ".child.ready"); err == nil {
					os.Exit(0)
				}
				if time.Now().After(deadline) {
					os.Exit(3)
				}
				time.Sleep(time.Millisecond)
			}
		}
	}
	if err := os.WriteFile(path+".ready", []byte("started"), 0600); err != nil {
		os.Exit(3)
	}
	time.Sleep(time.Minute)
	os.Exit(0)
}

func managedAwaitFile(t *testing.T, path string) []byte {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		data, err := os.ReadFile(path)
		if err == nil {
			return data
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("worker did not acknowledge startup: %s", path)
	return nil
}

func TestManagedLifecycle_CooperativeDrainAndWriterJoin(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(t.TempDir(), "stdin")
	mp, err := NewManagedProcess(t.Context(), exe, []string{"-test.run=^TestManagedHelperProcess$", "--", "--managed-worker-helper", "stdin", output})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(mp.Close)
	expected := strings.Repeat("queued\n", 100)
	for range 100 {
		if err := mp.Write([]byte("queued\n")); err != nil {
			t.Fatal(err)
		}
	}
	mp.CloseStdin()
	bounded, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()
	if err := mp.Join(bounded); err != nil {
		t.Fatal(err)
	}
	select {
	case <-mp.writerDone:
	default:
		t.Fatal("Join returned before writer task joined")
	}
	if got := string(managedAwaitFile(t, output)); got != expected {
		t.Fatalf("accepted queued writes lost: got %d bytes, want %d", len(got), len(expected))
	}
}

func TestManagedLifecycle_ForeignHandleDoesNotStopBystander(t *testing.T) {
	if !WorkerCancellationSupported() {
		t.Skip("POSIX cancellation guarantee")
	}
	_, foreign, _ := managedSleepFixture(t)
	other, own, mp := managedSleepFixture(t)
	result, err := Call(other, "Process", "cancelProcess", []eval.Value{foreign})
	if err != nil {
		t.Fatal(err)
	}
	if result.(*eval.TaggedValue).Fields[0].(*eval.TaggedValue).CtorName != "WorkerHandleInvalid" {
		t.Fatal(result)
	}
	select {
	case <-mp.Done():
		t.Fatal("foreign cancellation stopped bystander")
	default:
	}
	result, err = Call(other, "Process", "cancelProcess", []eval.Value{own})
	if err != nil || result.(*eval.TaggedValue).CtorName != "Ok" {
		t.Fatalf("own cancellation: %v %v", result, err)
	}
}

func TestManagedLifecycle_ConcurrentCancellationAndBlockedWriter(t *testing.T) {
	if !WorkerCancellationSupported() {
		t.Skip("POSIX cancellation guarantee")
	}
	ctx, handle, mp := managedSleepFixture(t)
	// sleep ignores stdin, leaving the runtime writer blocked on a full OS pipe.
	_ = mp.Write(make([]byte, 4<<20))
	var group sync.WaitGroup
	for range 10 {
		group.Go(func() {
			result, err := Call(ctx, "Process", "cancelProcess", []eval.Value{handle})
			if err != nil {
				t.Error(err)
				return
			}
			if result.(*eval.TaggedValue).CtorName == "Err" && result.(*eval.TaggedValue).Fields[0].(*eval.TaggedValue).CtorName != "WorkerHandleInvalid" {
				t.Errorf("unexpected cancellation failure: %v", result)
			}
		})
	}
	group.Wait()
	select {
	case <-mp.writerDone:
	default:
		t.Fatal("cancellation left writer blocked")
	}
}

func TestManagedLifecycle_ProviderFreeBlockingWorkerAndDescendant(t *testing.T) {
	if !WorkerCancellationSupported() {
		t.Skip("POSIX cancellation guarantee")
	}
	ps, err := exec.LookPath("ps")
	if err != nil {
		t.Skip("ps unavailable for exact PID evidence")
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	pidFile := filepath.Join(t.TempDir(), "tree")
	mp, err := NewManagedProcess(t.Context(), exe, []string{"-test.run=^TestManagedHelperProcess$", "--", "--managed-worker-helper", "tree", pidFile})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(mp.Close)
	descendantPID := strings.TrimSpace(string(managedAwaitFile(t, pidFile)))
	managedAwaitFile(t, pidFile+".child.ready")
	if err := mp.RequestStop(); err != nil {
		t.Fatal(err)
	}
	bounded, cancel := context.WithTimeout(t.Context(), WorkerShutdownTimeout)
	defer cancel()
	if err := mp.Join(bounded); err != nil {
		t.Fatal(err)
	}
	if mp.cmd.ProcessState == nil {
		t.Fatal("direct child was not reaped")
	}
	deadline := time.Now().Add(time.Second)
	for {
		state, err := exec.Command(ps, "-o", "stat=", "-p", descendantPID).Output()
		if err != nil {
			exit, ok := err.(*exec.ExitError)
			if !ok || exit.ExitCode() != 1 {
				t.Fatalf("PID probe failed: %v", err)
			}
		}
		if len(strings.TrimSpace(string(state))) == 0 || strings.Contains(string(state), "Z") {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("owned descendant PID %s remains executing: %s on %s", descendantPID, state, runtime.GOOS)
		}
		time.Sleep(10 * time.Millisecond)
	}
}
