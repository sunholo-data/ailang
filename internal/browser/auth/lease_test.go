package auth

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

func readonlyProfile(alias, version string, maxConcurrent int) SafeProfile {
	policy := testPolicy()
	policy.MaxConcurrent = maxConcurrent
	return SafeProfile{
		Alias:       alias,
		Version:     version,
		ProfileHash: "sha256:deadbeefdeadbeef",
		Policy:      policy,
	}
}

func run(id string) RunIdentity {
	return RunIdentity{RunID: id, Principal: "eval-harness"}
}

// fixedClock lets a test drive lease expiry without sleeping.
type fixedClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *fixedClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fixedClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

func newTestManager(t *testing.T) (*LeaseManager, *fixedClock) {
	t.Helper()
	clock := &fixedClock{now: time.Date(2026, 8, 25, 12, 0, 0, 0, time.UTC)}
	manager := NewLeaseManager(30 * time.Minute)
	manager.SetClock(clock.Now)
	return manager, clock
}

func TestRefreshLeaseIsExclusive(t *testing.T) {
	manager, _ := newTestManager(t)
	ctx := context.Background()
	profile := readonlyProfile("crm", "v1", 4)

	first, err := manager.Acquire(ctx, profile, run("run-1"), LeaseRefresh)
	if err != nil {
		t.Fatalf("first refresh acquire: %v", err)
	}
	if first.Mode != LeaseRefresh {
		t.Fatalf("lease mode = %q, want refresh", first.Mode)
	}

	// A second refresh must lose, and so must a read — an exclusive writer means
	// exclusive against every mode, not just against other writers.
	if _, err := manager.Acquire(ctx, profile, run("run-2"), LeaseRefresh); !IsFailure(err, FailureLeaseConflict) {
		t.Fatalf("second refresh returned %v, want %s", err, FailureLeaseConflict)
	}
	if _, err := manager.Acquire(ctx, profile, run("run-3"), LeaseRead); !IsFailure(err, FailureLeaseConflict) {
		t.Fatalf("read under an active refresh returned %v, want %s", err, FailureLeaseConflict)
	}

	if err := manager.Release(ctx, first); err != nil {
		t.Fatalf("release: %v", err)
	}
	if _, err := manager.Acquire(ctx, profile, run("run-4"), LeaseRefresh); err != nil {
		t.Fatalf("refresh after release: %v", err)
	}
}

func TestReadConcurrencyIsBoundedByPolicy(t *testing.T) {
	manager, _ := newTestManager(t)
	ctx := context.Background()
	profile := readonlyProfile("crm", "v1", 2)

	first, err := manager.Acquire(ctx, profile, run("run-1"), LeaseRead)
	if err != nil {
		t.Fatalf("first read: %v", err)
	}
	if _, err := manager.Acquire(ctx, profile, run("run-2"), LeaseRead); err != nil {
		t.Fatalf("second read within MaxConcurrent=2: %v", err)
	}
	if _, err := manager.Acquire(ctx, profile, run("run-3"), LeaseRead); !IsFailure(err, FailureLeaseConflict) {
		t.Fatalf("third read past MaxConcurrent returned %v, want %s", err, FailureLeaseConflict)
	}

	// Releasing one must free exactly one slot.
	if err := manager.Release(ctx, first); err != nil {
		t.Fatalf("release: %v", err)
	}
	if _, err := manager.Acquire(ctx, profile, run("run-4"), LeaseRead); err != nil {
		t.Fatalf("read after release: %v", err)
	}
}

// MaxConcurrent=1 in read mode is the setting for a site that invalidates
// simultaneous logins. It must be honored, not treated as "reads are free".
func TestMaxConcurrentOneSerializesEvenReads(t *testing.T) {
	manager, _ := newTestManager(t)
	ctx := context.Background()
	profile := readonlyProfile("crm", "v1", 1)

	if _, err := manager.Acquire(ctx, profile, run("run-1"), LeaseRead); err != nil {
		t.Fatalf("first read: %v", err)
	}
	if _, err := manager.Acquire(ctx, profile, run("run-2"), LeaseRead); !IsFailure(err, FailureLeaseConflict) {
		t.Fatalf("second read at MaxConcurrent=1 returned %v, want %s", err, FailureLeaseConflict)
	}
}

