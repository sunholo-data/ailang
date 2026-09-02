package ai

import "testing"

func TestResolveAnthropicCredential(t *testing.T) {
	tests := []struct {
		name      string
		apiKey    string
		authToken string
		wantValue string
		wantOAuth bool
		wantErr   bool
		why       string
	}{
		{
			name: "neither set is an error, not a default lane",
			// A silent default here would pick who gets billed.
			wantErr: true,
			why:     "CLAUDE.md §2: a fallback affecting billing must fail loudly",
		},
		{
			name: "api key alone selects the metered lane", apiKey: "sk-ant-x",
			wantValue: "sk-ant-x", wantOAuth: false,
		},
		{
			name: "auth token alone selects the OAuth lane", authToken: "oauth-tok",
			wantValue: "oauth-tok", wantOAuth: true,
		},
		{
			name:   "both set: the metered key wins",
			apiKey: "sk-ant-x", authToken: "oauth-tok",
			wantValue: "sk-ant-x", wantOAuth: false,
			why: "matches the Anthropic SDKs and the claude CLI; the surprise is real, which is why Lane() is loggable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(EnvAnthropicAPIKey, tt.apiKey)
			t.Setenv(EnvAnthropicAuthToken, tt.authToken)

			got, err := ResolveAnthropicCredential()
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected an error when no credential is set (%s)", tt.why)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Value != tt.wantValue {
				t.Errorf("Value = %q, want %q", got.Value, tt.wantValue)
			}
			if got.OAuth != tt.wantOAuth {
				t.Errorf("OAuth = %v, want %v (%s)", got.OAuth, tt.wantOAuth, tt.why)
			}
		})
	}
}

// The classifier and the client must never disagree about which lane ran —
// a metered label on a subscription run invents spend that never happened.
func TestAnthropicLaneIsOAuth_AgreesWithResolver(t *testing.T) {
	cases := []struct{ apiKey, authToken string }{
		{"", ""},
		{"sk-ant-x", ""},
		{"", "oauth-tok"},
		{"sk-ant-x", "oauth-tok"},
	}
	for _, c := range cases {
		t.Setenv(EnvAnthropicAPIKey, c.apiKey)
		t.Setenv(EnvAnthropicAuthToken, c.authToken)

		cred, err := ResolveAnthropicCredential()
		want := err == nil && cred.OAuth
		if got := AnthropicLaneIsOAuth(); got != want {
			t.Errorf("apiKey=%q authToken=%q: AnthropicLaneIsOAuth()=%v, resolver says OAuth=%v",
				c.apiKey, c.authToken, got, want)
		}
	}
}

func TestAnthropicCredential_LaneNeverLeaksTheSecret(t *testing.T) {
	for _, c := range []AnthropicCredential{
		{Value: "sk-ant-supersecret", OAuth: false},
		{Value: "oauth-supersecret", OAuth: true},
	} {
		if lane := c.Lane(); lane == "" || lane == c.Value {
			t.Errorf("Lane() = %q must be a safe-to-log label, not the credential", lane)
		}
	}
}
