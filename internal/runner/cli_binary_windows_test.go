//go:build windows

package runner

// processAlive treats every pid as alive on Windows, so the stale-build sweep
// removes nothing there. Only the POSIX CLI-boundary tests build the binary.
func processAlive(int) bool { return true }
