package cognition

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// ============================================================================
// Construction + naming
// ============================================================================

func TestLocalWorker_Name(t *testing.T) {
	tr := NewLocalWorker(64)
	if tr.Name() != "LocalWorker" {
		t.Errorf("Name: got %q, want LocalWorker", tr.Name())
	}
}

func TestLocalWorker_NegativeBufferSize_Clamped(t *testing.T) {
	tr := NewLocalWorker(-5)
	if tr.bufferSize != 0 {
		t.Errorf("expected negative buffer size clamped to 0, got %d", tr.bufferSize)
	}
}

// ============================================================================
// Send / Recv — FIFO + envelope preservation
// ============================================================================

func TestLocalWorker_SendRecv_FIFO(t *testing.T) {
	tr := NewLocalWorker(8)
	defer tr.Close()

	for i := 1; i <= 3; i++ {
		err := tr.Send(Envelope{
			MsgID:   sf("m%d", i),
			From:    "node_a",
			To:      "inbox_b",
			Payload: []byte(sf("payload_%d", i)),
			Clock:   LamportValue(i),
		})
		if err != nil {
			t.Fatalf("Send %d: %v", i, err)
		}
	}

	for i := 1; i <= 3; i++ {
		env, err := tr.Recv("inbox_b")
		if err != nil {
			t.Fatalf("Recv %d: %v", i, err)
		}
		if env.MsgID != sf("m%d", i) {
			t.Errorf("Recv %d: MsgID got %q, want m%d", i, env.MsgID, i)
		}
		if env.Clock != LamportValue(i) {
			t.Errorf("Recv %d: Clock got %d, want %d", i, env.Clock, i)
		}
		if string(env.Payload) != sf("payload_%d", i) {
			t.Errorf("Recv %d: Payload got %q", i, env.Payload)
		}
	}
}

func TestLocalWorker_Recv_DifferentMailboxesIsolated(t *testing.T) {
	tr := NewLocalWorker(4)
	defer tr.Close()

	_ = tr.Send(Envelope{To: "inbox_a", MsgID: "ma1"})
	_ = tr.Send(Envelope{To: "inbox_b", MsgID: "mb1"})

	envA, err := tr.Recv("inbox_a")
	if err != nil {
		t.Fatal(err)
	}
	envB, err := tr.Recv("inbox_b")
	if err != nil {
		t.Fatal(err)
	}
	if envA.MsgID != "ma1" {
		t.Errorf("inbox_a delivery wrong: %q", envA.MsgID)
	}
	if envB.MsgID != "mb1" {
		t.Errorf("inbox_b delivery wrong: %q", envB.MsgID)
	}
}

func TestLocalWorker_Recv_Blocks_UntilSend(t *testing.T) {
	tr := NewLocalWorker(4)
	defer tr.Close()

	recvDone := make(chan Envelope, 1)
	go func() {
		env, _ := tr.Recv("inbox_b")
		recvDone <- env
	}()

	// Give the goroutine a moment to start waiting
	time.Sleep(20 * time.Millisecond)

	// Verify Recv is still blocked
	select {
	case <-recvDone:
		t.Fatal("Recv should block on empty mailbox")
	default:
	}

	// Send unblocks
	_ = tr.Send(Envelope{To: "inbox_b", MsgID: "delayed"})

	select {
	case env := <-recvDone:
		if env.MsgID != "delayed" {
			t.Errorf("unblocked Recv got wrong envelope: %v", env)
		}
	case <-time.After(time.Second):
		t.Fatal("Recv did not unblock after Send")
	}
}

// ============================================================================
// Subscribe — fanout + cancel cleanup
// ============================================================================

func TestLocalWorker_Subscribe_DeliversOnSend(t *testing.T) {
	tr := NewLocalWorker(8)
	defer tr.Close()

	received := make([]string, 0)
	var mu sync.Mutex
	done := make(chan struct{})
	want := 3
	got := 0

	cancel, err := tr.Subscribe("inbox_b", func(env Envelope) {
		mu.Lock()
		received = append(received, env.MsgID)
		got++
		if got == want {
			close(done)
		}
		mu.Unlock()
	})
	if err != nil {
		t.Fatal(err)
	}
	defer cancel()

	for i := 1; i <= want; i++ {
		_ = tr.Send(Envelope{To: "inbox_b", MsgID: sf("m%d", i)})
	}

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("subscriber did not receive all 3 events in time")
	}

	mu.Lock()
	defer mu.Unlock()
	if len(received) != want {
		t.Errorf("expected %d deliveries, got %d", want, len(received))
	}
}

