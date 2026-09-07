package main

// `ailang mission quota` — the fleet-wide quota ledger (M-QUOTA-RATIONING-ROUTING M2).
//
// Routing has always been able to ask "is this lane up?" and never "can it afford to be
// used?". This is where the second question is answered: consumption per (bucket, window)
// against a 10%/day ration, fleet-wide because that is what the subscription is.

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/sunholo-data/ailang/internal/mission"
)

func missionQuota(args []string) error {
	return missionQuotaWithPaths(args, mission.DefaultPaths(), time.Now().UTC())
}

func missionQuotaWithPaths(args []string, paths mission.Paths, now time.Time) error {
	fs := flag.NewFlagSet("mission quota", flag.ContinueOnError)
	asJSON := fs.Bool("json", false, "Emit the ledger as JSON")
	bucket := fs.String("bucket", "", "Report only this bucket (codex, anthropic, openrouter, ollama)")
	consolidate := fs.Bool("consolidate", false, "Compact the journal into the ledger cache before reporting")
	over := fs.Bool("over", false, "Print buckets unavailable for quota routing, one per line. Codex uses local provider percentages. Ollama uses its OLLAMA_API_KEY usage gauge (95% cutoff), with optional verified pacing metadata. Both block unknown quota; other buckets require proven ledger exceedance.")
	if err := fs.Parse(args); err != nil {
		return err
	}

	codexHome := os.Getenv("CODEX_HOME")
	if codexHome == "" {
		codexHome = filepath.Join(paths.Home, ".codex")
	}
	var codex *mission.CodexQuotaObservation
	if *bucket == "" || *bucket == "codex" {
		observation := mission.ObserveCodexQuota(codexHome, now)
		codex = &observation
	}

	var ollama *mission.OllamaQuotaObservation
	if *bucket == "" || *bucket == "ollama" {
		observation := mission.ObserveOllamaQuota(paths, os.Getenv("OLLAMA_API_KEY"), now)
		ollama = &observation
	}

	if *consolidate {
		ran, err := mission.Consolidate(paths, now)
		if err != nil {
			return err
		}
		if !ran {
			// Not an error: the journal is durable and the report below folds it
			// anyway. Say so rather than let the flag look like it did nothing.
			fmt.Fprintln(os.Stderr, "quota: another process holds the consolidation lock; reporting from the journal instead")
		}
	}

	ledger, err := mission.LoadLedger(paths, now)
	if err != nil {
		if *over && emitProviderQuotaBlocks(codex, ollama) {
			fmt.Fprintf(os.Stderr, "quota: token ledger unavailable: %v\n", err)
			return nil
		}
		return err
	}
	if *bucket != "" {
		canon := *bucket
		filtered := ledger.Usage[:0:0]
		for _, u := range ledger.Usage {
			if u.Bucket == canon {
				filtered = append(filtered, u)
			}
		}
		if len(filtered) == 0 && canon != "codex" && canon != "ollama" {
			// An empty result is a claim ("nothing spent") that could equally mean
			// "wrong name". Distinguish them.
			known := map[string]bool{}
			for _, u := range ledger.Usage {
				known[u.Bucket] = true
			}
			if !known[canon] {
				names := make([]string, 0, len(known))
				for k := range known {
					names = append(names, k)
				}
				if len(names) == 0 {
					return fmt.Errorf("no spend recorded for any bucket yet")
				}
				return fmt.Errorf("no bucket %q in the ledger (have: %v)", canon, names)
			}
		}
		ledger.Usage = filtered
	}

	// --over is the machine seam for routing. It prints ONLY computed exceedances:
	// a bucket whose capacity is unknown is NOT listed, because rationing on a
	// guessed capacity would either idle a healthy fleet or wave through an empty
	// bucket, and either way be confident about it. Silence therefore means "no
	// bucket is PROVEN over", never "everything is fine".
	if *over {
		printed := map[string]bool{}
		for _, v := range ledger.Verdicts(now) {
			// Codex is admitted using provider percentages, never inferred token capacity.
			if v.Bucket != "codex" && v.Bucket != "ollama" && v.Over() && !printed[v.Bucket] {
				fmt.Println(v.Bucket)
				printed[v.Bucket] = true
			}
		}
		emitProviderQuotaBlocks(codex, ollama)
		return nil
	}

	verdicts := ledger.Verdicts(now)
	filteredVerdicts := verdicts[:0:0]
	for _, v := range verdicts {
		if v.Bucket != "codex" && v.Bucket != "ollama" {
			filteredVerdicts = append(filteredVerdicts, v)
		}
	}
	if *asJSON {
		out := struct {
			*mission.Ledger
			At       time.Time                       `json:"at"`
			Verdicts []mission.RationVerdict         `json:"verdicts"`
			Codex    *mission.CodexQuotaObservation  `json:"codex_provider_usage,omitempty"`
			Ollama   *mission.OllamaQuotaObservation `json:"ollama_provider_usage,omitempty"`
		}{Ledger: ledger, At: now, Verdicts: filteredVerdicts, Codex: codex, Ollama: ollama}
		body, err := json.MarshalIndent(out, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(body))
		return nil
	}

	display := *ledger
	display.Usage = nil
	for _, u := range ledger.Usage {
		if u.Bucket != "codex" && u.Bucket != "ollama" {
			display.Usage = append(display.Usage, u)
		}
	}
	fmt.Print(display.String(now))
	if codex != nil {
		fmt.Printf("codex provider usage: %s — %s\n", codex.State, codex.Reason)
		for _, w := range codex.Windows {
			fmt.Printf("  %dm: %.1f%% used / %.1f%% allowed; resets %s (observed %s)\n", w.WindowMinutes, w.UsedPercent, w.AllowancePercent, w.ResetsAt.Format(time.RFC3339), codex.ObservedAt.Format(time.RFC3339))
		}
	}
	if ollama != nil {
		fmt.Printf("ollama provider usage: %s — %s\n", ollama.State, ollama.Reason)
		if ollama.SessionUsage != nil && ollama.WeeklyUsage != nil {
			fmt.Printf("  fractional gauge: session %.1f%%; weekly %.1f%%\n", 100**ollama.SessionUsage, 100**ollama.WeeklyUsage)
		}
	}
	for _, u := range ledger.Usage {
		if u.Bucket == "codex" || u.Bucket == "ollama" {
			continue
		} // Provider percentages govern Codex admission.
		if u.Capacity <= 0 {
			// LOUD, per D-2: an unrationed bucket is a bucket nothing is pacing.
			fmt.Fprintf(os.Stderr, "quota: %s/%s has no known capacity — UNRATIONED until a provider probe supplies one\n", u.Bucket, u.Window)
		}
		if u.BoundarySource() == "local" {
			fmt.Fprintf(os.Stderr, "quota: %s/%s window boundaries are derived locally, not from a provider reset\n", u.Bucket, u.Window)
		}
	}
	return nil
}

// Preserve provider admission independently of the optional token journal.
func emitProviderQuotaBlocks(codex *mission.CodexQuotaObservation, ollama *mission.OllamaQuotaObservation) bool {
	blocked := false
	if codex != nil && codex.Blocked() {
		fmt.Println("codex")
		fmt.Fprintf(os.Stderr, "quota: codex %s: %s\n", codex.State, codex.Reason)
		blocked = true
	}
	if ollama != nil && ollama.Blocked() {
		fmt.Println("ollama")
		fmt.Fprintf(os.Stderr, "quota: ollama %s: %s\n", ollama.State, ollama.Reason)
		blocked = true
	}
	return blocked
}
