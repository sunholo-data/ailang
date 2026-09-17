package coordinator

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/messaging"
)

// An approval nobody hears about waits for its age to be noticed.
//
// task-c063b6d2, measured 2026-09-17: its agent `daneel-design-ailang` was
// deleted in the 12 Sept revert, so the notice was filed to that dead inbox and
// the approval sat SEVEN DAYS on a design doc already merged by PR #1138.
func TestApprovalNoticeInbox(t *testing.T) {
	reg := NewAgentRegistry()
	if err := reg.Register(&AgentConfig{ID: "pkg-sunholo-auth", Inbox: "pkg:sunholo/auth", Workspace: "w"}); err != nil {
		t.Fatalf("register: %v", err)
	}
	reg.SetTriageOnlyInboxes([]string{"user", HumanApprovalInbox})

	for _, tc := range []struct {
		name      string
		agentID   string
		wantInbox string
		wantFrom  string
	}{
		// The normal case: the agent exists, its own inbox is the right place,
		// and nothing is redirected.
		{"live agent keeps its own inbox", "pkg-sunholo-auth", "pkg:sunholo/auth", ""},
		// The measured case.
		{"deleted agent goes to the human inbox", "daneel-design-ailang", HumanApprovalInbox, "daneel-design-ailang"},
		// A task with no agent at all still needs a person.
		{"no agent at all", "", HumanApprovalInbox, ""},
	} {
		inbox, from := ApprovalNoticeInbox(reg, tc.agentID)
		if inbox != tc.wantInbox || from != tc.wantFrom {
			t.Errorf("%s: got (%q, %q), want (%q, %q)", tc.name, inbox, from, tc.wantInbox, tc.wantFrom)
		}
	}

	// A DECLARED triage inbox is read by a person on purpose — redirecting it
	// would be the opposite mistake, moving a notice away from its reader.
	reg2 := NewAgentRegistry()
	reg2.SetTriageOnlyInboxes([]string{"daneel", HumanApprovalInbox})
	if inbox, from := ApprovalNoticeInbox(reg2, "daneel"); inbox != "daneel" || from != "" {
		t.Errorf("a declared triage inbox must be left alone, got (%q, %q)", inbox, from)
	}

	// No registry: judge nothing, but do not file into the dark.
	if inbox, from := ApprovalNoticeInbox(nil, "whoever"); inbox != HumanApprovalInbox || from != "whoever" {
		t.Errorf("with no registry got (%q, %q), want the human inbox and the original named", inbox, from)
	}
}

// The redirect must be visible on the card, or the next one reads as a notice
// from nowhere.
func TestNotifyApproval_RedirectIsStatedOnTheCard(t *testing.T) {
	reg := NewAgentRegistry()
	reg.SetTriageOnlyInboxes([]string{HumanApprovalInbox})
	store, err := messaging.OpenStore(filepath.Join(t.TempDir(), "collaboration.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	f := &finalizer{
		deps: &FinalizeDeps{MsgStore: store, AgentRegistry: reg},
		in:   FinalizeInput{Task: &TaskRecord{ID: "task-orphan1", AgentID: "deleted-agent", Title: "a design doc"}},
	}
	f.notifyApproval(context.Background(), "Agent completed work on: a design doc")

	msgs, err := store.ListInboxMessages(messaging.InboxListOptions{IncludeRead: true})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected one notice, got %d", len(msgs))
	}
	m := msgs[0]
	if m.ToInbox != HumanApprovalInbox {
		t.Errorf("notice went to %q, want %q", m.ToInbox, HumanApprovalInbox)
	}
	if !strings.Contains(m.Payload, "deleted-agent") {
		t.Errorf("the card must name the inbox it was redirected from: %q", m.Payload)
	}
	if !strings.Contains(m.Payload, "coordinator approvals") {
		t.Errorf("the card must say the approval is in the queue regardless: %q", m.Payload)
	}
}
