// Command verify-model-pricing drift-checks the priced rows in
// internal/eval_harness/models.yml against the live OpenRouter models API.
//
// # Coverage, and why the vendor-direct half is advisory
//
// `provider: "openrouter"` rows are checked against the API that bills them, so a
// disagreement there is authoritative and fails the run.
//
// `google` / `openai` / `anthropic` rows are billed by vendors that publish NO
// machine-readable price list. Three designs were considered for them:
//
//  1. A checked-in expected-price table plus a staleness date. Rejected: the
//     table would restate the same number models.yml already holds, so it can
//     only catch someone editing one copy — not the vendor changing the price,
//     which is the actual failure. It is a second thing to be wrong.
//  2. A staleness date alone (`verified_on`, warn after N days). Rejected on
//     measurement: the miss that prompted this work — Google halving Gemini
//     Flash on 2026-08-13 — happened 23 days after that row was last verified.
//     Any window loose enough to avoid nagging (90 days) sails straight past it,
//     and back-filling the field across 51 rows would mean inventing dates.
//  3. Cross-check against OpenRouter's catalogue, which carries the same
//     first-party models. CHOSEN. It is machine-readable, already fetched by
//     this tool, needs no duplicated number, and — verified against the live
//     catalogue on 2026-08-13 — it had ALREADY picked up the Gemini Flash cut
//     ($0.75/$3.75) on day one, so it would have caught the miss immediately.
//
// The catch, also measured on 2026-08-13: OpenRouter runs promotions of its own.
// It listed google/gemini-3.7-flash at exactly HALF Google's published rate, and
// openai/gpt-5.6-terra, openai/gpt-5.6-luna and anthropic/claude-sonnet-5 at 0.50x,
// 0.50x and 0.67x of their vendor list prices. So a mismatch here means "an
// independent catalogue disagrees — go read the vendor's page", NOT "models.yml
// is wrong". Failing the build on it would make the tool red for reasons you are
// supposed to ignore, which is how a checker gets ignored altogether.
//
// Vendor mismatches are therefore reported and exit 0; --strict promotes them to
// exit 1 for a deliberate audit. What IS a hard failure for vendor rows is
// decidable offline and lives in CI instead: two rows for one model disagreeing
// (TestModels_PricingIsSlugConsistent), a $0 rate (TestModels_PricingIsPlausible),
// and a lapsed introductory price (TestModels_PricingScheduleIsHonoured). The
// widening on 2026-08-13 found five real bugs that way, including three rows still
// charging Claude Haiku 3.5's rates for Haiku 4.5 — a 4x understatement.
//
// Why this exists as a tool rather than a one-time correction: OpenRouter
// reprices with no notice and no changelog, so a hand-verified rate is evidence
// of when someone looked, not evidence that the number is still right. The
// 2026-08-13 audit found 23 of 39 openrouter rows stale — including one that had
// been verified against this very endpoint seven days earlier (or-glm-5-2, whose
// output rate had already moved 1.79x). A model's recorded price is an input to
// every cost-per-pass comparison we make, and drift is DIRECTIONAL: an
// understated rate silently flatters a model, an overstated one penalises it.
//
// Deliberately NOT part of `make ci`: it needs the network and a third party's
// uptime, and a CI job that goes red because someone else's API is down teaches
// people to ignore it. The offline invariant — two rows sharing an api_name must
// share a price — IS a CI gate, in models_pricing_consistency_test.go.
//
//	go run ./tools/verify-model-pricing [--yml PATH] [--json] [--strict]
//
// Exit codes:
//
//	0  every resolvable openrouter row matches the live rate, and no priced
//	   row has lapsed past its `pricing.expires` date
//	1  at least one openrouter row drifted, or a scheduled price has lapsed
//	   (or, with --strict, a slug no longer resolves / a vendor row disagrees
//	   with the OpenRouter mirror)
//	2  the check could not be performed (network, API shape, unreadable yml)
//
// Exit 2 is distinct on purpose: "prices are correct" and "I could not find out"
// are different answers, and collapsing them into one code is how a silently
// broken checker reads as a green one.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strconv"
	"time"

	"github.com/sunholo-data/ailang/internal/eval_harness"
)

