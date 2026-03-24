package coordinator

import (
	"context"
	"testing"
	"time"

	"github.com/sunholo/ailang/internal/pkg"
)

func TestCollectVersionHistory_Basic(t *testing.T) {
	now := time.Now().UTC()
	completed := now.Add(5 * time.Minute)

	task := &TaskRecord{
		ID:          "task-123",
		MessageID:   "msg-456",
		ThreadID:    "thread-789",
		AgentID:     "pkg-sunholo-auth",
		Status:      TaskStatusCompleted,
		CompletedAt: &completed,
	}

	history := CollectVersionHistory(
		context.Background(),
		nil, // no msgStore — skip message/approval lookups
		nil, // no taskStore — skip event lookups
		task,
		"sunholo/auth", "0.2.0", "0.1.0",
	)

	if history.Schema != pkg.VersionHistorySchema {
		t.Errorf("expected schema %q, got %q", pkg.VersionHistorySchema, history.Schema)
	}
	if history.Package != "sunholo/auth" {
		t.Errorf("expected package 'sunholo/auth', got %q", history.Package)
	}
	if history.Version != "0.2.0" {
		t.Errorf("expected version '0.2.0', got %q", history.Version)
	}
	if history.Previous != "0.1.0" {
		t.Errorf("expected previous '0.1.0', got %q", history.Previous)
	}

	// Should have at least the completion entry
	if len(history.Messages) < 1 {
		t.Fatalf("expected at least 1 history entry, got %d", len(history.Messages))
	}

	lastEntry := history.Messages[len(history.Messages)-1]
	if lastEntry.Kind != "task-completed" {
		t.Errorf("expected last entry kind 'task-completed', got %q", lastEntry.Kind)
	}
	if lastEntry.From != "pkg-sunholo-auth" {
		t.Errorf("expected from 'pkg-sunholo-auth', got %q", lastEntry.From)
	}
}

func TestTaskEventToHistoryEntry_StatusEvent(t *testing.T) {
	evt := &TaskEventRecord{
		StreamType: "status",
		Status:     "running",
		Text:       "Agent started execution",
		CreatedAt:  time.Now().UTC(),
	}
	entry := taskEventToHistoryEntry(evt)
	if entry == nil {
		t.Fatal("expected non-nil entry for status event")
	}
	if entry.Kind != "status" {
		t.Errorf("expected kind 'status', got %q", entry.Kind)
	}
	if entry.Status != "running" {
		t.Errorf("expected status 'running', got %q", entry.Status)
	}
}

func TestTaskEventToHistoryEntry_ErrorEvent(t *testing.T) {
	evt := &TaskEventRecord{
		StreamType: "error",
		ErrorMsg:   "compilation failed: type mismatch",
		CreatedAt:  time.Now().UTC(),
	}
	entry := taskEventToHistoryEntry(evt)
	if entry == nil {
		t.Fatal("expected non-nil entry for error event")
	}
	if entry.Kind != "error" {
		t.Errorf("expected kind 'error', got %q", entry.Kind)
	}
	if entry.Status != "failed" {
		t.Errorf("expected status 'failed', got %q", entry.Status)
	}
}

func TestTaskEventToHistoryEntry_SkipsTextEvents(t *testing.T) {
	// Text and tool events are too granular for version history
	for _, streamType := range []string{"text", "tool_use", "tool_result"} {
		evt := &TaskEventRecord{
			StreamType: streamType,
			CreatedAt:  time.Now().UTC(),
		}
		if entry := taskEventToHistoryEntry(evt); entry != nil {
			t.Errorf("expected nil for %q event, got %+v", streamType, entry)
		}
	}
}

func TestTruncateForHistory(t *testing.T) {
	short := "hello"
	if truncateForHistory(short, 10) != "hello" {
		t.Error("short string should not be truncated")
	}

	long := "this is a very long string that exceeds the limit"
	result := truncateForHistory(long, 20)
	if len(result) > 23 { // 20 + "..."
		t.Errorf("expected truncated to ~23 chars, got %d: %q", len(result), result)
	}
}
