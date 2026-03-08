package coordinator

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func testLogger() *log.Logger {
	return log.New(os.Stderr, "test: ", 0)
}

func TestVerifyWebhookSignature(t *testing.T) {
	secret := "test-secret-key"
	body := []byte(`{"action":"labeled"}`)

	// Compute valid signature
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	validSig := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	tests := []struct {
		name   string
		body   []byte
		sig    string
		secret string
		want   bool
	}{
		{
			name:   "valid signature",
			body:   body,
			sig:    validSig,
			secret: secret,
			want:   true,
		},
		{
			name:   "invalid signature",
			body:   body,
			sig:    "sha256=0000000000000000000000000000000000000000000000000000000000000000",
			secret: secret,
			want:   false,
		},
		{
			name:   "empty signature",
			body:   body,
			sig:    "",
			secret: secret,
			want:   false,
		},
		{
			name:   "missing sha256 prefix",
			body:   body,
			sig:    hex.EncodeToString(mac.Sum(nil)),
			secret: secret,
			want:   false,
		},
		{
			name:   "wrong body",
			body:   []byte(`{"action":"opened"}`),
			sig:    validSig,
			secret: secret,
			want:   false,
		},
		{
			name:   "wrong secret",
			body:   body,
			sig:    validSig,
			secret: "wrong-secret",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := verifyWebhookSignature(tt.body, tt.sig, tt.secret)
			if got != tt.want {
				t.Errorf("verifyWebhookSignature() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLabelToEventType(t *testing.T) {
	watcher := NewApprovalWatcher(nil, nil, 0)

	tests := []struct {
		name  string
		label string
		want  ApprovalEventType
	}{
		{"design-approved", "design-approved", ApprovalEventDesign},
		{"sprint-approved", "sprint-approved", ApprovalEventSprint},
		{"merge-approved", "merge-approved", ApprovalEventMerge},
		{"needs-revision", "needs-revision", ApprovalEventRevision},
		{"unknown label", "random-label", ""},
		{"empty label", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := labelToEventType(tt.label, watcher)
			if got != tt.want {
				t.Errorf("labelToEventType(%q) = %q, want %q", tt.label, got, tt.want)
			}
		})
	}
}

func TestHandleGitHubWebhookPing(t *testing.T) {
	d := &Daemon{logger: testLogger()}

	body := `{"zen":"Responsive is better than fast."}`
	req := httptest.NewRequest(http.MethodPost, "/github/webhook", strings.NewReader(body))
	req.Header.Set("X-GitHub-Event", "ping")

	w := httptest.NewRecorder()
	d.handleGitHubWebhook(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("ping: got status %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHandleGitHubWebhookMethodNotAllowed(t *testing.T) {
	d := &Daemon{logger: testLogger()}

	req := httptest.NewRequest(http.MethodGet, "/github/webhook", nil)
	w := httptest.NewRecorder()
	d.handleGitHubWebhook(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET: got status %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandleGitHubWebhookInvalidSignature(t *testing.T) {
	d := &Daemon{logger: testLogger()}

	body := `{"action":"labeled"}`
	req := httptest.NewRequest(http.MethodPost, "/github/webhook", strings.NewReader(body))
	req.Header.Set("X-GitHub-Event", "issues")
	req.Header.Set("X-Hub-Signature-256", "sha256=invalid")

	t.Setenv("GITHUB_WEBHOOK_SECRET", "test-secret")

	w := httptest.NewRecorder()
	d.handleGitHubWebhook(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("invalid sig: got status %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestHandleGitHubWebhookBotSender(t *testing.T) {
	d := &Daemon{logger: testLogger()}

	payload := webhookIssuePayload{
		Action: "labeled",
		Sender: struct {
			Login string `json:"login"`
			Type  string `json:"type"`
		}{Login: "github-actions[bot]", Type: "Bot"},
		Issue: struct {
			Number int    `json:"number"`
			Title  string `json:"title"`
			Body   string `json:"body"`
		}{Number: 42},
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/github/webhook", strings.NewReader(string(body)))
	req.Header.Set("X-GitHub-Event", "issues")

	w := httptest.NewRecorder()
	d.handleGitHubWebhook(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("bot sender: got status %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHandleGitHubWebhookUnknownEvent(t *testing.T) {
	d := &Daemon{logger: testLogger()}

	req := httptest.NewRequest(http.MethodPost, "/github/webhook", strings.NewReader("{}"))
	req.Header.Set("X-GitHub-Event", "push")

	w := httptest.NewRecorder()
	d.handleGitHubWebhook(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("unknown event: got status %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHandleGitHubWebhookLabeledNoWatcher(t *testing.T) {
	d := &Daemon{
		logger:          testLogger(),
		approvalWatcher: nil,
	}

	payload := webhookIssuePayload{
		Action: "labeled",
		Sender: struct {
			Login string `json:"login"`
			Type  string `json:"type"`
		}{Login: "human-user", Type: "User"},
		Issue: struct {
			Number int    `json:"number"`
			Title  string `json:"title"`
			Body   string `json:"body"`
		}{Number: 42},
		Label: struct {
			Name string `json:"name"`
		}{Name: "design-approved"},
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/github/webhook", strings.NewReader(string(body)))
	req.Header.Set("X-GitHub-Event", "issues")

	w := httptest.NewRecorder()
	d.handleGitHubWebhook(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("no watcher: got status %d, want %d", w.Code, http.StatusOK)
	}
}

func TestFormatWebhookSetupCommand(t *testing.T) {
	got := FormatWebhookSetupCommand("sunholo-data/ailang", "https://coordinator.example.com", "s3cr3t")
	want := `gh webhook create --repo sunholo-data/ailang --events issues --url "https://coordinator.example.com/github/webhook" --secret "s3cr3t"`
	if got != want {
		t.Errorf("FormatWebhookSetupCommand() =\n  %s\nwant:\n  %s", got, want)
	}
}
