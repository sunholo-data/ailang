package coordinator

import (
	"context"
	"fmt"
	"time"

	"github.com/sunholo-data/ailang/internal/messaging"
)

// The weekly shadow report (M-AILANG-LANE-EXECUTOR-SHADOW M7) goes to the
// `user` inbox from the cloud coordinator itself: Mondays from 09:00 UTC, once
// per ISO week. "Once" is enforced by the message ID, not by memory — the
// coordinator scales to zero, and PutMessageIfAbsent is atomic.

// shadowReportCheckEvery throttles the check; the post itself is idempotent.
const shadowReportCheckEvery = time.Hour

// ShadowReportMessageID is the inbox message ID for the week containing t.
func ShadowReportMessageID(t time.Time) string {
	y, w := t.UTC().ISOWeek()
	return fmt.Sprintf("shadow-report-%d-W%02d", y, w)
}

// ShadowReportMessage builds the weekly inbox message for the 7 days to now.
func ShadowReportMessage(cs []*ShadowComparison, now time.Time) *messaging.InboxMessage {
	r := BuildShadowReport(cs, now.Add(-7*24*time.Hour), now)
	y, w := now.UTC().ISOWeek()
	return &messaging.InboxMessage{
		ID:          ShadowReportMessageID(now),
		FromAgent:   "coordinator",
		ToInbox:     "user",
		MessageType: "notification", // the store's closed set (SQLite CHECK); Firestore would take anything
		Title:       fmt.Sprintf("Shadow report %d-W%02d: %d comparisons, %d disagreements", y, w, r.Comparisons, len(r.Disagreements)),
		Payload:     r.Markdown(),
	}
}

// postWeeklyShadowReport posts the week's report when it is due and some agent
// is actually shadowed. A shadowed fleet that produced nothing is itself news,
// so an empty week is still posted.
func (d *Daemon) postWeeklyShadowReport(store ShadowComparisonStore, now time.Time) {
	if d.msgStore == nil || !d.anyAgentShadowed() {
		return
	}
	now = now.UTC()
	if now.Weekday() != time.Monday || now.Hour() < 9 || now.Sub(d.lastShadowReportCheck) < shadowReportCheckEvery {
		return
	}
	d.lastShadowReportCheck = now
	cs, err := store.ListShadowComparisonsSince(d.ctx, now.Add(-7*24*time.Hour))
	if err != nil {
		d.logger.Printf("Shadow report: list: %v", err)
		return
	}
	ctx, cancel := context.WithTimeout(d.ctx, 30*time.Second)
	defer cancel()
	msg := ShadowReportMessage(cs, now)
	posted, err := d.msgStore.PutMessageIfAbsent(ctx, msg)
	switch {
	case err != nil:
		d.logger.Printf("Shadow report: post %s: %v", msg.ID, err)
	case posted:
		d.logger.Printf("Shadow report: posted %s to the user inbox", msg.ID)
	}
}

func (d *Daemon) anyAgentShadowed() bool {
	if d.agentRegistry == nil {
		return false
	}
	for _, a := range d.agentRegistry.ListAgents() {
		if a.Shadow != nil && a.Shadow.Enabled {
			return true
		}
	}
	return false
}
