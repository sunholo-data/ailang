package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/sunholo/ailang/internal/observatory"
)

// mockClaudeHooksBackend is a minimal mock implementing just the methods used by claude hooks.
type mockClaudeHooksBackend struct {
	observatory.Backend // embed to satisfy interface (panics on unimplemented methods)

	mu                    sync.Mutex
	upsertedSessions      []mockSession
	insertedToolStarts    []mockToolStart
	updatedToolEnds       []mockToolEnd
	endedSessions         []string
	backfilledSessions    []string
	latestUnfinishedTools map[string]string // key: sessionID+toolName, value: toolUseID
}

type mockSession struct {
	sessionID, workspace, version, source string
	corr                                  *observatory.SessionCorrelation
}

type mockToolStart struct {
	sessionID, toolUseID, toolName, toolInput string
}

type mockToolEnd struct {
	toolUseID, toolResponse string
	success                 bool
}

func (m *mockClaudeHooksBackend) UpsertSessionWithCorrelation(_ context.Context, sessionID, workspace, version, source string, corr *observatory.SessionCorrelation) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.upsertedSessions = append(m.upsertedSessions, mockSession{sessionID, workspace, version, source, corr})
	return nil
}

func (m *mockClaudeHooksBackend) InsertToolStart(_ context.Context, sessionID, toolUseID, toolName, toolInput string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.insertedToolStarts = append(m.insertedToolStarts, mockToolStart{sessionID, toolUseID, toolName, toolInput})
	return nil
}

func (m *mockClaudeHooksBackend) FindLatestUnfinishedTool(_ context.Context, sessionID, toolName string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := sessionID + ":" + toolName
	if id, ok := m.latestUnfinishedTools[key]; ok {
		return id, nil
	}
	return "", nil
}

func (m *mockClaudeHooksBackend) UpdateToolEnd(_ context.Context, toolUseID, toolResponse string, success bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.updatedToolEnds = append(m.updatedToolEnds, mockToolEnd{toolUseID, toolResponse, success})
	return nil
}

func (m *mockClaudeHooksBackend) UpdateSessionEnded(_ context.Context, sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.endedSessions = append(m.endedSessions, sessionID)
	return nil
}

func (m *mockClaudeHooksBackend) BackfillSpansWorkspace(_ context.Context, sessionID, _ string) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.backfilledSessions = append(m.backfilledSessions, sessionID)
	return 0, nil
}

// newMockClaudeHooksBackend creates a mock with zero values ready for assertions.
func newMockClaudeHooksBackend() *mockClaudeHooksBackend {
	return &mockClaudeHooksBackend{
		latestUnfinishedTools: make(map[string]string),
	}
}

// --- HTTP-level tests ---

func TestClaudeHooks_MethodNotAllowed(t *testing.T) {
	s := &Server{}
	for _, method := range []string{"GET", "PUT", "DELETE", "PATCH"} {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/hooks/claude", nil)
			w := httptest.NewRecorder()
			s.handleClaudeHooks(w, req)
			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("expected 405, got %d", w.Code)
			}
		})
	}
}

func TestClaudeHooks_NoBackend(t *testing.T) {
	s := &Server{obsBackend: nil}
	body := `{"hook_event_name":"SessionStart","session_id":"s1","cwd":"/tmp"}`
	req := httptest.NewRequest("POST", "/api/hooks/claude", bytes.NewReader([]byte(body)))
	w := httptest.NewRecorder()
	s.handleClaudeHooks(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}
}

