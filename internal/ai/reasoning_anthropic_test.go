package ai

import (
	"errors"
	"strings"
	"testing"
)

// TestAnthropicThinkingStyleFor pins generation classification, including the
// dated-snapshot aliasing and the fail-loud miss for anything unlisted.
func TestAnthropicThinkingStyleFor(t *testing.T) {
	tests := []struct {
		model        string
		wantKnown    bool
		wantAdaptive bool
		wantDisable  bool
	}{
		// Adaptive generation — thinking.budget_tokens is a 400 here.
		{"claude-opus-5", true, true, true},
		{"claude-opus-4-8", true, true, true},
		{"claude-opus-4-7", true, true, true},
		{"claude-sonnet-5", true, true, true},
		// Always-on thinking: explicit disablement is rejected by the API.
		{"claude-fable-5", true, true, false},
		{"claude-mythos-5", true, true, false},

		// Budget generation — budget_tokens still honored.
		{"claude-opus-4-6", true, false, true},
		{"claude-sonnet-4-6", true, false, true},
		{"claude-haiku-4-5", true, false, true},

		// Dated snapshots resolve to the same generation as the bare alias.
		{"claude-sonnet-4-5-20250929", true, false, true},
		{"claude-opus-4-5-20251101", true, false, true},
		{"claude-haiku-4-5-20251001", true, false, true},

		// Unlisted: no assumed generation.
		{"claude-3-haiku-20240307", false, false, false}, // no thinking control at all
		{"claude-opus-9", false, false, false},
		{"", false, false, false},
		// A trailing non-date segment must NOT be stripped as a snapshot.
		{"claude-sonnet-4-5-preview", false, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			got, known := AnthropicThinkingStyleFor(tt.model)
			if known != tt.wantKnown {
				t.Fatalf("known = %v, want %v (registered: %v)",
					known, tt.wantKnown, anthropicThinkingStyleModels())
			}
			if !known {
				return
			}
			if got.Adaptive != tt.wantAdaptive {
				t.Errorf("Adaptive = %v, want %v", got.Adaptive, tt.wantAdaptive)
			}
			if got.CanDisable != tt.wantDisable {
				t.Errorf("CanDisable = %v, want %v", got.CanDisable, tt.wantDisable)
			}
		})
	}
}

// TestAnthropicEffortLevel pins the mapping onto Anthropic's output_config
// vocabulary. "off" has no effort value — it is disablement, not a level.
func TestAnthropicEffortLevel(t *testing.T) {
	for _, tt := range []struct {
		effort string
		want   string
		wantOK bool
	}{
		{ReasoningEffortLow, "low", true},
		{ReasoningEffortMedium, "medium", true},
		{ReasoningEffortHigh, "high", true},
		{ReasoningEffortOff, "", false},
		{"", "", false},
		{"xhigh", "", false}, // not in this resolver's vocabulary
	} {
		got, ok := AnthropicEffortLevel(tt.effort)
		if got != tt.want || ok != tt.wantOK {
			t.Errorf("AnthropicEffortLevel(%q) = (%q, %v), want (%q, %v)",
				tt.effort, got, ok, tt.want, tt.wantOK)
		}
	}
}

// TestAnthropicThinkingStyle_CoversCapabilityTable is the invariant guard for
// the M7 live-smoke registration step: the moment an Anthropic model is
// declared reasoning-capable, its thinking-control generation MUST be known.
// Registering a 4.7+ model without classifying it is what would send the
// removed budget_tokens knob to the API and get a hard 400 on every benchmark.
//
// The shipped capability table is empty, so this currently passes vacuously —
// that is the point: it fails the moment someone populates it half-way.
func TestAnthropicThinkingStyle_CoversCapabilityTable(t *testing.T) {
	for model := range reasoningCapabilities[reasoningProviderAnthropic] {
		if _, known := AnthropicThinkingStyleFor(model); !known {
			t.Errorf("model %q is registered as reasoning-capable but has no thinking-control "+
				"generation; add it to anthropicThinkingStyles in reasoning_anthropic.go "+
				"(registered generations: %v)", model, anthropicThinkingStyleModels())
		}
	}
}

