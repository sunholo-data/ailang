package runner

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/testutil"
)

// The CLI-boundary tests in this package (worker exit, worker and terminal
// PTY, existing worker fixtures) run the real `ailang` binary. Each used to
// `go build` it inside the test, under a context.WithTimeout(120s) that
// several of them then also passed to every subtest: the build and ~34
// subtests shared ONE budget. On a loaded macOS runner
// TestWorkerCLIProcessExitLifecycle took 122s, and its last subtest
// (original-select-handler-false) "failed" after 0.00s — the shared deadline
// was spent before it started, and the failure named the wrong subtest.
//
// Now the binary is built once per test process, outside every subtest's
// budget (bounded only by the package -timeout), and each subtest runs under
// its own deadline sized from what it does (runCLIProbe).

var cliBinary struct {
	once  sync.Once
	dir   string
	path  string
	root  string
	err   error
	built time.Duration
}

// ailangCLIBinary returns a freshly built ailang binary (shared by every test
// in this process) and the repository root.
func ailangCLIBinary(t *testing.T) (binary, root string) {
	t.Helper()
	cliBinary.once.Do(func() {
		root, err := filepath.Abs(filepath.Join("..", ".."))
		if err != nil {
			cliBinary.err = err
			return
		}
		dir, err := os.MkdirTemp("", "runner-ailang-*")
		if err != nil {
			cliBinary.err = err
			return
		}
		cliBinary.dir, cliBinary.root = dir, root
		path := filepath.Join(dir, "ailang")
		// No per-test budget applies to the build; the package -timeout
		// (via HangGuard) is its only bound.
		ctx, cancel := context.WithTimeout(context.Background(), testutil.HangGuard(t, 15*time.Minute))
		defer cancel()
		start := time.Now()
		build := exec.CommandContext(ctx, "go", "build", "-o", path, "./cmd/ailang")
		build.Dir = root
		if out, err := build.CombinedOutput(); err != nil {
			cliBinary.err = fmt.Errorf("go build ./cmd/ailang: %w\n%s", err, out)
			return
		}
		cliBinary.path, cliBinary.built = path, time.Since(start)
	})
	if cliBinary.err != nil {
		t.Fatalf("building the CLI under test: %v", cliBinary.err)
	}
	t.Logf("ailang CLI built once for this package in %s", cliBinary.built.Round(time.Millisecond))
	return cliBinary.path, cliBinary.root
}

// removeCLIBinary deletes the shared build; TestMain calls it after m.Run.
func removeCLIBinary() {
	if cliBinary.dir != "" {
		_ = os.RemoveAll(cliBinary.dir)
	}
}

// runCLIProbe runs one subprocess under the calling subtest's OWN deadline:
// need, capped by what is left of the package -timeout. When that deadline
// kills the probe, the failure names this subtest, its budget and how long it
// actually ran, so an overrun is reported where it happened instead of
// spending a shared budget that the next subtest then fails on at 0.00s.
func runCLIProbe(t *testing.T, need time.Duration, configure func(*exec.Cmd), name string, args ...string) ([]byte, error) {
	t.Helper()
	budget := testutil.HangGuard(t, need)
	ctx, cancel := context.WithTimeout(t.Context(), budget)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	// A grandchild that inherits the output pipe must not hold Wait past
	// the deadline.
	cmd.WaitDelay = 5 * time.Second
	if configure != nil {
		configure(cmd)
	}
	start := time.Now()
	out, err := cmd.CombinedOutput()
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		t.Fatalf("%s overran its own %s budget: killed after %s (the package -timeout leaves %s); output so far:\n%s",
			t.Name(), budget, time.Since(start).Round(time.Millisecond), timeLeft(t), out)
	}
	return out, err
}

func timeLeft(t *testing.T) string {
	if deadline, ok := t.Deadline(); ok {
		return time.Until(deadline).Round(time.Second).String()
	}
	return "no deadline"
}
