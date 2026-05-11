//go:build !unix

package pkg

import (
	"os"
	"os/exec"
)

// setProcessGroup is a no-op on Windows (job objects would be needed).
func setProcessGroup(cmd *exec.Cmd) {
}

// killProcessGroup falls back to killing the leader process only on Windows.
func killProcessGroup(pid int) error {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return proc.Kill()
}
