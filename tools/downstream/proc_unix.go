//go:build !windows

package main

import (
	"os/exec"
	"syscall"
)

// ownGroup puts the check in its own process group so a timeout can kill
// everything it started (z3, a shell), not just the compiler.
func ownGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error { return killGroup(cmd) }
}

// killGroup SIGKILLs the check's whole process group. `nice` execs the
// compiler, so the group leader's pid is the compiler's.
func killGroup(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
}
