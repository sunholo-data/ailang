package mission

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// liveCodexRateLimits is the account/rateLimits/read result captured on the rig 2026-09-24.
const liveCodexRateLimits = `{"ordinaryUsageAllowed":true,"rateLimits":{"limitId":"codex","limitName":null,"normalModelSlug":null,"primary":{"usedPercent":59,"windowDurationMins":10080,"resetsAt":1790500432},"secondary":null,"credits":{"hasCredits":false,"unlimited":false,"balance":"0"},"individualLimit":null,"spendControlReached":false,"planType":"prolite","rateLimitReachedType":null},"rateLimitsByLimitId":{"codex":{"limitId":"codex","limitName":null,"normalModelSlug":null,"primary":{"usedPercent":59,"windowDurationMins":10080,"resetsAt":1790500432},"secondary":null,"credits":{"hasCredits":false,"unlimited":false,"balance":"0"},"individualLimit":null,"spendControlReached":false,"planType":"prolite","rateLimitReachedType":null}},"rateLimitResetCredits":{"availableCount":1,"credits":[{"id":"RateLimitResetCredit_2592524f","resetType":"codexRateLimits","status":"available","grantedAt":1790109465,"expiresAt":1792701465,"title":"Full reset","description":"Thanks for using Codex!"}]},"accountId":"a","rateLimitUpsell":null}`

// codexLiveNow is the moment of that capture: 4.0 days into a week that resets 09-27 09:13Z.
func codexLiveNow() time.Time { return time.Date(2026, 9, 24, 9, 20, 0, 0, time.UTC) }

func stubCodexAppServer(t *testing.T, f func(method, params string) (json.RawMessage, error)) {
	t.Helper()
	prev := codexAppServerCall
	codexAppServerCall = func(_ context.Context, _, method, params string) (json.RawMessage, error) {
		return f(method, params)
	}
	t.Cleanup(func() { codexAppServerCall = prev })
}

