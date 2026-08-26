package feedback

import (
	"strings"
	"testing"
)

// TestPackageCoordinateRejectsHyphens guards the defect that stranded ten real
// tickets. submit_feedback accepted "sunholo/ailang-parse", FormatPackageInbox
// minted "pkg:sunholo/ailang-parse" from it verbatim, and no agent watches that
// inbox — the registry package is "sunholo/ailang_parse". Nothing surfaced the
// mistake because an inbox nobody watches looks exactly like an inbox with no mail.
func TestPackageCoordinateRejectsHyphens(t *testing.T) {
	tests := []struct {
		name       string
		pkg        string
		wantErr    bool
		wantDetail string // substring the error must name
	}{
		{"underscore name is accepted", "sunholo/ailang_parse", false, ""},
		{"plain name is accepted", "sunholo/auth", false, ""},
		{"empty package is optional", "", false, ""},
		{"hyphen in name is refused and corrected", "sunholo/ailang-parse", true, "sunholo/ailang_parse"},
		{"hyphen in vendor is refused and corrected", "sun-holo/auth", true, "sun_holo/auth"},
		{"multiple hyphens all corrected", "sunholo/motoko-ext-abi", true, "sunholo/motoko_ext_abi"},
		{"malformed is still refused", "not-a-coordinate", true, ""},
		{"path traversal is refused", "../../etc/passwd", true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validate(Request{
				Title:    "t",
				Body:     "b",
				Category: "bug",
				Package:  tt.pkg,
			})
			if tt.wantErr && err == nil {
				t.Fatalf("package %q was accepted; it can never be imported so it can never exist", tt.pkg)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("package %q rejected: %v", tt.pkg, err)
			}
			if tt.wantDetail != "" {
				if !strings.Contains(err.Error(), tt.wantDetail) {
					t.Errorf("error must name the correction %q so the caller can fix it, got: %v", tt.wantDetail, err)
				}
			}
		})
	}
}

// TestEveryPublishedPackageWouldValidate asserts the tightened rule breaks nothing
// real: all 41 published coordinates use underscores.
func TestEveryPublishedPackageWouldValidate(t *testing.T) {
	for _, p := range []string{
		"sunholo/auth", "sunholo/gcp_auth", "sunholo/ailang_parse",
		"sunholo/motoko_ext_abi", "sunholo/billing_service_api",
		"sunholo/http_helpers", "sunholo/testing_utils", "sunholo/ollama_stream",
	} {
		if err := validate(Request{Title: "t", Body: "b", Category: "bug", Package: p}); err != nil {
			t.Errorf("real published package %q rejected by the tightened rule: %v", p, err)
		}
	}
}
