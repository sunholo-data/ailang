package coordinator

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/sunholo-data/ailang/internal/notify"
)

func discardDaemon() *Daemon {
	return &Daemon{logger: log.New(io.Discard, "", 0)}
}

// TestHandlePushApproval_ForwardsToNtfy: a kind=approval push is forwarded to
// the configured ntfy server as a notification POST.
func TestHandlePushApproval_ForwardsToNtfy(t *testing.T) {
	var mu sync.Mutex
	hits := 0
	title := ""
	ntfy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		hits++
		title = r.Header.Get("X-Title")
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer ntfy.Close()

	t.Setenv("AILANG_NTFY_SERVER_URL", ntfy.URL)
	t.Setenv("AILANG_NTFY_TOPIC", "ailang-approvals")

	notif := notify.Notification{
		Title:     "Secret requested: op://Prod/stripe/key",
		Body:      "agent-x requests op://Prod/stripe/key",
		EventType: "pending_approval",
	}
	data, _ := json.Marshal(notif)

	err := discardDaemon().handlePushApproval(context.Background(), "msg-1", data,
		map[string]string{"kind": "approval", "approval_id": "secret-1"})
	if err != nil {
		t.Fatalf("handlePushApproval: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if hits != 1 {
		t.Fatalf("expected 1 ntfy POST, got %d", hits)
	}
	if title == "" {
		t.Errorf("expected the notification title forwarded to ntfy (X-Title)")
	}
}

// TestHandlePushApproval_DedupsDuplicateDelivery: Pub/Sub is at-least-once, so
// the same message ID can arrive twice. The second delivery must NOT re-push to
// the phone, while a genuinely different message ID still does.
func TestHandlePushApproval_DedupsDuplicateDelivery(t *testing.T) {
	var mu sync.Mutex
	hits := 0
	ntfy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		hits++
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer ntfy.Close()
	t.Setenv("AILANG_NTFY_SERVER_URL", ntfy.URL)
	t.Setenv("AILANG_NTFY_TOPIC", "ailang-approvals")

	data, _ := json.Marshal(notify.Notification{Title: "t", EventType: "pending_approval"})
	d := discardDaemon() // one daemon → shared dedup state across deliveries
	attrs := map[string]string{"kind": "approval", "approval_id": "secret-dup"}

	// Same message ID delivered twice → exactly one forward.
	for i := 0; i < 2; i++ {
		if err := d.handlePushApproval(context.Background(), "dup-msg", data, attrs); err != nil {
			t.Fatalf("delivery %d: %v", i, err)
		}
	}
	// A different message ID still forwards.
	if err := d.handlePushApproval(context.Background(), "other-msg", data, attrs); err != nil {
		t.Fatalf("distinct delivery: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if hits != 2 {
		t.Fatalf("expected 2 ntfy POSTs (duplicate suppressed), got %d", hits)
	}
}

// TestHandlePushApproval_NoopWhenNtfyUnconfigured: with no ntfy env, forwarding
// is a no-op (the executor still learns the decision by polling).
func TestHandlePushApproval_NoopWhenNtfyUnconfigured(t *testing.T) {
	t.Setenv("AILANG_NTFY_SERVER_URL", "")
	t.Setenv("AILANG_NTFY_TOPIC", "")
	err := discardDaemon().handlePushApproval(context.Background(), "msg-x", []byte(`{}`),
		map[string]string{"approval_id": "x"})
	if err != nil {
		t.Fatalf("expected no-op without ntfy config, got %v", err)
	}
}
