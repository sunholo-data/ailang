package auth

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func newFileRegistry(t *testing.T) *FileRegistry {
	t.Helper()
	registry, err := NewFileRegistry(filepath.Join(t.TempDir(), "profiles"))
	if err != nil {
		t.Fatalf("NewFileRegistry: %v", err)
	}
	return registry
}

func publishFile(t *testing.T, registry *FileRegistry, alias, version, material string) SafeProfile {
	t.Helper()
	profile, err := registry.Publish(context.Background(), Record{
		Alias: alias, Version: version, Provider: "local-playwright",
		Policy: testPolicy(), Material: NewStorageStateMaterial([]byte(material)),
	})
	if err != nil {
		t.Fatalf("Publish(%s@%s): %v", alias, version, err)
	}
	return profile
}

func TestFileRegistryRoundTripsAcrossInstances(t *testing.T) {
	root := filepath.Join(t.TempDir(), "profiles")
	first, err := NewFileRegistry(root)
	if err != nil {
		t.Fatalf("NewFileRegistry: %v", err)
	}
	ctx := context.Background()
	publishFile(t, first, "crm", "v1", "sealed-bytes-v1")
	publishFile(t, first, "crm", "v2", "sealed-bytes-v2")

	// A separate instance over the same root is what the CLI and a later eval
	// run actually are: two processes, one registry.
	second, err := NewFileRegistry(root)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	resolved, err := second.Resolve(ctx, AuthProfileRef{Alias: "crm", Version: VersionLatest})
	if err != nil {
		t.Fatalf("Resolve latest: %v", err)
	}
	if resolved.Version != "v2" || resolved.Sequence != 2 {
		t.Fatalf("latest = %s (seq %d), want v2 seq 2", resolved.Version, resolved.Sequence)
	}
	if resolved.PreviousVersion != "v1" {
		t.Fatalf("rollback pointer = %q, want v1", resolved.PreviousVersion)
	}

	record, err := second.Open(ctx, AuthProfileRef{Alias: "crm", Version: "v1"})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	_, material, _ := record.Material.Materialize()
	if string(material) != "sealed-bytes-v1" {
		t.Fatalf("material did not survive the round trip")
	}
}

func TestFileRegistryUsesOwnerOnlyPermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		// Go reports 0666/0444 on Windows regardless of ACL, so asserting mode
		// bits there would be meaningless rather than lenient.
		t.Skip("POSIX mode bits are not meaningful on Windows")
	}
	registry := newFileRegistry(t)
	publishFile(t, registry, "crm", "v1", "sealed-bytes")

	rootInfo, err := os.Stat(registry.Root())
	if err != nil {
		t.Fatalf("stat root: %v", err)
	}
	if got := rootInfo.Mode().Perm(); got != profileDirMode {
		t.Fatalf("registry root mode = %o, want %o", got, profileDirMode)
	}

	for _, name := range []string{"v1.material", "v1.meta.json"} {
		info, err := os.Stat(filepath.Join(registry.Root(), "crm", name))
		if err != nil {
			t.Fatalf("stat %s: %v", name, err)
		}
		if got := info.Mode().Perm(); got != profileFileMode {
			t.Fatalf("%s mode = %o, want %o", name, got, profileFileMode)
		}
	}
}

// Metadata is written separately from material; a reader must never see the
// credential in the JSON that describes it.
func TestFileRegistryMetadataHoldsNoStorageState(t *testing.T) {
	registry := newFileRegistry(t)
	publishFile(t, registry, "crm", "v1", cookieJar)

	raw, err := os.ReadFile(filepath.Join(registry.Root(), "crm", "v1.meta.json"))
	if err != nil {
		t.Fatalf("read metadata: %v", err)
	}
	for _, marker := range leakMarkers {
		if strings.Contains(string(raw), marker) {
			t.Fatalf("profile metadata leaked %q", marker)
		}
	}
	if !strings.Contains(string(raw), "crm") {
		t.Fatalf("profile metadata dropped the alias")
	}
}

