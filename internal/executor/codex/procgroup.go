package codex

import (
	"errors"
	"github.com/sunholo-data/ailang/internal/proctree"
	"os"
	"os/exec"
	"runtime"
)

func configureProcessTree(cmd *exec.Cmd) { proctree.Configure(cmd) }
func killProcessTree(cmd *exec.Cmd)      { proctree.Kill(cmd) }

func terminateProcessTree(cmd *exec.Cmd) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}
	if runtime.GOOS == "windows" {
		// Reuse the command's process handle. Looking up a reaped PID can fail or
		// target a reused PID; Windows has no owned process-group guarantee here.
		err := cmd.Process.Kill()
		if errors.Is(err, os.ErrProcessDone) {
			return nil
		}
		return err
	}
	return proctree.KillGroup(cmd.Process.Pid)
}
