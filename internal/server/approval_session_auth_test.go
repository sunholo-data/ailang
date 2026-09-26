package server

import (
	"net/http/httptest"
	"testing"

	"github.com/sunholo-data/ailang/internal/server/auth"
)

// #920: with Firebase auth enabled, a dashboard approval action with NO
// secret-approval token and NO session must be rejected — before the fix it
// resolved as "dashboard-user" for any unauthenticated caller.
func TestHandleApproval_NoTokenNoSession_Rejected(t *testing.T) {
	// A non-nil verifier enables auth; the verifier itself is never reached
	// because the request carries no bearer token at all.
	s := &Server{tokenVerifier: auth.NewTokenVerifier(nil)}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/api/approvals/appr-1/approve", nil)
	s.handleApproval(w, r)
	if w.Code != 401 {
		t.Fatalf("unauthenticated approve: got %d, want 401", w.Code)
	}
}

func TestApproverSessionAuthorized_AuthDisabledPasses(t *testing.T) {
	s := &Server{} // no tokenVerifier — auth not configured
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/api/approvals/appr-1/approve", nil)
	if !s.approverSessionAuthorized(w, r) {
		t.Fatal("auth disabled: gate must pass through (same as requireApprover)")
	}
}

func TestApproverSessionAuthorized_NoBearerToken_Rejected(t *testing.T) {
	s := &Server{tokenVerifier: auth.NewTokenVerifier(nil)}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/api/approvals/appr-1/approve", nil)
	if s.approverSessionAuthorized(w, r) {
		t.Fatal("auth enabled, no bearer token: gate must reject")
	}
	if w.Code != 401 {
		t.Fatalf("got %d, want 401", w.Code)
	}
}
