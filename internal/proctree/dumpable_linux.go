//go:build linux

package proctree

import (
	"fmt"

	"golang.org/x/sys/unix"
)

// DisableDump marks the calling process non-dumpable (prctl PR_SET_DUMPABLE 0).
//
// Audit H-6, M-EXECUTOR-UID-SPLIT P0.2. The cloud executor parent and the agent
// CLI it spawns share one UID, so without this the child reads every secret the
// job was started with from /proc/$PPID/environ (or /proc/1/environ: the parent
// is the container entrypoint), and its memory from /proc/$PPID/mem. A
// non-dumpable process has those files owned by root and refuses ptrace from
// same-UID processes, which is the ssh-agent technique. The child's execve
// resets its own dumpable flag, so the agent CLI itself is unaffected.
//
// It reads the flag back rather than trusting the prctl return value.
func DisableDump() error {
	if err := unix.Prctl(unix.PR_SET_DUMPABLE, 0, 0, 0, 0); err != nil {
		return fmt.Errorf("prctl(PR_SET_DUMPABLE, 0): %w", err)
	}
	got, err := unix.PrctlRetInt(unix.PR_GET_DUMPABLE, 0, 0, 0, 0)
	if err != nil {
		return fmt.Errorf("prctl(PR_GET_DUMPABLE): %w", err)
	}
	if got != 0 {
		return fmt.Errorf("prctl(PR_SET_DUMPABLE, 0) did not take: PR_GET_DUMPABLE = %d", got)
	}
	return nil
}
