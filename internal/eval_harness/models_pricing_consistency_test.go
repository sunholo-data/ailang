package eval_harness

import (
	"fmt"
	"sort"
	"testing"
	"time"
)

// billedProviders are the providers whose rows represent real money. Everything
// here is checked by the offline pricing gates below.
//
// `ollama` is deliberately absent: those rows run on our own hardware, and all 22
// of them are legitimately $0. Folding them in would make the zero-rate check
// fire 22 times on rows that are correct, which is how a gate gets muted.
var billedProviders = map[string]bool{
	"openrouter": true,
	"google":     true,
	"openai":     true,
	"anthropic":  true,
}

// TestModels_PricingIsSlugConsistent (M-EVAL-PRICING-DRIFT) pins the
// half of the pricing gate that needs NO network: two rows that name the same
// `api_name` must record the same rate.
//
// WIDENED 2026-08-13 from openrouter-only to every billed provider. The
// openrouter-only scope was not a considered choice, and it cost us: the
// widening immediately surfaced four rows that had been wrong for months, none
// of which any existing gate could see —
//
//	opencode-haiku, pi-claude-haiku-4-5, motoko-claude-haiku-4-5
//	    $0.25/$1.25 against the canonical claude-haiku-4-5 row's $1.00/$5.00.
//	    They were still carrying Claude Haiku 3.5's rates, so every agent-harness
//	    run on Haiku 4.5 had been banking a 4x UNDERSTATEMENT — and these are
//	    exactly the rows whose purpose is cross-harness cost comparison.
//	opencode-gemini-2.5-flash
//	    $0.50/$3.00 (gemini-3-flash's rates) against gemini-2-5-flash's
//	    $0.30/$2.50.
//
// The argument for widening is the same one that justified the original test,
// and it never depended on OpenRouter: Anthropic and Google bill the MODEL. They
// have no idea which harness sent the request either.
//
// This is a real defect class, not a tidiness rule. We keep several rows per
// upstream model so it can be driven by different harnesses — or-glm-5,
// opencode-or-glm-5 and motoko-glm-5 are all `z-ai/glm-5`. So when those rows
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
func TestModels_PricingIsSlugConsistent(t *testing.T) {
	c, err := LoadModelsConfig("../modelreg/models.yml")
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
		if !billedProviders[m.Provider] {
			continue
		}
		// Keyed by provider AND api_name: the same underlying model reached
		// through two different providers (anthropic direct vs the same model
		// via openrouter) legitimately carries two different prices, because
		// the reseller sets its own. Only rows billed by the SAME party must
		// agree.
		key := m.Provider + "|" + m.APIName
		bySlug[key] = append(bySlug[key], row{
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

		msg := fmt.Sprintf("%q has %d different recorded prices across %d entries "+
			"— the provider bills the model, not the harness, so these rows cannot all be right:",
			slug, len(distinct), len(rows))
		for _, k := range keys {
			msg += fmt.Sprintf("\n    %s  <- %v", k, distinct[k])
		}
		msg += "\n  Fix: run `make verify-model-pricing` and set every row for this slug to the live rate."
		t.Error(msg)
	}
}

// TestModels_PricingIsPlausible catches the two ways a rate goes wrong
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
func TestModels_PricingIsPlausible(t *testing.T) {
	c, err := LoadModelsConfig("../modelreg/models.yml")
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
		if !billedProviders[m.Provider] {
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

// TestModels_PricingScheduleIsHonoured enforces `pricing.expires` / `pricing.next`
// — the schema for rates that are known in advance to change on a date.
//
// This is the gate for a failure mode nothing else could see. On 2026-08-13
// Google launched Gemini 3.7 Flash at an INTRODUCTORY $0.75/$3.75 per 1M that
// reverts to $1.50/$7.50 on 2027-01-01, and extended the same promo to 3.6
// Flash. Before this test, that reversion was recorded only in a YAML comment.
// Comments do not fail. On New Year's Day every Gemini Flash cost figure would
// have quietly become half of true spend — the same silent-understatement class
// as the drift audit, except pre-announced and therefore entirely preventable.
//
// It lives in CI rather than in `make verify-model-pricing` on purpose: an
// expiry is decidable from the file and the calendar alone. Putting it behind
// the network tool — which is deliberately NOT in `make ci` because a third
// party's downtime must not turn us red — would mean the one failure we can
// predict to the day is only caught if somebody remembers to run the tool.
func TestModels_PricingScheduleIsHonoured(t *testing.T) {
	c, err := LoadModelsConfig("../modelreg/models.yml")
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	const layout = "2006-01-02"
	today := time.Now().UTC()

	names := make([]string, 0, len(c.Models))
	for n := range c.Models {
		names = append(names, n)
	}
	sort.Strings(names)

	for _, name := range names {
		m := c.Models[name]
		p := m.Pricing

		// Half a schedule is a defect in both directions. `next` without
		// `expires` never fires; `expires` without `next` fails on the day
		// with no answer for what to replace the rate WITH, which in practice
		// means someone deletes the field to get green.
		if p.Expires == "" {
			if p.Next != nil {
				t.Errorf("%s: pricing.next is set with no pricing.expires — "+
					"a successor rate with no date can never take effect", name)
			}
			continue
		}
		if p.Next == nil {
			t.Errorf("%s: pricing.expires=%q with no pricing.next — record the rate "+
				"that takes over, or the row fails on the day with nothing to fix it to",
				name, p.Expires)
			continue
		}

		expiry, err := time.Parse(layout, p.Expires)
		if err != nil {
			t.Errorf("%s: pricing.expires=%q is not a YYYY-MM-DD date: %v", name, p.Expires, err)
			continue
		}
		if p.Next.InputPer1K <= 0 || p.Next.OutputPer1K <= 0 {
			t.Errorf("%s: pricing.next has a non-positive rate (in=%g out=%g) — a scheduled "+
				"price of zero would report real spend as free",
				name, p.Next.InputPer1K, p.Next.OutputPer1K)
			continue
		}
		if p.Next.InputPer1K == p.InputPer1K && p.Next.OutputPer1K == p.OutputPer1K {
			t.Errorf("%s: pricing.next is identical to the current rate — either the schedule "+
				"is stale (drop expires/next) or the successor rate was mis-transcribed", name)
			continue
		}

		// Expires is INCLUSIVE: the rate is billed through the end of that day,
		// so the row only goes stale once the calendar has passed it entirely.
		if today.After(expiry.AddDate(0, 0, 1)) {
			t.Errorf("%s: pricing lapsed on %s and the row still charges %g/%g per 1K. "+
				"The scheduled successor is %g/%g — promote it into input_per_1k/output_per_1k "+
				"and clear expires/next (confirm against the vendor's page first; a promo can be "+
				"extended). Every cost banked for this row since %s understates true spend.",
				name, p.Expires, p.InputPer1K, p.OutputPer1K,
				p.Next.InputPer1K, p.Next.OutputPer1K, p.Expires)
		}
	}
}
