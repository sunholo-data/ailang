package ai

import "testing"

// M-ANTHROPIC-CACHE-HIT-RATE M1.
//
// CachedPrefix is a splitting hint, not a content change. Providers that do not
// implement cache splitting must still send the whole prompt — FullUserPrompt
// is the single place that contract is expressed, so it is the single place it
// can regress.

func TestFullUserPrompt_ConcatenatesPrefix(t *testing.T) {
	cases := []struct {
		name         string
		cachedPrefix string
		userPrompt   string
		want         string
	}{
		{"no_prefix_is_identity", "", "Hello", "Hello"},
		{"prefix_concatenates", "TEACHING", "\n\n## Task\n\nDo it", "TEACHING\n\n## Task\n\nDo it"},
		{"prefix_only", "TEACHING", "", "TEACHING"},
		{"both_empty", "", "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := &Request{CachedPrefix: tc.cachedPrefix, UserPrompt: tc.userPrompt}
			if got := r.FullUserPrompt(); got != tc.want {
				t.Errorf("FullUserPrompt() = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestFullUserPrompt_SplitIsEquivalentToPreConcatenated is the byte-identity
// guarantee for non-Anthropic providers: a caller that splits its prompt into
// CachedPrefix+UserPrompt must produce exactly what a caller that pre-
// concatenated would, so migrating a call site to the split form cannot change
// what any provider sends.
func TestFullUserPrompt_SplitIsEquivalentToPreConcatenated(t *testing.T) {
	const teaching = "TEACHING PROMPT BODY"
	const task = "\n\n## Task\n\nSolve it"

	split := &Request{CachedPrefix: teaching, UserPrompt: task}
	preConcatenated := &Request{UserPrompt: teaching + task}

	if split.FullUserPrompt() != preConcatenated.FullUserPrompt() {
		t.Errorf("split form = %q, pre-concatenated = %q — must be identical",
			split.FullUserPrompt(), preConcatenated.FullUserPrompt())
	}
}
