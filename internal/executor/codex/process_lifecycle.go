package codex

import (
	"fmt"
	"os/exec"
	"sync"
	"time"
)

// nativeProcessLifecycle shares one Wait between the event reader and cleanup.
// Cleanup always kills the owned group, including children of an exited leader,
// then joins the refresh owner before a cloud credential lease can be released.
func nativeProcessLifecycle(cmd *exec.Cmd) (wait func() error, stop func(), finish func() error) {
	var waitOnce sync.Once
	waitDone := make(chan struct{})
	var waitErr error
	beginWait := func() { waitOnce.Do(func() { go func() { waitErr = cmd.Wait(); close(waitDone) }() }) }
	wait = func() error { beginWait(); <-waitDone; return waitErr }
	var stopOnce sync.Once
	var terminationErr error
	stop = func() { stopOnce.Do(func() { terminationErr = terminateProcessTree(cmd) }) }
	// Context cancellation and explicit timeout share the same successful group
	// kill. A repeated signal after reap can be rejected for zombie-only groups.
	cmd.Cancel = func() error { stop(); return terminationErr }
	finish = func() error {
		stop()
		killErr := terminationErr
		beginWait()
		select {
		case <-waitDone:
			if killErr != nil {
				return fmt.Errorf("%w: wait=%v terminate=%v", ErrProcessTerminationUnconfirmed, waitErr, killErr)
			}
		case <-time.After(5 * time.Second):
			return fmt.Errorf("%w: cleanup join timed out", ErrProcessTerminationUnconfirmed)
		}
		return nil
	}
	return wait, stop, finish
}
