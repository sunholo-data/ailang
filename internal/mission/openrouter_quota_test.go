package mission

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// VERBATIM shape of GET /api/v1/key on 2026-09-08. A hand-written approximation is what a
// schema change slips past.
const liveOpenRouterPayload = `{"data":{"label":"sk-or-v1-a20...c62","is_management_key":false,
 "is_provisioning_key":false,"limit":100,"limit_reset":"monthly","limit_remaining":92.634926647,
 "include_byok_in_limit":false,"usage":224.808602387,"usage_daily":0.131735148,
 "usage_weekly":0.69418805,"usage_monthly":7.365073353,"byok_usage":0,"is_free_tier":false,
 "expires_at":null,"creator_user_id":"user_x","rate_limit":{"requests":-1,"interval":"10s"}}}`

func orNow() time.Time { return time.Date(2026, 9, 8, 15, 0, 0, 0, time.UTC) }

func orServer(t *testing.T, status int, body string) (*http.Client, func()) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Errorf("Authorization = %q", got)
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	c := srv.Client()
	c.Transport = rewriteHost{base: srv.URL, rt: c.Transport}
	return c, srv.Close
}

func TestOpenRouter_LivePayloadIsWithinRation(t *testing.T) {
	c, done := orServer(t, http.StatusOK, liveOpenRouterPayload)
	defer done()
	o := observeOpenRouterQuota("test-key", orNow(), c)
	if o.State != "ok" {
		t.Fatalf("state = %q (%s), want ok — $0.13 spent today", o.State, o.Reason)
	}
	// $100 * 0.70 / 30 = $2.3333
	if o.AllowanceUSD == nil || *o.AllowanceUSD < 2.33 || *o.AllowanceUSD > 2.34 {
		t.Errorf("allowance = %v, want ~2.333 (the weekly rule's 70%% spend share over 30 days)", o.AllowanceUSD)
	}
	if o.Blocked() {
		t.Error("Blocked() = true for a lane at 7.4% of its monthly cap")
	}
}

func TestOpenRouter_OverDailyRationBlocks(t *testing.T) {
	body := `{"data":{"limit":100,"limit_reset":"monthly","limit_remaining":50,"usage_daily":9.5,"usage_monthly":50}}`
	c, done := orServer(t, http.StatusOK, body)
	defer done()
	o := observeOpenRouterQuota("test-key", orNow(), c)
	if o.State != "over" || !o.Blocked() {
		t.Fatalf("state = %q, want over: $9.50 today against a $2.33/day ration", o.State)
	}
}

// An exhausted cap blocks even when today's spend is zero: remaining is the harder bound.
func TestOpenRouter_ExhaustedCapBlocksRegardlessOfDailyRate(t *testing.T) {
	body := `{"data":{"limit":100,"limit_reset":"monthly","limit_remaining":0,"usage_daily":0,"usage_monthly":100}}`
	c, done := orServer(t, http.StatusOK, body)
	defer done()
	o := observeOpenRouterQuota("test-key", orNow(), c)
	if o.State != "over" {
		t.Fatalf("state = %q, want over on an exhausted monthly cap", o.State)
	}
}

// No cap means no denominator. That is neither an error nor "ok" — the same call the ollama
// reader makes about a missing capacity: pace nothing, say so, fail closed.
func TestOpenRouter_UncappedKeyIsUnknownNotOK(t *testing.T) {
	body := `{"data":{"limit":null,"limit_reset":"monthly","usage_daily":5}}`
	c, done := orServer(t, http.StatusOK, body)
	defer done()
	o := observeOpenRouterQuota("test-key", orNow(), c)
	if o.State == "ok" {
		t.Fatal("an uncapped key read as ok; nothing is pacing it")
	}
	if !o.Blocked() {
		t.Error("unknown quota must fail closed")
	}
}

func TestOpenRouter_MissingKeyAndHTTPErrorAreUnknown(t *testing.T) {
	if o := observeOpenRouterQuota("", orNow(), http.DefaultClient); o.State != "unknown" || !o.Blocked() {
		t.Errorf("missing key: state=%q blocked=%v", o.State, o.Blocked())
	}
	c, done := orServer(t, http.StatusUnauthorized, `{}`)
	defer done()
	if o := observeOpenRouterQuota("test-key", orNow(), c); o.State != "unknown" {
		t.Errorf("HTTP 401: state=%q, want unknown", o.State)
	}
}

func TestOpenRouter_MalformedIsUnknownNotOK(t *testing.T) {
	for _, b := range []string{`not json`, `{}`, `{"data":{}}`, `{"data":{"limit":100}}`} {
		if o := parseOpenRouterKey([]byte(b), orNow()); o.State == "ok" {
			t.Errorf("body %q read as ok", b)
		}
	}
}
