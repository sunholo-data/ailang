//go:build !windows

package eval_harness

import (
	"os/exec"
	"syscall"
)

// SetProcessGroup configures the command to run in its own process group (Unix only)
func SetProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// KillProcessGroup kills the entire process group (Unix only)
// Uses negative PID to kill all processes in the group
func KillProcessGroup(pid int) error {
	return syscall.Kill(-pid, syscall.SIGKILL)
}

// KillProcess kills a single process
func KillProcess(pid int) error {
	return syscall.Kill(pid, syscall.SIGKILL)
}
