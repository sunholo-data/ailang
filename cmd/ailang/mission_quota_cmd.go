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
	"time"

	"github.com/sunholo-data/ailang/internal/mission"
)

func missionQuota(args []string) error {
	fs := flag.NewFlagSet("mission quota", flag.ContinueOnError)
	asJSON := fs.Bool("json", false, "Emit the ledger as JSON")
	bucket := fs.String("bucket", "", "Report only this bucket (codex, anthropic, openrouter, ollama)")
	consolidate := fs.Bool("consolidate", false, "Compact the journal into the ledger cache before reporting")
	if err := fs.Parse(args); err != nil {
		return err
	}

	paths := mission.DefaultPaths()
	now := time.Now().UTC()

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
		if len(filtered) == 0 {
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

	if *asJSON {
		out := struct {
			*mission.Ledger
			At       time.Time               `json:"at"`
			Verdicts []mission.RationVerdict `json:"verdicts"`
		}{Ledger: ledger, At: now, Verdicts: ledger.Verdicts(now)}
		body, err := json.MarshalIndent(out, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(body))
		return nil
	}

	fmt.Print(ledger.String(now))
	for _, u := range ledger.Usage {
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
