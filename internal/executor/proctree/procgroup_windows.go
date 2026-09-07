//go:build windows

package proctree

import (
	"os"
	"os/exec"
)

func setProcessGroup(_ *exec.Cmd) {}

func killProcessGroup(pid int) error {
	process, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return process.Kill()
}
