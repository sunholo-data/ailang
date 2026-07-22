package ai

import (
	"errors"
	"testing"
)

// TestResolveReasoning_TableDriven is the AC15 resolver matrix: all five effort
// values, invalid values, invalid option types/ranges, every precedence combo
// of the four inputs (typed effort, reasoning_effort, thinking_budget_tokens,
// reasoning_max_tokens), reasoning_max_tokens on every provider and combined
// with each other control, unsupported models, MaxTokens==0, budget==MaxTokens,
// budget==MaxTokens-1, and Gemini/Anthropic B==0 with MaxTokens unset.
//
// The capability table is EMPTY, so any non-empty effort/enabled-budget on any
// model rejects with ErrUnsupportedReasoningEffort (fail-loud). "off"/B==0 also
// hits the capability gate (exact disablement must be verified too).
func TestResolveReasoning_TableDriven(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		model    string
		req      *Request
		wantKind ReasoningKind
		wantErr  error // sentinel; nil = expect success
	}{
		// --- all controls unset: preserve body (ReasoningNone) --------------
		{"unset_openai", reasoningProviderOpenAI, "gpt-5", &Request{}, ReasoningNone, nil},
		{"unset_gemini", reasoningProviderGemini, "gemini-3", &Request{}, ReasoningNone, nil},
		{"unset_anthropic", reasoningProviderAnthropic, "claude", &Request{}, ReasoningNone, nil},
		{"unset_openrouter", reasoningProviderOpenRouter, "x/y", &Request{}, ReasoningNone, nil},

		// --- invalid typed effort -------------------------------------------
		{"invalid_typed", reasoningProviderOpenAI, "gpt-5",
			&Request{ReasoningEffort: "bogus"}, ReasoningNone, ErrInvalidReasoningEffort},
		{"invalid_typed_caps", reasoningProviderGemini, "g",
			&Request{ReasoningEffort: "HIGH"}, ReasoningNone, ErrInvalidReasoningEffort},

		// --- invalid legacy reasoning_effort (non-string / bad value) -------
		{"legacy_effort_nonstring", reasoningProviderOpenAI, "gpt-5",
			&Request{Options: map[string]any{"reasoning_effort": 3}}, ReasoningNone, ErrInvalidReasoningEffort},
		{"legacy_effort_badvalue", reasoningProviderOpenRouter, "x/y",
			&Request{Options: map[string]any{"reasoning_effort": "turbo"}}, ReasoningNone, ErrInvalidReasoningEffort},

		// --- valid effort but capability table empty => unsupported ---------
		{"effort_off_unregistered", reasoningProviderOpenAI, "gpt-5",
			&Request{ReasoningEffort: "off"}, ReasoningNone, ErrUnsupportedReasoningEffort},
		{"effort_low_unregistered", reasoningProviderOpenRouter, "x/y",
			&Request{ReasoningEffort: "low"}, ReasoningNone, ErrUnsupportedReasoningEffort},
		{"effort_high_gemini_unregistered", reasoningProviderGemini, "gemini-3",
			&Request{ReasoningEffort: "high", MaxTokens: 20000}, ReasoningNone, ErrUnsupportedReasoningEffort},

		// --- precedence: typed + legacy identical => OK path (still gated) ---
		{"typed_legacy_identical_gated", reasoningProviderOpenAI, "gpt-5",
			&Request{ReasoningEffort: "medium", Options: map[string]any{"reasoning_effort": "medium"}},
			ReasoningNone, ErrUnsupportedReasoningEffort},
		// --- precedence: typed + legacy differ => conflict ------------------
		{"typed_legacy_conflict", reasoningProviderOpenAI, "gpt-5",
			&Request{ReasoningEffort: "high", Options: map[string]any{"reasoning_effort": "low"}},
			ReasoningNone, ErrConflictingReasoningConfig},

		// --- thinking_budget_tokens type checks -----------------------------
		{"budget_float_rejected", reasoningProviderGemini, "g",
			&Request{Options: map[string]any{"thinking_budget_tokens": 1024.0}}, ReasoningNone, ErrInvalidThinkingBudget},
		{"budget_string_rejected", reasoningProviderGemini, "g",
			&Request{Options: map[string]any{"thinking_budget_tokens": "1024"}}, ReasoningNone, ErrInvalidThinkingBudget},
		{"budget_negative_rejected", reasoningProviderGemini, "g",
			&Request{Options: map[string]any{"thinking_budget_tokens": -1}, MaxTokens: 5000}, ReasoningNone, ErrInvalidThinkingBudget},
		// Anthropic 1..1023 rejected
		{"anthropic_budget_below_min", reasoningProviderAnthropic, "claude",
			&Request{Options: map[string]any{"thinking_budget_tokens": 512}, MaxTokens: 5000}, ReasoningNone, ErrInvalidThinkingBudget},

		// --- thinking_budget_tokens on unsupported providers ----------------
		{"budget_on_openai_rejected", reasoningProviderOpenAI, "gpt-5",
			&Request{Options: map[string]any{"thinking_budget_tokens": 1024}}, ReasoningNone, ErrUnsupportedReasoningEffort},
		{"budget_on_openrouter_rejected", reasoningProviderOpenRouter, "x/y",
			&Request{Options: map[string]any{"thinking_budget_tokens": 1024}}, ReasoningNone, ErrUnsupportedReasoningEffort},

		// --- budget vs MaxTokens (Gemini/Anthropic), enabled thinking -------
		{"gemini_budget_maxtokens_unset", reasoningProviderGemini, "g",
			&Request{Options: map[string]any{"thinking_budget_tokens": 1024}}, ReasoningNone, ErrReasoningBudgetExceedsMaxTokens},
		{"gemini_budget_eq_maxtokens", reasoningProviderGemini, "g",
			&Request{Options: map[string]any{"thinking_budget_tokens": 1024}, MaxTokens: 1024}, ReasoningNone, ErrReasoningBudgetExceedsMaxTokens},
		{"anthropic_budget_eq_maxtokens", reasoningProviderAnthropic, "claude",
			&Request{Options: map[string]any{"thinking_budget_tokens": 1024}, MaxTokens: 1024}, ReasoningNone, ErrReasoningBudgetExceedsMaxTokens},
		// budget == MaxTokens-1 passes the MaxTokens gate (1023 < 1024) but hits
		// the empty capability table => unsupported.
		{"gemini_budget_maxtokens_minus1_capgate", reasoningProviderGemini, "g",
			&Request{Options: map[string]any{"thinking_budget_tokens": 1023}, MaxTokens: 1024}, ReasoningNone, ErrUnsupportedReasoningEffort},

		// --- B==0 exemption: no MaxTokens needed, but capability gate fires --
		{"gemini_budget_zero_maxtokens_unset_capgate", reasoningProviderGemini, "g",
			&Request{Options: map[string]any{"thinking_budget_tokens": 0}}, ReasoningNone, ErrUnsupportedReasoningEffort},
		{"anthropic_budget_zero_maxtokens_unset_capgate", reasoningProviderAnthropic, "claude",
			&Request{Options: map[string]any{"thinking_budget_tokens": 0}}, ReasoningNone, ErrUnsupportedReasoningEffort},

		// --- budget disagrees with effort => conflict -----------------------
		{"budget_effort_disagree", reasoningProviderGemini, "g",
			&Request{ReasoningEffort: "low", Options: map[string]any{"thinking_budget_tokens": 4096}, MaxTokens: 20000},
			ReasoningNone, ErrConflictingReasoningConfig},

		// --- reasoning_max_tokens (deprecated OpenRouter-only) --------------
		{"rmax_on_openai_rejected", reasoningProviderOpenAI, "gpt-5",
			&Request{Options: map[string]any{"reasoning_max_tokens": 2000}}, ReasoningNone, ErrUnsupportedReasoningEffort},
		{"rmax_on_gemini_rejected", reasoningProviderGemini, "g",
			&Request{Options: map[string]any{"reasoning_max_tokens": 2000}}, ReasoningNone, ErrUnsupportedReasoningEffort},
		{"rmax_on_anthropic_rejected", reasoningProviderAnthropic, "claude",
			&Request{Options: map[string]any{"reasoning_max_tokens": 2000}}, ReasoningNone, ErrUnsupportedReasoningEffort},
		{"rmax_bad_type", reasoningProviderOpenRouter, "x/y",
			&Request{Options: map[string]any{"reasoning_max_tokens": "2000"}}, ReasoningNone, ErrInvalidThinkingBudget},
		{"rmax_below_one", reasoningProviderOpenRouter, "x/y",
			&Request{Options: map[string]any{"reasoning_max_tokens": 0}}, ReasoningNone, ErrInvalidThinkingBudget},
		// rmax alone on OpenRouter: preserved (today's body)
		{"rmax_alone_openrouter_ok", reasoningProviderOpenRouter, "x/y",
			&Request{Options: map[string]any{"reasoning_max_tokens": 2000}}, ReasoningMaxTokensKind, nil},
		// rmax + effort on OpenRouter => conflict
		{"rmax_plus_effort_conflict", reasoningProviderOpenRouter, "x/y",
			&Request{ReasoningEffort: "low", Options: map[string]any{"reasoning_max_tokens": 2000}},
			ReasoningNone, ErrConflictingReasoningConfig},
		{"rmax_plus_legacy_effort_conflict", reasoningProviderOpenRouter, "x/y",
			&Request{Options: map[string]any{"reasoning_effort": "low", "reasoning_max_tokens": 2000}},
			ReasoningNone, ErrConflictingReasoningConfig},
		// rmax + thinking_budget_tokens => ALWAYS conflict on every provider
		{"rmax_plus_budget_conflict_openrouter", reasoningProviderOpenRouter, "x/y",
			&Request{Options: map[string]any{"reasoning_max_tokens": 2000, "thinking_budget_tokens": 1024}},
			ReasoningNone, ErrConflictingReasoningConfig},
		{"rmax_plus_budget_conflict_gemini", reasoningProviderGemini, "g",
			&Request{Options: map[string]any{"reasoning_max_tokens": 2000, "thinking_budget_tokens": 1024}, MaxTokens: 9000},
			ReasoningNone, ErrConflictingReasoningConfig},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dec, err := ResolveReasoning(tt.req, tt.provider, tt.model)
			if tt.wantErr == nil {
				if err != nil {
					t.Fatalf("expected success, got error: %v", err)
				}
				if dec.Kind != tt.wantKind {
					t.Fatalf("kind = %v, want %v", dec.Kind, tt.wantKind)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error %v, got nil (decision %+v)", tt.wantErr, dec)
			}
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("errors.Is mismatch: got %v, want sentinel %v", err, tt.wantErr)
			}
			// Wire shape: must be a non-retryable *AIError with SchemaValidation.
			var aiErr *AIError
			if !errors.As(err, &aiErr) {
				t.Fatalf("error is not *AIError: %T", err)
			}
			if aiErr.Code != CodeSchemaValidation {
				t.Fatalf("AIError.Code = %q, want %q", aiErr.Code, CodeSchemaValidation)
			}
			if aiErr.Retryable {
				t.Fatalf("reasoning error must be non-retryable")
			}
		})
	}
}

