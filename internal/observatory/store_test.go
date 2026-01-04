package observatory

import (
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func setupTestStore(t *testing.T) *Store {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	// Enable foreign keys
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("failed to enable foreign keys: %v", err)
	}

	if err := Migrate(db); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	return NewStore(db)
}

// ===== Workspace Tests =====

func TestStore_CreateAndGetWorkspace(t *testing.T) {
	store := setupTestStore(t)
	defer store.DB().Close()

	now := time.Now().Truncate(time.Second)
	ws := &Workspace{
		ID:        "ws-1",
		Name:      "Test Workspace",
		Path:      "/path/to/workspace",
		GitRemote: "https://github.com/test/repo",
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := store.CreateWorkspace(ws); err != nil {
		t.Fatalf("CreateWorkspace failed: %v", err)
	}

	got, err := store.GetWorkspace("ws-1")
	if err != nil {
		t.Fatalf("GetWorkspace failed: %v", err)
	}

	if got.ID != ws.ID {
		t.Errorf("ID mismatch: got %s, want %s", got.ID, ws.ID)
	}
	if got.Name != ws.Name {
		t.Errorf("Name mismatch: got %s, want %s", got.Name, ws.Name)
	}
	if got.Path != ws.Path {
		t.Errorf("Path mismatch: got %s, want %s", got.Path, ws.Path)
	}
	if got.GitRemote != ws.GitRemote {
		t.Errorf("GitRemote mismatch: got %s, want %s", got.GitRemote, ws.GitRemote)
	}
}

func TestStore_ListWorkspaces(t *testing.T) {
	store := setupTestStore(t)
	defer store.DB().Close()

	now := time.Now().Truncate(time.Second)
	for i := 0; i < 3; i++ {
		ws := &Workspace{
			ID:        "ws-" + string(rune('a'+i)),
			Name:      "Workspace " + string(rune('A'+i)),
			Path:      "/path/" + string(rune('a'+i)),
			CreatedAt: now,
			UpdatedAt: now.Add(time.Duration(i) * time.Hour),
		}
		if err := store.CreateWorkspace(ws); err != nil {
			t.Fatalf("CreateWorkspace failed: %v", err)
		}
	}

	workspaces, err := store.ListWorkspaces()
	if err != nil {
		t.Fatalf("ListWorkspaces failed: %v", err)
	}

	if len(workspaces) != 3 {
		t.Errorf("expected 3 workspaces, got %d", len(workspaces))
	}

	// Should be ordered by updated_at DESC
	if workspaces[0].ID != "ws-c" {
		t.Errorf("expected most recently updated first, got %s", workspaces[0].ID)
	}
}

func TestStore_UpdateWorkspace(t *testing.T) {
	store := setupTestStore(t)
	defer store.DB().Close()

	now := time.Now().Truncate(time.Second)
	ws := &Workspace{
		ID:        "ws-1",
		Name:      "Original",
		Path:      "/original",
		CreatedAt: now,
		UpdatedAt: now,
	}
	store.CreateWorkspace(ws)

	ws.Name = "Updated"
	ws.Path = "/updated"
	if err := store.UpdateWorkspace(ws); err != nil {
		t.Fatalf("UpdateWorkspace failed: %v", err)
	}

	got, _ := store.GetWorkspace("ws-1")
	if got.Name != "Updated" {
		t.Errorf("Name not updated: got %s", got.Name)
	}
	if got.Path != "/updated" {
		t.Errorf("Path not updated: got %s", got.Path)
	}
}

func TestStore_DeleteWorkspace(t *testing.T) {
	store := setupTestStore(t)
	defer store.DB().Close()

	now := time.Now().Truncate(time.Second)
	ws := &Workspace{
		ID:        "ws-1",
		Name:      "Test",
		Path:      "/test",
		CreatedAt: now,
		UpdatedAt: now,
	}
	store.CreateWorkspace(ws)

	if err := store.DeleteWorkspace("ws-1"); err != nil {
		t.Fatalf("DeleteWorkspace failed: %v", err)
	}

	_, err := store.GetWorkspace("ws-1")
	if err != sql.ErrNoRows {
		t.Errorf("expected ErrNoRows, got %v", err)
	}
}

// ===== Task Tests =====

func TestStore_CreateAndGetTask(t *testing.T) {
	store := setupTestStore(t)
	defer store.DB().Close()

	now := time.Now().Truncate(time.Second)
	ws := &Workspace{ID: "ws-1", Name: "Test", Path: "/test", CreatedAt: now, UpdatedAt: now}
	store.CreateWorkspace(ws)

	task := &Task{
		ID:          "task-1",
		WorkspaceID: "ws-1",
		Title:       "Test Task",
		Description: "Test description",
		SourceType:  TaskSourceGitHub,
		SourceRef:   "#123",
		Status:      TaskStatusPending,
		Priority:    "P1",
		CreatedAt:   now,
	}

	if err := store.CreateTask(task); err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	got, err := store.GetTask("task-1")
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}

	if got.Title != task.Title {
		t.Errorf("Title mismatch: got %s, want %s", got.Title, task.Title)
	}
	if got.SourceType != TaskSourceGitHub {
		t.Errorf("SourceType mismatch: got %s", got.SourceType)
	}
}

