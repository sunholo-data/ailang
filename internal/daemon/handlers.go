package daemon

import (
	"fmt"
	"strings"

	"github.com/sunholo-data/ailang/internal/messaging"
	"github.com/sunholo-data/ailang/internal/notify"
	"github.com/sunholo-data/ailang/internal/pubsub"
)

const (
	messageBodyMax  = 220
	taskGroupPrefix = "ailang-task-"
)

// taskNotification builds a Notification for a task event. The second return
// value reports whether to fire — only "pending_approval" / "completed" /
// "failed" produce notifications; intermediate states (running, queued) are
// skipped.
func taskNotification(t pubsub.TaskCompletion) (notify.Notification, bool) {
	switch t.Status {
	case "pending_approval":
		return notify.Notification{
			Title:     "⏳ Approval needed",
			Subtitle:  t.AgentID,
			Body:      fmt.Sprintf("%s: %s", t.AgentID, t.TaskID),
			Sound:     "Glass",
			Group:     taskGroupPrefix + t.TaskID,
			URL:       taskURL(t.TaskID),
			EventType: "pending_approval",
		}, true
	case "completed":
		body := fmt.Sprintf("%s: task %s", t.AgentID, t.TaskID)
		if t.NumTurns > 0 {
			body += fmt.Sprintf(" (%d turns, $%.4f)", t.NumTurns, t.CostUSD)
		}
		return notify.Notification{
			Title:     "✅ Task done",
			Subtitle:  t.AgentID,
			Body:      body,
			Sound:     "Ping",
			Group:     taskGroupPrefix + t.TaskID,
			URL:       taskURL(t.TaskID),
			EventType: "completed",
		}, true
	case "failed":
		body := fmt.Sprintf("%s: %s", t.AgentID, t.TaskID)
		if t.ErrorMsg != "" {
			body += " — " + t.ErrorMsg
		}
		return notify.Notification{
			Title:     "❌ Task failed",
			Subtitle:  t.AgentID,
			Body:      truncate(body, messageBodyMax),
			Sound:     "Basso",
			Group:     taskGroupPrefix + t.TaskID,
			URL:       taskURL(t.TaskID),
			EventType: "failed",
		}, true
	default:
		return notify.Notification{}, false
	}
}

// pkgInboxPrefix is the inbox-name prefix the feedback publisher uses to route
// package-scoped feedback (e.g. "pkg:sunholo/auth"). See
// internal/feedback/publisher.go.
const pkgInboxPrefix = "pkg:"

// isExternalFeedbackInbox reports whether an inbox carries externally-sourced
// user feedback that should reach Discord. Today that is the literal
// "public-feedback" inbox OR any package-scoped "pkg:*" inbox (which the
// feedback publisher routes package feedback to). Naming the rule here keeps it
// unit-testable and gives a single place for a future Source=external flag to
// extend it — instead of widening the Discord allow-list to all "message"
// traffic (which would leak internal inbox chatter to Discord).
func isExternalFeedbackInbox(inbox string) bool {
	return inbox == "public-feedback" || strings.HasPrefix(inbox, pkgInboxPrefix)
}

// messageNotification builds a Notification for an inbox message. Externally-
// sourced feedback (public-feedback + pkg:* inboxes) gets a dedicated 🌐 prefix
// and the "public-feedback" EventType (which the Discord allow-list accepts);
// everything else uses the generic shape and stays EventType "message" (macOS
// only, dropped by Discord).
func messageNotification(m *messaging.InboxMessage) (notify.Notification, bool) {
	if m == nil {
		return notify.Notification{}, false
	}
	if isExternalFeedbackInbox(m.ToInbox) {
		return notify.Notification{
			Title:     "🌐 External feedback",
			Subtitle:  m.FromAgent,
			Body:      truncate(fmt.Sprintf("[%s] %s", m.ToInbox, m.Title), messageBodyMax),
			Sound:     "Pop",
			Group:     "ailang-public-feedback",
			URL:       inboxURL(m.ToInbox),
			EventType: "public-feedback",
		}, true
	}
	return notify.Notification{
		Title:     "✉️  Message from " + m.FromAgent,
		Subtitle:  m.ToInbox,
		Body:      truncate(fmt.Sprintf("%s — %s", m.Title, m.Payload), messageBodyMax),
		Sound:     "Pop",
		Group:     "ailang-msg-" + m.ToInbox,
		URL:       inboxURL(m.ToInbox),
		EventType: "message",
	}, true
}

// shouldExclude reports whether the notification matches any exclude pattern
// (substring match on Title). Used to honour ~/.ailang/config/notify_excludes.conf.
func shouldExclude(n notify.Notification, excludes []string) bool {
	for _, p := range excludes {
		if p == "" {
			continue
		}
		if strings.Contains(n.Title, p) || strings.Contains(n.Body, p) {
			return true
		}
	}
	return false
}

func taskDedupKey(taskID, status string) string {
	return "task:" + taskID + ":" + status
}

func messageDedupKey(messageID string) string {
	return "msg:" + messageID
}

// dashboardURL returns the dashboard root. Earlier iterations tried to deep-link
// to /tasks/<id> and /inbox/<name>, but the dashboard is a single-page app
// without client-side routes for those paths — every deep link 404'd. Until
// the dashboard ships those routes, click-actions open the root and the user
// navigates from there.
const dashboardURL = "https://dashboard.ailang.sunholo.com/"

func taskURL(_ string) string {
	return dashboardURL
}

func inboxURL(_ string) string {
	return dashboardURL
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	const ellipsis = "…" // 3 bytes (UTF-8)
	cut := max - len(ellipsis)
	if cut < 0 {
		cut = 0
	}
	return s[:cut] + ellipsis
}
