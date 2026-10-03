package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"google.golang.org/api/idtoken"

	"github.com/sunholo-data/ailang/internal/coordinator"
)

const (
	testDashURL  = "https://ailang-dev-dashboard-abc-ew.a.run.app"
	testAgentSA  = "ailang-dev-agent@ailang-multivac-dev.iam.gserviceaccount.com"
	testOtherSA  = "someone-else@other-project.iam.gserviceaccount.com"
	testExtLane  = "ailang-dev-agent-external@ailang-multivac-dev.iam.gserviceaccount.com"
	testBadToken = "not-a-google-token"
)

// fakeIDTokens stands in for idtoken.Validate: each token string maps to the
// payload a genuinely Google-signed token would carry, and anything else fails
// signature verification. Audience/issuer/allowlist are left to the code under
// test, which is the point.
func fakeIDTokens() idTokenValidator {
	payload := func(iss, aud, email string, verified bool) *idtoken.Payload {
		return &idtoken.Payload{Issuer: iss, Audience: aud, Claims: map[string]interface{}{"email": email, "email_verified": verified}}
	}
	tokens := map[string]*idtoken.Payload{
		"valid-agent":    payload("https://accounts.google.com", testDashURL, testAgentSA, true),
		"valid-external": payload("accounts.google.com", testDashURL+"/", testExtLane, true),
		"wrong-audience": payload("https://accounts.google.com", "https://evil.example", testAgentSA, true),
		"wrong-issuer":   payload("https://cloud.google.com/iap", testDashURL, testAgentSA, true),
		"not-allowed":    payload("https://accounts.google.com", testDashURL, testOtherSA, true),
		"unverified":     payload("https://accounts.google.com", testDashURL, testAgentSA, false),
	}
	return func(_ context.Context, tok, audience string) (*idtoken.Payload, error) {
		if audience != "" {
			return nil, errors.New("test: the server must check audience itself (list of audiences)")
		}
		if p, ok := tokens[tok]; ok {
			return p, nil
		}
		return nil, errors.New("idtoken: invalid signature")
	}
}

// approvalAuthEnv sets every variable the gate reads, so a developer's shell
// cannot leak into the test. The dashboard URL arrives via the default
// (AILANG_APPROVAL_BASE_URL, with a trailing slash to prove normalization).
func approvalAuthEnv(t *testing.T, sharedToken, allowed string) {
	t.Helper()
	t.Setenv("AILANG_APPROVAL_INTAKE_AUTH", "")
	t.Setenv("AILANG_APPROVAL_TOKEN", sharedToken)
	t.Setenv("AILANG_APPROVAL_AUDIENCE", "")
	t.Setenv("AILANG_APPROVAL_BASE_URL", testDashURL+"/")
	t.Setenv("AILANG_APPROVAL_ALLOWED_CALLERS", allowed)
}

func newAuthApprovalServer() (*Server, *MockApprovalStore) {
	s, store := newSecretApprovalServer()
	s.idTokenValidate = fakeIDTokens()
	return s, store
}

func postIntake(s *Server, bearer string) *httptest.ResponseRecorder {
	body, _ := json.Marshal(secretIntakeRequest{Ref: "op://Prod/stripe/key", Purpose: "charge", Agent: "agent-x"})
	req := httptest.NewRequest(http.MethodPost, "/api/approvals", bytes.NewReader(body))
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	w := httptest.NewRecorder()
	s.handleApprovals(w, req)
	return w
}

// TestApprovalIntakeAuth_Matrix: the create route admits exactly a
// Google-signed ID token for this dashboard from an allowlisted, verified
// service account — 401 for a missing or bad credential, 403 for a real
// identity that is not allowed — and creates nothing when it refuses.
func TestApprovalIntakeAuth_Matrix(t *testing.T) {
	approvalAuthEnv(t, "", testAgentSA+", "+testExtLane)
	cases := []struct {
		name   string
		bearer string
		want   int
	}{
		{"no token", "", http.StatusUnauthorized},
		{"bad signature", testBadToken, http.StatusUnauthorized},
		{"wrong audience", "wrong-audience", http.StatusUnauthorized},
		{"wrong issuer (IAP assertion)", "wrong-issuer", http.StatusUnauthorized},
		{"SA not in allowlist", "not-allowed", http.StatusForbidden},
		{"unverified email", "unverified", http.StatusForbidden},
		{"valid agent SA", "valid-agent", http.StatusCreated},
		{"valid external-lane SA, aud with trailing slash", "valid-external", http.StatusCreated},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s, store := newAuthApprovalServer()
			w := postIntake(s, tc.bearer)
			if w.Code != tc.want {
				t.Fatalf("expected %d, got %d (%s)", tc.want, w.Code, w.Body.String())
			}
			if tc.want != http.StatusCreated && len(store.approvals) != 0 {
				t.Fatalf("a refused create must not write a record, got %d", len(store.approvals))
			}
			if tc.want == http.StatusUnauthorized && w.Header().Get("WWW-Authenticate") == "" {
				t.Error("401 should carry WWW-Authenticate")
			}
		})
	}
}

