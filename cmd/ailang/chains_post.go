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
// remote one — `--cloud <mode>` or `AILANG_CHAINS_CLOUD=<mode>`, resolved through
// internal/storage, the same selector `AILANG_STORAGE` uses — gets the iteration
// written to BOTH, under the SAME chain and stage ids so spans carrying those ids
// join either copy. The node is a parameter: nothing here assumes a particular
// machine, and with no remote named the behaviour is exactly what it was.
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
	cloud := fs.String("cloud", "", "Also write to a remote observatory in this storage mode (gcp). Default: $AILANG_CHAINS_CLOUD")
	fs.Parse(flag.Args()[2:])

	spPath := *spoolPath
	if spPath == "" {
		spPath = defaultSpoolPath()
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

	mode := cloudFlag
	if mode == "" {
		mode = os.Getenv("AILANG_CHAINS_CLOUD")
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

// checkRemoteIsElsewhere rejects a remote target that resolves to the SAME SQLite
// file as the local one. `local` and `hybrid` both put the observatory in
// $AILANG_STATE_DIR (default ~/.ailang/state), so without this the command would
// "dual-write" an iteration into one store twice, and the second write would fail
// on the pinned ids — loudly, but for a reason nobody would guess. Naming another
// node's state directory is still allowed; only writing to yourself is not.
func checkRemoteIsElsewhere(mode storage.Mode) error {
	if mode != storage.ModeLocal && mode != storage.ModeHybrid {
		return nil
	}
	remoteDir := os.Getenv("AILANG_STATE_DIR")
	if remoteDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("cannot resolve the %q remote target's directory: %w", mode, err)
		}
		remoteDir = filepath.Join(home, ".ailang", "state")
	}
	localDir := filepath.Dir(observatory.DefaultDatabasePath())
	if filepath.Clean(remoteDir) == filepath.Clean(localDir) {
		return fmt.Errorf("remote target %q resolves to this node's own observatory (%s); "+
			"use gcp, or point AILANG_STATE_DIR at a different store", mode, localDir)
	}
	return nil
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
