//go:build !windows

package riglock

import (
	"errors"
	"os"
	"syscall"
)

// pidAlive reports whether pid is a running process (Unix: signal 0).
// Inconclusive results (e.g. EPERM: the process exists but is not ours) count
// as alive — stealing a lock is only safe when the holder is positively gone.
func pidAlive(pid int) bool {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return true
	}
	sigErr := proc.Signal(syscall.Signal(0))
	if sigErr == nil {
		return true // process exists
	}
	if errors.Is(sigErr, os.ErrProcessDone) || errors.Is(sigErr, syscall.ESRCH) {
		return false // positively dead
	}
	return true // EPERM etc. — exists but not ours, or inconclusive
}
