package main

import (
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/eval_harness"
)

// TestVendorMirror pins the models.yml -> OpenRouter slug mapping. The cases are
// taken from live rows, because the mapping is only useful if it resolves the
// models we actually run: a rule that looks reasonable and matches nothing turns
// every vendor row into a silent "unmapped" and the cross-check into a no-op.
func TestVendorMirror(t *testing.T) {
	cases := []struct {
		provider, apiName, want string
	}{
		// google/openai: verbatim under the vendor prefix, -preview included.
		{"google", "gemini-3.6-flash", "google/gemini-3.6-flash"},
		{"google", "gemini-3.7-flash", "google/gemini-3.7-flash"},
		{"google", "gemini-3-flash-preview", "google/gemini-3-flash-preview"},
		{"google", "gemini-3.1-pro-preview", "google/gemini-3.1-pro-preview"},
		{"openai", "gpt-5.6-sol", "openai/gpt-5.6-sol"},
		{"openai", "gpt-5.4-mini", "openai/gpt-5.4-mini"},
		{"openai", "gpt-5.2-codex", "openai/gpt-5.2-codex"},

		// anthropic: drop the dated snapshot, rejoin the version with a dot.
		{"anthropic", "claude-haiku-4-5-20251001", "anthropic/claude-haiku-4.5"},
		{"anthropic", "claude-sonnet-4-5-20250929", "anthropic/claude-sonnet-4.5"},
		{"anthropic", "claude-sonnet-4-6", "anthropic/claude-sonnet-4.6"},
		// Single-component versions must NOT gain a dot.
		{"anthropic", "claude-opus-5", "anthropic/claude-opus-5"},
		{"anthropic", "claude-fable-5", "anthropic/claude-fable-5"},

		// Everything else has no mirror. openrouter rows are checked directly
		// against the API that bills them, and ollama runs on our own hardware.
		{"openrouter", "z-ai/glm-5.2", ""},
		{"ollama", "qwen3.6:35b-a3b-mxfp8", ""},
	}

	for _, c := range cases {
		if got := vendorMirror(c.provider, c.apiName); got != c.want {
			t.Errorf("vendorMirror(%q, %q) = %q, want %q", c.provider, c.apiName, got, c.want)
		}
	}
}

func scheduled(in, out float64) *eval_harness.ScheduledPricing {
	return &eval_harness.ScheduledPricing{InputPer1K: in, OutputPer1K: out}
}

// TestCheckSchedules drives the date-conditional pricing gate against a fixed
// "now" so it cannot rot into a test that passes because the calendar moved.
func TestCheckSchedules(t *testing.T) {
	now := time.Date(2027, 1, 15, 0, 0, 0, 0, time.UTC)

	cfg := &eval_harness.ModelsConfig{Models: map[string]eval_harness.ModelConfig{
		"lapsed": {
			APIName:  "gemini-3.7-flash",
			Provider: "google",
			Pricing: eval_harness.Pricing{
				InputPer1K: 0.00075, OutputPer1K: 0.00375,
				Expires: "2026-12-31", Next: scheduled(0.0015, 0.0075),
			},
		},
		"still-valid": {
			APIName:  "some-model",
			Provider: "google",
			Pricing: eval_harness.Pricing{
				InputPer1K: 1, OutputPer1K: 2,
				Expires: "2027-06-30", Next: scheduled(3, 4),
			},
		},
		"due-soon": {
			APIName:  "another-model",
			Provider: "google",
			Pricing: eval_harness.Pricing{
				InputPer1K: 1, OutputPer1K: 2,
				Expires: "2027-02-01", Next: scheduled(3, 4),
			},
		},
		"no-schedule": {
			APIName:  "plain",
			Provider: "google",
			Pricing:  eval_harness.Pricing{InputPer1K: 1, OutputPer1K: 2},
		},
		"expires-without-next": {
			APIName:  "half-a",
			Provider: "google",
			Pricing:  eval_harness.Pricing{InputPer1K: 1, OutputPer1K: 2, Expires: "2027-06-30"},
		},
		"next-without-expires": {
			APIName:  "half-b",
			Provider: "google",
			Pricing:  eval_harness.Pricing{InputPer1K: 1, OutputPer1K: 2, Next: scheduled(3, 4)},
		},
		"unparseable-date": {
			APIName:  "bad-date",
			Provider: "google",
			Pricing: eval_harness.Pricing{
				InputPer1K: 1, OutputPer1K: 2,
				Expires: "31/12/2026", Next: scheduled(3, 4),
			},
		},
	}}

	expired, soon, invalid := checkSchedules(cfg, now)

	names := func(fs []finding) map[string]bool {
		m := map[string]bool{}
		for _, f := range fs {
			m[f.Entry] = true
		}
		return m
	}

	gotExpired, gotSoon, gotInvalid := names(expired), names(soon), names(invalid)

	if !gotExpired["lapsed"] {
		t.Error("a row past its expires date was not reported as expired — this is the " +
			"exact silent halving the field exists to catch")
	}
	if len(expired) != 1 {
		t.Errorf("expected exactly 1 expired row, got %d: %v", len(expired), gotExpired)
	}
	if gotExpired["still-valid"] || gotSoon["still-valid"] {
		t.Error("a rate valid for another five months was flagged")
	}
	if !gotSoon["due-soon"] {
		t.Error("a rate lapsing inside the announcement window was not surfaced")
	}
	if gotExpired["no-schedule"] || gotSoon["no-schedule"] || gotInvalid["no-schedule"] {
		t.Error("a row with no schedule at all was flagged; the fields are optional")
	}

	for _, want := range []string{"expires-without-next", "next-without-expires", "unparseable-date"} {
		if !gotInvalid[want] {
			t.Errorf("half-written schedule %q was accepted; it must be rejected while it is "+
				"still cheap to fix, not on the day it fires", want)
		}
	}

	// The successor rate is what the operator has to type in; reporting the
	// wrong one is worse than reporting nothing.
	for _, f := range expired {
		if f.Entry == "lapsed" && (f.LiveIn != 0.0015 || f.LiveOut != 0.0075) {
			t.Errorf("expired finding carried successor %g/%g, want 0.0015/0.0075", f.LiveIn, f.LiveOut)
		}
	}
}

// TestCheckSchedules_IdenticalSuccessorIsNotFlagged documents a deliberate gap:
// the "next == current" case is caught by the CI test in internal/eval_harness,
// not here. This tool reports on money that moved; a schedule that changes
// nothing is a file-hygiene problem and belongs with the other offline checks.
func TestCheckSchedules_IdenticalSuccessorIsNotFlagged(t *testing.T) {
	cfg := &eval_harness.ModelsConfig{Models: map[string]eval_harness.ModelConfig{
		"noop-schedule": {
			APIName:  "m",
			Provider: "google",
			Pricing: eval_harness.Pricing{
				InputPer1K: 1, OutputPer1K: 2,
				Expires: "2099-01-01", Next: scheduled(1, 2),
			},
		},
	}}
	_, _, invalid := checkSchedules(cfg, time.Date(2027, 1, 15, 0, 0, 0, 0, time.UTC))
	if len(invalid) != 0 {
		t.Errorf("expected no findings from this tool, got %d", len(invalid))
	}
}
