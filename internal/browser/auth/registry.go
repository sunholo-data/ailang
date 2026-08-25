package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"sync"
	"time"
)

// Record is a full profile version: the safe metadata plus the credential-grade
// material. Only trusted control-plane code holds a Record. Everything that
// crosses a boundary carries SafeProfile instead.
//
// On Publish the caller fills Alias, Version, Provider, Policy, and Material;
// Safe is ignored. On Open the registry fills all of them.
type Record struct {
	Alias    string
	Version  string
	Provider string
	Policy   AuthProfilePolicy
	Material SensitiveProfileMaterial
	Safe     SafeProfile
}

// Registry stores immutable profile versions. A published version is never
// mutated: refresh publishes a new version and retires the old one.
type Registry interface {
	// Publish stores a new immutable version and returns its safe projection.
	Publish(ctx context.Context, record Record) (SafeProfile, error)

	// Resolve converts a reference (possibly latest) into safe metadata, or a
	// terminal failure for missing, revoked, or expired versions.
	Resolve(ctx context.Context, ref AuthProfileRef) (SafeProfile, error)

	// Open returns the private record including material. Trusted callers only.
	Open(ctx context.Context, ref AuthProfileRef) (Record, error)

	// List returns every version of an alias in publish order.
	List(ctx context.Context, alias string) ([]SafeProfile, error)

	// Revoke marks a version permanently unusable. It is idempotent.
	Revoke(ctx context.Context, ref AuthProfileRef, reason string) error

	// Retire removes a version from latest resolution while keeping pinned
	// references working, so a rollback can still name it.
	Retire(ctx context.Context, ref AuthProfileRef) error
}

// hashDomain separates this hash from any other sha256 over the same bytes.
const hashDomain = "ailang.browser.auth.profile.v1\x00"

// profileHash fingerprints the canonical material for audit and correlation. It
// is a truncated, domain-separated digest: enough to tell two versions apart,
// never enough to reconstruct one.
func profileHash(material SensitiveProfileMaterial) string {
	kind, state, contextID := material.Materialize()
	digest := sha256.New()
	digest.Write([]byte(hashDomain))
	digest.Write([]byte(kind))
	digest.Write([]byte{0})
	digest.Write(state)
	digest.Write([]byte{0})
	digest.Write([]byte(contextID))
	return "sha256:" + hex.EncodeToString(digest.Sum(nil))[:16]
}

// MemoryRegistry is the in-process reference implementation. It is the test
// registry and the base a durable backend wraps; the durable backend choice is
// an explicitly deferred decision in the design document.
type MemoryRegistry struct {
	mu      sync.RWMutex
	byAlias map[string][]*Record
	nextSeq map[string]int
	nowFunc func() time.Time
}

func NewMemoryRegistry() *MemoryRegistry {
	return &MemoryRegistry{
		byAlias: make(map[string][]*Record),
		nextSeq: make(map[string]int),
		nowFunc: time.Now,
	}
}

// SetClock injects a deterministic clock. Tests and replay use it; production
// leaves the default.
func (r *MemoryRegistry) SetClock(now func() time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.nowFunc = now
}

func (r *MemoryRegistry) now() time.Time {
	if r.nowFunc == nil {
		return time.Now()
	}
	return r.nowFunc()
}

