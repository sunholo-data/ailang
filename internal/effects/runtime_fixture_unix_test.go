//go:build unix

package effects

import "syscall"

func runtimeFixtureGone(pid int) bool { return syscall.Kill(pid, 0) == syscall.ESRCH }
