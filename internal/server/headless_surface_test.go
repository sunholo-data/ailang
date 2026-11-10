package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/coordinator"
)

// The handlers that lost their UI-only branches (m-dashboard-headless-api):
// what is left must refuse what was removed, plainly.

func TestApprovals_ListIsGone_IntakeOnly(t *testing.T) {
	s, _ := newSecretApprovalServer()
	for _, m := range []string{http.MethodGet, http.MethodPut, http.MethodDelete} {
		w := httptest.NewRecorder()
		s.handleApprovals(w, httptest.NewRequest(m, "/api/approvals?status=all", nil))
		if w.Code != http.StatusMethodNotAllowed || w.Header().Get("Allow") != http.MethodPost {
			t.Errorf("%s /api/approvals: got %d Allow=%q, want 405 Allow=POST", m, w.Code, w.Header().Get("Allow"))
		}
	}
}

func TestApproval_TaskApprovalPointsAtTheCLI(t *testing.T) {
	s, store := newSecretApprovalServer()
	store.approvals["apr-1"] = &coordinator.ApprovalRequestRecord{ID: "apr-1", Type: "merge", Status: "pending"}

	w := httptest.NewRecorder()
	s.handleApproval(w, httptest.NewRequest(http.MethodPost, "/api/approvals/apr-1/approve", nil))
	if w.Code != http.StatusConflict || !strings.Contains(w.Body.String(), "ailang coordinator approve") {
		t.Fatalf("task approval over HTTP: got %d %q, want 409 naming the CLI", w.Code, w.Body.String())
	}
	if got := store.approvals["apr-1"].Status; got != "pending" {
		t.Fatalf("a task approval must not be resolved over HTTP; status became %q", got)
	}
}

func TestApproval_DecisionErrors(t *testing.T) {
	s, _ := newSecretApprovalServer()
	cases := []struct {
		method, target string
		want           int
	}{
		{http.MethodPost, "/api/approvals/missing/approve", http.StatusNotFound},
		{http.MethodPost, "/api/approvals/x/merge", http.StatusBadRequest},
		{http.MethodPost, "/api/approvals/", http.StatusBadRequest},
		{http.MethodGet, "/api/approvals/x/approve", http.StatusMethodNotAllowed},
		{http.MethodDelete, "/api/approvals/x", http.StatusMethodNotAllowed},
	}
	for _, tc := range cases {
		w := httptest.NewRecorder()
		s.handleApproval(w, httptest.NewRequest(tc.method, tc.target, nil))
		if w.Code != tc.want {
			t.Errorf("%s %s: got %d, want %d", tc.method, tc.target, w.Code, tc.want)
		}
	}

	noStore := &Server{}
	w := httptest.NewRecorder()
	noStore.handleApproval(w, httptest.NewRequest(http.MethodPost, "/api/approvals/x/approve", nil))
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("no approval store: got %d, want 503", w.Code)
	}
}

func TestInbox_ReadOnly(t *testing.T) {
	s := &Server{}
	for _, m := range []string{http.MethodPost, http.MethodPut, http.MethodDelete} {
		w := httptest.NewRecorder()
		s.handleInbox(w, httptest.NewRequest(m, "/api/inbox", nil))
		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("%s /api/inbox: got %d, want 405", m, w.Code)
		}
	}
}

func TestTaskEvents_OnlyTheEventsAction(t *testing.T) {
	s := &Server{}
	s.SetTaskEventStore(NewMockTaskEventStore())
	for _, target := range []string{
		"/api/coordinator/tasks/t-1/diff", "/api/coordinator/tasks/t-1", "/api/coordinator/tasks/t-1/events/extra",
	} {
		w := httptest.NewRecorder()
		s.handleCoordinatorTaskEvents_(w, httptest.NewRequest(http.MethodGet, target, nil))
		if w.Code != http.StatusNotFound {
			t.Errorf("GET %s: got %d, want 404", target, w.Code)
		}
	}
}
