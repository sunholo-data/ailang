package coordinator

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/pubsub"
)

// M-SEC2 SEC2.2 — a completion is bound to its task's dispatch.

const bindingRejected = "COMPLETION_REJECTED"

type bindingFixture struct {
	store   *SQLiteStore
	handler *CompletionHandler
	logs    *bytes.Buffer
}

func newBindingFixture(t *testing.T) *bindingFixture {
	t.Helper()
	store, err := NewSQLiteStore(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	logs := &bytes.Buffer{}
	h := NewCompletionHandler(nil, store, nil, nil, log.New(logs, "", 0))
	return &bindingFixture{store: store, handler: h, logs: logs}
}

func (f *bindingFixture) seed(t *testing.T, id, agentID string, status TaskStatus) {
	t.Helper()
	if err := f.store.CreateTask(context.Background(), &TaskRecord{
		ID:        id,
		Title:     "binding test",
		Content:   "binding test",
		Workspace: "sunholo-data/ailang",
		AgentID:   agentID,
		Status:    status,
		CreatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("create task: %v", err)
	}
}

func (f *bindingFixture) deliver(t *testing.T, c pubsub.TaskCompletion) {
	t.Helper()
	data, err := json.Marshal(c)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	// nil = acked. Every path below must ack: a rejected claim redelivered is
	// still rejected, so a retry loop would only burn deliveries.
	if err := f.handler.HandleCompletion(data, nil); err != nil {
		t.Fatalf("HandleCompletion returned %v; a completion must be acked", err)
	}
}

func (f *bindingFixture) status(t *testing.T, id string) TaskStatus {
	t.Helper()
	task, err := f.store.GetTask(context.Background(), id)
	if err != nil || task == nil {
		t.Fatalf("get task %s: %v", id, err)
	}
	return task.Status
}

func (f *bindingFixture) ledgerLen(t *testing.T, id string) int {
	t.Helper()
	l, err := f.store.GetTaskFinalization(context.Background(), id)
	if err != nil {
		t.Fatalf("ledger: %v", err)
	}
	return len(l)
}

// TestCompletionArrivalTable_CoversEveryStatus: a status added without a row
// here would be refused by default — safe, but every completion for it would be
// dropped as a forgery, so the omission must fail a test rather than a deploy.
func TestCompletionArrivalTable_CoversEveryStatus(t *testing.T) {
	for _, s := range AllTaskStatuses() {
		if _, ok := completionArrivalByStatus[s]; !ok {
			t.Errorf("status %q has no row in completionArrivalByStatus", s)
		}
	}
	// Terminal statuses must be let through to the handler's quiet duplicate
	// skip; refusing them would turn every idempotent redelivery into an alert.
	for _, s := range TerminalStatuses() {
		if !completionMayArriveIn(s) {
			t.Errorf("terminal status %q is refused; a redelivery would be reported as a forgery", s)
		}
	}
	if completionMayArriveIn("some-future-status") {
		t.Error("an unknown status is accepted; it must be refused")
	}
	if completionMayArriveIn(TaskStatusPending) {
		t.Error("pending is accepted; a cloud task is claimed before its job starts, so nothing dispatched can report into pending")
	}
}

func TestCompletionBindingViolation(t *testing.T) {
	queued := &TaskRecord{ID: "t", AgentID: "sprint-executor", Status: TaskStatusQueued}
	cases := []struct {
		name   string
		task   *TaskRecord
		claim  pubsub.TaskCompletion
		reject bool
	}{
		{"matching agent, queued", queued, pubsub.TaskCompletion{TaskID: "t", AgentID: "sprint-executor"}, false},
		{"matching agent, running", &TaskRecord{AgentID: "a", Status: TaskStatusRunning}, pubsub.TaskCompletion{AgentID: "a"}, false},
		{"matching agent, pending_approval (redelivery)", &TaskRecord{AgentID: "a", Status: TaskStatusPendingApproval}, pubsub.TaskCompletion{AgentID: "a"}, false},
		{"matching agent, terminal (duplicate)", &TaskRecord{AgentID: "a", Status: TaskStatusFailed}, pubsub.TaskCompletion{AgentID: "a"}, false},
		{"other agent", queued, pubsub.TaskCompletion{TaskID: "t", AgentID: "external-user-lane"}, true},
		{"agent differs only by case", queued, pubsub.TaskCompletion{AgentID: "Sprint-Executor"}, true},
		{"empty claimed agent", queued, pubsub.TaskCompletion{TaskID: "t"}, true},
		{"empty task agent, empty claim", &TaskRecord{Status: TaskStatusQueued}, pubsub.TaskCompletion{}, true},
		{"empty task agent, some claim", &TaskRecord{Status: TaskStatusQueued}, pubsub.TaskCompletion{AgentID: "a"}, true},
		{"matching agent, pending", &TaskRecord{AgentID: "a", Status: TaskStatusPending}, pubsub.TaskCompletion{AgentID: "a"}, true},
		{"matching agent, unknown status", &TaskRecord{AgentID: "a", Status: "weird"}, pubsub.TaskCompletion{AgentID: "a"}, true},
		{"nil task", nil, pubsub.TaskCompletion{AgentID: "a"}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := completionBindingViolation(tc.task, tc.claim)
			if tc.reject && got == "" {
				t.Error("accepted; want rejected")
			}
			if !tc.reject && got != "" {
				t.Errorf("rejected (%s); want accepted", got)
			}
		})
	}
}

// A completion from another agent must not touch the task: not its status,
// not its ledger — and it is said loudly, with both identities.
func TestHandleCompletion_RejectsMismatchedAgent(t *testing.T) {
	f := newBindingFixture(t)
	f.seed(t, "task-victim", "sprint-evaluator", TaskStatusQueued)

	f.deliver(t, pubsub.TaskCompletion{TaskID: "task-victim", AgentID: "website-builder", Status: "failed", ErrorMsg: "forced"})

	if got := f.status(t, "task-victim"); got != TaskStatusQueued {
		t.Errorf("status = %q, want queued (unchanged)", got)
	}
	if n := f.ledgerLen(t, "task-victim"); n != 0 {
		t.Errorf("ledger has %d rows; a rejected claim must not start finalisation", n)
	}
	out := f.logs.String()
	for _, want := range []string{bindingRejected, "task-victim", `claimed_agent="website-builder"`, `expected_agent="sprint-evaluator"`} {
		if !strings.Contains(out, want) {
			t.Errorf("log lacks %q:\n%s", want, out)
		}
	}
}

func TestHandleCompletion_RejectsEmptyAgent(t *testing.T) {
	f := newBindingFixture(t)
	f.seed(t, "task-e", "sprint-executor", TaskStatusQueued)

	f.deliver(t, pubsub.TaskCompletion{TaskID: "task-e", Status: "completed"})

	if got := f.status(t, "task-e"); got != TaskStatusQueued {
		t.Errorf("status = %q, want queued (unchanged)", got)
	}
	if !strings.Contains(f.logs.String(), bindingRejected) {
		t.Errorf("empty agent not rejected loudly:\n%s", f.logs.String())
	}
}

// A task nobody dispatched — pending, or a local-lane task sitting in the
// shared store — cannot be completed from the cloud completions topic.
func TestHandleCompletion_RejectsUndispatchedTask(t *testing.T) {
	f := newBindingFixture(t)
	f.seed(t, "task-p", "eval-rig", TaskStatusPending)

	f.deliver(t, pubsub.TaskCompletion{TaskID: "task-p", AgentID: "eval-rig", Status: "completed"})

	if got := f.status(t, "task-p"); got != TaskStatusPending {
		t.Errorf("status = %q, want pending (unchanged)", got)
	}
	if n := f.ledgerLen(t, "task-p"); n != 0 {
		t.Errorf("ledger has %d rows; want none", n)
	}
	if !strings.Contains(f.logs.String(), bindingRejected) {
		t.Errorf("pending-task completion not rejected loudly:\n%s", f.logs.String())
	}
}

// Order: identity is checked before the terminal skip, so a forged claim
// against a finished task is still reported rather than passing as a duplicate.
func TestHandleCompletion_MismatchOnTerminalTaskIsStillReported(t *testing.T) {
	f := newBindingFixture(t)
	f.seed(t, "task-done", "sprint-planner", TaskStatusFailed)

	f.deliver(t, pubsub.TaskCompletion{TaskID: "task-done", AgentID: "someone-else", Status: "completed"})

	if got := f.status(t, "task-done"); got != TaskStatusFailed {
		t.Errorf("status = %q, want failed (unchanged)", got)
	}
	if !strings.Contains(f.logs.String(), bindingRejected) {
		t.Errorf("mismatch on a terminal task not reported:\n%s", f.logs.String())
	}
}

// The legitimate path: the agent the task was dispatched to reports from
// queued (every cloud task) or running.
func TestHandleCompletion_AcceptsMatchingAgent(t *testing.T) {
	cases := []struct {
		from   TaskStatus
		claim  string
		expect TaskStatus
	}{
		{TaskStatusQueued, "completed", TaskStatusPendingApproval},
		{TaskStatusQueued, "failed", TaskStatusFailed},
		{TaskStatusQueued, "no_changes", TaskStatusNoChanges},
		{TaskStatusQueued, "blocked", TaskStatusBlocked},
		{TaskStatusRunning, "completed", TaskStatusPendingApproval},
	}
	for _, tc := range cases {
		t.Run(string(tc.from)+"->"+tc.claim, func(t *testing.T) {
			f := newBindingFixture(t)
			f.seed(t, "task-ok", "sprint-executor", tc.from)

			f.deliver(t, pubsub.TaskCompletion{TaskID: "task-ok", AgentID: "sprint-executor", Status: tc.claim})

			if got := f.status(t, "task-ok"); got != tc.expect {
				t.Errorf("status = %q, want %q", got, tc.expect)
			}
			if strings.Contains(f.logs.String(), bindingRejected) {
				t.Errorf("legitimate completion rejected:\n%s", f.logs.String())
			}
		})
	}
}

// Redelivery of a legitimate completion stays idempotent and quiet: into
// pending_approval (finalisation re-runs and supersedes itself) and into a
// terminal state (skipped as a duplicate). Neither is a rejection.
func TestHandleCompletion_RedeliveryIsIdempotentNotRejected(t *testing.T) {
	cases := []struct {
		claim  string
		settle TaskStatus
	}{
		{"completed", TaskStatusPendingApproval},
		{"failed", TaskStatusFailed},
	}
	for _, tc := range cases {
		t.Run(tc.claim, func(t *testing.T) {
			f := newBindingFixture(t)
			f.seed(t, "task-r", "design-doc-creator", TaskStatusQueued)
			c := pubsub.TaskCompletion{TaskID: "task-r", AgentID: "design-doc-creator", Status: tc.claim, BranchName: "coordinator/task-r"}

			f.deliver(t, c)
			if got := f.status(t, "task-r"); got != tc.settle {
				t.Fatalf("first delivery: status = %q, want %q", got, tc.settle)
			}
			f.deliver(t, c)
			if got := f.status(t, "task-r"); got != tc.settle {
				t.Errorf("redelivery moved status to %q, want %q", got, tc.settle)
			}
			if strings.Contains(f.logs.String(), bindingRejected) {
				t.Errorf("redelivery of a legitimate completion was rejected:\n%s", f.logs.String())
			}
		})
	}
}

// The push endpoint acks a rejected claim (200) — a 5xx would have Pub/Sub
// redeliver a forgery until the dead-letter topic takes it.
func TestHandlePushCompletion_RejectedClaimIsAcked(t *testing.T) {
	f := newBindingFixture(t)
	f.seed(t, "task-push", "sprint-evaluator", TaskStatusQueued)
	d := &Daemon{logger: log.New(&bytes.Buffer{}, "", 0), completionHandler: f.handler}

	body := makePushBody(t, pubsub.TaskCompletion{TaskID: "task-push", AgentID: "intruder", Status: "completed"}, nil)
	req := httptest.NewRequest(http.MethodPost, "/pubsub/completions", strings.NewReader(body))
	w := httptest.NewRecorder()
	d.handlePushCompletion(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("push status = %d, want 200 (ack)", w.Code)
	}
	if got := f.status(t, "task-push"); got != TaskStatusQueued {
		t.Errorf("status = %q, want queued (unchanged)", got)
	}
	if !strings.Contains(f.logs.String(), bindingRejected) {
		t.Errorf("push-path mismatch not rejected loudly:\n%s", f.logs.String())
	}
}
