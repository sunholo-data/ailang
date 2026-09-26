package main

// `ailang chains post-iteration` (M-MISSION-COST-CHAINS M2, dual-write M3).
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
//
// DUAL-WRITE (M3). The local store is always written. A node that also names a
// remote one — `--cloud <mode>`, or a plane whose observatory store is in
// Firestore (`AILANG_STORAGE=gcp` / `AILANG_STORAGE_OBSERVATORY=gcp`, the ONE
// plane switch) — gets the iteration written to BOTH, under the SAME chain and
// stage ids so spans carrying those ids join either copy. The node is a
// parameter: nothing here assumes a particular machine, and with no remote named
// the behaviour is exactly what it was.
//
// Each target keeps its OWN bounded spool. That is deliberate: sharing one would
// let a long cloud outage evict local posts that were only waiting on a locked DB.

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/mission"
	"github.com/sunholo-data/ailang/internal/observatory"
	"github.com/sunholo-data/ailang/internal/statedir"
	"github.com/sunholo-data/ailang/internal/storage"
)

// defaultSpoolPath returns the mission iteration spool path next to
// observatory.db, or "" when no state directory resolves (the spool then
// refuses to open rather than landing beside the process).
func defaultSpoolPath() string {
	p, err := statedir.Path("chains-iteration-spool.jsonl")
	if err != nil {
		return ""
	}
	return p
}

// cloudSpoolPath derives the remote target's spool from the local one, so an
// override of --spool moves both.
func cloudSpoolPath(localPath string) string {
	ext := filepath.Ext(localPath)
	return strings.TrimSuffix(localPath, ext) + "-cloud" + ext
}

// postTarget is one observatory the iteration is written to. connErr non-nil means
// the target could not be opened at all — treated exactly like a failed write
// (spool it, say so, keep going), because to the mission loop they are the same
// event: telemetry is unavailable right now.
type postTarget struct {
	name    string
	backend observatory.Backend
	spool   *observatory.Spool
	connErr error
	close   func()
}

