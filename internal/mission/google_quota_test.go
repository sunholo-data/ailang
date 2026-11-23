package mission

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// The real `agy -p /quota` reading of 2026-11-23 (agy 1.5.15, Google AI Ultra).
const agyQuotaFixture = "Gemini Models\tWeekly Limit Remaining\t97%\t2026-11-30T14:09:17Z\n" +
	"Gemini Models\tFive Hour Limit Remaining\t95%\t2026-11-23T19:09:17Z\n" +
	"Claude and GPT models\tWeekly Limit Remaining\t100%\t2026-11-30T14:21:28Z\n" +
	"Claude and GPT models\tFive Hour Limit Remaining\t100%\t2026-11-23T19:21:28Z\n"

// quotaAt is a reading taken at now: the weekly window resets 2026-11-30T14:00Z, the
// five-hour window two hours after now (a real reading's short reset is always ahead).
func quotaAt(weekRemain, fiveRemain string, now time.Time) []byte {
	return []byte("Gemini Models\tWeekly Limit Remaining\t" + weekRemain + "\t2026-11-30T14:00:00Z\n" +
		"Gemini Models\tFive Hour Limit Remaining\t" + fiveRemain + "\t" + now.Add(2*time.Hour).UTC().Format(time.RFC3339) + "\n")
}

// The weekly window resets 2026-11-30T14:00Z, so it started 2026-11-23T14:00Z.
var day0 = time.Date(2026, 11, 23, 14, 0, 0, 0, time.UTC)

func TestGoogleQuota_FixtureIsWithinPace(t *testing.T) {
	now := time.Date(2026, 11, 23, 15, 30, 0, 0, time.UTC)
	o := evaluateGoogleQuota([]byte(agyQuotaFixture), nil, "/h", 15, now)
	// 3% used, 1h21m into the window: the pace line is ~0.84%, so the loops must wait
	// even though attended use (the smoke gate) is what spent it — usage is ACCOUNT usage.
	if o.State != "over" {
		t.Fatalf("state = %s (%s)", o.State, o.Reason)
	}
	if len(o.Windows) != 2 || o.Windows[0].UsedPercent != 3 || o.Windows[1].UsedPercent != 5 {
		t.Fatalf("windows = %+v (used must be 100 - remaining)", o.Windows)
	}
	if o.Windows[0].WindowMinutes != 7*24*60 || o.Windows[1].WindowMinutes != 300 {
		t.Errorf("window minutes = %d, %d", o.Windows[0].WindowMinutes, o.Windows[1].WindowMinutes)
	}
	if len(o.Other) != 2 || o.Other[0].Family != "Claude and GPT models" {
		t.Errorf("Claude/GPT rows must be reported, not paced: %+v", o.Other)
	}
}

func TestGoogleQuota_DailyPace(t *testing.T) {
	cases := []struct {
		name   string
		at     time.Time
		weekly string
		want   string
	}{
		{"day 2, 20% used, line 30%", day0.Add(48 * time.Hour), "80%", "ok"},
		{"day 2, 29% used, line 30% -> inside the 2pp margin", day0.Add(48 * time.Hour), "71%", "over"},
		{"day 2, 31% used, line 30% -> over", day0.Add(48 * time.Hour), "69%", "over"},
		{"day 1, 16% used, line 15% -> over", day0.Add(24 * time.Hour), "84%", "over"},
		{"day 6.9, 95% used, line capped 100%", day0.Add(165 * time.Hour), "5%", "ok"},
		{"unused window gets no margin at its start", day0.Add(time.Minute), "100%", "ok"},
		{"weekly exhausted", day0.Add(150 * time.Hour), "0%", "over"},
	}
	for _, c := range cases {
		o := evaluateGoogleQuota(quotaAt(c.weekly, "90%", c.at), nil, "/h", 15, c.at)
		if o.State != c.want {
			t.Errorf("%s: state=%s want %s (%s; allowance %.2f)", c.name, o.State, c.want, o.Reason, o.Windows[0].AllowancePercent)
		}
	}
	at := day0.Add(48 * time.Hour)
	if o := evaluateGoogleQuota(quotaAt("80%", "90%", at), nil, "/h", 15, at); o.Windows[0].AllowancePercent != 30 {
		t.Errorf("pace line on day 2 = %.2f, want 30", o.Windows[0].AllowancePercent)
	}
	if o := evaluateGoogleQuota(quotaAt("80%", "90%", at), nil, "/h", 10, at); o.State != "over" {
		t.Errorf("at 10%%/day, 20%% used on day 2 must be over, got %s", o.State)
	}
}

