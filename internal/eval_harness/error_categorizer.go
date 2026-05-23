package eval_harness

import "strings"

// CategorizeAgentError classifies an agent-mode failure into one of the typed
// ErrorCategory* values defined in metrics.go (M-EVAL-SWEET-SPOT, v0.19.0).
//
// Inputs:
//   - err:          the error returned from the agent executor, or nil
//   - finishReason: the executor finish_reason (e.g. motoko "cost_exhausted",
//     agent runner "step_exhausted"), or empty string
//
// Precedence: finishReason wins over err. Executors that emit a structured
// finish_reason know more about why they stopped than the surrounding RPC
// error wrapping, so we trust the structured signal first.
//
// Returns the most specific known category; falls back to ErrorCategoryAPI
// when neither signal points to a known cause. ErrorCategoryAPI is preserved
// as the catch-all so callers can distinguish "I tried to classify and
// couldn't" from a typed bucket.
//
// Pure function — safe to invoke for offline re-categorization of historical
// result JSONs.
func CategorizeAgentError(err error, finishReason string) string {
	// Structured finish signals from the executor are authoritative.
	switch finishReason {
	case "cost_exhausted":
		return ErrorCategoryCostKilled
	case "step_exhausted":
		return ErrorCategoryStepExhausted
	case "timeout":
		return ErrorCategoryTimeout
	case "thrash_aborted":
		return ErrorCategoryThrashAborted // M-EVAL-OS-LONGITUDINAL Phase 1
	}

	// Fallback: detect by error-message substring when finish_reason wasn't
	// plumbed through (e.g. opencode executor reports the kill via Error string).
	if err != nil && strings.Contains(strings.ToLower(err.Error()), "thrash abort") {
		return ErrorCategoryThrashAborted
	}

	if err == nil {
		// No error string and no recognized finish_reason: nothing to classify.
		return ErrorCategoryAPI
	}

	msg := strings.ToLower(err.Error())

	// Quota exhaustion — provider account/key cap. Distinct from a transient
	// 429: a quota kill says nothing about model capability and should be
	// excluded from capability scoring (see ShouldExcludeFromCapability).
	if containsAny(msg,
		"key limit exceeded",
		"monthly limit",
		"insufficient_quota",
		"insufficient quota",
		"quota exceeded",
		"billing",
	) {
		return ErrorCategoryQuotaExhausted
	}

	// Rate limit — transient. We treat it as provider-side noise and exclude
	// from capability scoring (the model didn't fail to solve, the provider
	// just throttled us).
	if containsAny(msg,
		"rate limit",
		"rate-limit",
		"too many requests",
		"429",
	) {
		return ErrorCategoryRateLimit
	}

	// Wall-clock / context-deadline timeout. Distinct from quota: a slow-but-
	// capable model just needed more time. This SHOULD show up in capability
	// stats as a failure so operators see "give it more wall-clock budget."
	if containsAny(msg,
		"context deadline exceeded",
		"deadline exceeded",
		"timed out",
		"i/o timeout",
		"timeout",
	) {
		return ErrorCategoryTimeout
	}

	// Unknown — keep as the catch-all rather than guessing.
	return ErrorCategoryAPI
}

// containsAny returns true if s contains any of the substrings in needles.
// Case-sensitive — callers normalize first if needed.
func containsAny(s string, needles ...string) bool {
	for _, n := range needles {
		if strings.Contains(s, n) {
			return true
		}
	}
	return false
}
