package mission

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// liveAnthropicPayload is the VERBATIM shape returned by /api/oauth/usage on 2026-09-08.
// Keeping the real response as the fixture is the point: a hand-written approximation is
// the thing a schema change slips past.
const liveAnthropicPayload = `{
  "five_hour": {"utilization": 41.0, "resets_at": "2026-09-08T10:20:00.057927+00:00",
                "limit_dollars": null, "used_dollars": null, "remaining_dollars": null, "locked_reason": null},
  "seven_day": {"utilization": 34.0, "resets_at": "2026-09-14T05:00:00.057946+00:00",
                "limit_dollars": null, "used_dollars": null, "remaining_dollars": null, "locked_reason": null},
  "seven_day_opus": null,
  "nimbus_quill": {"utilization": 0.0, "resets_at": null, "limit_dollars": null,
                   "used_dollars": null, "remaining_dollars": null, "locked_reason": null},
  "extra_usage": {"is_enabled": false, "monthly_limit": null, "used_credits": null,
                  "utilization": null, "user_disabled": true, "spend_limit_reached": false},
  "limits": [
    {"kind": "session", "group": "session", "percent": 41, "severity": "normal",
     "resets_at": "2026-09-08T10:20:00.057927+00:00", "scope": null, "is_active": true},
    {"kind": "weekly_all", "group": "weekly", "percent": 34, "severity": "normal",
     "resets_at": "2026-09-14T05:00:00.057946+00:00", "scope": null, "is_active": false},
    {"kind": "weekly_scoped", "group": "weekly", "percent": 25, "severity": "normal",
     "resets_at": "2026-09-14T05:00:00.058136+00:00",
     "scope": {"model": {"id": null, "display_name": "Fable"}, "surface": null}, "is_active": false}
  ],
  "spend": {"used": {"amount_minor": 0, "currency": "USD", "exponent": 2}, "percent": 0,
            "severity": "normal", "enabled": false},
  "member_dashboard_available": false
}`

func anthropicServer(t *testing.T, status int, body string) (*http.Client, func()) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("Authorization = %q, want bearer test-token", got)
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	client := srv.Client()
	client.Transport = rewriteHost{base: srv.URL, rt: client.Transport}
	return client, srv.Close
}

// rewriteHost sends the request to the test server while the code under test keeps using
// the real production URL — so the URL constant itself stays covered.
type rewriteHost struct {
	base string
	rt   http.RoundTripper
}

func (h rewriteHost) RoundTrip(r *http.Request) (*http.Response, error) {
	u, err := r.URL.Parse(h.base)
	if err != nil {
		return nil, err
	}
	r.URL.Scheme, r.URL.Host = u.Scheme, u.Host
	return h.rt.RoundTrip(r)
}

// now is inside both live windows: 26.7h into the 7-day window, before the 5h reset.
func anthropicNow() time.Time {
	return time.Date(2026, 9, 8, 7, 44, 0, 0, time.UTC)
}

func TestParseAnthropicUsage_LivePayloadUsesUnscopedWindowsOnly(t *testing.T) {
	o := parseAnthropicUsage([]byte(liveAnthropicPayload), anthropicNow())
	if len(o.Windows) != 2 {
		t.Fatalf("windows = %d, want 2 (session + weekly_all; weekly_scoped must be excluded)", len(o.Windows))
	}
	if o.Windows[0].UsedPercent != 41 || o.Windows[0].WindowMinutes != 300 {
		t.Errorf("session window = %+v, want 41%% / 300m", o.Windows[0])
	}
	if o.Windows[1].UsedPercent != 34 || o.Windows[1].WindowMinutes != 7*24*60 {
		t.Errorf("weekly window = %+v, want 34%% / 10080m", o.Windows[1])
	}
	// The Fable-scoped 25% must never appear: it caps one model, not the account.
	for _, w := range o.Windows {
		if w.UsedPercent == 25 {
			t.Error("weekly_scoped entry leaked into the account ration")
		}
	}
	want := time.Date(2026, 9, 14, 5, 0, 0, 57946000, time.UTC)
	if !o.Windows[1].ResetsAt.Equal(want) {
		t.Errorf("weekly reset = %s, want %s", o.Windows[1].ResetsAt, want)
	}
}

