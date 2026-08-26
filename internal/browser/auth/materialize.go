package auth

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	// materializationPrefix marks a directory as owned by this subsystem. The
	// startup orphan audit uses it to tell our disposable state apart from
	// anything else that shares the root, so it must never change casually.
	//
	// It deliberately encodes nothing about the profile: the directory path
	// reaches child argv, and a path that embedded the alias or profile hash
	// would publish the canonical object reference to every process listing.
	materializationPrefix = "bstate-"

	// storageStateFilename is fixed rather than derived, for the same reason.
	storageStateFilename = "storage-state.json"

	// OpOrphanAudit is the Op recorded on the audit events the startup sweep
	// writes. It uses the AuditEvent vocabulary rather than a second one, so a
	// sink that already stores profile decisions stores these too.
	OpOrphanAudit = "orphan_audit"
)

// MaterializeOptions configures one disposable materialization.
type MaterializeOptions struct {
	// SessionRoot is the directory that holds materializations. It is required:
	// defaulting it would let a caller silently scatter decrypted credentials
	// into a directory the orphan audit does not sweep.
	SessionRoot string

	// RunID and ProfileHash are safe correlation values used in diagnostics.
	// Neither is used to build the path.
	RunID       string
	ProfileHash string
}

// Materialization is credential-grade browser state decrypted to disk for
// exactly one run. It owns its directory and is responsible for destroying it.
//
// The handle is deliberately independent of the browser controller: it can be
// created, used, and destroyed on its own, so the destroy-on-every-exit-path
// guarantee is testable without a provider or a session.
type Materialization struct {
	mu        sync.Mutex
	destroyed bool

	dir         string
	path        string
	runID       string
	profileHash string

	// removeAll is the deletion primitive, injectable so that cleanup-failure
	// handling can be exercised without an unwritable filesystem.
	removeAll func(string) error
}

// Path is the materialized storage-state file. It is safe to pass to a child
// process as an argument; its CONTENTS are not safe to log.
func (m *Materialization) Path() string { return m.path }

// Dir is the session-owned directory holding the file.
func (m *Materialization) Dir() string { return m.dir }

func (m *Materialization) Destroyed() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.destroyed
}

func (m *Materialization) String() string {
	return fmt.Sprintf("browser auth materialization %s (run %s, profile %s)",
		Redacted, safeOrUnset(m.runID), safeOrUnset(m.profileHash))
}

func (m *Materialization) GoString() string { return m.String() }

func safeOrUnset(value string) string {
	if value == "" {
		return "unset"
	}
	return value
}

