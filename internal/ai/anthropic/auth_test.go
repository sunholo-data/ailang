package anthropic

import (
	"net/http"
	"testing"
)

// The header shape IS the credential type: Anthropic selects how to interpret
// the secret from which header carries it. An OAuth token sent as x-api-key is
// a 401, so these assertions are the contract, not style.
func TestApplyAuthHeaders_ModeSelectsHeaderShape(t *testing.T) {
	tests := []struct {
		name        string
		opts        []ClientOption
		wantAPIKey  string
		wantBearer  string
		wantBeta    string
		description string
	}{
		{
			name:        "default is metered x-api-key",
			opts:        nil,
			wantAPIKey:  "secret",
			description: "an unconfigured client must keep billing exactly as before",
		},
		{
			name:        "WithOAuth sends Bearer plus the oauth beta",
			opts:        []ClientOption{WithOAuth()},
			wantBearer:  "Bearer secret",
			wantBeta:    oauthBeta,
			description: "the beta flag is required; without it the token is rejected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewClient("secret", tt.opts...)
			h := http.Header{}
			c.applyAuthHeaders(h)

			if got := h.Get("x-api-key"); got != tt.wantAPIKey {
				t.Errorf("x-api-key = %q, want %q (%s)", got, tt.wantAPIKey, tt.description)
			}
			if got := h.Get("Authorization"); got != tt.wantBearer {
				t.Errorf("Authorization = %q, want %q (%s)", got, tt.wantBearer, tt.description)
			}
			if got := h.Get("anthropic-beta"); got != tt.wantBeta {
				t.Errorf("anthropic-beta = %q, want %q", got, tt.wantBeta)
			}
			// Version is mode-independent and every path needs it.
			if got := h.Get("anthropic-version"); got != defaultAPIVersion {
				t.Errorf("anthropic-version = %q, want %q", got, defaultAPIVersion)
			}
		})
	}
}

// The two lanes must never both be present: sending an API key AND a Bearer
// token lets the server pick, which decides billing by accident.
func TestApplyAuthHeaders_LanesAreMutuallyExclusive(t *testing.T) {
	oauth := http.Header{}
	NewClient("t", WithOAuth()).applyAuthHeaders(oauth)
	if oauth.Get("x-api-key") != "" {
		t.Error("OAuth client must not also send x-api-key — the server would choose the lane, and with it who gets billed")
	}

	keyed := http.Header{}
	NewClient("k").applyAuthHeaders(keyed)
	if keyed.Get("Authorization") != "" {
		t.Error("API-key client must not send Authorization")
	}
	if keyed.Get("anthropic-beta") != "" {
		t.Error("API-key client must not send the oauth beta flag")
	}
}

// AuthMode is what cost accounting branches on, so it must report the truth.
func TestAuthMode_IsReported(t *testing.T) {
	if got := NewClient("k").AuthMode(); got != AuthAPIKey {
		t.Errorf("default AuthMode = %v, want AuthAPIKey", got)
	}
	if got := NewClient("t", WithOAuth()).AuthMode(); got != AuthOAuth {
		t.Errorf("WithOAuth AuthMode = %v, want AuthOAuth", got)
	}
}
