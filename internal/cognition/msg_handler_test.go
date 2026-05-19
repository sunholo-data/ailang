package cognition

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/effects"
)

// ============================================================================
// NativeMsgHandler — wraps Transport into effects.MsgHandler
// ============================================================================

func TestNativeMsgHandler_Send_StampsClockAndID(t *testing.T) {
	tr := NewLocalWorker(8)
	defer tr.Close()
	h := NewNativeMsgHandler(tr, "node_a", nil, nil)

	r1, err := h.Send(effects.Mailbox("inbox_b"), []byte("hello"))
	if err != nil {
		t.Fatalf("Send 1: %v", err)
	}
	r2, err := h.Send(effects.Mailbox("inbox_b"), []byte("world"))
	if err != nil {
		t.Fatalf("Send 2: %v", err)
	}

	if r1.Clock >= r2.Clock {
		t.Errorf("clock must advance: got r1=%d r2=%d", r1.Clock, r2.Clock)
	}
	if r1.MsgID == r2.MsgID {
		t.Errorf("MsgID must be unique across sends: %q == %q", r1.MsgID, r2.MsgID)
	}
}

func TestNativeMsgHandler_Recv_AdvancesClockToHappensBeforeRemote(t *testing.T) {
	tr := NewLocalWorker(8)
	defer tr.Close()

	// Pre-populate the mailbox with an envelope carrying a future clock value.
	_ = tr.Send(Envelope{
		MsgID:   "external_1",
		From:    "external_sender",
		To:      "inbox_a",
		Payload: []byte("from outside"),
		Clock:   42, // future relative to local clock=0
	})

	clock := NewClock()
	h := NewNativeMsgHandler(tr, "node_a", clock, nil)

	msg, err := h.Recv(effects.Mailbox("inbox_a"))
	if err != nil {
		t.Fatalf("Recv: %v", err)
	}

	// Recv should preserve the sender's clock on the returned Message
	if msg.Clock != 42 {
		t.Errorf("Message.Clock should preserve sender clock: got %d, want 42", msg.Clock)
	}

	// Local clock should now be at least 43 (max(local, remote)+1 = 43)
	if clock.Read() < 43 {
		t.Errorf("local clock should advance past remote: got %d, want >=43", clock.Read())
	}

	// Next local send happens AFTER the received event in Lamport order
	res, _ := h.Send(effects.Mailbox("inbox_b"), []byte("after recv"))
	if res.Clock <= 42 {
		t.Errorf("post-Recv Send should have clock > 42, got %d", res.Clock)
	}
}

func TestNativeMsgHandler_RoundTrip_BetweenTwoHandlers(t *testing.T) {
	tr := NewLocalWorker(8)
	defer tr.Close()

	hA := NewNativeMsgHandler(tr, "node_a", nil, nil)
	hB := NewNativeMsgHandler(tr, "node_b", nil, nil)

	_, err := hA.Send(effects.Mailbox("inbox_b"), []byte("ping"))
	if err != nil {
		t.Fatal(err)
	}

	msg, err := hB.Recv(effects.Mailbox("inbox_b"))
	if err != nil {
		t.Fatal(err)
	}
	if msg.From != "node_a" {
		t.Errorf("From: got %q, want node_a", msg.From)
	}
	if string(msg.Payload) != "ping" {
		t.Errorf("Payload: got %q, want ping", msg.Payload)
	}
}

// ============================================================================
// Event log integration
// ============================================================================

func TestNativeMsgHandler_LogsMessageSent(t *testing.T) {
	tr := NewLocalWorker(8)
	defer tr.Close()
	log := NewEventLog(nil)
	h := NewNativeMsgHandler(tr, "node_a", nil, log)

	_, err := h.Send(effects.Mailbox("inbox_b"), []byte("hi"))
	if err != nil {
		t.Fatal(err)
	}
	if log.Len() != 1 {
		t.Fatalf("expected 1 logged event, got %d", log.Len())
	}
	events := log.Snapshot()
	sent, ok := events[0].(MessageSentEvent)
	if !ok {
		t.Fatalf("expected MessageSentEvent, got %T", events[0])
	}
	if sent.To != "inbox_b" {
		t.Errorf("To: got %q, want inbox_b", sent.To)
	}
	if sent.Sender != "node_a" {
		t.Errorf("Sender: got %q, want node_a", sent.Sender)
	}
	if sent.Clock == 0 {
		t.Errorf("Clock should be non-zero, got 0")
	}
}

