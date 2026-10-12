package claudegateway

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestCapabilityCannotChangeAccountOrOutliveLease(t *testing.T) {
	key := []byte(strings.Repeat("k", 32))
	now := time.Date(2026, 10, 10, 10, 0, 0, 0, time.UTC)
	token, err := SignCapability(key, Capability{AccountID: "anthropic-api-credits", TaskID: "task-a", AttemptID: "attempt-a", JobIdentity: "job-sa", ExpiresAt: now.Add(time.Minute)})
	if err != nil {
		t.Fatal(err)
	}
	cap, err := VerifyCapability(key, token, now)
	if err != nil || cap.AccountID != "anthropic-api-credits" {
		t.Fatalf("verify: %+v %v", cap, err)
	}
	if _, err = VerifyCapability(key, token+"x", now); err == nil {
		t.Fatal("tampered capability admitted")
	}
	if _, err = VerifyCapability(key, token, now.Add(time.Minute)); err == nil {
		t.Fatal("expired capability admitted")
	}
}
func TestUnauthenticatedAndUnsupportedRequestsNeverForward(t *testing.T) {
	calls := 0
	h := &Gateway{SigningKey: []byte(strings.Repeat("k", 32)), Client: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		t.Fatal("unexpected upstream send")
		return nil, nil
	})}}
	for _, path := range []string{"/v1/messages", "/v1/messages/batches", "/admin/tasks"} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("POST", path, strings.NewReader(`{"model":"claude-haiku-5-5","max_tokens":1}`)))
		if w.Code < 400 {
			t.Fatalf("%s: %d", path, w.Code)
		}
	}
	if calls != 0 {
		t.Fatal("forwarded")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
