package auth

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func testPolicy() AuthProfilePolicy {
	return AuthProfilePolicy{
		AllowedOrigins:     []string{"https://crm.example.com"},
		AccountClass:       AccountReadonly,
		MaxConcurrent:      1,
		AllowArtifacts:     []string{},
		AllowHumanTakeover: false,
		EgressBoundary:     EgressOperatorAcknowledged,
	}
}

func publishStorageState(t *testing.T, reg *MemoryRegistry, alias, version, state string) SafeProfile {
	t.Helper()
	profile, err := reg.Publish(context.Background(), Record{
		Alias:    alias,
		Version:  version,
		Provider: "local-playwright",
		Policy:   testPolicy(),
		Material: NewStorageStateMaterial([]byte(state)),
	})
	if err != nil {
		t.Fatalf("Publish(%s@%s): %v", alias, version, err)
	}
	return profile
}

func TestPublishAssignsHashAndSequence(t *testing.T) {
	reg := NewMemoryRegistry()

	first := publishStorageState(t, reg, "crm", "v1", cookieJar)
	if first.ProfileHash == "" {
		t.Fatalf("Publish did not assign a profile hash")
	}
	if !strings.HasPrefix(first.ProfileHash, "sha256:") {
		t.Fatalf("ProfileHash = %q, want a sha256: prefix", first.ProfileHash)
	}
	if strings.Contains(first.ProfileHash, "SUPER-SECRET-SESSION") {
		t.Fatalf("ProfileHash embedded the plaintext")
	}
	if first.Sequence != 1 {
		t.Fatalf("first Sequence = %d, want 1", first.Sequence)
	}

	second := publishStorageState(t, reg, "crm", "v2", cookieJar+" ")
	if second.Sequence != 2 {
		t.Fatalf("second Sequence = %d, want 2", second.Sequence)
	}
	if second.ProfileHash == first.ProfileHash {
		t.Fatalf("different material produced the same hash")
	}
}

func TestPublishRejectsRepublishingAnExistingVersion(t *testing.T) {
	reg := NewMemoryRegistry()
	publishStorageState(t, reg, "crm", "v1", cookieJar)

	_, err := reg.Publish(context.Background(), Record{
		Alias:    "crm",
		Version:  "v1",
		Provider: "local-playwright",
		Policy:   testPolicy(),
		Material: NewStorageStateMaterial([]byte("different")),
	})
	if err == nil {
		t.Fatalf("republishing v1 succeeded — versions must be immutable")
	}
}

func TestPublishRejectsReservedAndMalformedVersions(t *testing.T) {
	reg := NewMemoryRegistry()
	for _, version := range []string{VersionLatest, "", "../escape", "v 1", "Latest"} {
		_, err := reg.Publish(context.Background(), Record{
			Alias:    "crm",
			Version:  version,
			Provider: "local-playwright",
			Policy:   testPolicy(),
			Material: NewStorageStateMaterial([]byte(cookieJar)),
		})
		if err == nil {
			t.Fatalf("Publish accepted version %q", version)
		}
	}
}

func TestPublishRejectsEmptyMaterialAndInvalidPolicy(t *testing.T) {
	reg := NewMemoryRegistry()
	ctx := context.Background()

	if _, err := reg.Publish(ctx, Record{Alias: "crm", Version: "v1", Provider: "local-playwright", Policy: testPolicy()}); err == nil {
		t.Fatalf("Publish accepted empty material")
	}

	badClass := testPolicy()
	badClass.AccountClass = "admin"
	if _, err := reg.Publish(ctx, Record{
		Alias: "crm", Version: "v1", Provider: "local-playwright",
		Policy: badClass, Material: NewStorageStateMaterial([]byte(cookieJar)),
	}); err == nil {
		t.Fatalf("Publish accepted an invalid account class")
	}

	badOrigin := testPolicy()
	badOrigin.AllowedOrigins = []string{"https://*.example.com"}
	if _, err := reg.Publish(ctx, Record{
		Alias: "crm", Version: "v1", Provider: "local-playwright",
		Policy: badOrigin, Material: NewStorageStateMaterial([]byte(cookieJar)),
	}); err == nil {
		t.Fatalf("Publish accepted a wildcard origin")
	}
}

