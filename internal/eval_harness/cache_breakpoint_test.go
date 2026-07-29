package eval_harness

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/ai"
)

// M-ANTHROPIC-CACHE-HIT-RATE M2 — the D3 default-ON policy and its guards.
//
// The regression this file exists to prevent is subtle and was hit during
// implementation: gating on "will THIS adapter make >= 2 calls" looks correct
// and silently disables caching everywhere, because the eval harness builds a
// fresh AIAgent per benchmark while Anthropic's cache lives server-side.

func TestCacheBreakpointsFor(t *testing.T) {
	const prefix = "a long stable teaching prompt"

	cases := []struct {
		name           string
		provider       ai.ProviderType
		cachedPrefix   string
		expectedCalls  int
		wantBreakpoint bool
	}{
		{
			// The load-bearing case: one call per adapter is the NORMAL eval
			// shape, and it must still cache.
			name:           "anthropic_undeclared_calls_caches",
			provider:       ai.ProviderAnthropic,
			cachedPrefix:   prefix,
			expectedCalls:  0,
			wantBreakpoint: true,
		},
		{
			name:           "anthropic_declared_one_shot_does_not_cache",
			provider:       ai.ProviderAnthropic,
			cachedPrefix:   prefix,
			expectedCalls:  1,
			wantBreakpoint: false,
		},
		{
			name:           "anthropic_multi_call_caches",
			provider:       ai.ProviderAnthropic,
			cachedPrefix:   prefix,
			expectedCalls:  30,
			wantBreakpoint: true,
		},
		{
			name:           "no_prefix_nothing_to_cache",
			provider:       ai.ProviderAnthropic,
			cachedPrefix:   "",
			expectedCalls:  30,
			wantBreakpoint: false,
		},
		{
			// Other providers no-op on hints; declaring one buys nothing and
			// emits a per-session "hint ignored" warning.
			name:           "openai_does_not_declare",
			provider:       ai.ProviderOpenAI,
			cachedPrefix:   prefix,
			expectedCalls:  30,
			wantBreakpoint: false,
		},
		{
			name:           "gemini_does_not_declare",
			provider:       ai.ProviderGoogle,
			cachedPrefix:   prefix,
			expectedCalls:  30,
			wantBreakpoint: false,
		},
		{
			name:           "ollama_does_not_declare",
			provider:       ai.ProviderOllama,
			cachedPrefix:   prefix,
			expectedCalls:  30,
			wantBreakpoint: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := cacheBreakpointsFor(tc.provider, tc.cachedPrefix, tc.expectedCalls)
			if tc.wantBreakpoint {
				if len(got) != 1 {
					t.Fatalf("got %d breakpoints, want exactly 1", len(got))
				}
				if got[0].Position != "user_prefix" {
					t.Errorf("breakpoint position = %q, want \"user_prefix\" — the cacheable content is the teaching prompt in the USER turn, not the ~70-token system prompt", got[0].Position)
				}
				if got[0].TTL != "ephemeral" {
					t.Errorf("breakpoint TTL = %q, want \"ephemeral\"", got[0].TTL)
				}
			} else if len(got) != 0 {
				t.Errorf("got %d breakpoints, want none", len(got))
			}
		})
	}
}
