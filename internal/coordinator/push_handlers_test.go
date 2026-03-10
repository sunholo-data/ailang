package coordinator

import (
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/sunholo/ailang/internal/pubsub"
)

// makePushBody creates a Pub/Sub push envelope JSON string.
func makePushBody(t *testing.T, payload interface{}, attrs map[string]string) string {
	t.Helper()
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	envelope := pubsub.PushEnvelope{
		Message: pubsub.PushMessage{
			Data:       base64.StdEncoding.EncodeToString(data),
			MessageID:  "test-msg-001",
			Attributes: attrs,
		},
		Subscription: "projects/test/subscriptions/test-sub",
	}
	body, err := json.Marshal(envelope)
	if err != nil {
		t.Fatalf("marshal envelope: %v", err)
	}
	return string(body)
}

func TestHandlePushMessage_Valid(t *testing.T) {
	logger := log.New(os.Stderr, "test: ", 0)

	adapter := NewPubSubInboxAdapter(nil, "test-sub", "coordinator", nil, logger)

	d := &Daemon{
		logger:            logger,
		cloudInboxAdapter: adapter,
	}

	notification := pubsub.MessageNotification{MessageID: "msg-abc-123"}
	body := makePushBody(t, notification, map[string]string{
		"inbox":      "design-doc-creator",
		"from_agent": "user",
		"category":   "feature",
	})

	req := httptest.NewRequest(http.MethodPost, "/pubsub/push", strings.NewReader(body))
	w := httptest.NewRecorder()

	d.handlePushMessage(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	// Verify message was buffered
	msgs, err := adapter.ListUnread()
	if err != nil {
		t.Fatalf("ListUnread: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("buffered %d messages, want 1", len(msgs))
	}
	if msgs[0].ID != "msg-abc-123" {
		t.Errorf("msg.ID = %q, want %q", msgs[0].ID, "msg-abc-123")
	}
	if msgs[0].Inbox != "design-doc-creator" {
		t.Errorf("msg.Inbox = %q, want %q", msgs[0].Inbox, "design-doc-creator")
	}
	if msgs[0].From != "user" {
		t.Errorf("msg.From = %q, want %q", msgs[0].From, "user")
	}
}

func TestHandlePushMessage_MalformedJSON(t *testing.T) {
	logger := log.New(os.Stderr, "test: ", 0)
	d := &Daemon{
		logger:            logger,
		cloudInboxAdapter: NewPubSubInboxAdapter(nil, "test-sub", "coordinator", nil, logger),
	}

	req := httptest.NewRequest(http.MethodPost, "/pubsub/push", strings.NewReader("not json"))
	w := httptest.NewRecorder()

	d.handlePushMessage(w, req)

	// Malformed messages should ack (200) to prevent infinite retry
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d (ack malformed)", w.Code, http.StatusOK)
	}
}

func TestHandlePushMessage_MethodNotAllowed(t *testing.T) {
	d := &Daemon{logger: log.New(os.Stderr, "test: ", 0)}

	req := httptest.NewRequest(http.MethodGet, "/pubsub/push", nil)
	w := httptest.NewRecorder()

	d.handlePushMessage(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandlePushMessage_NoAdapter(t *testing.T) {
	d := &Daemon{
		logger:            log.New(os.Stderr, "test: ", 0),
		cloudInboxAdapter: nil, // No adapter configured
	}

	notification := pubsub.MessageNotification{MessageID: "msg-orphan"}
	body := makePushBody(t, notification, nil)

	req := httptest.NewRequest(http.MethodPost, "/pubsub/push", strings.NewReader(body))
	w := httptest.NewRecorder()

	d.handlePushMessage(w, req)

	// Should ack even without adapter
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHandlePushCompletion_MethodNotAllowed(t *testing.T) {
	d := &Daemon{logger: log.New(os.Stderr, "test: ", 0)}

	req := httptest.NewRequest(http.MethodGet, "/pubsub/completions", nil)
	w := httptest.NewRecorder()

	d.handlePushCompletion(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandlePushCompletion_NoHandler(t *testing.T) {
	d := &Daemon{
		logger:            log.New(os.Stderr, "test: ", 0),
		completionHandler: nil,
	}

	completion := pubsub.TaskCompletion{TaskID: "task-001", Status: "completed"}
	body := makePushBody(t, completion, nil)

	req := httptest.NewRequest(http.MethodPost, "/pubsub/completions", strings.NewReader(body))
	w := httptest.NewRecorder()

	d.handlePushCompletion(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHandlePushCompletion_MalformedJSON(t *testing.T) {
	d := &Daemon{
		logger:            log.New(os.Stderr, "test: ", 0),
		completionHandler: NewCompletionHandler(nil, nil, nil, nil, log.New(os.Stderr, "test: ", 0)),
	}

	req := httptest.NewRequest(http.MethodPost, "/pubsub/completions", strings.NewReader("{bad"))
	w := httptest.NewRecorder()

	d.handlePushCompletion(w, req)

	// Malformed → ack
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}
