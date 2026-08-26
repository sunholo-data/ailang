package auth

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// BrokerOptions wires the broker to its collaborators. Every one is required
// except Audit and Now: a broker with no registry, no leases, no key protector,
// or no session root cannot be made safe by a default.
type BrokerOptions struct {
	Registry    Registry
	Leases      *LeaseManager
	Protector   KeyProtector
	Audit       AuditSink
	SessionRoot string
	Now         func() time.Time
}

// Broker is the single boundary between a run and a canonical profile. It
// resolves, preflights, leases, materializes, and destroys — and the model
// reaches none of it. The agent receives an already-authenticated browser and a
// safe alias/hash in the result.
type Broker struct {
	registry    Registry
	leases      *LeaseManager
	protector   KeyProtector
	audit       AuditSink
	sessionRoot string
	nowFunc     func() time.Time

	mu   sync.Mutex
	live map[string]*Materialization
}

func NewBroker(options BrokerOptions) (*Broker, error) {
	switch {
	case options.Registry == nil:
		return nil, errors.New("browser auth broker requires a registry")
	case options.Leases == nil:
		return nil, errors.New("browser auth broker requires a lease manager")
	case options.Protector == nil:
		return nil, errors.New("browser auth broker requires a key protector")
	case options.SessionRoot == "":
		return nil, errors.New("browser auth broker requires a session root")
	}
	audit := options.Audit
	if audit == nil {
		audit = NewMemoryAuditSink()
	}
	now := options.Now
	if now == nil {
		now = time.Now
	}
	return &Broker{
		registry:    options.Registry,
		leases:      options.Leases,
		protector:   options.Protector,
		audit:       audit,
		sessionRoot: options.SessionRoot,
		nowFunc:     now,
		live:        make(map[string]*Materialization),
	}, nil
}

func (b *Broker) SessionRoot() string   { return b.sessionRoot }
func (b *Broker) Leases() *LeaseManager { return b.leases }
func (b *Broker) now() time.Time        { return b.nowFunc() }
func (b *Broker) record(ctx context.Context, event AuditEvent) {
	// A sink failure must never fail the operation being audited. Callers that
	// need durable audit should use a sink that blocks, not a broker that aborts.
	_ = b.audit.Record(ctx, event)
}

// ProvisionRequest is everything one authenticated run needs. It carries no
// secrets, so it is safe to log verbatim when a provisioning is denied.
type ProvisionRequest struct {
	Ref      AuthProfileRef
	Run      RunIdentity
	Mode     LeaseMode
	Provider string

	RequestedOrigins     []string
	RequestedArtifacts   []string
	RequestHumanTakeover bool
	RequestPersistence   bool
}

// Provisioned is a live authenticated identity: a resolved profile, a held
// lease, and — for local providers — a disposable decrypted file. It must be
// passed to Teardown on every exit path.
type Provisioned struct {
	Profile SafeProfile
	Lease   ProfileLease

	// Material is the provider-specific opaque value. For a hosted provider it
	// carries the context reference; for a local one it is the sealed envelope's
	// kind marker and the real payload is on disk at StorageStatePath.
	Material SensitiveProfileMaterial

	// StorageStatePath is the disposable 0600 file for local providers, empty
	// for hosted ones. The path is safe to pass as a child argument; its
	// contents are not safe to log.
	StorageStatePath string

	materialization *Materialization
}