func TestStore_ListTasks_WithFilters(t *testing.T) {
	store := setupTestStore(t)
	defer store.DB().Close()

	now := time.Now().Truncate(time.Second)
	ws := &Workspace{ID: "ws-1", Name: "Test", Path: "/test", CreatedAt: now, UpdatedAt: now}
	store.CreateWorkspace(ws)

	// Create tasks with different statuses
	statuses := []TaskStatus{TaskStatusPending, TaskStatusRunning, TaskStatusCompleted}
	for i, status := range statuses {
		task := &Task{
			ID:          "task-" + string(rune('a'+i)),
			WorkspaceID: "ws-1",
			Title:       "Task " + string(rune('A'+i)),
			SourceType:  TaskSourceManual,
			Status:      status,
			Priority:    "P1",
			CreatedAt:   now,
		}
		store.CreateTask(task)
	}

	// Filter by status
	tasks, err := store.ListTasks(TaskListOptions{Status: TaskStatusPending})
	if err != nil {
		t.Fatalf("ListTasks failed: %v", err)
	}
	if len(tasks) != 1 {
		t.Errorf("expected 1 pending task, got %d", len(tasks))
	}

	// Filter by workspace
	tasks, err = store.ListTasks(TaskListOptions{WorkspaceID: "ws-1"})
	if err != nil {
		t.Fatalf("ListTasks failed: %v", err)
	}
	if len(tasks) != 3 {
		t.Errorf("expected 3 tasks in workspace, got %d", len(tasks))
	}
}

func TestStore_UpdateTask(t *testing.T) {
	store := setupTestStore(t)
	defer store.DB().Close()

	now := time.Now().Truncate(time.Second)
	ws := &Workspace{ID: "ws-1", Name: "Test", Path: "/test", CreatedAt: now, UpdatedAt: now}
	store.CreateWorkspace(ws)

	task := &Task{
		ID:          "task-1",
		WorkspaceID: "ws-1",
		Title:       "Original",
		SourceType:  TaskSourceManual,
		Status:      TaskStatusPending,
		Priority:    "P2",
		CreatedAt:   now,
	}
	store.CreateTask(task)

	task.Status = TaskStatusRunning
	startedAt := now.Add(time.Hour)
	task.StartedAt = &startedAt
	task.TotalTokensIn = 1000
	task.TotalCostUSD = 0.05

	if err := store.UpdateTask(task); err != nil {
		t.Fatalf("UpdateTask failed: %v", err)
	}

	got, _ := store.GetTask("task-1")
	if got.Status != TaskStatusRunning {
		t.Errorf("Status not updated")
	}
	if got.StartedAt == nil {
		t.Error("StartedAt not set")
	}
	if got.TotalTokensIn != 1000 {
		t.Errorf("TotalTokensIn mismatch: got %d", got.TotalTokensIn)
	}
}

