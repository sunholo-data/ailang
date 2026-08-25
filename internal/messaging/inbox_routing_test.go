package messaging

import "testing"

// TestNormalizeInboxRouting guards the defect that made 36 prod / 787 dev messages
// permanently unreadable: a message stored with ToInbox == "" is retained and billed
// but matches no --inbox query, so task-failure notifications — the traffic whose
// entire purpose is to be seen — vanished silently.
func TestNormalizeInboxRouting(t *testing.T) {
	tests := []struct {
		name        string
		in          string
		wantInbox   string
		wantChanged bool
	}{
		{"empty inbox is rerouted", "", UnroutedInbox, true},
		{"addressed message is untouched", "public-feedback", "public-feedback", false},
		{"agent inbox is untouched", "sprint-executor", "sprint-executor", false},
		{"package inbox is untouched", "pkg:sunholo/ailang-parse", "pkg:sunholo/ailang-parse", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := &InboxMessage{ToInbox: tt.in}
			changed := NormalizeInboxRouting(msg)

			if changed != tt.wantChanged {
				t.Errorf("changed = %v, want %v", changed, tt.wantChanged)
			}
			if msg.ToInbox != tt.wantInbox {
				t.Errorf("ToInbox = %q, want %q", msg.ToInbox, tt.wantInbox)
			}
		})
	}
}

// TestNormalizeInboxRoutingNilSafe asserts the guard cannot itself panic on the
// insert path.
func TestNormalizeInboxRoutingNilSafe(t *testing.T) {
	if NormalizeInboxRouting(nil) {
		t.Error("nil message must report no change, not claim a reroute")
	}
}

// TestUnroutedInboxIsListable asserts the reroute target is a plain inbox name.
// If it were empty or contained a filter-breaking character the reroute would
// recreate the very invisibility it exists to prevent.
func TestUnroutedInboxIsListable(t *testing.T) {
	if UnroutedInbox == "" {
		t.Fatal("UnroutedInbox must be a real inbox name; empty recreates the unreachable-message bug")
	}
	msg := &InboxMessage{}
	NormalizeInboxRouting(msg)
	if msg.ToInbox == "" {
		t.Fatal("reroute left ToInbox empty")
	}
}
