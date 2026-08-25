package browser

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/browser/auth"
)

// The auth package duplicates Redacted rather than importing it, to keep the
// dependency one-way. This test is the reason that duplication is safe.
func TestRedactedConstantsStayIdentical(t *testing.T) {
	if Redacted != auth.Redacted {
		t.Fatalf("browser.Redacted = %q but auth.Redacted = %q", Redacted, auth.Redacted)
	}
}

// TestManifestCarriesSafeProfileIdentityOnly is the M1 criterion: a manifest
// built from a real authenticated run identifies which profile ran without
// carrying anything that could impersonate the account.
func TestManifestCarriesSafeProfileIdentityOnly(t *testing.T) {
	manifest := BrowserRunManifest{
		RunID:              "run-42",
		Provider:           "browserbase",
		ProviderSessionID:  "sess-7",
		ToolSurface:        "playwright-mcp",
		ProfileHash:        "sha256:0f1e2d3c4b5a6978",
		AuthProfileAlias:   "crm-readonly-eu",
		AuthProfileVersion: "v7",
		AuthLeaseID:        "lease-01HZX",
		AuthMode:           "read",
		Termination:        TerminationCompleted,
	}

	encoded, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	rendered := string(encoded)

	// The identity an operator needs must survive.
	for _, want := range []string{"crm-readonly-eu", "v7", "lease-01HZX", "sha256:0f1e2d3c4b5a6978", `"auth_mode":"read"`} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("manifest JSON dropped safe identity %q: %s", want, rendered)
		}
	}

	// Everything that could impersonate the account must not.
	for _, forbidden := range []string{
		auth.VersionLatest,
		"ctx_",
		"sid=",
		"access_token",
		"storage_state",
		"BEGIN PRIVATE KEY",
	} {
		if strings.Contains(rendered, forbidden) {
			t.Fatalf("manifest JSON leaked %q: %s", forbidden, rendered)
		}
	}
}

// A manifest must never be able to record "latest": the resolved version is what
// ran, and "latest" would make the result unreproducible.
func TestResolvedProfileRefIsNeverLatest(t *testing.T) {
	profile := auth.SafeProfile{Alias: "crm-readonly-eu", Version: "v7"}
	spec := SessionSpec{ProfileRef: profile.Ref().String()}
	if strings.Contains(spec.ProfileRef, auth.VersionLatest) {
		t.Fatalf("SessionSpec.ProfileRef = %q, which pins nothing", spec.ProfileRef)
	}
	if spec.ProfileRef != "crm-readonly-eu@v7" {
		t.Fatalf("SessionSpec.ProfileRef = %q, want crm-readonly-eu@v7", spec.ProfileRef)
	}
}

// TestSanitizeDiagnosticsRedactsAuthProfileMaterial covers the keys
// M-BROWSER-AUTH-PROFILES added, at the nesting depth a provider diagnostic
// actually arrives at.
func TestSanitizeDiagnosticsRedactsAuthProfileMaterial(t *testing.T) {
	diagnostics := map[string]any{
		"provider": "browserbase",
		"profile": map[string]any{
			"alias":         "crm-readonly-eu",
			"version":       "v7",
			"context_id":    "ctx_01HZX9QK4M8N2P7R3T5V6W8Y0Z",
			"storage_state": `{"cookies":[{"name":"sid","value":"SUPER-SECRET"}]}`,
			"sealed_blob":   "gAAAAABm...",
			"material":      "raw bytes",
		},
		"attempts": []any{
			map[string]any{
				"local_storage": "access_token=eyJLEAKME",
				"passkey":       "-----BEGIN PRIVATE KEY-----",
				"recovery_code": "8842-1193",
				"credential":    "hunter2",
				"duration_ms":   1200,
			},
		},
	}

	sanitized := SanitizeDiagnostics(diagnostics)
	encoded, err := json.Marshal(sanitized)
	if err != nil {
		t.Fatalf("marshal sanitized diagnostics: %v", err)
	}
	rendered := string(encoded)

	for _, forbidden := range []string{
		"ctx_01HZX9QK4M8N2P7R3T5V6W8Y0Z",
		"SUPER-SECRET",
		"gAAAAABm",
		"eyJLEAKME",
		"BEGIN PRIVATE KEY",
		"8842-1193",
		"hunter2",
		"raw bytes",
	} {
		if strings.Contains(rendered, forbidden) {
			t.Fatalf("SanitizeDiagnostics leaked %q: %s", forbidden, rendered)
		}
	}

	// Redaction must not swallow the safe fields that make a diagnostic useful.
	for _, want := range []string{"crm-readonly-eu", "v7", "browserbase", "1200"} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("SanitizeDiagnostics over-redacted %q: %s", want, rendered)
		}
	}
}

// The auth failure vocabulary must not collide with the browser one: an eval
// query for a browser_* category must not accidentally match a browser_auth_*
// row, and neither set may be silently renamed.
func TestAuthFailureCategoriesDoNotCollideWithBrowserCategories(t *testing.T) {
	browserCategories := []FailureCategory{
		FailurePolicyDenied, FailureCapacityExhausted, FailureProvision,
		FailureConnect, FailureActionTimeout, FailureSessionTimeout,
		FailureRemoteDisconnected, FailureArtifactExport, FailureCleanup,
		FailureCostUnknown,
	}

	existing := make(map[string]bool, len(browserCategories))
	for _, category := range browserCategories {
		existing[string(category)] = true
	}

	for _, authCategory := range auth.AllFailureCategories() {
		if existing[string(authCategory)] {
			t.Fatalf("auth category %q collides with a browser category", authCategory)
		}
		if !strings.HasPrefix(string(authCategory), "browser_auth_") {
			t.Fatalf("auth category %q does not use the browser_auth_ prefix", authCategory)
		}
	}
}
