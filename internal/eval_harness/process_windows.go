//go:build windows

package eval_harness

import (
	"os"
	"os/exec"
)

// SetProcessGroup is a no-op on Windows
// Windows uses job objects for process groups, which requires different implementation
func SetProcessGroup(cmd *exec.Cmd) {
	// No-op on Windows - process groups work differently
	// Future: Could implement using Windows Job Objects
}

// KillProcessGroup attempts to kill the process on Windows
// Note: This only kills the main process, not child processes
// Windows would need Job Objects for full process tree management
func KillProcessGroup(pid int) error {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return proc.Kill()
}

// KillProcess kills a single process on Windows
func KillProcess(pid int) error {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return proc.Kill()
}

// ProcessGroupRSS is unavailable on Windows (no ps, no process groups).
// Returning ok=false leaves the memory watchdog inert; the wall-clock
// timeout still guards runs.
func ProcessGroupRSS(pid int) (int64, bool) {
	return 0, false
}