func TestLocalWorker_Subscribe_FilterByMailbox(t *testing.T) {
	tr := NewLocalWorker(8)
	defer tr.Close()

	var inboxAcount, inboxBcount int32

	cancelA, _ := tr.Subscribe("inbox_a", func(env Envelope) {
		atomic.AddInt32(&inboxAcount, 1)
	})
	cancelB, _ := tr.Subscribe("inbox_b", func(env Envelope) {
		atomic.AddInt32(&inboxBcount, 1)
	})
	defer cancelA()
	defer cancelB()

	_ = tr.Send(Envelope{To: "inbox_a", MsgID: "a1"})
	_ = tr.Send(Envelope{To: "inbox_a", MsgID: "a2"})
	_ = tr.Send(Envelope{To: "inbox_b", MsgID: "b1"})

	// Wait for fanout
	time.Sleep(50 * time.Millisecond)

	if atomic.LoadInt32(&inboxAcount) != 2 {
		t.Errorf("inbox_a subscriber: got %d events, want 2", inboxAcount)
	}
	if atomic.LoadInt32(&inboxBcount) != 1 {
		t.Errorf("inbox_b subscriber: got %d events, want 1", inboxBcount)
	}
}

func TestLocalWorker_Subscribe_CancelStopsDelivery(t *testing.T) {
	tr := NewLocalWorker(8)
	defer tr.Close()

	var count int32
	cancel, _ := tr.Subscribe("inbox_b", func(env Envelope) {
		atomic.AddInt32(&count, 1)
	})

	_ = tr.Send(Envelope{To: "inbox_b", MsgID: "before_cancel"})
	time.Sleep(20 * time.Millisecond)
	cancel()

	_ = tr.Send(Envelope{To: "inbox_b", MsgID: "after_cancel"})
	time.Sleep(20 * time.Millisecond)

	got := atomic.LoadInt32(&count)
	if got != 1 {
		t.Errorf("expected exactly 1 delivery (the pre-cancel one), got %d", got)
	}
}

func TestLocalWorker_Subscribe_NilCallback_ReturnsError(t *testing.T) {
	tr := NewLocalWorker(4)
	defer tr.Close()
	_, err := tr.Subscribe("inbox_b", nil)
	if err == nil {
		t.Fatal("expected error for nil onMsg, got nil")
	}
}

// ============================================================================
// Close — graceful shutdown
// ============================================================================

func TestLocalWorker_Close_BlocksFurtherSends(t *testing.T) {
	tr := NewLocalWorker(4)
	if err := tr.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	err := tr.Send(Envelope{To: "inbox_b"})
	if !errors.Is(err, ErrTransportClosed) {
		t.Errorf("expected ErrTransportClosed, got %v", err)
	}
}

func TestLocalWorker_Close_UnblocksRecv(t *testing.T) {
	tr := NewLocalWorker(4)

	recvErr := make(chan error, 1)
	go func() {
		_, err := tr.Recv("inbox_b")
		recvErr <- err
	}()

	time.Sleep(20 * time.Millisecond)
	_ = tr.Close()

	select {
	case err := <-recvErr:
		if !errors.Is(err, ErrTransportClosed) {
			t.Errorf("expected ErrTransportClosed, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Recv did not unblock after Close")
	}
}

func TestLocalWorker_Close_Idempotent(t *testing.T) {
	tr := NewLocalWorker(4)
	if err := tr.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}
	if err := tr.Close(); err != nil {
		t.Errorf("second Close should be no-op, got: %v", err)
	}
}

// ============================================================================
// Concurrency — multiple senders + subscribers
// ============================================================================

func TestLocalWorker_Concurrent_SendRecv(t *testing.T) {
	tr := NewLocalWorker(32) // small buffer; receiver must drain concurrently
	defer tr.Close()

	const senders = 10
	const perSender = 20
	total := senders * perSender

	// Drain concurrently with sends so the mailbox buffer doesn't fill.
	received := make(chan struct{}, total)
	go func() {
		for i := 0; i < total; i++ {
			_, err := tr.Recv("inbox_b")
			if err != nil {
				return
			}
			received <- struct{}{}
		}
	}()

	var wg sync.WaitGroup
	for s := 0; s < senders; s++ {
		s := s
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < perSender; i++ {
				_ = tr.Send(Envelope{
					To:    "inbox_b",
					From:  sf("sender_%d", s),
					MsgID: sf("s%d_m%d", s, i),
				})
			}
		}()
	}
	wg.Wait()

	// Wait for the drainer to confirm all envelopes received.
	count := 0
	timeout := time.After(2 * time.Second)
	for count < total {
		select {
		case <-received:
			count++
		case <-timeout:
			t.Fatalf("only received %d/%d under concurrent send", count, total)
		}
	}
}

// ============================================================================
// Helpers
// ============================================================================

func sf(format string, args ...interface{}) string {
	return fmt.Sprintf(format, args...)
}
