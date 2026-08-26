package eval_harness

import (
	"errors"
	"testing"
)

// ollamaSessionLimitBody is the EXACT body Ollama Cloud returns on session
// exhaustion, captured verbatim 2026-08-26 by deliberately spending a 5-hour
// window (M-OLLAMA-CLOUD V22). Only the ref UUID varies between occurrences.
//
// Two properties make this the hard case, and both are why the design quorum
// refused an earlier draft that GUESSED the shape:
//   - it arrives as HTTP 429, the same status as a transient rate-limit
//   - its type is "api_error", not a distinct quota type
//
// So neither the status nor the type can discriminate. Only the message can.
const ollamaSessionLimitBody = `429 {"error":{"message":"you (marked) have reached your session usage limit, ` +
	`upgrade for higher limits: https://ollama.com/upgrade or add extra usage: https://ollama.com/settings ` +
	`(ref: f47a0d10-34b5-4c56-9b96-350d29d791af)","type":"api_error","param":null,"code":null}}`

// TestQuotaExhaustionIsNotRetried is AC8. Before the carve-out, isRetryableError
// matched any error containing "429" and returned true — so the harness would
// retry into a bucket that cannot recover for hours.
func TestQuotaExhaustionIsNotRetried(t *testing.T) {
	if isRetryableError(errors.New(ollamaSessionLimitBody)) {
		t.Error("Ollama session-limit 429 was treated as RETRYABLE — the harness would " +
			"retry into a spent bucket that does not clear until the 5-hour window rolls")
	}
}

// TestTransientRateLimitStillRetried is the other direction, and matters as much:
// over-broad matching would stop retrying genuine transient limits, turning a
// recoverable blip into a failed trial.
func TestTransientRateLimitStillRetried(t *testing.T) {
	for _, msg := range []string{
		"429 Too Many Requests",
		"rate limit exceeded, please slow down",
		"429 {\"error\":{\"message\":\"Rate limit reached for requests\",\"type\":\"rate_limit_error\"}}",
	} {
		if !isRetryableError(errors.New(msg)) {
			t.Errorf("transient rate-limit should still be retryable: %q", msg)
		}
	}
}

// TestQuotaExhaustionCategorised is AC7. `api_error` means "cause unknown" per
// CLAUDE.md, and `rate_limit` means "transient" — banking exhaustion as either
// loses the one fact that decides whether to re-run on the other route.
func TestQuotaExhaustionCategorised(t *testing.T) {
	got := CategorizeAgentError(errors.New(ollamaSessionLimitBody), "")
	if got != ErrorCategoryQuotaExhausted {
		t.Errorf("CategorizeError = %q, want %q — exhaustion must not bank as "+
			"rate_limit (transient) or api_error (cause unknown)", got, ErrorCategoryQuotaExhausted)
	}
}

// TestWeeklyLimitAlsoRecognised: the session window rolls every 5 hours, but the
// weekly one takes days. Both are exhaustion; missing the weekly variant would
// reintroduce the retry bug on the longer, more expensive window.
func TestWeeklyLimitAlsoRecognised(t *testing.T) {
	weekly := `429 {"error":{"message":"you (marked) have reached your weekly usage limit, ` +
		`upgrade for higher limits: https://ollama.com/upgrade","type":"api_error"}}`
	if isRetryableError(errors.New(weekly)) {
		t.Error("weekly usage limit was treated as retryable")
	}
	if got := CategorizeAgentError(errors.New(weekly), ""); got != ErrorCategoryQuotaExhausted {
		t.Errorf("weekly limit categorised %q, want %q", got, ErrorCategoryQuotaExhausted)
	}
}

// TestOtherProvidersStillMatch guards the pre-existing matchers that the shared
// helper absorbed — the refactor must not have dropped any.
func TestOtherProvidersStillMatch(t *testing.T) {
	for _, msg := range []string{
		"key limit exceeded", "monthly limit reached",
		"insufficient_quota", "quota exceeded", "billing issue",
	} {
		if !isQuotaExhaustion(msg) {
			t.Errorf("pre-existing quota matcher lost in the refactor: %q", msg)
		}
	}
}
