package coordinator

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/messaging"
	"github.com/sunholo-data/ailang/internal/observatory"
)

// newTestDaemonT creates a minimal Daemon for HTTP handler testing.
func newTestDaemonT(t *testing.T) *Daemon {
	t.Helper()
	tmpDir := t.TempDir()
	return &Daemon{
		startedAt: time.Now().Add(-5 * time.Minute),
		config: &Config{
			PIDFile:  tmpDir + "/coordinator.pid",
			StateDir: tmpDir,
		},
	}
}

func TestHandleHealth(t *testing.T) {
	d := newTestDaemonT(t)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	d.handleHealth(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}

	if resp["status"] != "ok" {
		t.Errorf("expected status ok, got %v", resp["status"])
	}
	if resp["component"] != "coordinator" {
		t.Errorf("expected component coordinator, got %v", resp["component"])
	}
	if resp["uptime"] == nil || resp["uptime"] == "" {
		t.Error("expected non-empty uptime")
	}
}

func TestHandleStatus_NoStore(t *testing.T) {
	d := newTestDaemonT(t)

	req := httptest.NewRequest(http.MethodGet, "/status", nil)
	rec := httptest.NewRecorder()

	d.handleStatus(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp Status
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
}

func TestHandleChainsActive_NilBackend(t *testing.T) {
	d := newTestDaemonT(t)

	req := httptest.NewRequest(http.MethodGet, "/chains/active", nil)
	rec := httptest.NewRecorder()

	d.handleChainsActive(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp []interface{}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if len(resp) != 0 {
		t.Errorf("expected empty array, got %d items", len(resp))
	}
}

func TestHandleChainsStats_NilBackend(t *testing.T) {
	d := newTestDaemonT(t)

	req := httptest.NewRequest(http.MethodGet, "/chains/stats?hours=24", nil)
	rec := httptest.NewRecorder()

	d.handleChainsStats(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp observatory.ChainStatusCounts
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp.Total != 0 {
		t.Errorf("expected 0 total, got %d", resp.Total)
	}
}

func TestHandlePending_NilBackend(t *testing.T) {
	d := newTestDaemonT(t)

	req := httptest.NewRequest(http.MethodGet, "/pending", nil)
	rec := httptest.NewRecorder()

	d.handlePending(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp []interface{}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if len(resp) != 0 {
		t.Errorf("expected empty array, got %d items", len(resp))
	}
}

func TestHandleChainsStats_DefaultHours(t *testing.T) {
	d := newTestDaemonT(t)

	req := httptest.NewRequest(http.MethodGet, "/chains/stats", nil)
	rec := httptest.NewRecorder()

	d.handleChainsStats(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestHandleChainsStats_InvalidHours(t *testing.T) {
	d := newTestDaemonT(t)

	req := httptest.NewRequest(http.MethodGet, "/chains/stats?hours=abc", nil)
	rec := httptest.NewRecorder()

	d.handleChainsStats(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

// mockTaskStore implements just GetTaskStats for handler testing.
type mockTaskStore struct {
	Store // embed to satisfy interface
	stats *TaskStats
	err   error
}

func (m *mockTaskStore) GetTaskStats(ctx context.Context) (*TaskStats, error) {
	return m.stats, m.err
}

func TestHandleStatus_WithStore(t *testing.T) {
	d := newTestDaemonT(t)
	d.taskStore = &mockTaskStore{
		stats: &TaskStats{
			CompletedTasks:   10,
			PendingTasks:     2,
			RunningTasks:     1,
			PendingApprovals: 3,
			FailedTasks:      0,
			TotalCost:        1.50,
			TotalTokens:      5000,
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/status", nil)
	rec := httptest.NewRecorder()

	d.handleStatus(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp Status
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}

	if resp.TasksRun != 10 {
		t.Errorf("expected 10 tasks run, got %d", resp.TasksRun)
	}
	if resp.PendingTasks != 2 {
		t.Errorf("expected 2 pending, got %d", resp.PendingTasks)
	}
	if resp.TotalCost != 1.50 {
		t.Errorf("expected cost 1.50, got %f", resp.TotalCost)
	}
}

// M-CLOUD-ENDPOINT-AUTH: API key middleware tests

func TestRequireAPIKey_NoKeyConfigured(t *testing.T) {
	// When COORDINATOR_API_KEY is unset, requests pass through (local mode)
	d := newTestDaemonT(t)
	t.Setenv("COORDINATOR_API_KEY", "")

	handler := d.requireAPIKey(d.handleHealth)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 (no key = open), got %d", rec.Code)
	}
}

func TestRequireAPIKey_ValidToken(t *testing.T) {
	d := newTestDaemonT(t)
	t.Setenv("COORDINATOR_API_KEY", "test-secret-key")

	handler := d.requireAPIKey(d.handleHealth)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("Authorization", "Bearer test-secret-key")
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 with valid token, got %d", rec.Code)
	}
}

func TestRequireAPIKey_InvalidToken(t *testing.T) {
	d := newTestDaemonT(t)
	t.Setenv("COORDINATOR_API_KEY", "test-secret-key")

	handler := d.requireAPIKey(d.handleHealth)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("Authorization", "Bearer wrong-key")
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 with invalid token, got %d", rec.Code)
	}
}

func TestRequireAPIKey_MissingHeader(t *testing.T) {
	d := newTestDaemonT(t)
	t.Setenv("COORDINATOR_API_KEY", "test-secret-key")

	handler := d.requireAPIKey(d.handleHealth)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	// No Authorization header
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 with missing header, got %d", rec.Code)
	}
}

func TestRequireAPIKey_WrongScheme(t *testing.T) {
	d := newTestDaemonT(t)
	t.Setenv("COORDINATOR_API_KEY", "test-secret-key")

	handler := d.requireAPIKey(d.handleHealth)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("Authorization", "Basic test-secret-key")
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 with wrong scheme, got %d", rec.Code)
	}
}

func TestStatusEndpoint_RequiresAPIKey(t *testing.T) {
	// Verify /status is protected when key is configured
	d := newTestDaemonT(t)
	t.Setenv("COORDINATOR_API_KEY", "my-key")

	handler := d.requireAPIKey(d.handleStatus)

	// Without token → 401
	req := httptest.NewRequest(http.MethodGet, "/status", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without token, got %d", rec.Code)
	}

	// With token → 200
	req = httptest.NewRequest(http.MethodGet, "/status", nil)
	req.Header.Set("Authorization", "Bearer my-key")
	rec = httptest.NewRecorder()
	handler(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 with token, got %d", rec.Code)
	}
}

// M-REST-INGESTION: POST /api/messages tests

// newTestDaemonWithMsgStore creates a Daemon with a real SQLite message store.
func newTestDaemonWithMsgStore(t *testing.T) *Daemon {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_collab.db")
	store, err := messaging.OpenStore(dbPath)
	if err != nil {
		t.Fatalf("failed to open test message store: %v", err)
	}
	t.Cleanup(func() { store.Close() })

	return &Daemon{
		startedAt: time.Now().Add(-5 * time.Minute),
		config: &Config{
			PIDFile:  tmpDir + "/coordinator.pid",
			StateDir: tmpDir,
		},
		msgStore: store,
		logger:   log.Default(),
	}
}

func TestHandlePostMessage_Success(t *testing.T) {
	d := newTestDaemonWithMsgStore(t)

	body := `{"inbox":"user","title":"Test message","content":"Hello world","from":"test-client"}`
	req := httptest.NewRequest(http.MethodPost, "/api/messages", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	d.handlePostMessage(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp postMessageResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp.MessageID == "" {
		t.Error("expected non-empty message_id")
	}
	if resp.Inbox != "user" {
		t.Errorf("expected inbox 'user', got %q", resp.Inbox)
	}
	if resp.Status != "unread" {
		t.Errorf("expected status 'unread', got %q", resp.Status)
	}
}

func TestHandlePostMessage_MissingFields(t *testing.T) {
	d := newTestDaemonWithMsgStore(t)

	tests := []struct {
		name string
		body string
		want string
	}{
		{"missing inbox", `{"title":"T","content":"C","from":"F"}`, "inbox"},
		{"missing title", `{"inbox":"I","content":"C","from":"F"}`, "title"},
		{"missing content", `{"inbox":"I","title":"T","from":"F"}`, "content"},
		{"missing from", `{"inbox":"I","title":"T","content":"C"}`, "from"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/messages", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			d.handlePostMessage(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
			}

			var resp map[string]string
			json.NewDecoder(rec.Body).Decode(&resp)
			if !strings.Contains(resp["error"], tt.want) {
				t.Errorf("expected error mentioning %q, got %q", tt.want, resp["error"])
			}
		})
	}
}

func TestHandleMessages_MethodDispatch(t *testing.T) {
	d := newTestDaemonWithMsgStore(t)

	// DELETE → 405
	req := httptest.NewRequest(http.MethodDelete, "/api/messages", nil)
	rec := httptest.NewRecorder()
	d.handleMessages(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("DELETE: expected 405, got %d", rec.Code)
	}

	// GET → dispatches to handleGetMessages (200)
	req = httptest.NewRequest(http.MethodGet, "/api/messages", nil)
	rec = httptest.NewRecorder()
	d.handleMessages(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// POST with valid body → dispatches to handlePostMessage (201)
	body := `{"inbox":"user","title":"T","content":"C","from":"F"}`
	req = httptest.NewRequest(http.MethodPost, "/api/messages", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	d.handleMessages(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandlePostMessage_NoStore(t *testing.T) {
	d := newTestDaemonT(t) // No msgStore set
	d.logger = log.Default()

	body := `{"inbox":"user","title":"T","content":"C","from":"F"}`
	req := httptest.NewRequest(http.MethodPost, "/api/messages", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	d.handlePostMessage(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandlePostMessage_DefaultValues(t *testing.T) {
	d := newTestDaemonWithMsgStore(t)

	// Send without category or message_type — should default to "general" and "request"
	body := `{"inbox":"coordinator","title":"Defaults test","content":"Body","from":"tester"}`
	req := httptest.NewRequest(http.MethodPost, "/api/messages", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	d.handlePostMessage(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp postMessageResponse
	json.NewDecoder(rec.Body).Decode(&resp)

	// Verify the stored message has correct defaults
	msg, err := d.msgStore.GetInboxMessage(resp.MessageID)
	if err != nil {
		t.Fatalf("failed to get stored message: %v", err)
	}
	if msg.Category != "general" {
		t.Errorf("expected category 'general', got %q", msg.Category)
	}
	if msg.MessageType != "request" {
		t.Errorf("expected message_type 'request', got %q", msg.MessageType)
	}
	if msg.ToInbox != "coordinator" {
		t.Errorf("expected to_inbox 'coordinator', got %q", msg.ToInbox)
	}
}

func TestHandlePostMessage_InvalidJSON(t *testing.T) {
	d := newTestDaemonWithMsgStore(t)

	req := httptest.NewRequest(http.MethodPost, "/api/messages", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	d.handlePostMessage(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

// M-REST-INGESTION: GET /api/messages tests

func TestHandleGetMessages_Empty(t *testing.T) {
	d := newTestDaemonWithMsgStore(t)

	req := httptest.NewRequest(http.MethodGet, "/api/messages", nil)
	rec := httptest.NewRecorder()

	d.handleGetMessages(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp getMessagesResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp.Count != 0 {
		t.Errorf("expected 0 messages, got %d", resp.Count)
	}
	if resp.Limit != 50 {
		t.Errorf("expected default limit 50, got %d", resp.Limit)
	}
	if resp.Messages == nil {
		t.Error("expected non-nil messages array")
	}
}

func TestHandleGetMessages_WithMessages(t *testing.T) {
	d := newTestDaemonWithMsgStore(t)

	// Insert two messages.
	for _, title := range []string{"First", "Second"} {
		msg := &messaging.InboxMessage{
			FromAgent:   "test",
			ToInbox:     "user",
			MessageType: "request",
			Title:       title,
			Payload:     "body",
			Category:    "general",
			Status:      messaging.InboxStatusUnread,
		}
		if err := d.msgStore.InsertInboxMessageWithContext(context.Background(), msg); err != nil {
			t.Fatalf("failed to insert message: %v", err)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/api/messages", nil)
	rec := httptest.NewRecorder()

	d.handleGetMessages(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp getMessagesResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Count != 2 {
		t.Errorf("expected 2 messages, got %d", resp.Count)
	}
}

func TestHandleGetMessages_FilterByInbox(t *testing.T) {
	d := newTestDaemonWithMsgStore(t)

	// Insert messages to different inboxes.
	for _, inbox := range []string{"user", "coordinator", "user"} {
		msg := &messaging.InboxMessage{
			FromAgent:   "test",
			ToInbox:     inbox,
			MessageType: "request",
			Title:       "Msg to " + inbox,
			Payload:     "body",
			Category:    "general",
			Status:      messaging.InboxStatusUnread,
		}
		d.msgStore.InsertInboxMessageWithContext(context.Background(), msg)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/messages?inbox=user", nil)
	rec := httptest.NewRecorder()

	d.handleGetMessages(rec, req)

	var resp getMessagesResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Count != 2 {
		t.Errorf("expected 2 messages for inbox 'user', got %d", resp.Count)
	}
}

func TestHandleGetMessages_FilterByStatus(t *testing.T) {
	d := newTestDaemonWithMsgStore(t)

	// Insert one unread and one read message.
	for _, status := range []string{messaging.InboxStatusUnread, messaging.InboxStatusRead} {
		msg := &messaging.InboxMessage{
			FromAgent:   "test",
			ToInbox:     "user",
			MessageType: "request",
			Title:       "Msg " + status,
			Payload:     "body",
			Category:    "general",
			Status:      status,
		}
		d.msgStore.InsertInboxMessageWithContext(context.Background(), msg)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/messages?status=unread", nil)
	rec := httptest.NewRecorder()

	d.handleGetMessages(rec, req)

	var resp getMessagesResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Count != 1 {
		t.Errorf("expected 1 unread message, got %d", resp.Count)
	}
}

func TestHandleGetMessages_NoStore(t *testing.T) {
	d := newTestDaemonT(t) // No msgStore
	d.logger = log.Default()

	req := httptest.NewRequest(http.MethodGet, "/api/messages", nil)
	rec := httptest.NewRecorder()

	d.handleGetMessages(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
}

func TestHandleGetMessages_CustomLimit(t *testing.T) {
	d := newTestDaemonWithMsgStore(t)

	// Insert 5 messages.
	for i := 0; i < 5; i++ {
		msg := &messaging.InboxMessage{
			FromAgent:   "test",
			ToInbox:     "user",
			MessageType: "request",
			Title:       "Msg",
			Payload:     "body",
			Category:    "general",
			Status:      messaging.InboxStatusUnread,
		}
		d.msgStore.InsertInboxMessageWithContext(context.Background(), msg)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/messages?limit=2", nil)
	rec := httptest.NewRecorder()

	d.handleGetMessages(rec, req)

	var resp getMessagesResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Count != 2 {
		t.Errorf("expected 2 messages (limit=2), got %d", resp.Count)
	}
	if resp.Limit != 2 {
		t.Errorf("expected limit 2, got %d", resp.Limit)
	}
}

func TestWriteJSON(t *testing.T) {
	rec := httptest.NewRecorder()
	writeJSON(rec, http.StatusOK, map[string]string{"key": "value"})

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	ct := rec.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("expected application/json, got %s", ct)
	}

	var resp map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp["key"] != "value" {
		t.Errorf("expected value, got %s", resp["key"])
	}
}
