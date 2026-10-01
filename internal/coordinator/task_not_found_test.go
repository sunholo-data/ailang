package coordinator

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/pubsub"
)

func newTaskNotFoundTestStore(t *testing.T) *SQLiteStore {
	t.Helper()
	store, err := NewSQLiteStore(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}

// GetTask on an unknown ID reports ErrTaskNotFound, and keeps sql.ErrNoRows in
// the chain so a caller that predates the sentinel still matches.
func TestSQLiteGetTask_UnknownIDIsErrTaskNotFound(t *testing.T) {
	store := newTaskNotFoundTestStore(t)
	task, err := store.GetTask(context.Background(), "no-such-task")
	if task != nil {
		t.Fatalf("task = %+v, want nil", task)
	}
	if !errors.Is(err, ErrTaskNotFound) {
		t.Fatalf("err = %v, want ErrTaskNotFound", err)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("err = %v, want sql.ErrNoRows still in the chain", err)
	}
	if !strings.Contains(err.Error(), "no-such-task") {
		t.Fatalf("err = %q, want the task ID in the message", err)
	}
}

func completionBytes(t *testing.T, taskID string) []byte {
	t.Helper()
	data, err := json.Marshal(pubsub.TaskCompletion{TaskID: taskID, AgentID: "a", Status: "completed"})
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// A completion for a task this coordinator has never seen is acked. Before the
// sentinel, both stores' not-found error was returned, the push endpoint
// answered 500, and Pub/Sub redelivered the orphan for 24h.
func TestHandleCompletion_UnknownTaskIsAcked(t *testing.T) {
	logger := log.New(io.Discard, "", 0)

	t.Run("sqlite store", func(t *testing.T) {
		h := NewCompletionHandler(nil, newTaskNotFoundTestStore(t), nil, nil, logger)
		if err := h.HandleCompletion(completionBytes(t, "sec2probe-unknown"), nil); err != nil {
			t.Fatalf("HandleCompletion = %v, want nil (ack)", err)
		}
	})

	t.Run("store returning the wrapped sentinel", func(t *testing.T) {
		store := NewMockStore()
		store.getTaskErr = fmt.Errorf("%w: %s", ErrTaskNotFound, "sec2probe-unknown")
		h := NewCompletionHandler(nil, store, nil, nil, logger)
		if err := h.HandleCompletion(completionBytes(t, "sec2probe-unknown"), nil); err != nil {
			t.Fatalf("HandleCompletion = %v, want nil (ack)", err)
		}
	})

	t.Run("store failure is still retried", func(t *testing.T) {
		store := NewMockStore()
		store.getTaskErr = errors.New("firestore unavailable")
		h := NewCompletionHandler(nil, store, nil, nil, logger)
		if err := h.HandleCompletion(completionBytes(t, "t1"), nil); err == nil {
			t.Fatal("HandleCompletion = nil, want an error so Pub/Sub redelivers")
		}
	})
}

// The push endpoint answers 200 for an unknown task, which is what stops the
// redelivery loop.
func TestHandlePushCompletion_UnknownTaskReturns200(t *testing.T) {
	logger := log.New(io.Discard, "", 0)
	d := &Daemon{
		logger:            logger,
		completionHandler: NewCompletionHandler(nil, newTaskNotFoundTestStore(t), nil, nil, logger),
	}
	body := makePushBody(t, pubsub.TaskCompletion{TaskID: "sec2probe-unknown", AgentID: "a", Status: "completed"}, nil)
	req := httptest.NewRequest(http.MethodPost, "/pubsub/completions", strings.NewReader(body))
	w := httptest.NewRecorder()

	d.handlePushCompletion(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
}
