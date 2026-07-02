package eval_harness

import (
	"errors"
	"testing"
)

// TestCategorizeAgentError uses real error strings sampled from the
// v0_18_5_core_3harness eval dataset plus synthetic edge cases.
func TestCategorizeAgentError(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		finishReason string
		want         string
	}{
		// Structured finish_reason wins.
		{
			name:         "finish_reason cost_exhausted",
			err:          errors.New("motoko returned non-zero exit"),
			finishReason: "cost_exhausted",
			want:         ErrorCategoryCostKilled,
		},
		{
			name:         "finish_reason step_exhausted",
			err:          nil,
			finishReason: "step_exhausted",
			want:         ErrorCategoryStepExhausted,
		},
		{
			// M-RIG-RELIABILITY M2: the motoko executor passes "max_steps" through
			// verbatim (with a "motoko terminated with finish_reason=max_steps" error).
			// This is a real completed run that hit the step cap — it MUST classify as
			// step_exhausted, not api_error (the bug that made docx unmeasurable).
			name:         "finish_reason max_steps (motoko vocabulary)",
			err:          errors.New("motoko terminated with finish_reason=max_steps"),
			finishReason: "max_steps",
			want:         ErrorCategoryStepExhausted,
		},
		{
			// M-RIG-RELIABILITY M2: the motoko v2 loop reports step exhaustion ONLY
			// as an error string with an EMPTY finish_reason (this exact string was
			// pulled from a failing docx result JSON). It MUST classify as
			// step_exhausted, not api_error — the run happened, it just hit the cap.
			name:         "step budget exhausted error string, empty finish_reason",
			err:          errors.New(`executor "motoko" failed for model "ollama/qwen3.6:35b-a3b-mxfp8": v2 loop: step budget exhausted`),
			finishReason: "",
			want:         ErrorCategoryStepExhausted,
		},
		{
			name:         "finish_reason timeout",
			err:          nil,
			finishReason: "timeout",
			want:         ErrorCategoryTimeout,
		},

		// OpenRouter monthly quota — real string from
		// eval_results/v0_18_5_core_3harness/agent/*.json (44 occurrences).
		{
			name:         "openrouter monthly quota",
			err:          errors.New(`executor "motoko" failed for model "openrouter/anthropic/claude-haiku-4-5": {"error":{"message":"Key limit exceeded (monthly limit). Manage it using https://openrouter.ai/settings/keys","code":403}}`),
			finishReason: "",
			want:         ErrorCategoryQuotaExhausted,
		},
		{
			name:         "openai insufficient_quota",
			err:          errors.New(`openai api error: insufficient_quota — please add credits`),
			finishReason: "",
			want:         ErrorCategoryQuotaExhausted,
		},

		// Rate limit / 429.
		{
			name:         "http 429",
			err:          errors.New("429 Too Many Requests"),
			finishReason: "",
			want:         ErrorCategoryRateLimit,
		},
		{
			name:         "rate limit phrase",
			err:          errors.New("anthropic: rate limit reached for model claude-sonnet"),
			finishReason: "",
			want:         ErrorCategoryRateLimit,
		},

		// Timeout.
		{
			name:         "context deadline exceeded",
			err:          errors.New("post https://api.openai.com: context deadline exceeded"),
			finishReason: "",
			want:         ErrorCategoryTimeout,
		},
		{
			name:         "i/o timeout",
			err:          errors.New("read tcp 1.2.3.4:443: i/o timeout"),
			finishReason: "",
			want:         ErrorCategoryTimeout,
		},
		{
			name:         "generic timed out",
			err:          errors.New("operation timed out after 60s"),
			finishReason: "",
			want:         ErrorCategoryTimeout,
		},

		// Fallback / unknown.
		{
			name:         "500 server error not auto-classified",
			err:          errors.New("500 internal server error"),
			finishReason: "",
			want:         ErrorCategoryAPI,
		},
		{
			name:         "nil error nil finish",
			err:          nil,
			finishReason: "",
			want:         ErrorCategoryAPI,
		},
		{
			name:         "unknown finish_reason value",
			err:          errors.New("some random failure"),
			finishReason: "weird_unknown_reason",
			want:         ErrorCategoryAPI,
		},

		// Precedence: structured finish_reason overrides err string.
		{
			name:         "finish step_exhausted overrides timeout-looking err",
			err:          errors.New("context deadline exceeded"),
			finishReason: "step_exhausted",
			want:         ErrorCategoryStepExhausted,
		},

		// Case-insensitive matching.
		{
			name:         "uppercase RATE LIMIT",
			err:          errors.New("RATE LIMIT EXCEEDED"),
			finishReason: "",
			want:         ErrorCategoryRateLimit,
		},
		{
			name:         "mixed case Key Limit",
			err:          errors.New("Key Limit Exceeded (Monthly Limit)"),
			finishReason: "",
			want:         ErrorCategoryQuotaExhausted,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CategorizeAgentError(tt.err, tt.finishReason)
			if got != tt.want {
				t.Errorf("CategorizeAgentError(err=%v, finish=%q) = %q, want %q",
					tt.err, tt.finishReason, got, tt.want)
			}
		})
	}
}

