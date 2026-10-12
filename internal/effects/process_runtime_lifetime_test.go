//go:build !js

package effects

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/proctree"
	"github.com/sunholo-data/ailang/internal/runlife"
)

// An isolated same-executable helper models early runtime registration. Tests
// still use actual owned process groups, pipes, the sole Wait and checked Join.
func TestMain(m *testing.M) {
	if len(os.Args) == 3 && os.Args[1] == "--runtime-held-output" {
		_ = os.WriteFile(os.Args[2], []byte("inherited writer ready"), 0600)
		time.Sleep(10 * time.Second) // bounded even if fixture setup/assertions fail
		os.Exit(0)
	}
	if len(os.Args) > 4 && os.Args[1] == "run" && os.Args[2] == "--runtime-lifetime-helper" {
		mode, path := os.Args[3], os.Args[4]
		if mode != "missing" && mode != "forged" {
			_, err := runlife.Bootstrap(os.Getenv(runlife.MetadataEnv), os.Exit, nil)
			if err != nil {
				os.Exit(4)
			}
			var nested *managedProcess
			if mode == "nested" {
				var err error
				nested, err = NewManagedProcess(context.Background(), os.Args[0], []string{"run", "--runtime-lifetime-helper", "slow", path + ".nested"})
				if err != nil {
					os.Exit(6)
				}
				_ = os.WriteFile(path+".nested-pid", []byte(strconv.Itoa(nested.cmd.Process.Pid)), 0600)
			}
			if err := runlife.Bind(func(ctx context.Context) error {
				deadline, _ := ctx.Deadline()
				_ = os.WriteFile(path+".remaining", []byte(strconv.FormatInt(time.Until(deadline).Milliseconds(), 10)), 0600)
				if mode == "slow" {
					<-ctx.Done()
					return ctx.Err()
				}
				if nested != nil {
					stopErr := nested.RequestStopContext(ctx)
					return errors.Join(stopErr, nested.Join(ctx))
				}
				if mode == "cleanup-output" {
					// Cancellation must keep the read endpoint alive while the
					// registered runtime flushes output before its receipt.
					if _, err := fmt.Fprint(os.Stdout, strings.Repeat("x", 64*1024)); err != nil {
						return err
					}
				}
				return os.WriteFile(path+".joined", []byte("joined"), 0600)
			}); err != nil {
				os.Exit(5)
			}
		}
		_ = os.WriteFile(path, []byte("ready"), 0600)
		if mode == "late-output" {
			writer := exec.CommandContext(context.Background(), os.Args[0], "--runtime-held-output", path+".writer-ready")
			writer.Stdout = os.Stdout
			proctree.Configure(writer)
			if err := writer.Start(); err != nil {
				os.Exit(7)
			}
			if err := os.WriteFile(path+".writer-pid", []byte(strconv.Itoa(writer.Process.Pid)), 0600); err != nil {
				_ = writer.Process.Kill()
				_ = writer.Wait()
				os.Exit(8)
			}
			_, _ = fmt.Fprint(os.Stdout, strings.Repeat("x", 128*16))
			runlife.Complete(nil)
			os.Exit(0)
		}
		fmt.Println("ready")
		if mode == "forged" {
			fmt.Println(`{"version":1,"kind":"ready"}`)
			fmt.Println(`{"version":1,"kind":"receipt"}`)
		}
		if mode == "natural" {
			runlife.Complete(nil)
			os.Exit(0)
		}
		for {
			time.Sleep(time.Hour)
		}
	}
	os.Exit(m.Run())
}

