package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/sunholo-data/ailang/internal/observatory"
)

// `ailang chains reconcile` — close chains that can never progress
// (M-COMPLETION-PATH-PARITY M4).
//
// Measured in prod 2026-09-03: 400 chains, 311 "active", the oldest since
// 2026-04-27, none progressing. The cloud completion path advanced no chain, so
// once a chain was created nothing ever moved it. "Active chains" was therefore
// never a health signal, only a count that grew.
//
// This is a CLI command rather than a background sweep on purpose. It rewrites
// history in bulk, and the first run does so for hundreds of rows; that should
// be something a person invokes and reads the output of, not something that
// happens quietly at 3am. The recurring guard is a separate, much narrower
// thing: it reports chains that go stranded from now on, it does not rewrite.

func chainsReconcileCommand() {
	fs := flag.NewFlagSet("chains reconcile", flag.ExitOnError)
	remote := fs.String("remote", "", "backend: local|gcp (default $AILANG_CHAINS_READ, else local)")
	minAgeH := fs.Float64("min-age-hours", 1, "only consider chains older than this")
	apply := fs.Bool("apply", false, "actually write; without it this is a dry run")
	reason := fs.String("reason", observatory.AbandonReasonPreFix, "reason recorded on each abandoned chain")
	limit := fs.Int("limit", 0, "stop after N chains (0 = no limit)")
	_ = fs.Parse(os.Args[3:])

	ctx := context.Background()
	backend, closeBackend, err := openChainsReadBackend(ctx, *remote)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer closeBackend()

	minAge := time.Duration(*minAgeH * float64(time.Hour))
	stranded, err := backend.FindStrandedChains(ctx, minAge)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error finding stranded chains: %v\n", err)
		os.Exit(1)
	}

	if len(stranded) == 0 {
		fmt.Printf("No stranded chains older than %s. Nothing to reconcile.\n", minAge)
		return
	}

	// Show the shape of what is about to change before changing it. An operator
	// who cannot see the age distribution cannot tell a genuine backlog from a
	// scan that has caught something live.
	var oldest, newest time.Time
	for i, c := range stranded {
		if i == 0 || c.CreatedAt.Before(oldest) {
			oldest = c.CreatedAt
		}
		if i == 0 || c.CreatedAt.After(newest) {
			newest = c.CreatedAt
		}
	}
	fmt.Printf("Stranded chains: %d (older than %s)\n", len(stranded), minAge)
	fmt.Printf("  oldest: %s (%.0f days)\n", oldest.Format("2006-01-02"), time.Since(oldest).Hours()/24)
	fmt.Printf("  newest: %s (%.0f days)\n", newest.Format("2006-01-02"), time.Since(newest).Hours()/24)
	fmt.Printf("  reason: %q\n\n", *reason)

	if !*apply {
		fmt.Println("DRY RUN — nothing written. Re-run with --apply to abandon these chains.")
		shown := len(stranded)
		if shown > 10 {
			shown = 10
		}
		for _, c := range stranded[:shown] {
			fmt.Printf("  would abandon %s (created %s)\n", c.ChainID, c.CreatedAt.Format("2006-01-02 15:04"))
		}
		if len(stranded) > shown {
			fmt.Printf("  … and %d more\n", len(stranded)-shown)
		}
		return
	}

	done, skipped := 0, 0
	for i, c := range stranded {
		if *limit > 0 && i >= *limit {
			break
		}
		if err := backend.AbandonChain(ctx, c.ChainID, *reason); err != nil {
			// Report and continue: one unwritable chain must not strand the rest,
			// and the count printed at the end must be what actually happened.
			fmt.Fprintf(os.Stderr, "  skip %s: %v\n", c.ChainID, err)
			skipped++
			continue
		}
		done++
	}

	fmt.Printf("\nAbandoned %d chain(s)", done)
	if skipped > 0 {
		fmt.Printf(", %d skipped (see errors above)", skipped)
	}
	fmt.Println(".")
	fmt.Println("These are marked 'abandoned' with a reason — NOT completed or failed.")
	fmt.Println("No stage transitions were created: the record says they stopped, not what they achieved.")
}
