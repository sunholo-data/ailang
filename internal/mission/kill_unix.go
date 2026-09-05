package mission

import "syscall"

// syscallKill0 tests for a live process without delivering a signal.
func syscallKill0(pid int) error { return syscall.Kill(pid, 0) }
