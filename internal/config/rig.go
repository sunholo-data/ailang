package config

import (
	"fmt"
	"strconv"
	"time"
)

// The rig lock: one GPU, one holder (internal/riglock).
const (
	EnvRigLockDir     = "RIG_LOCK_DIR"
	EnvRigSharedDir   = "RIG_SHARED_DIR"
	EnvRigLockStale   = "RIG_LOCK_STALE_MIN"
	EnvRigLockHeld    = "AILANG_RIG_LOCK_HELD"
	EnvRigHandoffFile = "RIG_HANDOFF_FILE"
)

// DefaultRigSharedDir is the shared directory every user on the rig can
// reach, so agents under different accounts hold ONE lock.
const DefaultRigSharedDir = "/Users/Shared/ailang"

// DefaultRigLockStaleMin is the minutes after which a holder with no
// heartbeat is presumed dead (6h, matching tools/launchd/rig-lock.sh).
const DefaultRigLockStaleMin = 360

var rigVars = []Var{
	{EnvRigLockDir, "", AreaRig, "Lock directory; unset derives <RIG_SHARED_DIR>/rig.lock.d when that shared directory exists, else rig.lock.d under the state dir."},
	{EnvRigSharedDir, DefaultRigSharedDir, AreaRig, "Machine-wide parent for the lock so agents under different OS users contend for one; used only when an operator has created it."},
	{EnvRigLockStale, strconv.Itoa(DefaultRigLockStaleMin), AreaRig, "Minutes without a heartbeat before a holder is presumed dead; a non-positive or malformed value keeps the default."},
	{EnvRigLockHeld, "0", AreaRig, "Set to 1 by a lock holder for its children, which then skip their own acquire."},
	{EnvRigHandoffFile, "", AreaRig, "Path of the yield hand-off file; unset derives rig.handoff beside the lock directory."},
}

// RigLockDir returns RIG_LOCK_DIR, "" when unset.
func RigLockDir() string { return get(EnvRigLockDir) }

// RigSharedDir returns RIG_SHARED_DIR, default DefaultRigSharedDir.
func RigSharedDir() string { return getOr(EnvRigSharedDir) }

// RigLockStaleWindow returns RIG_LOCK_STALE_MIN as a duration, default
// DefaultRigLockStaleMin minutes. Parsed with Sscanf %d as the lock always
// did, so "30abc" still reads 30.
func RigLockStaleWindow() time.Duration {
	if v := get(EnvRigLockStale); v != "" {
		var n int
		if _, err := fmt.Sscanf(v, "%d", &n); err == nil && n > 0 {
			return time.Duration(n) * time.Minute
		}
	}
	return DefaultRigLockStaleMin * time.Minute
}

// RigLockHeld reports AILANG_RIG_LOCK_HELD=1.
func RigLockHeld() bool { return getOr(EnvRigLockHeld) == "1" }

// RigHandoffFile returns RIG_HANDOFF_FILE, "" when unset.
func RigHandoffFile() string { return get(EnvRigHandoffFile) }
