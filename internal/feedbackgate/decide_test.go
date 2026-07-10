package feedbackgate

import (
	"context"
	"strings"
	"testing"
)

// baseInput is a well-formed, dispatch-eligible input. Individual tests mutate
// one field to trip a specific rule.
func baseInput() Input {
	return Input{
		ID:       "inbox_1_abc",
		Category: "auto:bug",
		Body:     "The compiler crashes on empty records.\n\n---\nailang_version: v0.28.0\n",
		From:     "mcp-public",
		Inbox:    "pkg:acme/widget",
		Source:   "public",
	}
}

func TestDecideRules(t *testing.T) {
	longBody := strings.Repeat("x", 9000)
	manyURLs := "check " + strings.Repeat("https://spam.example/a ", 6)
	base64Blob := strings.Repeat("QWxhZGRpbjpvcGVuIHNlc2FtZQ", 60) // > 1KB base64-ish

	cases := []struct {
		name       string
		mutate     func(in *Input)
		wantAction string
		wantReason string
	}{
		{
			name:       "well-formed dispatches",
			mutate:     func(*Input) {},
			wantAction: ActionDispatch,
			wantReason: ReasonPassed,
		},
		{
			name:       "no auto prefix files",
			mutate:     func(in *Input) { in.Category = "bug" },
			wantAction: ActionFile,
			wantReason: ReasonNotAuthorized,
		},
		{
			name:       "oversized body rejects",
			mutate:     func(in *Input) { in.Body = longBody },
			wantAction: ActionReject,
			wantReason: ReasonBodyTooLarge,
		},
		{
			name:       "too many urls rejects",
			mutate:     func(in *Input) { in.Body = manyURLs },
			wantAction: ActionReject,
			wantReason: ReasonSpamPattern,
		},
		{
			name:       "large base64 blob rejects",
			mutate:     func(in *Input) { in.Body = base64Blob },
			wantAction: ActionReject,
			wantReason: ReasonSpamPattern,
		},
		{
			name:       "untrusted sender rejects",
			mutate:     func(in *Input) { in.From = "random-attacker" },
			wantAction: ActionReject,
			wantReason: ReasonUntrustedSource,
		},
		{
			name:       "agent sender is trusted",
			mutate:     func(in *Input) { in.From = "agent-eval-suite" },
			wantAction: ActionDispatch,
			wantReason: ReasonPassed,
		},
		{
			name:       "unknown inbox rejects",
			mutate:     func(in *Input) { in.Inbox = "not-a-real-inbox" },
			wantAction: ActionReject,
			wantReason: ReasonUnknownInbox,
		},
		{
			name:       "internal inbox allowed",
			mutate:     func(in *Input) { in.Inbox = "design-doc-creator" },
			wantAction: ActionDispatch,
			wantReason: ReasonPassed,
		},
		{
			name:       "unknown category files",
			mutate:     func(in *Input) { in.Category = "auto:rant" },
			wantAction: ActionFile,
			wantReason: ReasonUnknownCategory,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := baseInput()
			tc.mutate(&in)
			// M1: no cooldown, no classifier injected — Decide is pure.
			v, err := Decide(context.Background(), in, FeedbackGateConfig{})
			if err != nil {
				t.Fatalf("Decide returned error: %v", err)
			}
			if v.Action != tc.wantAction {
				t.Errorf("Action = %q, want %q (reason %q)", v.Action, tc.wantAction, v.Reason)
			}
			if v.Reason != tc.wantReason {
				t.Errorf("Reason = %q, want %q", v.Reason, tc.wantReason)
			}
		})
	}
}

// TestDecidePureAtM1 asserts that with no injected stages, Decide performs no
// IO and returns dispatch for a clean input (cooldown/classifier are no-ops).
func TestDecidePureAtM1(t *testing.T) {
	v, err := Decide(context.Background(), baseInput(), FeedbackGateConfig{})
	if err != nil {
		t.Fatalf("Decide error: %v", err)
	}
	if v.Action != ActionDispatch {
		t.Fatalf("Action = %q, want dispatch", v.Action)
	}
	if v.Cost <= 0 {
		t.Errorf("dispatch Cost = %v, want > 0", v.Cost)
	}
}

// TestConfigNormalizedDefaults locks in the default values so a later edit
// can't silently change the abuse thresholds.
func TestConfigNormalizedDefaults(t *testing.T) {
	c := FeedbackGateConfig{}.normalized()
	if c.Mode != ModeFull {
		t.Errorf("Mode = %q, want %q", c.Mode, ModeFull)
	}
	if c.MaxBodyBytes != 8192 {
		t.Errorf("MaxBodyBytes = %d, want 8192", c.MaxBodyBytes)
	}
	if c.MaxDispatchPerHour != 3 || c.MaxDispatchPerDay != 10 {
		t.Errorf("dispatch limits = %d/%d, want 3/10", c.MaxDispatchPerHour, c.MaxDispatchPerDay)
	}
	if c.DailyBudgetUSD != 5.0 {
		t.Errorf("DailyBudgetUSD = %v, want 5.0", c.DailyBudgetUSD)
	}
	if c.ClassifierModel != "claude-haiku-4-5" {
		t.Errorf("ClassifierModel = %q, want claude-haiku-4-5", c.ClassifierModel)
	}
}
