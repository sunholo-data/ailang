package auth

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// assertOwnerOnlyFile asserts a regular file exists and, on POSIX platforms,
// that its permission bits are exactly 0600.
//
// Windows has no POSIX mode bits: Go reports 0666/0444 for every file there
// regardless of the ACL, so the bit assertion would be meaningless rather than
// merely lenient. Existence is still asserted on every platform.
func assertOwnerOnlyFile(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", filepath.ToSlash(path), err)
	}
	if info.IsDir() {
		t.Fatalf("%s is a directory, want a file", filepath.ToSlash(path))
	}
	if runtime.GOOS == "windows" {
		t.Log("skipping POSIX mode-bit assertion on Windows (no POSIX permission bits)")
		return
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Fatalf("%s mode = %#o, want 0600", filepath.ToSlash(path), perm)
	}
}

// assertOwnerOnlyDir is the directory counterpart of assertOwnerOnlyFile.
func assertOwnerOnlyDir(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", filepath.ToSlash(path), err)
	}
	if !info.IsDir() {
		t.Fatalf("%s is not a directory", filepath.ToSlash(path))
	}
	if runtime.GOOS == "windows" {
		t.Log("skipping POSIX mode-bit assertion on Windows (no POSIX permission bits)")
		return
	}
	if perm := info.Mode().Perm(); perm != 0700 {
		t.Fatalf("%s mode = %#o, want 0700", filepath.ToSlash(path), perm)
	}
}

const canonicalState = `{"cookies":[{"name":"sid","value":"CANONICAL-SESSION-VALUE"}]}`

func sealedCanonical(t *testing.T) (*StaticKeyProtector, SealedEnvelope) {
	t.Helper()
	protector := testProtector(t)
	sealed, err := Seal(context.Background(), protector, []byte(canonicalState))
	if err != nil {
		t.Fatal(err)
	}
	return protector, sealed
}

func materializeForTest(t *testing.T, root string, opts MaterializeOptions) *Materialization {
	t.Helper()
	protector, sealed := sealedCanonical(t)
	opts.SessionRoot = root
	handle, err := MaterializeStorageState(context.Background(), protector, sealed, opts)
	if err != nil {
		t.Fatal(err)
	}
	return handle
}

// Criterion 2: the materialized file is 0600 inside a 0700 session-owned
// directory. Asserted on the mode bits, not just the path.
func TestMaterializeCreatesOwnerOnlyFileInOwnerOnlyDirectory(t *testing.T) {
	root := filepath.Join(t.TempDir(), "auth-state")
	handle := materializeForTest(t, root, MaterializeOptions{RunID: "run-1"})
	defer func() { _ = handle.Destroy() }()

	assertOwnerOnlyFile(t, handle.Path())
	assertOwnerOnlyDir(t, handle.Dir())
	assertOwnerOnlyDir(t, root)

	body, err := os.ReadFile(handle.Path())
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != canonicalState {
		t.Fatalf("materialized content = %q, want canonical state", body)
	}
	if filepath.Dir(handle.Path()) != handle.Dir() {
		t.Fatalf("file %s is not inside owned dir %s", filepath.ToSlash(handle.Path()), filepath.ToSlash(handle.Dir()))
	}
}

// The materialized path reaches child argv, so it must not embed the canonical
// object reference (profile hash) or anything derived from the material.
func TestMaterializedPathDoesNotEmbedCanonicalReference(t *testing.T) {
	root := filepath.Join(t.TempDir(), "auth-state")
	const profileHash = "sha256:deadbeefdeadbeef"
	handle := materializeForTest(t, root, MaterializeOptions{RunID: "run-1", ProfileHash: profileHash})
	defer func() { _ = handle.Destroy() }()

	path := filepath.ToSlash(handle.Path())
	if strings.Contains(path, "deadbeef") || strings.Contains(path, profileHash) {
		t.Fatalf("materialized path leaks the canonical reference: %s", path)
	}
	if strings.Contains(path, "CANONICAL-SESSION-VALUE") {
		t.Fatalf("materialized path leaks material: %s", path)
	}
}

func TestMaterializeIsUniquePerRun(t *testing.T) {
	root := filepath.Join(t.TempDir(), "auth-state")
	first := materializeForTest(t, root, MaterializeOptions{RunID: "run-1"})
	defer func() { _ = first.Destroy() }()
	second := materializeForTest(t, root, MaterializeOptions{RunID: "run-1"})
	defer func() { _ = second.Destroy() }()

	if first.Dir() == second.Dir() || first.Path() == second.Path() {
		t.Fatalf("two materializations share storage: %s / %s", first.Path(), second.Path())
	}
}

