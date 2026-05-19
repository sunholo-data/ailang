package effects

import (
	"errors"
	"fmt"
	"sync"

	"github.com/sunholo-data/ailang/internal/eval"
)

// ============================================================================
// Msg Effect — Cognitive OS substrate (M-COG-RUNTIME, v0.21.x)
// ============================================================================
//
// The Msg effect provides a runtime messaging fabric for inter-agent
// communication within the Cognitive OS. The handler interface mirrors
// the step-pattern established by AIHandler (Call/Step/StepWithStream)
// in ai.go — see design_docs/planned/v0_21_0/m-cog-runtime.md.
//
// Key design principles (locked by the umbrella design freeze):
//   - Transport-independent: same agent code runs over LocalWorker /
//     BroadcastChannel / WebSocket / ... — semantics don't depend on transport
//   - Typed envelopes: NodeId + Mailbox identifiers, payload-as-bytes,
//     handler-assigned msgId, Lamport clock attached at send-time
//   - Subscribe streams arrivals back — the M3 deterministic scheduler
//     hooks this for clock-ordered event dispatch
//   - Pluggable handler: cmd/wasm/effects.go sets a transport-backed
//     handler at runtime; native CLI uses internal/messaging.Store
//
// The runtime Msg API is a SIBLING to the existing `ailang messages` CLI.
// CLI behavior remains byte-identical because internal/messaging/ is not
// modified — this file only consumes its types.

// ErrNoMsgHandler is returned when Msg ops are invoked without a configured handler.
//
// Configuration paths:
//   - Browser (WASM): set by cmd/wasm/effects.go via the WasmREPL init
//   - Native CLI: set by the runtime to a Store-backed handler (Day 5-6)
//   - Tests: ctx.Msg = NewMsgContext(NewStubMsgHandler())
var ErrNoMsgHandler = errors.New("no Msg handler configured — set one via cmd/wasm/effects.go (browser) or configure a StubMsgHandler for tests")

// ============================================================================
// Identity types — typed strings for clarity at the codec boundary
// ============================================================================

// NodeID identifies an agent node in a Cognitive OS topology.
//
// In single-tab LocalWorker scenarios, NodeIDs are agent-instance labels.
// In cross-tab BroadcastChannel scenarios, NodeIDs are origin-scoped.
// In distributed M-COG-MESH scenarios, NodeIDs are content-hashed pubkeys.
type NodeID string

// Mailbox addresses a destination — typically a NodeID, but Cognitive OS
// also supports topic-style mailboxes (e.g. "trace-reports", "patch-proposals")
// used by the M-COG-MESH 4-agent demo for fan-in routing.
type Mailbox string

// MsgID is a handler-assigned envelope identifier — globally unique within
// a runtime instance. The handler computes it (typically content-hash +
// clock + sender to ensure replay-determinism).
type MsgID string

// LamportClock is a non-decreasing logical timestamp. Tiebreaks via sender
// NodeID (handled by the M2 scheduler, not this layer). Stored as int64 to
// match the eventual event log on-wire format.
type LamportClock int64

// ============================================================================
// Message envelope — typed payload-as-bytes shape
// ============================================================================

// Message is the typed envelope flowing through the fabric.
//
// Payload is opaque bytes to this layer; AILANG code serializes/deserializes
// at the boundary via std/cognition bindings (Day 4). Keeping payload as
// []byte at the handler boundary means transports (Day 6+ in M-COG-MESH)
// don't need to know AILANG value shapes.
type Message struct {
	ID      MsgID
	From    NodeID
	To      Mailbox
	Payload []byte
	Clock   LamportClock
}

// SendResult is the typed return of Send — handler-assigned envelope ID,
// the Lamport clock the message was sent at, and budget telemetry.
//
// BudgetRemaining = -1 means unbounded (no budget configured); 0 means the
// next call will trap with a typed CapabilityExceeded event in M2's log.
type SendResult struct {
	MsgID           MsgID
	Clock           LamportClock
	BudgetRemaining int
}

// ============================================================================
// MsgHandler — pluggable step-pattern interface
// ============================================================================

// MsgHandler is the runtime contract for inter-agent messaging.
//
// The interface mirrors the AIHandler step pattern (ai.go:29). The three
// methods give us:
//   - Send: fire-and-record (like AIHandler.Call but with envelope shape)
//   - Recv: blocking single-message pull (analog of step-then-block patterns)
//   - Subscribe: long-lived streaming callback for the deterministic
//     scheduler — every arrival is a clock-ordered event in M2's event log
//
// Subscribe is what makes the fabric reflective. Without it, agents can
// only poll, which breaks determinism (poll timing varies between runs).
type MsgHandler interface {
	// Send dispatches a payload to a mailbox. Returns the handler-assigned
	// envelope ID, the clock the send happened at, and budget telemetry.
	// Returns an error for malformed addresses, budget exhaustion (which
	// surfaces as CapabilityExceeded in M2's log), or transport failure.
	Send(to Mailbox, payload []byte) (*SendResult, error)

	// Recv blocks for the next message in the named mailbox. Returns an
	// error if the handler is shut down (graceful cleanup contract — no
	// race-prone "channel closed" semantics at this layer).
	Recv(mailbox Mailbox) (*Message, error)

	// Subscribe registers a long-lived callback that fires for each message
	// arriving in the named mailbox. The returned cancel function
	// unregisters the callback and must be called to avoid resource leaks
	// (transport may hold a reference for the callback's lifetime).
	//
	// Analogous to AIHandler.StepWithStream's onChunk, but long-lived
	// rather than per-call. The M2 scheduler is the primary consumer.
	Subscribe(mailbox Mailbox, onMsg func(Message)) (cancel func(), err error)
}

