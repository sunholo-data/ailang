package effects

import (
	"sync"
	"testing"

	"github.com/sunholo-data/ailang/internal/eval"
)

// ============================================================================
// Op registration
// ============================================================================

func TestMsgOpsRegistered(t *testing.T) {
	required := []string{"send", "recv"}
	msgOps, ok := Registry["Msg"]
	if !ok {
		t.Fatalf("Registry[\"Msg\"] not initialized — init() did not register ops")
	}
	for _, op := range required {
		if _, ok := msgOps[op]; !ok {
			t.Errorf("Registry[\"Msg\"][%q] missing — RegisterOp call absent from init()", op)
		}
	}
}

// ============================================================================
// Nil-handler behavior — typed error, not panic
// ============================================================================

func TestMsgContext_NilHandler_ReturnsErr(t *testing.T) {
	var ctx *MsgContext
	_, err := ctx.Send("inbox", []byte("hi"))
	if err == nil {
		t.Fatal("expected ErrNoMsgHandler from nil context, got nil")
	}
	if err != ErrNoMsgHandler {
		t.Errorf("expected ErrNoMsgHandler, got %v", err)
	}
}

func TestMsgContext_NilUnderlyingHandler_ReturnsErr(t *testing.T) {
	ctx := &MsgContext{handler: nil, Self: "node_a"}
	_, err := ctx.Send("inbox", []byte("hi"))
	if err != ErrNoMsgHandler {
		t.Errorf("expected ErrNoMsgHandler from Send, got %v", err)
	}
	_, err = ctx.Recv("inbox")
	if err != ErrNoMsgHandler {
		t.Errorf("expected ErrNoMsgHandler from Recv, got %v", err)
	}
	_, err = ctx.Subscribe("inbox", nil)
	if err != ErrNoMsgHandler {
		t.Errorf("expected ErrNoMsgHandler from Subscribe, got %v", err)
	}
}

// ============================================================================
// StubMsgHandler — Send / Recv FIFO + clock monotonicity
// ============================================================================

func TestStubMsgHandler_Send_AssignsMonotonicClock(t *testing.T) {
	h := NewStubMsgHandler("node_a")

	r1, err := h.Send("inbox_b", []byte("first"))
	if err != nil {
		t.Fatalf("Send 1 failed: %v", err)
	}
	r2, err := h.Send("inbox_b", []byte("second"))
	if err != nil {
		t.Fatalf("Send 2 failed: %v", err)
	}
	if r2.Clock <= r1.Clock {
		t.Errorf("expected monotonic clock, got r1.Clock=%d r2.Clock=%d", r1.Clock, r2.Clock)
	}
	if r1.MsgID == r2.MsgID {
		t.Errorf("expected distinct MsgIDs, got %q == %q", r1.MsgID, r2.MsgID)
	}
	if len(h.Sent) != 2 {
		t.Errorf("expected 2 recorded sends, got %d", len(h.Sent))
	}
}

func TestStubMsgHandler_Recv_FIFO(t *testing.T) {
	h := NewStubMsgHandler("node_a")
	_, _ = h.Send("inbox_b", []byte("first"))
	_, _ = h.Send("inbox_b", []byte("second"))

	m1, err := h.Recv("inbox_b")
	if err != nil {
		t.Fatalf("Recv 1 failed: %v", err)
	}
	if string(m1.Payload) != "first" {
		t.Errorf("expected FIFO first payload, got %q", m1.Payload)
	}
	if m1.From != "node_a" {
		t.Errorf("expected sender stamp 'node_a', got %q", m1.From)
	}

	m2, err := h.Recv("inbox_b")
	if err != nil {
		t.Fatalf("Recv 2 failed: %v", err)
	}
	if string(m2.Payload) != "second" {
		t.Errorf("expected FIFO second payload, got %q", m2.Payload)
	}

	if m2.Clock <= m1.Clock {
		t.Errorf("expected later clock on second message, got %d <= %d", m2.Clock, m1.Clock)
	}
}

func TestStubMsgHandler_Recv_EmptyReturnsErrNoMsgAvailable(t *testing.T) {
	h := NewStubMsgHandler("")

	_, err := h.Recv("empty_mailbox")
	if err != ErrNoMsgAvailable {
		t.Errorf("expected ErrNoMsgAvailable, got %v", err)
	}
}

// ============================================================================
// Subscribe — region routing + cancel cleanup
// ============================================================================

