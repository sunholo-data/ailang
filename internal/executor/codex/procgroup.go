package codex

import (
	"os/exec"
	"time"
)

// configureProcessTree makes the Codex CLI the leader of a process group. MCP
// servers and Chromium are descendants of that group, so timeout/cancellation
// cannot orphan them after the CLI leader exits.
func configureProcessTree(cmd *exec.Cmd) {
	setProcessGroup(cmd)
	cmd.Cancel = func() error {
		killProcessTree(cmd)
		return nil
	}
	cmd.WaitDelay = 2 * time.Second
}

func killProcessTree(cmd *exec.Cmd) {
	if cmd != nil && cmd.Process != nil {
		_ = killProcessGroup(cmd.Process.Pid)
	}
}
