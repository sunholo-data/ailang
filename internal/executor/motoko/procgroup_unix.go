//go:build !windows

package motoko

import (
	"os/exec"
	"syscall"
)

// setProcessGroup runs the child in its OWN process group so the whole group
// can be killed together on timeout. motoko's env-server runs inside the bun
// host process and (per the eval adapter) binds the fixed ENV_PORT=8080; a bare
// Process.Kill() on the leader can orphan children and leave 8080 LISTENing,
// which silently crashes every later motoko spawn with "terminated without
// emitting run_summary" — the 10h rig-wedge of 2026-06-29. Killing the GROUP
// (negative pid) reaps them all. Mirrors internal/eval_harness/process_unix.go
// (can't import it: that package imports the executors → cycle).
func setProcessGroup(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setpgid = true
}

// killProcessGroup SIGKILLs the entire process group led by pid (Setpgid makes
// pgid == pid). The negative pid targets the group, not just the leader.
func killProcessGroup(pid int) error {
	return syscall.Kill(-pid, syscall.SIGKILL)
}

// killProcess SIGKILLs a single process (fallback when it is not a group leader).
func killProcess(pid int) error {
	return syscall.Kill(pid, syscall.SIGKILL)
}
