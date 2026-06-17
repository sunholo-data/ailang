// Package riglock provides mutual exclusion for local-rig eval jobs that share
// a single GPU. It is the Go counterpart to tools/launchd/rig-lock.sh and uses
// the SAME lock directory and staleness semantics, so a Go-side `ailang
// eval-suite` and a shell-side launchd job (nightly-eval, lang-eval,
// os-rotation-filler) never run concurrently.
//
// Why: the rig is a single GPU / bandwidth-bound box. Concurrent runs thrash,
// and an ollama model reload mid-request silently kills a stream — which (on the
// /v1 tool-calling path) can hang a run for hours. Historically only the shell
// jobs took the lock; an ad-hoc `ailang eval-suite` from a shell did NOT, so it
// could collide with the rotation. This package closes that gap so the lock is
// enforced by the command we already run, not by remembering to source a script.
//
// The lock is an atomic mkdir (portable; macOS has no flock). A lock whose
// directory is older than the staleness window is stolen (crash recovery).
package riglock

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	// EnvHeld is exported by an ancestor that already holds the lock
	// (tools/launchd/rig-lock.sh sets it after acquiring, and Acquire sets it
	// for child processes). When present, Acquire is a no-op so a wrapper-driven
	// eval-suite does not deadlock against its own parent's lock.
	EnvHeld = "AILANG_RIG_LOCK_HELD"

	// EnvLockDir overrides the lock directory (mirrors rig-lock.sh RIG_LOCK_DIR).
	EnvLockDir = "RIG_LOCK_DIR"

	// EnvStaleMin overrides the staleness window in minutes (RIG_LOCK_STALE_MIN).
	EnvStaleMin = "RIG_LOCK_STALE_MIN"

	defaultStaleMin = 360 // steal a lock older than 6h (matches rig-lock.sh)
)

// Mode controls Acquire's blocking behaviour.
type Mode int

const (
	// NoWait returns immediately with held=false if the lock is taken.
	NoWait Mode = iota
	// Wait blocks (polling) until the lock is free.
	Wait
)

// Release frees the lock. It is always safe to call (idempotent, nil-safe).
type Release func()

func lockDir() string {
	if d := os.Getenv(EnvLockDir); d != "" {
		return d
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		home = os.TempDir()
	}
	return filepath.Join(home, ".ailang", "state", "rig.lock.d")
}

func staleWindow() time.Duration {
	if v := os.Getenv(EnvStaleMin); v != "" {
		var n int
		if _, err := fmt.Sscanf(v, "%d", &n); err == nil && n > 0 {
			return time.Duration(n) * time.Minute
		}
	}
	return defaultStaleMin * time.Minute
}

// HeldByAncestor reports whether an ancestor process already holds the lock
// (via EnvHeld). Callers can skip their own acquire when true.
func HeldByAncestor() bool {
	return os.Getenv(EnvHeld) == "1"
}

// Holder returns a human-readable description of the current lock holder
// ("PID 123 since 2026-...") or "" if the lock is free/unreadable. Best-effort.
func Holder() string {
	b, err := os.ReadFile(filepath.Join(lockDir(), "holder"))
	if err != nil {
		return ""
	}
	fields := strings.Fields(strings.TrimSpace(string(b)))
	switch len(fields) {
	case 0:
		return ""
	case 1:
		return "PID " + fields[0]
	default:
		return "PID " + fields[0] + " since " + fields[1]
	}
}

// Acquire attempts to take the rig lock.
//
//   - If an ancestor already holds it (EnvHeld=1), returns (true, noop-release,
//     nil) without touching the filesystem — the ancestor owns release.
//   - In NoWait mode, returns (false, noop, nil) if the lock is held by another
//     live holder; the caller should report Holder() and exit.
//   - In Wait mode, blocks until the lock is free.
//
// A stale lock (directory older than the staleness window) is stolen. On
// success Acquire writes a holder file and sets EnvHeld=1 for child processes.
func Acquire(mode Mode) (bool, Release, error) {
	if HeldByAncestor() {
		return true, func() {}, nil
	}
	dir := lockDir()
	if err := os.MkdirAll(filepath.Dir(dir), 0o755); err != nil {
		return false, func() {}, fmt.Errorf("riglock: cannot create state dir: %w", err)
	}
	for {
		err := os.Mkdir(dir, 0o755)
		if err == nil {
			break // acquired
		}
		if !os.IsExist(err) {
			return false, func() {}, fmt.Errorf("riglock: mkdir lock: %w", err)
		}
		// Lock is held — steal it if stale (holder crashed without releasing).
		if fi, statErr := os.Stat(dir); statErr == nil {
			if time.Since(fi.ModTime()) > staleWindow() {
				_ = os.RemoveAll(dir)
				continue
			}
		}
		if mode == NoWait {
			return false, func() {}, nil
		}
		time.Sleep(30 * time.Second)
	}

	pid := os.Getpid()
	_ = os.WriteFile(filepath.Join(dir, "holder"),
		[]byte(fmt.Sprintf("%d %s", pid, time.Now().UTC().Format(time.RFC3339))), 0o644)
	_ = os.Setenv(EnvHeld, "1")

	var released bool
	release := func() {
		if released {
			return
		}
		released = true
		_ = os.RemoveAll(dir)
		_ = os.Unsetenv(EnvHeld)
	}
	return true, release, nil
}