func TestFileRegistryEnforcesImmutability(t *testing.T) {
	registry := newFileRegistry(t)
	publishFile(t, registry, "crm", "v1", "sealed")

	if _, err := registry.Publish(context.Background(), Record{
		Alias: "crm", Version: "v1", Provider: "local-playwright",
		Policy: testPolicy(), Material: NewStorageStateMaterial([]byte("different")),
	}); err == nil {
		t.Fatalf("republishing v1 succeeded — versions must be immutable")
	}
}

func TestFileRegistryRevokeRetirePersist(t *testing.T) {
	registry := newFileRegistry(t)
	ctx := context.Background()
	publishFile(t, registry, "crm", "v1", "sealed-1")
	publishFile(t, registry, "crm", "v2", "sealed-2")

	if err := registry.Retire(ctx, AuthProfileRef{Alias: "crm", Version: "v2"}); err != nil {
		t.Fatalf("Retire: %v", err)
	}
	if err := registry.Revoke(ctx, AuthProfileRef{Alias: "crm", Version: "v1"}, "rotation"); err != nil {
		t.Fatalf("Revoke: %v", err)
	}

	// Reopen: both states must have survived the process boundary.
	reopened, err := NewFileRegistry(registry.Root())
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	if _, err := reopened.Resolve(ctx, AuthProfileRef{Alias: "crm", Version: "v1"}); !IsFailure(err, FailureProfileRevoked) {
		t.Fatalf("revocation did not persist: %v", err)
	}
	if _, err := reopened.Resolve(ctx, AuthProfileRef{Alias: "crm", Version: "v2"}); err != nil {
		t.Fatalf("a retired version must stay pinnable: %v", err)
	}
	if _, err := reopened.Resolve(ctx, AuthProfileRef{Alias: "crm", Version: VersionLatest}); !IsFailure(err, FailureProfileNotFound) {
		t.Fatalf("latest over revoked+retired returned %v, want %s", err, FailureProfileNotFound)
	}
}

func TestFileRegistryExpiryPersists(t *testing.T) {
	registry := newFileRegistry(t)
	ctx := context.Background()
	policy := testPolicy()
	policy.ExpiresAt = time.Now().Add(-time.Hour)

	if _, err := registry.Publish(ctx, Record{
		Alias: "crm", Version: "v1", Provider: "local-playwright",
		Policy: policy, Material: NewStorageStateMaterial([]byte("sealed")),
	}); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	if _, err := registry.Resolve(ctx, AuthProfileRef{Alias: "crm", Version: "v1"}); !IsFailure(err, FailureProfileExpired) {
		t.Fatalf("expired profile resolved")
	}
}

// A corrupt metadata file must fail loudly. Skipping it silently would make
// `latest` resolve to an older version and nobody would notice.
func TestFileRegistryFailsLoudlyOnCorruptMetadata(t *testing.T) {
	registry := newFileRegistry(t)
	ctx := context.Background()
	publishFile(t, registry, "crm", "v1", "sealed-1")
	publishFile(t, registry, "crm", "v2", "sealed-2")

	corrupt := filepath.Join(registry.Root(), "crm", "v2.meta.json")
	if err := os.WriteFile(corrupt, []byte("{not json"), profileFileMode); err != nil {
		t.Fatalf("corrupt fixture: %v", err)
	}

	_, err := registry.Resolve(ctx, AuthProfileRef{Alias: "crm", Version: VersionLatest})
	if err == nil {
		t.Fatalf("a corrupt v2 silently resolved to an older version")
	}
	if IsFailure(err, FailureProfileNotFound) {
		t.Fatalf("a corrupt file was reported as not_found, hiding the corruption")
	}
}

func TestFileRegistryHostedContextRoundTrip(t *testing.T) {
	registry := newFileRegistry(t)
	ctx := context.Background()

	if _, err := registry.Publish(ctx, Record{
		Alias: "dashboard", Version: "v1", Provider: "browserbase",
		Policy: testPolicy(), Material: NewProviderContextMaterial(hostedContextID),
	}); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	record, err := registry.Open(ctx, AuthProfileRef{Alias: "dashboard", Version: "v1"})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	kind, _, contextID := record.Material.Materialize()
	if kind != MaterialProviderContext || contextID != hostedContextID {
		t.Fatalf("hosted material did not round-trip")
	}
}

