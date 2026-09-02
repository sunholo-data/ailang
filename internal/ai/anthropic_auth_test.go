package ai

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/testutil"
)

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
			// Isolate HOME: the resolver now falls back to the real
			// ~/.claude/.credentials.json, which would make "no credential"
			// cases pass or fail depending on whose machine runs the suite.
			testutil.SetHomeDir(t, t.TempDir())
			t.Setenv(EnvClaudeCodeOAuthToken, "")
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
		testutil.SetHomeDir(t, t.TempDir())
		t.Setenv(EnvClaudeCodeOAuthToken, "")
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

// The credential FILE is what makes standard mode work with no setup wherever
// agent mode already works — the `claude` CLI reads this same file.
func TestResolveAnthropicCredential_FallsBackToClaudeCredentialsFile(t *testing.T) {
	write := func(t *testing.T, body string) {
		t.Helper()
		home := t.TempDir()
		testutil.SetHomeDir(t, home)
		t.Setenv(EnvAnthropicAPIKey, "")
		t.Setenv(EnvAnthropicAuthToken, "")
		t.Setenv(EnvClaudeCodeOAuthToken, "")
		dir := filepath.Join(home, ".claude")
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, ".credentials.json"), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	future := time.Now().Add(time.Hour).UnixMilli()
	past := time.Now().Add(-time.Hour).UnixMilli()

	t.Run("live token resolves as OAuth", func(t *testing.T) {
		write(t, fmt.Sprintf(`{"claudeAiOauth":{"accessToken":"tok","expiresAt":%d}}`, future))
		got, err := ResolveAnthropicCredential()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Value != "tok" || !got.OAuth {
			t.Errorf("got %+v, want the file's accessToken on the OAuth lane", got)
		}
	})

	// An expired token must fail HERE, not as an opaque 401 that banks every
	// benchmark in the run as api_error for an invisible cause.
	t.Run("expired token errors with a refresh hint", func(t *testing.T) {
		write(t, fmt.Sprintf(`{"claudeAiOauth":{"accessToken":"tok","expiresAt":%d}}`, past))
		_, err := ResolveAnthropicCredential()
		if err == nil {
			t.Fatal("an expired token must not be sent")
		}
		if !strings.Contains(err.Error(), "expired") || !strings.Contains(err.Error(), "claude") {
			t.Errorf("error should name the expiry and how to refresh; got %v", err)
		}
	})

	t.Run("an explicit env credential still outranks the file", func(t *testing.T) {
		write(t, fmt.Sprintf(`{"claudeAiOauth":{"accessToken":"file-tok","expiresAt":%d}}`, future))
		t.Setenv(EnvAnthropicAPIKey, "sk-ant-x")
		got, err := ResolveAnthropicCredential()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Value != "sk-ant-x" || got.OAuth {
			t.Errorf("got %+v, want the metered env key to win", got)
		}
	})
}
