package coordinator

import (
	"encoding/json"
	"time"
)

// The finalisation ledger (M-COMPLETION-PATH-PARITY M1 / C1).
//
// Task finalisation performs a dozen distinct effects across the coordinator,
// observatory and messaging stores. Delivery is at-least-once, so it is replayed;
// and the three stores are NOT one transactional unit on either path — the daemon
// keeps them in three separate SQLite files, and no store method accepts a
// caller-supplied transaction. Cross-store atomicity is therefore unavailable,
// and designing around it would re-create the very path divergence this work
// exists to remove.
//
// So the ledger does not promise atomicity. It records progress, while each
// effect is made individually idempotent (M0b). At-least-once delivery plus
// idempotent effects gives effectively-once, identically on both paths — which is
// the point: the two paths converge on one protocol instead of two.
//
// The ledger lives on the task record because that is the one store both paths
// already have, so it never spans stores.

// FinalizationState is the lifecycle of a single effect.
type FinalizationState string

const (
	// FinalizationPending: claimed, not yet confirmed applied. A crash here is
	// expected and safe — the effect is idempotent, so recovery re-applies it.
	FinalizationPending FinalizationState = "pending"
	// FinalizationDone: applied and confirmed.
	FinalizationDone FinalizationState = "done"
	// FinalizationSuperseded: the effect no longer applies because another step
	// legitimately advanced the record — e.g. a stale replay tried to write
	// pending_approval over a task a human has already approved. A normal
	// outcome, not an error.
	FinalizationSuperseded FinalizationState = "superseded"
	// FinalizationFailed: terminal. Reached only after the attempt bound, and
	// deliberately visible: an effect that cannot be applied must never be
	// silently skipped.
	FinalizationFailed FinalizationState = "failed"
)

// MaxFinalizationAttempts bounds retries so a permanently failing effect becomes
// a recorded terminal state rather than an unbounded retry loop.
const MaxFinalizationAttempts = 3

// StaleFinalizationClaim is how long a pending claim may sit before the sweep
// takes it over. Every effect is a single-store write taking milliseconds, so a
// claim older than this has certainly lost its owner.
//
// This threshold governs the SWEEP ONLY. A Pub/Sub redelivery proceeds
// immediately: it carries the completion payload and every write is guarded, so
// it cannot corrupt — and making a 60s redelivery wait out a ten-minute lease
// would be an unbounded wait for no safety gain.
const StaleFinalizationClaim = 10 * time.Minute

// Effect names. These are the ledger's keys and must stay stable: renaming one
// makes every in-flight claim for it unrecoverable.
const (
	EffectTaskStatus      = "task_status"
	EffectCompletionMsg   = "completion_message"
	EffectApproval        = "approval"
	EffectHandoff         = "handoff"
	EffectStageStatus     = "stage_status"
	EffectChainStatus     = "chain_status"
	EffectMetrics         = "metrics"
	EffectStageSession    = "stage_session"
	EffectStageError      = "stage_error"
	EffectGitHubStage     = "github_stage"
	EffectWorktreeCleanup = "worktree_cleanup"
)

// FinalizationEntry is one effect's ledger row.
type FinalizationEntry struct {
	State   FinalizationState `json:"state"`
	At      time.Time         `json:"at"`
	Attempt int               `json:"attempt"`
	// Owner identifies the coordinator instance holding the claim, so the sweep
	// can report who stalled rather than merely that something did.
	Owner string `json:"owner,omitempty"`
	// Error is the last failure, kept when State is failed so the terminal state
	// explains itself.
	Error string `json:"error,omitempty"`
}

// FinalizationLedger is the per-task effect map.
type FinalizationLedger map[string]FinalizationEntry

// IsDone reports whether an effect needs no further work. Superseded counts as
// done: the record moved on, and re-applying would regress it.
func (l FinalizationLedger) IsDone(effect string) bool {
	e, ok := l[effect]
	return ok && (e.State == FinalizationDone || e.State == FinalizationSuperseded)
}

// IsExhausted reports whether an effect has hit the attempt bound and become
// terminal.
func (l FinalizationLedger) IsExhausted(effect string) bool {
	e, ok := l[effect]
	return ok && e.State == FinalizationFailed
}

// Claim marks an effect in progress and returns the updated ledger.
//
// Claiming an effect already done or superseded is a no-op, so a replay that
// races a finished finalizer does not reopen settled work.
func (l FinalizationLedger) Claim(effect, owner string, now time.Time) FinalizationLedger {
	if l == nil {
		l = FinalizationLedger{}
	}
	if l.IsDone(effect) {
		return l
	}
	prev := l[effect]
	l[effect] = FinalizationEntry{
		State:   FinalizationPending,
		At:      now,
		Attempt: prev.Attempt + 1,
		Owner:   owner,
	}
	return l
}

// Resolve records an effect's outcome.
func (l FinalizationLedger) Resolve(effect string, state FinalizationState, now time.Time, errMsg string) FinalizationLedger {
	if l == nil {
		l = FinalizationLedger{}
	}
	prev := l[effect]
	l[effect] = FinalizationEntry{
		State:   state,
		At:      now,
		Attempt: prev.Attempt,
		Owner:   prev.Owner,
		Error:   errMsg,
	}
	return l
}

// IsStale reports whether a pending claim has outlived its owner and may be
// taken over by the sweep.
func (l FinalizationLedger) IsStale(effect string, now time.Time) bool {
	e, ok := l[effect]
	if !ok || e.State != FinalizationPending {
		return false
	}
	return now.Sub(e.At) > StaleFinalizationClaim
}

// MarshalLedger serialises the ledger for storage. An empty ledger stores as the
// empty string rather than "null", so a task that has not been finalised is
// distinguishable from one whose ledger failed to write.
func MarshalLedger(l FinalizationLedger) (string, error) {
	if len(l) == 0 {
		return "", nil
	}
	b, err := json.Marshal(l)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// UnmarshalLedger parses a stored ledger. An empty or absent value is an empty
// ledger, not an error: every task predating this feature has one.
func UnmarshalLedger(s string) (FinalizationLedger, error) {
	if s == "" {
		return FinalizationLedger{}, nil
	}
	var l FinalizationLedger
	if err := json.Unmarshal([]byte(s), &l); err != nil {
		return nil, err
	}
	if l == nil {
		l = FinalizationLedger{}
	}
	return l, nil
}
