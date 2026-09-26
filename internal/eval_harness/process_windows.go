//go:build windows

package eval_harness

// ProcessGroupRSS is unavailable on Windows (no ps, no process groups).
// Returning ok=false leaves the memory watchdog inert; the wall-clock
// timeout still guards runs.
func ProcessGroupRSS(pid int) (int64, bool) {
	return 0, false
}
