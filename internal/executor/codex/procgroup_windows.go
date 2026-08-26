//go:build windows

package codex

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