const openRouterModelsURL = "https://openrouter.ai/api/v1/models"

// relTolerance is the fractional difference below which two rates are "the same".
// The yml carries exact decimal literals copied from this API, so any real drift
// is orders of magnitude larger than this; the tolerance exists only to absorb
// float64 round-tripping of values like 5.9999999999999995e-05.
const relTolerance = 1e-9

type apiResponse struct {
	Data []struct {
		ID      string `json:"id"`
		Pricing struct {
			Prompt         string `json:"prompt"`
			Completion     string `json:"completion"`
			InputCacheRead string `json:"input_cache_read"`
		} `json:"pricing"`
	} `json:"data"`
}

// livePrice holds per-1K rates. A nil field means the API did not publish that
// rate, which is not the same as publishing zero.
type livePrice struct {
	in, out, cacheRead *float64
}

// vendorMirror maps a vendor-direct row onto the slug OpenRouter uses for the
// same model. Empty means "no mapping" — reported, never guessed at.
//
// google/openai slugs are the api_name verbatim under the vendor prefix
// (google/gemini-3.6-flash, openai/gpt-5.6-sol), including any -preview suffix.
// Anthropic is the odd one: models.yml carries the API's dashed version and
// dated snapshot (claude-haiku-4-5-20251001) while OpenRouter uses the marketing
// form (anthropic/claude-haiku-4.5), so the date is dropped and the trailing
// major-minor pair rejoined with a dot. Measured coverage on 2026-08-13:
// 47 of 51 vendor rows map; the 4 that do not are reported as unmapped.
var (
	anthropicSnapshotSuffix = regexp.MustCompile(`-\d{8}$`)
	anthropicVersionDashes  = regexp.MustCompile(`-(\d+)-(\d+)$`)
)

func vendorMirror(provider, apiName string) string {
	switch provider {
	case "google", "openai":
		return provider + "/" + apiName
	case "anthropic":
		n := anthropicSnapshotSuffix.ReplaceAllString(apiName, "")
		n = anthropicVersionDashes.ReplaceAllString(n, "-$1.$2")
		return "anthropic/" + n
	default:
		return ""
	}
}

// expiringSoonWindow is how far ahead a scheduled price change is announced in
// the report. Long enough that a quarterly glance catches it before it bites,
// short enough that it is not permanently on screen.
const expiringSoonWindow = 45 * 24 * time.Hour

type finding struct {
	Entry    string  `json:"entry"`
	Slug     string  `json:"slug"`
	Kind     string  `json:"kind"` // "drift" | "unresolved" | "cache_undeclared" | "vendor_mismatch" | "vendor_unmapped" | "expired" | "expiring_soon" | "schedule_invalid"
	YMLIn    float64 `json:"yml_input_per_1k"`
	YMLOut   float64 `json:"yml_output_per_1k"`
	LiveIn   float64 `json:"live_input_per_1k,omitempty"`
	LiveOut  float64 `json:"live_output_per_1k,omitempty"`
	InRatio  float64 `json:"input_ratio,omitempty"`
	OutRatio float64 `json:"output_ratio,omitempty"`
	Note     string  `json:"note,omitempty"`
}

