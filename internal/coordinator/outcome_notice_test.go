package coordinator

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/messaging"
)

// TestIsOutcomeNotice pins which kinds are reports ABOUT work rather than
// requests FOR work. Getting this wrong in either direction is expensive:
// admitting a notice spawns a task whose own failure notice spawns another (40
// tasks in 3 hours, rig, 2026-08-26), and excluding a real request drops work
// silently.
func TestIsOutcomeNotice(t *testing.T) {
	notices := []string{
		messaging.InboxTypeCompletion,
		// A notice TO an approver, not a request FOR work. The backstop sweep
		// flagged four of these as recoverable in prod on 2026-09-07; dispatching
		// one would ask an agent to perform its own approval request.
		messaging.InboxTypeApprovalRequest,
	}
	for _, kind := range notices {
		if !isOutcomeNotice(&Message{Kind: kind}) {
			t.Errorf("kind %q must be an outcome notice — dispatching it creates work from a report", kind)
		}
	}

	work := []string{
		messaging.InboxTypeNotification,
		messaging.InboxTypeRequest,
		// A handoff IS the request that carries a chain forward. Filtering it
		// would break every pipeline edge.
		messaging.InboxTypeHandoff,
		// User feedback to a package agent is real work.
		messaging.InboxTypeFeedback,
	}
	for _, kind := range work {
		if isOutcomeNotice(&Message{Kind: kind}) {
			t.Errorf("kind %q is a request for work — filtering it drops the work silently", kind)
		}
	}

	if isOutcomeNotice(nil) {
		t.Error("nil must not be an outcome notice")
	}
}