// A crashed run must not wedge a profile forever: an expired lease is swept
// before the conflict decision, not left to block indefinitely.
func TestExpiredLeaseIsSweptBeforeConflictIsDecided(t *testing.T) {
	manager, clock := newTestManager(t)
	ctx := context.Background()
	profile := readonlyProfile("crm", "v1", 1)

	if _, err := manager.Acquire(ctx, profile, run("crashed"), LeaseRefresh); err != nil {
		t.Fatalf("acquire: %v", err)
	}
	if _, err := manager.Acquire(ctx, profile, run("blocked"), LeaseRead); !IsFailure(err, FailureLeaseConflict) {
		t.Fatalf("expected a conflict while the lease is live, got %v", err)
	}

	clock.Advance(31 * time.Minute)

	recovered, err := manager.Acquire(ctx, profile, run("recovered"), LeaseRefresh)
	if err != nil {
		t.Fatalf("acquire after TTL expiry: %v", err)
	}
	if recovered.Run.RunID != "recovered" {
		t.Fatalf("recovered lease belongs to %q", recovered.Run.RunID)
	}
	if active := manager.Active(profile.Ref()); len(active) != 1 {
		t.Fatalf("Active() = %d leases, want 1 after the sweep", len(active))
	}
}

func TestLeaseCarriesSafeIdentityOnly(t *testing.T) {
	manager, clock := newTestManager(t)
	profile := readonlyProfile("crm", "v7", 1)

	lease, err := manager.Acquire(context.Background(), profile, run("run-1"), LeaseRead)
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}
	if lease.SafeID == "" {
		t.Fatalf("lease has no SafeID")
	}
	if lease.Alias != "crm" || lease.Version != "v7" {
		t.Fatalf("lease identity = %s@%s, want crm@v7", lease.Alias, lease.Version)
	}
	if lease.ProfileHash != profile.ProfileHash {
		t.Fatalf("lease hash = %q, want %q", lease.ProfileHash, profile.ProfileHash)
	}
	if !lease.AcquiredAt.Equal(clock.Now()) {
		t.Fatalf("AcquiredAt = %s, want the injected clock time", lease.AcquiredAt)
	}
	if !lease.ExpiresAt.Equal(clock.Now().Add(30 * time.Minute)) {
		t.Fatalf("ExpiresAt = %s, want acquire + TTL", lease.ExpiresAt)
	}
}

func TestSafeIDsAreDistinct(t *testing.T) {
	manager, _ := newTestManager(t)
	ctx := context.Background()
	profile := readonlyProfile("crm", "v1", 8)

	seen := make(map[string]bool)
	for i := range 8 {
		lease, err := manager.Acquire(ctx, profile, run(fmt.Sprintf("run-%d", i)), LeaseRead)
		if err != nil {
			t.Fatalf("acquire %d: %v", i, err)
		}
		if seen[lease.SafeID] {
			t.Fatalf("duplicate lease SafeID %q", lease.SafeID)
		}
		seen[lease.SafeID] = true
	}
}

func TestReleaseIsIdempotentAndToleratesUnknownLeases(t *testing.T) {
	manager, _ := newTestManager(t)
	ctx := context.Background()
	profile := readonlyProfile("crm", "v1", 1)

	lease, err := manager.Acquire(ctx, profile, run("run-1"), LeaseRead)
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}
	if err := manager.Release(ctx, lease); err != nil {
		t.Fatalf("first release: %v", err)
	}
	// Release runs on every controller exit path, including paths that already
	// released. A second release must not become a spurious cleanup failure.
	if err := manager.Release(ctx, lease); err != nil {
		t.Fatalf("second release: %v", err)
	}
	if err := manager.Release(ctx, ProfileLease{SafeID: "never-issued", Alias: "crm", Version: "v1"}); err != nil {
		t.Fatalf("release of an unknown lease: %v", err)
	}
}