func TestGoogleQuota_FiveHourExhaustedBlocks(t *testing.T) {
	o := evaluateGoogleQuota(quotaAt("90%", "0%", day0.Add(48*time.Hour)), nil, "/h", 15, day0.Add(48*time.Hour))
	if o.State != "over" || !strings.Contains(o.Reason, "exhausted") {
		t.Fatalf("state=%s reason=%s", o.State, o.Reason)
	}
}

func TestGoogleQuota_FailsClosed(t *testing.T) {
	now := day0.Add(time.Hour)
	for name, tc := range map[string]struct {
		out  string
		err  error
		want string
	}{
		"not signed in":  {"Error: Please sign in to view available models.", errors.New("exit status 1"), "make agy-signin"},
		"auth required":  {"Authentication required. Please visit the URL to log in:", nil, "make agy-signin"},
		"command failed": {"boom", errors.New("exit status 2"), "agy /quota failed"},
		"garbage":        {"nothing useful here", nil, "unparseable"},
		"bad percent":    {"Gemini Models\tWeekly Limit Remaining\tlots%\t2026-11-30T14:00:00Z\n", nil, "unparseable"},
		"no weekly row":  {"Gemini Models\tFive Hour Limit Remaining\t50%\t2026-11-23T19:00:00Z\n", nil, "no Gemini weekly row"},
	} {
		o := evaluateGoogleQuota([]byte(tc.out), tc.err, "/h", 15, now)
		if o.State != "unknown" || !o.Blocked() || !strings.Contains(o.Reason, tc.want) {
			t.Errorf("%s: state=%s blocked=%v reason=%q (want %q)", name, o.State, o.Blocked(), o.Reason, tc.want)
		}
	}
	if o := evaluateGoogleQuota(quotaAt("90%", "90%", day0), nil, "/h", 15, time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)); o.State != "expired" {
		t.Errorf("a past reset must not read as ok: %s", o.State)
	}
}

func TestObserveGoogleQuota_UsesTheExecutorsSubscriptionHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("AILANG_AGY_HOME_ROOT", "")
	t.Setenv("PATH", "/nonexistent")
	sub := filepath.Join(home, ".ailang", "state", "agy", "subscription")
	bin := filepath.Join(home, ".local", "bin")
	for _, d := range []string{sub, bin} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(bin, "agy"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	var gotHome, gotPath string
	old := agyQuotaRunner
	agyQuotaRunner = func(agyPath, h string) ([]byte, error) {
		gotPath, gotHome = agyPath, h
		return quotaAt("80%", "90%", day0.Add(48*time.Hour)), nil
	}
	defer func() { agyQuotaRunner = old }()

	o := ObserveGoogleQuota(Paths{Home: home}, day0.Add(48*time.Hour))
	if gotHome != sub || gotPath != filepath.Join(bin, "agy") {
		t.Fatalf("ran %q with HOME %q; want %q with %q", gotPath, gotHome, filepath.Join(bin, "agy"), sub)
	}
	if o.State != "ok" || o.PacePerDay != 15 {
		t.Fatalf("state=%s pace=%v reason=%s", o.State, o.PacePerDay, o.Reason)
	}

	t.Setenv("AILANG_GOOGLE_DAILY_PACE", "abc")
	if o := ObserveGoogleQuota(Paths{Home: home}, day0); o.State != "unknown" || !strings.Contains(o.Reason, "AILANG_GOOGLE_DAILY_PACE") {
		t.Errorf("a bad pace must fail loudly, got %s: %s", o.State, o.Reason)
	}
	t.Setenv("AILANG_GOOGLE_DAILY_PACE", "")
	if err := os.RemoveAll(sub); err != nil {
		t.Fatal(err)
	}
	if o := ObserveGoogleQuota(Paths{Home: home}, day0); o.State != "unknown" || !strings.Contains(o.Reason, "agy-signin") {
		t.Errorf("a missing subscription HOME must block naming the fix, got %s: %s", o.State, o.Reason)
	}
}
