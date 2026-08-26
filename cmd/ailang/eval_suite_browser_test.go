package main

import "testing"

func TestValidateBrowserEvalFlags(t *testing.T) {
	for _, tc := range []struct {
		name     string
		agent    bool
		provider string
		profile  string
		wantErr  bool
	}{
		{name: "disabled", agent: false, provider: "", wantErr: false},
		{name: "agent without browser", agent: true, provider: "", wantErr: false},
		{name: "local", agent: true, provider: "local-playwright", wantErr: false},
		{name: "browserbase", agent: true, provider: "browserbase", wantErr: false},
		{name: "browser needs agent", agent: false, provider: "local-playwright", wantErr: true},
		{name: "unknown provider", agent: true, provider: "unknown", wantErr: true},

		// A profile is meaningless without a provider to attach it to, and a
		// malformed reference must fail before any model is billed.
		{name: "profile without provider", agent: false, provider: "", profile: "crm@v1", wantErr: true},
		{name: "profile with provider", agent: true, provider: "local-playwright", profile: "crm@v1", wantErr: false},
		{name: "profile latest", agent: true, provider: "local-playwright", profile: "crm@latest", wantErr: false},
		{name: "bare alias means latest", agent: true, provider: "local-playwright", profile: "crm", wantErr: false},
		{name: "malformed profile", agent: true, provider: "local-playwright", profile: "Crm@@v1", wantErr: true},
		{name: "path-escaping profile", agent: true, provider: "local-playwright", profile: "../escape@v1", wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := validateBrowserEvalFlags(tc.agent, tc.provider, tc.profile)
			if (got != nil) != tc.wantErr {
				t.Errorf("agent=%v provider=%q profile=%q error=%v wantErr=%v",
					tc.agent, tc.provider, tc.profile, got, tc.wantErr)
			}
		})
	}
}
