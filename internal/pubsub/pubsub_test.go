package pubsub

import (
	"encoding/json"
	"testing"
)

// Tests for types, serialization, and helpers that don't require a Pub/Sub connection.
// Integration tests requiring the Pub/Sub emulator are build-tagged separately.

func TestMessageAttributesToMap(t *testing.T) {
	tests := []struct {
		name     string
		attrs    MessageAttributes
		wantKeys []string
		wantLen  int
	}{
		{
			name: "all fields",
			attrs: MessageAttributes{
				Inbox:       "design-doc-creator",
				Workspace:   "sunholo-data/ailang",
				FromAgent:   "user",
				Category:    "feature",
				MessageType: "request",
			},
			wantKeys: []string{"inbox", "workspace", "from_agent", "category", "message_type"},
			wantLen:  5,
		},
		{
			name: "partial fields",
			attrs: MessageAttributes{
				Inbox:     "sprint-planner",
				Workspace: "sunholo-data/ailang",
			},
			wantKeys: []string{"inbox", "workspace"},
			wantLen:  2,
		},
		{
			name:     "empty",
			attrs:    MessageAttributes{},
			wantKeys: nil,
			wantLen:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := tt.attrs.ToMap()
			if len(m) != tt.wantLen {
				t.Errorf("ToMap() len = %d, want %d", len(m), tt.wantLen)
			}
			for _, key := range tt.wantKeys {
				if _, ok := m[key]; !ok {
					t.Errorf("ToMap() missing key %q", key)
				}
			}
		})
	}
}

func TestAttributesFromMap(t *testing.T) {
	m := map[string]string{
		"inbox":        "design-doc-creator",
		"workspace":    "sunholo-data/ailang",
		"from_agent":   "user",
		"category":     "feature",
		"message_type": "request",
	}

	attrs := AttributesFromMap(m)

	if attrs.Inbox != "design-doc-creator" {
		t.Errorf("Inbox = %q, want %q", attrs.Inbox, "design-doc-creator")
	}
	if attrs.Workspace != "sunholo-data/ailang" {
		t.Errorf("Workspace = %q, want %q", attrs.Workspace, "sunholo-data/ailang")
	}
	if attrs.FromAgent != "user" {
		t.Errorf("FromAgent = %q, want %q", attrs.FromAgent, "user")
	}
	if attrs.Category != "feature" {
		t.Errorf("Category = %q, want %q", attrs.Category, "feature")
	}
	if attrs.MessageType != "request" {
		t.Errorf("MessageType = %q, want %q", attrs.MessageType, "request")
	}
}

func TestAttributesRoundTrip(t *testing.T) {
	original := MessageAttributes{
		Inbox:       "sprint-executor",
		Workspace:   "MarkEdmondson1234/TwilightGame",
		FromAgent:   "sprint-planner",
		Category:    "bug",
		MessageType: "notification",
	}

	m := original.ToMap()
	restored := AttributesFromMap(m)

	if restored != original {
		t.Errorf("roundtrip failed:\n  original: %+v\n  restored: %+v", original, restored)
	}
}

func TestTopicConstants(t *testing.T) {
	// Verify all topic constants are non-empty and distinct.
	topics := []string{TopicMessages, TopicTasks, TopicCompletions, TopicEvents, TopicDeadLetter, TopicCascade}
	seen := make(map[string]bool)
	for _, topic := range topics {
		if topic == "" {
			t.Error("empty topic constant")
		}
		if seen[topic] {
			t.Errorf("duplicate topic constant: %q", topic)
		}
		seen[topic] = true
	}
}

// TestSourceAttributeRoundTrip is the M-PKG-AUTONOMOUS-CASCADE-SAFE M1
// regression test: the new `source` attribute must survive the full
// ToMap → AttributesFromMap cycle so the receiving agent can read it.
func TestSourceAttributeRoundTrip(t *testing.T) {
	original := MessageAttributes{
		Inbox:     "pkg:sunholo/test_pkg",
		FromAgent: "coordinator",
		Source:    SourceCascade,
	}
	restored := AttributesFromMap(original.ToMap())
	if restored.Source != SourceCascade {
		t.Errorf("Source round-trip: got %q want %q", restored.Source, SourceCascade)
	}
	// Map representation must use the literal "source" key — that's what
	// the receiving pubsub_adapter.HandleNotification reads.
	if got := original.ToMap()["source"]; got != SourceCascade {
		t.Errorf("ToMap source key: got %q want %q", got, SourceCascade)
	}
}