func main() {
	ymlPath := flag.String("yml", "internal/eval_harness/models.yml", "path to models.yml")
	asJSON := flag.Bool("json", false, "emit findings as JSON")
	strict := flag.Bool("strict", false, "also fail when a slug no longer resolves on the API")
	flag.Parse()

	cfg, err := eval_harness.LoadModelsConfig(*ymlPath)
	if err != nil {
		fatal("read %s: %v", *ymlPath, err)
	}

	live, err := fetchLivePricing()
	if err != nil {
		fatal("%v", err)
	}

	drift, unresolved, cacheGaps := compare(cfg, live)
	vendorMismatch, vendorUnmapped := compareVendor(cfg, live)
	expired, expiringSoon, scheduleInvalid := checkSchedules(cfg, time.Now().UTC())

	if *asJSON {
		var all []finding
		for _, set := range [][]finding{
			drift, expired, scheduleInvalid,
			vendorMismatch, unresolved, vendorUnmapped, expiringSoon, cacheGaps,
		} {
			all = append(all, set...)
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(map[string]any{
			"checked_at": time.Now().UTC().Format(time.RFC3339),
			"source":     openRouterModelsURL,
			"findings":   all,
		}); err != nil {
			fatal("encode json: %v", err)
		}
	} else {
		report(drift, unresolved, cacheGaps)
		reportVendor(vendorMismatch, vendorUnmapped)
		reportSchedule(expired, expiringSoon, scheduleInvalid)
	}

	// Hard: an openrouter row disagrees with the API that bills it, or a
	// scheduled price has lapsed / is malformed. Both are decidable without
	// anyone's judgement.
	if len(drift) > 0 || len(expired) > 0 || len(scheduleInvalid) > 0 {
		os.Exit(1)
	}
	// Advisory, promoted only on request: a retired slug, or a vendor row the
	// OpenRouter mirror disagrees with (which may just be an OpenRouter promo).
	if *strict && (len(unresolved) > 0 || len(vendorMismatch) > 0) {
		os.Exit(1)
	}
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "verify-model-pricing: "+format+"\n", args...)
	// Exit 2: the check did not run. Never 1 — that would read as "prices differ".
	os.Exit(2)
}

func fetchLivePricing() (map[string]livePrice, error) {
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Get(openRouterModelsURL)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", openRouterModelsURL, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch %s: HTTP %d", openRouterModelsURL, resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var parsed apiResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	if len(parsed.Data) == 0 {
		// An empty list would silently pass every row as "unresolved" and, without
		// --strict, exit 0 — a green result from an API that told us nothing.
		return nil, fmt.Errorf("API returned 0 models; refusing to report on an empty catalogue")
	}

	// The API quotes per-TOKEN prices as strings; models.yml stores per-1K.
	perK := func(s string) *float64 {
		if s == "" {
			return nil
		}
		v, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return nil
		}
		v *= 1000
		return &v
	}

	out := make(map[string]livePrice, len(parsed.Data))
	for _, m := range parsed.Data {
		out[m.ID] = livePrice{
			in:        perK(m.Pricing.Prompt),
			out:       perK(m.Pricing.Completion),
			cacheRead: perK(m.Pricing.InputCacheRead),
		}
	}
	return out, nil
}

func compare(cfg *eval_harness.ModelsConfig, live map[string]livePrice) (drift, unresolved, cacheGaps []finding) {
	names := make([]string, 0, len(cfg.Models))
	for n := range cfg.Models {
		names = append(names, n)
	}
	sort.Strings(names)

	for _, name := range names {
		m := cfg.Models[name]
		if m.Provider != "openrouter" {
			continue
		}
		lp, ok := live[m.APIName]
		if !ok {
			unresolved = append(unresolved, finding{
				Entry: name, Slug: m.APIName, Kind: "unresolved",
				YMLIn: m.Pricing.InputPer1K, YMLOut: m.Pricing.OutputPer1K,
				Note: "slug absent from the live catalogue — a run against this row would 404",
			})
			continue
		}
		if lp.in == nil || lp.out == nil {
			unresolved = append(unresolved, finding{
				Entry: name, Slug: m.APIName, Kind: "unresolved",
				YMLIn: m.Pricing.InputPer1K, YMLOut: m.Pricing.OutputPer1K,
				Note: "API published no prompt/completion rate for this slug",
			})
			continue
		}
		// OpenRouter reports -1/token for openrouter/auto: the resolved model is
		// not known until the call, so there is no rate to drift against.
		if *lp.in < 0 || *lp.out < 0 {
			continue
		}

		inOK := closeEnough(m.Pricing.InputPer1K, *lp.in)
		outOK := closeEnough(m.Pricing.OutputPer1K, *lp.out)
		if !inOK || !outOK {
			drift = append(drift, finding{
				Entry: name, Slug: m.APIName, Kind: "drift",
				YMLIn: m.Pricing.InputPer1K, YMLOut: m.Pricing.OutputPer1K,
				LiveIn: *lp.in, LiveOut: *lp.out,
				InRatio:  ratio(*lp.in, m.Pricing.InputPer1K),
				OutRatio: ratio(*lp.out, m.Pricing.OutputPer1K),
			})
			continue
		}

		// Informational only. An undeclared cache rate bills cache reads at the
		// FULL input rate — an overstatement, which is the safe direction and is
		// visible in a budget. Worth surfacing, never worth failing on.
		if m.Pricing.CacheReadPer1K == 0 && lp.cacheRead != nil && *lp.cacheRead > 0 {
			cacheGaps = append(cacheGaps, finding{
				Entry: name, Slug: m.APIName, Kind: "cache_undeclared",
				Note: fmt.Sprintf("API publishes input_cache_read %.10g per 1K; row declares none, "+
					"so cache reads bill at the full input rate", *lp.cacheRead),
			})
		}
	}
	return drift, unresolved, cacheGaps
}

