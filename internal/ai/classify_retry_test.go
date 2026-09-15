package ai

import (
	"context"
	"errors"
	"fmt"
	"testing"
)

// ollamaSessionLimitBody is the EXACT body Ollama Cloud returns on session
// exhaustion (captured 2026-08-26, M-OLLAMA-CLOUD V22): HTTP 429, type
// "api_error" — indistinguishable from a transient rate limit by status or
// type. Only the message separates them.
const ollamaSessionLimitBody = `{"error":{"message":"you (marked) have reached your session usage limit, ` +
	`upgrade for higher limits: https://ollama.com/upgrade or add extra usage: https://ollama.com/settings ` +
	`(ref: f47a0d10-34b5-4c56-9b96-350d29d791af)","type":"api_error","param":null,"code":null}}`

// A 429 carried by a *ProviderError — the shape configdriven.Provider.Generate
// returns, which used to bypass ClassifyHTTPError entirely and land as
// CodeInternal — classifies as a rate limit, and retries.
func TestClassifyError_ProviderError429IsRateLimit(t *testing.T) {
	err := NewProviderError("my-config-provider", 429, "Rate limit reached for requests", nil)
	got := ClassifyError(err)
	if got.Code != CodeRateLimit {
		t.Fatalf("Code = %q, want %q", got.Code, CodeRateLimit)
	}
	if !got.Retryable || !ShouldRetry(err) {
		t.Fatalf("a transient 429 must be retryable (Retryable=%v ShouldRetry=%v)", got.Retryable, ShouldRetry(err))
	}
	// Wrapped once more (the way callers annotate) still classifies structurally.
	wrapped := fmt.Errorf("generate: %w", err)
	if c := ClassifyError(wrapped).Code; c != CodeRateLimit {
		t.Fatalf("wrapped: Code = %q, want %q", c, CodeRateLimit)
	}
}

// The trap AC8 named: a 429 whose message says the bucket is SPENT must not
// retry — session windows refill in hours, weekly ones in days. This holds for
// the structured path (ProviderError with status), the HTTP path, and a bare
// string error that merely contains "429".
func TestClassifyError_QuotaExhaustion429IsNotRetried(t *testing.T) {
	cases := map[string]error{
		"provider-error": NewProviderError("ollama", 429, ollamaSessionLimitBody, nil),
		"http":           ClassifyHTTPError("ollama", 429, ollamaSessionLimitBody),
		"string":         errors.New("429 " + ollamaSessionLimitBody),
		"weekly":         errors.New(`429 {"error":{"message":"you (marked) have reached your weekly usage limit, upgrade for higher limits: https://ollama.com/upgrade","type":"api_error"}}`),
		"payment-402":    NewProviderError("openrouter", 402, "Insufficient credits", nil),
	}
	for name, err := range cases {
		got := ClassifyError(err)
		if got.Code != CodeBudgetExhausted {
			t.Errorf("%s: Code = %q, want %q", name, got.Code, CodeBudgetExhausted)
		}
		if got.Retryable || ShouldRetry(err) {
			t.Errorf("%s: treated as RETRYABLE — the harness would retry into a spent bucket", name)
		}
	}
}

// The other direction matters as much: over-broad matching would stop
// retrying genuine transient limits.
func TestShouldRetry_TransientSignals(t *testing.T) {
	for _, msg := range []string{
		"429 Too Many Requests",
		"rate limit exceeded, please slow down",
		`429 {"error":{"message":"Rate limit reached for requests","type":"rate_limit_error"}}`,
		"status code 429",
		"connection timeout",
		"network error",
		"dial tcp: connection refused",
		"500 internal server error",
		"502 bad gateway",
		"openai error (503): overloaded",
	} {
		if !ShouldRetry(errors.New(msg)) {
			t.Errorf("transient error should retry: %q", msg)
		}
	}
	if !ShouldRetry(context.DeadlineExceeded) {
		t.Error("context.DeadlineExceeded should retry")
	}
}

// ShouldRetry is conservative where ClassifyError is not: an error nobody
// recognised is CodeInternal+Retryable for AILANG callers (a new adapter code
// must not read as fatal) but is NOT retried by a loop that spends money.
func TestShouldRetry_UnknownAndNonTransient(t *testing.T) {
	for _, msg := range []string{
		"syntax error in code",
		"invalid input",
		"model produced 1500 tokens", // "500" inside a number is not a status
		"request id 4290 failed",     // nor is 429
	} {
		err := errors.New(msg)
		if ShouldRetry(err) {
			t.Errorf("should NOT retry: %q", msg)
		}
		if c := ClassifyError(err); c.Code != CodeInternal || !c.Retryable {
			t.Errorf("ClassifyError(%q) = %s/%v, want Internal/retryable (AILANG-level default)", msg, c.Code, c.Retryable)
		}
	}
	if ShouldRetry(nil) {
		t.Error("nil never retries")
	}
	for _, err := range []error{
		NewProviderError("anthropic", 401, "invalid x-api-key", nil),
		NewProviderError("openai", 400, "context length exceeded", nil),
		NewAIError(CodeSchemaValidation, "bad schema", false),
	} {
		if ShouldRetry(err) {
			t.Errorf("non-transient must not retry: %v", err)
		}
	}
}
