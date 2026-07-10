// Package feedbackgate is a deterministic-first cost & abuse gate that sits
// between the coordinator's cloud Pub/Sub pickup and CreateTask. It stops a
// flood of anonymous public submit_feedback submissions from fanning out to
// Sonnet-driven agents.
//
// # Naming disambiguation (READ FIRST)
//
// There is a DIFFERENT, already-shipped feature — M-MSG-TRIAGE-ROUTER — living
// in internal/coordinator/triage_router.go with config key coordinator.triage,
// type coordinator.Decision (Hold/Promote/Drop), TriageConfig, and classify().
// That feature does local intake-inbox → design-doc-creator promotion. This
// package is deliberately NOT that: it uses feedbackgate.Verdict
// (Dispatch/File/Reject), FeedbackGateConfig, config key coordinator.feedback_gate,
// and env AILANG_FEEDBACK_GATE_*. Do NOT entangle the two.
//
// # Import-cycle avoidance
//
// The M4 wiring point (pollAndProcessTasksCloud) iterates []*coordinator.Message.
// To keep this package importable without a coordinator import cycle, Decide
// operates on a narrow Input struct the coordinator populates from its Message.
// This package MUST NOT import internal/coordinator.
package feedbackgate

import (
	"context"
	"strings"
)

// Verdict is the gate outcome for a single feedback submission.
//
// Action is one of the ActionDispatch/ActionFile/ActionReject constants:
//   - dispatch: allow the message through to CreateTask (agentic fan-out).
//   - file:     suppress dispatch but keep the doc for later human triage.
//   - reject:   suppress dispatch and mark the doc rejected (TTL cleanup).
type Verdict struct {
	Action string  // ActionDispatch | ActionFile | ActionReject
	Reason string  // structured reason code (see reason constants)
	Cost   float64 // estimated USD if dispatched (0 for file/reject)
}

// Action constants for Verdict.Action. Using named constants keeps the string
// values in one place and lets callers switch without magic strings.
const (
	ActionDispatch = "dispatch"
	ActionFile     = "file"
	ActionReject   = "reject"
)

// Input is the coordinator-free view of a message the gate needs. The
// coordinator populates it from its own Message type (see the M4 wiring), so
// this package never imports internal/coordinator.
type Input struct {
	ID       string // coordinator.Message.ID
	Category string // coordinator.Message.Type (may carry an "auto:" prefix)
	Body     string // coordinator.Message.Content
	From     string // coordinator.Message.From (sender)
	Inbox    string // coordinator.Message.Inbox (routing target)
	Source   string // coordinator.Message.Source (Pub/Sub topic)
}

// estimatedDispatchCostUSD is the assumed agent cost of a single dispatched
// feedback message. Used only to populate Verdict.Cost for audit/telemetry;
// it is not a live pricing lookup. Rough Sonnet-run estimate from the design
// doc's attack-scenario math ($0.015–0.05/call).
const estimatedDispatchCostUSD = 0.03

// Decide is the single entry point. It runs the deterministic pre-filter
// (M1), then — when the pre-filter would dispatch — the per-contact cooldown
// (M2) and last-resort classifier (M3). Each later stage is injected via cfg
// and is a no-op when nil, so at M1 Decide is pure (no IO).
//
// Ordering: cheap CPU rules first; cooldown only when rules say dispatch;
// classifier only when cooldown says dispatch AND a heuristic flags the
// message. Every non-dispatch path carries a structured Reason so the caller
// can emit an audit record (no silent drops — CLAUDE.md rule 2).
func Decide(ctx context.Context, in Input, cfg FeedbackGateConfig) (Verdict, error) {
	cfg = cfg.normalized()

	// Stage 1: deterministic rules (pure, <1ms).
	v := applyRules(in, cfg)
	if v.Action != ActionDispatch {
		return v, nil
	}

	// Stage 2: per-contact sliding cooldown (M2). Only consulted when the
	// rules would dispatch. Nil store => no-op (pure M1 behavior).
	if cfg.Cooldown != nil {
		cdVerdict, err := applyCooldown(ctx, in, cfg)
		if err != nil {
			return Verdict{}, err
		}
		if cdVerdict.Action != ActionDispatch {
			return cdVerdict, nil
		}
	}

	// Stage 3: last-resort classifier (M3). Only consulted when rules +
	// cooldown would dispatch. Nil classifier => no-op.
	if cfg.Classifier != nil {
		clVerdict, err := applyClassifier(ctx, in, cfg)
		if err != nil {
			return Verdict{}, err
		}
		if clVerdict.Action != ActionDispatch {
			return clVerdict, nil
		}
	}

	return Verdict{Action: ActionDispatch, Reason: ReasonPassed, Cost: estimatedDispatchCostUSD}, nil
}

// strippedCategory returns the category with any leading "auto:" prefix
// removed. The publisher (internal/feedback/publisher.go) folds the
// auto_dispatch bool into an "auto:" category prefix on the wire; there is no
// separate boolean, so the gate reads the prefix off the category.
func strippedCategory(category string) string {
	return strings.TrimPrefix(category, autoPrefix)
}

// hasAutoPrefix reports whether the category is authorized for dispatch, i.e.
// it carries the "auto:" prefix the publisher stamps when AutoDispatch is set.
func hasAutoPrefix(category string) bool {
	return strings.HasPrefix(category, autoPrefix)
}
