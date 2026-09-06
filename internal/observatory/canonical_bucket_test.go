package observatory

import "testing"

// The bucket is parsed out of a FREE-TEXT agent_id, so nothing has ever constrained how it
// is written. Four spellings of codex accumulated in the v1 mission alone — harmless while
// the value is only displayed, fatal the moment anything RATIONS against it, because a
// limit computed over "codex" would see 70 stages and miss 11.
func TestCanonicalBucket_FoldsEverySpellingSeenLive(t *testing.T) {
	cases := []struct{ raw, want string }{
		// The four real codex spellings, copied from `chains stats --by-source-prefix`.
		{"codex", "codex"},
		{"codex-chatgpt", "codex"},
		{"Codex-OAuth", "codex"},
		{"codex-oauth", "codex"},
		{"chatgpt", "codex"},
		{"chatgpt-subscription", "codex"},
		{"codex-gpt-5.6-sol", "codex"},
		{"gpt-6-astra", "codex"},
		{"gpt-5.6-sol", "codex"},
		// Anthropic, which is likewise written several ways.
		{"opus", "anthropic"},
		{"fable", "anthropic"},
		{"sonnet", "anthropic"},
		{"weekly-sonnet", "anthropic"},
		{"claude-opus-5", "anthropic"},
		// The other two providers.
		{"openrouter-flat-rate", "openrouter"},
		{"ollama-minimax-m3-cloud", "ollama"},
	}
	for _, tc := range cases {
		if got := CanonicalQuotaBucket(tc.raw); got != tc.want {
			t.Errorf("CanonicalQuotaBucket(%q) = %q, want %q", tc.raw, got, tc.want)
		}
	}
}

// An unrecognised bucket must stay VISIBLE AS ITSELF. Folding it into the nearest real one
// is how a ration ends up measuring the wrong thing and saying nothing about it.
func TestCanonicalBucket_UnknownStaysItself(t *testing.T) {
	for _, raw := range []string{"some-new-provider", "mistral", "none"} {
		got := CanonicalQuotaBucket(raw)
		if got != raw {
			t.Errorf("CanonicalQuotaBucket(%q) = %q — an unknown bucket must not be folded into a neighbour", raw, got)
		}
	}
	if got := CanonicalQuotaBucket(""); got != "" {
		t.Errorf("empty must stay empty, got %q", got)
	}
	// Case and whitespace are normalised, which is not the same as folding.
	if got := CanonicalQuotaBucket("  Mistral  "); got != "mistral" {
		t.Errorf("expected trim+lower, got %q", got)
	}
}

// Canonicalising must not silently merge two DIFFERENT providers. This is the assertion
// that would fail if a future prefix rule got too greedy.
func TestCanonicalBucket_DistinctProvidersStayDistinct(t *testing.T) {
	seen := map[string]string{}
	for _, raw := range []string{"codex", "opus", "openrouter-flat-rate", "ollama-minimax-m3-cloud"} {
		c := CanonicalQuotaBucket(raw)
		if prev, dup := seen[c]; dup {
			t.Errorf("%q and %q both canonicalise to %q — two providers merged", prev, raw, c)
		}
		seen[c] = raw
	}
	if len(seen) != 4 {
		t.Errorf("expected 4 distinct canonical buckets, got %d", len(seen))
	}
}
