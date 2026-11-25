package main

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/mission"
	storefs "github.com/sunholo-data/ailang/internal/storage/firestore"
)

// noOverride stubs the Firestore override read: none is set.
func noOverride(t *testing.T) {
	t.Helper()
	old := quotaOverrideLookup
	t.Cleanup(func() { quotaOverrideLookup = old })
	quotaOverrideLookup = func(context.Context, string, string) (*storefs.QuotaOverride, error) { return nil, nil }
}

// withOverride stubs the override read to return o (or err) and records the bucket asked.
func withOverride(t *testing.T, o *storefs.QuotaOverride, err error) *string {
	t.Helper()
	asked := new(string)
	old := quotaOverrideLookup
	t.Cleanup(func() { quotaOverrideLookup = old })
	quotaOverrideLookup = func(_ context.Context, _, bucket string) (*storefs.QuotaOverride, error) {
		*asked = bucket
		return o, err
	}
	return asked
}

func stubCodex(t *testing.T, state, reason string) {
	t.Helper()
	old := codexQuotaObserve
	t.Cleanup(func() { codexQuotaObserve = old })
	codexQuotaObserve = func(string, time.Time) mission.CodexQuotaObservation {
		return mission.CodexQuotaObservation{State: state, Reason: reason}
	}
}

func stubAnthropic(t *testing.T, o mission.AnthropicQuotaObservation) *string {
	t.Helper()
	got := new(string)
	old := anthropicQuotaObserve
	t.Cleanup(func() { anthropicQuotaObserve = old })
	anthropicQuotaObserve = func(token string, _ time.Time) mission.AnthropicQuotaObservation {
		*got = token
		return o
	}
	return got
}

func TestCloudRationBucket(t *testing.T) {
	for p, want := range map[string]string{"agy": "google", "codex": "codex", "claude": "anthropic", "pi": "", "motoko": "", "opencode": "", "gemini_agents": ""} {
		if got := cloudRationBucket(p); got != want {
			t.Errorf("%s: got %q, want %q", p, got, want)
		}
	}
}

// Providers without a subscription ration are never observed or refused.
func TestCheckCloudRation_UnrationedProvidersPass(t *testing.T) {
	withOverride(t, nil, fmt.Errorf("must not be read"))
	for _, p := range []string{"pi", "motoko", "opencode", "gemini_agents"} {
		if err := checkCloudRation(t.Context(), p, "proj"); err != nil {
			t.Errorf("%s: %v", p, err)
		}
	}
}

func TestCheckCloudRation_Codex(t *testing.T) {
	t.Setenv("CODEX_HOME", t.TempDir())
	asked := withOverride(t, nil, nil)
	stubCodex(t, "ok", "within ration")
	if err := checkCloudRation(t.Context(), "codex", "proj"); err != nil {
		t.Fatalf("within ration: %v", err)
	}
	if *asked != "" {
		t.Fatal("override read on the happy path")
	}
	// Over ration and unreadable both refuse, and name the override command.
	for _, state := range []string{"over", "unknown", "stale"} {
		stubCodex(t, state, "reason-"+state)
		err := checkCloudRation(t.Context(), "codex", "proj")
		if err == nil || !strings.Contains(err.Error(), "not cloning") || !strings.Contains(err.Error(), "--allow codex") {
			t.Fatalf("%s: want a refusal naming the override, got %v", state, err)
		}
		if *asked != "codex" {
			t.Fatalf("%s: override not consulted for the codex bucket (asked %q)", state, *asked)
		}
	}
}

// Cloud claude cannot read its usage today (setup-token: HTTP 403), so unknown admits;
// a real over-ration reading still refuses.
func TestCheckCloudRation_Anthropic(t *testing.T) {
	t.Setenv("AILANG_AUTH_MODE", "")
	t.Setenv("CLAUDE_CODE_OAUTH_TOKEN", "sk-ant-oat01-test")
	noOverride(t)
	token := stubAnthropic(t, mission.AnthropicQuotaObservation{State: "unknown", Reason: "HTTP 403", Enforced: true})
	if err := checkCloudRation(t.Context(), "claude", "proj"); err != nil {
		t.Fatalf("unmeasurable must admit: %v", err)
	}
	if *token != "sk-ant-oat01-test" {
		t.Fatalf("reader got token %q, want the job's CLAUDE_CODE_OAUTH_TOKEN", *token)
	}
	stubAnthropic(t, mission.AnthropicQuotaObservation{State: "over", Reason: "over ration", Enforced: true})
	if err := checkCloudRation(t.Context(), "claude", "proj"); err == nil || !strings.Contains(err.Error(), "anthropic ration over") {
		t.Fatalf("over ration must refuse: %v", err)
	}
	// AILANG_ANTHROPIC_RATION=0 (Enforced=false) is honoured like on the rig.
	stubAnthropic(t, mission.AnthropicQuotaObservation{State: "over", Reason: "over ration", Enforced: false})
	if err := checkCloudRation(t.Context(), "claude", "proj"); err != nil {
		t.Fatalf("unenforced must admit: %v", err)
	}
}

// The metered credit lane is not the subscription and is never observed.
func TestCheckCloudRation_ClaudeCreditLaneSkipped(t *testing.T) {
	t.Setenv("AILANG_AUTH_MODE", "apikey")
	stubAnthropic(t, mission.AnthropicQuotaObservation{State: "over", Enforced: true})
	withOverride(t, nil, fmt.Errorf("must not be read"))
	if err := checkCloudRation(t.Context(), "claude", "proj"); err != nil {
		t.Fatalf("credit lane: %v", err)
	}
}

func TestCheckCloudRation_Override(t *testing.T) {
	t.Setenv("CODEX_HOME", t.TempDir())
	stubCodex(t, "over", "over ration")
	now := time.Now()

	// An active override admits an over-ration job.
	withOverride(t, &storefs.QuotaOverride{Bucket: "codex", Until: now.Add(time.Hour), GrantedBy: "mark", Reason: "smoke"}, nil)
	if err := checkCloudRation(t.Context(), "codex", "proj"); err != nil {
		t.Fatalf("active override: %v", err)
	}
	// An expired one does not.
	withOverride(t, &storefs.QuotaOverride{Bucket: "codex", Until: now.Add(-time.Minute)}, nil)
	if err := checkCloudRation(t.Context(), "codex", "proj"); err == nil {
		t.Fatal("expired override admitted")
	}
	// An unreadable override store refuses, and says so.
	withOverride(t, nil, fmt.Errorf("firestore down"))
	if err := checkCloudRation(t.Context(), "codex", "proj"); err == nil || !strings.Contains(err.Error(), "override unreadable: firestore down") {
		t.Fatalf("unreadable override: %v", err)
	}
}

func TestQuotaOverrideActive(t *testing.T) {
	now := time.Now()
	var nilO *storefs.QuotaOverride
	if nilO.Active(now) {
		t.Fatal("nil override active")
	}
	if !(&storefs.QuotaOverride{Until: now.Add(time.Second)}).Active(now) {
		t.Fatal("future end not active")
	}
	if (&storefs.QuotaOverride{Until: now}).Active(now) {
		t.Fatal("override active at its end time")
	}
}