// ===== Agent Assignment Tests =====

func TestStore_CreateAndGetAgentAssignment(t *testing.T) {
	store := setupTestStore(t)
	defer store.DB().Close()

	now := time.Now().Truncate(time.Second)
	ws := &Workspace{ID: "ws-1", Name: "Test", Path: "/test", CreatedAt: now, UpdatedAt: now}
	if err := store.CreateWorkspace(ws); err != nil {
		t.Fatalf("CreateWorkspace failed: %v", err)
	}
	task := &Task{ID: "task-1", WorkspaceID: "ws-1", Title: "Test", SourceType: TaskSourceManual, Status: TaskStatusRunning, Priority: "P1", CreatedAt: now}
	if err := store.CreateTask(task); err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	assignment := &AgentAssignment{
		ID:         "aa-1",
		TaskID:     "task-1",
		AgentID:    "claude-code",
		Provider:   ProviderClaude,
		Status:     AgentStatusRunning,
		AssignedAt: now,
		TokensIn:   500,
		TokensOut:  100,
		CostUSD:    0.02,
	}

	if err := store.CreateAgentAssignment(assignment); err != nil {
		t.Fatalf("CreateAgentAssignment failed: %v", err)
	}

	got, err := store.GetAgentAssignment("aa-1")
	if err != nil {
		t.Fatalf("GetAgentAssignment failed: %v", err)
	}

	if got.AgentID != "claude-code" {
		t.Errorf("AgentID mismatch: got %s", got.AgentID)
	}
	if got.Provider != ProviderClaude {
		t.Errorf("Provider mismatch: got %s", got.Provider)
	}
}

func TestStore_ListAgentAssignments(t *testing.T) {
	store := setupTestStore(t)
	defer store.DB().Close()

	now := time.Now().Truncate(time.Second)
	ws := &Workspace{ID: "ws-1", Name: "Test", Path: "/test", CreatedAt: now, UpdatedAt: now}
	if err := store.CreateWorkspace(ws); err != nil {
		t.Fatalf("CreateWorkspace failed: %v", err)
	}
	task := &Task{ID: "task-1", WorkspaceID: "ws-1", Title: "Test", SourceType: TaskSourceManual, Status: TaskStatusRunning, Priority: "P1", CreatedAt: now}
	if err := store.CreateTask(task); err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	for i := 0; i < 3; i++ {
		aa := &AgentAssignment{
			ID:         "aa-" + string(rune('a'+i)),
			TaskID:     "task-1",
			AgentID:    "agent-" + string(rune('a'+i)),
			Provider:   ProviderClaude,
			Status:     AgentStatusCompleted,
			AssignedAt: now.Add(time.Duration(i) * time.Hour),
		}
		if err := store.CreateAgentAssignment(aa); err != nil {
			t.Fatalf("CreateAgentAssignment failed: %v", err)
		}
	}

	assignments, err := store.ListAgentAssignments("task-1")
	if err != nil {
		t.Fatalf("ListAgentAssignments failed: %v", err)
	}

	if len(assignments) != 3 {
		t.Errorf("expected 3 assignments, got %d", len(assignments))
	}

	// Should be ordered by assigned_at ASC
	if assignments[0].ID != "aa-a" {
		t.Errorf("expected earliest assignment first, got %s", assignments[0].ID)
	}
}

// ===== Span Tests =====

