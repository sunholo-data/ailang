package eval_analysis

import "strings"

// refusalPatterns are case-insensitive substrings that indicate the model
// refused the task instead of producing code. Compared against lowercased
// surfaces where model output or compiler errors end up (Code, Stderr,
// Stdout). Keep this list curated — false positives here silently drop
// real results from tag/saturation aggregates.
var refusalPatterns = []string{
	"apologies",
	"i cannot",
	"i'm sorry, but",
	"i am sorry, but",
	"as an ai",
	"i'm unable to",
	"i am unable to",
	"i can't help",
	"i can not",
}

// DetectRefusal returns true when any refusal pattern appears in code,
// stderr, or stdout. Matching is case-insensitive and substring-based,
// so "I CANNOT" and "i cannot" both trigger.
func DetectRefusal(code, stderr, stdout string) bool {
	haystack := strings.ToLower(code + "\n" + stderr + "\n" + stdout)
	for _, p := range refusalPatterns {
		if strings.Contains(haystack, p) {
			return true
		}
	}
	return false
}

// RefusalPatterns returns a copy of the refusal pattern list for tests and
// documentation. Exported so the M4 acceptance test can assert the
// ≥4 patterns invariant without reaching into package internals.
func RefusalPatterns() []string {
	out := make([]string, len(refusalPatterns))
	copy(out, refusalPatterns)
	return out
}
