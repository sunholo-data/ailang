package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/coordinator"
	"github.com/sunholo-data/ailang/internal/messaging"
)

// Does a DASHBOARD approval start the next stage?
//
// The CLI path was fixed on 2026-09-07 (ProcessApprovalRequest now fires the
// gated handoffs), and pipeline_stage_crossing_test.go proves it at the
// processor. But /api/coordinator/{action}/{id} never called the processor: it
// resolved the approval record by hand and returned {success:true}. So an
// approval clicked on that route recorded "approved", dispatched nothing, and
// left the task pending_approval — indistinguishable from the fixed path in
// every response field. This test follows the work across the HTTP seam.

// dashboardApprovalFixture is one approved stage-1 task behind a Server wired
// the way cmd/ailang/server.go wires it: the same SQLite coordinator store as
// both approvalStore and coordStoreRaw, and a real messaging store.
type dashboardApprovalFixture struct {
	srv      *Server
	store    *coordinator.SQLiteStore
	msgStore *messaging.Store
	task     *coordinator.TaskRecord
}

func newDashboardApprovalFixture(t *testing.T) *dashboardApprovalFixture {
	t.Helper()
	dir := t.TempDir()

	// The processor loads the agent registry from AILANG_CONFIG and the
	// messaging/GitHub config from $HOME/.ailang/config.yaml. Point both at
	// this test so a developer's real config (and a real Pub/Sub topic) is
	// never touched. With no messaging config the handoff row is stored and
	// the notify step reports "not dispatched" — the row is the observable.
	t.Setenv("HOME", dir)
	cfgPath := filepath.Join(dir, "coordinator.yaml")
	cfg := `coordinator:
  agents:
    - id: design-doc-creator
      label: Design Doc Creator
      inbox: design-doc-creator
      workspace: ` + dir + `
      trigger_on_complete: [sprint-planner]
    - id: sprint-planner
      label: Sprint Planner
      inbox: sprint-planner
      workspace: ` + dir + `
`
	if err := os.WriteFile(cfgPath, []byte(cfg), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	t.Setenv("AILANG_CONFIG", cfgPath)

	store, err := coordinator.NewSQLiteStore(filepath.Join(dir, "coordinator.db"))
	if err != nil {
		t.Fatalf("coordinator store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	msgStore, err := messaging.OpenStore(filepath.Join(dir, "collaboration.db"))
	if err != nil {
		t.Fatalf("message store: %v", err)
	}
	t.Cleanup(func() { _ = msgStore.Close() })

	// No worktree: the branch is already pushed, so approval completes the
	// task without a local merge and no git runs in this test.
	task := &coordinator.TaskRecord{
		ID:        "task-d45hb0ad",
		AgentID:   "design-doc-creator",
		Title:     "Design: dashboard approval handoff",
		Content:   "Design the thing",
		Type:      coordinator.TaskTypeFeature,
		Status:    coordinator.TaskStatusPendingApproval,
		Workspace: dir,
		CreatedAt: time.Now().Add(-5 * time.Minute),
	}
	ctx := context.Background()
	if err := store.CreateTask(ctx, task); err != nil {
		t.Fatalf("create task: %v", err)
	}
	if err := store.CreateApprovalRequest(ctx, &coordinator.ApprovalRequestRecord{
		ID:          coordinator.ApprovalIDForTask(task.ID),
		TaskID:      task.ID,
		Type:        string(coordinator.ApprovalTypeMerge),
		Description: "Agent completed work on: " + task.Title,
		Status:      "pending",
		ContextJSON: `{"handoff_targets":["sprint-planner"],"source_agent":"design-doc-creator"}`,
		CreatedAt:   time.Now(),
	}); err != nil {
		t.Fatalf("create approval record: %v", err)
	}

	srv := &Server{store: msgStore}
	srv.SetApprovalStore(store)
	srv.SetCoordinatorStoreRaw(store)

	return &dashboardApprovalFixture{srv: srv, store: store, msgStore: msgStore, task: task}
}

func (f *dashboardApprovalFixture) post(t *testing.T, action string) *httptest.ResponseRecorder {
	t.Helper()
	id := coordinator.ApprovalIDForTask(f.task.ID)
	req := httptest.NewRequest(http.MethodPost, "/api/coordinator/"+action+"/"+id, nil)
	w := httptest.NewRecorder()
	f.srv.handleCoordinatorApproval(w, req)
	return w
}

func TestDashboardApproval_FiresTheHandoff(t *testing.T) {
	f := newDashboardApprovalFixture(t)
	ctx := context.Background()

	w := f.post(t, "approve")
	if w.Code != http.StatusOK {
		t.Fatalf("approve: status %d: %s", w.Code, w.Body.String())
	}

	// The response shape the documented curl route promises.
	var resp struct {
		Success   bool   `json:"success"`
		Action    string `json:"action"`
		ID        string `json:"id"`
		Timestamp int64  `json:"timestamp"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("response is not the {success,action,id,timestamp} shape: %v: %s", err, w.Body.String())
	}
	if !resp.Success || resp.Action != "approve" || resp.ID != coordinator.ApprovalIDForTask(f.task.ID) || resp.Timestamp == 0 {
		t.Errorf("response shape changed: %+v", resp)
	}

	// THE INBOX of the next stage — where dispatch actually looks. This is the
	// assertion the 2026-09-07 fix never had on this route.
	msgs, err := f.msgStore.ListInboxMessages(messaging.InboxListOptions{Inbox: "sprint-planner"})
	if err != nil {
		t.Fatalf("list sprint-planner inbox: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("dashboard approval put %d messages in sprint-planner's inbox, want 1 — the pipeline stops here", len(msgs))
	}
	got := msgs[0]
	if got.MessageType != "handoff" {
		t.Errorf("message_type = %q, want handoff", got.MessageType)
	}
	if got.ParentTaskID != f.task.ID {
		t.Errorf("parent_task_id = %q, want %s", got.ParentTaskID, f.task.ID)
	}
	if !strings.Contains(got.Payload, "Handoff from Design Doc Creator") {
		t.Errorf("payload does not name the source agent:\n%s", got.Payload)
	}

	// And the record and task both moved: approved, and off pending_approval.
	rec, err := f.store.GetApprovalRequest(ctx, coordinator.ApprovalIDForTask(f.task.ID))
	if err != nil {
		t.Fatalf("get approval: %v", err)
	}
	if rec.Status != "approved" || rec.ResolvedBy != "dashboard-user" {
		t.Errorf("approval record = %s by %q, want approved by dashboard-user", rec.Status, rec.ResolvedBy)
	}
	task, err := f.store.GetTask(ctx, f.task.ID)
	if err != nil {
		t.Fatalf("get task: %v", err)
	}
	if task.Status == coordinator.TaskStatusPendingApproval {
		t.Errorf("task still pending_approval after a dashboard approval — the old stranding signature")
	}

	// A second click is refused, not re-run: the handoff must fire once.
	if w2 := f.post(t, "approve"); w2.Code != http.StatusConflict {
		t.Errorf("second approve: status %d, want 409", w2.Code)
	}
	msgs, _ = f.msgStore.ListInboxMessages(messaging.InboxListOptions{Inbox: "sprint-planner"})
	if len(msgs) != 1 {
		t.Errorf("second approve changed the inbox: %d messages", len(msgs))
	}
}

func TestDashboardReject_GoesThroughTheProcessor(t *testing.T) {
	f := newDashboardApprovalFixture(t)
	ctx := context.Background()

	w := f.post(t, "reject")
	if w.Code != http.StatusOK {
		t.Fatalf("reject: status %d: %s", w.Code, w.Body.String())
	}
	rec, err := f.store.GetApprovalRequest(ctx, coordinator.ApprovalIDForTask(f.task.ID))
	if err != nil {
		t.Fatalf("get approval: %v", err)
	}
	if rec.Status != "rejected" {
		t.Errorf("approval record = %s, want rejected", rec.Status)
	}
	// A rejection hands nothing to the next stage.
	msgs, _ := f.msgStore.ListInboxMessages(messaging.InboxListOptions{Inbox: "sprint-planner"})
	if len(msgs) != 0 {
		t.Errorf("reject put %d messages in sprint-planner's inbox, want 0", len(msgs))
	}
}

// The old route resolved the record with only approvalStore when the
// coordinator store was absent — an approval that could never hand off. That
// is now a loud 503, not a quiet half-approval.
func TestDashboardApproval_NoCoordinatorStoreIsLoud(t *testing.T) {
	approvalStore := NewMockApprovalStore()
	approvalStore.approvals["apr-1"] = &coordinator.ApprovalRequestRecord{
		ID: "apr-1", TaskID: "task-1", Status: "pending", CreatedAt: time.Now(),
	}
	s := &Server{}
	s.SetApprovalStore(approvalStore)

	req := httptest.NewRequest(http.MethodPost, "/api/coordinator/approve/apr-1", nil)
	w := httptest.NewRecorder()
	s.handleCoordinatorApproval(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status %d, want 503", w.Code)
	}
	if approvalStore.approvals["apr-1"].Status != "pending" {
		t.Errorf("record was resolved to %q without the processor", approvalStore.approvals["apr-1"].Status)
	}
}