// TestApprovalIntakeAuth_RecordsVerifiedCaller: the record carries the
// identity the server verified, not just the agent name the client claimed.
func TestApprovalIntakeAuth_RecordsVerifiedCaller(t *testing.T) {
	approvalAuthEnv(t, "", testAgentSA)
	s, store := newAuthApprovalServer()
	w := postIntake(s, "valid-agent")
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d (%s)", w.Code, w.Body.String())
	}
	var resp map[string]string
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	var sc secretApprovalContext
	_ = json.Unmarshal([]byte(store.approvals[resp["id"]].ContextJSON), &sc)
	if sc.Caller != testAgentSA {
		t.Fatalf("expected caller %s recorded, got %q", testAgentSA, sc.Caller)
	}
}

// TestApprovalIntakeAuth_SharedToken: AILANG_APPROVAL_TOKEN is actually
// verified (it was sent and ignored before). Right value → 201; wrong value →
// 401, including when ID-token auth is unconfigured.
func TestApprovalIntakeAuth_SharedToken(t *testing.T) {
	approvalAuthEnv(t, "s3cret-shared", "")
	s, _ := newAuthApprovalServer()
	if w := postIntake(s, "s3cret-shared"); w.Code != http.StatusCreated {
		t.Fatalf("matching shared token: expected 201, got %d (%s)", w.Code, w.Body.String())
	}
	if w := postIntake(s, "s3cret-sharedX"); w.Code != http.StatusUnauthorized {
		t.Fatalf("wrong shared token: expected 401, got %d", w.Code)
	}
	// A valid ID token is still refused when no allowlist is configured.
	if w := postIntake(s, "valid-agent"); w.Code != http.StatusUnauthorized {
		t.Fatalf("ID token with no allowlist: expected 401, got %d", w.Code)
	}
}

// TestApprovalIntakeAuth_FailsClosedUnconfigured: nothing configured means
// nobody, not everybody — the bug this file fixes was the opposite.
func TestApprovalIntakeAuth_FailsClosedUnconfigured(t *testing.T) {
	approvalAuthEnv(t, "", "")
	t.Setenv("AILANG_APPROVAL_BASE_URL", "")
	s, _ := newAuthApprovalServer()
	for _, bearer := range []string{"", "anything", "valid-agent"} {
		if w := postIntake(s, bearer); w.Code != http.StatusUnauthorized {
			t.Fatalf("bearer %q with nothing configured: expected 401, got %d", bearer, w.Code)
		}
	}
}

// TestApprovalIntakeAuth_OffSwitch: AILANG_APPROVAL_INTAKE_AUTH=off is the
// only way to admit an anonymous create; a typo still enforces.
func TestApprovalIntakeAuth_OffSwitch(t *testing.T) {
	approvalAuthEnv(t, "", "")
	t.Setenv("AILANG_APPROVAL_INTAKE_AUTH", "of")
	s, _ := newAuthApprovalServer()
	if w := postIntake(s, ""); w.Code != http.StatusUnauthorized {
		t.Fatalf("typo'd switch must enforce: expected 401, got %d", w.Code)
	}
	t.Setenv("AILANG_APPROVAL_INTAKE_AUTH", "OFF")
	if w := postIntake(s, ""); w.Code != http.StatusCreated {
		t.Fatalf("intake auth off: expected 201, got %d (%s)", w.Code, w.Body.String())
	}
}

// TestApprovalStatusAuth: the poll needs the same credential, and a refused
// poll reveals nothing about the record.
func TestApprovalStatusAuth(t *testing.T) {
	approvalAuthEnv(t, "", testAgentSA)
	s, store := newAuthApprovalServer()
	store.approvals["secret-77"] = &coordinator.ApprovalRequestRecord{ID: "secret-77", Type: "secret", Status: "approved"}

	get := func(bearer string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/api/approvals/secret-77", nil)
		if bearer != "" {
			req.Header.Set("Authorization", "Bearer "+bearer)
		}
		w := httptest.NewRecorder()
		s.handleApproval(w, req)
		return w
	}
	for bearer, want := range map[string]int{"": 401, testBadToken: 401, "wrong-audience": 401, "not-allowed": 403} {
		w := get(bearer)
		if w.Code != want {
			t.Fatalf("poll with %q: expected %d, got %d", bearer, want, w.Code)
		}
		if strings.Contains(w.Body.String(), "approved") {
			t.Fatalf("refused poll leaked the decision: %s", w.Body.String())
		}
	}
	if w := get("valid-agent"); w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "approved") {
		t.Fatalf("valid poll: expected 200 with status, got %d (%s)", w.Code, w.Body.String())
	}
}

// TestApprovalResolve_NotGatedByIntakeAuth: approve/reject keep their own auth
// (signed ntfy token or Approver session). With intake auth enforced and no
// bearer at all, an auth-less dashboard still resolves as before.
func TestApprovalResolve_NotGatedByIntakeAuth(t *testing.T) {
	approvalAuthEnv(t, "", testAgentSA)
	s, store := newAuthApprovalServer()
	store.approvals["secret-88"] = &coordinator.ApprovalRequestRecord{ID: "secret-88", Type: "secret", Status: "pending"}
	w := httptest.NewRecorder()
	s.handleApproval(w, httptest.NewRequest(http.MethodPost, "/api/approvals/secret-88/reject", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("reject: expected 200, got %d (%s)", w.Code, w.Body.String())
	}
	if store.approvals["secret-88"].Status != "rejected" {
		t.Fatalf("expected rejected, got %q", store.approvals["secret-88"].Status)
	}
}
