//go:build windows

package mission

// syscall.Kill does not exist on Windows, and os.FindProcess succeeds for any
// pid there, so process liveness cannot be probed the way missionBusy needs.
//
// Report the pid as LIVE rather than dead. missionBusy reads a non-nil error as
// "not busy", which is the unsafe direction — it would let `apply` overwrite the
// artifacts of a running mission. The registry renders launchd plists, so apply
// never runs on Windows; this file exists so the package COMPILES there, and it
// fails closed, matching the design's own "refuse on a live pidfile" decision.
func syscallKill0(int) error { return nil }