// TestResolveReasoning_CapabilityRegistered proves the resolver honors a
// registered capability entry (positive path), using a temporary table override
// so the shipped table stays empty.
func TestResolveReasoning_CapabilityRegistered(t *testing.T) {
	orig := reasoningCapabilities
	defer func() { reasoningCapabilities = orig }()
	reasoningCapabilities = map[string]map[string]map[string]bool{
		reasoningProviderOpenAI: {
			"gpt-verified": {
				ReasoningEffortLow:    true,
				ReasoningEffortMedium: true,
				ReasoningEffortHigh:   true,
			},
		},
		reasoningProviderGemini: {
			"gemini-verified": {
				ReasoningEffortOff:  true,
				ReasoningEffortHigh: true,
			},
		},
	}

	// OpenAI qualitative effort on a registered model.
	dec, err := ResolveReasoning(&Request{ReasoningEffort: "high"}, reasoningProviderOpenAI, "gpt-verified")
	if err != nil {
		t.Fatalf("registered openai high: unexpected error %v", err)
	}
	if dec.Kind != ReasoningEffortKind || dec.Effort != "high" {
		t.Fatalf("registered openai high: got %+v", dec)
	}

	// Gemini effort maps to an absolute budget with MaxTokens > budget.
	dec, err = ResolveReasoning(&Request{ReasoningEffort: "high", MaxTokens: 20000}, reasoningProviderGemini, "gemini-verified")
	if err != nil {
		t.Fatalf("registered gemini high: unexpected error %v", err)
	}
	if dec.Kind != ReasoningEffortKind || dec.Budget != 16384 || !dec.BudgetSet {
		t.Fatalf("registered gemini high: got %+v", dec)
	}

	// Gemini "off" on a registered model => budget 0, no MaxTokens needed.
	dec, err = ResolveReasoning(&Request{ReasoningEffort: "off"}, reasoningProviderGemini, "gemini-verified")
	if err != nil {
		t.Fatalf("registered gemini off: unexpected error %v", err)
	}
	if dec.Kind != ReasoningEffortKind || dec.Budget != 0 || !dec.BudgetSet {
		t.Fatalf("registered gemini off: got %+v", dec)
	}

	// Registered gemini "high" but MaxTokens == budget => reject.
	_, err = ResolveReasoning(&Request{ReasoningEffort: "high", MaxTokens: 16384}, reasoningProviderGemini, "gemini-verified")
	if !errors.Is(err, ErrReasoningBudgetExceedsMaxTokens) {
		t.Fatalf("registered gemini high budget==maxtokens: want ErrReasoningBudgetExceedsMaxTokens, got %v", err)
	}

	// "medium" not in gemini-verified's set => unsupported.
	_, err = ResolveReasoning(&Request{ReasoningEffort: "medium", MaxTokens: 20000}, reasoningProviderGemini, "gemini-verified")
	if !errors.Is(err, ErrUnsupportedReasoningEffort) {
		t.Fatalf("registered gemini medium (not in set): want ErrUnsupportedReasoningEffort, got %v", err)
	}
}