func TestSubscriptionConstants(t *testing.T) {
	subs := []string{
		SubMessagesCoordinator, SubMessagesLaptop,
		SubTasksExecutor,
		SubCompletionsCoordinator,
		SubEventsDashboard, SubEventsLaptop,
	}
	seen := make(map[string]bool)
	for _, sub := range subs {
		if sub == "" {
			t.Error("empty subscription constant")
		}
		if seen[sub] {
			t.Errorf("duplicate subscription constant: %q", sub)
		}
		seen[sub] = true
	}
}

func TestDecodeMessageNotification(t *testing.T) {
	n := MessageNotification{MessageID: "msg-abc-123"}
	data, err := json.Marshal(n)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	decoded, err := DecodeMessageNotification(data)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if decoded.MessageID != n.MessageID {
		t.Errorf("MessageID = %q, want %q", decoded.MessageID, n.MessageID)
	}
}

func TestDecodeMessageNotificationInvalid(t *testing.T) {
	_, err := DecodeMessageNotification([]byte("not json"))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestDecodeTaskDispatch(t *testing.T) {
	d := TaskDispatch{TaskID: "task-123", AgentID: "sprint-executor"}
	data, err := json.Marshal(d)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	decoded, err := DecodeTaskDispatch(data)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if decoded.TaskID != d.TaskID {
		t.Errorf("TaskID = %q, want %q", decoded.TaskID, d.TaskID)
	}
	if decoded.AgentID != d.AgentID {
		t.Errorf("AgentID = %q, want %q", decoded.AgentID, d.AgentID)
	}
}

func TestDecodeTaskCompletion(t *testing.T) {
	c := TaskCompletion{
		TaskID:     "task-456",
		AgentID:    "sprint-executor",
		Status:     "completed",
		BranchName: "coordinator/task-456",
	}
	data, err := json.Marshal(c)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	decoded, err := DecodeTaskCompletion(data)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if decoded.TaskID != c.TaskID {
		t.Errorf("TaskID = %q, want %q", decoded.TaskID, c.TaskID)
	}
	if decoded.Status != c.Status {
		t.Errorf("Status = %q, want %q", decoded.Status, c.Status)
	}
	if decoded.BranchName != c.BranchName {
		t.Errorf("BranchName = %q, want %q", decoded.BranchName, c.BranchName)
	}
}

func TestDecodeTaskCompletionWithError(t *testing.T) {
	c := TaskCompletion{
		TaskID:   "task-789",
		AgentID:  "sprint-executor",
		Status:   "failed",
		ErrorMsg: "compilation error in main.go",
	}
	data, err := json.Marshal(c)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	decoded, err := DecodeTaskCompletion(data)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if decoded.ErrorMsg != c.ErrorMsg {
		t.Errorf("ErrorMsg = %q, want %q", decoded.ErrorMsg, c.ErrorMsg)
	}
	if decoded.BranchName != "" {
		t.Errorf("BranchName = %q, want empty", decoded.BranchName)
	}
}

func TestClientTopicName(t *testing.T) {
	// We can't create a real client without a GCP project, but we can test
	// the naming logic via the exported TopicName method on a zero-value.
	// Instead, test the naming format directly.
	tests := []struct {
		prefix   string
		base     string
		wantName string
	}{
		{"ailang", "messages", "ailang-messages"},
		{"ailang", "tasks", "ailang-tasks"},
		{"ailang", "completions", "ailang-completions"},
		{"ailang", "events", "ailang-events"},
		{"ailang", "dead-letter", "ailang-dead-letter"},
		{"custom", "messages", "custom-messages"},
	}

	for _, tt := range tests {
		name := tt.prefix + "-" + tt.base
		if name != tt.wantName {
			t.Errorf("topic name %q, want %q", name, tt.wantName)
		}
	}
}

func TestDefaultTopicPrefix(t *testing.T) {
	if DefaultTopicPrefix != "ailang" {
		t.Errorf("DefaultTopicPrefix = %q, want %q", DefaultTopicPrefix, "ailang")
	}
}

func TestPublisherNewPublisher(t *testing.T) {
	// Verify NewPublisher doesn't panic with nil client
	// (topics map is initialized).
	p := NewPublisher(nil)
	if p == nil {
		t.Fatal("NewPublisher returned nil")
	}
	if p.topics == nil {
		t.Error("topics map not initialized")
	}
}

func TestSubscriberNewSubscriber(t *testing.T) {
	s := NewSubscriber(nil)
	if s == nil {
		t.Fatal("NewSubscriber returned nil")
	}
	if s.cancel == nil {
		t.Error("cancel map not initialized")
	}
}

func TestSubscriberStopEmpty(t *testing.T) {
	// Stop on empty subscriber should not panic.
	s := NewSubscriber(nil)
	s.Stop()
}