func TestFileRegistryPurgeRequiresRevocationFirst(t *testing.T) {
	registry := newFileRegistry(t)
	ctx := context.Background()
	publishFile(t, registry, "crm", "v1", "sealed-1")
	ref := AuthProfileRef{Alias: "crm", Version: "v1"}

	if err := registry.Purge(ctx, ref); !IsFailure(err, FailureWritebackDenied) {
		t.Fatalf("Purge on a live profile returned %v, want %s", err, FailureWritebackDenied)
	}
	if _, err := os.Stat(filepath.Join(registry.Root(), "crm", "v1.material")); err != nil {
		t.Fatalf("Purge deleted material despite refusing: %v", err)
	}

	if err := registry.Revoke(ctx, ref, "rotation"); err != nil {
		t.Fatalf("Revoke: %v", err)
	}
	if err := registry.Purge(ctx, ref); err != nil {
		t.Fatalf("Purge after revocation: %v", err)
	}
	if _, err := os.Stat(filepath.Join(registry.Root(), "crm", "v1.material")); !os.IsNotExist(err) {
		t.Fatalf("Purge left the material on disk")
	}
	// Purge is idempotent: an incident runbook may run it twice.
	if err := registry.Purge(ctx, ref); err != nil {
		t.Fatalf("second Purge: %v", err)
	}
}

func TestFileRegistryAliases(t *testing.T) {
	registry := newFileRegistry(t)
	publishFile(t, registry, "crm", "v1", "sealed")
	publishFile(t, registry, "shop", "v1", "sealed")

	aliases, err := registry.Aliases()
	if err != nil {
		t.Fatalf("Aliases: %v", err)
	}
	if len(aliases) != 2 || aliases[0] != "crm" || aliases[1] != "shop" {
		t.Fatalf("Aliases() = %v, want [crm shop]", aliases)
	}
}

func TestFileRegistryRejectsPathEscapingAliases(t *testing.T) {
	registry := newFileRegistry(t)
	ctx := context.Background()

	for _, alias := range []string{"../escape", "..", "/etc", "a/b"} {
		if _, err := registry.Publish(ctx, Record{
			Alias: alias, Version: "v1", Provider: "local-playwright",
			Policy: testPolicy(), Material: NewStorageStateMaterial([]byte("sealed")),
		}); err == nil {
			t.Fatalf("Publish accepted alias %q — profile names become filesystem paths", alias)
		}
		if _, err := registry.Resolve(ctx, AuthProfileRef{Alias: alias, Version: "v1"}); err == nil {
			t.Fatalf("Resolve accepted alias %q", alias)
		}
	}
}

// The broker must work over the durable registry exactly as it does over the
// in-memory one — otherwise the CLI and the eval harness would diverge.
func TestBrokerWorksOverTheFileRegistry(t *testing.T) {
	protector, err := NewRandomStaticKeyProtector("test-key")
	if err != nil {
		t.Fatalf("protector: %v", err)
	}
	registry := newFileRegistry(t)
	broker, err := NewBroker(BrokerOptions{
		Registry:    registry,
		Leases:      NewLeaseManager(time.Minute),
		Protector:   protector,
		SessionRoot: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("NewBroker: %v", err)
	}
	ctx := context.Background()

	if _, err := broker.Bootstrap(ctx, BootstrapRequest{
		Alias: "crm", Version: "v1", Provider: "local-playwright",
		Policy: testPolicy(), State: []byte(cookieJar),
		Run: RunIdentity{RunID: "bootstrap", Principal: "operator"},
	}); err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}

	provisioned, err := broker.Provision(ctx, ProvisionRequest{
		Ref:      AuthProfileRef{Alias: "crm", Version: VersionLatest},
		Run:      RunIdentity{RunID: "run-1", Principal: "eval-harness"},
		Mode:     LeaseRead,
		Provider: "local-playwright",
	})
	if err != nil {
		t.Fatalf("Provision: %v", err)
	}
	contents, err := os.ReadFile(provisioned.StorageStatePath)
	if err != nil {
		t.Fatalf("read materialized state: %v", err)
	}
	if string(contents) != cookieJar {
		t.Fatalf("the durable path did not round-trip the plaintext")
	}
	if err := broker.Teardown(ctx, provisioned); err != nil {
		t.Fatalf("Teardown: %v", err)
	}
}