func TestStore_CreateAndGetSpan(t *testing.T) {
	store := setupTestStore(t)
	defer store.DB().Close()

	now := time.Now().Truncate(time.Second)
	span := &Span{
		ID:        "span-1",
		TraceID:   "trace-1",
		Name:      "test.operation",
		Kind:      SpanKindInternal,
		Status:    SpanStatusOK,
		StartTime: now,
		TokensIn:  100,
		TokensOut: 50,
		CostUSD:   0.01,
		Model:     "claude-3-sonnet",
		Provider:  ProviderClaude,
		Attributes: map[string]any{
			"custom.key": "value",
		},
		CreatedAt: now,
	}

	if err := store.CreateSpan(span); err != nil {
		t.Fatalf("CreateSpan failed: %v", err)
	}

	got, err := store.GetSpan("span-1")
	if err != nil {
		t.Fatalf("GetSpan failed: %v", err)
	}

	if got.Name != "test.operation" {
		t.Errorf("Name mismatch: got %s", got.Name)
	}
	if got.Model != "claude-3-sonnet" {
		t.Errorf("Model mismatch: got %s", got.Model)
	}
	if got.Attributes["custom.key"] != "value" {
		t.Errorf("Attributes not preserved")
	}
}

func TestStore_ListSpans_WithFilters(t *testing.T) {
	store := setupTestStore(t)
	defer store.DB().Close()

	now := time.Now().Truncate(time.Second)

	// Create spans with different trace IDs
	for i := 0; i < 5; i++ {
		traceID := "trace-1"
		if i >= 3 {
			traceID = "trace-2"
		}
		span := &Span{
			ID:        "span-" + string(rune('a'+i)),
			TraceID:   traceID,
			Name:      "operation",
			Kind:      SpanKindInternal,
			Status:    SpanStatusOK,
			StartTime: now.Add(time.Duration(i) * time.Hour),
			CreatedAt: now,
		}
		store.CreateSpan(span)
	}

	// Filter by trace ID
	spans, err := store.ListSpans(SpanListOptions{TraceID: "trace-1"})
	if err != nil {
		t.Fatalf("ListSpans failed: %v", err)
	}
	if len(spans) != 3 {
		t.Errorf("expected 3 spans for trace-1, got %d", len(spans))
	}

	// Filter by time range (inclusive on both ends)
	spans, err = store.ListSpans(SpanListOptions{
		StartAfter:  now.Add(1 * time.Hour),
		StartBefore: now.Add(3 * time.Hour),
	})
	if err != nil {
		t.Fatalf("ListSpans failed: %v", err)
	}
	// Spans at: now, now+1hr, now+2hr, now+3hr, now+4hr
	// Filter: >= now+1hr AND <= now+3hr → spans at now+1hr, now+2hr, now+3hr = 3 spans
	if len(spans) != 3 {
		t.Errorf("expected 3 spans in time range (inclusive), got %d", len(spans))
	}
}

func TestStore_GetTrace(t *testing.T) {
	store := setupTestStore(t)
	defer store.DB().Close()

	now := time.Now().Truncate(time.Second)

	// Create a trace with multiple spans
	spans := []*Span{
		{ID: "root", TraceID: "trace-1", Name: "root", Kind: SpanKindServer, Status: SpanStatusOK, StartTime: now, CreatedAt: now},
		{ID: "child-1", TraceID: "trace-1", ParentSpanID: "root", Name: "child1", Kind: SpanKindInternal, Status: SpanStatusOK, StartTime: now.Add(time.Second), CreatedAt: now},
		{ID: "child-2", TraceID: "trace-1", ParentSpanID: "root", Name: "child2", Kind: SpanKindInternal, Status: SpanStatusOK, StartTime: now.Add(2 * time.Second), CreatedAt: now},
	}
	for _, s := range spans {
		store.CreateSpan(s)
	}

	trace, err := store.GetTrace("trace-1")
	if err != nil {
		t.Fatalf("GetTrace failed: %v", err)
	}

	if trace.TraceID != "trace-1" {
		t.Errorf("TraceID mismatch")
	}
	if trace.SpanCount != 3 {
		t.Errorf("SpanCount mismatch: got %d", trace.SpanCount)
	}
	if trace.RootSpan == nil {
		t.Error("RootSpan not found")
	}
	if trace.RootSpan != nil && trace.RootSpan.ID != "root" {
		t.Errorf("wrong RootSpan: got %s", trace.RootSpan.ID)
	}
}

// ===== Span Event Tests =====