func TestParseAnthropicUsage_FallsBackToTopLevelWindows(t *testing.T) {
	body := `{"five_hour": {"utilization": 12.0, "resets_at": "2026-09-08T10:20:00Z"},
	          "seven_day": {"utilization": 8.0, "resets_at": "2026-09-14T05:00:00Z"}}`
	o := parseAnthropicUsage([]byte(body), anthropicNow())
	if len(o.Windows) != 2 {
		t.Fatalf("windows = %d, want 2 from the top-level objects", len(o.Windows))
	}
	if o.Windows[1].UsedPercent != 8 {
		t.Errorf("weekly = %v, want 8", o.Windows[1].UsedPercent)
	}
}

func TestParseAnthropicUsage_LockedWindowBlocksRegardlessOfPercent(t *testing.T) {
	body := `{"five_hour": {"utilization": 1.0, "resets_at": "2026-09-08T10:20:00Z",
	                        "locked_reason": "plan suspended"},
	          "seven_day": {"utilization": 1.0, "resets_at": "2026-09-14T05:00:00Z"}}`
	o := parseAnthropicUsage([]byte(body), anthropicNow())
	if o.State != "over" {
		t.Fatalf("state = %q, want over for a locked window at 1%% usage", o.State)
	}
}

func TestParseAnthropicUsage_MalformedIsUnknownNotOK(t *testing.T) {
	for _, body := range []string{`not json`, `{}`, `{"limits": []}`} {
		o := parseAnthropicUsage([]byte(body), anthropicNow())
		if o.State == "ok" {
			t.Errorf("body %q produced state ok; unknown quota must never read as ok", body)
		}
	}
}

func TestObserveAnthropicQuota_OverRationIsReportedButNotEnforcedByDefault(t *testing.T) {
	t.Setenv("AILANG_ANTHROPIC_RATION", "")
	client, done := anthropicServer(t, http.StatusOK, liveAnthropicPayload)
	defer done()

	o := observeAnthropicQuota("test-token", anthropicNow(), client)
	// 34% used, 26.7h into a 7-day window => 11.1% allowance under 10%/day.
	if o.State != "over" {
		t.Fatalf("state = %q, want over (34%% used vs ~11%% allowance)", o.State)
	}
	if o.Blocked() {
		t.Error("Blocked() = true with enforcement off; this would refuse every controller probe")
	}
	if o.Enforced {
		t.Error("Enforced = true without AILANG_ANTHROPIC_RATION=1")
	}
}

func TestObserveAnthropicQuota_EnforcementOptInBlocks(t *testing.T) {
	t.Setenv("AILANG_ANTHROPIC_RATION", "1")
	client, done := anthropicServer(t, http.StatusOK, liveAnthropicPayload)
	defer done()

	o := observeAnthropicQuota("test-token", anthropicNow(), client)
	if !o.Blocked() {
		t.Fatalf("Blocked() = false with enforcement on and state %q", o.State)
	}
}

func TestObserveAnthropicQuota_MissingCredentialIsUnknownAndUnenforced(t *testing.T) {
	t.Setenv("AILANG_ANTHROPIC_RATION", "1")
	o := observeAnthropicQuota("", anthropicNow(), http.DefaultClient)
	if o.State != "unknown" {
		t.Errorf("state = %q, want unknown", o.State)
	}
	if !o.Blocked() {
		t.Error("an unknown quota under enforcement must fail closed")
	}
}

func TestObserveAnthropicQuota_HTTPErrorIsUnknown(t *testing.T) {
	t.Setenv("AILANG_ANTHROPIC_RATION", "")
	client, done := anthropicServer(t, http.StatusUnauthorized, `{}`)
	defer done()

	o := observeAnthropicQuota("test-token", anthropicNow(), client)
	if o.State != "unknown" {
		t.Errorf("state = %q, want unknown on HTTP 401", o.State)
	}
}

func TestObserveAnthropicQuota_WithinRationIsOK(t *testing.T) {
	t.Setenv("AILANG_ANTHROPIC_RATION", "1")
	// 5% into a 7-day window that is 26.7h old: inside the 11.1% allowance.
	body := `{"limits": [
	   {"kind": "weekly_all", "group": "weekly", "percent": 5, "severity": "normal",
	    "resets_at": "2026-09-14T05:00:00Z", "scope": null, "is_active": false}]}`
	client, done := anthropicServer(t, http.StatusOK, body)
	defer done()

	o := observeAnthropicQuota("test-token", anthropicNow(), client)
	if o.State != "ok" {
		t.Fatalf("state = %q (%s), want ok", o.State, o.Reason)
	}
	if o.Blocked() {
		t.Error("Blocked() = true for an in-ration observation")
	}
}
