package auth

import (
	"errors"
	"testing"
	"time"
)

func TestFailureCategoriesAreCompleteAndUnique(t *testing.T) {
	// The ten categories the design document reserved. A category that changes
	// spelling silently breaks every downstream eval query, so they are pinned
	// here as literals rather than derived from the constants.
	want := []string{
		"browser_auth_profile_not_found",
		"browser_auth_profile_expired",
		"browser_auth_profile_revoked",
		"browser_auth_lease_conflict",
		"browser_auth_scope_denied",
		"browser_auth_refresh_required",
		"browser_auth_materialize_failed",
		"browser_auth_writeback_denied",
		"browser_auth_artifact_policy_denied",
		"browser_auth_cleanup_failed",
	}

	got := AllFailureCategories()
	if len(got) != len(want) {
		t.Fatalf("AllFailureCategories() has %d entries, want %d", len(got), len(want))
	}

	seen := make(map[FailureCategory]bool, len(got))
	for _, category := range got {
		if seen[category] {
			t.Fatalf("duplicate failure category %q", category)
		}
		seen[category] = true
	}
	for _, expected := range want {
		if !seen[FailureCategory(expected)] {
			t.Fatalf("missing failure category %q", expected)
		}
	}
}

func TestFailureNeverPrintsTheUnderlyingCause(t *testing.T) {
	cause := errors.New("vault returned password hunter2 for user admin@example.com")
	failure := NewFailure(FailureMaterializeFailed, "open_envelope", cause)

	message := failure.Error()
	if got := "hunter2"; contains(message, got) {
		t.Fatalf("Failure.Error() leaked the cause: %s", message)
	}
	if !contains(message, string(FailureMaterializeFailed)) {
		t.Fatalf("Failure.Error() = %q, want the category", message)
	}
	if !contains(message, "open_envelope") {
		t.Fatalf("Failure.Error() = %q, want the op", message)
	}
}

func TestIsFailureMatchesThroughWrapping(t *testing.T) {
	failure := NewFailure(FailureLeaseConflict, "acquire", nil)
	wrapped := errWrap{inner: failure}

	if !IsFailure(failure, FailureLeaseConflict) {
		t.Fatalf("IsFailure did not match the direct failure")
	}
	if !IsFailure(wrapped, FailureLeaseConflict) {
		t.Fatalf("IsFailure did not match through Unwrap")
	}
	if IsFailure(wrapped, FailureScopeDenied) {
		t.Fatalf("IsFailure matched the wrong category")
	}
	if IsFailure(nil, FailureLeaseConflict) {
		t.Fatalf("IsFailure matched a nil error")
	}
}

type errWrap struct{ inner error }

func (e errWrap) Error() string { return "wrapped: " + e.inner.Error() }
func (e errWrap) Unwrap() error { return e.inner }

func TestParseRef(t *testing.T) {
	cases := []struct {
		in          string
		wantAlias   string
		wantVersion string
		wantErr     bool
	}{
		{in: "crm-readonly-eu", wantAlias: "crm-readonly-eu", wantVersion: VersionLatest},
		{in: "crm-readonly-eu@latest", wantAlias: "crm-readonly-eu", wantVersion: VersionLatest},
		{in: "dashboard-readonly@v7", wantAlias: "dashboard-readonly", wantVersion: "v7"},
		{in: "shop.test_1@v12", wantAlias: "shop.test_1", wantVersion: "v12"},
		{in: "", wantErr: true},
		{in: "@v7", wantErr: true},
		{in: "alias@", wantErr: true},
		{in: "alias@v1@v2", wantErr: true},
		{in: "Alias-With-Caps@v1", wantErr: true},
		{in: "alias with space@v1", wantErr: true},
		{in: "../escape@v1", wantErr: true},
		{in: "alias@../escape", wantErr: true},
	}

	for _, tc := range cases {
		ref, err := ParseRef(tc.in)
		if tc.wantErr {
			if err == nil {
				t.Fatalf("ParseRef(%q) = %+v, want error", tc.in, ref)
			}
			continue
		}
		if err != nil {
			t.Fatalf("ParseRef(%q): %v", tc.in, err)
		}
		if ref.Alias != tc.wantAlias || ref.Version != tc.wantVersion {
			t.Fatalf("ParseRef(%q) = %s@%s, want %s@%s", tc.in, ref.Alias, ref.Version, tc.wantAlias, tc.wantVersion)
		}
	}
}