func (r *MemoryRegistry) Publish(_ context.Context, record Record) (SafeProfile, error) {
	if err := validateAlias(record.Alias); err != nil {
		return SafeProfile{}, err
	}
	if err := validateVersion(record.Version); err != nil {
		return SafeProfile{}, err
	}
	if record.Provider == "" {
		return SafeProfile{}, fmt.Errorf("profile %s@%s has no provider", record.Alias, record.Version)
	}
	if record.Material.Empty() {
		return SafeProfile{}, fmt.Errorf("profile %s@%s has no material", record.Alias, record.Version)
	}
	if err := record.Policy.Validate(); err != nil {
		return SafeProfile{}, fmt.Errorf("profile %s@%s: %w", record.Alias, record.Version, err)
	}
	policy, err := record.Policy.Normalized()
	if err != nil {
		return SafeProfile{}, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	existing := r.byAlias[record.Alias]
	for _, candidate := range existing {
		if candidate.Version == record.Version {
			return SafeProfile{}, fmt.Errorf("profile %s@%s already exists; versions are immutable", record.Alias, record.Version)
		}
	}

	previousVersion := ""
	if len(existing) > 0 {
		previousVersion = existing[len(existing)-1].Version
	}

	r.nextSeq[record.Alias]++
	stored := &Record{
		Alias:    record.Alias,
		Version:  record.Version,
		Provider: record.Provider,
		Policy:   policy,
		Material: record.Material,
		Safe: SafeProfile{
			Alias:           record.Alias,
			Version:         record.Version,
			Sequence:        r.nextSeq[record.Alias],
			ProfileHash:     profileHash(record.Material),
			Provider:        record.Provider,
			Policy:          policy,
			CreatedAt:       r.now().UTC(),
			PreviousVersion: previousVersion,
			ExpiresAtOrZero: policy.ExpiresAt,
		},
	}
	r.byAlias[record.Alias] = append(existing, stored)
	return stored.Safe, nil
}

// find locates a stored record. It returns nil when the alias or version is
// unknown; lifecycle checks are applied by the caller.
func (r *MemoryRegistry) find(ref AuthProfileRef) *Record {
	versions := r.byAlias[ref.Alias]
	if len(versions) == 0 {
		return nil
	}
	if !ref.IsLatest() {
		for _, candidate := range versions {
			if candidate.Version == ref.Version {
				return candidate
			}
		}
		return nil
	}

	// latest is the highest sequence that is neither revoked nor retired.
	// Expiry is deliberately NOT a filter here: an expired latest should
	// surface as browser_auth_profile_expired, which tells the operator to
	// refresh, rather than as not_found, which does not.
	var best *Record
	for _, candidate := range versions {
		if candidate.Safe.Revoked() || candidate.Safe.Retired() {
			continue
		}
		if best == nil || candidate.Safe.Sequence > best.Safe.Sequence {
			best = candidate
		}
	}
	return best
}

// check applies the terminal lifecycle rules in a fixed order so the reported
// category is deterministic: revoked outranks expired.
func (r *MemoryRegistry) check(record *Record, op string, ref AuthProfileRef) error {
	if record == nil {
		return NewFailureReason(FailureProfileNotFound, op, "unknown "+ref.String())
	}
	if record.Safe.Revoked() {
		return NewFailureReason(FailureProfileRevoked, op, "revoked "+record.Safe.Ref().String())
	}
	if record.Safe.Expired(r.now()) {
		return NewFailureReason(FailureProfileExpired, op, "expired "+record.Safe.Ref().String())
	}
	return nil
}

func (r *MemoryRegistry) Resolve(_ context.Context, ref AuthProfileRef) (SafeProfile, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	record := r.find(ref)
	if err := r.check(record, "resolve", ref); err != nil {
		return SafeProfile{}, err
	}
	return record.Safe, nil
}

func (r *MemoryRegistry) Open(_ context.Context, ref AuthProfileRef) (Record, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	record := r.find(ref)
	if err := r.check(record, "open", ref); err != nil {
		return Record{}, err
	}
	return *record, nil
}

func (r *MemoryRegistry) List(_ context.Context, alias string) ([]SafeProfile, error) {
	if err := validateAlias(alias); err != nil {
		return nil, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()

	versions := r.byAlias[alias]
	out := make([]SafeProfile, 0, len(versions))
	for _, candidate := range versions {
		out = append(out, candidate.Safe)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Sequence < out[j].Sequence })
	return out, nil
}

func (r *MemoryRegistry) Revoke(_ context.Context, ref AuthProfileRef, reason string) error {
	if ref.IsLatest() {
		return fmt.Errorf("revoke requires a concrete version, got %s", ref)
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	record := r.find(ref)
	if record == nil {
		return NewFailureReason(FailureProfileNotFound, "revoke", "unknown "+ref.String())
	}
	if record.Safe.Revoked() {
		return nil // idempotent: incident response may revoke twice
	}
	record.Safe.RevokedAt = r.now().UTC()
	record.Safe.RevocationReason = reason
	return nil
}

func (r *MemoryRegistry) Retire(_ context.Context, ref AuthProfileRef) error {
	if ref.IsLatest() {
		return fmt.Errorf("retire requires a concrete version, got %s", ref)
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	record := r.find(ref)
	if record == nil {
		return NewFailureReason(FailureProfileNotFound, "retire", "unknown "+ref.String())
	}
	if record.Safe.Retired() {
		return nil
	}
	record.Safe.RetiredAt = r.now().UTC()
	return nil
}

var _ Registry = (*MemoryRegistry)(nil)