// TestResolveReasoning_AnthropicFixedBudgetByGeneration pins that an explicit
// thinking_budget_tokens is rejected with an actionable message on Opus 4.7+
// (which removed the knob) and still honored on 4.6-and-older.
func TestResolveReasoning_AnthropicFixedBudgetByGeneration(t *testing.T) {
	orig := reasoningCapabilities
	defer func() { reasoningCapabilities = orig }()
	reasoningCapabilities = map[string]map[string]map[string]bool{
		reasoningProviderAnthropic: {
			"claude-opus-5": {
				ReasoningEffortOff: true, ReasoningEffortLow: true,
				ReasoningEffortMedium: true, ReasoningEffortHigh: true,
			},
			"claude-sonnet-4-6": {
				ReasoningEffortOff: true, ReasoningEffortHigh: true,
			},
		},
	}

	// Adaptive generation + a fixed budget: rejected BEFORE dispatch, and the
	// message must name the real cause (not a MaxTokens complaint, which would
	// send the caller off to fix the wrong thing).
	_, err := ResolveReasoning(
		&Request{MaxTokens: 64000, Options: map[string]any{"thinking_budget_tokens": 8192}},
		reasoningProviderAnthropic, "claude-opus-5")
	if !errors.Is(err, ErrUnsupportedReasoningEffort) {
		t.Fatalf("opus-5 + fixed budget: err = %v, want ErrUnsupportedReasoningEffort", err)
	}
	if !strings.Contains(err.Error(), "does not support a fixed thinking budget") {
		t.Errorf("opus-5 + fixed budget: message %q should name the removed knob", err.Error())
	}

	// Same request with MaxTokens unset must give the SAME root-cause error,
	// not ErrReasoningBudgetExceedsMaxTokens — the budget is unsupported here
	// at any MaxTokens.
	_, err = ResolveReasoning(
		&Request{Options: map[string]any{"thinking_budget_tokens": 8192}},
		reasoningProviderAnthropic, "claude-opus-5")
	if !errors.Is(err, ErrUnsupportedReasoningEffort) {
		t.Errorf("opus-5 + fixed budget, no MaxTokens: err = %v, want ErrUnsupportedReasoningEffort", err)
	}

	// Budget generation: unchanged, still honored exactly.
	dec, err := ResolveReasoning(
		&Request{MaxTokens: 64000, Options: map[string]any{"thinking_budget_tokens": 8192}},
		reasoningProviderAnthropic, "claude-sonnet-4-6")
	if err != nil {
		t.Fatalf("sonnet-4-6 + fixed budget: unexpected error %v", err)
	}
	if dec.Kind != ReasoningBudgetKind || dec.Budget != 8192 || !dec.BudgetSet {
		t.Errorf("sonnet-4-6 + fixed budget: decision = %+v, want budget 8192", dec)
	}

	// Budget 0 on an adaptive model is EXEMPT — that is disablement, which is
	// expressible as an explicit thinking:{type:"disabled"}.
	dec, err = ResolveReasoning(
		&Request{Options: map[string]any{"thinking_budget_tokens": 0}},
		reasoningProviderAnthropic, "claude-opus-5")
	if err != nil {
		t.Fatalf("opus-5 + budget 0: unexpected error %v", err)
	}
	if dec.Kind != ReasoningBudgetKind || dec.Budget != 0 || !dec.BudgetSet {
		t.Errorf("opus-5 + budget 0: decision = %+v, want budget 0 set", dec)
	}

	// A qualitative effort on an adaptive model resolves normally — the client
	// turns Effort (not Budget) into the wire shape.
	dec, err = ResolveReasoning(
		&Request{ReasoningEffort: ReasoningEffortHigh, MaxTokens: 64000},
		reasoningProviderAnthropic, "claude-opus-5")
	if err != nil {
		t.Fatalf("opus-5 + effort high: unexpected error %v", err)
	}
	if dec.Kind != ReasoningEffortKind || dec.Effort != ReasoningEffortHigh {
		t.Errorf("opus-5 + effort high: decision = %+v", dec)
	}

	// An unregistered model is still rejected by the capability gate first,
	// regardless of generation classification.
	_, err = ResolveReasoning(
		&Request{MaxTokens: 64000, Options: map[string]any{"thinking_budget_tokens": 8192}},
		reasoningProviderAnthropic, "claude-opus-4-8")
	if !errors.Is(err, ErrUnsupportedReasoningEffort) {
		t.Errorf("unregistered opus-4-8: err = %v, want ErrUnsupportedReasoningEffort", err)
	}
}