// Criterion 1 applied at the materialization boundary: tampered canonical
// material must fail closed and must leave nothing on disk.
func TestMaterializeTamperedEnvelopeFailsClosedAndLeavesNoPlaintext(t *testing.T) {
	protector, sealed := sealedCanonical(t)
	raw := sealed.Bytes()
	raw[len(raw)-1] ^= 0xff
	tampered, err := ParseSealedEnvelope(raw)
	if err != nil {
		t.Fatalf("tampered envelope should still parse for this test: %v", err)
	}

	root := filepath.Join(t.TempDir(), "auth-state")
	handle, err := MaterializeStorageState(context.Background(), protector, tampered, MaterializeOptions{SessionRoot: root, RunID: "run-1"})
	if !IsFailure(err, FailureMaterializeFailed) {
		t.Fatalf("error = %v, want %s", err, FailureMaterializeFailed)
	}
	if handle != nil {
		t.Fatalf("failed materialization returned a handle: %#v", handle)
	}
	assertNoPlaintextUnder(t, root)
}

func assertNoPlaintextUnder(t *testing.T, root string) {
	t.Helper()
	if _, err := os.Stat(root); os.IsNotExist(err) {
		return
	}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if strings.Contains(string(body), "CANONICAL-SESSION-VALUE") {
			return fmt.Errorf("plaintext survives at %s", filepath.ToSlash(path))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// Criterion 4: destruction is idempotent.
func TestDestroyIsIdempotent(t *testing.T) {
	root := filepath.Join(t.TempDir(), "auth-state")
	handle := materializeForTest(t, root, MaterializeOptions{RunID: "run-1"})
	dir := handle.Dir()

	for attempt := 0; attempt < 3; attempt++ {
		if err := handle.Destroy(); err != nil {
			t.Fatalf("destroy attempt %d: %v", attempt, err)
		}
		if !handle.Destroyed() {
			t.Fatalf("attempt %d: handle does not report destroyed", attempt)
		}
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("materialization directory survives destroy: %v", err)
	}
}

// Criterion 4: destruction runs on success, error, cancellation, and timeout.
func TestUseDestroysOnEveryExitPath(t *testing.T) {
	primary := errors.New("executor blew up")

	cases := []struct {
		name    string
		ctx     func(t *testing.T) (context.Context, context.CancelFunc)
		fn      func(ctx context.Context) error
		wantErr func(t *testing.T, err error)
	}{
		{
			name: "success",
			ctx: func(*testing.T) (context.Context, context.CancelFunc) {
				return context.WithCancel(context.Background())
			},
			fn: func(context.Context) error { return nil },
			wantErr: func(t *testing.T, err error) {
				if err != nil {
					t.Fatalf("primary = %v, want nil", err)
				}
			},
		},
		{
			name: "error",
			ctx: func(*testing.T) (context.Context, context.CancelFunc) {
				return context.WithCancel(context.Background())
			},
			fn: func(context.Context) error { return primary },
			wantErr: func(t *testing.T, err error) {
				if !errors.Is(err, primary) {
					t.Fatalf("primary = %v, want %v", err, primary)
				}
			},
		},
		{
			name: "cancellation",
			ctx: func(*testing.T) (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx, cancel
			},
			fn: func(ctx context.Context) error { <-ctx.Done(); return ctx.Err() },
			wantErr: func(t *testing.T, err error) {
				if !errors.Is(err, context.Canceled) {
					t.Fatalf("primary = %v, want context.Canceled", err)
				}
			},
		},
		{
			name: "timeout",
			ctx: func(*testing.T) (context.Context, context.CancelFunc) {
				return context.WithTimeout(context.Background(), 10*time.Millisecond)
			},
			fn: func(ctx context.Context) error { <-ctx.Done(); return ctx.Err() },
			wantErr: func(t *testing.T, err error) {
				if !errors.Is(err, context.DeadlineExceeded) {
					t.Fatalf("primary = %v, want context.DeadlineExceeded", err)
				}
			},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			root := filepath.Join(t.TempDir(), "auth-state")
			handle := materializeForTest(t, root, MaterializeOptions{RunID: "run-1"})
			dir := handle.Dir()

			ctx, cancel := testCase.ctx(t)
			defer cancel()

			outcome := handle.Use(ctx, testCase.fn)
			testCase.wantErr(t, outcome.Err)
			if outcome.CleanupErr != nil {
				t.Fatalf("cleanup error = %v, want nil", outcome.CleanupErr)
			}
			if !handle.Destroyed() {
				t.Fatal("handle not destroyed after Use")
			}
			if _, err := os.Stat(dir); !os.IsNotExist(err) {
				t.Fatalf("materialization survives %s path: %v", testCase.name, err)
			}
		})
	}
}

func TestUsePanicStillDestroys(t *testing.T) {
	root := filepath.Join(t.TempDir(), "auth-state")
	handle := materializeForTest(t, root, MaterializeOptions{RunID: "run-1"})
	dir := handle.Dir()

	func() {
		defer func() {
			if recovered := recover(); recovered == nil {
				t.Error("panic did not propagate")
			}
		}()
		_ = handle.Use(context.Background(), func(context.Context) error { panic("boom") })
	}()

	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("materialization survives a panic: %v", err)
	}
}

