package eval_harness

import (
	"fmt"
	"sort"
	"testing"
)

// TestModels_OpenRouterPricingIsSlugConsistent (M-EVAL-PRICING-DRIFT) pins the
// half of the pricing gate that needs NO network: two rows that name the same
// `api_name` must record the same rate.
//
// This is a real defect class, not a tidiness rule. We keep several rows per
// upstream slug so a model can be driven by different harnesses — or-glm-5,
// opencode-or-glm-5 and motoko-glm-5 are all `z-ai/glm-5`. OpenRouter bills the
// slug; it has no idea which harness sent the request. So when those rows
// disagree, the SAME tokens cost different amounts depending on who invoked
// them, and the harness comparison the rows exist to enable — "does motoko's
// scaffolding earn its cost over opencode's?" — is reading a bookkeeping
// artifact as a result.
//
// The 2026-08-13 audit found four such splits, the worst being
// `google/gemma-4-26b-a4b-it` carrying THREE different prices across four rows
// (motoko-gemma-4 at $0.10/$0.30, motoko-or-gemma-4-26b at $0.06/$0.24, and two
// more at $0.06/$0.33). None of the three was the live price.
//
// Why this test and `make verify-model-pricing` are separate: drift against the
// vendor can only be caught with the network and a third party's uptime, so it
// must not gate CI. Internal disagreement is decidable from the file alone and
// is therefore a hard gate — and it is the failure that survives being offline,
// because two rows cannot BOTH be right no matter what the vendor charges.
func TestModels_OpenRouterPricingIsSlugConsistent(t *testing.T) {
	c, err := LoadModelsConfig("models.yml")
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	type row struct {
		name    string
		in, out float64
		cacheRd float64
	}
	bySlug := map[string][]row{}
	for name, m := range c.Models {
		if m.Provider != "openrouter" {
			continue
		}
		bySlug[m.APIName] = append(bySlug[m.APIName], row{
			name:    name,
			in:      m.Pricing.InputPer1K,
			out:     m.Pricing.OutputPer1K,
			cacheRd: m.Pricing.CacheReadPer1K,
		})
	}

	slugs := make([]string, 0, len(bySlug))
	for s := range bySlug {
		slugs = append(slugs, s)
	}
	sort.Strings(slugs)

	for _, slug := range slugs {
		rows := bySlug[slug]
		if len(rows) < 2 {
			continue
		}
		sort.Slice(rows, func(i, j int) bool { return rows[i].name < rows[j].name })

		distinct := map[string][]string{}
		for _, r := range rows {
			// Exact float equality is correct here: these are literals parsed
			// from the same file, not computed values. Two rows for one slug
			// should be byte-identical numbers, and "close enough" is exactly
			// the tolerance that let $0.00057 and $0.00074 coexist for kimi-k2.6.
			key := fmt.Sprintf("in=%g out=%g cache=%g", r.in, r.out, r.cacheRd)
			distinct[key] = append(distinct[key], r.name)
		}
		if len(distinct) == 1 {
			continue
		}

		keys := make([]string, 0, len(distinct))
		for k := range distinct {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		msg := fmt.Sprintf("api_name %q has %d different recorded prices across %d entries "+
			"— OpenRouter bills the slug, so these rows cannot all be right:",
			slug, len(distinct), len(rows))
		for _, k := range keys {
			msg += fmt.Sprintf("\n    %s  <- %v", k, distinct[k])
		}
		msg += "\n  Fix: run `make verify-model-pricing` and set every row for this slug to the live rate."
		t.Error(msg)
	}
}

// TestModels_OpenRouterPricingIsPlausible catches the two ways a rate goes wrong
// that need neither the network nor a second row to compare against: a negative
// price, and a non-free row priced at zero.
//
// Zero is the one that matters. `CalculateCostForModel` multiplies tokens by the
// rate, so a zero-rate row reports $0.00 for real spend — it does not error, it
// does not warn, it just makes an expensive model look free in every budget,
// rollup and cost-per-pass table downstream. That is precisely the "silent
// fallback that corrupts business logic" CLAUDE.md principle #2 forbids.
//
// Two rows are legitimately $0 and are exempt by name: `or-auto` (OpenRouter's
// router — the resolved model's price is unknowable ahead of the call, and the
// live API reports a -1/token sentinel for it) and `or-laguna-xs-2` (a `:free`
// tier, where $0 is correct by construction).
func TestModels_OpenRouterPricingIsPlausible(t *testing.T) {
	c, err := LoadModelsConfig("models.yml")
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	// Exempt rows must justify themselves here, so adding one is a visible edit.
	zeroIsCorrect := map[string]string{
		"or-auto":        "router: resolved model unknown until the call; live API reports a -1/token sentinel",
		"or-laguna-xs-2": ":free tier — $0 is correct by construction",
	}

	var bad []string
	for name, m := range c.Models {
		if m.Provider != "openrouter" {
			continue
		}
		in, out := m.Pricing.InputPer1K, m.Pricing.OutputPer1K
		if in < 0 || out < 0 || m.Pricing.CacheReadPer1K < 0 {
			bad = append(bad, fmt.Sprintf("%s: negative rate (in=%g out=%g cache=%g)",
				name, in, out, m.Pricing.CacheReadPer1K))
			continue
		}
		if in == 0 && out == 0 {
			if _, ok := zeroIsCorrect[name]; !ok {
				bad = append(bad, fmt.Sprintf(
					"%s (%s): priced at $0 with no exemption — real spend would report as free",
					name, m.APIName))
			}
		}
	}
	sort.Strings(bad)
	for _, b := range bad {
		t.Error(b)
	}
}