// MaterializeStorageState decrypts canonical material into a fresh 0600 file
// inside a fresh 0700 directory under opts.SessionRoot.
//
// It is all-or-nothing: if anything fails after the directory exists, the
// directory is removed before returning, so a failed materialization never
// leaves plaintext behind. On failure the handle is nil.
func MaterializeStorageState(ctx context.Context, protector KeyProtector, sealed SealedEnvelope, opts MaterializeOptions) (*Materialization, error) {
	if opts.SessionRoot == "" {
		return nil, NewFailureReason(FailureMaterializeFailed, "materialize storage state", "no session root")
	}
	if err := ctx.Err(); err != nil {
		return nil, NewFailureReason(FailureMaterializeFailed, "materialize storage state", "context ended")
	}

	// Decrypt first. A tampered or unopenable envelope must fail before any
	// directory exists, so there is nothing to clean up on the common failure.
	plaintext, err := Open(ctx, protector, sealed)
	if err != nil {
		return nil, err
	}
	defer zeroBytes(plaintext)

	if err := os.MkdirAll(opts.SessionRoot, 0700); err != nil {
		return nil, NewFailureReason(FailureMaterializeFailed, "materialize storage state", "create session root")
	}
	// MkdirAll leaves a pre-existing root's permissions alone, so tighten it
	// explicitly: an inherited world-readable root would defeat the 0600 file.
	if err := os.Chmod(opts.SessionRoot, 0700); err != nil {
		return nil, NewFailureReason(FailureMaterializeFailed, "materialize storage state", "restrict session root")
	}

	dir, err := os.MkdirTemp(opts.SessionRoot, materializationPrefix)
	if err != nil {
		return nil, NewFailureReason(FailureMaterializeFailed, "materialize storage state", "create session directory")
	}
	if err := os.Chmod(dir, 0700); err != nil {
		_ = os.RemoveAll(dir)
		return nil, NewFailureReason(FailureMaterializeFailed, "materialize storage state", "restrict session directory")
	}

	path := filepath.Join(dir, storageStateFilename)
	// O_EXCL: the file must not already exist. If it does, something else owns
	// this path and writing into it would be writing a credential somewhere
	// unknown.
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		_ = os.RemoveAll(dir)
		return nil, NewFailureReason(FailureMaterializeFailed, "materialize storage state", "create state file")
	}
	if _, err := file.Write(plaintext); err != nil {
		_ = file.Close()
		_ = os.RemoveAll(dir)
		return nil, NewFailureReason(FailureMaterializeFailed, "materialize storage state", "write state file")
	}
	if err := file.Close(); err != nil {
		_ = os.RemoveAll(dir)
		return nil, NewFailureReason(FailureMaterializeFailed, "materialize storage state", "close state file")
	}

	return &Materialization{
		dir:         dir,
		path:        path,
		runID:       opts.RunID,
		profileHash: opts.ProfileHash,
		removeAll:   os.RemoveAll,
	}, nil
}

// Destroy removes the materialized file and its directory. It is idempotent:
// destroying an already-destroyed handle succeeds and does nothing.
//
// A removal failure returns FailureCleanupFailed and leaves the handle
// undestroyed, so a later attempt retries rather than silently abandoning
// plaintext on disk.
func (m *Materialization) Destroy() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.destroyed {
		return nil
	}
	remove := m.removeAll
	if remove == nil {
		remove = os.RemoveAll
	}
	if err := remove(m.dir); err != nil {
		// The cause can name paths and device state; keep it out of the public
		// error and let the caller log a sanitized diagnostic.
		return NewFailureReason(FailureCleanupFailed, "destroy materialization", "remove failed")
	}
	m.destroyed = true
	return nil
}

// RunOutcome separates the primary failure from the cleanup failure.
//
// They are two fields rather than one wrapped error on purpose: a cleanup
// failure must never mask why the run actually failed, and collapsing them
// into a single error makes that masking the default.
type RunOutcome struct {
	// Err is whatever the guarded function returned, untouched.
	Err error
	// CleanupErr is FailureCleanupFailed when destruction failed, else nil.
	CleanupErr error
}

// Failed reports whether either half failed.
func (o RunOutcome) Failed() bool { return o.Err != nil || o.CleanupErr != nil }

// Primary returns the failure that should drive the run's result: the run's own
// error when there is one, and the cleanup failure only when there is not.
func (o RunOutcome) Primary() error {
	if o.Err != nil {
		return o.Err
	}
	return o.CleanupErr
}

// Use runs fn and then destroys the materialization on EVERY exit path —
// success, error, context cancellation, deadline, and panic.
//
// Destruction happens in a defer, so a panic in fn still removes the plaintext
// before the panic continues to unwind.
func (m *Materialization) Use(ctx context.Context, fn func(context.Context) error) (outcome RunOutcome) {
	defer func() {
		if err := m.Destroy(); err != nil {
			outcome.CleanupErr = err
		}
	}()
	if fn == nil {
		return outcome
	}
	outcome.Err = fn(ctx)
	return outcome
}

