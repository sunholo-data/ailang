package eval_harness

import (
	"context"
	"testing"

	"github.com/sunholo-data/ailang/internal/ai"
)

// recordingProvider captures the last request so tests can assert on what the
// adapter actually sent, without any network.
type recordingProvider struct{ last *ai.Request }

func (p *recordingProvider) Generate(_ context.Context, req *ai.Request) (*ai.Response, error) {
	p.last = req
	return &ai.Response{Text: "ok", InputTokens: 1, OutputTokens: 1}, nil
}
func (p *recordingProvider) Step(_ context.Context, _ *ai.Request) (*ai.Response, error) {
	return nil, nil
}
func (p *recordingProvider) Name() string { return "recording" }

func newTestAgent(p ai.Provider, providerType ai.ProviderType) (*AIAgent, *providerAdapter) {
	ad := &providerAdapter{provider: p, providerType: providerType, model: "test-model", maxTokens: 4096}
	return &AIAgent{friendlyName: "test-model", model: "test-model", adapter: ad}, ad
}

// M-ANTHROPIC-CACHE-HIT-RATE D4: the warm-up narrows the output budget to make
// the prefill cheap. If it failed to restore the previous budget, every real
// generation after a warm-up would be truncated to a single token — a far worse
// bug than the cost it saves. Nothing else pins this.
func TestGenerateCodeWarmup_RestoresPreviousMaxTokens(t *testing.T) {
	p := &recordingProvider{}
	agent, ad := newTestAgent(p, ai.ProviderAnthropic)

	if _, err := agent.GenerateCodeWarmup(context.Background(), "PREFIX", "\n\n## Task\n\nok", 1); err != nil {
		t.Fatalf("warm-up failed: %v", err)
	}

	if p.last.MaxTokens != 1 {
		t.Errorf("warm-up sent MaxTokens=%d, want 1 — the prefill should be cheap", p.last.MaxTokens)
	}
	if ad.maxTokens != 4096 {
		t.Errorf("adapter budget left at %d after warm-up, want the original 4096 restored", ad.maxTokens)
	}
}

// The warm-up must carry the cacheable prefix as CachedPrefix (so a breakpoint
// can attach), not fold it into the user prompt.
func TestGenerateCodeWarmup_SendsPrefixAsCachedPrefix(t *testing.T) {
	p := &recordingProvider{}
	agent, _ := newTestAgent(p, ai.ProviderAnthropic)

	if _, err := agent.GenerateCodeWarmup(context.Background(), "PREFIX", "TASK", 1); err != nil {
		t.Fatalf("warm-up failed: %v", err)
	}
	if p.last.CachedPrefix != "PREFIX" {
		t.Errorf("CachedPrefix = %q, want %q", p.last.CachedPrefix, "PREFIX")
	}
	if len(p.last.CacheBreakpoints) != 1 {
		t.Errorf("warm-up declared %d breakpoints, want 1 — without one it caches nothing and is pure cost",
			len(p.last.CacheBreakpoints))
	}
	if got := p.last.FullUserPrompt(); got != "PREFIXTASK" {
		t.Errorf("model would see %q, want %q", got, "PREFIXTASK")
	}
}

// SetExpectedCalls(1) is the documented opt-out for one-shot callers.
func TestSetExpectedCalls_OptsOutOfCaching(t *testing.T) {
	p := &recordingProvider{}
	agent, _ := newTestAgent(p, ai.ProviderAnthropic)
	agent.SetExpectedCalls(1)

	if _, err := agent.GenerateCodeSplit(context.Background(), "PREFIX", "TASK"); err != nil {
		t.Fatalf("generate failed: %v", err)
	}
	if len(p.last.CacheBreakpoints) != 0 {
		t.Error("a declared one-shot run must not declare a cache breakpoint")
	}
	// The prefix must still REACH the model — opting out of caching is not
	// opting out of the prompt.
	if got := p.last.FullUserPrompt(); got != "PREFIXTASK" {
		t.Errorf("model would see %q, want %q", got, "PREFIXTASK")
	}
}