// compareVendor cross-checks vendor-direct rows against OpenRouter's listing of
// the same model. Advisory by construction — see the package comment for why a
// disagreement here is a prompt to go read the vendor's page, not a verdict.
func compareVendor(cfg *eval_harness.ModelsConfig, live map[string]livePrice) (mismatch, unmapped []finding) {
	for _, name := range sortedModelNames(cfg) {
		m := cfg.Models[name]
		slug := vendorMirror(m.Provider, m.APIName)
		if slug == "" {
			continue // not a vendor-direct provider
		}
		lp, ok := live[slug]
		if !ok || lp.in == nil || lp.out == nil {
			unmapped = append(unmapped, finding{
				Entry: name, Slug: slug, Kind: "vendor_unmapped",
				YMLIn: m.Pricing.InputPer1K, YMLOut: m.Pricing.OutputPer1K,
				Note: "no OpenRouter listing under this slug — this row has no cross-check, " +
					"so its rate rests entirely on whoever last read the vendor's page",
			})
			continue
		}
		if closeEnough(m.Pricing.InputPer1K, *lp.in) && closeEnough(m.Pricing.OutputPer1K, *lp.out) {
			continue
		}
		mismatch = append(mismatch, finding{
			Entry: name, Slug: slug, Kind: "vendor_mismatch",
			YMLIn: m.Pricing.InputPer1K, YMLOut: m.Pricing.OutputPer1K,
			LiveIn: *lp.in, LiveOut: *lp.out,
			InRatio:  ratio(*lp.in, m.Pricing.InputPer1K),
			OutRatio: ratio(*lp.out, m.Pricing.OutputPer1K),
		})
	}
	return mismatch, unmapped
}