func TestStore_CreateAndGetSpanEvents(t *testing.T) {
	store := setupTestStore(t)
	defer store.DB().Close()

	now := time.Now().Truncate(time.Second)
	span := &Span{ID: "span-1", TraceID: "trace-1", Name: "test", Kind: SpanKindInternal, Status: SpanStatusOK, StartTime: now, CreatedAt: now}
	store.CreateSpan(span)

	events := []SpanEvent{
		{SpanID: "span-1", Name: "tool.call", Timestamp: now, EventType: EventTypeTool, ToolName: "Read"},
		{SpanID: "span-1", Name: "approval.request", Timestamp: now.Add(time.Second), EventType: EventTypeApproval, ApprovalStatus: ApprovalStatusPending},
	}

	for i := range events {
		if err := store.CreateSpanEvent(&events[i]); err != nil {
			t.Fatalf("CreateSpanEvent failed: %v", err)
		}
		if events[i].ID == 0 {
			t.Error("Event ID not set")
		}
	}

	got, err := store.GetSpanEvents("span-1")
	if err != nil {
		t.Fatalf("GetSpanEvents failed: %v", err)
	}

	if len(got) != 2 {
		t.Errorf("expected 2 events, got %d", len(got))
	}
	if got[0].ToolName != "Read" {
		t.Errorf("ToolName mismatch: got %s", got[0].ToolName)
	}
	if got[1].ApprovalStatus != ApprovalStatusPending {
		t.Errorf("ApprovalStatus mismatch")
	}
}

// ===== Message Tests =====

func TestStore_CreateAndGetMessage(t *testing.T) {
	store := setupTestStore(t)
	defer store.DB().Close()

	now := time.Now().Truncate(time.Second)
	msg := &Message{
		ID:                "msg-1",
		Inbox:             "design-doc-creator",
		FromAgent:         "user",
		Title:             "New Feature Request",
		Content:           "Please create a design doc for...",
		MessageType:       "request",
		Status:            MessageStatusUnread,
		Priority:          "P1",
		GitHubIssueNumber: 42,
		GitHubRepo:        "sunholo-data/ailang",
		CreatedAt:         now,
	}

	if err := store.CreateMessage(msg); err != nil {
		t.Fatalf("CreateMessage failed: %v", err)
	}

	got, err := store.GetMessage("msg-1")
	if err != nil {
		t.Fatalf("GetMessage failed: %v", err)
	}

	if got.Title != msg.Title {
		t.Errorf("Title mismatch")
	}
	if got.GitHubIssueNumber != 42 {
		t.Errorf("GitHubIssueNumber mismatch: got %d", got.GitHubIssueNumber)
	}
}

func TestStore_ListMessages_WithFilters(t *testing.T) {
	store := setupTestStore(t)
	defer store.DB().Close()

	now := time.Now().Truncate(time.Second)

	// Create messages in different inboxes
	inboxes := []string{"design-doc-creator", "sprint-planner", "sprint-executor"}
	for i, inbox := range inboxes {
		msg := &Message{
			ID:          "msg-" + string(rune('a'+i)),
			Inbox:       inbox,
			FromAgent:   "user",
			Title:       "Message " + string(rune('A'+i)),
			Content:     "Content",
			MessageType: "request",
			Status:      MessageStatusUnread,
			Priority:    "P1",
			CreatedAt:   now,
		}
		store.CreateMessage(msg)
	}

	// Filter by inbox
	messages, err := store.ListMessages(MessageListOptions{Inbox: "sprint-planner"})
	if err != nil {
		t.Fatalf("ListMessages failed: %v", err)
	}
	if len(messages) != 1 {
		t.Errorf("expected 1 message, got %d", len(messages))
	}

	// Filter by status
	store.MarkMessageRead("msg-a")
	messages, err = store.ListMessages(MessageListOptions{Status: MessageStatusUnread})
	if err != nil {
		t.Fatalf("ListMessages failed: %v", err)
	}
	if len(messages) != 2 {
		t.Errorf("expected 2 unread messages, got %d", len(messages))
	}
}

