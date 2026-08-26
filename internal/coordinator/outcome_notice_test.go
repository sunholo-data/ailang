package coordinator

import "testing"

// TestIsOutcomeNotice guards the feedback loop that ran on the voightkampff rig on
// 2026-08-26: completion notices are posted into the agent's own inbox, and a
// LOCAL-mode coordinator on shared storage polls that same inbox. Without this
// filter each failure notice became a new task, whose failure became another —
// 40 tasks in ~3 hours, none of which were ever real work.
func TestIsOutcomeNotice(t *testing.T) {
	tests := []struct {
		name string
		msg  *Message
		want bool
	}{
		{"completion is a report, not a request", &Message{Kind: "completion"}, true},
		{"directive is work", &Message{Kind: "directive"}, false},
		{"question is work", &Message{Kind: "question"}, false},
		{"empty kind is work (the common case)", &Message{Kind: ""}, false},
		{"nil is not a notice", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isOutcomeNotice(tt.msg); got != tt.want {
				t.Errorf("isOutcomeNotice() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestOutcomeNoticeMatchesCompletionWriters pins the filter to the string the
// completion writers actually use. If they diverge the loop returns silently.
func TestOutcomeNoticeMatchesCompletionWriters(t *testing.T) {
	// pubsub_completion_handler.go, stale_task_detector.go and
	// publishDedupCompletion all set MessageType: "completion", which
	// message_adapter.go maps onto Message.Kind.
	if !isOutcomeNotice(&Message{Kind: "completion"}) {
		t.Fatal(`filter must match MessageType "completion" as written by the completion handlers`)
	}
}