// checkSchedules enforces pricing.expires / pricing.next. A lapsed rate is a
// HARD failure: unlike a vendor mismatch it needs no third-party opinion, only
// the calendar. The same rule is a CI test (TestModels_PricingScheduleIsHonoured)
// so it fires even when nobody runs this tool.
func checkSchedules(cfg *eval_harness.ModelsConfig, now time.Time) (expired, soon, invalid []finding) {
	const layout = "2006-01-02"
	for _, name := range sortedModelNames(cfg) {
		m := cfg.Models[name]
		p := m.Pricing

		if p.Expires == "" {
			if p.Next != nil {
				invalid = append(invalid, finding{
					Entry: name, Slug: m.APIName, Kind: "schedule_invalid",
					Note: "pricing.next set with no pricing.expires — it can never take effect",
				})
			}
			continue
		}
		if p.Next == nil {
			invalid = append(invalid, finding{
				Entry: name, Slug: m.APIName, Kind: "schedule_invalid",
				Note: fmt.Sprintf("pricing.expires=%s with no pricing.next — nothing to promote on the day", p.Expires),
			})
			continue
		}
		exp, err := time.Parse(layout, p.Expires)
		if err != nil {
			invalid = append(invalid, finding{
				Entry: name, Slug: m.APIName, Kind: "schedule_invalid",
				Note: fmt.Sprintf("pricing.expires=%q is not YYYY-MM-DD", p.Expires),
			})
			continue
		}

		// Expires is inclusive: billed through the end of that day.
		lapse := exp.AddDate(0, 0, 1)
		switch {
		case now.After(lapse):
			expired = append(expired, finding{
				Entry: name, Slug: m.APIName, Kind: "expired",
				YMLIn: p.InputPer1K, YMLOut: p.OutputPer1K,
				LiveIn: p.Next.InputPer1K, LiveOut: p.Next.OutputPer1K,
				InRatio:  ratio(p.Next.InputPer1K, p.InputPer1K),
				OutRatio: ratio(p.Next.OutputPer1K, p.OutputPer1K),
				Note:     fmt.Sprintf("introductory rate lapsed %s; row still charges the old rate", p.Expires),
			})
		case now.Add(expiringSoonWindow).After(lapse):
			soon = append(soon, finding{
				Entry: name, Slug: m.APIName, Kind: "expiring_soon",
				YMLIn: p.InputPer1K, YMLOut: p.OutputPer1K,
				LiveIn: p.Next.InputPer1K, LiveOut: p.Next.OutputPer1K,
				Note: fmt.Sprintf("rate changes after %s (in %d days)", p.Expires,
					int(time.Until(lapse).Hours()/24)),
			})
		}
	}
	return expired, soon, invalid
}

