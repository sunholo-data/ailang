package coordinator

import (
	"context"
	"os"
	"strings"

	"github.com/sunholo-data/ailang/internal/feedbackgate"
)

// feedbackGateDecider is the narrow behavior the daemon needs from the gate.
// feedbackgate.Decide satisfies it via feedbackGateFunc; tests inject a fake
// without standing up the whole gate. Keeping this an interface (rather than a
// bare func field) makes the daemon wiring and its tests symmetric with the
// rest of the coordinator.
type feedbackGateDecider interface {
	Decide(ctx context.Context, in feedbackgate.Input, cfg feedbackgate.FeedbackGateConfig) (feedbackgate.Verdict, error)
}

// feedbackGateFunc adapts the package-level feedbackgate.Decide to the
// feedbackGateDecider interface. This is the production decider.
type feedbackGateFunc struct{}

func (feedbackGateFunc) Decide(ctx context.Context, in feedbackgate.Input, cfg feedbackgate.FeedbackGateConfig) (feedbackgate.Verdict, error) {
	return feedbackgate.Decide(ctx, in, cfg)
}

// FeedbackGateConfig is the coordinator.feedback_gate config block. It embeds
// the feedbackgate.FeedbackGateConfig (YAML-tagged) so operators configure the
// gate under coordinator.feedback_gate in ~/.ailang/config.yaml. It is DISTINCT
// from the shipped coordinator.TriageConfig (coordinator.triage) — see the
// naming-disambiguation note in internal/feedbackgate/decide.go.
//
// This type alias exists so agent_config.go can reference a coordinator-package
// name in the YAML struct without importing feedbackgate at that site.
type FeedbackGateConfig = feedbackgate.FeedbackGateConfig

// resolveFeedbackGateMode applies the AILANG_FEEDBACK_GATE_MODE env override on
// top of the config value. Env wins (operator kill-switch). Returns the config
// unchanged when the env is unset/empty.
func resolveFeedbackGateMode(cfg feedbackgate.FeedbackGateConfig) feedbackgate.FeedbackGateConfig {
	if m := strings.TrimSpace(os.Getenv("AILANG_FEEDBACK_GATE_MODE")); m != "" {
		cfg.Mode = m
	}
	if isTruthyEnv(os.Getenv("AILANG_FEEDBACK_GATE_DRY_RUN")) {
		cfg.DryRun = true
	}
	return cfg
}

// isTruthyEnv reports whether an env value means "on".
func isTruthyEnv(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}

// enableFeedbackGate installs the production decider and attaches the
// cloud-constructed dependencies (M-FEEDBACK-GATE-CLOUD-ADAPTER) onto the gate
// config the decider reads. Called from initTaskProcessing when
// coordinator.feedback_gate.enabled is set. Extracted as a method so the
// deps-attachment CALL-SITE (not just the SetFeedbackGateDeps setter) is
// directly testable — the M-ENV-FORWARD lesson: guard the call-site.
//
// d.feedbackGateCfg is set to the same pointer as cfg, so the Cooldown/
// Classifier assignments below are visible to every Decide call. When a dep is
// nil (local mode / no ANTHROPIC_API_KEY), the field stays nil and that stage
// is a no-op — behavior identical to the pre-adapter, rules-only gate.
func (d *Daemon) enableFeedbackGate(cfg *FeedbackGateConfig) {
	d.feedbackGate = feedbackGateFunc{}
	d.feedbackGateCfg = cfg

	if d.feedbackGateCooldown != nil {
		d.feedbackGateCfg.Cooldown = d.feedbackGateCooldown
	}
	if d.feedbackGateClassifier != nil {
		d.feedbackGateCfg.Classifier = d.feedbackGateClassifier
	}

	resolved := resolveFeedbackGateMode(*cfg)
	d.logger.Printf("Feedback gate enabled (mode=%s, dry_run=%v, cooldown=%s, classifier=%s, budget=%s)",
		resolved.Mode, resolved.DryRun,
		feedbackGateCooldownStage(d.feedbackGateCfg),
		feedbackGateClassifierStage(d.feedbackGateCfg),
		feedbackGateBudgetStage(d.feedbackGateCfg),
	)
}

// feedbackGateCooldownStage names the cooldown stage for the startup log:
// "firestore" when a store is attached, "none" (rules-only) otherwise.
func feedbackGateCooldownStage(cfg *feedbackgate.FeedbackGateConfig) string {
	if cfg != nil && cfg.Cooldown != nil {
		return "firestore"
	}
	return "none"
}

