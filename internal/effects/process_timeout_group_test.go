//go:build unix

package effects

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/eval"
)

// A timed-out exec must take its whole process tree with it, not just the
// direct child. Measured 2026-10-16: a motoko agent's BashExec ran
// `cd /tmp/qtest && python3 gen.py && …`; the 30 s timeout SIGKILLed bash and
// returned, but python3 — a grandchild that never wrote, so never got SIGPIPE
// — was reparented to launchd and kept growing. Three of them reached ~18 GB
// and the box ran out of memory.
func TestProcessExec_TimeoutKillsGrandchildren(t *testing.T) {
	pidFile := filepath.Join(t.TempDir(), "grandchild.pid")
	ctx := newProcessCtx()
	ctx.Process.Timeout = 300 * time.Millisecond

	// sh forks sleep (a grandchild of this process), records its pid, waits.
	script := "sleep 300 & echo $! > " + pidFile + "; wait"
	result, err := Call(ctx, "Process", "exec", []eval.Value{
		&eval.StringValue{Value: "sh"},
		&eval.ListValue{Elements: []eval.Value{&eval.StringValue{Value: "-c"}, &eval.StringValue{Value: script}}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ctor, _ := unwrapErr(t, result); ctor != "Timeout" {
		t.Fatalf("error constructor = %q, want Timeout", ctor)
	}

	raw, err := os.ReadFile(pidFile)
	if err != nil {
		t.Fatalf("grandchild never recorded its pid: %v", err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(raw)))
	if err != nil {
		t.Fatalf("bad pid %q: %v", raw, err)
	}
	t.Cleanup(func() { _ = syscall.Kill(pid, syscall.SIGKILL) })

	deadline := time.Now().Add(2 * time.Second)
	for {
		if syscall.Kill(pid, 0) == syscall.ESRCH {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("grandchild %d is still alive after exec returned Timeout — the timeout killed only the direct child", pid)
		}
		time.Sleep(20 * time.Millisecond)
	}
}