// OrphanAuditOptions configures the startup sweep.
type OrphanAuditOptions struct {
	Now func() time.Time

	// MinAge, when positive, only removes materializations older than it. The
	// zero value removes every materialization found, which is the correct
	// startup semantics: at startup none of ours can legitimately still exist,
	// so anything present survived a crash.
	MinAge time.Duration

	// Sink receives one AuditEvent per orphan removed or failed. It reuses the
	// profile audit vocabulary (DecisionOrphanRemoved, DecisionCleanupFailed)
	// rather than introducing a second one.
	Sink AuditSink

	// Run correlates the sweep with a run when one exists. A startup sweep has
	// no run and leaves this zero.
	Run RunIdentity

	// Where is a free-form operator-safe label for the call site, e.g. "startup".
	Where string

	removeAll func(string) error
}

// OrphanAuditResult is the summary of one sweep. Every field is safe to
// serialize: it counts directories and never reads their contents.
type OrphanAuditResult struct {
	Op              string          `json:"op"`
	Root            string          `json:"root"`
	Where           string          `json:"where,omitempty"`
	Scanned         int             `json:"scanned"`
	Removed         int             `json:"removed"`
	Failed          int             `json:"failed"`
	FailureCategory FailureCategory `json:"failure_category,omitempty"`
	AuditedAt       time.Time       `json:"audited_at"`
}

// AuditOrphanMaterializations removes materializations that survived a crash
// and records a structured audit event per orphan.
//
// A per-entry removal failure is counted, not fatal: one undeletable directory
// must not stop the sweep from clearing the rest. A missing root is not a
// failure — it just means nothing was ever materialized. A sink error is also
// not fatal, per the AuditSink contract: recording must never fail the
// operation being audited.
func AuditOrphanMaterializations(ctx context.Context, root string, opts OrphanAuditOptions) (OrphanAuditResult, error) {
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	remove := opts.removeAll
	if remove == nil {
		remove = os.RemoveAll
	}

	result := OrphanAuditResult{
		Op:        OpOrphanAudit,
		Root:      root,
		Where:     opts.Where,
		AuditedAt: now(),
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return result, nil
		}
		return result, NewFailureReason(FailureCleanupFailed, "audit orphan materializations", "read session root")
	}

	for _, entry := range entries {
		result.Scanned++
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), materializationPrefix) {
			continue
		}
		if opts.MinAge > 0 {
			info, err := entry.Info()
			if err != nil {
				result.Failed++
				recordOrphan(ctx, opts, now(), DecisionCleanupFailed, "stat failed")
				continue
			}
			if now().Sub(info.ModTime()) < opts.MinAge {
				continue
			}
		}
		if err := remove(filepath.Join(root, entry.Name())); err != nil {
			result.Failed++
			recordOrphan(ctx, opts, now(), DecisionCleanupFailed, "remove failed")
			continue
		}
		result.Removed++
		recordOrphan(ctx, opts, now(), DecisionOrphanRemoved, opts.Where)
	}

	if result.Failed > 0 {
		result.FailureCategory = FailureCleanupFailed
	}
	return result, nil
}

// recordOrphan writes one sweep event. Alias, version, and profile hash are
// deliberately absent: the materialization path encodes no profile identity, so
// the sweep genuinely does not know which profile an orphan came from and must
// not invent one.
func recordOrphan(ctx context.Context, opts OrphanAuditOptions, at time.Time, decision AuditDecision, reason string) {
	if opts.Sink == nil {
		return
	}
	event := AuditEvent{
		Op:       OpOrphanAudit,
		Run:      opts.Run,
		Decision: decision,
		Reason:   reason,
		At:       at.UTC(),
	}
	if decision == DecisionCleanupFailed {
		event.FailureCategory = FailureCleanupFailed
	}
	// A sink error must not fail the sweep; the plaintext is already gone.
	_ = opts.Sink.Record(ctx, event)
}

// zeroBytes overwrites a decrypted buffer once it is no longer needed. It is
// best-effort hygiene, not a guarantee: the Go runtime may already have copied
// the bytes during a stack or heap move.
func zeroBytes(buffer []byte) {
	for i := range buffer {
		buffer[i] = 0
	}
}