func TestStore_MarkMessageRead(t *testing.T) {
	store := setupTestStore(t)
	defer store.DB().Close()

	now := time.Now().Truncate(time.Second)
	msg := &Message{
		ID:          "msg-1",
		Inbox:       "user",
		FromAgent:   "agent",
		Title:       "Test",
		Content:     "Test",
		MessageType: "info",
		Status:      MessageStatusUnread,
		Priority:    "P2",
		CreatedAt:   now,
	}
	store.CreateMessage(msg)

	if err := store.MarkMessageRead("msg-1"); err != nil {
		t.Fatalf("MarkMessageRead failed: %v", err)
	}

	got, _ := store.GetMessage("msg-1")
	if got.Status != MessageStatusRead {
		t.Errorf("Status not updated to read")
	}
	if got.ReadAt == nil {
		t.Error("ReadAt not set")
	}
}

func TestStore_MarkMessageArchived(t *testing.T) {
	store := setupTestStore(t)
	defer store.DB().Close()

	now := time.Now().Truncate(time.Second)
	msg := &Message{
		ID:          "msg-1",
		Inbox:       "user",
		FromAgent:   "agent",
		Title:       "Test",
		Content:     "Test",
		MessageType: "info",
		Status:      MessageStatusUnread,
		Priority:    "P2",
		CreatedAt:   now,
	}
	store.CreateMessage(msg)

	if err := store.MarkMessageArchived("msg-1"); err != nil {
		t.Fatalf("MarkMessageArchived failed: %v", err)
	}

	got, _ := store.GetMessage("msg-1")
	if got.Status != MessageStatusArchived {
		t.Errorf("Status not updated to archived")
	}
	if got.ArchivedAt == nil {
		t.Error("ArchivedAt not set")
	}
}

// ===== Aggregate Query Tests =====

func TestStore_GetMetricsSummary(t *testing.T) {
	store := setupTestStore(t)
	defer store.DB().Close()

	now := time.Now().Truncate(time.Second)

	// Create test data
	ws := &Workspace{ID: "ws-1", Name: "Test", Path: "/test", CreatedAt: now, UpdatedAt: now}
	store.CreateWorkspace(ws)

	for i := 0; i < 3; i++ {
		status := TaskStatusCompleted
		if i == 2 {
			status = TaskStatusFailed
		}
		task := &Task{
			ID:          "task-" + string(rune('a'+i)),
			WorkspaceID: "ws-1",
			Title:       "Task",
			SourceType:  TaskSourceManual,
			Status:      status,
			Priority:    "P1",
			CreatedAt:   now,
		}
		store.CreateTask(task)
	}

	span := &Span{
		ID:        "span-1",
		TraceID:   "trace-1",
		Name:      "test",
		Kind:      SpanKindInternal,
		Status:    SpanStatusOK,
		StartTime: now,
		TokensIn:  1000,
		TokensOut: 500,
		CostUSD:   0.05,
		CreatedAt: now,
	}
	store.CreateSpan(span)

	summary, err := store.GetMetricsSummary()
	if err != nil {
		t.Fatalf("GetMetricsSummary failed: %v", err)
	}

	if summary.TotalWorkspaces != 1 {
		t.Errorf("TotalWorkspaces mismatch: got %d", summary.TotalWorkspaces)
	}
	if summary.TotalTasks != 3 {
		t.Errorf("TotalTasks mismatch: got %d", summary.TotalTasks)
	}
	if summary.TotalSpans != 1 {
		t.Errorf("TotalSpans mismatch: got %d", summary.TotalSpans)
	}
	if summary.TotalTokensIn != 1000 {
		t.Errorf("TotalTokensIn mismatch: got %d", summary.TotalTokensIn)
	}
	// 2 completed, 1 failed = 66.67% success rate
	if summary.SuccessRate < 66 || summary.SuccessRate > 67 {
		t.Errorf("SuccessRate mismatch: got %f", summary.SuccessRate)
	}
}
