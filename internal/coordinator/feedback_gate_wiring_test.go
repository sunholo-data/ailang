package coordinator

import (
	"context"
	"errors"
	"log"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/feedbackgate"
	"github.com/sunholo-data/ailang/internal/messaging"
)

// fakeGateStore is a minimal messaging.MessageStore for the gate wiring tests.
// It embeds the interface (so all unused methods exist but panic if called)
// and records only the two operations the gate path uses: audit inserts and
// rejected-message acks. Pattern borrowed from pubsub_adapter_tag_filter_test.go.
type fakeGateStore struct {
	messaging.MessageStore
	audits []*messaging.InboxMessage
	acked  []string
}

func (f *fakeGateStore) InsertInboxMessage(msg *messaging.InboxMessage) error {
	f.audits = append(f.audits, msg)
	return nil
}

func (f *fakeGateStore) MarkInboxMessageRead(id string) error {
	f.acked = append(f.acked, id)
	return nil
}

// stubGate is a fake feedbackGateDecider returning a canned verdict/error.
type stubGate struct {
	verdict feedbackgate.Verdict
	err     error
	calls   int
}

func (s *stubGate) Decide(_ context.Context, _ feedbackgate.Input, _ feedbackgate.FeedbackGateConfig) (feedbackgate.Verdict, error) {
	s.calls++
	return s.verdict, s.err
}

// newGateTestDaemon builds a minimal Daemon wired with a fake store + gate.
func newGateTestDaemon(gate feedbackGateDecider, cfg *FeedbackGateConfig) (*Daemon, *fakeGateStore) {
	store := &fakeGateStore{}
	d := &Daemon{
		logger:          log.New(&nopWriter{}, "", 0),
		ctx:             context.Background(),
		msgStore:        store,
		feedbackGate:    gate,
		feedbackGateCfg: cfg,
	}
	return d, store
}

type nopWriter struct{}

func (nopWriter) Write(p []byte) (int, error) { return len(p), nil }

func gateMsg() *Message {
	return &Message{ID: "m1", Type: "auto:bug", Content: "body", From: "mcp-public", Inbox: "pkg:a/b", Source: "public"}
}

func enabledCfg() *FeedbackGateConfig {
	return &FeedbackGateConfig{Enabled: true}
}

// TestGateInactiveWhenDisabled: nil gate or disabled config => pass-through
// (allow dispatch, no audit).
func TestGateInactiveWhenDisabled(t *testing.T) {
	// nil gate
	d, store := newGateTestDaemon(nil, nil)
	if !d.gateFeedbackMessage(gateMsg(), "pkg:a/b") {
		t.Fatal("nil gate must pass through (allow dispatch)")
	}
	if len(store.audits) != 0 {
		t.Fatal("nil gate must not emit audits")
	}

	// disabled config
	d2, store2 := newGateTestDaemon(&stubGate{}, &FeedbackGateConfig{Enabled: false})
	if !d2.gateFeedbackMessage(gateMsg(), "pkg:a/b") {
		t.Fatal("disabled gate must pass through")
	}
	if len(store2.audits) != 0 {
		t.Fatal("disabled gate must not emit audits")
	}
}

// TestGateDispatchAllows: dispatch verdict => allow, no audit, no ack.
func TestGateDispatchAllows(t *testing.T) {
	gate := &stubGate{verdict: feedbackgate.Verdict{Action: feedbackgate.ActionDispatch, Reason: feedbackgate.ReasonPassed}}
	d, store := newGateTestDaemon(gate, enabledCfg())
	if !d.gateFeedbackMessage(gateMsg(), "pkg:a/b") {
		t.Fatal("dispatch verdict must allow")
	}
	if len(store.audits) != 0 || len(store.acked) != 0 {
		t.Fatalf("dispatch must not audit/ack; audits=%d acked=%d", len(store.audits), len(store.acked))
	}
	if gate.calls != 1 {
		t.Fatalf("gate called %d times, want 1", gate.calls)
	}
}

