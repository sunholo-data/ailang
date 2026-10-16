package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/sunholo-data/ailang/internal/eval_analysis"
	"github.com/sunholo-data/ailang/internal/eval_harness"
	"github.com/sunholo-data/ailang/internal/modelreg"
)

// runEvalPaired emits a PAIRED comparison of two A/B arms as JSON.
//
// Exists so the weekly A/B in tools/launchd/nightly-eval.sh can bank
// per-benchmark pairs and a McNemar result without reimplementing the join in
// bash — the previous shell-side arithmetic is exactly what produced the
// 2026-07-20 artefact (an empty pass count coerced to a literal 0).
//
// Arms are read WITHOUT the re-run dedup: that key omits Trial, so a --trials N
// run would otherwise lose half its observations and the arms would stop
// lining up. See eval_analysis.LoadArmForPairing.
func runEvalPaired() {
	fs := flag.NewFlagSet("eval-paired", flag.ExitOnError)
	indent := registerJSONOnlyOutputFlags(fs)
	withPairs := fs.Bool("with-pairs", true, "Include the per-benchmark pairs array (set false for a compact summary)")
	wire := fs.Bool("wire", true, "Compare what each arm actually sent, from OpenRouter Broadcast traces (OpenRouter arms only)")
	requireWire := fs.Bool("require-wire-parity", false, "Exit 3 unless the wire check finds both arms sent the same request parameters")
	spansURL := fs.String("spans-url", "https://dashboard.ailang.sunholo.com", "Observatory holding the Broadcast traces (prod)")
	if err := fs.Parse(os.Args[2:]); err != nil {
		os.Exit(1)
	}

	args := fs.Args()
	if len(args) != 2 {
		fmt.Fprintln(os.Stderr, "Usage: ailang eval-paired [flags] <on-arm-dir> <off-arm-dir>")
		fmt.Fprintln(os.Stderr, "\nEmits paired per-benchmark outcomes plus a McNemar result.")
		os.Exit(1)
	}

	on, err := eval_analysis.LoadArmForPairing(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading ON arm %s: %v\n", args[0], err)
		os.Exit(1)
	}
	off, err := eval_analysis.LoadArmForPairing(args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading OFF arm %s: %v\n", args[1], err)
		os.Exit(1)
	}

	result := eval_analysis.PairArms(on, off)
	if *wire {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
		wp := eval_analysis.CheckWireParity(ctx, on, off, openRouterAPIName, observatorySpanFetcher(*spansURL))
		cancel()
		result.WireParity = &wp
		switch wp.Status {
		case eval_analysis.WireMismatch:
			fmt.Fprintf(os.Stderr, "WARNING: %s. The pass rates below compare two different requests, not two harnesses.\n", wp.Message)
		case eval_analysis.WireUnverified:
			fmt.Fprintf(os.Stderr, "WARNING: wire parity UNVERIFIED — %s\n", wp.Message)
		}
	}

	// Loud on stderr so it is seen by whoever runs the comparison, while stdout
	// stays clean JSON for the nightly script to bank. Advisory only — a
	// saturated control arm is a reason to doubt a null result, not a reason to
	// refuse to measure.
	if result.Headroom.Warn {
		fmt.Fprintf(os.Stderr, "WARNING: %s\n", result.Headroom.Message)
	}
	if result.Unpaired > 0 {
		fmt.Fprintf(os.Stderr, "WARNING: %d row(s) could not be paired — one arm is missing benchmarks the other ran. The comparison is over the %d pairs that matched.\n",
			result.Unpaired, len(result.Pairs))
	}
	if !*withPairs {
		result.Pairs = nil
	}

	enc := json.NewEncoder(os.Stdout)
	if indent() {
		enc.SetIndent("", "  ")
	}
	if err := enc.Encode(result); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding result: %v\n", err)
		os.Exit(1)
	}
	if *requireWire && (result.WireParity == nil || result.WireParity.Status != eval_analysis.WireMatch) {
		os.Exit(3)
	}
}

// openRouterAPIName resolves an arm's registry name to the api_name its
// Broadcast traces are filed under.
func openRouterAPIName(model string) (string, bool) {
	if err := eval_harness.InitModelsConfig(); err != nil || modelreg.GlobalModelsConfig == nil {
		return "", false
	}
	m, err := modelreg.GlobalModelsConfig.GetModel(model)
	if err != nil || m.Provider != "openrouter" || m.APIName == "" {
		return "", false
	}
	return m.APIName, true
}

// observatorySpanFetcher reads LLM Generation spans for one model from the
// observatory's spans API.
func observatorySpanFetcher(base string) eval_analysis.SpanFetcher {
	client := &http.Client{Timeout: 3 * time.Minute}
	return func(ctx context.Context, apiName string, start, end time.Time, limit int) ([]map[string]any, error) {
		q := url.Values{}
		q.Set("model", apiName)
		q.Set("limit", fmt.Sprint(limit))
		q.Set("start_after", start.UTC().Format(time.RFC3339))
		q.Set("start_before", end.UTC().Format(time.RFC3339))
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/api/observatory/spans?"+q.Encode(), nil)
		if err != nil {
			return nil, err
		}
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("observatory returned %s", resp.Status)
		}
		var spans []struct {
			Name       string         `json:"name"`
			Attributes map[string]any `json:"attributes"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&spans); err != nil {
			return nil, err
		}
		out := make([]map[string]any, 0, len(spans))
		for _, s := range spans {
			if s.Name == "LLM Generation" {
				out = append(out, s.Attributes)
			}
		}
		if len(spans) == limit && len(out) < limit {
			// The page was full: report it as full so the caller splits it.
			for len(out) < limit {
				out = append(out, map[string]any{})
			}
		}
		return out, nil
	}
}