// Criterion 4: a cleanup failure is reported as FailureCleanupFailed and does
// NOT mask the primary failure.
func TestCleanupFailureDoesNotMaskPrimaryFailure(t *testing.T) {
	primary := errors.New("the run itself failed")
	root := filepath.Join(t.TempDir(), "auth-state")
	handle := materializeForTest(t, root, MaterializeOptions{RunID: "run-1"})
	handle.removeAll = func(string) error { return errors.New("device busy") }

	outcome := handle.Use(context.Background(), func(context.Context) error { return primary })

	if !errors.Is(outcome.Err, primary) {
		t.Fatalf("primary = %v, want %v; cleanup masked it", outcome.Err, primary)
	}
	if !IsFailure(outcome.CleanupErr, FailureCleanupFailed) {
		t.Fatalf("cleanup error = %v, want %s", outcome.CleanupErr, FailureCleanupFailed)
	}
	// The underlying cause must not leak into the public cleanup error.
	if strings.Contains(outcome.CleanupErr.Error(), "device busy") {
		t.Fatalf("cleanup error leaked its cause: %v", outcome.CleanupErr)
	}
	// And with no primary failure, the cleanup failure is the only one reported.
	handle.destroyed = false
	outcome = handle.Use(context.Background(), func(context.Context) error { return nil })
	if outcome.Err != nil {
		t.Fatalf("primary = %v, want nil", outcome.Err)
	}
	if !IsFailure(outcome.CleanupErr, FailureCleanupFailed) {
		t.Fatalf("cleanup error = %v, want %s", outcome.CleanupErr, FailureCleanupFailed)
	}
}

func TestDestroyFailureReportsCleanupCategory(t *testing.T) {
	root := filepath.Join(t.TempDir(), "auth-state")
	handle := materializeForTest(t, root, MaterializeOptions{RunID: "run-1"})
	handle.removeAll = func(string) error { return errors.New("device busy") }

	err := handle.Destroy()
	if !IsFailure(err, FailureCleanupFailed) {
		t.Fatalf("error = %v, want %s", err, FailureCleanupFailed)
	}
	if handle.Destroyed() {
		t.Fatal("handle reports destroyed after a failed destroy")
	}
}

// Criterion 5: two-run isolation. Run 1 mutates its materialized state; run 2
// materializes from canonical and cannot observe the mutation.
func TestTwoRunIsolationMutationIsNotObservable(t *testing.T) {
	protector, sealed := sealedCanonical(t)
	root := filepath.Join(t.TempDir(), "auth-state")

	runOne, err := MaterializeStorageState(context.Background(), protector, sealed, MaterializeOptions{SessionRoot: root, RunID: "run-1"})
	if err != nil {
		t.Fatal(err)
	}
	const mutation = `{"cookies":[{"name":"sid","value":"RUN-1-MUTATED-VALUE"}]}`
	if err := os.WriteFile(runOne.Path(), []byte(mutation), 0600); err != nil {
		t.Fatal(err)
	}
	runOnePath := runOne.Path()
	if err := runOne.Destroy(); err != nil {
		t.Fatal(err)
	}

	runTwo, err := MaterializeStorageState(context.Background(), protector, sealed, MaterializeOptions{SessionRoot: root, RunID: "run-2"})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = runTwo.Destroy() }()

	if runTwo.Path() == runOnePath {
		t.Fatal("run 2 reused run 1's path")
	}
	body, err := os.ReadFile(runTwo.Path())
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != canonicalState {
		t.Fatalf("run 2 state = %q, want canonical state", body)
	}
	if strings.Contains(string(body), "RUN-1-MUTATED-VALUE") {
		t.Fatal("run 2 observed run 1's mutation")
	}
}

