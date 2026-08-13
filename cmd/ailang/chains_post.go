package main

// `ailang chains post-iteration` (M-MISSION-COST-CHAINS M2).
//
// Posts ONE chain per mission iteration to the observatory so the loop's spend
// (metered $ + quota buckets) shows up in `ailang chains`. Invoked by the
// mission-control skill's Gate 4. Fail-soft + bounded + LOUD: if the observatory
// write fails, the post is buffered to a bounded JSONL spool (stderr-loud,
// drop-oldest) that the NEXT invocation flushes. It NEVER blocks or fails the
// iteration — telemetry problems exit 0 with a stderr warning.
//
// Input is a JSON IterationPost read from --file or stdin:
//
//	{
//	  "source": "mission:v1/iter-42",
//	  "stages": [
//	    {"role":"codex-executor","provider":"codex","model":"claude-sonnet-4-5","cost_usd":0.42,"tokens_in":1000,"tokens_out":500,"status":"completed"},
//	    {"role":"controller","quota_bucket":"opus","status":"completed"},
//	    {"role":"evaluator","quota_bucket":"sonnet","status":"failed"}
//	  ]
//	}
//
// `status` is optional (omitting it leaves the stage `pending`, the pre-v0.33.2
// behaviour) but SHOULD be posted. Post the stage's REAL outcome: marking every
// stage `completed` hides the failures this record exists to surface.
//
// The success line echoes the totals actually recorded, so a payload that forgot
// its token counts is visible at the call site rather than three weeks later in a
// cost rollup.

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/sunholo-data/ailang/internal/observatory"
)

// defaultSpoolPath returns the mission iteration spool path next to observatory.db.
func defaultSpoolPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "chains-iteration-spool.jsonl"
	}
	return filepath.Join(home, ".ailang", "state", "chains-iteration-spool.jsonl")
}

func chainsPostIterationCommand() {
	fs := flag.NewFlagSet("chains post-iteration", flag.ExitOnError)
	file := fs.String("file", "", "Read the iteration JSON from this file (default: stdin)")
	spoolPath := fs.String("spool", "", "Override the spool path (default: ~/.ailang/state/chains-iteration-spool.jsonl)")
	flushOnly := fs.Bool("flush-only", false, "Only flush any buffered spool; do not read a new post")
	fs.Parse(flag.Args()[2:])

	spPath := *spoolPath
	if spPath == "" {
		spPath = defaultSpoolPath()
	}
	spool := observatory.NewSpool(spPath)

	// Connecting to the store can itself fail (locked/missing). Treat that as the
	// server-down path: buffer the new post and return fail-soft.
	backend, connErr := observatory.NewSQLiteBackendFromPath(observatory.DefaultDatabasePath())
	if backend != nil {
		defer backend.Close()
	}

	ctx := context.Background()

	// 1. Flush any previously-spooled posts first (best-effort).
	if connErr == nil {
		flushSpool(ctx, backend, spool)
	}

	if *flushOnly {
		return
	}

	// 2. Read the new post.
	post, err := readIterationPost(*file)
	if err != nil {
		// A malformed post is a caller bug, not a telemetry outage — surface it,
		// but still exit 0 so the mission loop is never blocked by telemetry.
		fmt.Fprintf(os.Stderr, "chains post-iteration: invalid input: %v\n", err)
		return
	}

	// 3. Try to post; on failure, spool it (LOUD, bounded, fail-soft).
	if connErr != nil {
		fmt.Fprintf(os.Stderr, "chains post-iteration: observatory unreachable (%v)\n", connErr)
		_ = spool.Append(post)
		return
	}

	chainID, postErr := observatory.PostIteration(ctx, backend, post)
	if postErr != nil {
		fmt.Fprintf(os.Stderr, "chains post-iteration: write failed (%v)\n", postErr)
		_ = spool.Append(post)
		return
	}
	cost, tokens, unreported := postTotals(post)
	fmt.Printf("Posted iteration chain %s (source %s, %d stages, $%.4f, %d tokens)\n",
		chainID, post.Source, len(post.Stages), cost, tokens)
	if unreported > 0 {
		// Not an error — a payload may legitimately post mid-flight stages. But an
		// unreported stage reads back `pending` forever, so say it out loud here
		// rather than let it be discovered in a cost rollup weeks later.
		fmt.Fprintf(os.Stderr, "chains post-iteration: %d of %d stages posted no status and stay 'pending'\n",
			unreported, len(post.Stages))
	}
}

// postTotals sums what the post claims, for the echo line: metered dollars, total
// tokens, and how many stages reported no outcome.
func postTotals(post *observatory.IterationPost) (cost float64, tokens, unreported int) {
	for _, st := range post.Stages {
		cost += st.CostUSD
		tokens += st.TokensIn + st.TokensOut
		if st.Status == "" {
			unreported++
		}
	}
	return cost, tokens, unreported
}

// flushSpool drains buffered posts and re-posts them; posts that still fail are
// re-spooled (LOUD) so nothing is lost.
func flushSpool(ctx context.Context, backend *observatory.SQLiteBackend, spool *observatory.Spool) {
	entries, err := spool.Drain()
	if err != nil {
		fmt.Fprintf(os.Stderr, "chains post-iteration: could not read spool (%v)\n", err)
		return
	}
	for _, p := range entries {
		if _, err := observatory.PostIteration(ctx, backend, p); err != nil {
			fmt.Fprintf(os.Stderr, "chains post-iteration: re-post of spooled %q failed (%v); re-buffering\n", p.Source, err)
			_ = spool.Append(p)
		}
	}
}

// readIterationPost reads and decodes an IterationPost from a file or stdin.
func readIterationPost(file string) (*observatory.IterationPost, error) {
	var raw []byte
	var err error
	if file != "" {
		raw, err = os.ReadFile(file)
	} else {
		raw, err = io.ReadAll(os.Stdin)
	}
	if err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		return nil, fmt.Errorf("empty input (provide --file or pipe JSON on stdin)")
	}
	var post observatory.IterationPost
	if err := json.Unmarshal(raw, &post); err != nil {
		return nil, fmt.Errorf("JSON decode: %w", err)
	}
	return &post, nil
}
