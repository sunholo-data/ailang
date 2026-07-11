//go:build windows

package riglock

import (
	"errors"
	"syscall"
)

const (
	// PROCESS_QUERY_LIMITED_INFORMATION — enough to query liveness without the
	// broader rights PROCESS_QUERY_INFORMATION needs (not defined in syscall).
	processQueryLimitedInformation = 0x1000
	// STILL_ACTIVE — GetExitCodeProcess result while the process is running.
	stillActive = 259
	// ERROR_INVALID_PARAMETER — OpenProcess result for a nonexistent PID.
	errorInvalidParameter = syscall.Errno(87)
)

// pidAlive reports whether pid is a running process. Windows has no signal 0:
// os.FindProcess errors for ANY unopenable PID (which the Unix-style caller
// would misread as "assume alive"), so use OpenProcess semantics directly —
// ERROR_INVALID_PARAMETER means the PID does not exist (positively dead);
// access-denied means it exists but is not ours (alive). An opened handle can
// outlive its process, so double-check with GetExitCodeProcess. Inconclusive
// results count as alive — stealing is only safe when the holder is
// positively gone.
func pidAlive(pid int) bool {
	h, err := syscall.OpenProcess(processQueryLimitedInformation, false, uint32(pid))
	if err != nil {
		return !errors.Is(err, errorInvalidParameter)
	}
	defer syscall.CloseHandle(h)
	var code uint32
	if err := syscall.GetExitCodeProcess(h, &code); err != nil {
		return true // inconclusive — do not steal
	}
	return code == stillActive
}