// Criterion 6: the startup orphan audit finds and removes stale
// materializations and records a structured audit event.
func TestOrphanAuditRemovesStaleMaterializationsAndRecordsEvent(t *testing.T) {
	root := filepath.Join(t.TempDir(), "auth-state")

	// Two materializations that "survived a crash": created but never destroyed.
	orphanOne := materializeForTest(t, root, MaterializeOptions{RunID: "crashed-1"})
	orphanTwo := materializeForTest(t, root, MaterializeOptions{RunID: "crashed-2"})

	// An unrelated neighbour that the audit must not touch.
	neighbour := filepath.Join(root, "not-a-materialization")
	if err := os.MkdirAll(neighbour, 0700); err != nil {
		t.Fatal(err)
	}

	sink := NewMemoryAuditSink()
	result, err := AuditOrphanMaterializations(context.Background(), root, OrphanAuditOptions{
		Now:   func() time.Time { return time.Unix(1000, 0) },
		Sink:  sink,
		Where: "startup",
	})
	if err != nil {
		t.Fatal(err)
	}

	if result.Op != OpOrphanAudit {
		t.Fatalf("op = %q, want %q", result.Op, OpOrphanAudit)
	}
	if result.Removed != 2 {
		t.Fatalf("removed = %d, want 2", result.Removed)
	}
	if result.Failed != 0 {
		t.Fatalf("failed = %d, want 0", result.Failed)
	}
	if result.Scanned < 3 {
		t.Fatalf("scanned = %d, want at least 3 entries", result.Scanned)
	}
	if !result.AuditedAt.Equal(time.Unix(1000, 0)) {
		t.Fatalf("audited_at = %v, want injected clock", result.AuditedAt)
	}
	if result.Where != "startup" {
		t.Fatalf("where = %q, want startup", result.Where)
	}
	events := sink.Events()
	if len(events) != 2 {
		t.Fatalf("recorded %d audit events, want 2", len(events))
	}
	for i, event := range events {
		if event.Op != OpOrphanAudit {
			t.Fatalf("event %d op = %q, want %q", i, event.Op, OpOrphanAudit)
		}
		if event.Decision != DecisionOrphanRemoved {
			t.Fatalf("event %d decision = %q, want %q", i, event.Decision, DecisionOrphanRemoved)
		}
		if event.FailureCategory != "" {
			t.Fatalf("event %d carries a failure category on a clean removal: %q", i, event.FailureCategory)
		}
		if !event.At.Equal(time.Unix(1000, 0)) {
			t.Fatalf("event %d at = %v, want injected clock", i, event.At)
		}
		// The sweep cannot know which profile an orphan held, and must not guess.
		if event.Alias != "" || event.Version != "" || event.ProfileHash != "" {
			t.Fatalf("event %d invented a profile identity: %#v", i, event)
		}
	}

	for _, orphan := range []*Materialization{orphanOne, orphanTwo} {
		if _, err := os.Stat(orphan.Dir()); !os.IsNotExist(err) {
			t.Fatalf("orphan %s survives the audit: %v", filepath.ToSlash(orphan.Dir()), err)
		}
	}
	if _, err := os.Stat(neighbour); err != nil {
		t.Fatalf("audit removed an unrelated directory: %v", err)
	}
	assertNoPlaintextUnder(t, root)
}

