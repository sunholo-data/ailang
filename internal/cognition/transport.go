package cognition

import (
	"errors"
	"sync"
)

// ============================================================================
// Transport trait — M-COG-RUNTIME M2, v0.21.x
// ============================================================================
//
// Transport is the pluggable boundary between agent message-passing
// semantics and the underlying delivery medium. Same agent code runs
// over LocalWorker (in-process), BroadcastChannel (cross-tab same-
// origin, M2 Day 11), WebSocket (M-COG-MESH), FirestoreRelay (M-COG-
// MESH), WebRTC (M-COG-MESH) — without source-code changes.
//
// Key principle: agent semantics must not depend on transport.
// The Transport trait surface is intentionally minimal:
//   - Send: dispatch a payload to a mailbox
//   - Recv: pull the next payload from a mailbox (blocking)
//   - Subscribe: long-lived callback on arrival (M3 scheduler hook)
//
// Higher-level concerns (Lamport-clock stamping, sender identity,
// envelope serialization) are handled by MsgHandler in internal/effects/
// — Transport just moves bytes.

// ErrTransportClosed is returned when a graceful shutdown was initiated
// on the transport. Distinct from transport errors (network failures,
// peer disconnect) which return wrapped Go errors.
var ErrTransportClosed = errors.New("transport: closed")

// Envelope is the payload-with-metadata flowing through the transport.
//
// Transports treat Payload as opaque bytes; the Msg handler in
// internal/effects/msg.go is responsible for AILANG-value encoding.
// Clock + Sender are populated by the Msg handler at send-time and
// preserved across transport hops for replay determinism.
type Envelope struct {
	MsgID   string
	From    string
	To      string
	Payload []byte
	Clock   LamportValue
}

// Transport is the pluggable medium for inter-agent messaging.
//
// Implementations must be safe for concurrent Send / Recv / Subscribe
// calls (the M3 scheduler may interleave them across multiple agents
// in one event loop).
type Transport interface {
	// Send dispatches an envelope. Returns an error for transport-level
	// failures (network, closed transport); message-level concerns
	// (budget, malformed payload) are upstream of this layer.
	Send(env Envelope) error

	// Recv blocks for the next envelope in the named mailbox. Returns
	// ErrTransportClosed when the transport shuts down gracefully.
	Recv(mailbox string) (Envelope, error)

	// Subscribe registers a long-lived callback for arrivals in the
	// named mailbox. The returned cancel removes the subscription;
	// callers must invoke cancel to avoid leaking goroutines.
	Subscribe(mailbox string, onMsg func(Envelope)) (cancel func(), err error)

	// Name identifies this transport for logging / manifest validation.
	// Examples: "LocalWorker", "BroadcastChannel", "WebSocket".
	Name() string

	// Close shuts down the transport. Subsequent Send / Recv calls
	// return ErrTransportClosed. Idempotent.
	Close() error
}

// ============================================================================
// LocalWorker — in-process transport (the M2 default)
// ============================================================================
//
// LocalWorkerTransport routes envelopes through Go channels for in-process
// agent communication. Each mailbox is a buffered channel; Subscribe
// fan-outs by reading from a shared per-mailbox tap.
//
// Semantics:
//   - FIFO per mailbox (preserves Lamport-clock order at the transport
//     layer; the M3 scheduler enforces global ordering)
//   - Send is non-blocking up to the per-mailbox buffer capacity;
//     beyond that it blocks (matches what BroadcastChannel does in
//     practice — back-pressure rather than drop)
//   - Subscribe receives a copy of every envelope delivered AFTER the
//     subscription is registered (no replay of past messages — that's
//     the event log's job)

// LocalWorkerTransport is the in-process Transport implementation.
type LocalWorkerTransport struct {
	mu         sync.Mutex
	mailboxes  map[string]chan Envelope
	subs       map[string][]*localSubscriber
	bufferSize int
	closed     bool
	closeChan  chan struct{}
}

type localSubscriber struct {
	ch chan Envelope
}

