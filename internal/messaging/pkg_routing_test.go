package messaging

import "testing"

func TestParseInboxAddress(t *testing.T) {
	tests := []struct {
		input    string
		wantType InboxAddressType
		wantName string
	}{
		{"pkg:sunholo/auth", InboxAddrPackage, "sunholo/auth"},
		{"pkg:sunholo/http-helpers", InboxAddrPackage, "sunholo/http-helpers"},
		{"workspace:docparse", InboxAddrWorkspace, "docparse"},
		{"workspace:web-api-demo", InboxAddrWorkspace, "web-api-demo"},
		{"team:registry-admin", InboxAddrTeam, "registry-admin"},
		{"team:core", InboxAddrTeam, "core"},
		{"user", InboxAddrPlain, "user"},
		{"coordinator", InboxAddrPlain, "coordinator"},
		{"", InboxAddrPlain, ""},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			addr := ParseInboxAddress(tc.input)
			if addr.Type != tc.wantType {
				t.Errorf("type: got %q, want %q", addr.Type, tc.wantType)
			}
			if addr.Name != tc.wantName {
				t.Errorf("name: got %q, want %q", addr.Name, tc.wantName)
			}
			if addr.Raw != tc.input {
				t.Errorf("raw: got %q, want %q", addr.Raw, tc.input)
			}
		})
	}
}

func TestFormatInboxAddresses(t *testing.T) {
	if got := FormatPackageInbox("sunholo/auth"); got != "pkg:sunholo/auth" {
		t.Errorf("FormatPackageInbox: got %q", got)
	}
	if got := FormatWorkspaceInbox("docparse"); got != "workspace:docparse" {
		t.Errorf("FormatWorkspaceInbox: got %q", got)
	}
	if got := FormatTeamInbox("core"); got != "team:core" {
		t.Errorf("FormatTeamInbox: got %q", got)
	}
}

func TestParseInboxAddressRoundTrip(t *testing.T) {
	// Format then parse should be identity
	pkgAddr := FormatPackageInbox("sunholo/auth")
	parsed := ParseInboxAddress(pkgAddr)
	if parsed.Type != InboxAddrPackage || parsed.Name != "sunholo/auth" {
		t.Errorf("round-trip failed for package: %+v", parsed)
	}

	wsAddr := FormatWorkspaceInbox("docparse")
	parsed = ParseInboxAddress(wsAddr)
	if parsed.Type != InboxAddrWorkspace || parsed.Name != "docparse" {
		t.Errorf("round-trip failed for workspace: %+v", parsed)
	}
}
