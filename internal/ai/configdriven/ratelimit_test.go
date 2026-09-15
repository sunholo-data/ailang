package configdriven

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sunholo-data/ailang/internal/ai"
)

// A 429 from a config-driven provider used to bypass ai.ClassifyHTTPError
// entirely (Generate returned a bare ProviderError and ClassifyError read only
// the message), so it landed as CodeInternal and never became CodeRateLimit.
// It now classifies as a rate limit; a 429 whose body says the quota is spent
// classifies as BudgetExhausted and is not retried (M-V1-SIMPLIFY-S3 M4).
func TestGenerate_429ClassifiesAsRateLimit(t *testing.T) {
	t.Setenv("TEST_PROVIDER_KEY", "sk-test")

	serve := func(body string) *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(body))
		}))
	}

	transient := serve(`{"error": {"message": "Rate limit reached for requests", "type": "rate_limit_error"}}`)
	defer transient.Close()
	_, err := New(openaiChatSpec(transient.URL)).Generate(context.Background(), &ai.Request{Model: "x", UserPrompt: "Hi"})
	if err == nil {
		t.Fatal("expected error")
	}
	var perr *ai.ProviderError
	if !errors.As(err, &perr) || perr.StatusCode != 429 {
		t.Fatalf("want ProviderError 429, got %v", err)
	}
	if e := ai.ClassifyError(err); e.Code != ai.CodeRateLimit || !e.Retryable {
		t.Fatalf("classified %s/%v, want RateLimit/retryable", e.Code, e.Retryable)
	}
	if !ai.ShouldRetry(err) {
		t.Fatal("a transient 429 should retry")
	}

	spent := serve(`{"error":{"message":"you (marked) have reached your session usage limit, upgrade for higher limits: https://ollama.com/upgrade","type":"api_error"}}`)
	defer spent.Close()
	_, err = New(openaiChatSpec(spent.URL)).Generate(context.Background(), &ai.Request{Model: "x", UserPrompt: "Hi"})
	if e := ai.ClassifyError(err); e == nil || e.Code != ai.CodeBudgetExhausted || e.Retryable {
		t.Fatalf("spent quota classified %+v, want BudgetExhausted/non-retryable", e)
	}
	if ai.ShouldRetry(err) {
		t.Fatal("a spent quota 429 must not retry into a bucket that cannot recover")
	}
}
