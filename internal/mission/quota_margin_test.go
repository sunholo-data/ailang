package mission

import (
	"strings"
	"testing"
	"time"
)

// marginWindows is the 2026-09-24 Thursday reading (see TestWeeklyPace_...) with the weekly
// usage placed HEADROOM points under the pace line.
func marginWindows(t *testing.T, now time.Time, headroom float64) []CodexQuotaWindow {
	t.Helper()
	w := []CodexQuotaWindow{
		{UsedPercent: 6, WindowMinutes: 300, ResetsAt: time.Date(2026, 9, 24, 13, 20, 0, 0, time.UTC)},
		{UsedPercent: 0, WindowMinutes: 7 * 24 * 60, ResetsAt: time.Date(2026, 9, 28, 5, 0, 0, 0, time.UTC)},
	}
	probe := CodexQuotaObservation{ObservedAt: now, Windows: append([]CodexQuotaWindow(nil), w...)}
	probe.evaluateWithMargin(now, 0)
	w[1].UsedPercent = probe.Windows[1].AllowancePercent - headroom
	return w
}

func TestStartMargin_DefaultsAndOverride(t *testing.T) {
	for bucket, want := range map[string]float64{"codex": 2, "anthropic": 1, "ollama": 3, "openrouter": 0.60} {
		if got := StartMargin(bucket); got != want {
			t.Errorf("StartMargin(%q) = %v, want %v", bucket, got, want)
		}
	}
	t.Setenv("AILANG_QUOTA_MARGIN_CODEX", "5")
	if got := StartMargin("codex"); got != 5 {
		t.Errorf("override: StartMargin(codex) = %v, want 5", got)
	}
	for _, bad := range []string{"-1", "abc", "NaN", "Inf"} {
		t.Setenv("AILANG_QUOTA_MARGIN_CODEX", bad)
		if got := StartMargin("codex"); got != 2 {
			t.Errorf("invalid override %q: StartMargin(codex) = %v, want the default 2", bad, got)
		}
	}
}

// The case Mark named: under the allowance by half a point is not worth starting.
func TestCodexMargin_HalfAPointUnderThePaceIsBlocked(t *testing.T) {
	now := time.Date(2026, 9, 24, 9, 1, 0, 0, time.UTC)
	o := CodexQuotaObservation{ObservedAt: now, Windows: marginWindows(t, now, 0.5)}
	o.evaluate(now)
	if !o.Blocked() || !o.HeadroomShort {
		t.Fatalf("state=%q short=%v, want blocked as headroom-short", o.State, o.HeadroomShort)
	}
	if !strings.Contains(o.Reason, "headroom 0.5pp is below the 2pp start margin") {
		t.Errorf("reason = %q, want the measured headroom and the margin", o.Reason)
	}
}

func TestCodexMargin_EnoughHeadroomIsAdmitted(t *testing.T) {
	now := time.Date(2026, 9, 24, 9, 1, 0, 0, time.UTC)
	o := CodexQuotaObservation{ObservedAt: now, Windows: marginWindows(t, now, 2.5)}
	o.evaluate(now)
	if o.Blocked() || o.HeadroomShort {
		t.Fatalf("state=%q (%s), want ok with 2.5pp headroom against a 2pp margin", o.State, o.Reason)
	}
}

// Over the allowance keeps its existing reason: the margin never relabels a real overrun.
func TestCodexMargin_OverrunKeepsOverReason(t *testing.T) {
	now := time.Date(2026, 9, 24, 9, 1, 0, 0, time.UTC)
	o := CodexQuotaObservation{ObservedAt: now, Windows: marginWindows(t, now, -1)}
	o.evaluate(now)
	if !o.Blocked() || o.HeadroomShort || !strings.Contains(o.Reason, "exceeds ration") {
		t.Fatalf("state=%q short=%v reason=%q, want a plain overrun", o.State, o.HeadroomShort, o.Reason)
	}
}

// Each bucket uses its own margin: 1.5pp of headroom is short for Codex (2pp) and enough for
// Anthropic (1pp) on the same reading.
func TestAnthropicMargin_UsesItsOwnMargin(t *testing.T) {
	now := time.Date(2026, 9, 24, 9, 1, 0, 0, time.UTC)
	a := AnthropicQuotaObservation{ObservedAt: now, Enforced: true, Windows: marginWindows(t, now, 1.5)}
	evaluateAnthropicQuota(&a, now)
	if a.Blocked() {
		t.Fatalf("anthropic state=%q (%s), want ok: 1.5pp headroom against a 1pp margin", a.State, a.Reason)
	}
	a = AnthropicQuotaObservation{ObservedAt: now, Enforced: true, Windows: marginWindows(t, now, 0.4)}
	evaluateAnthropicQuota(&a, now)
	if !a.Blocked() || !strings.HasPrefix(a.Reason, "Anthropic headroom 0.4pp") {
		t.Fatalf("anthropic state=%q reason=%q, want blocked as Anthropic headroom-short", a.State, a.Reason)
	}
	c := CodexQuotaObservation{ObservedAt: now, Windows: marginWindows(t, now, 1.5)}
	c.evaluate(now)
	if !c.Blocked() {
		t.Fatalf("codex state=%q, want blocked: 1.5pp headroom against a 2pp margin", c.State)
	}
}

func TestOllamaMargin_DailyRationHeadroom(t *testing.T) {
	now := time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)
	// 8pp consumed over 20h: under the 10pp/day ration, but 2pp short of the 3pp margin.
	got := applyOllamaRateRation(gaugeOK(0.38), obs(now, [2]float64{20, 0.30}, [2]float64{0, 0.38}), now)
	if !got.Blocked() || !strings.Contains(got.Reason, "start margin") {
		t.Fatalf("state=%q reason=%q, want blocked by the 3pp margin at 8pp of 10pp", got.State, got.Reason)
	}
	// 5pp consumed: 5pp of headroom clears the margin.
	got = applyOllamaRateRation(gaugeOK(0.35), obs(now, [2]float64{20, 0.30}, [2]float64{0, 0.35}), now)
	if got.Blocked() {
		t.Fatalf("state=%q reason=%q, want ok at 5pp of 10pp", got.State, got.Reason)
	}
}

func TestOpenRouterMargin_DailyAndMonthlyHeadroom(t *testing.T) {
	limit := 100.0
	for _, tc := range []struct {
		name            string
		day, remaining  float64
		wantBlocked     bool
		wantReasonMatch string
	}{
		{"plenty", 0.50, 50, false, "within ration"},
		{"inside the daily margin", 2.00, 50, true, "start margin"},
		{"month nearly spent", 0.10, 0.40, true, "left this month"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			day, rem := tc.day, tc.remaining
			o := evaluateOpenRouterQuota(OpenRouterQuotaObservation{LimitUSD: &limit, UsedDayUSD: &day, RemainingUSD: &rem}, time.Time{})
			if o.Blocked() != tc.wantBlocked || !strings.Contains(o.Reason, tc.wantReasonMatch) {
				t.Fatalf("state=%q reason=%q, want blocked=%v matching %q", o.State, o.Reason, tc.wantBlocked, tc.wantReasonMatch)
			}
		})
	}
}
