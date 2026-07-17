package gemini

import "testing"

// TestNormalizeGeminiFinishReason locks the mapping from the Gemini/Vertex
// finishReason vocabulary to the normalized ai.Response.FinishReason values.
// The load-bearing row is MAX_TOKENS→length: the quorum reviewer keys its
// loud truncation error off "length" (mission iter 42 regression guard).
func TestNormalizeGeminiFinishReason(t *testing.T) {
	cases := []struct{ in, want string }{
		{"STOP", "stop"},
		{"MAX_TOKENS", "length"},
		{"SAFETY", "content_filter"},
		{"RECITATION", "content_filter"},
		{"PROHIBITED_CONTENT", "content_filter"},
		{"", ""},
		{"OTHER", "other"}, // unknown → lowercased, never dropped
	}
	for _, c := range cases {
		if got := normalizeGeminiFinishReason(c.in); got != c.want {
			t.Errorf("normalizeGeminiFinishReason(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
