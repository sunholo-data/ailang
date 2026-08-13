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
// M-MISSION-LOOP-UNIFIED-TELEMETRY M2 added a per-stage `status` and chain-total
// aggregation; M3 made the destination NODE-GENERIC — this node dual-writes to
// local AND cloud when a cloud observatory is configured, each leg with its own
// bounded spool.
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
// `status` is OPTIONAL (an older payload omitting it keeps working and leaves the
// stage pending) and is PER STAGE: a stage that failed must say `failed`, because
// blanket-completing an iteration's stages would satisfy "nothing left pending"
// while hiding the failure.

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/sunholo-data/ailang/internal/observatory"
	"github.com/sunholo-data/ailang/internal/storage"
)

// defaultSpoolPath returns the mission iteration spool path next to observatory.db.
func defaultSpoolPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "chains-iteration-spool.jsonl"
	}
	return filepath.Join(home, ".ailang", "state", "chains-iteration-spool.jsonl")
}

// cloudSpoolPath derives the cloud leg's spool from the local one. The legs spool
// SEPARATELY on purpose: a shared buffer would replay a post the local leg
// already stored, duplicating that chain on every flush.
func cloudSpoolPath(localSpool string) string {
	ext := filepath.Ext(localSpool)
	return strings.TrimSuffix(localSpool, ext) + "-cloud" + ext
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

	ctx := context.Background()

	legs, closeLegs := iterationLegs(ctx, observatory.DefaultDatabasePath(), spPath)
	defer closeLegs()

	// 1. Flush any previously-spooled posts first, per leg (best-effort).
	observatory.FlushLegs(ctx, legs, os.Stderr)

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

	// 3. Write to every leg; per-leg failures spool (LOUD, bounded, fail-soft).
	// Report where the data ACTUALLY went — claiming delivery to a leg that only
	// buffered would make a cloud outage invisible in the loop's own output.
	delivered, spooled := observatory.PostToLegs(ctx, legs, post, os.Stderr)
	if len(delivered) > 0 {
		fmt.Printf("Posted iteration %s (%d stages) to %s\n", post.Source, len(post.Stages), strings.Join(delivered, "+"))
	}
	if len(spooled) > 0 {
		fmt.Printf("Buffered iteration %s for %s (will flush next invocation)\n", post.Source, strings.Join(spooled, "+"))
	}
}

// iterationLegs builds this NODE's dual-write destinations. The local SQLite
// observatory is always a leg (`ailang chains` is offline-first). The cloud leg
// exists only when this node is configured for one — nothing here is
// rig-specific, so a laptop, this server or a Cloud Run job with the same
// AILANG_STORAGE configuration behave identically, and a node with no cloud
// configured behaves exactly as it did before M3.
//
// A leg that cannot be OPENED is still returned, with Sink nil and Err set, so
// the post is buffered rather than dropped.
func iterationLegs(ctx context.Context, dbPath, localSpool string) ([]observatory.IterationLeg, func()) {
	var closers []func()
	closeAll := func() {
		for _, c := range closers {
			c()
		}
	}

	local := observatory.IterationLeg{
		Name:  "local",
		Spool: observatory.NewSpool(localSpool),
	}
	backend, err := observatory.NewSQLiteBackendFromPath(dbPath)
	if err != nil {
		local.Err = err
	} else {
		local.Sink = backend.Store()
		closers = append(closers, func() { _ = backend.Close() })
	}
	legs := []observatory.IterationLeg{local}

	// The cloud leg is opt-in via the SAME selector every other AILANG service
	// uses (AILANG_STORAGE); adding a second selection mechanism would be one
	// more thing to keep in sync. Only "gcp" names a genuinely remote
	// observatory — "hybrid" still resolves the observatory to local SQLite, so
	// treating it as a cloud leg would dual-write the same database twice.
	if storage.GetMode() != storage.ModeGCP {
		return legs, closeAll
	}

	cloud := observatory.IterationLeg{
		Name:  "cloud",
		Spool: observatory.NewSpool(cloudSpoolPath(localSpool)),
	}
	backends, err := storage.NewBackends(ctx)
	if err != nil {
		cloud.Err = err
	} else {
		cloud.Sink = backends.Observatory
		closers = append(closers, func() { _ = backends.Close() })
	}
	return append(legs, cloud), closeAll
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
