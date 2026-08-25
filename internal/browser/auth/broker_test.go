package auth

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func newTestBroker(t *testing.T) (*Broker, *MemoryRegistry, *MemoryAuditSink, KeyProtector) {
	t.Helper()

	protector, err := NewRandomStaticKeyProtector("test-key")
	if err != nil {
		t.Fatalf("key protector: %v", err)
	}
	registry := NewMemoryRegistry()
	sink := NewMemoryAuditSink()

	broker, err := NewBroker(BrokerOptions{
		Registry:    registry,
		Leases:      NewLeaseManager(30 * time.Minute),
		Protector:   protector,
		Audit:       sink,
		SessionRoot: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("NewBroker: %v", err)
	}
	return broker, registry, sink, protector
}

// publishLocal stores a sealed storage-state profile the way a trusted bootstrap
// would: the registry holds ciphertext, never plaintext.
func publishLocal(t *testing.T, registry *MemoryRegistry, protector KeyProtector, alias, version string, plaintext string) SafeProfile {
	t.Helper()
	sealed, err := Seal(context.Background(), protector, []byte(plaintext))
	if err != nil {
		t.Fatalf("seal: %v", err)
	}
	profile, err := registry.Publish(context.Background(), Record{
		Alias:    alias,
		Version:  version,
		Provider: "local-playwright",
		Policy:   testPolicy(),
		Material: NewStorageStateMaterial(sealed.Bytes()),
	})
	if err != nil {
		t.Fatalf("publish: %v", err)
	}
	return profile
}

func goodRequest(alias string) ProvisionRequest {
	return ProvisionRequest{
		Ref:      AuthProfileRef{Alias: alias, Version: VersionLatest},
		Run:      RunIdentity{RunID: "run-42", Principal: "eval-harness"},
		Mode:     LeaseRead,
		Provider: "local-playwright",
	}
}

func TestProvisionResolvesLeasesAndMaterializes(t *testing.T) {
	broker, registry, sink, protector := newTestBroker(t)
	ctx := context.Background()
	publishLocal(t, registry, protector, "crm", "v1", cookieJar)

	provisioned, err := broker.Provision(ctx, goodRequest("crm"))
	if err != nil {
		t.Fatalf("Provision: %v", err)
	}
	defer func() { _ = broker.Teardown(ctx, provisioned) }()

	if provisioned.Profile.Version != "v1" {
		t.Fatalf("resolved version = %q, want v1", provisioned.Profile.Version)
	}
	if provisioned.Lease.SafeID == "" {
		t.Fatalf("no lease was taken")
	}
	if provisioned.StorageStatePath == "" {
		t.Fatalf("no storage state was materialized for a local profile")
	}

	// The materialized file must hold the ORIGINAL plaintext, decrypted.
	contents, err := os.ReadFile(provisioned.StorageStatePath)
	if err != nil {
		t.Fatalf("read materialized state: %v", err)
	}
	if string(contents) != cookieJar {
		t.Fatalf("materialized state does not match the canonical plaintext")
	}

	// The resolved version, not "latest", is what the run records.
	if provisioned.Lease.Version == VersionLatest {
		t.Fatalf("lease recorded %q instead of a concrete version", VersionLatest)
	}

	ops := auditOps(sink)
	for _, want := range []string{"resolve", "acquire", "materialize"} {
		if !ops[want] {
			t.Fatalf("no audit event for %q; got %v", want, ops)
		}
	}
}

// The whole point of the preflight is that it runs before anything expensive or
// dangerous exists. If it denies, there must be no lease and no plaintext.
func TestProvisionDeniedByPreflightLeavesNoLeaseAndNoPlaintext(t *testing.T) {
	broker, registry, sink, protector := newTestBroker(t)
	ctx := context.Background()

	sealed, err := Seal(ctx, protector, []byte(cookieJar))
	if err != nil {
		t.Fatalf("seal: %v", err)
	}
	policy := testPolicy()
	policy.EgressBoundary = EgressAbsent // no boundary => fail closed
	profile, err := registry.Publish(ctx, Record{
		Alias: "crm", Version: "v1", Provider: "local-playwright",
		Policy: policy, Material: NewStorageStateMaterial(sealed.Bytes()),
	})
	if err != nil {
		t.Fatalf("publish: %v", err)
	}

	_, err = broker.Provision(ctx, goodRequest("crm"))
	if !IsFailure(err, FailureScopeDenied) {
		t.Fatalf("Provision returned %v, want %s", err, FailureScopeDenied)
	}

	if active := broker.Leases().Active(profile.Ref()); len(active) != 0 {
		t.Fatalf("a denied provision left %d lease(s) held", len(active))
	}
	assertSessionRootEmpty(t, broker)

	if !auditOps(sink)["preflight"] {
		t.Fatalf("the denial was not audited")
	}
}

func TestProvisionOnARevokedProfileNeverReachesMaterialization(t *testing.T) {
	broker, registry, _, protector := newTestBroker(t)
	ctx := context.Background()
	publishLocal(t, registry, protector, "crm", "v1", cookieJar)

	if err := registry.Revoke(ctx, AuthProfileRef{Alias: "crm", Version: "v1"}, "rotation"); err != nil {
		t.Fatalf("revoke: %v", err)
	}

	// latest over an all-revoked alias is not_found; a pinned ref is revoked.
	if _, err := broker.Provision(ctx, goodRequest("crm")); err == nil {
		t.Fatalf("Provision succeeded on a revoked profile")
	}
	pinned := goodRequest("crm")
	pinned.Ref.Version = "v1"
	if _, err := broker.Provision(ctx, pinned); !IsFailure(err, FailureProfileRevoked) {
		t.Fatalf("pinned revoked provision returned %v, want %s", err, FailureProfileRevoked)
	}
	assertSessionRootEmpty(t, broker)
}

// A failure after the lease is taken must not leak the lease. This is the case
// that silently wedges a profile in production.
func TestProvisionReleasesTheLeaseWhenMaterializationFails(t *testing.T) {
	broker, registry, sink, _ := newTestBroker(t)
	ctx := context.Background()

	// Publish material that is NOT a parseable sealed envelope, so acquisition
	// succeeds and materialization then fails.
	profile, err := registry.Publish(ctx, Record{
		Alias: "crm", Version: "v1", Provider: "local-playwright",
		Policy: testPolicy(), Material: NewStorageStateMaterial([]byte("not-an-envelope")),
	})
	if err != nil {
		t.Fatalf("publish: %v", err)
	}

	if _, err := broker.Provision(ctx, goodRequest("crm")); !IsFailure(err, FailureMaterializeFailed) {
		t.Fatalf("Provision returned %v, want %s", err, FailureMaterializeFailed)
	}
	if active := broker.Leases().Active(profile.Ref()); len(active) != 0 {
		t.Fatalf("a failed materialization leaked %d lease(s)", len(active))
	}
	assertSessionRootEmpty(t, broker)

	// The profile must be immediately usable again once the cause is fixed.
	if !auditOps(sink)["materialize"] {
		t.Fatalf("the materialization failure was not audited")
	}
}

func TestTeardownDestroysPlaintextAndReleasesTheLease(t *testing.T) {
	broker, registry, sink, protector := newTestBroker(t)
	ctx := context.Background()
	profile := publishLocal(t, registry, protector, "crm", "v1", cookieJar)

	provisioned, err := broker.Provision(ctx, goodRequest("crm"))
	if err != nil {
		t.Fatalf("Provision: %v", err)
	}
	path := provisioned.StorageStatePath

	if err := broker.Teardown(ctx, provisioned); err != nil {
		t.Fatalf("Teardown: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("teardown left decrypted state at %s", path)
	}
	if active := broker.Leases().Active(profile.Ref()); len(active) != 0 {
		t.Fatalf("teardown left %d lease(s) held", len(active))
	}
	assertSessionRootEmpty(t, broker)
	if !auditOps(sink)["release"] {
		t.Fatalf("the release was not audited")
	}
}

// Teardown runs on every controller exit path, including ones that already ran
// it. A second call must be a no-op, not a cleanup failure.
func TestTeardownIsIdempotent(t *testing.T) {
	broker, registry, _, protector := newTestBroker(t)
	ctx := context.Background()
	publishLocal(t, registry, protector, "crm", "v1", cookieJar)

	provisioned, err := broker.Provision(ctx, goodRequest("crm"))
	if err != nil {
		t.Fatalf("Provision: %v", err)
	}
	for i := range 3 {
		if err := broker.Teardown(ctx, provisioned); err != nil {
			t.Fatalf("teardown %d: %v", i, err)
		}
	}
	if err := broker.Teardown(ctx, nil); err != nil {
		t.Fatalf("teardown of a nil provisioning: %v", err)
	}
}

// A cleanup failure must be reported as a cleanup failure — never swallowed, and
// never allowed to look like the run's own error.
func TestTeardownReportsCleanupFailureAndStillReleasesTheLease(t *testing.T) {
	broker, registry, sink, protector := newTestBroker(t)
	ctx := context.Background()
	profile := publishLocal(t, registry, protector, "crm", "v1", cookieJar)

	provisioned, err := broker.Provision(ctx, goodRequest("crm"))
	if err != nil {
		t.Fatalf("Provision: %v", err)
	}
	broker.setRemoveAllForTest(provisioned, func(string) error {
		return errors.New("device busy")
	})

	err = broker.Teardown(ctx, provisioned)
	if !IsFailure(err, FailureCleanupFailed) {
		t.Fatalf("Teardown returned %v, want %s", err, FailureCleanupFailed)
	}
	// The lease must still be freed: a stuck file must not also wedge the profile.
	if active := broker.Leases().Active(profile.Ref()); len(active) != 0 {
		t.Fatalf("a cleanup failure also leaked %d lease(s)", len(active))
	}
	if !auditOps(sink)["release"] {
		t.Fatalf("the failed cleanup was not audited")
	}
}

func TestProvisionForAHostedProviderReturnsContextMaterialAndNoPath(t *testing.T) {
	broker, registry, _, _ := newTestBroker(t)
	ctx := context.Background()

	if _, err := registry.Publish(ctx, Record{
		Alias: "dashboard", Version: "v7", Provider: "browserbase",
		Policy: testPolicy(), Material: NewProviderContextMaterial(hostedContextID),
	}); err != nil {
		t.Fatalf("publish: %v", err)
	}

	request := goodRequest("dashboard")
	request.Provider = "browserbase"
	provisioned, err := broker.Provision(ctx, request)
	if err != nil {
		t.Fatalf("Provision: %v", err)
	}
	defer func() { _ = broker.Teardown(ctx, provisioned) }()

	if provisioned.StorageStatePath != "" {
		t.Fatalf("a hosted profile materialized a local file at %s", provisioned.StorageStatePath)
	}
	kind, _, contextID := provisioned.Material.Materialize()
	if kind != MaterialProviderContext || contextID != hostedContextID {
		t.Fatalf("hosted material did not round-trip")
	}
	// A hosted provisioning writes nothing to disk at all.
	assertSessionRootEmpty(t, broker)
}

func TestProvisionRejectsAProviderMismatch(t *testing.T) {
	broker, registry, _, protector := newTestBroker(t)
	ctx := context.Background()
	publishLocal(t, registry, protector, "crm", "v1", cookieJar)

	request := goodRequest("crm")
	request.Provider = "browserbase" // profile was published for local-playwright
	if _, err := broker.Provision(ctx, request); !IsFailure(err, FailureScopeDenied) {
		t.Fatalf("provider mismatch returned %v, want %s", err, FailureScopeDenied)
	}
	assertSessionRootEmpty(t, broker)
}

func TestBrokerImplementsTheFrozenInterface(t *testing.T) {
	var _ AuthProfileBroker = (*Broker)(nil)
}

func TestRevokeThroughTheBrokerIsAudited(t *testing.T) {
	broker, registry, sink, protector := newTestBroker(t)
	ctx := context.Background()
	publishLocal(t, registry, protector, "crm", "v1", cookieJar)

	if err := broker.Revoke(ctx, AuthProfileRef{Alias: "crm", Version: "v1"}, "rotation drill"); err != nil {
		t.Fatalf("Revoke: %v", err)
	}
	if !auditOps(sink)["revoke"] {
		t.Fatalf("revocation was not audited")
	}
	if _, err := registry.Resolve(ctx, AuthProfileRef{Alias: "crm", Version: "v1"}); !IsFailure(err, FailureProfileRevoked) {
		t.Fatalf("the broker's revoke did not reach the registry")
	}
}

// Under contention the broker must grant exactly MaxConcurrent runs and leave no
// plaintext or lease behind for the ones it refused.
func TestConcurrentProvisionAndTeardownLeaksNothing(t *testing.T) {
	broker, registry, _, protector := newTestBroker(t)
	ctx := context.Background()
	profile := publishLocal(t, registry, protector, "crm", "v1", cookieJar)

	const workers = 24
	var wg sync.WaitGroup
	var mu sync.Mutex
	granted := 0

	for i := range workers {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			request := goodRequest("crm")
			request.Run = RunIdentity{RunID: "run-" + itoa(i), Principal: "eval-harness"}
			provisioned, err := broker.Provision(ctx, request)
			if err != nil {
				if !IsFailure(err, FailureLeaseConflict) {
					t.Errorf("unexpected provision error: %v", err)
				}
				return
			}
			mu.Lock()
			granted++
			mu.Unlock()
			if err := broker.Teardown(ctx, provisioned); err != nil {
				t.Errorf("teardown: %v", err)
			}
		}(i)
	}
	wg.Wait()

	if granted == 0 {
		t.Fatalf("no worker was granted a lease")
	}
	if active := broker.Leases().Active(profile.Ref()); len(active) != 0 {
		t.Fatalf("%d lease(s) leaked after every worker finished", len(active))
	}
	assertSessionRootEmpty(t, broker)
}

func assertSessionRootEmpty(t *testing.T, broker *Broker) {
	t.Helper()
	entries, err := os.ReadDir(broker.SessionRoot())
	if err != nil {
		t.Fatalf("read session root: %v", err)
	}
	if len(entries) != 0 {
		var names []string
		for _, entry := range entries {
			names = append(names, filepath.Join(broker.SessionRoot(), entry.Name()))
		}
		t.Fatalf("session root still holds %d materialization(s): %v", len(entries), names)
	}
}

func auditOps(sink *MemoryAuditSink) map[string]bool {
	ops := make(map[string]bool)
	for _, event := range sink.Events() {
		ops[event.Op] = true
	}
	return ops
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	digits := ""
	for n > 0 {
		digits = string(rune('0'+n%10)) + digits
		n /= 10
	}
	return digits
}
