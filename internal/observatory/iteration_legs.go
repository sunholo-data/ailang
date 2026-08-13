package observatory

import (
	"context"
	"fmt"
	"io"
	"os"
)

// Node-generic dual-write for mission iterations
// (M-MISSION-LOOP-UNIFIED-TELEMETRY M3).
//
// RATIFIED (Mark, 2026-08-13): dual-write rather than mirror, and deliberately
// NODE-GENERIC — "this server, laptop, cloud, other nodes in the future".
// Nothing here knows about "the rig": a leg is just a sink plus its own spool,
// and WHICH legs exist is decided by the node's configuration at the call site
// (cmd/ailang/chains_post.go), not by any host-specific test in this package.
//
// RATIFIED: never block when a destination is unreachable — "no block if not
// available, at least until we harden availability." That is already satisfied
// structurally by the bounded+loud spool (spool.go), so this EXTENDS the spool
// to a second destination rather than inventing a fail-soft policy. Availability
// hardening is deferred, not forgotten; the spool's bounded+loud contract is
// what keeps "no block" honest rather than silently lossy.
//
// PER-LEG SPOOLS ARE LOAD-BEARING. A shared spool would replay a post that the
// local leg already stored, duplicating that chain on every subsequent flush.
// Each leg therefore buffers only its OWN failures.

// The local SQLite store is the always-present leg, and the only one that can
// record a stage's model.
var (
	_ IterationSink      = (*Store)(nil)
	_ IterationModelSink = (*Store)(nil)
)

// IterationLeg is one destination of a dual-write, with the bounded spool that
// covers its outages.
type IterationLeg struct {
	// Name identifies the leg in stderr notices ("local", "cloud", …).
	Name string
	// Sink is the destination. Nil means the destination could not be opened —
	// see Err. A nil sink still spools; it does not drop.
	Sink IterationSink
	// Err records why Sink is nil (connect-time failure).
	Err error
	// Spool buffers THIS leg's failures only.
	Spool *Spool
}

// warnTo resolves the loud-notice sink (nil => stderr, matching Spool).
func warnTo(w io.Writer) io.Writer {
	if w == nil {
		return os.Stderr
	}
	return w
}

// PostToLegs writes post to every leg, buffering per-leg failures to that leg's
// spool. It never fails: telemetry must not block or wedge a mission iteration.
// Every buffering event is loud.
//
// It returns which legs STORED the post and which only BUFFERED it, so the
// caller reports where the data actually went rather than claiming delivery it
// did not get.
//
// A post that fails VALIDATION is a caller bug, not an outage, so it is reported
// and dropped rather than spooled — buffering it would replay the same rejection
// on every future iteration and evict recoverable posts from the bounded spool.
func PostToLegs(ctx context.Context, legs []IterationLeg, post *IterationPost, warn io.Writer) (delivered, spooled []string) {
	w := warnTo(warn)

	if err := post.Validate(); err != nil {
		fmt.Fprintf(w, "chains post-iteration: invalid post %q not buffered (%v)\n", post.Source, err)
		return nil, nil
	}

	for _, leg := range legs {
		if leg.Sink == nil {
			fmt.Fprintf(w, "chains post-iteration: %s observatory unreachable (%v)\n", leg.Name, leg.Err)
			_ = leg.Spool.Append(post)
			spooled = append(spooled, leg.Name)
			continue
		}
		if _, err := PostIterationTo(ctx, leg.Sink, post); err != nil {
			fmt.Fprintf(w, "chains post-iteration: %s write failed (%v)\n", leg.Name, err)
			_ = leg.Spool.Append(post)
			spooled = append(spooled, leg.Name)
			continue
		}
		delivered = append(delivered, leg.Name)
	}
	return delivered, spooled
}

// FlushLegs drains each leg's spool and replays it against that leg. A replay
// that still fails is re-buffered (loudly) so nothing is lost. A leg whose sink
// could not be opened is skipped with its backlog intact.
func FlushLegs(ctx context.Context, legs []IterationLeg, warn io.Writer) {
	w := warnTo(warn)

	for _, leg := range legs {
		if leg.Sink == nil {
			continue // still down; leave the backlog where it is
		}
		entries, err := leg.Spool.Drain()
		if err != nil {
			fmt.Fprintf(w, "chains post-iteration: could not read %s spool (%v)\n", leg.Name, err)
			continue
		}
		for _, p := range entries {
			if _, err := PostIterationTo(ctx, leg.Sink, p); err != nil {
				fmt.Fprintf(w, "chains post-iteration: %s re-post of spooled %q failed (%v); re-buffering\n", leg.Name, p.Source, err)
				_ = leg.Spool.Append(p)
			}
		}
	}
}