// ============================================================================
// MsgContext — stored on EffContext.Msg
// ============================================================================

// MsgContext holds the active Msg handler + this node's own NodeID.
//
// Self identifies the node for sender-stamped envelopes; defaults to "self"
// in test scenarios and is set by cmd/wasm/effects.go (Day 5-6) or by the
// CLI wrapper (when the CLI Msg API ships in a later sprint).
type MsgContext struct {
	handler MsgHandler
	Self    NodeID
}

// NewMsgContext creates a context with the given handler and self identity.
func NewMsgContext(handler MsgHandler, self NodeID) *MsgContext {
	return &MsgContext{handler: handler, Self: self}
}

// Send delegates to the configured handler with a nil-handler check.
func (c *MsgContext) Send(to Mailbox, payload []byte) (*SendResult, error) {
	if c == nil || c.handler == nil {
		return nil, ErrNoMsgHandler
	}
	return c.handler.Send(to, payload)
}

// Recv delegates to the configured handler with a nil-handler check.
func (c *MsgContext) Recv(mailbox Mailbox) (*Message, error) {
	if c == nil || c.handler == nil {
		return nil, ErrNoMsgHandler
	}
	return c.handler.Recv(mailbox)
}

// Subscribe delegates to the configured handler with a nil-handler check.
func (c *MsgContext) Subscribe(mailbox Mailbox, onMsg func(Message)) (func(), error) {
	if c == nil || c.handler == nil {
		return nil, ErrNoMsgHandler
	}
	return c.handler.Subscribe(mailbox, onMsg)
}

// ============================================================================
// StubMsgHandler — deterministic in-memory handler for tests
// ============================================================================

// StubMsgHandler is a deterministic in-memory MsgHandler for tests.
//
// Mirrors StubDOMHandler (dom.go) and StubAIHandler (ai.go:226):
//   - Records every Send call (Sent field) for assertion
//   - Maintains per-mailbox FIFO queues for Recv
//   - Tracks subscribers per mailbox; Inject delivers to matching subs
//   - Assigns monotonic Lamport clock + sequential MsgID (test determinism)
//
// Self defaults to "stub_self" if not set via NewStubMsgHandler.
type StubMsgHandler struct {
	mu          sync.Mutex
	Self        NodeID
	Sent        []StubSentMsg
	mailboxes   map[Mailbox][]Message
	subscribers map[Mailbox][]*stubMsgSubscriber
	clock       LamportClock
	nextMsgID   int
}

// StubSentMsg records one Send call.
type StubSentMsg struct {
	To      Mailbox
	Payload []byte
	Result  *SendResult
}

type stubMsgSubscriber struct {
	mailbox Mailbox
	onMsg   func(Message)
}

// NewStubMsgHandler creates an empty stub handler with the given self identity.
//
// If self is "", defaults to "stub_self".
func NewStubMsgHandler(self NodeID) *StubMsgHandler {
	if self == "" {
		self = "stub_self"
	}
	return &StubMsgHandler{
		Self:        self,
		mailboxes:   make(map[Mailbox][]Message),
		subscribers: make(map[Mailbox][]*stubMsgSubscriber),
	}
}

// Send records the call, advances the clock, assigns a sequential MsgID,
// and enqueues the message in the destination mailbox so Recv can pull it.
// Also delivers to any subscribers of the destination mailbox.
func (h *StubMsgHandler) Send(to Mailbox, payload []byte) (*SendResult, error) {
	h.mu.Lock()
	h.clock++
	h.nextMsgID++
	id := MsgID(fmt.Sprintf("stub_msg_%d", h.nextMsgID))
	msg := Message{
		ID:      id,
		From:    h.Self,
		To:      to,
		Payload: append([]byte(nil), payload...), // defensive copy
		Clock:   h.clock,
	}
	h.mailboxes[to] = append(h.mailboxes[to], msg)
	res := &SendResult{
		MsgID:           id,
		Clock:           h.clock,
		BudgetRemaining: -1,
	}
	h.Sent = append(h.Sent, StubSentMsg{To: to, Payload: msg.Payload, Result: res})
	subs := append([]*stubMsgSubscriber(nil), h.subscribers[to]...)
	h.mu.Unlock()

	// Deliver to subscribers outside the lock to avoid deadlock on
	// re-entrant Send from inside a callback.
	for _, s := range subs {
		s.onMsg(msg)
	}
	return res, nil
}

