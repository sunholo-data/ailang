package cognition

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/effects"
)

// ============================================================================
// NativeMsgHandler — wraps a Transport for non-WASM use
// ============================================================================
//
// M-COG-RUNTIME M2 Day 9-10: bridges the in-process Transport (typically
// LocalWorker) into the effects.MsgHandler interface that internal/effects/
// expects. With this, AILANG code running in a CLI / test / non-browser
// environment can use the Msg effect just like in WASM.
//
// Optionally records every Send / Recv to a cognitive event log so the
// session is replayable (M3 scheduler picks up the log).
//
// The native handler does NOT modify internal/messaging/. The CLI-facing
// `ailang messages` commands remain byte-identical because the message
// store is consumed (not modified) here — the runtime API is a sibling
// (locked in the M1 design freeze).

// NativeMsgHandler implements effects.MsgHandler over a Transport.
//
// Embeds a Clock and optional EventLog so every send/recv gets a
// monotonic Lamport timestamp and is recorded for replay.
type NativeMsgHandler struct {
	transport Transport
	clock     *Clock
	log       *EventLog // optional; nil = no logging
	self      string    // sender NodeID for outgoing envelopes
	msgIDCtr  int64     // monotonic counter for envelope IDs
}

// NewNativeMsgHandler constructs a handler over the given transport.
//
// self identifies this node in outgoing envelopes (sender stamp).
// clock is the Lamport clock to stamp envelopes with (nil = create fresh).
// log is the cognitive event log to record into (nil = no recording).
func NewNativeMsgHandler(transport Transport, self string, clock *Clock, log *EventLog) *NativeMsgHandler {
	if clock == nil {
		clock = NewClock()
	}
	return &NativeMsgHandler{
		transport: transport,
		clock:     clock,
		log:       log,
		self:      self,
	}
}

// Send dispatches a payload to a mailbox via the underlying Transport.
//
// Lamport tick happens first (assigns clock to the outgoing envelope),
// then MessageSentEvent is logged (if a log is configured), then the
// transport delivers. If the transport fails, the log entry remains —
// callers can detect dropped sends by absence of the corresponding
// MessageReceivedEvent on the peer.
func (h *NativeMsgHandler) Send(to effects.Mailbox, payload []byte) (*effects.SendResult, error) {
	clock := h.clock.Tick()
	h.msgIDCtr++
	msgID := fmt.Sprintf("native_%d_%d", clock, h.msgIDCtr)

	if h.log != nil {
		_ = h.log.Append(MessageSentEvent{
			EventBase: NewEventBase("MessageSent", h.self, clock),
			To:        string(to),
			MsgID:     msgID,
		})
	}

	err := h.transport.Send(Envelope{
		MsgID:   msgID,
		From:    h.self,
		To:      string(to),
		Payload: append([]byte(nil), payload...), // defensive copy
		Clock:   clock,
	})
	if err != nil {
		return nil, fmt.Errorf("native_msg: transport send failed: %w", err)
	}

	return &effects.SendResult{
		MsgID:           effects.MsgID(msgID),
		Clock:           effects.LamportClock(clock),
		BudgetRemaining: -1, // unbounded in M2; M-CAPABILITY-BUDGETS hooks here in v0.22+
	}, nil
}

// Recv blocks for the next envelope in the named mailbox.
//
// On receipt, advances the local Lamport clock past the sender's clock
// (happens-before establishment) and logs a MessageReceivedEvent. The
// returned Message preserves the sender's clock — the M3 scheduler may
// re-sort events by sender clock during replay.
func (h *NativeMsgHandler) Recv(mailbox effects.Mailbox) (*effects.Message, error) {
	env, err := h.transport.Recv(string(mailbox))
	if err != nil {
		return nil, fmt.Errorf("native_msg: transport recv failed: %w", err)
	}

	// Happens-before: advance local clock past the sender's value.
	h.clock.Update(env.Clock)

	if h.log != nil {
		_ = h.log.Append(MessageReceivedEvent{
			EventBase: NewEventBase("MessageReceived", h.self, env.Clock),
			From:      env.From,
			MsgID:     env.MsgID,
		})
	}

	return &effects.Message{
		ID:      effects.MsgID(env.MsgID),
		From:    effects.NodeID(env.From),
		To:      effects.Mailbox(env.To),
		Payload: env.Payload,
		Clock:   effects.LamportClock(env.Clock),
	}, nil
}

// Subscribe registers a long-lived callback for arrivals in the named
// mailbox. The Go-side onMsg closure runs in the transport's subscriber
// goroutine — the FnCaller bridge (M3 scheduler scope) re-invokes
// AILANG closures on the evaluator's goroutine.
//
// For M2 callers that just want a Go-side callback (the test scenario,
// non-AILANG use), this works directly. For AILANG callback dispatch,
// M3's scheduler is the canonical consumer.
func (h *NativeMsgHandler) Subscribe(mailbox effects.Mailbox, onMsg func(effects.Message)) (func(), error) {
	cancel, err := h.transport.Subscribe(string(mailbox), func(env Envelope) {
		// Happens-before clock advance for each arrival (same as Recv).
		h.clock.Update(env.Clock)
		if h.log != nil {
			_ = h.log.Append(MessageReceivedEvent{
				EventBase: NewEventBase("MessageReceived", h.self, env.Clock),
				From:      env.From,
				MsgID:     env.MsgID,
			})
		}
		onMsg(effects.Message{
			ID:      effects.MsgID(env.MsgID),
			From:    effects.NodeID(env.From),
			To:      effects.Mailbox(env.To),
			Payload: env.Payload,
			Clock:   effects.LamportClock(env.Clock),
		})
	})
	if err != nil {
		return nil, fmt.Errorf("native_msg: transport subscribe failed: %w", err)
	}
	return cancel, nil
}