func TestOrphanAuditHonoursMinAge(t *testing.T) {
	root := filepath.Join(t.TempDir(), "auth-state")
	live := materializeForTest(t, root, MaterializeOptions{RunID: "live"})
	defer func() { _ = live.Destroy() }()

	result, err := AuditOrphanMaterializations(context.Background(), root, OrphanAuditOptions{
		Now:    time.Now,
		MinAge: time.Hour,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Removed != 0 {
		t.Fatalf("removed = %d, want 0 (younger than MinAge)", result.Removed)
	}
	if _, err := os.Stat(live.Dir()); err != nil {
		t.Fatalf("audit removed a fresh materialization: %v", err)
	}
}

func TestOrphanAuditOnMissingRootIsNotAnError(t *testing.T) {
	result, err := AuditOrphanMaterializations(context.Background(), filepath.Join(t.TempDir(), "never-created"), OrphanAuditOptions{})
	if err != nil {
		t.Fatalf("audit on a missing root: %v", err)
	}
	if result.Scanned != 0 || result.Removed != 0 {
		t.Fatalf("result = %#v, want an empty audit", result)
	}
}

func TestOrphanAuditReportsFailuresWithoutAborting(t *testing.T) {
	root := filepath.Join(t.TempDir(), "auth-state")
	first := materializeForTest(t, root, MaterializeOptions{RunID: "crashed-1"})
	second := materializeForTest(t, root, MaterializeOptions{RunID: "crashed-2"})
	defer func() { _ = first.Destroy() }()
	defer func() { _ = second.Destroy() }()

	sink := NewMemoryAuditSink()
	options := OrphanAuditOptions{Now: func() time.Time { return time.Unix(1000, 0) }, Sink: sink}
	options.removeAll = func(string) error { return errors.New("device busy") }

	result, err := AuditOrphanMaterializations(context.Background(), root, options)
	if err != nil {
		t.Fatalf("a per-entry removal failure must not abort the audit: %v", err)
	}
	if result.Failed != 2 {
		t.Fatalf("failed = %d, want 2", result.Failed)
	}
	if result.Removed != 0 {
		t.Fatalf("removed = %d, want 0", result.Removed)
	}
	if result.FailureCategory != FailureCleanupFailed {
		t.Fatalf("failure category = %q, want %s", result.FailureCategory, FailureCleanupFailed)
	}
	events := sink.Events()
	if len(events) != 2 {
		t.Fatalf("recorded %d audit events, want 2", len(events))
	}
	for i, event := range events {
		if event.Decision != DecisionCleanupFailed {
			t.Fatalf("event %d decision = %q, want %q", i, event.Decision, DecisionCleanupFailed)
		}
		if event.FailureCategory != FailureCleanupFailed {
			t.Fatalf("event %d failure category = %q, want %s", i, event.FailureCategory, FailureCleanupFailed)
		}
	}
}

// A sink that errors must not fail the sweep: the plaintext is already gone,
// and turning a logging problem into a cleanup failure would misreport it.
func TestOrphanAuditSurvivesASinkFailure(t *testing.T) {
	root := filepath.Join(t.TempDir(), "auth-state")
	orphan := materializeForTest(t, root, MaterializeOptions{RunID: "crashed-1"})

	result, err := AuditOrphanMaterializations(context.Background(), root, OrphanAuditOptions{
		Sink: failingSink{},
	})
	if err != nil {
		t.Fatalf("sink failure aborted the sweep: %v", err)
	}
	if result.Removed != 1 || result.Failed != 0 {
		t.Fatalf("result = %#v, want 1 removed and 0 failed", result)
	}
	if _, err := os.Stat(orphan.Dir()); !os.IsNotExist(err) {
		t.Fatalf("orphan survives: %v", err)
	}
}

type failingSink struct{}

func (failingSink) Record(context.Context, AuditEvent) error {
	return errors.New("sink unavailable")
}

func TestMaterializeRequiresSessionRoot(t *testing.T) {
	protector, sealed := sealedCanonical(t)
	_, err := MaterializeStorageState(context.Background(), protector, sealed, MaterializeOptions{RunID: "run-1"})
	if !IsFailure(err, FailureMaterializeFailed) {
		t.Fatalf("error = %v, want %s", err, FailureMaterializeFailed)
	}
}

func TestMaterializeRejectsCancelledContext(t *testing.T) {
	protector, sealed := sealedCanonical(t)
	root := filepath.Join(t.TempDir(), "auth-state")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	handle, err := MaterializeStorageState(ctx, protector, sealed, MaterializeOptions{SessionRoot: root, RunID: "run-1"})
	if !IsFailure(err, FailureMaterializeFailed) {
		t.Fatalf("error = %v, want %s", err, FailureMaterializeFailed)
	}
	if handle != nil {
		t.Fatal("cancelled materialization returned a handle")
	}
	assertNoPlaintextUnder(t, root)
}

func TestMaterializationRedactsInPresentations(t *testing.T) {
	root := filepath.Join(t.TempDir(), "auth-state")
	handle := materializeForTest(t, root, MaterializeOptions{RunID: "run-1"})
	defer func() { _ = handle.Destroy() }()

	for i, presentation := range []string{
		handle.String(),
		handle.GoString(),
		format("%v", handle),
		format("%s", handle),
		format("%#v", handle),
	} {
		if strings.Contains(presentation, "CANONICAL-SESSION-VALUE") {
			t.Fatalf("presentation %d leaked material: %s", i, presentation)
		}
	}
}
