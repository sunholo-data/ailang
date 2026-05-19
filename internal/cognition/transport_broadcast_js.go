//go:build js && wasm

package cognition

import (
	"errors"
	"fmt"
	"sync"
	"syscall/js"
)

// ============================================================================
// BroadcastChannel transport — M-COG-RUNTIME M2 Day 11, v0.21.x
// ============================================================================
//
// BroadcastChannel routes envelopes across browser tabs in the same origin
// via the BroadcastChannel Web API. Build-tagged js && wasm — compiles only
// for the WASM target.
//
// Properties:
//   - Same-origin only (browser security model)
//   - Async delivery via the browser event loop
//   - Per-channel name (used as the mailbox prefix below)
//   - No back-pressure — the browser delivers what it can; overflow drops
//     silently. The M3 scheduler should apply explicit budgets.
//
// JS shape per envelope (postMessage data field):
//   {
//     msg_id: string,
//     from:   string,
//     to:     string,
//     payload: string (base64 if binary; M2 keeps it string),
//     clock:  number
//   }

// BroadcastChannelTransport implements Transport over the BroadcastChannel
// Web API. One BroadcastChannel object per mailbox, lazy-created.
type BroadcastChannelTransport struct {
	mu         sync.Mutex
	channels   map[string]js.Value      // mailbox name → BroadcastChannel object
	queues     map[string]chan Envelope // pending arrivals (Recv pulls from these)
	subs       map[string][]*localSubscriber
	closed     bool
	closeChan  chan struct{}
	bufferSize int
}

// NewBroadcastChannel constructs a BroadcastChannel-backed transport.
// bufferSize: per-mailbox arrival buffer (envelopes queued for Recv).
func NewBroadcastChannel(bufferSize int) *BroadcastChannelTransport {
	if bufferSize < 0 {
		bufferSize = 0
	}
	return &BroadcastChannelTransport{
		channels:   make(map[string]js.Value),
		queues:     make(map[string]chan Envelope),
		subs:       make(map[string][]*localSubscriber),
		closeChan:  make(chan struct{}),
		bufferSize: bufferSize,
	}
}

// Name returns "BroadcastChannel".
func (t *BroadcastChannelTransport) Name() string { return "BroadcastChannel" }

// ensureChannel creates the BroadcastChannel object + onmessage handler
// for the named mailbox. Caller must hold t.mu.
func (t *BroadcastChannelTransport) ensureChannel(name string) (js.Value, chan Envelope) {
	if ch, ok := t.channels[name]; ok {
		return ch, t.queues[name]
	}
	bcCtor := js.Global().Get("BroadcastChannel")
	if !bcCtor.Truthy() {
		// Fallback: return js.Null, the caller treats this as
		// "no transport available" — surfaces as Send error.
		return js.Null(), nil
	}
	bc := bcCtor.New(name)
	queue := make(chan Envelope, t.bufferSize)
	t.channels[name] = bc
	t.queues[name] = queue

	// Wire the onmessage callback. The callback runs on the JS event
	// loop; we push the envelope into the queue and fan out to subs.
	bc.Set("onmessage", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) < 1 {
			return nil
		}
		data := args[0].Get("data")
		env := jsToEnvelope(data)
		t.deliver(name, env)
		return nil
	}))

	return bc, queue
}

// deliver pushes an envelope to the mailbox queue + fans out to subs.
// Called from the BroadcastChannel onmessage handler.
//
// Non-blocking on the queue: if the buffer is full, the envelope is
// dropped (matches the no-back-pressure semantics of BroadcastChannel).
// The cognitive event log SHOULD record the drop in M3+.
func (t *BroadcastChannelTransport) deliver(mailbox string, env Envelope) {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return
	}
	queue := t.queues[mailbox]
	subs := append([]*localSubscriber(nil), t.subs[mailbox]...)
	t.mu.Unlock()

	// Try non-blocking push to the Recv queue.
	if queue != nil {
		select {
		case queue <- env:
		default:
			// Drop — buffer full. M3 scheduler will record the drop.
		}
	}

	// Fan out to subscribers.
	for _, s := range subs {
		select {
		case s.ch <- env:
		default:
			// Drop for slow subscribers.
		}
	}
}

