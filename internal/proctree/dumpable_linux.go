//go:build linux

package proctree

import (
	"fmt"
	"syscall"
)

// prctl options (linux/prctl.h). Raw syscall, not golang.org/x/sys/unix:
// proctree is a stdlib-only leaf (TestProctreeIsALeaf).
const (
	prGetDumpable = 3
	prSetDumpable = 4
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
	if _, _, errno := syscall.RawSyscall(syscall.SYS_PRCTL, prSetDumpable, 0, 0); errno != 0 {
		return fmt.Errorf("prctl(PR_SET_DUMPABLE, 0): %w", errno)
	}
	got, _, errno := syscall.RawSyscall(syscall.SYS_PRCTL, prGetDumpable, 0, 0)
	if errno != 0 {
		return fmt.Errorf("prctl(PR_GET_DUMPABLE): %w", errno)
	}
	if got != 0 {
		return fmt.Errorf("prctl(PR_SET_DUMPABLE, 0) did not take: PR_GET_DUMPABLE = %d", got)
	}
	return nil
}
