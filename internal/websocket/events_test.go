package websocket

import (
	"encoding/json"
	"testing"
	"time"
)

func TestNewEvent(t *testing.T) {
	data := &SubscribeEvent{
		ThreadID: "thread_123",
		FromSeq:  10,
	}

	event, err := NewEvent(EventTypeSubscribe, data)
	if err != nil {
		t.Fatalf("NewEvent failed: %v", err)
	}

	if event.Type != EventTypeSubscribe {
		t.Errorf("Expected type %s, got %s", EventTypeSubscribe, event.Type)
	}
	if event.Timestamp == 0 {
		t.Error("Expected timestamp to be set")
	}
	if len(event.Data) == 0 {
		t.Error("Expected data to be set")
	}
}

func TestParseEvent(t *testing.T) {
	original := &Event{
		Type:      EventTypeAck,
		Timestamp: time.Now().UnixMilli(),
		Data:      json.RawMessage(`{"thread_id":"thread_123","ack_seq":5}`),
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal event: %v", err)
	}

	parsed, err := ParseEvent(data)
	if err != nil {
		t.Fatalf("ParseEvent failed: %v", err)
	}

	if parsed.Type != original.Type {
		t.Errorf("Expected type %s, got %s", original.Type, parsed.Type)
	}
	if parsed.Timestamp != original.Timestamp {
		t.Errorf("Expected timestamp %d, got %d", original.Timestamp, parsed.Timestamp)
	}
}

func TestParseSubscribe(t *testing.T) {
	sub := &SubscribeEvent{
		ThreadID: "thread_456",
		FromSeq:  20,
	}

	event, err := NewEvent(EventTypeSubscribe, sub)
	if err != nil {
		t.Fatalf("NewEvent failed: %v", err)
	}

	parsed, err := ParseSubscribe(event)
	if err != nil {
		t.Fatalf("ParseSubscribe failed: %v", err)
	}

	if parsed.ThreadID != sub.ThreadID {
		t.Errorf("Expected thread_id %s, got %s", sub.ThreadID, parsed.ThreadID)
	}
	if parsed.FromSeq != sub.FromSeq {
		t.Errorf("Expected from_seq %d, got %d", sub.FromSeq, parsed.FromSeq)
	}
}

func TestParseSubscribeWrongType(t *testing.T) {
	event, _ := NewEvent(EventTypeAck, &AckEvent{ThreadID: "thread_123", AckSeq: 1})

	_, err := ParseSubscribe(event)
	if err == nil {
		t.Error("Expected error when parsing wrong event type")
	}
}

func TestParseAck(t *testing.T) {
	ack := &AckEvent{
		ThreadID: "thread_789",
		AckSeq:   15,
	}

	event, err := NewEvent(EventTypeAck, ack)
	if err != nil {
		t.Fatalf("NewEvent failed: %v", err)
	}

	parsed, err := ParseAck(event)
	if err != nil {
		t.Fatalf("ParseAck failed: %v", err)
	}

	if parsed.ThreadID != ack.ThreadID {
		t.Errorf("Expected thread_id %s, got %s", ack.ThreadID, parsed.ThreadID)
	}
	if parsed.AckSeq != ack.AckSeq {
		t.Errorf("Expected ack_seq %d, got %d", ack.AckSeq, parsed.AckSeq)
	}
}

func TestParseAckWrongType(t *testing.T) {
	event, _ := NewEvent(EventTypeSubscribe, &SubscribeEvent{ThreadID: "thread_123", FromSeq: 0})

	_, err := ParseAck(event)
	if err == nil {
		t.Error("Expected error when parsing wrong event type")
	}
}

func TestToJSON(t *testing.T) {
	event := &Event{
		Type:      EventTypePing,
		Timestamp: 1234567890,
	}

	data, err := event.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	// Verify it's valid JSON
	var parsed Event
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if parsed.Type != event.Type {
		t.Errorf("Expected type %s, got %s", event.Type, parsed.Type)
	}
}

