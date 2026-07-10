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
	})

	title := fmt.Sprintf("feedback-gate %s: %s", verdict.Action, verdict.Reason)
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
