package codex

import "testing"

// TestSplitCodexInputTokens pins the property that matters downstream:
// cmd/ailang/eval_benchmark.go banks agent-mode input as
// InputTokens + CacheCreationInputTokens + CacheReadInputTokens, so the split
// MUST be total-preserving. codex's cached_input_tokens is a subset of
// input_tokens (OpenAI Responses API semantics), which is why it is split out
// rather than added on top.
func TestSplitCodexInputTokens(t *testing.T) {
	tests := []struct {
		name       string
		input      int
		cached     int
		wantFresh  int
		wantCached int
	}{
		{"no cache reported", 1000, 0, 1000, 0},
		{"partial cache", 1000, 400, 600, 400},
		{"fully cached", 1000, 1000, 0, 1000},
		{"zero usage", 0, 0, 0, 0},
		// Defensive: a malformed stream must never yield negative InputTokens.
		{"cached exceeds input", 500, 900, 0, 500},
		{"negative cached ignored", 800, -5, 800, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fresh, cached := splitCodexInputTokens(tt.input, tt.cached)
			if fresh != tt.wantFresh || cached != tt.wantCached {
				t.Errorf("splitCodexInputTokens(%d, %d) = (%d, %d), want (%d, %d)",
					tt.input, tt.cached, fresh, cached, tt.wantFresh, tt.wantCached)
			}
			if fresh < 0 {
				t.Errorf("fresh = %d, must never be negative", fresh)
			}
			// The invariant the eval banking layer depends on.
			if tt.input >= 0 && tt.cached >= 0 && fresh+cached != tt.input {
				t.Errorf("fresh+cached = %d, want %d (split must preserve the input total)",
					fresh+cached, tt.input)
			}
		})
	}
}
