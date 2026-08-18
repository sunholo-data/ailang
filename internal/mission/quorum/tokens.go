package quorum

import "fmt"

// TokenAccountingGaps names every reviewer outcome that recorded a NON-ZERO
// cost while reporting ZERO tokens.
//
// Why this exists (#708): the mission-control Gate-3 chain ledger requires each
// metered stage to post tokens_in/tokens_out, and a quorum stage could only
// ever supply zeros — the provider's counts were read to derive cost and then
// discarded. With the counts recorded, the remaining failure mode is a provider
// or executor that bills without reporting usage. A zero there is
// INDISTINGUISHABLE from a free call, so it must be surfaced by name rather
// than defaulted away (Critical Principle 2 — no silent fallback on anything
// that feeds a spend record).
//
// This is a REPORT, not a refusal: an unreconcilable reviewer must never wedge
// a quorum the loop depends on. Callers print it loudly; tests assert on it.
//
// The enumeration deliberately covers BOTH tiers. A checker that walked only
// q.Reviewers would be blind by construction to every escalated agentic
// reviewer — precisely the tier whose cost is OBSERVED rather than derived, and
// therefore the one most likely to bill without usage.
func (q *QuorumResult) TokenAccountingGaps() []string {
	if q == nil {
		return nil
	}
	var gaps []string
	collect := func(tier string, outcomes []*ReviewerOutcome) {
		for _, o := range outcomes {
			if o == nil {
				continue
			}
			if o.CostUSD > 0 && o.TokensIn == 0 && o.TokensOut == 0 {
				gaps = append(gaps, fmt.Sprintf(
					"%s%s: $%.4f billed with ZERO reported tokens — spend cannot be reconciled",
					o.Model, tier, o.CostUSD))
			}
		}
	}
	collect("", q.Reviewers)
	if q.Tier2 != nil {
		collect(" (tier2)", q.Tier2.Reviewers)
	}
	return gaps
}
