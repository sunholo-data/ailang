//go:build !windows

package pkg

import (
	"os/exec"
	"syscall"
)

// setProcessGroup makes the spawned command its own process-group leader so
// killing the group also reaps any sub-processes the smoke test forked
// (find, sleep, child shells, etc).
func setProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// killProcessGroup sends SIGKILL to the entire process group identified by pid.
// pid here is the leader's pid (Setpgid makes pid == pgid).
func killProcessGroup(pid int) error {
	return syscall.Kill(-pid, syscall.SIGKILL)
}