func TestPublishNormalizesOrigins(t *testing.T) {
	reg := NewMemoryRegistry()
	policy := testPolicy()
	policy.AllowedOrigins = []string{"https://CRM.Example.com:443/"}

	profile, err := reg.Publish(context.Background(), Record{
		Alias: "crm", Version: "v1", Provider: "local-playwright",
		Policy: policy, Material: NewStorageStateMaterial([]byte(cookieJar)),
	})
	if err != nil {
		t.Fatalf("Publish: %v", err)
	}
	if got, want := profile.Policy.AllowedOrigins[0], "https://crm.example.com"; got != want {
		t.Fatalf("stored origin = %q, want %q", got, want)
	}
}

func TestResolveLatestPicksHighestSequenceAndNeverReturnsLatest(t *testing.T) {
	reg := NewMemoryRegistry()
	ctx := context.Background()

	publishStorageState(t, reg, "crm", "v1", cookieJar)
	publishStorageState(t, reg, "crm", "v2", cookieJar+"2")
	publishStorageState(t, reg, "crm", "v3", cookieJar+"3")

	resolved, err := reg.Resolve(ctx, AuthProfileRef{Alias: "crm", Version: VersionLatest})
	if err != nil {
		t.Fatalf("Resolve latest: %v", err)
	}
	if resolved.Version != "v3" {
		t.Fatalf("latest resolved to %q, want v3", resolved.Version)
	}
	if resolved.Ref().IsLatest() {
		t.Fatalf("resolved profile still reports latest")
	}
}

// Publish order, not version-string order, defines latest. A hand-numbered v10
// published before v9 must not win.
func TestResolveLatestIgnoresVersionStringOrdering(t *testing.T) {
	reg := NewMemoryRegistry()
	publishStorageState(t, reg, "crm", "v10", cookieJar)
	publishStorageState(t, reg, "crm", "v9", cookieJar+"9")

	resolved, err := reg.Resolve(context.Background(), AuthProfileRef{Alias: "crm", Version: VersionLatest})
	if err != nil {
		t.Fatalf("Resolve latest: %v", err)
	}
	if resolved.Version != "v9" {
		t.Fatalf("latest resolved to %q, want v9 (published last)", resolved.Version)
	}
}

func TestResolveMissingAliasAndVersion(t *testing.T) {
	reg := NewMemoryRegistry()
	ctx := context.Background()
	publishStorageState(t, reg, "crm", "v1", cookieJar)

	if _, err := reg.Resolve(ctx, AuthProfileRef{Alias: "nope", Version: VersionLatest}); !IsFailure(err, FailureProfileNotFound) {
		t.Fatalf("missing alias returned %v, want %s", err, FailureProfileNotFound)
	}
	if _, err := reg.Resolve(ctx, AuthProfileRef{Alias: "crm", Version: "v99"}); !IsFailure(err, FailureProfileNotFound) {
		t.Fatalf("missing version returned %v, want %s", err, FailureProfileNotFound)
	}
}

func TestRevokedProfileIsTerminal(t *testing.T) {
	reg := NewMemoryRegistry()
	ctx := context.Background()
	publishStorageState(t, reg, "crm", "v1", cookieJar)

	if err := reg.Revoke(ctx, AuthProfileRef{Alias: "crm", Version: "v1"}, "credential rotation"); err != nil {
		t.Fatalf("Revoke: %v", err)
	}

	_, err := reg.Resolve(ctx, AuthProfileRef{Alias: "crm", Version: "v1"})
	if !IsFailure(err, FailureProfileRevoked) {
		t.Fatalf("resolving a revoked profile returned %v, want %s", err, FailureProfileRevoked)
	}

	// Opening the private record must fail too — revocation is not cosmetic.
	if _, err := reg.Open(ctx, AuthProfileRef{Alias: "crm", Version: "v1"}); !IsFailure(err, FailureProfileRevoked) {
		t.Fatalf("opening a revoked record returned %v, want %s", err, FailureProfileRevoked)
	}

	// latest must skip the revoked version rather than resolving to it.
	if _, err := reg.Resolve(ctx, AuthProfileRef{Alias: "crm", Version: VersionLatest}); !IsFailure(err, FailureProfileNotFound) {
		t.Fatalf("latest over an all-revoked alias returned %v, want %s", err, FailureProfileNotFound)
	}
}

