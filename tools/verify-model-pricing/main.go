// Command verify-model-pricing drift-checks every `provider: "openrouter"` row in
// internal/eval_harness/models.yml against the live OpenRouter models API.
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
//	0  every resolvable row matches the live rate
//	1  at least one row drifted (or, with --strict, a slug no longer resolves)
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

type finding struct {
	Entry    string  `json:"entry"`
	Slug     string  `json:"slug"`
	Kind     string  `json:"kind"` // "drift" | "unresolved" | "cache_undeclared"
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

	if *asJSON {
		all := append(append([]finding{}, drift...), unresolved...)
		all = append(all, cacheGaps...)
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
	}

	if len(drift) > 0 || (*strict && len(unresolved) > 0) {
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
