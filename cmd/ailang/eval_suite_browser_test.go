package main

import "testing"

func TestValidateBrowserEvalFlags(t *testing.T) {
	for _, tc := range []struct {
		agent    bool
		provider string
		wantErr  bool
	}{
		{false, "", false},
		{true, "", false},
		{true, "local-playwright", false},
		{true, "browserbase", false},
		{false, "local-playwright", true},
		{true, "unknown", true},
	} {
		if got := validateBrowserEvalFlags(tc.agent, tc.provider); (got != nil) != tc.wantErr {
			t.Errorf("agent=%v provider=%q error=%v wantErr=%v", tc.agent, tc.provider, got, tc.wantErr)
		}
	}
}