// NewLocalWorker creates an in-process Transport with the given per-
// mailbox buffer size. Buffer size 0 means unbuffered (synchronous);
// 64 is a reasonable default for agent topologies.
func NewLocalWorker(bufferSize int) *LocalWorkerTransport {
	if bufferSize < 0 {
		bufferSize = 0
	}
	return &LocalWorkerTransport{
		mailboxes:  make(map[string]chan Envelope),
		subs:       make(map[string][]*localSubscriber),
		bufferSize: bufferSize,
		closeChan:  make(chan struct{}),
	}
}

// Name returns "LocalWorker".
func (t *LocalWorkerTransport) Name() string { return "LocalWorker" }

// getOrCreateMailbox returns the channel for the named mailbox, creating
// it lazily. Called from within Send / Recv / Subscribe.
//
// Concurrency: caller must hold t.mu.
func (t *LocalWorkerTransport) getOrCreateMailbox(name string) chan Envelope {
	if ch, ok := t.mailboxes[name]; ok {
		return ch
	}
	ch := make(chan Envelope, t.bufferSize)
	t.mailboxes[name] = ch
	return ch
}

// Send dispatches an envelope. Blocks if the mailbox buffer is full.
//
// Subscriber fanout happens synchronously here — every subscriber
// receives a copy before Send returns. This matches the deterministic
// scheduling story: if Send returns, all observers have seen the event.
func (t *LocalWorkerTransport) Send(env Envelope) error {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return ErrTransportClosed
	}
	ch := t.getOrCreateMailbox(env.To)
	subs := append([]*localSubscriber(nil), t.subs[env.To]...)
	t.mu.Unlock()

	// Deliver to mailbox queue. This is the "Recv" path.
	select {
	case ch <- env:
	case <-t.closeChan:
		return ErrTransportClosed
	}

	// Fan out to subscribers (no lock — copy taken above).
	for _, s := range subs {
		select {
		case s.ch <- env:
		case <-t.closeChan:
			return ErrTransportClosed
		}
	}
	return nil
}

// Recv blocks until the named mailbox receives an envelope or the
// transport is closed.
func (t *LocalWorkerTransport) Recv(mailbox string) (Envelope, error) {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return Envelope{}, ErrTransportClosed
	}
	ch := t.getOrCreateMailbox(mailbox)
	t.mu.Unlock()

	select {
	case env := <-ch:
		return env, nil
	case <-t.closeChan:
		return Envelope{}, ErrTransportClosed
	}
}

// Subscribe registers a long-lived callback. The callback runs in a
// dedicated goroutine; the returned cancel function stops the goroutine
// and removes the subscription.
//
// Goroutine lifecycle:
//   - Started when Subscribe returns
//   - Receives from a per-subscriber buffered channel (size = bufferSize)
//   - Exits when cancel is called OR the transport is closed
//   - Does NOT receive past messages — only those sent after Subscribe
func (t *LocalWorkerTransport) Subscribe(mailbox string, onMsg func(Envelope)) (func(), error) {
	if onMsg == nil {
		return nil, errors.New("transport: Subscribe requires non-nil onMsg")
	}
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return nil, ErrTransportClosed
	}
	sub := &localSubscriber{
		ch: make(chan Envelope, t.bufferSize),
	}
	t.subs[mailbox] = append(t.subs[mailbox], sub)
	t.mu.Unlock()

	stopChan := make(chan struct{})

	go func() {
		for {
			select {
			case env := <-sub.ch:
				onMsg(env)
			case <-stopChan:
				return
			case <-t.closeChan:
				return
			}
		}
	}()

	cancel := func() {
		close(stopChan)
		// Remove from subscribers list
		t.mu.Lock()
		defer t.mu.Unlock()
		subs := t.subs[mailbox]
		for i, s := range subs {
			if s == sub {
				t.subs[mailbox] = append(subs[:i], subs[i+1:]...)
				return
			}
		}
	}
	return cancel, nil
}

// Close shuts down the transport. All pending Recv / Subscribe goroutines
// exit; subsequent operations return ErrTransportClosed. Idempotent.
func (t *LocalWorkerTransport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed {
		return nil
	}
	t.closed = true
	close(t.closeChan)
	return nil
}