// Send posts an envelope on the destination mailbox's BroadcastChannel.
//
// All listeners in OTHER browser tabs receive the message. The sender
// tab does NOT receive its own postMessage — that's the BroadcastChannel
// spec. So Send is fire-and-forget from the sender's perspective.
func (t *BroadcastChannelTransport) Send(env Envelope) error {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return ErrTransportClosed
	}
	bc, _ := t.ensureChannel(env.To)
	t.mu.Unlock()

	if !bc.Truthy() {
		return errors.New("broadcast_channel: BroadcastChannel Web API not available in this runtime")
	}

	bc.Call("postMessage", envelopeToJS(env))
	return nil
}

// Recv blocks on the named mailbox's arrival queue. Note: the sender
// tab does NOT receive its own sends — Recv only fires for messages
// from OTHER tabs.
func (t *BroadcastChannelTransport) Recv(mailbox string) (Envelope, error) {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return Envelope{}, ErrTransportClosed
	}
	_, queue := t.ensureChannel(mailbox)
	t.mu.Unlock()

	if queue == nil {
		return Envelope{}, errors.New("broadcast_channel: BroadcastChannel Web API not available")
	}

	select {
	case env := <-queue:
		return env, nil
	case <-t.closeChan:
		return Envelope{}, ErrTransportClosed
	}
}

// Subscribe registers a long-lived callback for arrivals from other tabs.
// Lifecycle matches LocalWorker.Subscribe — see transport.go for details.
func (t *BroadcastChannelTransport) Subscribe(mailbox string, onMsg func(Envelope)) (func(), error) {
	if onMsg == nil {
		return nil, errors.New("broadcast_channel: Subscribe requires non-nil onMsg")
	}
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return nil, ErrTransportClosed
	}
	_, _ = t.ensureChannel(mailbox) // wire the onmessage handler
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

// Close shuts down the transport. Closes every BroadcastChannel object
// and unblocks any pending Recv / Subscribe goroutines. Idempotent.
func (t *BroadcastChannelTransport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed {
		return nil
	}
	t.closed = true
	close(t.closeChan)
	for _, bc := range t.channels {
		bc.Call("close")
	}
	return nil
}

// ============================================================================
// JS ↔ Envelope codecs
// ============================================================================

// envelopeToJS serializes an Envelope to a JS object for postMessage.
func envelopeToJS(env Envelope) interface{} {
	return map[string]interface{}{
		"msg_id":  env.MsgID,
		"from":    env.From,
		"to":      env.To,
		"payload": string(env.Payload),
		"clock":   int(env.Clock),
	}
}

// jsToEnvelope parses a JS object from postMessage data into Envelope.
// Forgiving on missing fields — returns zero values.
func jsToEnvelope(data js.Value) Envelope {
	if data.Type() != js.TypeObject {
		return Envelope{}
	}
	env := Envelope{
		MsgID: jsStringField(data, "msg_id"),
		From:  jsStringField(data, "from"),
		To:    jsStringField(data, "to"),
	}
	if p := jsStringField(data, "payload"); p != "" {
		env.Payload = []byte(p)
	}
	if c := data.Get("clock"); c.Type() == js.TypeNumber {
		env.Clock = LamportValue(c.Float())
	}
	return env
}

// jsStringField extracts a string field; "" if missing/wrong-type.
func jsStringField(v js.Value, key string) string {
	f := v.Get(key)
	if f.Type() == js.TypeString {
		return f.String()
	}
	return ""
}

// avoid unused-import warning on non-js builds (defensive).
var _ = fmt.Sprintf
