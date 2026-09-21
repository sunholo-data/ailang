package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/policy"
	"github.com/sunholo-data/ailang/internal/proctree"
)

// M-EXECUTOR-POLICY-HARDENING M3 (D5) — the supervisor around a policy run.
//
// `ailang run --policy P prog.ail` is the PARENT. It resolves the policy,
// arms the wall-clock deadline, and starts this same binary as a worker
// (`run --policy-worker 3 …` with the same argv tail) in its own process
// group. The worker admits and executes; the parent:
//
//   - reads the admission decision from a dedicated control pipe (fd 3 in
//     the worker) and re-emits it in the documented shape — a program that
//     prints `policy: {"ok":true…}` on its own stdout is not believed;
//   - copies stdout/stderr through a shared byte counter and stops the
//     worker at max_output_bytes instead of buffering unbounded output;
//   - kills the whole process group at timeout_ms (proctree), so a
//     trusted_host Process child cannot outlive the run;
//   - hands the worker an allowlisted environment in restricted mode.
//
// The deadline starts BEFORE the worker is spawned, so source loading and
// compilation are inside it (design §5).

// workerEnvAllow is the restricted worker's whole environment. Provider
// credentials stay in the model-facing host; AILANG_* widening knobs never
// reach the worker; the policy's sandbox is set by the worker itself.
var workerEnvAllow = []string{
	"PATH", "HOME", "TMPDIR", "TMP", "TEMP", "TZ", "LANG", "LC_ALL", "LC_CTYPE",
	"USER", "LOGNAME", "AILANG_STDLIB_PATH", "AILANG_QUIET_WARNINGS", "AILANG_TRACE",
	"AILANG_SEED", "GOCACHE", "GOFLAGS",
}

// supervisePolicyRun runs the worker and returns the exit code the parent
// should use.
func supervisePolicyRun(policyPath string, w runPolicyWidening, argsAfterRun []string) int {
	res, _ := resolveRunPolicy(policyPath, w)

	exe, err := os.Executable()
	if err != nil {
		refusePolicy("cannot locate the ailang binary for the worker: %v", err)
	}

	// The clock starts now: before the worker exists, let alone reads source.
	ctx, cancel := context.WithTimeout(context.Background(), res.Timeout)
	defer cancel()

	ctrlR, ctrlW, err := os.Pipe()
	if err != nil {
		refusePolicy("cannot create the control pipe: %v", err)
	}

	args := append([]string{"run", "--policy-worker", "3"}, argsAfterRun...)
	cmd := exec.CommandContext(ctx, exe, args...)
	cmd.ExtraFiles = []*os.File{ctrlW} // fd 3 in the worker
	cmd.Stdin = os.Stdin
	cmd.Env = workerEnv(res)
	proctree.Configure(cmd)
	cmd.WaitDelay = proctree.DefaultWaitDelay

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		refusePolicy("stdout pipe: %v", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		refusePolicy("stderr pipe: %v", err)
	}
	if err := cmd.Start(); err != nil {
		refusePolicy("cannot start the worker: %v", err)
	}
	_ = ctrlW.Close() // the worker holds the write end now

	// Output cap: one counter across both streams; at the cap we stop
	// copying and kill the group — never capture-then-truncate.
	var written atomic.Int64
	var overCap atomic.Bool
	limit := res.MaxOutputBytes
	copyCapped := func(dst io.Writer, src io.Reader) {
		buf := make([]byte, 32*1024)
		for {
			n, rerr := src.Read(buf)
			if n > 0 {
				if limit > 0 && written.Add(int64(n)) > limit {
					if overCap.CompareAndSwap(false, true) {
						cancel()
					}
					return
				}
				_, _ = dst.Write(buf[:n])
			}
			if rerr != nil {
				return
			}
		}
	}
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); copyCapped(os.Stdout, stdout) }()
	go func() { defer wg.Done(); copyCapped(os.Stderr, stderr) }()

	// The control line: exactly one message, or none if the worker died
	// before deciding.
	var ctrl controlMessage
	var gotCtrl bool
	ctrlDone := make(chan struct{})
	go func() {
		defer close(ctrlDone)
		defer ctrlR.Close()
		sc := bufio.NewScanner(ctrlR)
		sc.Buffer(make([]byte, 0, 64*1024), 4<<20)
		if sc.Scan() {
			if jerr := json.Unmarshal(sc.Bytes(), &ctrl); jerr == nil {
				gotCtrl = true
			}
		}
	}()

	waitErr := cmd.Wait()
	wg.Wait()
	<-ctrlDone

	// Re-emit the decision in the documented shape, from the control pipe
	// only.
	if gotCtrl {
		switch ctrl.Kind {
		case "denied":
			emitJSON(ctrl.Decision)
			return ctrl.Code
		case "admitted":
			if ctrl.Admission != nil {
				b, _ := json.Marshal(ctrl.Admission)
				fmt.Fprintf(os.Stderr, "policy: %s\n", b)
			}
		}
	}

	switch {
	case overCap.Load():
		fmt.Fprintf(os.Stderr, "policy-result: %s\n", limitEnvelope(res, "execute", "output_limit",
			fmt.Sprintf("combined stdout+stderr exceeded max_output_bytes (%d); worker killed", limit)))
		return 3
	case errors.Is(ctx.Err(), context.DeadlineExceeded):
		fmt.Fprintf(os.Stderr, "policy-result: %s\n", limitEnvelope(res, stageFor(gotCtrl), "timeout",
			fmt.Sprintf("exceeded timeout_ms (%s); worker process group killed", res.Timeout)))
		return 3
	}

	if waitErr != nil {
		var exitErr *exec.ExitError
		if errors.As(waitErr, &exitErr) {
			return exitErr.ExitCode()
		}
		fmt.Fprintf(os.Stderr, "policy-result: %s\n", limitEnvelope(res, stageFor(gotCtrl), "worker_failed", waitErr.Error()))
		return 1
	}
	if !gotCtrl {
		// The worker exited 0 without deciding: never treat as admitted.
		fmt.Fprintf(os.Stderr, "policy-result: %s\n", limitEnvelope(res, "admit", "no_decision",
			"the worker exited without reporting an admission decision on the control channel"))
		return 1
	}
	return 0
}

func stageFor(admitted bool) string {
	if admitted {
		return "execute"
	}
	return "admit"
}

// workerEnv is the worker's environment: the full parent environment for
// trusted_host (operator-approved host integrations need their
// credentials), the allowlist for restricted.
func workerEnv(res *policy.Resolved) []string {
	if !res.Restricted() {
		return os.Environ()
	}
	out := make([]string, 0, len(workerEnvAllow))
	for _, name := range workerEnvAllow {
		if config.RawSet(name) {
			out = append(out, name+"="+config.Raw(name))
		}
	}
	return out
}
