package executor

import (
	"errors"
	"os/exec"
)

// sigtermExitCode is 128+SIGTERM: what a CLI that traps SIGTERM and exits
// "cleanly" reports (Claude Code does exactly this — `exit status 143`).
const sigtermExitCode = 143

// IsExternalTermination reports whether a child's Wait error says it died of
// SIGTERM: either killed by the signal ("signal: terminated") or a CLI that
// trapped it and exited 143. The harness never sends SIGTERM — its own kills
// are SIGKILL through proctree — so this is a kill from outside the harness
// (see FinishExternalKill). Callers must check their own kill flags (timeout,
// cost, thrash) first: those outrank this.
func IsExternalTermination(err error) bool {
	var ee *exec.ExitError
	if !errors.As(err, &ee) || ee.ProcessState == nil {
		return false
	}
	return ee.ExitCode() == sigtermExitCode || ee.ProcessState.String() == "signal: terminated"
}
