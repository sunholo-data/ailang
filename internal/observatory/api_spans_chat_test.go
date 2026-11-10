package observatory

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// include_chat embeds the first user prompt and assistant reply of a span's
// session. It no longer carries a full_chat_url: that pointed at
// /api/claude-history/db/session/<id>, a dashboard route only the removed UI
// called (m-dashboard-headless-api).
func TestEnrichSpansWithChat_EmbedsPromptWithoutDeadLink(t *testing.T) {
	store := setupTestStore(t)
	backend, err := NewSQLiteBackend(store.DB())
	if err != nil {
		t.Fatalf("NewSQLiteBackend: %v", err)
	}
	now := time.Now().UTC()
	for _, m := range []struct {
		id, role, text string
	}{{"m1", "user", "fix the parser"}, {"m2", "assistant", "done"}} {
		if _, err := store.DB().Exec(
			`INSERT INTO chat_messages (id, session_id, turn_number, role, content_text, timestamp) VALUES (?, ?, 1, ?, ?, ?)`,
			m.id, "sess-chat", m.role, m.text, now); err != nil {
			t.Fatalf("insert chat message: %v", err)
		}
	}

	spans := []*Span{{ID: "sp-1", Attributes: map[string]any{"session.id": "sess-chat"}}}
	out := NewAPI(backend).enrichSpansWithChat(context.Background(), spans)

	cc := out[0].ChatContext
	if cc == nil || cc.UserPrompt != "fix the parser" || cc.AssistantResponse != "done" {
		t.Fatalf("chat context: got %+v", cc)
	}
	raw, _ := json.Marshal(out[0])
	if strings.Contains(string(raw), "full_chat_url") || strings.Contains(string(raw), "/api/claude-history") {
		t.Errorf("span JSON still links to a removed route: %s", raw)
	}
}
