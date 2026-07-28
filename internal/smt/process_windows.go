//go:build windows

package smt

import (
	"os"
	"os/exec"
)

func setProcessGroup(_ *exec.Cmd) {}

func killProcessGroup(pid int) error {
	// Windows requires job objects to reliably kill a process tree. Until that
	// is added, WaitDelay still bounds inherited-pipe waits after killing the
	// command process.
	process, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return process.Kill()
}