// Different versions of the same alias are different leases: pinning v1 must not
// be blocked by a refresh publishing v2.
func TestLeasesAreScopedToAliasAndVersion(t *testing.T) {
	manager, _ := newTestManager(t)
	ctx := context.Background()

	if _, err := manager.Acquire(ctx, readonlyProfile("crm", "v1", 1), run("run-1"), LeaseRefresh); err != nil {
		t.Fatalf("acquire v1: %v", err)
	}
	if _, err := manager.Acquire(ctx, readonlyProfile("crm", "v2", 1), run("run-2"), LeaseRefresh); err != nil {
		t.Fatalf("acquire v2 was blocked by a v1 lease: %v", err)
	}
	if _, err := manager.Acquire(ctx, readonlyProfile("other", "v1", 1), run("run-3"), LeaseRefresh); err != nil {
		t.Fatalf("acquire other@v1 was blocked: %v", err)
	}
}

func TestAcquireRejectsInvalidMode(t *testing.T) {
	manager, _ := newTestManager(t)
	if _, err := manager.Acquire(context.Background(), readonlyProfile("crm", "v1", 1), run("run-1"), LeaseMode("write")); err == nil {
		t.Fatalf("Acquire accepted an unknown lease mode")
	}
}

// The whole point of compare-and-set is that N racing workers see exactly the
// policy limit granted, never N.
func TestConcurrentAcquireGrantsExactlyMaxConcurrent(t *testing.T) {
	manager, _ := newTestManager(t)
	ctx := context.Background()
	profile := readonlyProfile("crm", "v1", 3)

	const workers = 32
	var granted, conflicts int
	var mu sync.Mutex
	var wg sync.WaitGroup

	for i := range workers {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, err := manager.Acquire(ctx, profile, run(fmt.Sprintf("run-%d", i)), LeaseRead)
			mu.Lock()
			defer mu.Unlock()
			switch {
			case err == nil:
				granted++
			case IsFailure(err, FailureLeaseConflict):
				conflicts++
			default:
				t.Errorf("unexpected error: %v", err)
			}
		}(i)
	}
	wg.Wait()

	if granted != 3 {
		t.Fatalf("granted %d leases, want exactly 3", granted)
	}
	if conflicts != workers-3 {
		t.Fatalf("saw %d conflicts, want %d", conflicts, workers-3)
	}
}

func TestAccountPoolAllocatesDistinctProfiles(t *testing.T) {
	manager, _ := newTestManager(t)
	ctx := context.Background()

	members := []SafeProfile{
		readonlyProfile("shop-test-1", "v1", 1),
		readonlyProfile("shop-test-2", "v1", 1),
		readonlyProfile("shop-test-3", "v1", 1),
	}
	pool := NewAccountPool("shop-test-users", members)

	seen := make(map[string]bool)
	for i := range members {
		lease, err := pool.Allocate(ctx, manager, run(fmt.Sprintf("worker-%d", i)), LeaseRead)
		if err != nil {
			t.Fatalf("allocate %d: %v", i, err)
		}
		if seen[lease.Alias] {
			t.Fatalf("pool handed out %q twice — a shared mutable account is the failure this prevents", lease.Alias)
		}
		seen[lease.Alias] = true
	}
}

func TestAccountPoolExhaustionIsDeterministic(t *testing.T) {
	manager, _ := newTestManager(t)
	ctx := context.Background()

	pool := NewAccountPool("shop-test-users", []SafeProfile{
		readonlyProfile("shop-test-1", "v1", 1),
		readonlyProfile("shop-test-2", "v1", 1),
	})

	for i := range 2 {
		if _, err := pool.Allocate(ctx, manager, run(fmt.Sprintf("worker-%d", i)), LeaseRead); err != nil {
			t.Fatalf("allocate %d: %v", i, err)
		}
	}

	_, err := pool.Allocate(ctx, manager, run("worker-overflow"), LeaseRead)
	if !IsFailure(err, FailureLeaseConflict) {
		t.Fatalf("pool exhaustion returned %v, want %s", err, FailureLeaseConflict)
	}
	// Exhaustion must never silently reuse an account already in use.
	if len(manager.Active(AuthProfileRef{Alias: "shop-test-1", Version: "v1"})) != 1 {
		t.Fatalf("pool exhaustion double-leased an account")
	}
}

func TestEmptyAccountPoolFailsClosed(t *testing.T) {
	manager, _ := newTestManager(t)
	pool := NewAccountPool("empty", nil)
	if _, err := pool.Allocate(context.Background(), manager, run("worker"), LeaseRead); err == nil {
		t.Fatalf("an empty pool allocated a lease")
	}
}