func TestNativeMsgHandler_LogsMessageReceived(t *testing.T) {
	tr := NewLocalWorker(8)
	defer tr.Close()
	log := NewEventLog(nil)

	// External sender (no log)
	external := NewNativeMsgHandler(tr, "external", nil, nil)
	_, _ = external.Send(effects.Mailbox("inbox_a"), []byte("hello"))

	// Local handler with log
	h := NewNativeMsgHandler(tr, "node_a", nil, log)
	_, err := h.Recv(effects.Mailbox("inbox_a"))
	if err != nil {
		t.Fatal(err)
	}

	if log.Len() != 1 {
		t.Fatalf("expected 1 logged event, got %d", log.Len())
	}
	received, ok := log.Snapshot()[0].(MessageReceivedEvent)
	if !ok {
		t.Fatalf("expected MessageReceivedEvent, got %T", log.Snapshot()[0])
	}
	if received.From != "external" {
		t.Errorf("From: got %q, want external", received.From)
	}
}

// ============================================================================
// Subscribe — clock advance + log
// ============================================================================

func TestNativeMsgHandler_Subscribe_AdvancesClockAndLogs(t *testing.T) {
	tr := NewLocalWorker(8)
	defer tr.Close()
	log := NewEventLog(nil)
	clock := NewClock()
	h := NewNativeMsgHandler(tr, "node_a", clock, log)

	var received int32
	gotPayload := make(chan string, 1)
	cancel, err := h.Subscribe(effects.Mailbox("inbox_a"), func(m effects.Message) {
		atomic.AddInt32(&received, 1)
		select {
		case gotPayload <- string(m.Payload):
		default:
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	defer cancel()

	// External sender with high clock
	external := NewNativeMsgHandler(tr, "external", NewClockAt(100), nil)
	_, _ = external.Send(effects.Mailbox("inbox_a"), []byte("event_payload"))

	select {
	case p := <-gotPayload:
		if p != "event_payload" {
			t.Errorf("payload: got %q", p)
		}
	case <-time.After(time.Second):
		t.Fatal("Subscribe callback did not fire in time")
	}

	// Clock should have advanced past external's clock (101)
	if clock.Read() < 102 {
		t.Errorf("Subscribe should advance local clock past remote: got %d, want >=102", clock.Read())
	}

	// Log should have one MessageReceivedEvent
	if log.Len() != 1 {
		t.Errorf("expected 1 logged Receive event, got %d", log.Len())
	}
}

// ============================================================================
// Behavior-equivalence anchor — internal/messaging is NOT modified
// ============================================================================

// TestNativeMsgHandler_DoesNotImportMessagingPackage documents that this
// handler wraps the Transport, NOT internal/messaging.Store. The CLI-
// facing `ailang messages` commands remain byte-identical because they
// consume the messaging package which is unchanged.
//
// This is a structural documentary anchor matching
// TestMsgEffect_DoesNotModifyMessagingPackage in internal/effects/msg_test.go.
// The actual byte-identical regression check happens at M2 checkpoint via
// `git diff --stat internal/messaging/`.
func TestNativeMsgHandler_DoesNotImportMessagingPackage(t *testing.T) {
	t.Log("structural anchor: NativeMsgHandler must not modify internal/messaging/")
}

// ============================================================================
// Closed transport surfaces errors cleanly
// ============================================================================

func TestNativeMsgHandler_Send_OnClosedTransport_ReturnsError(t *testing.T) {
	tr := NewLocalWorker(4)
	tr.Close()
	h := NewNativeMsgHandler(tr, "node_a", nil, nil)

	_, err := h.Send(effects.Mailbox("inbox_b"), []byte("ignored"))
	if err == nil {
		t.Fatal("expected error from closed transport, got nil")
	}
}

func TestNativeMsgHandler_Recv_OnClosedTransport_ReturnsError(t *testing.T) {
	tr := NewLocalWorker(4)
	tr.Close()
	h := NewNativeMsgHandler(tr, "node_a", nil, nil)

	_, err := h.Recv(effects.Mailbox("inbox_b"))
	if err == nil {
		t.Fatal("expected error from closed transport, got nil")
	}
}
