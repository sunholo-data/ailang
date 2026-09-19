package coordinator

import (
	"encoding/json"
	"fmt"

	"github.com/sunholo-data/ailang/internal/feedbackgate"
	"github.com/sunholo-data/ailang/internal/messaging"
)

// feedbackGateAuditInbox is the inbox where the gate records every suppressed
// (filed/rejected) or dry-run would-be-suppressed message. Named to avoid any
// collision with the shipped M-MSG-TRIAGE-ROUTER audit surfaces.
const feedbackGateAuditInbox = "feedback-gate-audit"

// gateAuditPayload is the structured body of an audit record. JSON so a
// dashboard/CLI can group by reason without parsing prose.
type gateAuditPayload struct {
	MessageID   string  `json:"message_id"`
	Action      string  `json:"action"`
	Reason      string  `json:"reason"`
	Category    string  `json:"category"`
	From        string  `json:"from"`
	Inbox       string  `json:"inbox"`
	EstCostUSD  float64 `json:"est_cost_usd"`
	DryRun      bool    `json:"dry_run"`
	WouldReject bool    `json:"would_reject"`
	// Shadow is present only when the System One shadow ran beside the
	// classifier (M-AI-DECIDE-SYSTEM-ONE audit site #1). Additive: rows
	// without it are unchanged.
	Shadow *gateShadowAudit `json:"shadow,omitempty"`
}

// gateShadowAudit is the banked shadow row (design rule D6: the whole
// decision travels — model, id, distributions, usage — never argmax alone).
type gateShadowAudit struct {
	Transport    string             `json:"transport"`
	Model        string             `json:"model"`
	ID           string             `json:"id,omitempty"`
	Degraded     bool               `json:"degraded"`
	DegradedWhy  string             `json:"degraded_why,omitempty"`
	WouldAction  string             `json:"would_action"`
	WouldReason  string             `json:"would_reason"`
	Agrees       bool               `json:"agrees"`
	Genuine      float64            `json:"genuine_p"`
	Injection    float64            `json:"injection_p"`
	Category     string             `json:"category"`
	CategoryConf float64            `json:"category_confidence"`
	CategoryDist map[string]float64 `json:"category_dist,omitempty"`
	Value        string             `json:"value"`
	ValueScore   float64            `json:"value_score"`
	ValueConf    float64            `json:"value_confidence"`
	ValueDist    map[string]float64 `json:"value_dist,omitempty"`
	LatencyMs    int                `json:"latency_ms"`
	InputTokens  int                `json:"input_tokens"`
	CostUSD      float64            `json:"cost_usd"`
	ListPriceUSD float64            `json:"list_price_usd"`
	ModelError   string             `json:"model_error,omitempty"`
	RunnerError  string             `json:"runner_error,omitempty"`
}

// shadowAudit flattens a ShadowVerdict into the audit row; nil when the shadow
// did not run.
func shadowAudit(sv *feedbackgate.ShadowVerdict) *gateShadowAudit {
	if sv == nil {
		return nil
	}
	r := sv.Result
	return &gateShadowAudit{
		Transport: r.Transport, Model: r.Model, ID: r.ID, Degraded: r.Degraded, DegradedWhy: r.DegradedWhy,
		WouldAction: sv.WouldAction, WouldReason: sv.WouldReason, Agrees: sv.Agrees,
		Genuine: r.Genuine.P, Injection: r.Injection.P,
		Category: r.Category.Choice, CategoryConf: r.Category.Confidence, CategoryDist: r.Category.Probabilities,
		Value: r.Value.Label, ValueScore: r.Value.Score, ValueConf: r.Value.Confidence, ValueDist: r.Value.Probabilities,
		LatencyMs: r.LatencyMs, InputTokens: r.InputTokens, CostUSD: r.CostUSD, ListPriceUSD: r.ListPriceUSD,
		ModelError: r.Error, RunnerError: sv.Err,
	}
}

// emitGateAudit writes a feedback-gate-audit inbox message. Failures to write
// are logged (not fatal) — the primary decision has already been made; the
// audit is for review. Never a silent drop of the DECISION (the caller always
// logs), only best-effort persistence of the audit record.
func (d *Daemon) emitGateAudit(msg *Message, verdict feedbackgate.Verdict, dryRun bool) {
	if d.msgStore == nil {
		return
	}
	payload, _ := json.Marshal(gateAuditPayload{
		MessageID:   msg.ID,
		Action:      verdict.Action,
		Reason:      verdict.Reason,
		Category:    msg.Type,
		From:        msg.From,
		Inbox:       msg.Inbox,
		EstCostUSD:  verdict.Cost,
		DryRun:      dryRun,
		WouldReject: verdict.Action == feedbackgate.ActionReject,
		Shadow:      shadowAudit(verdict.Shadow),
	})

	title := fmt.Sprintf("feedback-gate %s: %s", verdict.Action, verdict.Reason)
	if verdict.Shadow != nil {
		agree := "agrees"
		if !verdict.Shadow.Agrees {
			agree = "DISAGREES"
		}
		title = fmt.Sprintf("%s [shadow %s: %s]", title, agree, verdict.Shadow.WouldAction)
	}
	if dryRun {
		title = "DRY-RUN " + title
	}

	audit := &messaging.InboxMessage{
		FromAgent:   "feedback-gate",
		ToInbox:     feedbackGateAuditInbox,
		MessageType: "audit",
		Title:       title,
		Payload:     string(payload),
		Category:    verdict.Reason,
		Status:      messaging.InboxStatusUnread,
	}
	if err := d.msgStore.InsertInboxMessage(audit); err != nil {
		d.logger.Printf("[feedback-gate] failed to write audit for %s: %v", msg.ID, err)
	}
}

// markFeedbackRejected records a rejected message's disposition. Per the
// adopted decision (Open Q1), it does NOT destructively delete the Firestore
// doc — it acks the source message (so it isn't re-processed) and relies on TTL
// for cleanup; the audit record above preserves the reason.
//
// NOTE (deviation): the messaging store exposes no "set status=rejected"
// mutator, so we ack via MarkInboxMessageRead. The rejected status lives in the
// audit payload (WouldReject/action=reject), not on the source row. This keeps
// the change non-destructive per the plan while staying within the existing
// store API.
func (d *Daemon) markFeedbackRejected(msg *Message, verdict feedbackgate.Verdict) {
	if d.msgStore == nil {
		return
	}
	if err := d.msgStore.MarkInboxMessageRead(msg.ID); err != nil {
		d.logger.Printf("[feedback-gate] failed to ack rejected message %s: %v", msg.ID, err)
		return
	}
	d.logger.Printf("[feedback-gate] marked message %s rejected (reason=%s, TTL cleanup)", msg.ID, verdict.Reason)
}
