package openai

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sunholo-data/ailang/internal/ai"
	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/riglock"
)

// The seam, not the helper: a real Generate against a local model endpoint
// (motoko's OPENAI_BASE_URL=http://localhost:11434/v1) must carry the holder's
// lease, or the rig gateway refuses the holder's own eval with 423.
func TestGenerate_CarriesRigLeaseToLocalEndpoint(t *testing.T) {
	const tok = "0123456789abcdef0123456789abcdef"
	t.Setenv(config.EnvRigLease, tok)
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get(riglock.LeaseHeader)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"c","object":"chat.completion","model":"qwen3.8:27b",
			"choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}],
			"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`))
	}))
	defer srv.Close()

	c := NewClient("", WithBaseURL(srv.URL))
	if _, err := c.Generate(context.Background(), &ai.Request{Model: "gpt-4o", UserPrompt: "hi", MaxTokens: 8}); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if got != tok {
		t.Fatalf("X-Rig-Lease = %q, want the lease %q", got, tok)
	}
}