// Recv pulls the next FIFO message from the mailbox. Non-blocking in the stub
// (unlike a real handler) — returns ErrNoMsgAvailable if empty so tests
// can assert exhaustion deterministically.
var ErrNoMsgAvailable = errors.New("no message available (stub does not block)")

// Recv pulls the head of the named mailbox or returns ErrNoMsgAvailable.
func (h *StubMsgHandler) Recv(mailbox Mailbox) (*Message, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	queue := h.mailboxes[mailbox]
	if len(queue) == 0 {
		return nil, ErrNoMsgAvailable
	}
	msg := queue[0]
	h.mailboxes[mailbox] = queue[1:]
	return &msg, nil
}

// Subscribe registers the callback for the named mailbox.
//
// The returned cancel removes the subscriber. New messages arriving via
// Send fire the callback in arrival order (test-deterministic since the
// stub serializes Send through its mutex).
func (h *StubMsgHandler) Subscribe(mailbox Mailbox, onMsg func(Message)) (func(), error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	sub := &stubMsgSubscriber{mailbox: mailbox, onMsg: onMsg}
	h.subscribers[mailbox] = append(h.subscribers[mailbox], sub)
	cancel := func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		subs := h.subscribers[mailbox]
		for i, s := range subs {
			if s == sub {
				h.subscribers[mailbox] = append(subs[:i], subs[i+1:]...)
				return
			}
		}
	}
	return cancel, nil
}

// ============================================================================
// Ops — registered with the effect registry; called from AILANG code
// ============================================================================

// init registers Msg effect operations.
//
// Bare ops (send, recv) are the M1-shippable surface. Result-returning
// variants (sendResult, recvResult) land in Day 4 alongside DOM's variants.
// The subscribe op (which needs ctx.FnCaller to call AILANG closures) lands
// in Day 5-6 alongside cmd/wasm/effects.go wiring.
func init() {
	RegisterOp("Msg", "send", msgSend)
	RegisterOp("Msg", "recv", msgRecv)
}

// msgSend implements Msg.send(to: string, payload: string) -> {msg_id: string, clock: int, budget_remaining: int}
//
// Payload is accepted as a string at this layer; M2 may extend with a
// bytes-payload variant once std/bytes integration is decided.
func msgSend(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("E_MSG_TYPE_ERROR: send: expected 2 arguments, got %d", len(args))
	}
	to, err := msgStringArg(args[0], "to")
	if err != nil {
		return nil, fmt.Errorf("E_MSG_TYPE_ERROR: send: %w", err)
	}
	payload, err := msgStringArg(args[1], "payload")
	if err != nil {
		return nil, fmt.Errorf("E_MSG_TYPE_ERROR: send: %w", err)
	}
	res, err := ctx.Msg.Send(Mailbox(to), []byte(payload))
	if err != nil {
		return nil, err
	}
	return encodeSendResult(res), nil
}

// msgRecv implements Msg.recv(mailbox: string) -> {msg_id: string, from: string, payload: string, clock: int}
//
// Returns a typed error on empty mailbox in the stub case; the real
// browser-bridged handler will block until a message arrives or the
// transport closes.
func msgRecv(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("E_MSG_TYPE_ERROR: recv: expected 1 argument, got %d", len(args))
	}
	mailbox, err := msgStringArg(args[0], "mailbox")
	if err != nil {
		return nil, fmt.Errorf("E_MSG_TYPE_ERROR: recv: %w", err)
	}
	msg, err := ctx.Msg.Recv(Mailbox(mailbox))
	if err != nil {
		return nil, err
	}
	return encodeMessage(msg), nil
}

// ============================================================================
// Codecs — AILANG values <-> Go types
// ============================================================================

// msgStringArg extracts a string argument by name.
func msgStringArg(v eval.Value, name string) (string, error) {
	s, ok := v.(*eval.StringValue)
	if !ok {
		return "", fmt.Errorf("expected %s as string, got %T", name, v)
	}
	return s.Value, nil
}

// encodeSendResult builds the AILANG record returned by send.
func encodeSendResult(r *SendResult) eval.Value {
	return &eval.RecordValue{
		Fields: map[string]eval.Value{
			"msg_id":           &eval.StringValue{Value: string(r.MsgID)},
			"clock":            &eval.IntValue{Value: int(r.Clock)},
			"budget_remaining": &eval.IntValue{Value: r.BudgetRemaining},
		},
	}
}

// encodeMessage builds the AILANG record returned by recv.
//
// Payload is exposed as a string for now; bytes variant lands when std/bytes
// integration is decided (likely M2 when transport semantics solidify).
func encodeMessage(m *Message) eval.Value {
	return &eval.RecordValue{
		Fields: map[string]eval.Value{
			"msg_id":  &eval.StringValue{Value: string(m.ID)},
			"from":    &eval.StringValue{Value: string(m.From)},
			"to":      &eval.StringValue{Value: string(m.To)},
			"payload": &eval.StringValue{Value: string(m.Payload)},
			"clock":   &eval.IntValue{Value: int(m.Clock)},
		},
	}
}