func TestExpiredProfileIsTerminal(t *testing.T) {
	reg := NewMemoryRegistry()
	ctx := context.Background()
	past := time.Now().Add(-time.Hour)

	policy := testPolicy()
	policy.ExpiresAt = past
	if _, err := reg.Publish(ctx, Record{
		Alias: "crm", Version: "v1", Provider: "local-playwright",
		Policy: policy, Material: NewStorageStateMaterial([]byte(cookieJar)),
	}); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	if _, err := reg.Resolve(ctx, AuthProfileRef{Alias: "crm", Version: "v1"}); !IsFailure(err, FailureProfileExpired) {
		t.Fatalf("resolving an expired profile returned %v, want %s", err, FailureProfileExpired)
	}
}

func TestRetireKeepsTheVersionResolvableButOutOfLatest(t *testing.T) {
	reg := NewMemoryRegistry()
	ctx := context.Background()
	publishStorageState(t, reg, "crm", "v1", cookieJar)
	publishStorageState(t, reg, "crm", "v2", cookieJar+"2")

	if err := reg.Retire(ctx, AuthProfileRef{Alias: "crm", Version: "v2"}); err != nil {
		t.Fatalf("Retire: %v", err)
	}

	// A pinned reference to a retired version still works — rollback needs it.
	if _, err := reg.Resolve(ctx, AuthProfileRef{Alias: "crm", Version: "v2"}); err != nil {
		t.Fatalf("resolving a retired pinned version failed: %v", err)
	}
	resolved, err := reg.Resolve(ctx, AuthProfileRef{Alias: "crm", Version: VersionLatest})
	if err != nil {
		t.Fatalf("Resolve latest: %v", err)
	}
	if resolved.Version != "v1" {
		t.Fatalf("latest = %q, want v1 after v2 was retired", resolved.Version)
	}
}

func TestPublishRecordsARollbackPointer(t *testing.T) {
	reg := NewMemoryRegistry()
	publishStorageState(t, reg, "crm", "v1", cookieJar)
	second := publishStorageState(t, reg, "crm", "v2", cookieJar+"2")

	if second.PreviousVersion != "v1" {
		t.Fatalf("PreviousVersion = %q, want v1", second.PreviousVersion)
	}
}

func TestOpenReturnsMaterialAndResolveDoesNot(t *testing.T) {
	reg := NewMemoryRegistry()
	ctx := context.Background()
	publishStorageState(t, reg, "crm", "v1", cookieJar)

	record, err := reg.Open(ctx, AuthProfileRef{Alias: "crm", Version: "v1"})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	_, state, _ := record.Material.Materialize()
	if string(state) != cookieJar {
		t.Fatalf("Open did not return the canonical material")
	}

	// SafeProfile is what leaves the package boundary; it must not carry bytes.
	profile, err := reg.Resolve(ctx, AuthProfileRef{Alias: "crm", Version: "v1"})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	encoded, err := json.Marshal(profile)
	if err != nil {
		t.Fatalf("marshal SafeProfile: %v", err)
	}
	for _, marker := range leakMarkers {
		if strings.Contains(string(encoded), marker) {
			t.Fatalf("SafeProfile JSON leaked %q: %s", marker, encoded)
		}
	}
	if strings.Contains(string(encoded), VersionLatest) {
		t.Fatalf("SafeProfile JSON contains %q: %s", VersionLatest, encoded)
	}
}

func TestListReturnsPublishOrderAndNoMaterial(t *testing.T) {
	reg := NewMemoryRegistry()
	ctx := context.Background()
	publishStorageState(t, reg, "crm", "v1", cookieJar)
	publishStorageState(t, reg, "crm", "v2", cookieJar+"2")
	publishStorageState(t, reg, "other", "v1", cookieJar+"o")

	profiles, err := reg.List(ctx, "crm")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(profiles) != 2 {
		t.Fatalf("List returned %d profiles, want 2", len(profiles))
	}
	if profiles[0].Version != "v1" || profiles[1].Version != "v2" {
		t.Fatalf("List order = %s,%s want v1,v2", profiles[0].Version, profiles[1].Version)
	}
}

func TestMemoryRegistryIsConcurrencySafe(t *testing.T) {
	reg := NewMemoryRegistry()
	ctx := context.Background()
	publishStorageState(t, reg, "crm", "v1", cookieJar)

	done := make(chan struct{})
	for range 8 {
		go func() {
			defer func() { done <- struct{}{} }()
			for range 50 {
				if _, err := reg.Resolve(ctx, AuthProfileRef{Alias: "crm", Version: VersionLatest}); err != nil {
					t.Errorf("Resolve: %v", err)
					return
				}
			}
		}()
	}
	for range 8 {
		<-done
	}
}