// TestGateFileSuppresses: file verdict => not allowed, audit emitted, no ack.
func TestGateFileSuppresses(t *testing.T) {
	gate := &stubGate{verdict: feedbackgate.Verdict{Action: feedbackgate.ActionFile, Reason: feedbackgate.ReasonUnknownCategory}}
	d, store := newGateTestDaemon(gate, enabledCfg())
	if d.gateFeedbackMessage(gateMsg(), "pkg:a/b") {
		t.Fatal("file verdict must suppress dispatch")
	}
	if len(store.audits) != 1 {
		t.Fatalf("file verdict must emit exactly 1 audit, got %d", len(store.audits))
	}
	if store.audits[0].ToInbox != feedbackGateAuditInbox {
		t.Errorf("audit inbox = %q, want %q", store.audits[0].ToInbox, feedbackGateAuditInbox)
	}
	if len(store.acked) != 0 {
		t.Fatal("file (not reject) must not ack the source message")
	}
}

// TestGateRejectSuppressesAndAcks: reject => not allowed, audit + ack.
func TestGateRejectSuppressesAndAcks(t *testing.T) {
	gate := &stubGate{verdict: feedbackgate.Verdict{Action: feedbackgate.ActionReject, Reason: feedbackgate.ReasonSpamPattern}}
	d, store := newGateTestDaemon(gate, enabledCfg())
	if d.gateFeedbackMessage(gateMsg(), "pkg:a/b") {
		t.Fatal("reject verdict must suppress dispatch")
	}
	if len(store.audits) != 1 {
		t.Fatalf("reject must emit 1 audit, got %d", len(store.audits))
	}
	if len(store.acked) != 1 || store.acked[0] != "m1" {
		t.Fatalf("reject must ack the source message, acked=%v", store.acked)
	}
}

// TestGateErrorFailsClosed: a gate error => not allowed (no dispatch) + audit.
func TestGateErrorFailsClosed(t *testing.T) {
	gate := &stubGate{err: errors.New("boom")}
	d, store := newGateTestDaemon(gate, enabledCfg())
	if d.gateFeedbackMessage(gateMsg(), "pkg:a/b") {
		t.Fatal("gate error must fail closed (no dispatch)")
	}
	if len(store.audits) != 1 {
		t.Fatalf("gate error must emit 1 audit, got %d", len(store.audits))
	}
}

// TestGateDryRunAlwaysDispatches: dry-run => allow even for a reject verdict,
// but still audits the would-be verdict.
func TestGateDryRunAlwaysDispatches(t *testing.T) {
	gate := &stubGate{verdict: feedbackgate.Verdict{Action: feedbackgate.ActionReject, Reason: feedbackgate.ReasonSpamPattern}}
	cfg := &FeedbackGateConfig{Enabled: true, DryRun: true}
	d, store := newGateTestDaemon(gate, cfg)
	if !d.gateFeedbackMessage(gateMsg(), "pkg:a/b") {
		t.Fatal("dry-run must always allow dispatch")
	}
	if len(store.audits) != 1 {
		t.Fatalf("dry-run must still audit the would-be verdict, got %d", len(store.audits))
	}
	if !strings.HasPrefix(store.audits[0].Title, "DRY-RUN ") {
		t.Errorf("dry-run audit title = %q, want DRY-RUN prefix", store.audits[0].Title)
	}
	if len(store.acked) != 0 {
		t.Fatal("dry-run must not ack (it dispatched)")
	}
}

// TestModeOffEnvDisablesGate: AILANG_FEEDBACK_GATE_MODE=off => pass-through.
func TestModeOffEnvDisablesGate(t *testing.T) {
	t.Setenv("AILANG_FEEDBACK_GATE_MODE", "off")
	gate := &stubGate{verdict: feedbackgate.Verdict{Action: feedbackgate.ActionReject}}
	d, store := newGateTestDaemon(gate, enabledCfg())
	if !d.gateFeedbackMessage(gateMsg(), "pkg:a/b") {
		t.Fatal("mode=off must pass through")
	}
	if gate.calls != 0 {
		t.Fatalf("mode=off must not call the gate, calls=%d", gate.calls)
	}
	if len(store.audits) != 0 {
		t.Fatal("mode=off must not audit")
	}
}