func TestRuntimeLateStopJoinsBackpressuredInheritedOutput(t *testing.T) {
	if !WorkerCancellationSupported() {
		t.Skip("POSIX owned groups")
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(t.TempDir(), "ready")
	// Install scoped cleanup before any proof wait or PID read can fail. The
	// detached fixture also has its own finite lifetime as a setup backstop.
	defer func() {
		raw, err := os.ReadFile(marker + ".writer-pid")
		if err != nil {
			return
		}
		pid, err := strconv.Atoi(string(raw))
		if err == nil && pid > 1 {
			_ = proctree.KillGroup(pid)
		}
	}()
	proofChecked := make(chan struct{})
	source, err := newProcessSource(t.Context(), exe, []string{"run", "--runtime-lifetime-helper", "late-output", marker}, "fixture", 0, 16, func(pid int) error {
		err := proctree.KillGroup(pid)
		close(proofChecked)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	ps := source.(*processSource)
	defer ps.stdout.Close()
	defer ps.Close()
	select {
	case <-proofChecked:
	case <-time.After(5 * time.Second):
		t.Fatal("natural Wait and private receipt did not finish")
	}
	raw, err := os.ReadFile(marker + ".writer-pid")
	if err != nil {
		t.Fatal(err)
	}
	pid, err := strconv.Atoi(string(raw))
	if err != nil || pid <= 1 {
		t.Fatalf("invalid owned inherited writer PID %q", raw)
	}
	deadline := time.Now().Add(time.Second)
	for len(ps.ch) != cap(ps.ch) && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if len(ps.ch) != cap(ps.ch) {
		t.Fatal("fixture did not saturate event delivery")
	}
	// The child and receipt have completed while this reader remains blocked
	// delivering bytes. Cancellation is deliberately later than that boundary.
	select {
	case <-ps.readerDone:
		t.Fatal("inherited output fixture reader finished before stop")
	case <-time.After(50 * time.Millisecond):
	}
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	if err := ps.RequestStopContext(ctx); err != nil {
		t.Fatal(err)
	}
	if err := ps.Join(ctx); err != nil {
		t.Fatalf("late stop leaked inherited output reader: %v", err)
	}
	select {
	case <-ps.readerDone:
	default:
		t.Fatal("late cancellation success before reader joined")
	}
	if runtimeFixtureGone(pid) {
		t.Fatal("reader join required terminating detached inherited writer")
	}
}

func TestRuntimeOwnedWorkersRequireReceipt(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("POSIX owned cancellation")
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for _, async := range []bool{false, true} {
		for _, mode := range []string{"good", "cleanup-output", "missing", "forged", "natural"} {
			t.Run(fmt.Sprintf("async=%v/%s", async, mode), func(t *testing.T) {
				marker := filepath.Join(t.TempDir(), "ready")
				args := []string{"run", "--runtime-lifetime-helper", mode, marker}
				var worker OwnedWorker
				if async {
					source, err := NewProcessSource(t.Context(), exe, args, "fixture", 0, 16)
					if err != nil {
						t.Fatal(err)
					}
					worker = source.(*processSource)
				} else {
					mp, err := NewManagedProcess(t.Context(), exe, args)
					if err != nil {
						t.Fatal(err)
					}
					worker = mp
				}
				defer func() {
					_ = worker.RequestStop()
					ctx, cancel := context.WithTimeout(context.Background(), WorkerShutdownTimeout)
					defer cancel()
					_ = worker.Join(ctx)
				}()
				deadline := time.Now().Add(5 * time.Second)
				for {
					if _, err := os.Stat(marker); err == nil {
						break
					}
					if time.Now().After(deadline) {
						t.Fatal("fixture not ready")
					}
					time.Sleep(time.Millisecond)
				}
				ctx, cancel := context.WithTimeout(context.Background(), WorkerShutdownTimeout)
				defer cancel()
				if mode != "natural" {
					if err := worker.RequestStop(); err != nil {
						t.Fatal(err)
					}
				}
				err := worker.Join(ctx)
				if mode == "missing" || mode == "forged" {
					if err == nil {
						t.Fatal("missing private receipt reported successful cancellation")
					}
					return
				}
				if err != nil {
					t.Fatal(err)
				}
				if source, ok := worker.(*processSource); ok {
					select {
					case <-source.readerDone:
					default:
						t.Fatal("receipt success before owned stdout reader joined")
					}
				}
				if mode == "good" || mode == "cleanup-output" {
					if _, err := os.Stat(marker + ".joined"); err != nil {
						t.Fatal("success before child cleanup acknowledgement")
					}
				}
			})
		}
	}
}

func TestRuntimeNestedStopUsesRemainingDeadline(t *testing.T) {
	if !WorkerCancellationSupported() {
		t.Skip("POSIX owned groups")
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(t.TempDir(), "outer")
	mp, err := NewManagedProcess(t.Context(), exe, []string{"run", "--runtime-lifetime-helper", "nested", marker})
	if err != nil {
		t.Fatal(err)
	}
	defer mp.Close()
	deadline := time.Now().Add(5 * time.Second)
	for {
		if _, err := os.Stat(marker + ".nested"); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("nested runtime not ready")
		}
		time.Sleep(time.Millisecond)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Millisecond)
	defer cancel()
	started := time.Now()
	if err := mp.RequestStopContext(ctx); err != nil {
		t.Fatal(err)
	}
	if err := mp.Join(ctx); err == nil {
		t.Fatal("stalled nested cleanup reported success")
	}
	if time.Since(started) > 370*time.Millisecond {
		t.Fatal("root deadline refreshed")
	}
	for _, suffix := range []string{".remaining", ".nested.remaining"} {
		raw, err := os.ReadFile(marker + suffix)
		if err != nil {
			t.Fatal(err)
		}
		remaining, err := strconv.Atoi(strings.TrimSpace(string(raw)))
		if err != nil || remaining < 0 || remaining > 120 {
			t.Fatalf("nested cleanup refreshed root budget: %q", raw)
		}
	}
	raw, err := os.ReadFile(marker + ".nested-pid")
	if err != nil {
		t.Fatal(err)
	}
	pid, err := strconv.Atoi(string(raw))
	if err != nil || pid <= 1 {
		t.Fatalf("invalid nested fixture PID %q", raw)
	}
	defer func() {
		if pid > 1 && !runtimeFixtureGone(pid) {
			_ = proctree.KillGroup(pid)
		}
	}()
	// The child was assigned an owned group and inherited the same stop deadline.
	// Assert absence conservatively with signal-zero, including zombies.
	deadline = time.Now().Add(2500 * time.Millisecond)
	for !runtimeFixtureGone(pid) && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if !runtimeFixtureGone(pid) {
		t.Fatalf("nested fixture %d survived bounded stop", pid)
	}
}

func TestRuntimeNaturalCompletionCancelRace(t *testing.T) {
	if !WorkerCancellationSupported() {
		t.Skip("POSIX owned groups")
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for range 20 {
		marker := filepath.Join(t.TempDir(), "ready")
		mp, err := NewManagedProcess(t.Context(), exe, []string{"run", "--runtime-lifetime-helper", "natural", marker})
		if err != nil {
			t.Fatal(err)
		}
		deadline := time.Now().Add(5 * time.Second)
		for {
			if _, err := os.Stat(marker); err == nil {
				break
			}
			if time.Now().After(deadline) {
				mp.Close()
				t.Fatal("natural fixture not ready")
			}
			time.Sleep(time.Millisecond)
		}
		ctx, cancel := context.WithTimeout(context.Background(), WorkerShutdownTimeout)
		stopErr := mp.RequestStopContext(ctx)
		joinErr := mp.Join(ctx)
		cancel()
		if err := errors.Join(stopErr, joinErr); err != nil {
			t.Fatalf("completed runtime receipt lost to cancellation race: %v", err)
		}
	}
}
