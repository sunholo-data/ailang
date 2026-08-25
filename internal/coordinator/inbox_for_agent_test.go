package coordinator

import "testing"

// TestInboxForAgent guards the ID-vs-inbox conflation. Agent ID and inbox name
// coincide for simple agents ("sprint-executor"), and that coincidence hid the bug:
// package agents register as ID "pkg-sunholo-auth" watching inbox "pkg:sunholo/auth",
// so completion handlers addressing task.AgentID posted replies to a literal inbox
// "pkg-sunholo-auth" that nothing watches. Prod held BOTH spellings with unread
// messages in each on 2026-08-25.
func TestInboxForAgent(t *testing.T) {
	registry := NewAgentRegistry()

	if err := registry.Register(&AgentConfig{
		ID: "pkg-sunholo-auth", Inbox: "pkg:sunholo/auth", Workspace: "/tmp/t",
	}); err != nil {
		t.Fatalf("register package agent: %v", err)
	}
	if err := registry.Register(&AgentConfig{
		ID: "sprint-executor", Inbox: "sprint-executor", Workspace: "/tmp/t",
	}); err != nil {
		t.Fatalf("register simple agent: %v", err)
	}

	t.Run("package agent resolves to its colon-form inbox", func(t *testing.T) {
		got, ok := registry.InboxForAgent("pkg-sunholo-auth")
		if !ok {
			t.Fatal("registered package agent must resolve")
		}
		if got != "pkg:sunholo/auth" {
			t.Errorf("inbox = %q, want %q — the dash form is the agent ID, not an inbox", got, "pkg:sunholo/auth")
		}
	})

	t.Run("simple agent where ID equals inbox", func(t *testing.T) {
		got, ok := registry.InboxForAgent("sprint-executor")
		if !ok || got != "sprint-executor" {
			t.Errorf("got (%q, %v), want (\"sprint-executor\", true)", got, ok)
		}
	})

	t.Run("unknown agent reports false so callers keep their fallback", func(t *testing.T) {
		if _, ok := registry.InboxForAgent("no-such-agent"); ok {
			t.Error("unknown agent must report false, not invent an inbox")
		}
	})

	t.Run("nil registry is safe", func(t *testing.T) {
		var nilRegistry *AgentRegistry
		if _, ok := nilRegistry.InboxForAgent("anything"); ok {
			t.Error("nil registry must report false")
		}
	})
}