func TestStubMsgHandler_Subscribe_DeliversOnSend(t *testing.T) {
	h := NewStubMsgHandler("node_a")
	received := []Message{}
	var mu sync.Mutex

	cancel, err := h.Subscribe("inbox_b", func(m Message) {
		mu.Lock()
		defer mu.Unlock()
		received = append(received, m)
	})
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	_, _ = h.Send("inbox_b", []byte("hello"))
	_, _ = h.Send("inbox_b", []byte("world"))
	_, _ = h.Send("other_inbox", []byte("ignored")) // should not deliver

	mu.Lock()
	got := len(received)
	mu.Unlock()
	if got != 2 {
		t.Errorf("expected 2 deliveries to inbox_b subscriber, got %d", got)
	}

	cancel()
	_, _ = h.Send("inbox_b", []byte("after_cancel"))
	mu.Lock()
	got = len(received)
	mu.Unlock()
	if got != 2 {
		t.Errorf("expected cancel to stop delivery, got %d after cancel", got)
	}
}

// ============================================================================
// Codecs — AILANG values <-> Go types via ops
// ============================================================================

func TestMsgSend_RoundTrip(t *testing.T) {
	h := NewStubMsgHandler("node_a")
	ctx := &EffContext{Msg: NewMsgContext(h, "node_a")}

	result, err := msgSend(ctx, []eval.Value{
		&eval.StringValue{Value: "inbox_b"},
		&eval.StringValue{Value: "ping"},
	})
	if err != nil {
		t.Fatalf("msgSend failed: %v", err)
	}

	rec, ok := result.(*eval.RecordValue)
	if !ok {
		t.Fatalf("expected RecordValue, got %T", result)
	}
	for _, field := range []string{"msg_id", "clock", "budget_remaining"} {
		if _, ok := rec.Fields[field]; !ok {
			t.Errorf("send result missing %q field", field)
		}
	}

	if len(h.Sent) != 1 {
		t.Fatalf("expected 1 recorded send, got %d", len(h.Sent))
	}
	if h.Sent[0].To != "inbox_b" {
		t.Errorf("expected To=inbox_b, got %q", h.Sent[0].To)
	}
	if string(h.Sent[0].Payload) != "ping" {
		t.Errorf("expected payload=ping, got %q", h.Sent[0].Payload)
	}
}

func TestMsgRecv_RoundTrip(t *testing.T) {
	h := NewStubMsgHandler("node_a")
	ctx := &EffContext{Msg: NewMsgContext(h, "node_a")}

	_, _ = h.Send("inbox_a", []byte("hello"))

	result, err := msgRecv(ctx, []eval.Value{
		&eval.StringValue{Value: "inbox_a"},
	})
	if err != nil {
		t.Fatalf("msgRecv failed: %v", err)
	}
	rec := result.(*eval.RecordValue)
	payload, _ := rec.Fields["payload"].(*eval.StringValue)
	if payload == nil || payload.Value != "hello" {
		t.Errorf("expected payload=hello, got %v", rec.Fields["payload"])
	}
	from, _ := rec.Fields["from"].(*eval.StringValue)
	if from == nil || from.Value != "node_a" {
		t.Errorf("expected from=node_a, got %v", rec.Fields["from"])
	}
}

func TestMsgRecv_EmptyMailbox_ReturnsError(t *testing.T) {
	h := NewStubMsgHandler("node_a")
	ctx := &EffContext{Msg: NewMsgContext(h, "node_a")}

	_, err := msgRecv(ctx, []eval.Value{&eval.StringValue{Value: "empty"}})
	if err == nil {
		t.Fatal("expected error for empty mailbox, got nil")
	}
}

func TestMsgSend_WrongArgCount_ReturnsError(t *testing.T) {
	ctx := &EffContext{Msg: NewMsgContext(NewStubMsgHandler(""), "")}

	_, err := msgSend(ctx, []eval.Value{&eval.StringValue{Value: "only_one"}})
	if err == nil {
		t.Fatal("expected error for missing payload arg, got nil")
	}
}

// ============================================================================
// Behavior-equivalence: this package does NOT modify internal/messaging
// ============================================================================

// TestMsgEffect_DoesNotModifyMessagingPackage is a structural assertion: the
// Msg effect must be a SIBLING to the `ailang messages` CLI, not a
// replacement. CLI behavior remains byte-identical because we add only.
//
// This test does nothing dynamic — it's a documentary anchor for the
// design freeze. The real regression check happens at Day 5-6 when the
// native handler wraps internal/messaging.Store; a `git diff` over
// internal/messaging/* should be empty across the M1 sprint.
func TestMsgEffect_DoesNotModifyMessagingPackage(t *testing.T) {
	// Intentionally empty — see comment above. The acceptance criterion is
	// verified at sprint-checkpoint time via `git diff --stat
	// internal/messaging/`, not by a Go test.
	t.Log("structural anchor: M-COG-RUNTIME M1 Msg effect must not modify internal/messaging/")
}
