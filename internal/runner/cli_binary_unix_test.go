//go:build !windows

package runner

import (
	"errors"
	"syscall"
)

// processAlive reports whether pid exists: signal 0 delivers nothing and only
// checks. EPERM means it exists but belongs to someone else.
func processAlive(pid int) bool {
	err := syscall.Kill(pid, 0)
	return err == nil || errors.Is(err, syscall.EPERM)
}
