package quorum

// Who reviews a design doc.
//
// Mark's rules (attended 2026-09-25):
//   - The author's vendor sits out: "if anthropic does the design, it doesn't review;
//     if astra does the design, it's not on the quorum."
//   - Reviewers come from a POOL, three per doc: "we keep three reviewers but we
//     select from the pool so we can expand it further."
//   - The author's vendor is benched, not dropped: "if all other reviewers are
//     blocked it's ok to relax that a bit."
//
// Selection is deterministic per doc (keyed on the doc path), so a revised doc's
// second round is judged by the same seats that blocked the first, while different
// docs rotate through the pool. Seats prefer distinct vendors. An absent seat is
// replaced from the rest of the pool before anyone from the author's vendor is
// recalled.

import (
	"hash/fnv"
	"strings"
	"sync"
)

// DefaultSeats is how many independent reviewers a quorum aims for.
const DefaultSeats = 3

// TierAuthorVendor labels a benched seat that was called back because every
// independent seat was absent.
const TierAuthorVendor = "author-vendor-fallback"

// vendorKeys maps a substring of a model or lane id to its vendor. Harness
// prefixes that are not vendors ("pi:", "opencode:", "motoko:") match no key, so a
// lane id resolves by its model part without parsing.
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
// "codex:gpt-6-astra", "claude:claude-opus-5-5", "pi:ollama/kimi-k3:cloud").
// "" when unrecognised — the caller must not guess.
func VendorOf(id string) string {
	s := strings.ToLower(id)
	for _, k := range vendorKeys {
		if strings.Contains(s, k.key) {
			return k.vendor
		}
	}
	return ""
}

// Selection is the per-doc seating plan.
type Selection struct {
	Primary []string // seated first
	Reserve []string // replace absent seats, in order
	Benched []string // the author's vendor; recalled only if nothing else answers
}

// SelectReviewers seats `seats` reviewers from the pool for one doc. The eligible
// pool (everyone not of the author's vendor) is rotated by a hash of docKey, then
// seats are taken preferring vendors not yet seated. With an unrecognised author
// nobody is benched.
func SelectReviewers(pool []string, author, docKey string, seats int) Selection {
	var sel Selection
	av := VendorOf(author)
	var eligible []string
	for _, m := range pool {
		if av != "" && VendorOf(m) == av {
			sel.Benched = append(sel.Benched, m)
		} else {
			eligible = append(eligible, m)
		}
	}
	if len(eligible) == 0 {
		return sel
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(docKey))
	off := int(h.Sum32() % uint32(len(eligible)))
	rotated := append(append([]string{}, eligible[off:]...), eligible[:off]...)

	seen := map[string]bool{}
	var rest []string
	for _, m := range rotated {
		v := VendorOf(m)
		if len(sel.Primary) < seats && (v == "" || !seen[v]) {
			sel.Primary = append(sel.Primary, m)
			seen[v] = true
		} else {
			rest = append(rest, m)
		}
	}
	for len(sel.Primary) < seats && len(rest) > 0 { // fewer distinct vendors than seats
		sel.Primary, rest = append(sel.Primary, rest[0]), rest[1:]
	}
	sel.Reserve = rest
	return sel
}

// RunSeatedQuorum runs the primary seats, replaces absent ones from the reserve
// until `seats` verdicts are in or the reserve is spent, and recalls the benched
// author's-vendor seats only if nobody at all answered. Every absent seat stays in
// the artifact with its reason.
func RunSeatedQuorum(docPath, docBody, isoTS string, sel Selection, seats int, maxCostUSD float64, controller *ControllerReview, runner func(model, docPath, docBody string, maxCostUSD float64) *ReviewerOutcome) *QuorumResult {
	q := RunQuorum(docPath, docBody, isoTS, sel.Primary, maxCostUSD, controller, runner)
	reserve := sel.Reserve
	for need := seats - presentCount(q); need > 0 && len(reserve) > 0; need = seats - presentCount(q) {
		if need > len(reserve) {
			need = len(reserve)
		}
		q.Reviewers = append(q.Reviewers, runParallel(reserve[:need], docPath, docBody, maxCostUSD, runner)...)
		reserve = reserve[need:]
	}
	if presentCount(q) == 0 && len(sel.Benched) > 0 {
		for _, o := range runParallel(sel.Benched, docPath, docBody, maxCostUSD, runner) {
			o.Tier = TierAuthorVendor
			q.Reviewers = append(q.Reviewers, o)
		}
	}
	q.Synthesis = synthesize(q.Reviewers, controller)
	return q
}

func presentCount(q *QuorumResult) int {
	n := 0
	for _, o := range q.Reviewers {
		if o != nil && o.Present {
			n++
		}
	}
	return n
}

func runParallel(models []string, docPath, docBody string, maxCostUSD float64, runner func(model, docPath, docBody string, maxCostUSD float64) *ReviewerOutcome) []*ReviewerOutcome {
	out := make([]*ReviewerOutcome, len(models))
	var wg sync.WaitGroup
	for i, m := range models {
		wg.Add(1)
		go func(i int, m string) {
			defer wg.Done()
			out[i] = runner(m, docPath, docBody, maxCostUSD)
		}(i, m)
	}
	wg.Wait()
	return out
}
