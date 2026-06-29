//go:build windows

package motoko

import (
	"os"
	"os/exec"
)

// setProcessGroup is a no-op on Windows (process groups need Job Objects).
// motoko's eval rig is Unix; this exists only to keep the build cross-platform.
func setProcessGroup(cmd *exec.Cmd) {}

// killProcessGroup kills the main process on Windows (no group/tree management).
func killProcessGroup(pid int) error {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return proc.Kill()
}

// killProcess kills a single process on Windows.
func killProcess(pid int) error {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return proc.Kill()
}
