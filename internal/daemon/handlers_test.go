package daemon

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/messaging"
	"github.com/sunholo-data/ailang/internal/notify"
	"github.com/sunholo-data/ailang/internal/pubsub"
)

func TestTaskNotification_Pending(t *testing.T) {
	n, fire := taskNotification(pubsub.TaskCompletion{
		TaskID:  "t1",
		AgentID: "design-doc-creator",
		Status:  "pending_approval",
	})
	if !fire {
		t.Fatal("expected fire=true for pending_approval")
	}
	if !strings.Contains(n.Title, "Approval needed") {
		t.Errorf("expected approval title, got %q", n.Title)
	}
	if !strings.Contains(n.Body, "design-doc-creator") {
		t.Errorf("expected agent in body, got %q", n.Body)
	}
}

func TestTaskNotification_Completed(t *testing.T) {
	n, fire := taskNotification(pubsub.TaskCompletion{
		TaskID:   "t2",
		AgentID:  "sprint-executor",
		Status:   "completed",
		NumTurns: 12,
		CostUSD:  0.058,
	})
	if !fire {
		t.Fatal("expected fire=true for completed")
	}
	if !strings.Contains(n.Title, "done") {
		t.Errorf("expected done in title, got %q", n.Title)
	}
	if !strings.Contains(n.Body, "12 turns") {
		t.Errorf("expected turn count, got %q", n.Body)
	}
	if !strings.Contains(n.Body, "$0.0") {
		t.Errorf("expected cost, got %q", n.Body)
	}
}

func TestTaskNotification_Failed(t *testing.T) {
	n, fire := taskNotification(pubsub.TaskCompletion{
		TaskID:   "t3",
		AgentID:  "agent-x",
		Status:   "failed",
		ErrorMsg: "timeout after 600s",
	})
	if !fire {
		t.Fatal("expected fire=true for failed")
	}
	if !strings.Contains(n.Title, "failed") {
		t.Errorf("expected failed in title, got %q", n.Title)
	}
	if !strings.Contains(n.Body, "timeout") {
		t.Errorf("expected error in body, got %q", n.Body)
	}
	if n.Sound != "Basso" {
		t.Errorf("expected Basso sound for failure, got %q", n.Sound)
	}
}

func TestTaskNotification_UnknownStatusSkipped(t *testing.T) {
	_, fire := taskNotification(pubsub.TaskCompletion{Status: "running"})
	if fire {
		t.Error("expected fire=false for unknown status")
	}
}

func TestMessageNotification_PublicFeedback(t *testing.T) {
	n, fire := messageNotification(&messaging.InboxMessage{
		MessageID: "msg_xyz",
		ToInbox:   "public-feedback",
		FromAgent: "mcp-public",
		Title:     "Effect row mismatch confused parser",
		Payload:   "Long body",
	})
	if !fire {
		t.Fatal("expected fire=true")
	}
	if !strings.Contains(n.Title, "External feedback") {
		t.Errorf("expected External feedback title for public-feedback, got %q", n.Title)
	}
	if !strings.Contains(n.Body, "Effect row mismatch") {
		t.Errorf("expected message title in body, got %q", n.Body)
	}
}

func TestMessageNotification_PkgInboxIsExternalFeedback(t *testing.T) {
	cases := []struct {
		name            string
		inbox           string
		wantEventType   string
		wantExternal    bool // 🌐 External feedback shape
		wantInboxInBody bool
	}{
		{"public-feedback", "public-feedback", "public-feedback", true, true},
		{"pkg scoped", "pkg:sunholo/auth", "public-feedback", true, true},
		{"pkg ailang", "pkg:sunholo/ailang", "public-feedback", true, true},
		{"internal user", "user", "message", false, false},
		{"internal controlplane", "controlplane", "message", false, false},
		{"internal agent", "sprint-executor", "message", false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			n, fire := messageNotification(&messaging.InboxMessage{
				MessageID: "m_" + tc.name,
				ToInbox:   tc.inbox,
				FromAgent: "mcp-public",
				Title:     "Effect row mismatch confused parser",
			})
			if !fire {
				t.Fatal("expected fire=true")
			}
			if n.EventType != tc.wantEventType {
				t.Errorf("EventType = %q, want %q", n.EventType, tc.wantEventType)
			}
			gotExternal := strings.Contains(n.Title, "External feedback")
			if gotExternal != tc.wantExternal {
				t.Errorf("external-feedback shape = %v, want %v (title=%q)", gotExternal, tc.wantExternal, n.Title)
			}
			if tc.wantInboxInBody && !strings.Contains(n.Body, tc.inbox) {
				t.Errorf("expected inbox %q visible in body, got %q", tc.inbox, n.Body)
			}
		})
	}
}

func TestIsExternalFeedbackInbox(t *testing.T) {
	cases := map[string]bool{
		"public-feedback":    true,
		"pkg:sunholo/auth":   true,
		"pkg:sunholo/ailang": true,
		"pkg:":               true,
		"user":               false,
		"controlplane":       false,
		"sprint-executor":    false,
		"feedback":           false, // note: not the literal "public-feedback"
		"":                   false,
	}
	for inbox, want := range cases {
		if got := isExternalFeedbackInbox(inbox); got != want {
			t.Errorf("isExternalFeedbackInbox(%q) = %v, want %v", inbox, got, want)
		}
	}
}

func TestMessageNotification_GenericInbox(t *testing.T) {
	n, fire := messageNotification(&messaging.InboxMessage{
		MessageID: "msg_abc",
		ToInbox:   "user",
		FromAgent: "sprint-executor",
		Title:     "Sprint M-FOO complete",
	})
	if !fire {
		t.Fatal("expected fire=true")
	}
	if !strings.Contains(n.Title, "Message from") {
		t.Errorf("expected generic message title, got %q", n.Title)
	}
	if !strings.Contains(n.Title, "sprint-executor") {
		t.Errorf("expected from_agent in title, got %q", n.Title)
	}
}

func TestMessageNotification_BodyTruncated(t *testing.T) {
	long := strings.Repeat("a", 500)
	n, fire := messageNotification(&messaging.InboxMessage{
		MessageID: "m1",
		ToInbox:   "user",
		FromAgent: "x",
		Title:     "t",
		Payload:   long,
	})
	if !fire {
		t.Fatal("expected fire=true")
	}
	if len(n.Body) > 220 {
		t.Errorf("expected body truncated to <=220 chars, got %d", len(n.Body))
	}
}

func TestApplyExcludes(t *testing.T) {
	excludes := []string{"Package published", "📦"}
	n := notify.Notification{Title: "📦 Package published", Body: "x"}
	if !shouldExclude(n, excludes) {
		t.Error("expected exclude match on emoji")
	}
	other := notify.Notification{Title: "✅ Task done", Body: "x"}
	if shouldExclude(other, excludes) {
		t.Error("did not expect exclude match")
	}
}

func TestDedupKey(t *testing.T) {
	if k := taskDedupKey("t1", "completed"); k != "task:t1:completed" {
		t.Errorf("got %q", k)
	}
	if k := messageDedupKey("m1"); k != "msg:m1" {
		t.Errorf("got %q", k)
	}
}
