package notify

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNtfyChannel_SendBuildsActionableRequest(t *testing.T) {
	var gotTitle, gotActions, gotBody, gotAuth string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotTitle = r.Header.Get("X-Title")
		gotActions = r.Header.Get("X-Actions")
		gotAuth = r.Header.Get("Authorization")
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	c := NewNtfyChannel(ts.URL, "ailang-approvals", "ntfy-secret")
	n := Notification{
		Title:     "Secret requested: op://Prod/stripe/api-key",
		Body:      "Agent agent-x — charge a card",
		EventType: "pending_approval",
		Actions: []NotificationAction{
			{Label: "Approve", URL: "https://coord/api/approvals/a1/approve?token=TOK_A", Method: "POST"},
			{Label: "Deny", URL: "https://coord/api/approvals/a1/reject?token=TOK_R", Method: "POST"},
		},
	}
	if err := c.Send(context.Background(), n); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if gotTitle != n.Title {
		t.Fatalf("X-Title = %q", gotTitle)
	}
	if gotBody != n.Body {
		t.Fatalf("body = %q", gotBody)
	}
	if gotAuth != "Bearer ntfy-secret" {
		t.Fatalf("Authorization = %q", gotAuth)
	}
	for _, want := range []string{"label=Approve", "TOK_A", "label=Deny", "TOK_R", "method=POST"} {
		if !strings.Contains(gotActions, want) {
			t.Fatalf("X-Actions %q missing %q", gotActions, want)
		}
	}
}

func TestNtfyChannel_OnlyAcceptsApprovals(t *testing.T) {
	c := NewNtfyChannel("https://x", "t", "")
	if !c.Accepts("pending_approval") {
		t.Fatal("should accept pending_approval")
	}
	if c.Accepts("completed") {
		t.Fatal("should NOT accept completed (dedicated approvals channel)")
	}
}

func TestNtfyChannel_NonGreenStatusIsError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer ts.Close()
	c := NewNtfyChannel(ts.URL, "t", "")
	if err := c.Send(context.Background(), Notification{Body: "x"}); err == nil {
		t.Fatal("expected error on non-2xx response")
	}
}

func TestNtfyChannel_IsRemoteAuthoritative(t *testing.T) {
	// NtfyChannel must NOT be a LocalChannel (it is remote/authoritative).
	var ch Channel = NewNtfyChannel("https://x", "t", "")
	if _, isLocal := ch.(LocalChannel); isLocal {
		t.Fatal("NtfyChannel must not implement LocalChannel")
	}
}