func TestNewErrorEvent(t *testing.T) {
	event, err := NewErrorEvent("TEST_ERROR", "This is a test error")
	if err != nil {
		t.Fatalf("NewErrorEvent failed: %v", err)
	}

	if event.Type != EventTypeError {
		t.Errorf("Expected type %s, got %s", EventTypeError, event.Type)
	}

	var errorData ErrorEvent
	if err := json.Unmarshal(event.Data, &errorData); err != nil {
		t.Fatalf("Failed to unmarshal error data: %v", err)
	}

	if errorData.Code != "TEST_ERROR" {
		t.Errorf("Expected code 'TEST_ERROR', got %s", errorData.Code)
	}
	if errorData.Message != "This is a test error" {
		t.Errorf("Expected message 'This is a test error', got %s", errorData.Message)
	}
}

func TestNewMessageEvent(t *testing.T) {
	msg := &MessageEvent{
		ID:         "msg_123",
		ThreadID:   "thread_456",
		MessageSeq: 5,
		CreatedAt:  time.Now().UnixMilli(),
		FromType:   "ailang_instance",
		FromID:     "agent1",
		ToType:     "human",
		ToID:       "user",
		Kind:       "status",
		Content:    "Test message",
	}

	event, err := NewMessageEvent(msg)
	if err != nil {
		t.Fatalf("NewMessageEvent failed: %v", err)
	}

	if event.Type != EventTypeMessage {
		t.Errorf("Expected type %s, got %s", EventTypeMessage, event.Type)
	}

	var parsedMsg MessageEvent
	if err := json.Unmarshal(event.Data, &parsedMsg); err != nil {
		t.Fatalf("Failed to unmarshal message data: %v", err)
	}

	if parsedMsg.ID != msg.ID {
		t.Errorf("Expected ID %s, got %s", msg.ID, parsedMsg.ID)
	}
	if parsedMsg.MessageSeq != msg.MessageSeq {
		t.Errorf("Expected message_seq %d, got %d", msg.MessageSeq, parsedMsg.MessageSeq)
	}
}

func TestNewBatchEvent(t *testing.T) {
	messages := []MessageEvent{
		{ID: "msg_1", ThreadID: "thread_123", MessageSeq: 1, Content: "Message 1"},
		{ID: "msg_2", ThreadID: "thread_123", MessageSeq: 2, Content: "Message 2"},
	}

	event, err := NewBatchEvent("thread_123", messages, true)
	if err != nil {
		t.Fatalf("NewBatchEvent failed: %v", err)
	}

	if event.Type != EventTypeBatch {
		t.Errorf("Expected type %s, got %s", EventTypeBatch, event.Type)
	}

	var batch BatchEvent
	if err := json.Unmarshal(event.Data, &batch); err != nil {
		t.Fatalf("Failed to unmarshal batch data: %v", err)
	}

	if batch.ThreadID != "thread_123" {
		t.Errorf("Expected thread_id 'thread_123', got %s", batch.ThreadID)
	}
	if len(batch.Messages) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(batch.Messages))
	}
	if !batch.HasMore {
		t.Error("Expected has_more to be true")
	}
}

func TestNewThreadStateEvent(t *testing.T) {
	now := time.Now()
	event, err := NewThreadStateEvent("thread_123", "active", 42, now)
	if err != nil {
		t.Fatalf("NewThreadStateEvent failed: %v", err)
	}

	if event.Type != EventTypeThreadState {
		t.Errorf("Expected type %s, got %s", EventTypeThreadState, event.Type)
	}

	var state ThreadStateEvent
	if err := json.Unmarshal(event.Data, &state); err != nil {
		t.Fatalf("Failed to unmarshal thread state data: %v", err)
	}

	if state.ThreadID != "thread_123" {
		t.Errorf("Expected thread_id 'thread_123', got %s", state.ThreadID)
	}
	if state.Status != "active" {
		t.Errorf("Expected status 'active', got %s", state.Status)
	}
	if state.LastSeq != 42 {
		t.Errorf("Expected last_seq 42, got %d", state.LastSeq)
	}
}

func TestNewPingPongEvents(t *testing.T) {
	ping, err := NewPingEvent()
	if err != nil {
		t.Fatalf("NewPingEvent failed: %v", err)
	}
	if ping.Type != EventTypePing {
		t.Errorf("Expected type %s, got %s", EventTypePing, ping.Type)
	}

	pong, err := NewPongEvent()
	if err != nil {
		t.Fatalf("NewPongEvent failed: %v", err)
	}
	if pong.Type != EventTypePong {
		t.Errorf("Expected type %s, got %s", EventTypePong, pong.Type)
	}
}