// Provision runs the full sequence in the only safe order: resolve, check the
// provider matches, preflight, lease, materialize.
//
// Everything that can deny happens BEFORE a lease is taken, and the lease is
// released if materialization then fails — a leaked lease wedges a profile for
// its whole TTL and is the failure most likely to go unnoticed.
func (b *Broker) Provision(ctx context.Context, request ProvisionRequest) (*Provisioned, error) {
	profile, err := b.Resolve(ctx, request.Ref)
	if err != nil {
		return nil, err
	}

	if request.Provider != "" && profile.Provider != "" && request.Provider != profile.Provider {
		failure := NewFailureReason(FailureScopeDenied, "provision",
			fmt.Sprintf("provider_mismatch: profile is %s, run requested %s", profile.Provider, request.Provider))
		b.record(ctx, NewAuditEvent("preflight", profile, ProfileLease{}, request.Run, b.now(), failure))
		return nil, failure
	}

	preflight := PreflightInput{
		Profile:              profile,
		Mode:                 request.Mode,
		Now:                  b.now(),
		RequestedOrigins:     request.RequestedOrigins,
		RequestedArtifacts:   request.RequestedArtifacts,
		RequestHumanTakeover: request.RequestHumanTakeover,
		RequestPersistence:   request.RequestPersistence,
	}
	if err := Preflight(preflight); err != nil {
		b.record(ctx, NewAuditEvent("preflight", profile, ProfileLease{}, request.Run, b.now(), err))
		return nil, err
	}

	lease, err := b.Acquire(ctx, profile, request.Run, request.Mode)
	if err != nil {
		return nil, err
	}

	material, err := b.Materialize(ctx, lease, request.Provider, b.sessionRoot)
	if err != nil {
		// Release before returning: the caller has no handle to release with.
		_ = b.leases.Release(ctx, lease)
		return nil, err
	}

	provisioned := &Provisioned{
		Profile:  profile,
		Lease:    lease,
		Material: material,
	}
	b.mu.Lock()
	if materialization, ok := b.live[lease.SafeID]; ok {
		provisioned.materialization = materialization
		provisioned.StorageStatePath = materialization.Path()
	}
	b.mu.Unlock()

	return provisioned, nil
}

// Teardown destroys any decrypted state and frees the lease. It is idempotent
// and nil-safe because it runs on every controller exit path, including ones
// that already ran it and ones reached by panic.
//
// A cleanup failure is returned as FailureCleanupFailed, and the lease is freed
// regardless: a file that will not delete must not also wedge the profile.
func (b *Broker) Teardown(ctx context.Context, provisioned *Provisioned) error {
	if provisioned == nil {
		return nil
	}

	var cleanupErr error
	if provisioned.materialization != nil {
		if err := provisioned.materialization.Destroy(); err != nil {
			cleanupErr = NewFailure(FailureCleanupFailed, "destroy materialization", err)
		}
	}

	b.mu.Lock()
	delete(b.live, provisioned.Lease.SafeID)
	b.mu.Unlock()

	if err := b.leases.Release(ctx, provisioned.Lease); err != nil && cleanupErr == nil {
		cleanupErr = NewFailure(FailureCleanupFailed, "release lease", err)
	}

	b.record(ctx, NewAuditEvent("release", provisioned.Profile, provisioned.Lease, provisioned.Lease.Run, b.now(), cleanupErr))
	return cleanupErr
}

// Resolve implements AuthProfileBroker.
func (b *Broker) Resolve(ctx context.Context, ref AuthProfileRef) (SafeProfile, error) {
	profile, err := b.registry.Resolve(ctx, ref)
	if err != nil {
		// The profile is unknown, so the audit record carries only the request.
		b.record(ctx, NewAuditEvent("resolve", SafeProfile{Alias: ref.Alias, Version: ref.Version}, ProfileLease{}, RunIdentity{}, b.now(), err))
		return SafeProfile{}, err
	}
	b.record(ctx, NewAuditEvent("resolve", profile, ProfileLease{}, RunIdentity{}, b.now(), nil))
	return profile, nil
}

// Acquire implements AuthProfileBroker.
func (b *Broker) Acquire(ctx context.Context, profile SafeProfile, run RunIdentity, mode LeaseMode) (ProfileLease, error) {
	lease, err := b.leases.Acquire(ctx, profile, run, mode)
	b.record(ctx, NewAuditEvent("acquire", profile, lease, run, b.now(), err))
	if err != nil {
		return ProfileLease{}, err
	}
	return lease, nil
}

