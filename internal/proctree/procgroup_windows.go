//go:build windows

package proctree

import (
	"os"
	"os/exec"
)

// Windows has no process groups without Job Objects; only the leader dies.
func setProcessGroup(_ *exec.Cmd) {}

func killProcessGroup(pid int) error { return killProcess(pid) }

func killProcess(pid int) error {
	process, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return process.Kill()
}
