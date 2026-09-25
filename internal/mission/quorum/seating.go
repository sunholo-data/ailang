package quorum

// Who reviews a design doc: every vendor on the roster EXCEPT the author's.
//
// Mark's rule (attended 2026-09-25, restating "ideally no model provider marks its
// own work"): if Anthropic designs, Anthropic does not review; if astra designs,
// OpenAI is not on the quorum. The author's vendor is benched, not dropped — when
// every seated reviewer is absent, the benched seats are called back and labelled,
// because a same-vendor verdict beats the zero-signal block ("if all other
// reviewers are blocked it's ok to relax that a bit").
//
// This replaces the hand-applied astra workaround (substitute gpt5-6-sol on astra's
// turn), which kept an OpenAI model reviewing an OpenAI doc.

import "strings"

// TierAuthorVendor labels a benched seat that was called back because every
// independent seat was absent.
const TierAuthorVendor = "author-vendor-fallback"

// vendorKeys maps a substring of a model or lane id to its vendor. Order matters
// only where one key is inside another; none are.
var vendorKeys = []struct{ key, vendor string }{
	{"claude", "anthropic"}, {"opus", "anthropic"}, {"sonnet", "anthropic"},
	{"fable", "anthropic"}, {"haiku", "anthropic"},
	{"gpt", "openai"}, {"astra", "openai"}, {"codex", "openai"},
	{"gemini", "google"},
	{"glm", "zai"}, {"z-ai", "zai"}, {"zai", "zai"},
	{"kimi", "moonshot"}, {"moonshot", "moonshot"},
	{"deepseek", "deepseek"},
	{"minimax", "minimax"},
	{"qwen", "alibaba"},
}

// VendorOf names the vendor behind a model or lane id ("gpt6-astra",
// "codex:gpt-6-astra", "claude:claude-opus-5-5", "pi:ollama/deepseek-v4-flash:0731-cloud",
// "oc-glm-5-3"). "" when unrecognised — the caller must not guess.
func VendorOf(id string) string {
	// Harness prefixes that are not vendors ("pi:", "opencode:", "motoko:") match
	// no key, so a lane id resolves by its model part without parsing.
	s := strings.ToLower(id)
	for _, k := range vendorKeys {
		if strings.Contains(s, k.key) {
			return k.vendor
		}
	}
	return ""
}

// SeatReviewers splits the roster into seated reviewers and the author's-vendor
// seats that sit out. With an unrecognised author vendor nobody is benched.
func SeatReviewers(roster []string, author string) (seated, benched []string) {
	av := VendorOf(author)
	for _, m := range roster {
		if av != "" && VendorOf(m) == av {
			benched = append(benched, m)
		} else {
			seated = append(seated, m)
		}
	}
	return seated, benched
}

// anyPresent reports whether at least one reviewer produced a verdict.
func anyPresent(q *QuorumResult) bool {
	for _, o := range q.Reviewers {
		if o != nil && o.Present {
			return true
		}
	}
	return false
}

// RecallBenched runs the benched seats when no seated reviewer produced a verdict,
// labels them, and re-synthesizes. It reports whether it ran them.
func RecallBenched(q *QuorumResult, benched []string, docPath, docBody string, maxCostUSD float64, runner func(model, docPath, docBody string, maxCostUSD float64) *ReviewerOutcome) bool {
	if q == nil || len(benched) == 0 || anyPresent(q) {
		return false
	}
	for _, m := range benched {
		o := runner(m, docPath, docBody, maxCostUSD)
		o.Tier = TierAuthorVendor
		q.Reviewers = append(q.Reviewers, o)
	}
	q.Synthesis = synthesize(q.Reviewers, q.ControllerInSession)
	return true
}