// Materialize implements AuthProfileBroker. For a hosted provider it returns the
// opaque context reference and touches no disk. For a local provider it decrypts
// the canonical blob into a disposable 0600 file under dst and returns a marker;
// the path is reachable through Provisioned or MaterializedPath.
func (b *Broker) Materialize(ctx context.Context, lease ProfileLease, _ string, dst string) (SensitiveProfileMaterial, error) {
	profile, err := b.registry.Resolve(ctx, lease.Ref())
	if err != nil {
		return SensitiveProfileMaterial{}, err
	}

	record, err := b.registry.Open(ctx, lease.Ref())
	if err != nil {
		b.record(ctx, NewAuditEvent("materialize", profile, lease, lease.Run, b.now(), err))
		return SensitiveProfileMaterial{}, err
	}

	kind, sealedBytes, contextID := record.Material.Materialize()
	switch kind {
	case MaterialProviderContext:
		b.record(ctx, NewAuditEvent("materialize", profile, lease, lease.Run, b.now(), nil))
		return NewProviderContextMaterial(contextID), nil

	case MaterialStorageState:
		sealed, err := ParseSealedEnvelope(sealedBytes)
		if err != nil {
			failure := NewFailureReason(FailureMaterializeFailed, "materialize", "unparseable_canonical_envelope")
			b.record(ctx, NewAuditEvent("materialize", profile, lease, lease.Run, b.now(), failure))
			return SensitiveProfileMaterial{}, failure
		}
		root := dst
		if root == "" {
			root = b.sessionRoot
		}
		materialization, err := MaterializeStorageState(ctx, b.protector, sealed, MaterializeOptions{
			SessionRoot: root,
			RunID:       lease.Run.RunID,
			ProfileHash: profile.ProfileHash,
		})
		if err != nil {
			b.record(ctx, NewAuditEvent("materialize", profile, lease, lease.Run, b.now(), err))
			return SensitiveProfileMaterial{}, err
		}
		b.mu.Lock()
		b.live[lease.SafeID] = materialization
		b.mu.Unlock()
		b.record(ctx, NewAuditEvent("materialize", profile, lease, lease.Run, b.now(), nil))
		// The marker carries the kind, not the bytes: the plaintext lives only in
		// the 0600 file, so there is exactly one copy to destroy.
		return NewStorageStateMaterial(nil), nil

	default:
		failure := NewFailureReason(FailureMaterializeFailed, "materialize", "unknown_material_kind")
		b.record(ctx, NewAuditEvent("materialize", profile, lease, lease.Run, b.now(), failure))
		return SensitiveProfileMaterial{}, failure
	}
}

// MaterializedPath returns the disposable storage-state path held for a lease.
func (b *Broker) MaterializedPath(lease ProfileLease) (string, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	materialization, ok := b.live[lease.SafeID]
	if !ok {
		return "", false
	}
	return materialization.Path(), true
}

// Release implements AuthProfileBroker.
func (b *Broker) Release(ctx context.Context, lease ProfileLease) error {
	b.mu.Lock()
	materialization := b.live[lease.SafeID]
	delete(b.live, lease.SafeID)
	b.mu.Unlock()

	var cleanupErr error
	if materialization != nil {
		if err := materialization.Destroy(); err != nil {
			cleanupErr = NewFailure(FailureCleanupFailed, "destroy materialization", err)
		}
	}
	if err := b.leases.Release(ctx, lease); err != nil && cleanupErr == nil {
		cleanupErr = NewFailure(FailureCleanupFailed, "release lease", err)
	}
	b.record(ctx, NewAuditEvent("release", SafeProfile{Alias: lease.Alias, Version: lease.Version, ProfileHash: lease.ProfileHash},
		lease, lease.Run, b.now(), cleanupErr))
	return cleanupErr
}

// Revoke implements AuthProfileBroker.
func (b *Broker) Revoke(ctx context.Context, ref AuthProfileRef, reason string) error {
	err := b.registry.Revoke(ctx, ref, reason)
	event := NewAuditEvent("revoke", SafeProfile{Alias: ref.Alias, Version: ref.Version}, ProfileLease{}, RunIdentity{}, b.now(), err)
	if err == nil {
		event.Decision = DecisionRevoked
		event.Reason = reason
	}
	b.record(ctx, event)
	return err
}

// setRemoveAllForTest replaces the deletion primitive so cleanup-failure
// handling can be exercised without an unwritable filesystem.
func (b *Broker) setRemoveAllForTest(provisioned *Provisioned, removeAll func(string) error) {
	if provisioned == nil || provisioned.materialization == nil {
		return
	}
	provisioned.materialization.mu.Lock()
	defer provisioned.materialization.mu.Unlock()
	provisioned.materialization.removeAll = removeAll
}

var _ AuthProfileBroker = (*Broker)(nil)