func sortedModelNames(cfg *eval_harness.ModelsConfig) []string {
	names := make([]string, 0, len(cfg.Models))
	for n := range cfg.Models {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

func closeEnough(a, b float64) bool {
	if a == b {
		return true
	}
	scale := a
	if b > scale {
		scale = b
	}
	if scale == 0 {
		return false
	}
	d := a - b
	if d < 0 {
		d = -d
	}
	return d/scale < relTolerance
}

func ratio(live, yml float64) float64 {
	if yml == 0 {
		return 0
	}
	return live / yml
}

func report(drift, unresolved, cacheGaps []finding) {
	if len(drift) > 0 {
		fmt.Printf("PRICING DRIFT — %d row(s) disagree with the live OpenRouter catalogue\n\n", len(drift))
		fmt.Printf("  %-30s %-42s %-24s %-24s %s\n", "ENTRY", "SLUG", "RECORDED in/out per 1K", "LIVE in/out per 1K", "FACTOR")
		for _, f := range drift {
			fmt.Printf("  %-30s %-42s %-24s %-24s in x%.3f out x%.3f\n",
				f.Entry, f.Slug,
				fmt.Sprintf("%-.10g / %-.10g", f.YMLIn, f.YMLOut),
				fmt.Sprintf("%-.10g / %-.10g", f.LiveIn, f.LiveOut),
				f.InRatio, f.OutRatio)
		}
		fmt.Println()
		fmt.Println("  A factor > 1 means the recorded rate UNDERSTATES cost: every banked figure for")
		fmt.Println("  that row is too low, and the model looks cheaper per pass than it was.")
		fmt.Println("  Correct models.yml going forward. Do NOT recompute banked results — annotate")
		fmt.Println("  the affected baseline instead (see the pricing banner at the top of models.yml).")
		fmt.Println()
	}

	if len(unresolved) > 0 {
		fmt.Printf("UNRESOLVED SLUGS — %d row(s) name a model the catalogue no longer lists\n", len(unresolved))
		fmt.Println("  (reachability, not pricing: a run against these would 404. Not a failure by")
		fmt.Println("   default — retired rows are kept on purpose to attribute banked results.)")
		fmt.Println()
		for _, f := range unresolved {
			fmt.Printf("  %-30s %-42s %s\n", f.Entry, f.Slug, f.Note)
		}
		fmt.Println()
	}

	if len(cacheGaps) > 0 {
		fmt.Printf("CACHE RATE UNDECLARED — %d row(s), informational\n", len(cacheGaps))
		for _, f := range cacheGaps {
			fmt.Printf("  %-30s %s\n", f.Entry, f.Note)
		}
		fmt.Println()
	}

	if len(drift) == 0 {
		fmt.Println("✓ every resolvable openrouter row matches the live OpenRouter catalogue")
	}
}

func reportVendor(mismatch, unmapped []finding) {
	if len(mismatch) > 0 {
		fmt.Printf("VENDOR CROSS-CHECK — %d row(s) disagree with OpenRouter's listing of the same model\n\n", len(mismatch))
		fmt.Printf("  %-30s %-42s %-24s %-24s %s\n", "ENTRY", "MIRROR SLUG", "RECORDED in/out per 1K", "OPENROUTER in/out", "FACTOR")
		for _, f := range mismatch {
			fmt.Printf("  %-30s %-42s %-24s %-24s in x%.3f out x%.3f\n",
				f.Entry, f.Slug,
				fmt.Sprintf("%-.10g / %-.10g", f.YMLIn, f.YMLOut),
				fmt.Sprintf("%-.10g / %-.10g", f.LiveIn, f.LiveOut),
				f.InRatio, f.OutRatio)
		}
		fmt.Println()
		fmt.Println("  ADVISORY, not a verdict. We bill these models DIRECT, so the vendor's own")
		fmt.Println("  pricing page is the authority and OpenRouter is only a second opinion — one")
		fmt.Println("  that runs its own promotions (measured 2026-08-13: it listed gemini-3.7-flash")
		fmt.Println("  at half Google's published rate). Read the vendor page before editing a row.")
		fmt.Println("  A uniform x0.50 across several rows is the signature of an OpenRouter promo;")
		fmt.Println("  a lone row off by an odd factor is the signature of a stale rate in models.yml.")
		fmt.Println()
	}

	if len(unmapped) > 0 {
		fmt.Printf("NO CROSS-CHECK — %d vendor row(s) have no OpenRouter mirror, informational\n", len(unmapped))
		fmt.Println("  (nothing is known to be wrong; there is simply no second source for these.)")
		for _, f := range unmapped {
			fmt.Printf("  %-30s %s\n", f.Entry, f.Slug)
		}
		fmt.Println()
	}
}

func reportSchedule(expired, soon, invalid []finding) {
	if len(expired) > 0 {
		fmt.Printf("LAPSED PRICING — %d row(s) are past their pricing.expires date\n\n", len(expired))
		for _, f := range expired {
			fmt.Printf("  %-30s %s\n", f.Entry, f.Note)
			fmt.Printf("  %-30s still charging %-.10g / %-.10g, scheduled successor is %-.10g / %-.10g (in x%.3f out x%.3f)\n",
				"", f.YMLIn, f.YMLOut, f.LiveIn, f.LiveOut, f.InRatio, f.OutRatio)
		}
		fmt.Println()
		fmt.Println("  Confirm against the vendor's page first — an introductory rate can be extended,")
		fmt.Println("  in which case push the expires date out rather than promoting the successor.")
		fmt.Println()
	}

	if len(invalid) > 0 {
		fmt.Printf("MALFORMED SCHEDULE — %d row(s)\n", len(invalid))
		for _, f := range invalid {
			fmt.Printf("  %-30s %s\n", f.Entry, f.Note)
		}
		fmt.Println()
	}

	if len(soon) > 0 {
		fmt.Printf("UPCOMING PRICE CHANGE — %d row(s), informational\n", len(soon))
		for _, f := range soon {
			fmt.Printf("  %-30s %s: %-.10g / %-.10g -> %-.10g / %-.10g\n",
				f.Entry, f.Note, f.YMLIn, f.YMLOut, f.LiveIn, f.LiveOut)
		}
		fmt.Println()
	}
}