func TestClaudeHooks_InvalidJSON(t *testing.T) {
	mock := newMockClaudeHooksBackend()
	s := &Server{obsBackend: mock}
	req := httptest.NewRequest("POST", "/api/hooks/claude", bytes.NewReader([]byte("not json")))
	w := httptest.NewRecorder()
	s.handleClaudeHooks(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestClaudeHooks_MissingEventName(t *testing.T) {
	mock := newMockClaudeHooksBackend()
	s := &Server{obsBackend: mock}
	body := `{"session_id":"s1","cwd":"/tmp"}`
	req := httptest.NewRequest("POST", "/api/hooks/claude", bytes.NewReader([]byte(body)))
	w := httptest.NewRecorder()
	s.handleClaudeHooks(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestClaudeHooks_UnknownEvent(t *testing.T) {
	mock := newMockClaudeHooksBackend()
	s := &Server{obsBackend: mock}
	body := `{"hook_event_name":"FutureEvent","session_id":"s1","cwd":"/tmp"}`
	req := httptest.NewRequest("POST", "/api/hooks/claude", bytes.NewReader([]byte(body)))
	w := httptest.NewRecorder()
	s.handleClaudeHooks(w, req)
	// Unknown events still return 200 OK (logged but not rejected)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

// --- Event-specific tests ---

func TestClaudeHooks_SessionStart(t *testing.T) {
	mock := newMockClaudeHooksBackend()
	s := &Server{obsBackend: mock}

	payload := ClaudeHookPayload{
		HookEventName: "SessionStart",
		SessionID:     "sess-abc",
		Cwd:           "/home/user/project",
		Model:         "claude-opus-4-6",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/api/hooks/claude", bytes.NewReader(body))
	w := httptest.NewRecorder()
	s.handleClaudeHooks(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	mock.mu.Lock()
	defer mock.mu.Unlock()
	if len(mock.upsertedSessions) != 1 {
		t.Fatalf("expected 1 upserted session, got %d", len(mock.upsertedSessions))
	}
	sess := mock.upsertedSessions[0]
	if sess.sessionID != "sess-abc" {
		t.Errorf("session_id: want sess-abc, got %s", sess.sessionID)
	}
	if sess.workspace != "/home/user/project" {
		t.Errorf("workspace: want /home/user/project, got %s", sess.workspace)
	}
	if sess.version != "claude-opus-4-6" {
		t.Errorf("version: want claude-opus-4-6, got %s", sess.version)
	}
	if sess.source != "http-hook" {
		t.Errorf("source: want http-hook, got %s", sess.source)
	}
	if sess.corr != nil {
		t.Errorf("corr should be nil when no headers set")
	}
}

func TestClaudeHooks_SessionStart_WithCorrelation(t *testing.T) {
	mock := newMockClaudeHooksBackend()
	s := &Server{obsBackend: mock}

	payload := ClaudeHookPayload{
		HookEventName: "SessionStart",
		SessionID:     "sess-corr",
		Cwd:           "/tmp",
		Model:         "opus",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/api/hooks/claude", bytes.NewReader(body))
	req.Header.Set("X-Ailang-Task-Id", "task-123")
	req.Header.Set("X-Ailang-Chain-Id", "chain-456")
	req.Header.Set("X-Ailang-Stage-Id", "stage-789")
	req.Header.Set("X-Ailang-Message-Id", "msg-abc")
	w := httptest.NewRecorder()
	s.handleClaudeHooks(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	mock.mu.Lock()
	defer mock.mu.Unlock()
	if len(mock.upsertedSessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(mock.upsertedSessions))
	}
	sess := mock.upsertedSessions[0]
	if sess.corr == nil {
		t.Fatal("expected correlation IDs, got nil")
	}
	if sess.corr.TaskID != "task-123" {
		t.Errorf("TaskID: want task-123, got %s", sess.corr.TaskID)
	}
	if sess.corr.ChainID != "chain-456" {
		t.Errorf("ChainID: want chain-456, got %s", sess.corr.ChainID)
	}
	if sess.corr.StageID != "stage-789" {
		t.Errorf("StageID: want stage-789, got %s", sess.corr.StageID)
	}
	if sess.corr.MessageID != "msg-abc" {
		t.Errorf("MessageID: want msg-abc, got %s", sess.corr.MessageID)
	}
}

func TestClaudeHooks_PreToolUse(t *testing.T) {
	mock := newMockClaudeHooksBackend()
	s := &Server{obsBackend: mock}

	body := `{"hook_event_name":"PreToolUse","session_id":"sess-1","tool_name":"Bash","tool_use_id":"tu-1","tool_input":{"command":"ls -la"}}`
	req := httptest.NewRequest("POST", "/api/hooks/claude", bytes.NewReader([]byte(body)))
	w := httptest.NewRecorder()
	s.handleClaudeHooks(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	mock.mu.Lock()
	defer mock.mu.Unlock()
	if len(mock.insertedToolStarts) != 1 {
		t.Fatalf("expected 1 tool start, got %d", len(mock.insertedToolStarts))
	}
	ts := mock.insertedToolStarts[0]
	if ts.sessionID != "sess-1" {
		t.Errorf("sessionID: want sess-1, got %s", ts.sessionID)
	}
	if ts.toolUseID != "tu-1" {
		t.Errorf("toolUseID: want tu-1, got %s", ts.toolUseID)
	}
	if ts.toolName != "Bash" {
		t.Errorf("toolName: want Bash, got %s", ts.toolName)
	}
	if ts.toolInput != `{"command":"ls -la"}` {
		t.Errorf("toolInput: want {\"command\":\"ls -la\"}, got %s", ts.toolInput)
	}
}

func TestClaudeHooks_PreToolUse_GeneratesID(t *testing.T) {
	mock := newMockClaudeHooksBackend()
	s := &Server{obsBackend: mock}

	// No tool_use_id — handler should generate a UUID
	body := `{"hook_event_name":"PreToolUse","session_id":"sess-1","tool_name":"Read"}`
	req := httptest.NewRequest("POST", "/api/hooks/claude", bytes.NewReader([]byte(body)))
	w := httptest.NewRecorder()
	s.handleClaudeHooks(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	mock.mu.Lock()
	defer mock.mu.Unlock()
	if len(mock.insertedToolStarts) != 1 {
		t.Fatalf("expected 1 tool start, got %d", len(mock.insertedToolStarts))
	}
	if mock.insertedToolStarts[0].toolUseID == "" {
		t.Error("expected generated toolUseID, got empty")
	}
}

func TestClaudeHooks_PostToolUse(t *testing.T) {
	mock := newMockClaudeHooksBackend()
	s := &Server{obsBackend: mock}

	body := `{"hook_event_name":"PostToolUse","session_id":"sess-1","tool_name":"Bash","tool_use_id":"tu-1","tool_response":"output text"}`
	req := httptest.NewRequest("POST", "/api/hooks/claude", bytes.NewReader([]byte(body)))
	w := httptest.NewRecorder()
	s.handleClaudeHooks(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	mock.mu.Lock()
	defer mock.mu.Unlock()
	if len(mock.updatedToolEnds) != 1 {
		t.Fatalf("expected 1 tool end, got %d", len(mock.updatedToolEnds))
	}
	te := mock.updatedToolEnds[0]
	if te.toolUseID != "tu-1" {
		t.Errorf("toolUseID: want tu-1, got %s", te.toolUseID)
	}
	if !te.success {
		t.Error("expected success=true for PostToolUse")
	}
}

func TestClaudeHooks_PostToolUseFailure(t *testing.T) {
	mock := newMockClaudeHooksBackend()
	s := &Server{obsBackend: mock}

	body := `{"hook_event_name":"PostToolUseFailure","session_id":"sess-1","tool_name":"Bash","tool_use_id":"tu-fail","tool_response":"permission denied"}`
	req := httptest.NewRequest("POST", "/api/hooks/claude", bytes.NewReader([]byte(body)))
	w := httptest.NewRecorder()
	s.handleClaudeHooks(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	mock.mu.Lock()
	defer mock.mu.Unlock()
	if len(mock.updatedToolEnds) != 1 {
		t.Fatalf("expected 1 tool end, got %d", len(mock.updatedToolEnds))
	}
	te := mock.updatedToolEnds[0]
	if te.toolUseID != "tu-fail" {
		t.Errorf("toolUseID: want tu-fail, got %s", te.toolUseID)
	}
	if te.success {
		t.Error("expected success=false for PostToolUseFailure")
	}
}

func TestClaudeHooks_Stop(t *testing.T) {
	mock := newMockClaudeHooksBackend()
	s := &Server{obsBackend: mock}

	body := `{"hook_event_name":"Stop","session_id":"sess-end"}`
	req := httptest.NewRequest("POST", "/api/hooks/claude", bytes.NewReader([]byte(body)))
	w := httptest.NewRecorder()
	s.handleClaudeHooks(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	mock.mu.Lock()
	defer mock.mu.Unlock()
	if len(mock.endedSessions) != 1 {
		t.Fatalf("expected 1 ended session, got %d", len(mock.endedSessions))
	}
	if mock.endedSessions[0] != "sess-end" {
		t.Errorf("ended session: want sess-end, got %s", mock.endedSessions[0])
	}
}

func TestClaudeHooks_SubagentStart(t *testing.T) {
	mock := newMockClaudeHooksBackend()
	s := &Server{obsBackend: mock}

	body := `{"hook_event_name":"SubagentStart","session_id":"sess-1","agent_id":"agent-abc","agent_type":"general-purpose"}`
	req := httptest.NewRequest("POST", "/api/hooks/claude", bytes.NewReader([]byte(body)))
	w := httptest.NewRecorder()
	s.handleClaudeHooks(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	mock.mu.Lock()
	defer mock.mu.Unlock()
	if len(mock.insertedToolStarts) != 1 {
		t.Fatalf("expected 1 tool start for SubagentStart, got %d", len(mock.insertedToolStarts))
	}
	ts := mock.insertedToolStarts[0]
	if ts.toolName != "Subagent:general-purpose" {
		t.Errorf("toolName: want Subagent:general-purpose, got %s", ts.toolName)
	}
	if ts.toolUseID != "agent-abc" {
		t.Errorf("toolUseID: want agent-abc, got %s", ts.toolUseID)
	}
}

func TestClaudeHooks_SubagentStop(t *testing.T) {
	mock := newMockClaudeHooksBackend()
	s := &Server{obsBackend: mock}

	body := `{"hook_event_name":"SubagentStop","session_id":"sess-1","agent_id":"agent-abc","agent_type":"general-purpose"}`
	req := httptest.NewRequest("POST", "/api/hooks/claude", bytes.NewReader([]byte(body)))
	w := httptest.NewRecorder()
	s.handleClaudeHooks(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	mock.mu.Lock()
	defer mock.mu.Unlock()
	if len(mock.updatedToolEnds) != 1 {
		t.Fatalf("expected 1 tool end for SubagentStop, got %d", len(mock.updatedToolEnds))
	}
	te := mock.updatedToolEnds[0]
	if te.toolUseID != "agent-abc" {
		t.Errorf("toolUseID: want agent-abc, got %s", te.toolUseID)
	}
	if !te.success {
		t.Error("expected success=true for SubagentStop")
	}
}

func TestClaudeHooks_TaskCompleted(t *testing.T) {
	mock := newMockClaudeHooksBackend()
	s := &Server{obsBackend: mock}

	body := `{"hook_event_name":"TaskCompleted","session_id":"sess-1","agent_type":"sprint-executor"}`
	req := httptest.NewRequest("POST", "/api/hooks/claude", bytes.NewReader([]byte(body)))
	req.Header.Set("X-Ailang-Task-Id", "task-xyz")
	w := httptest.NewRecorder()
	s.handleClaudeHooks(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	// TaskCompleted only logs — verify no side effects
	mock.mu.Lock()
	defer mock.mu.Unlock()
	if len(mock.upsertedSessions) != 0 {
		t.Errorf("TaskCompleted should not upsert sessions")
	}
}

func TestClaudeHooks_ResponseFormat(t *testing.T) {
	mock := newMockClaudeHooksBackend()
	s := &Server{obsBackend: mock}

	body := `{"hook_event_name":"Stop","session_id":"sess-1"}`
	req := httptest.NewRequest("POST", "/api/hooks/claude", bytes.NewReader([]byte(body)))
	w := httptest.NewRecorder()
	s.handleClaudeHooks(w, req)

	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type: want application/json, got %s", ct)
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if resp["status"] != "ok" {
		t.Errorf("status: want ok, got %s", resp["status"])
	}
}

func TestClaudeHooks_ToolResponseTruncation(t *testing.T) {
	mock := newMockClaudeHooksBackend()
	s := &Server{obsBackend: mock}

	// Create response > 10000 chars
	longResp := make([]byte, 15000)
	for i := range longResp {
		longResp[i] = 'x'
	}

	payload := map[string]interface{}{
		"hook_event_name": "PostToolUse",
		"session_id":      "sess-1",
		"tool_name":       "Read",
		"tool_use_id":     "tu-long",
		"tool_response":   string(longResp),
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/api/hooks/claude", bytes.NewReader(body))
	w := httptest.NewRecorder()
	s.handleClaudeHooks(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	mock.mu.Lock()
	defer mock.mu.Unlock()
	if len(mock.updatedToolEnds) != 1 {
		t.Fatalf("expected 1 tool end, got %d", len(mock.updatedToolEnds))
	}
	resp := mock.updatedToolEnds[0].toolResponse
	if len(resp) > 10020 { // 10000 + "...[truncated]"
		t.Errorf("tool response not truncated: length %d", len(resp))
	}
}

func TestClaudeHooks_AllEventTypes(t *testing.T) {
	// Verify all 8 event types return 200 with a configured backend
	mock := newMockClaudeHooksBackend()
	s := &Server{obsBackend: mock}

	events := []string{
		"SessionStart", "PreToolUse", "PostToolUse", "PostToolUseFailure",
		"Stop", "SubagentStart", "SubagentStop", "TaskCompleted",
	}

	for _, evt := range events {
		t.Run(evt, func(t *testing.T) {
			payload := map[string]interface{}{
				"hook_event_name": evt,
				"session_id":      "sess-all",
				"cwd":             "/tmp",
				"tool_name":       "TestTool",
				"tool_use_id":     "tu-all",
				"agent_type":      "test-agent",
				"agent_id":        "agent-all",
			}
			body, _ := json.Marshal(payload)
			req := httptest.NewRequest("POST", "/api/hooks/claude", bytes.NewReader(body))
			w := httptest.NewRecorder()
			s.handleClaudeHooks(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("%s: expected 200, got %d", evt, w.Code)
			}
		})
	}
}

func TestClaudeHooks_PostToolUse_FindsUnfinished(t *testing.T) {
	mock := newMockClaudeHooksBackend()
	mock.latestUnfinishedTools["sess-1:Bash"] = "tu-found"
	s := &Server{obsBackend: mock}

	// No tool_use_id in payload — handler should find via FindLatestUnfinishedTool
	body := `{"hook_event_name":"PostToolUse","session_id":"sess-1","tool_name":"Bash","tool_response":"done"}`
	req := httptest.NewRequest("POST", "/api/hooks/claude", bytes.NewReader([]byte(body)))
	w := httptest.NewRecorder()
	s.handleClaudeHooks(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	mock.mu.Lock()
	defer mock.mu.Unlock()
	if len(mock.updatedToolEnds) != 1 {
		t.Fatalf("expected 1 tool end, got %d", len(mock.updatedToolEnds))
	}
	if mock.updatedToolEnds[0].toolUseID != "tu-found" {
		t.Errorf("toolUseID: want tu-found, got %s", mock.updatedToolEnds[0].toolUseID)
	}
}

// --- Payload parsing tests ---

func TestClaudeHookPayload_AllFields(t *testing.T) {
	raw := `{
		"session_id": "s1",
		"transcript_path": "/path/to/transcript",
		"cwd": "/home/user",
		"permission_mode": "plan",
		"hook_event_name": "PreToolUse",
		"agent_id": "agent-1",
		"agent_type": "general-purpose",
		"tool_name": "Bash",
		"tool_input": {"command": "ls"},
		"tool_response": "output",
		"tool_use_id": "tu-1",
		"source": "vscode",
		"model": "claude-opus-4-6"
	}`

	var p ClaudeHookPayload
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if p.SessionID != "s1" {
		t.Errorf("SessionID: want s1, got %s", p.SessionID)
	}
	if p.TranscriptPath != "/path/to/transcript" {
		t.Errorf("TranscriptPath mismatch")
	}
	if p.Cwd != "/home/user" {
		t.Errorf("Cwd mismatch")
	}
	if p.PermissionMode != "plan" {
		t.Errorf("PermissionMode mismatch")
	}
	if p.HookEventName != "PreToolUse" {
		t.Errorf("HookEventName mismatch")
	}
	if p.AgentID != "agent-1" {
		t.Errorf("AgentID mismatch")
	}
	if p.AgentType != "general-purpose" {
		t.Errorf("AgentType mismatch")
	}
	if p.ToolName != "Bash" {
		t.Errorf("ToolName mismatch")
	}
	if p.ToolUseID != "tu-1" {
		t.Errorf("ToolUseID mismatch")
	}
	if p.Source != "vscode" {
		t.Errorf("Source mismatch")
	}
	if p.Model != "claude-opus-4-6" {
		t.Errorf("Model mismatch")
	}
	if p.ToolInput == nil {
		t.Error("ToolInput should not be nil")
	}
	if p.ToolResponse == nil {
		t.Error("ToolResponse should not be nil")
	}
}

func TestClaudeHookPayload_OmittedFields(t *testing.T) {
	raw := `{"session_id":"s1","hook_event_name":"Stop","cwd":"/tmp"}`
	var p ClaudeHookPayload
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if p.ToolName != "" {
		t.Errorf("ToolName should be empty, got %s", p.ToolName)
	}
	if p.AgentID != "" {
		t.Errorf("AgentID should be empty, got %s", p.AgentID)
	}
	if p.ToolInput != nil {
		t.Errorf("ToolInput should be nil")
	}
}

func TestClaudeHookPayload_CorrelationIDsNotInJSON(t *testing.T) {
	// Correlation IDs use `json:"-"` so they must not appear in JSON output
	p := ClaudeHookPayload{
		SessionID:     "s1",
		HookEventName: "SessionStart",
		TaskID:        "task-secret",
		ChainID:       "chain-secret",
	}
	body, _ := json.Marshal(p)
	if bytes.Contains(body, []byte("task-secret")) {
		t.Error("TaskID should not be in JSON output (has json:\"-\" tag)")
	}
	if bytes.Contains(body, []byte("chain-secret")) {
		t.Error("ChainID should not be in JSON output (has json:\"-\" tag)")
	}
}

// --- Benchmark ---

func BenchmarkClaudeHooks_SessionStart(b *testing.B) {
	mock := newMockClaudeHooksBackend()
	s := &Server{obsBackend: mock}
	body := []byte(`{"hook_event_name":"SessionStart","session_id":"bench","cwd":"/tmp","model":"opus"}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/api/hooks/claude", bytes.NewReader(body))
		w := httptest.NewRecorder()
		s.handleClaudeHooks(w, req)
	}
}

// --- Hook Token Auth Tests ---

func TestClaudeHooks_AuthToken_Rejected(t *testing.T) {
	mock := newMockClaudeHooksBackend()
	s := &Server{obsBackend: mock, hookToken: "secret-token-123"}
	body := `{"hook_event_name":"SessionStart","session_id":"s1","cwd":"/tmp"}`

	// No auth header
	req := httptest.NewRequest("POST", "/api/hooks/claude", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	s.handleClaudeHooks(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}

	// Wrong token
	req = httptest.NewRequest("POST", "/api/hooks/claude", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer wrong-token")
	w = httptest.NewRecorder()
	s.handleClaudeHooks(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}

	// Not Bearer scheme
	req = httptest.NewRequest("POST", "/api/hooks/claude", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
	w = httptest.NewRecorder()
	s.handleClaudeHooks(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestClaudeHooks_AuthToken_Accepted(t *testing.T) {
	mock := newMockClaudeHooksBackend()
	s := &Server{obsBackend: mock, hookToken: "secret-token-123"}
	body := `{"hook_event_name":"SessionStart","session_id":"s1","cwd":"/tmp"}`

	req := httptest.NewRequest("POST", "/api/hooks/claude", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer secret-token-123")
	w := httptest.NewRecorder()
	s.handleClaudeHooks(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestClaudeHooks_NoToken_PassThrough(t *testing.T) {
	mock := newMockClaudeHooksBackend()
	s := &Server{obsBackend: mock} // hookToken is empty = local mode
	body := `{"hook_event_name":"SessionStart","session_id":"s1","cwd":"/tmp"}`

	req := httptest.NewRequest("POST", "/api/hooks/claude", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	s.handleClaudeHooks(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 (no token = pass through), got %d", w.Code)
	}
}

func BenchmarkClaudeHooks_PreToolUse(b *testing.B) {
	mock := newMockClaudeHooksBackend()
	s := &Server{obsBackend: mock}
	body := []byte(`{"hook_event_name":"PreToolUse","session_id":"bench","tool_name":"Bash","tool_use_id":"tu-bench","tool_input":{"command":"echo hi"}}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/api/hooks/claude", bytes.NewReader(body))
		w := httptest.NewRecorder()
		s.handleClaudeHooks(w, req)
	}
}
