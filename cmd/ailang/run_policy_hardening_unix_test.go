//go:build !windows

package main

import "syscall"

// processGone reports whether pid no longer exists (ESRCH on a signal-0 probe).
func processGone(pid int) bool { return syscall.Kill(pid, 0) == syscall.ESRCH }

// killProcess is the test's own cleanup for a survivor it is about to report.
func killProcess(pid int) { _ = syscall.Kill(pid, syscall.SIGKILL) }
