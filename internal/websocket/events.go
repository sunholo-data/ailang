package websocket

import (
	"encoding/json"
	"fmt"
	"time"
)

// EventType represents the type of WebSocket event
type EventType string

const (
	EventTypeSubscribe    EventType = "subscribe"
	EventTypeAck          EventType = "ack"
	EventTypeMessage      EventType = "message"
	EventTypeBatch        EventType = "batch"
	EventTypeError        EventType = "error"
	EventTypePing         EventType = "ping"
	EventTypePong         EventType = "pong"
	EventTypeThreadState  EventType = "thread_state"
	EventTypeTelemetry    EventType = "telemetry"     // Process telemetry updates
	EventTypeInboxMessage EventType = "inbox_message" // Async inbox messages
)

// Event represents a WebSocket message envelope
type Event struct {
	Type      EventType       `json:"type"`
	Timestamp int64           `json:"timestamp"` // Unix milliseconds
	Data      json.RawMessage `json:"data,omitempty"`
}

// SubscribeEvent - Client subscribes to a thread
type SubscribeEvent struct {
	ThreadID string `json:"thread_id"`
	FromSeq  int    `json:"from_seq"` // Resume from this sequence (0 = from beginning)
}

// AckEvent - Client acknowledges messages up to a sequence number
type AckEvent struct {
	ThreadID string `json:"thread_id"`
	AckSeq   int    `json:"ack_seq"` // Last message_seq processed
}

// MessageEvent - Server sends a single message to client
type MessageEvent struct {
	ID           string `json:"id"`
	ThreadID     string `json:"thread_id"`
	MessageSeq   int    `json:"message_seq"`
	CreatedAt    int64  `json:"created_at"`
	FromType     string `json:"from_type"`
	FromID       string `json:"from_id"`
	ToType       string `json:"to_type"`
	ToID         string `json:"to_id"`
	Kind         string `json:"kind"`
	Subject      string `json:"subject,omitempty"`
	Content      string `json:"content"`
	MetadataJSON string `json:"metadata_json,omitempty"`
}

// BatchEvent - Server sends multiple messages in a batch
type BatchEvent struct {
	ThreadID string         `json:"thread_id"`
	Messages []MessageEvent `json:"messages"`
	HasMore  bool           `json:"has_more"` // More messages available after this batch
}

// ErrorEvent - Server sends an error to client
type ErrorEvent struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ThreadStateEvent - Server sends thread state updates
type ThreadStateEvent struct {
	ThreadID  string `json:"thread_id"`
	Status    string `json:"status"`
	LastSeq   int    `json:"last_seq"`
	UpdatedAt int64  `json:"updated_at"`
}

// TelemetryEvent - Server sends process telemetry updates
type TelemetryEvent struct {
	InstanceID  string  `json:"instance_id"`
	PID         int     `json:"pid"`
	Turns       int     `json:"turns"`
	TokensIn    int     `json:"tokens_in"`
	TokensOut   int     `json:"tokens_out"`
	Cost        float64 `json:"cost"`
	Status      string  `json:"status"` // running, completed, error
	DurationSec int     `json:"duration_sec"`
}

// InboxMessageEvent - Server sends async inbox message updates
type InboxMessageEvent struct {
	ID            string `json:"id"`
	MessageID     string `json:"message_id"`
	CorrelationID string `json:"correlation_id,omitempty"`
	FromAgent     string `json:"from_agent"`
	ToInbox       string `json:"to_inbox"`
	MessageType   string `json:"message_type"`
	Title         string `json:"title"`
	Payload       string `json:"payload,omitempty"`
	Status        string `json:"status"`
	CreatedAt     int64  `json:"created_at"`
}

// NewEvent creates a new event with timestamp
func NewEvent(eventType EventType, data interface{}) (*Event, error) {
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal event data: %w", err)
	}

	return &Event{
		Type:      eventType,
		Timestamp: time.Now().UnixMilli(),
		Data:      dataBytes,
	}, nil
}

// ParseEvent parses an event from JSON bytes
func ParseEvent(data []byte) (*Event, error) {
	var event Event
	if err := json.Unmarshal(data, &event); err != nil {
		return nil, fmt.Errorf("failed to unmarshal event: %w", err)
	}
	return &event, nil
}

// ParseSubscribe parses a subscribe event
func ParseSubscribe(event *Event) (*SubscribeEvent, error) {
	if event.Type != EventTypeSubscribe {
		return nil, fmt.Errorf("expected subscribe event, got %s", event.Type)
	}

	var sub SubscribeEvent
	if err := json.Unmarshal(event.Data, &sub); err != nil {
		return nil, fmt.Errorf("failed to unmarshal subscribe event: %w", err)
	}
	return &sub, nil
}

// ParseAck parses an ack event
func ParseAck(event *Event) (*AckEvent, error) {
	if event.Type != EventTypeAck {
		return nil, fmt.Errorf("expected ack event, got %s", event.Type)
	}

	var ack AckEvent
	if err := json.Unmarshal(event.Data, &ack); err != nil {
		return nil, fmt.Errorf("failed to unmarshal ack event: %w", err)
	}
	return &ack, nil
}

// ToJSON serializes an event to JSON bytes
func (e *Event) ToJSON() ([]byte, error) {
	return json.Marshal(e)
}

// NewErrorEvent creates an error event
func NewErrorEvent(code, message string) (*Event, error) {
	return NewEvent(EventTypeError, &ErrorEvent{
		Code:    code,
		Message: message,
	})
}

// NewMessageEvent creates a message event from a message
func NewMessageEvent(msg *MessageEvent) (*Event, error) {
	return NewEvent(EventTypeMessage, msg)
}

// NewBatchEvent creates a batch event
func NewBatchEvent(threadID string, messages []MessageEvent, hasMore bool) (*Event, error) {
	return NewEvent(EventTypeBatch, &BatchEvent{
		ThreadID: threadID,
		Messages: messages,
		HasMore:  hasMore,
	})
}

// NewThreadStateEvent creates a thread state event
func NewThreadStateEvent(threadID, status string, lastSeq int, updatedAt time.Time) (*Event, error) {
	return NewEvent(EventTypeThreadState, &ThreadStateEvent{
		ThreadID:  threadID,
		Status:    status,
		LastSeq:   lastSeq,
		UpdatedAt: updatedAt.UnixMilli(),
	})
}

// NewPingEvent creates a ping event
func NewPingEvent() (*Event, error) {
	return &Event{
		Type:      EventTypePing,
		Timestamp: time.Now().UnixMilli(),
	}, nil
}

// NewPongEvent creates a pong event
func NewPongEvent() (*Event, error) {
	return &Event{
		Type:      EventTypePong,
		Timestamp: time.Now().UnixMilli(),
	}, nil
}

// NewTelemetryEvent creates a telemetry event
func NewTelemetryEvent(telem *TelemetryEvent) (*Event, error) {
	return NewEvent(EventTypeTelemetry, telem)
}

// NewInboxMessageEvent creates an inbox message event
func NewInboxMessageEvent(msg *InboxMessageEvent) (*Event, error) {
	return NewEvent(EventTypeInboxMessage, msg)
}