// TestCategorizeAgentError_QuotaTakesPrecedenceOverRateLimit verifies that
// when an error string contains both "Key limit exceeded" AND the literal
// "429", we pick quota_exhausted (the more specific, less recoverable cause).
func TestCategorizeAgentError_QuotaTakesPrecedenceOverRateLimit(t *testing.T) {
	err := errors.New("Key limit exceeded (monthly limit). status 429.")
	got := CategorizeAgentError(err, "")
	if got != ErrorCategoryQuotaExhausted {
		t.Errorf("expected quota_exhausted to win over rate_limit, got %q", got)
	}
}

// TestErrorCategoryConstants_StableValues guards against renaming the
// canonical string values, which would invalidate every shipped JSON.
func TestErrorCategoryConstants_StableValues(t *testing.T) {
	expected := map[string]string{
		"ErrorCategoryTimeout":        "timeout",
		"ErrorCategoryQuotaExhausted": "quota_exhausted",
		"ErrorCategoryRateLimit":      "rate_limit",
		"ErrorCategoryCostKilled":     "cost_killed",
		"ErrorCategoryStepExhausted":  "step_exhausted",
		"ErrorCategoryAPI":            "api_error",
	}
	actual := map[string]string{
		"ErrorCategoryTimeout":        ErrorCategoryTimeout,
		"ErrorCategoryQuotaExhausted": ErrorCategoryQuotaExhausted,
		"ErrorCategoryRateLimit":      ErrorCategoryRateLimit,
		"ErrorCategoryCostKilled":     ErrorCategoryCostKilled,
		"ErrorCategoryStepExhausted":  ErrorCategoryStepExhausted,
		"ErrorCategoryAPI":            ErrorCategoryAPI,
	}
	for k, want := range expected {
		if actual[k] != want {
			t.Errorf("constant %s = %q, want %q (renaming breaks shipped JSON)", k, actual[k], want)
		}
	}
}

// TestCategorizeAgentError_ThrashAborted covers the M-EVAL-OS-LONGITUDINAL
// Phase 1 wiring: when the opencode executor kills a thrashing subprocess
// because cumulative tokens exceeded Task.MaxTokensPerBench, the error
// must categorize to ErrorCategoryThrashAborted whether the signal arrives
// as finish_reason or as a substring of the error message.
func TestCategorizeAgentError_ThrashAborted(t *testing.T) {
	cases := []struct {
		name         string
		err          error
		finishReason string
		want         string
	}{
		{
			name:         "finish_reason thrash_aborted",
			err:          errors.New("opencode exited with error: signal: killed"),
			finishReason: "thrash_aborted",
			want:         ErrorCategoryThrashAborted,
		},
		{
			name:         "error string contains 'thrash abort' (opencode kill path)",
			err:          errors.New("thrash abort: cumulative tokens 600000 exceeded MaxTokensPerBench=500000 — opencode exited with error: signal: killed"),
			finishReason: "",
			want:         ErrorCategoryThrashAborted,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := CategorizeAgentError(tc.err, tc.finishReason)
			if got != tc.want {
				t.Errorf("CategorizeAgentError() = %q, want %q", got, tc.want)
			}
		})
	}
}