func chainsPostIterationCommand() {
	fs := flag.NewFlagSet("chains post-iteration", flag.ExitOnError)
	file := fs.String("file", "", "Read the iteration JSON from this file (default: stdin)")
	spoolPath := fs.String("spool", "", "Override the spool path (default: ~/.ailang/state/chains-iteration-spool.jsonl)")
	flushOnly := fs.Bool("flush-only", false, "Only flush any buffered spool; do not read a new post")
	cloud := fs.String("cloud", "", "Also write to a remote observatory in this storage mode (gcp). Default: gcp when $AILANG_STORAGE_OBSERVATORY (or $AILANG_STORAGE) is gcp")
	fs.Parse(flag.Args()[2:])

	spPath := *spoolPath
	if spPath == "" {
		spPath = defaultSpoolPath()
	}
	if spPath == "" {
		fmt.Fprintf(os.Stderr, "chains post-iteration: no spool path: set %s (or HOME), or pass --spool\n", statedir.EnvVar)
		os.Exit(1)
	}

	ctx := context.Background()
	targets := openPostTargets(ctx, spPath, *cloud)
	for _, t := range targets {
		if t.close != nil {
			defer t.close()
		}
	}

	// 1. Flush any previously-spooled posts first (best-effort, per target).
	for _, t := range targets {
		if t.connErr == nil {
			flushSpool(ctx, t)
		}
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

	// 3. Write to every target. The local target runs first and backfills the post
	//    with the ids it wrote, so the remote copy shares them.
	written := []string{}
	for _, t := range targets {
		if writeToTarget(ctx, t, post) {
			written = append(written, t.name)
		}
	}
	if len(written) == 0 {
		// Everything is buffered and every failure was already reported. Do NOT
		// claim a post happened — exit 0 without a success line.
		fmt.Fprintf(os.Stderr, "chains post-iteration: %s buffered for retry, nothing written\n", post.Source)
		return
	}

	// 3b. Record subscription spend in the fleet quota ledger.
	//
	// UNCONDITIONAL, and deliberately not gated on the write above: the tokens were
	// consumed whether or not telemetry accepted the record, and a ration fed only by
	// successful writes would under-count exactly during an outage. The ledger append
	// takes no lock and cannot block, so this cannot delay the iteration.
	recordQuotaSpend(post)

	cost, tokens, unreported := postTotals(post)
	fmt.Printf("Posted iteration chain %s (source %s, %d stages, $%.4f, %d tokens)\n",
		post.ChainID, post.Source, len(post.Stages), cost, tokens)
	if unreported > 0 {
		// Not an error — a payload may legitimately post mid-flight stages. But an
		// unreported stage reads back `pending` forever, so say it out loud here
		// rather than let it be discovered in a cost rollup weeks later.
		fmt.Fprintf(os.Stderr, "chains post-iteration: %d of %d stages posted no status and stay 'pending'\n",
			unreported, len(post.Stages))
	}
}

// openPostTargets builds the write targets: always local, plus a remote one when
// this node names a storage mode for it. A target that cannot be opened is still
// RETURNED, carrying its connErr — dropping it here would silently lose the post
// instead of spooling it.
func openPostTargets(ctx context.Context, spPath, cloudFlag string) []*postTarget {
	local := &postTarget{name: "local", spool: observatory.NewSpool(spPath)}
	backend, err := observatory.NewSQLiteBackendFromPath(observatory.DefaultDatabasePath())
	local.connErr = err
	if backend != nil {
		local.backend = backend
		local.close = func() { _ = backend.Close() }
	}
	targets := []*postTarget{local}

	mode, err := chainsCloudMode(cloudFlag)
	if err != nil {
		// The remote target is named but not resolvable (a retired selector,
		// an unknown value): keep the local write and spool the remote one
		// under the error, so the post is neither lost nor silently local-only.
		remote := &postTarget{name: "cloud", spool: observatory.NewSpool(cloudSpoolPath(spPath)), connErr: err}
		return append(targets, remote)
	}
	if mode == "" {
		return targets // no remote named: unchanged, offline-first behaviour
	}

	remote := &postTarget{name: "cloud", spool: observatory.NewSpool(cloudSpoolPath(spPath))}
	if err := checkRemoteIsElsewhere(storage.Mode(mode)); err != nil {
		remote.connErr = err
		return append(targets, remote)
	}
	// Same local/gcp/hybrid resolution AILANG_STORAGE goes through — an explicit
	// mode rather than a second selector, so this node's own storage mode (and its
	// coordinator/messaging stores) are left alone.
	backends, err := storage.NewBackendsForMode(ctx, storage.Mode(mode))
	remote.connErr = err
	if backends != nil {
		remote.backend = backends.Observatory
		remote.close = func() { _ = backends.Close() }
	}
	return append(targets, remote)
}

// chainsCloudMode resolves the dual-write target: --cloud when given, else
// gcp when the plane's observatory store is in Firestore (AILANG_STORAGE=gcp
// or AILANG_STORAGE_OBSERVATORY=gcp — the scoped selector this
// replaced is a hard error naming it). "" means no remote target.
func chainsCloudMode(cloudFlag string) (string, error) {
	if cloudFlag != "" {
		if _, err := config.ParsePlane(cloudFlag); err != nil {
			return "", err
		}
		return cloudFlag, nil
	}
	sel, err := config.StoragePlane()
	if err != nil {
		return "", err
	}
	if sel.Observatory.Mode == config.StoreGCP {
		return string(config.StoreGCP), nil
	}
	return "", nil
}

// checkRemoteIsElsewhere rejects a remote target that resolves to the SAME SQLite
// file as the local one. `local` and `hybrid` both put the observatory in
// statedir.Dir() — the very directory the local target writes — so the command
// would "dual-write" an iteration into one store twice, and the second write
// would fail on the pinned ids: loudly, but for a reason nobody would guess.
//
// Before M-V1-SIMPLIFY-S2 M4 the local target ignored AILANG_STATE_DIR while
// this check read it, so the variable could name "another node's directory".
// That was the two-resolver disagreement the audit found; with one resolver
// there is no such thing as a local remote target. Only gcp is elsewhere.
func checkRemoteIsElsewhere(mode storage.Mode) error {
	if mode != storage.ModeLocal && mode != storage.ModeHybrid {
		return nil
	}
	dir, err := statedir.Dir()
	if err != nil {
		return fmt.Errorf("cannot resolve the %q remote target's directory: %w", mode, err)
	}
	return fmt.Errorf("remote target %q resolves to this node's own observatory (%s); use gcp", mode, dir)
}

// writeToTarget posts to one target, spooling on failure, and reports whether the
// write landed. It never returns an error: a telemetry outage must not block or
// fail the mission iteration.
func writeToTarget(ctx context.Context, t *postTarget, post *observatory.IterationPost) bool {
	if t.connErr != nil {
		fmt.Fprintf(os.Stderr, "chains post-iteration: %s observatory unreachable (%v); buffering\n", t.name, t.connErr)
		_ = t.spool.Append(post)
		return false
	}
	if _, err := observatory.PostIteration(ctx, t.backend, post); err != nil {
		fmt.Fprintf(os.Stderr, "chains post-iteration: %s write failed (%v); buffering\n", t.name, err)
		_ = t.spool.Append(post)
		return false
	}
	return true
}

// flushSpool drains one target's buffered posts and re-posts them; posts that
// still fail are re-spooled (LOUD) so nothing is lost.
func flushSpool(ctx context.Context, t *postTarget) {
	entries, err := t.spool.Drain()
	if err != nil {
		fmt.Fprintf(os.Stderr, "chains post-iteration: could not read %s spool (%v)\n", t.name, err)
		return
	}
	for _, p := range entries {
		if _, err := observatory.PostIteration(ctx, t.backend, p); err != nil {
			fmt.Fprintf(os.Stderr, "chains post-iteration: re-post of spooled %q to %s failed (%v); re-buffering\n", p.Source, t.name, err)
			_ = t.spool.Append(p)
		}
	}
}

// recordQuotaSpend appends this iteration's subscription spend to the fleet-wide quota
// ledger, folded to canonical buckets.
//
// Folding matters: the agent_id bucket is free text and four spellings of codex already
// exist in v1 alone, so a ledger keyed on the raw value would see two half-full buckets
// where there is one full one — and conclude both were within ration.
//
// Failures are reported and swallowed. A telemetry or bookkeeping problem must never fail
// a mission iteration; the cost of a missed append is a ration that measures low for one
// fire, which is strictly better than a fleet that stops.
func recordQuotaSpend(post *observatory.IterationPost) {
	byBucket := map[string]int64{}
	stages := map[string]int{}
	for _, st := range post.Stages {
		if st.QuotaTokens <= 0 {
			continue
		}
		ck := observatory.CanonicalQuotaBucket(st.QuotaBucket)
		if ck == "" {
			// Validate already rejects quota tokens without a bucket, so this is
			// unreachable via the CLI; keep the spend visible rather than dropping it.
			ck = "unlabeled"
		}
		byBucket[ck] += st.QuotaTokens
		stages[ck]++
	}
	if len(byBucket) == 0 {
		return
	}
	paths := mission.DefaultPaths()
	now := time.Now().UTC()
	for bucket, tok := range byBucket {
		if err := mission.AppendSpend(paths, bucket, tok, stages[bucket], now); err != nil {
			fmt.Fprintf(os.Stderr, "chains post-iteration: quota ledger append failed for %s (%v); ration will measure low\n", bucket, err)
		}
	}
	// Best-effort compaction. Skipped silently when another process holds the lock —
	// the journal is already durable and every reader folds it.
	if _, err := mission.Consolidate(paths, now); err != nil {
		fmt.Fprintf(os.Stderr, "chains post-iteration: quota ledger consolidation: %v\n", err)
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
