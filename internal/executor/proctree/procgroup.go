package proctree

import (
	"os/exec"
	"time"
)

// Configure isolates a command in its own process group on Unix and installs
// context cancellation that kills that group. Call before Start on CommandContext.
// Windows preserves leader-only termination; descendant cleanup is unsupported.
func Configure(cmd *exec.Cmd) {
	setProcessGroup(cmd)
	cmd.Cancel = func() error {
		Kill(cmd)
		return nil
	}
	cmd.WaitDelay = 2 * time.Second
}

// Kill terminates the owned process group, including descendants whose leader
// has exited. It must only be used for a command prepared with Configure.
func Kill(cmd *exec.Cmd) {
	if cmd != nil && cmd.Process != nil {
		_ = killProcessGroup(cmd.Process.Pid)
	}
}
