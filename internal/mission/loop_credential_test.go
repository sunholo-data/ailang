package mission

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"
)

// The loops' own setup-token must never be tried before the login credential: it is
// FORBIDDEN from the usage endpoint (HTTP 403, measured 2026-09-22).
func TestOrderAnthropicTokens_LoginBeforeEnv(t *testing.T) {
	for _, c := range []struct {
		login, env string
		want       []string
	}{
		{"login", "setup", []string{"login", "setup"}},
		{"", "setup", []string{"setup"}}, // cloud container: env is all there is
		{"login", "", []string{"login"}},
		{"same", "same", []string{"same"}},
		{"", "", nil},
	} {
		if got := orderAnthropicTokens(c.login, c.env); !reflect.DeepEqual(got, c.want) {
			t.Errorf("orderAnthropicTokens(%q, %q) = %v, want %v", c.login, c.env, got, c.want)
		}
	}
}

func TestAnthropicOAuthTokens_ExplicitlyEmptyEnvMeansNoCredential(t *testing.T) {
	t.Setenv("CLAUDE_CODE_OAUTH_TOKEN", "")
	if got := anthropicOAuthTokens(t.Context()); got != nil {
		t.Fatalf("anthropicOAuthTokens = %v, want nil for an explicitly empty env var", got)
	}
}

// tokenServer answers 200 for the login token and 403 for the setup-token, the way the real
// endpoint does.
func tokenServer(t *testing.T, okToken, body string) *http.Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+okToken {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	client := srv.Client()
	client.Transport = rewriteHost{base: srv.URL, rt: client.Transport}
	return client
}

const withinRationBody = `{"limits": [
   {"kind": "weekly_all", "group": "weekly", "percent": 5, "severity": "normal",
    "resets_at": "2026-09-14T05:00:00Z", "scope": null, "is_active": false}]}`

func TestObserveAnthropicQuotaWith_LoginReadsUsageWhileSetupTokenIsSet(t *testing.T) {
	t.Setenv("AILANG_ANTHROPIC_RATION", "1")
	client := tokenServer(t, "login", withinRationBody)
	o := observeAnthropicQuotaWith([]string{"login", "setup"}, anthropicNow(), client)
	if o.State != "ok" {
		t.Fatalf("state = %q (%s), want ok from the login credential", o.State, o.Reason)
	}
}

// A stale login falls through to the env token; when both fail, the reason names both.
func TestObserveAnthropicQuotaWith_FallsThroughAndNamesBothFailures(t *testing.T) {
	t.Setenv("AILANG_ANTHROPIC_RATION", "1")
	client := tokenServer(t, "setup", withinRationBody)
	if o := observeAnthropicQuotaWith([]string{"stale", "setup"}, anthropicNow(), client); o.State != "ok" {
		t.Fatalf("state = %q (%s), want ok from the second token", o.State, o.Reason)
	}
	client = tokenServer(t, "nobody", withinRationBody)
	o := observeAnthropicQuotaWith([]string{"stale", "setup"}, anthropicNow(), client)
	if o.State != "unknown" || strings.Count(o.Reason, "HTTP 403") != 2 ||
		!strings.Contains(o.Reason, "then with CLAUDE_CODE_OAUTH_TOKEN") {
		t.Fatalf("state=%q reason=%q, want unknown naming both attempts", o.State, o.Reason)
	}
}

// The 2026-10-07 deadlock: an idle Codex window reads 0% used with resets_at = now + 7d, so
// its pace allowance is 0pp, and a 2pp margin refused it forever.
func TestCodexMargin_IdleWindowIsAdmitted(t *testing.T) {
	now := time.Date(2026, 10, 7, 15, 36, 41, 0, time.UTC)
	o := CodexQuotaObservation{ObservedAt: now, Windows: []CodexQuotaWindow{
		{UsedPercent: 0, WindowMinutes: 7 * 24 * 60, ResetsAt: now.Add(7*24*time.Hour + time.Second)},
	}}
	o.evaluate(now)
	if o.Blocked() || o.HeadroomShort {
		t.Fatalf("state=%q (%s), want an unused window admitted", o.State, o.Reason)
	}
	// One point in, the margin applies again.
	o.Windows[0].UsedPercent = 1
	o.evaluate(now)
	if !o.Blocked() {
		t.Fatalf("state=%q (%s), want a used window over its 0pp allowance blocked", o.State, o.Reason)
	}
}
