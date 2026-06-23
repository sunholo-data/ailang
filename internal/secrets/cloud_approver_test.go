package secrets

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// secretApprover mirrors effects.SecretApprover so we can assert CloudSecretApprover
// satisfies the interface shape without importing internal/effects (import cycle).
type secretApprover interface {
	Approve(ctx context.Context, ref, purpose string) error
}

var _ secretApprover = (*CloudSecretApprover)(nil)

// fakeCoordinator is an httptest server that records the create request and
// serves a scripted decision after a given number of polls.
type fakeCoordinator struct {
	mu          sync.Mutex
	createBody  []byte
	createCount int
	pollCount   int
	decideAfter int    // polls returning "pending" before the decision
	decision    string // "approved" | "denied"
	reason      string
	createCode  int // status code for POST (0 → 200)
}

func (f *fakeCoordinator) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/approvals", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		body, _ := io.ReadAll(r.Body)
		f.mu.Lock()
		f.createBody = body
		f.createCount++
		code := f.createCode
		f.mu.Unlock()
		if code != 0 {
			w.WriteHeader(code)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"id": "appr-1"})
	})
	mux.HandleFunc("/api/approvals/", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		f.pollCount++
		n := f.pollCount
		f.mu.Unlock()
		resp := map[string]string{"status": "pending"}
		if n > f.decideAfter {
			resp["status"] = f.decision
			if f.reason != "" {
				resp["reason"] = f.reason
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})
	return mux
}

func newApproverFor(t *testing.T, srv *httptest.Server) *CloudSecretApprover {
	t.Helper()
	return NewCloudSecretApprover(srv.URL,
		WithApproverIdentity("agent-x", "task-42"),
		WithApproverDeadline(2*time.Second),
		WithApproverPollInterval(5*time.Millisecond),
		WithApproverHTTPClient(srv.Client()),
	)
}

func TestCloudApprover_Approved(t *testing.T) {
	fc := &fakeCoordinator{decideAfter: 1, decision: "approved"}
	srv := httptest.NewServer(fc.handler())
	defer srv.Close()

	if err := newApproverFor(t, srv).Approve(context.Background(), "op://Prod/stripe/key", "charge"); err != nil {
		t.Fatalf("expected approval, got error: %v", err)
	}
}

func TestCloudApprover_Denied(t *testing.T) {
	// The coordinator store resolves to "rejected" (not "denied").
	fc := &fakeCoordinator{decideAfter: 0, decision: "rejected", reason: "not now"}
	srv := httptest.NewServer(fc.handler())
	defer srv.Close()

	err := newApproverFor(t, srv).Approve(context.Background(), "op://Prod/stripe/key", "charge")
	if err == nil {
		t.Fatal("expected denial error, got nil")
	}
	if !strings.Contains(err.Error(), "rejected") {
		t.Errorf("error should reflect the rejected status: %v", err)
	}
}

func TestCloudApprover_Timeout(t *testing.T) {
	fc := &fakeCoordinator{decideAfter: 1 << 30, decision: "approved"} // never decides
	srv := httptest.NewServer(fc.handler())
	defer srv.Close()

	a := NewCloudSecretApprover(srv.URL,
		WithApproverDeadline(80*time.Millisecond),
		WithApproverPollInterval(5*time.Millisecond),
		WithApproverHTTPClient(srv.Client()),
	)
	err := a.Approve(context.Background(), "op://Prod/x/y", "use")
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Errorf("error should mention timeout: %v", err)
	}
}

// TestCloudApprover_RequestIsValueFree asserts the approval request carries the
// reference/purpose/agent but never a resolved secret value.
func TestCloudApprover_RequestIsValueFree(t *testing.T) {
	fc := &fakeCoordinator{decideAfter: 0, decision: "approved"}
	srv := httptest.NewServer(fc.handler())
	defer srv.Close()

	if err := newApproverFor(t, srv).Approve(context.Background(), "op://Prod/stripe/key", "charge customer"); err != nil {
		t.Fatalf("approve: %v", err)
	}

	fc.mu.Lock()
	body := string(fc.createBody)
	fc.mu.Unlock()

	var got map[string]any
	if err := json.Unmarshal([]byte(body), &got); err != nil {
		t.Fatalf("create body not JSON: %v (%s)", err, body)
	}
	if got["ref"] != "op://Prod/stripe/key" {
		t.Errorf("expected ref in request, got %v", got["ref"])
	}
	if got["purpose"] != "charge customer" {
		t.Errorf("expected purpose in request, got %v", got["purpose"])
	}
	if got["agent"] != "agent-x" {
		t.Errorf("expected agent in request, got %v", got["agent"])
	}
	if _, leaked := got["value"]; leaked {
		t.Errorf("approval request must not contain a value field: %s", body)
	}
}

// TestCloudApprover_CreateFailureFailsClosed: if the coordinator rejects the
// request, Approve must deny (fail closed), not silently allow.
func TestCloudApprover_CreateFailureFailsClosed(t *testing.T) {
	fc := &fakeCoordinator{createCode: http.StatusInternalServerError}
	srv := httptest.NewServer(fc.handler())
	defer srv.Close()

	err := newApproverFor(t, srv).Approve(context.Background(), "op://Prod/x/y", "use")
	if err == nil {
		t.Fatal("expected error when coordinator rejects the request (fail closed), got nil")
	}
}

// TestCloudApprover_ContextCanceled: a canceled context aborts the wait.
func TestCloudApprover_ContextCanceled(t *testing.T) {
	fc := &fakeCoordinator{decideAfter: 1 << 30, decision: "approved"}
	srv := httptest.NewServer(fc.handler())
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	go func() { time.Sleep(40 * time.Millisecond); cancel() }()

	a := NewCloudSecretApprover(srv.URL,
		WithApproverDeadline(5*time.Second),
		WithApproverPollInterval(5*time.Millisecond),
		WithApproverHTTPClient(srv.Client()),
	)
	if err := a.Approve(ctx, "op://Prod/x/y", "use"); err == nil {
		t.Fatal("expected cancellation error, got nil")
	}
}
