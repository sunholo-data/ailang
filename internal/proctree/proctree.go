// Package proctree owns the one process-group recipe for every subprocess
// AILANG spawns: the child becomes its own group leader, and cancelling its
// context kills the whole group, so descendants (an MCP server, motoko's
// env-server on port 8080, a smoke test's forked shells) die with it.
//
// It is a LEAF (stdlib + syscall only). internal/smt and internal/pkg are
// language-core packages and must never import anything under
// internal/executor, which is where the previous shared copy lived; three
// further byte-equivalent copies (executor/motoko, smt, pkg) existed only
// because of that boundary. M-V1-SIMPLIFY-S3 M5.
package proctree

import (
	"os/exec"
	"time"
)

// DefaultWaitDelay bounds Wait after the group kill lands; callers that need
// a longer grace period (a solver, a whole agent run) set cmd.WaitDelay after
// Configure.
const DefaultWaitDelay = 2 * time.Second

// Configure isolates a command in its own process group on Unix and installs
// context cancellation that kills that group. Call before Start on a command
// created with exec.CommandContext. Windows preserves leader-only
// termination; descendant cleanup is unsupported there.
func Configure(cmd *exec.Cmd) {
	SetGroup(cmd)
	cmd.Cancel = func() error {
		Kill(cmd)
		return nil
	}
	cmd.WaitDelay = DefaultWaitDelay
}

// Kill terminates the owned process group, including descendants whose leader
// has exited. It must only be used for a command prepared with Configure or
// SetGroup.
func Kill(cmd *exec.Cmd) {
	if cmd != nil && cmd.Process != nil {
		_ = KillGroup(cmd.Process.Pid)
	}
}

// SetGroup makes the command its own process-group leader (Setpgid) without
// installing a Cancel func. Use Configure unless the caller owns cancellation.
func SetGroup(cmd *exec.Cmd) { setProcessGroup(cmd) }

// KillGroup SIGKILLs the whole process group led by pid (Setpgid makes
// pgid == pid). A group that is already gone is not an error.
func KillGroup(pid int) error { return killProcessGroup(pid) }

// KillProcess SIGKILLs a single process — the fallback when pid is not a
// group leader (an orphan found by port, not one we spawned).
func KillProcess(pid int) error { return killProcess(pid) }
