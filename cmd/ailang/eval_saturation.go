package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/sunholo-data/ailang/internal/eval_harness"
	"github.com/sunholo-data/ailang/internal/observatory"

	_ "github.com/mattn/go-sqlite3"
)

// runEvalTrendSaturation implements `ailang eval-trend tier-saturation`: a
// per-mode report of how saturated the benchmark suite is, derived from the ELO
// ratings (M-EVAL-RATING-EFFICIENCY part 2, M4). Saturated = Trivial band (every
// model passes — another trial yields no information) → demotion candidates; the
// rest is the discriminating set worth running. Reported per mode because
// standard and agent saturate differently.
func runEvalTrendSaturation() {
	fs := flag.NewFlagSet("eval-trend tier-saturation", flag.ExitOnError)
	dbPath := fs.String("db", "", "Ratings DB path (default: ~/.ailang/state/observatory.db)")
	if err := fs.Parse(flag.Args()[2:]); err != nil {
		os.Exit(1)
	}
	path := *dbPath
	if path == "" {
		path = defaultObservatoryDB()
	}
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open ratings db %s: %v\n", path, err)
		os.Exit(1)
	}
	defer func() { _ = db.Close() }()
	ctx := context.Background()

	fmt.Printf("Tier saturation report — %s\n", path)
	any := false
	for _, mode := range []string{"standard", "agent"} {
		benches, err := observatory.LoadBenchmarkRatings(ctx, db, mode)
		if err != nil {
			fmt.Fprintf(os.Stderr, "load %s ratings: %v\n", mode, err)
			os.Exit(1)
		}
		if len(benches) == 0 {
			continue
		}
		any = true

		bands := map[string]int{}
		var trivial []string
		discriminating := 0
		for _, b := range benches {
			band := eval_harness.Band(b.Rating)
			bands[band]++
			if band == "Trivial" {
				trivial = append(trivial, b.BenchmarkID)
			} else {
				discriminating++
			}
		}
		sort.Strings(trivial)

		fmt.Printf("\n%s (%d benchmarks):\n", mode, len(benches))
		for _, band := range []string{"Very hard", "Hard", "Moderate", "Easy", "Trivial"} {
			if bands[band] > 0 {
				fmt.Printf("  %-10s %d\n", band, bands[band])
			}
		}
		fmt.Printf("  → %d discriminating (worth running)\n", discriminating)
		fmt.Printf("  → %d saturated (Trivial — demotion candidates): %s\n", len(trivial), strings.Join(trivial, ", "))

		pct := float64(len(trivial)) / float64(len(benches)) * 100
		fmt.Printf("  Recommendation: %.0f%% saturated. ", pct)
		if pct >= 30 {
			fmt.Println("Demote the Trivial set and add harder benchmarks (M-EVAL-FRONTIER-TIER).")
		} else {
			fmt.Println("Suite still discriminating; no action needed.")
		}
	}
	if !any {
		fmt.Fprintf(os.Stderr, "\nno ratings found in %s — seed it: eval-elo <baseline> --mode all --persist %s\n", path, path)
		os.Exit(1)
	}
}