// feedbackGateClassifierStage names the classifier stage for the startup log:
// "anthropic" when a live-provider classifier is attached, "fail-closed" when a
// classifier is attached but has no provider (no ANTHROPIC_API_KEY → files
// heuristic-flagged messages), "none" when no classifier is attached (stage
// skipped entirely).
func feedbackGateClassifierStage(cfg *feedbackgate.FeedbackGateConfig) string {
	if cfg == nil || cfg.Classifier == nil {
		return "none"
	}
	if cfg.Classifier.HasProvider() {
		return "anthropic"
	}
	return "fail-closed"
}

// feedbackGateBudgetStage names the budget stage for the startup log. The
// budget rides inside the classifier (feedbackgate.NewClassifier(provider,
// prompt, budget)), so "firestore" whenever a classifier is attached (the cloud
// wiring always builds the classifier with a Firestore budget), "none"
// otherwise.
func feedbackGateBudgetStage(cfg *feedbackgate.FeedbackGateConfig) string {
	if cfg != nil && cfg.Classifier != nil {
		return "firestore"
	}
	return "none"
}

// feedbackGateActive reports whether the gate should run for the daemon. It is
// opt-in: nil decider, nil config, disabled config, or Mode=off all mean
// pass-through (zero behavior change). Env AILANG_FEEDBACK_GATE_MODE=off also
// disables it.
func (d *Daemon) feedbackGateActive() bool {
	if d.feedbackGate == nil || d.feedbackGateCfg == nil {
		return false
	}
	if !d.feedbackGateCfg.Enabled {
		return false
	}
	if resolveFeedbackGateMode(*d.feedbackGateCfg).Mode == feedbackgate.ModeOff {
		return false
	}
	return true
}

// gateFeedbackMessage runs the gate for one cloud message and reports whether
// the coordinator should proceed to CreateTask (allowDispatch). It:
//   - builds a coordinator-free feedbackgate.Input from the Message,
//   - applies the env mode/dry-run overrides,
//   - fails CLOSED on a gate error (allowDispatch=false) with an audit,
//   - emits a feedback-gate-audit record for every non-dispatch verdict,
//   - marks a rejected message (no destructive delete),
//   - in dry-run, always allows dispatch but still audits the would-be verdict.
//
// It returns allowDispatch=true when the gate is inactive (opt-in pass-through).
func (d *Daemon) gateFeedbackMessage(msg *Message, inbox string) bool {
	if !d.feedbackGateActive() {
		return true
	}

	cfg := resolveFeedbackGateMode(*d.feedbackGateCfg)
	in := feedbackgate.Input{
		ID:       msg.ID,
		Category: msg.Type,
		Body:     msg.Content,
		From:     msg.From,
		Inbox:    inbox,
		Source:   msg.Source,
	}

	verdict, err := d.feedbackGate.Decide(d.ctx, in, cfg)
	if err != nil {
		// Fail closed: a gate error must NOT dispatch. Emit an audit so the
		// suppression is never silent (CLAUDE.md rule 2).
		d.emitGateAudit(msg, feedbackgate.Verdict{
			Action: feedbackgate.ActionFile,
			Reason: "gate_error:" + err.Error(),
		}, cfg.DryRun)
		d.logger.Printf("[feedback-gate] error deciding message %s: %v (failing closed, not dispatched)", msg.ID, err)
		return cfg.DryRun // dry-run still dispatches even on error, but audits it
	}

	if verdict.Action == feedbackgate.ActionDispatch {
		return true
	}

	// Non-dispatch verdict. Always audit.
	d.emitGateAudit(msg, verdict, cfg.DryRun)

	if cfg.DryRun {
		// Dry-run: record what we WOULD have done, but dispatch anyway.
		d.logger.Printf("[feedback-gate] DRY-RUN message %s would be %s (reason=%s) — dispatching anyway",
			msg.ID, verdict.Action, verdict.Reason)
		return true
	}

	if verdict.Action == feedbackgate.ActionReject {
		d.markFeedbackRejected(msg, verdict)
	}
	d.logger.Printf("[feedback-gate] message %s %s (reason=%s) — not dispatched",
		msg.ID, verdict.Action, verdict.Reason)
	return false
}
