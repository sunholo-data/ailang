package riglock

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// EnvOwner identifies the wrapper that owns the lock while its child evaluates.
// The wrapper is blocked waiting for that child, so the child can cooperatively
// lend the lock between trials and restore the same ownership before returning.
const EnvOwner = "AILANG_RIG_LOCK_OWNER_PID"

// PriorityPending reads PID-named requests outside the lock directory, so releasing
// the lock cannot erase a waiter. A live requester keeps its reservation through
// all of its stages. Dead requesters do not block evaluation indefinitely.
func PriorityPending() bool {
	entries, err := os.ReadDir(lockDir() + ".priority")
	if os.IsNotExist(err) {
		return false
	}
	if err != nil {
		return true // unreadable coordination state must not grant the GPU
	}
	for _, entry := range entries {
		pid, err := strconv.Atoi(entry.Name())
		if err != nil || pid <= 0 || entry.IsDir() {
			continue
		}
		if pidAlive(pid) {
			return true
		}
		_ = os.Remove(filepath.Join(lockDir()+".priority", entry.Name()))
	}
	return false
}

func ownedBy(pid string) bool {
	b, err := os.ReadFile(filepath.Join(lockDir(), "holder"))
	f := strings.Fields(string(b))
	return err == nil && len(f) > 0 && f[0] == pid
}

// Yield lends the GPU to a waiting priority task. The caller MUST drain all
// in-flight GPU work first. It returns only after restoring the original lock,
// or an error, in which case no further evaluation may run.
func Yield(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !HeldByAncestor() || !PriorityPending() {
		return nil
	}
	owner := os.Getenv(EnvOwner)
	if owner == "" || !ownedBy(owner) {
		return fmt.Errorf("riglock: cannot yield a lock without verified ownership")
	}
	if err := os.RemoveAll(lockDir()); err != nil {
		return err
	}
	fmt.Fprintln(os.Stderr, "riglock: yielded at a safe boundary for priority work")
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if !PriorityPending() {
			err := os.Mkdir(lockDir(), 0o755)
			if err == nil {
				// A requester may have appeared between the check and mkdir.
				if PriorityPending() {
					_ = os.Remove(lockDir())
				} else {
					err = os.WriteFile(filepath.Join(lockDir(), "holder"), []byte(owner+" "+time.Now().UTC().Format(time.RFC3339)), 0o644)
					if err != nil {
						_ = os.RemoveAll(lockDir())
						return err
					}
					fmt.Fprintln(os.Stderr, "riglock: priority work finished; evaluation resumed")
					return nil
				}
			} else if os.IsExist(err) {
				if !holderAlive(lockDir()) {
					_ = os.RemoveAll(lockDir())
				}
			} else {
				return err
			}
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(200 * time.Millisecond):
		}
	}
}
