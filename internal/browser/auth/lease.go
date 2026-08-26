package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// DefaultLeaseTTL bounds how long a profile stays held by a run that never
// released it. It is short on purpose: a crashed run must not wedge a profile,
// and a long-running refresh should renew rather than hold indefinitely.
const DefaultLeaseTTL = 30 * time.Minute

// LeaseManager grants at most one writer per profile version and bounds readers
// by the profile's own MaxConcurrent.
//
// The in-memory implementation is the reference and the test double. A durable
// backend (SQLite locally, Firestore/Redis remotely) is an explicitly deferred
// decision in the design document; whatever ships must preserve the two
// properties this type guarantees: compare-and-set acquisition and TTL sweeping.
type LeaseManager struct {
	mu      sync.Mutex
	held    map[string][]ProfileLease
	ttl     time.Duration
	nowFunc func() time.Time
	idFunc  func() (string, error)
}

func NewLeaseManager(ttl time.Duration) *LeaseManager {
	if ttl <= 0 {
		ttl = DefaultLeaseTTL
	}
	return &LeaseManager{
		held:    make(map[string][]ProfileLease),
		ttl:     ttl,
		nowFunc: time.Now,
		idFunc:  randomLeaseID,
	}
}

// SetClock injects a deterministic clock so expiry is testable without sleeping.
func (m *LeaseManager) SetClock(now func() time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nowFunc = now
}

// SetIDFunc injects lease-ID generation. Tests use it; production does not.
func (m *LeaseManager) SetIDFunc(next func() (string, error)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.idFunc = next
}

func (m *LeaseManager) now() time.Time {
	if m.nowFunc == nil {
		return time.Now()
	}
	return m.nowFunc()
}

func randomLeaseID() (string, error) {
	buffer := make([]byte, 16)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return "lease-" + hex.EncodeToString(buffer), nil
}

func leaseKey(ref AuthProfileRef) string { return ref.String() }

// sweepLocked drops expired leases. It runs before every conflict decision so a
// crashed holder is never the reason a healthy run is refused.
func (m *LeaseManager) sweepLocked(key string, now time.Time) []ProfileLease {
	existing := m.held[key]
	live := existing[:0:0]
	for _, lease := range existing {
		if !lease.Expired(now) {
			live = append(live, lease)
		}
	}
	if len(live) == 0 {
		delete(m.held, key)
	} else {
		m.held[key] = live
	}
	return live
}

// Acquire takes a lease or returns FailureLeaseConflict. Refresh mode requires
// the profile to be completely free; read mode is refused while any refresh
// lease is live and otherwise bounded by MaxConcurrent.
func (m *LeaseManager) Acquire(_ context.Context, profile SafeProfile, run RunIdentity, mode LeaseMode) (ProfileLease, error) {
	if !mode.Valid() {
		return ProfileLease{}, fmt.Errorf("unknown lease mode %q", mode)
	}
	maxConcurrent := max(profile.Policy.MaxConcurrent, 1)

	m.mu.Lock()
	defer m.mu.Unlock()

	now := m.now()
	key := leaseKey(profile.Ref())
	live := m.sweepLocked(key, now)

	if mode.Exclusive() {
		if len(live) > 0 {
			return ProfileLease{}, NewFailureReason(FailureLeaseConflict, "acquire",
				fmt.Sprintf("refresh needs exclusive access; %d lease(s) held", len(live)))
		}
	} else {
		for _, lease := range live {
			if lease.Mode.Exclusive() {
				return ProfileLease{}, NewFailureReason(FailureLeaseConflict, "acquire",
					"a refresh lease is active")
			}
		}
		if len(live) >= maxConcurrent {
			return ProfileLease{}, NewFailureReason(FailureLeaseConflict, "acquire",
				fmt.Sprintf("max_concurrent=%d reached", maxConcurrent))
		}
	}

	safeID, err := m.idFunc()
	if err != nil {
		return ProfileLease{}, NewFailure(FailureLeaseConflict, "acquire", err)
	}

	lease := ProfileLease{
		SafeID:      safeID,
		Alias:       profile.Alias,
		Version:     profile.Version,
		ProfileHash: profile.ProfileHash,
		Mode:        mode,
		Run:         run,
		AcquiredAt:  now,
		ExpiresAt:   now.Add(m.ttl),
	}
	m.held[key] = append(m.held[key], lease)
	return lease, nil
}

// Release frees a lease. It is idempotent and tolerates leases it never issued,
// because it runs on every controller exit path including ones that already
// released — a spurious error there would turn cleanup into a false alarm.
func (m *LeaseManager) Release(_ context.Context, lease ProfileLease) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := leaseKey(lease.Ref())
	existing := m.held[key]
	remaining := existing[:0:0]
	for _, candidate := range existing {
		if candidate.SafeID != lease.SafeID {
			remaining = append(remaining, candidate)
		}
	}
	if len(remaining) == 0 {
		delete(m.held, key)
	} else {
		m.held[key] = remaining
	}
	return nil
}

// Active returns the live leases for a profile version, sweeping expired ones
// first so the answer reflects reality rather than history.
func (m *LeaseManager) Active(ref AuthProfileRef) []ProfileLease {
	m.mu.Lock()
	defer m.mu.Unlock()

	live := m.sweepLocked(leaseKey(ref), m.now())
	return append([]ProfileLease(nil), live...)
}

// AccountPool holds distinct least-privilege accounts for parallel work that
// changes server-side state. It exists so parallel mutation never means eight
// sessions on one shared account.
type AccountPool struct {
	Name    string
	Members []SafeProfile
}

func NewAccountPool(name string, members []SafeProfile) *AccountPool {
	return &AccountPool{Name: name, Members: append([]SafeProfile(nil), members...)}
}

// Allocate leases the first free member. Exhaustion is a structured conflict,
// never a silent second lease on an account already in use.
func (p *AccountPool) Allocate(ctx context.Context, manager *LeaseManager, run RunIdentity, mode LeaseMode) (ProfileLease, error) {
	if p == nil || len(p.Members) == 0 {
		return ProfileLease{}, fmt.Errorf("account pool %q has no members", poolName(p))
	}
	for _, member := range p.Members {
		lease, err := manager.Acquire(ctx, member, run, mode)
		if err == nil {
			return lease, nil
		}
		if !IsFailure(err, FailureLeaseConflict) {
			return ProfileLease{}, err
		}
	}
	return ProfileLease{}, NewFailureReason(FailureLeaseConflict, "pool_allocate",
		fmt.Sprintf("all %d accounts in pool %q are leased", len(p.Members), p.Name))
}

func poolName(p *AccountPool) string {
	if p == nil {
		return ""
	}
	return p.Name
}