func TestRefStringAndIsLatest(t *testing.T) {
	ref := AuthProfileRef{Alias: "crm", Version: VersionLatest}
	if !ref.IsLatest() {
		t.Fatalf("IsLatest() = false for %s", ref)
	}
	if got, want := ref.String(), "crm@latest"; got != want {
		t.Fatalf("String() = %q, want %q", got, want)
	}

	pinned := AuthProfileRef{Alias: "crm", Version: "v7"}
	if pinned.IsLatest() {
		t.Fatalf("IsLatest() = true for a pinned ref")
	}
	if got, want := pinned.String(), "crm@v7"; got != want {
		t.Fatalf("String() = %q, want %q", got, want)
	}
}

func TestNormalizeOriginRejectsAnythingButAnExactOrigin(t *testing.T) {
	cases := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{in: "https://crm.example.com", want: "https://crm.example.com"},
		{in: "https://CRM.Example.com", want: "https://crm.example.com"},
		{in: "https://crm.example.com:8443", want: "https://crm.example.com:8443"},
		{in: "https://crm.example.com:443", want: "https://crm.example.com"},
		{in: "http://localhost:3000", want: "http://localhost:3000"},
		{in: "https://crm.example.com/", want: "https://crm.example.com"},
		{in: "https://crm.example.com/inbox", wantErr: true},
		{in: "https://*.example.com", wantErr: true},
		{in: "*", wantErr: true},
		{in: "crm.example.com", wantErr: true},
		{in: "ftp://crm.example.com", wantErr: true},
		{in: "https://user:pass@crm.example.com", wantErr: true},
		{in: "https://crm.example.com?q=1", wantErr: true},
		{in: "", wantErr: true},
	}

	for _, tc := range cases {
		got, err := NormalizeOrigin(tc.in)
		if tc.wantErr {
			if err == nil {
				t.Fatalf("NormalizeOrigin(%q) = %q, want error", tc.in, got)
			}
			continue
		}
		if err != nil {
			t.Fatalf("NormalizeOrigin(%q): %v", tc.in, err)
		}
		if got != tc.want {
			t.Fatalf("NormalizeOrigin(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestPolicyAllowsOriginIsExactMatchOnly(t *testing.T) {
	policy := AuthProfilePolicy{AllowedOrigins: []string{"https://crm.example.com"}}

	if !policy.AllowsOrigin("https://crm.example.com/deals/17") {
		t.Fatalf("exact origin with a path was denied")
	}
	if policy.AllowsOrigin("https://evil.example.com") {
		t.Fatalf("a different host was allowed")
	}
	if policy.AllowsOrigin("http://crm.example.com") {
		t.Fatalf("a scheme downgrade was allowed")
	}
	if policy.AllowsOrigin("https://crm.example.com.evil.test") {
		t.Fatalf("a suffix-extension host was allowed")
	}
	if policy.AllowsOrigin("https://sub.crm.example.com") {
		t.Fatalf("a subdomain was allowed by an exact-origin policy")
	}

	empty := AuthProfilePolicy{}
	if empty.AllowsOrigin("https://crm.example.com") {
		t.Fatalf("a policy with no origins allowed a navigation — must fail closed")
	}
}

func TestPolicyArtifactsDenyByDefault(t *testing.T) {
	absent := AuthProfilePolicy{}
	if absent.ArtifactPolicyPresent() {
		t.Fatalf("nil AllowArtifacts reported a present artifact policy")
	}
	if absent.AllowsArtifact("screenshot") {
		t.Fatalf("nil AllowArtifacts allowed an artifact — must deny by default")
	}

	// An explicitly empty (non-nil) list is a real decision: allow nothing.
	explicitNone := AuthProfilePolicy{AllowArtifacts: []string{}}
	if !explicitNone.ArtifactPolicyPresent() {
		t.Fatalf("an explicit empty list is a present policy meaning allow-nothing")
	}
	if explicitNone.AllowsArtifact("screenshot") {
		t.Fatalf("an explicit allow-nothing policy allowed an artifact")
	}

	allowed := AuthProfilePolicy{AllowArtifacts: []string{"screenshot"}}
	if !allowed.AllowsArtifact("screenshot") {
		t.Fatalf("an allowed artifact class was denied")
	}
	if allowed.AllowsArtifact("video") {
		t.Fatalf("an unlisted artifact class was allowed")
	}
}

func TestEgressBoundaryFailsClosedWhenAbsent(t *testing.T) {
	if (AuthProfilePolicy{}).EgressBoundaryPresent() {
		t.Fatalf("an absent egress boundary reported present — must fail closed")
	}
	acknowledged := AuthProfilePolicy{EgressBoundary: EgressOperatorAcknowledged}
	if !acknowledged.EgressBoundaryPresent() {
		t.Fatalf("an operator-acknowledged boundary reported absent")
	}
	if acknowledged.EgressBoundaryEnforced() {
		t.Fatalf("operator acknowledgement must not claim enforcement")
	}
	enforced := AuthProfilePolicy{EgressBoundary: EgressEnforced}
	if !enforced.EgressBoundaryEnforced() {
		t.Fatalf("an enforced boundary reported unenforced")
	}
}

func TestAccountClassValidity(t *testing.T) {
	for _, valid := range []AccountClass{AccountReadonly, AccountMutable, AccountPrivileged} {
		if !valid.Valid() {
			t.Fatalf("%q reported invalid", valid)
		}
	}
	for _, invalid := range []AccountClass{"", "admin", "READONLY"} {
		if invalid.Valid() {
			t.Fatalf("%q reported valid", invalid)
		}
	}
}

func TestLeaseModeExclusivity(t *testing.T) {
	if LeaseRead.Exclusive() {
		t.Fatalf("read mode reported exclusive")
	}
	if !LeaseRefresh.Exclusive() {
		t.Fatalf("refresh mode reported non-exclusive")
	}
	if LeaseRead.Writes() {
		t.Fatalf("read mode reported as writing")
	}
	if !LeaseRefresh.Writes() {
		t.Fatalf("refresh mode reported as non-writing")
	}
}

func TestSafeProfileLifecycleStates(t *testing.T) {
	now := time.Date(2026, 8, 25, 12, 0, 0, 0, time.UTC)

	live := SafeProfile{Alias: "crm", Version: "v7", ExpiresAtOrZero: time.Time{}}
	if live.Revoked() {
		t.Fatalf("a fresh profile reported revoked")
	}
	if live.Expired(now) {
		t.Fatalf("a profile with no expiry reported expired")
	}

	expired := SafeProfile{Alias: "crm", Version: "v7", ExpiresAtOrZero: now.Add(-time.Second)}
	if !expired.Expired(now) {
		t.Fatalf("a past expiry reported live")
	}
	if expired.Expired(now.Add(-time.Hour)) {
		t.Fatalf("a future expiry reported expired")
	}

	revoked := SafeProfile{Alias: "crm", Version: "v7", RevokedAt: now, RevocationReason: "credential rotation"}
	if !revoked.Revoked() {
		t.Fatalf("a revoked profile reported live")
	}
}

func TestSafeProfileRefNeverResolvesToLatest(t *testing.T) {
	profile := SafeProfile{Alias: "crm", Version: "v7"}
	ref := profile.Ref()
	if ref.IsLatest() {
		t.Fatalf("a stored profile produced a latest ref")
	}
	if ref.Alias != "crm" || ref.Version != "v7" {
		t.Fatalf("Ref() = %s, want crm@v7", ref)
	}
}

func contains(haystack, needle string) bool {
	return len(needle) == 0 || (len(haystack) >= len(needle) && indexOf(haystack, needle) >= 0)
}

func indexOf(haystack, needle string) int {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}