func codexHomeWithAuth(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	if err := os.WriteFile(filepath.Join(home, "auth.json"), []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	return home
}

func TestParseCodexAppServer_LivePayloadIsOverWithReserve(t *testing.T) {
	o := parseCodexAppServerRateLimits([]byte(liveCodexRateLimits), codexLiveNow())
	if o.State != "over" {
		t.Fatalf("state = %q (%s), want over: 59%% used vs ~40%% allowed", o.State, o.Reason)
	}
	if len(o.Windows) != 1 || o.Windows[0].UsedPercent != 59 || o.Windows[0].WindowMinutes != 10080 {
		t.Fatalf("windows = %+v", o.Windows)
	}
	if o.Source != codexAppServerSource {
		t.Errorf("source = %q", o.Source)
	}
	rc := o.ResetCredits
	if rc == nil || rc.Available != 1 || len(rc.Credits) != 1 || rc.Credits[0].Title != "Full reset" {
		t.Fatalf("reset credits = %+v", rc)
	}
	if got := rc.Credits[0].ExpiresAt.Format("2006-01-02"); got != "2026-10-22" {
		t.Errorf("expiry = %s, want 2026-10-22", got)
	}
	for _, want := range []string{"1 Codex reset credit(s) in reserve", "next expires 2026-10-22", "attended only", "--codex-reset --yes"} {
		if !strings.Contains(o.Reason, want) {
			t.Errorf("blocked reason %q does not carry %q", o.Reason, want)
		}
	}
}

func TestParseCodexAppServer_WithinRationIsOkAndCarriesNoHint(t *testing.T) {
	body := strings.ReplaceAll(liveCodexRateLimits, `"usedPercent":59`, `"usedPercent":20`)
	o := parseCodexAppServerRateLimits([]byte(body), codexLiveNow())
	if o.State != "ok" {
		t.Fatalf("state = %q (%s), want ok", o.State, o.Reason)
	}
	if strings.Contains(o.Reason, "reset credit") {
		t.Errorf("an ok bucket should not advertise spending a reset: %q", o.Reason)
	}
	if o.ResetCredits == nil || o.ResetCredits.Available != 1 {
		t.Errorf("credits must be recorded even when ok: %+v", o.ResetCredits)
	}
}

func TestParseCodexAppServer_ProviderStopOutranksArithmetic(t *testing.T) {
	low := strings.ReplaceAll(liveCodexRateLimits, `"usedPercent":59`, `"usedPercent":20`)
	for name, body := range map[string]string{
		"reached type":  strings.ReplaceAll(low, `"rateLimitReachedType":null`, `"rateLimitReachedType":"primary"`),
		"spend control": strings.ReplaceAll(low, `"spendControlReached":false`, `"spendControlReached":true`),
		"not allowed":   strings.Replace(low, `"ordinaryUsageAllowed":true`, `"ordinaryUsageAllowed":false`, 1),
	} {
		if o := parseCodexAppServerRateLimits([]byte(body), codexLiveNow()); o.State != "over" {
			t.Errorf("%s: state = %q, want over", name, o.State)
		}
	}
}

func TestParseCodexAppServer_UnvalidatableIsUnknownNeverOk(t *testing.T) {
	for name, body := range map[string]string{
		"not json":      `{`,
		"no codex":      `{"rateLimitsByLimitId":{"other":{"limitId":"other"}}}`,
		"bad percent":   strings.ReplaceAll(liveCodexRateLimits, `"usedPercent":59`, `"usedPercent":159`),
		"reset in past": strings.ReplaceAll(liveCodexRateLimits, `1790500432`, `1700000000`),
		"no windows":    strings.ReplaceAll(liveCodexRateLimits, `"primary":{"usedPercent":59,"windowDurationMins":10080,"resetsAt":1790500432}`, `"primary":null`),
	} {
		if o := parseCodexAppServerRateLimits([]byte(body), codexLiveNow()); o.State != "unknown" {
			t.Errorf("%s: state = %q, want unknown", name, o.State)
		}
	}
}

// THE REGRESSION (2026-09-24): no session in 24h used to read "unknown", which blocked
// Codex, which wrote no session — forever. The app-server answers with no session at all.
func TestObserveCodexQuota_NoRecentSessionsStillGetsAProviderVerdict(t *testing.T) {
	calls := 0
	stubCodexAppServer(t, func(method, _ string) (json.RawMessage, error) {
		calls++
		if method != "account/rateLimits/read" {
			t.Errorf("method = %q", method)
		}
		return json.RawMessage(liveCodexRateLimits), nil
	})
	o := ObserveCodexQuota(codexHomeWithAuth(t), codexLiveNow())
	if o.State != "over" || o.Source != codexAppServerSource || calls != 1 {
		t.Fatalf("state=%q source=%q calls=%d, want over from the app-server in one call", o.State, o.Source, calls)
	}
}

func TestObserveCodexQuota_AppServerFailureFallsBackAndSaysWhy(t *testing.T) {
	stubCodexAppServer(t, func(string, string) (json.RawMessage, error) {
		return nil, errors.New("codex app-server: exec: \"codex\": executable file not found")
	})
	o := ObserveCodexQuota(codexHomeWithAuth(t), codexLiveNow())
	if o.State != "unknown" {
		t.Fatalf("state = %q, want unknown (no sessions either)", o.State)
	}
	if !strings.Contains(o.Reason, "app-server read also failed") || !strings.Contains(o.Reason, "executable file not found") {
		t.Errorf("reason %q hides the app-server failure", o.Reason)
	}
}

func TestObserveCodexQuota_NoCredentialNeverSpawns(t *testing.T) {
	stubCodexAppServer(t, func(string, string) (json.RawMessage, error) {
		t.Error("app-server called without a credential")
		return nil, errors.New("unreachable")
	})
	o := ObserveCodexQuota(t.TempDir(), codexLiveNow())
	if o.State != "unknown" || !strings.Contains(o.Reason, "no Codex credential") {
		t.Errorf("state=%q reason=%q", o.State, o.Reason)
	}
}

func TestScanCodexAppServerReply_SkipsNoiseAndReportsErrors(t *testing.T) {
	stream := strings.Join([]string{
		`not json at all`,
		`{"id":1,"result":{"userAgent":"x"}}`,
		`{"method":"account/rateLimits/updated","params":{}}`,
		`{"id":2,"result":{"ok":true}}`,
	}, "\n")
	got, err := scanCodexAppServerReply(context.Background(), strings.NewReader(stream), "m")
	if err != nil || string(got) != `{"ok":true}` {
		t.Fatalf("got %s, %v", got, err)
	}
	_, err = scanCodexAppServerReply(context.Background(), strings.NewReader(`{"id":2,"error":{"message":"not logged in"}}`), "m")
	if err == nil || !strings.Contains(err.Error(), "not logged in") {
		t.Errorf("error reply: %v", err)
	}
	if _, err = scanCodexAppServerReply(context.Background(), strings.NewReader(`{"id":1,"result":{}}`), "m"); err == nil {
		t.Error("EOF before the reply must be an error")
	}
}

func TestConsumeCodexResetCredit_SendsKeyAndCredit(t *testing.T) {
	var gotMethod, gotParams string
	stubCodexAppServer(t, func(method, params string) (json.RawMessage, error) {
		gotMethod, gotParams = method, params
		return json.RawMessage(`{"outcome":"reset"}`), nil
	})
	out, err := ConsumeCodexResetCredit(context.Background(), "/h", "RateLimitResetCredit_1", "k-1")
	if err != nil || out != "reset" {
		t.Fatalf("out=%q err=%v", out, err)
	}
	if gotMethod != "account/rateLimitResetCredit/consume" {
		t.Errorf("method = %q", gotMethod)
	}
	var p map[string]string
	if json.Unmarshal([]byte(gotParams), &p) != nil || p["idempotencyKey"] != "k-1" || p["creditId"] != "RateLimitResetCredit_1" {
		t.Errorf("params = %s", gotParams)
	}
	if _, err := ConsumeCodexResetCredit(context.Background(), "/h", "", ""); err == nil {
		t.Error("an empty idempotency key must be refused: a retry could spend twice")
	}
}
