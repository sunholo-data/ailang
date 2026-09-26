package effects

import (
	"fmt"
	"time"

	"github.com/sunholo-data/ailang/internal/eval"
)

// M-SERVEAPI-WS-BRIDGE M2: the verdict fold.
//
// bridge(client, up, init, step) relays frames between two WebSocket
// connections. Go owns the byte path: it dequeues one frame at a time from
// either side, asks the AILANG step function for a verdict, and applies it.
// The step is a Mealy machine — (state, frame) -> (state, verdict) — so the
// policy is a pure value-threading fold and can be replayed offline with
// foldl over a recorded frame list.
//
// Verdicts (std/stream/bridge Verdict):
//
//	Forward               re-send the ORIGINAL bytes to the other side
//	Drop                  send nothing
//	ReplaceText(s)        send s as a text frame to the other side
//	ReplaceBin(b)         send b as a binary frame to the other side
//	CloseBridge(code, r)  close the client with code/r, the upstream with 1000
//
// When bridge returns the upstream leg is closed, and so is the client leg
// unless the result is UpstreamClosed — then the handler still holds the
// client and may dial a new upstream. Both legs are released from the
// StreamContext only when closed.

// BridgeDecision is one line of the per-frame decision log (M4). It never
// carries a payload: direction, kind, size, verdict and step cost only.
type BridgeDecision struct {
	Seq     int    `json:"seq"`
	Dir     string `json:"dir"`  // "client" (client→upstream) or "up" (upstream→client)
	Kind    string `json:"kind"` // "text" or "bin"
	Bytes   int    `json:"bytes"`
	Verdict string `json:"verdict"`
	StepUS  int64  `json:"step_us"`
}

// bridgeLeg is one side of the bridge.
type bridgeLeg struct {
	id     int
	conn   *StreamConnection
	client bool
}

// bridgeRun holds the state of one bridge call.
type bridgeRun struct {
	ctx   *EffContext
	legs  [2]*bridgeLeg // [0] = client, [1] = upstream
	step  eval.Value
	seq   int
	rot   int
	logFn func(BridgeDecision)
}

// bridgeEnd is the Go form of the BridgeEnd ADT.
type bridgeEnd struct {
	ctor   string // ClientClosed | UpstreamClosed | ClosedByVerdict | TimedOut | StepFailed
	code   int
	reason string
}

func (e bridgeEnd) value() eval.Value {
	switch e.ctor {
	case "TimedOut", "StepFailed":
		return &eval.TaggedValue{CtorName: e.ctor, Fields: []eval.Value{&eval.StringValue{Value: e.reason}}}
	default:
		return &eval.TaggedValue{CtorName: e.ctor, Fields: []eval.Value{
			&eval.IntValue{Value: e.code}, &eval.StringValue{Value: e.reason},
		}}
	}
}

// StreamBridge implements Stream.bridge.
//
// Args: [client: StreamConn, up: StreamConn, init: s, step: (s, BridgeFrame) -> (s, Verdict)]
// Returns: (s, BridgeEnd)
func StreamBridge(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	if len(args) != 4 {
		return nil, fmt.Errorf("_stream_bridge: expected 4 arguments, got %d", len(args))
	}
	if ctx.Stream == nil {
		return nil, fmt.Errorf("E_STREAM_NO_CONTEXT: Stream effect not configured (missing --caps Stream)")
	}
	if ctx.FnCallerN == nil {
		return nil, fmt.Errorf("_stream_bridge: FnCallerN not set on EffContext (evaluator not wired)")
	}
	b := &bridgeRun{ctx: ctx, step: args[3], logFn: ctx.Stream.BridgeLog}
	for i, isClient := range []bool{true, false} {
		id, err := extractConnID(args[i])
		if err != nil {
			return nil, fmt.Errorf("_stream_bridge: %w", err)
		}
		conn, ok := ctx.Stream.GetConnection(id)
		if !ok {
			return nil, fmt.Errorf("_stream_bridge: connection %d not found (already disconnected?)", id)
		}
		if conn.protocol != "WebSocket" {
			return nil, fmt.Errorf("_stream_bridge: connection %d is %s; bridge relays WebSocket connections only", id, conn.protocol)
		}
		b.legs[i] = &bridgeLeg{id: id, conn: conn, client: isClient}
	}
	if b.legs[0].id == b.legs[1].id {
		return nil, fmt.Errorf("_stream_bridge: client and upstream are the same connection (%d)", b.legs[0].id)
	}

	state, end := b.run(args[2], ctx.Stream.IdleTimeout, ctx.Stream.MaxDuration)
	b.finish(end)
	return &eval.TupleValue{Elements: []eval.Value{state, end.value()}}, nil
}

// run is the fold: one frame, one step, one verdict, until an end.
func (b *bridgeRun) run(state eval.Value, idle, maxDur time.Duration) (eval.Value, bridgeEnd) {
	idleT := time.NewTimer(idle)
	defer idleT.Stop()
	maxT := time.NewTimer(maxDur)
	defer maxT.Stop()

	for {
		leg, evt, end, ok := b.next(idleT, maxT, idle, maxDur)
		if !ok {
			return state, end
		}
		resetTimer(idleT, idle)
		switch evt.kind {
		case "message", "binary":
			var stop bool
			state, end, stop = b.onFrame(state, leg, evt)
			if stop {
				return state, end
			}
		case "closed":
			return state, b.legClosed(leg, evt.code, evt.reason)
		case "error":
			return state, b.legClosed(leg, 1006, evt.text)
		default:
			// Opened / Ping: consumed by the loop, never shown to step.
		}
	}
}

// legClosed is the end for a leg the peer (or the network) closed.
func (b *bridgeRun) legClosed(leg *bridgeLeg, code int, reason string) bridgeEnd {
	if leg.client {
		return bridgeEnd{ctor: "ClientClosed", code: code, reason: reason}
	}
	return bridgeEnd{ctor: "UpstreamClosed", code: code, reason: reason}
}

// next dequeues the next event, round-robin across the two legs so neither
// starves (the selectEvents same-band rule), blocking when neither is ready.
func (b *bridgeRun) next(idleT, maxT *time.Timer, idle, maxDur time.Duration) (*bridgeLeg, streamEvent, bridgeEnd, bool) {
	for i := 0; i < 2; i++ {
		leg := b.legs[(b.rot+i)%2]
		select {
		case evt := <-leg.conn.eventBuffer:
			b.rot = (b.rot + i + 1) % 2
			return leg, evt, bridgeEnd{}, true
		default:
		}
	}
	c, u := b.legs[0], b.legs[1]
	select {
	case evt := <-c.conn.eventBuffer:
		b.rot = 1
		return c, evt, bridgeEnd{}, true
	case evt := <-u.conn.eventBuffer:
		b.rot = 0
		return u, evt, bridgeEnd{}, true
	case <-c.conn.done:
		return nil, streamEvent{}, b.legClosed(c, 1000, "closed locally"), false
	case <-u.conn.done:
		return nil, streamEvent{}, b.legClosed(u, 1000, "closed locally"), false
	case <-idleT.C:
		return nil, streamEvent{}, bridgeEnd{ctor: "TimedOut", reason: fmt.Sprintf("idle timeout after %s", idle)}, false
	case <-maxT.C:
		return nil, streamEvent{}, bridgeEnd{ctor: "TimedOut", reason: fmt.Sprintf("max duration %s exceeded", maxDur)}, false
	}
}

// onFrame runs step on one data frame and applies its verdict.
func (b *bridgeRun) onFrame(state eval.Value, leg *bridgeLeg, evt streamEvent) (eval.Value, bridgeEnd, bool) {
	if err := b.charge("stream.recv"); err != nil {
		return state, bridgeEnd{ctor: "StepFailed", reason: "BudgetExhausted: " + err.Error()}, true
	}
	text := evt.kind == "message"
	data := evt.data
	if text && data == nil {
		data = []byte(evt.text)
	}
	frame := bridgeFrame(leg.client, text, evt, data)

	b.seq++
	start := time.Now()
	next, verdict, err := b.callStep(state, frame)
	stepUS := time.Since(start).Microseconds()
	if err != nil {
		b.log(leg, text, len(data), "StepFailed", stepUS)
		return state, bridgeEnd{ctor: "StepFailed", reason: err.Error()}, true
	}
	b.log(leg, text, len(data), verdict.CtorName, stepUS)

	dest := b.legs[1]
	if !leg.client {
		dest = b.legs[0]
	}
	switch verdict.CtorName {
	case "Forward":
		end, stop := b.send(dest, text, data)
		return next, end, stop
	case "Drop":
		return next, bridgeEnd{}, false
	case "ReplaceText":
		s, ok := fieldString(verdict, 0)
		if !ok {
			return next, bridgeEnd{ctor: "StepFailed", reason: "ReplaceText: expected a string"}, true
		}
		end, stop := b.send(dest, true, []byte(s))
		return next, end, stop
	case "ReplaceBin":
		if len(verdict.Fields) != 1 {
			return next, bridgeEnd{ctor: "StepFailed", reason: "ReplaceBin: expected bytes"}, true
		}
		bv, ok := verdict.Fields[0].(*eval.BytesValue)
		if !ok {
			return next, bridgeEnd{ctor: "StepFailed", reason: fmt.Sprintf("ReplaceBin: expected bytes, got %T", verdict.Fields[0])}, true
		}
		end, stop := b.send(dest, false, bv.Value)
		return next, end, stop
	case "CloseBridge":
		return next, closeVerdict(verdict), true
	default:
		return next, bridgeEnd{ctor: "StepFailed", reason: "unknown verdict " + verdict.CtorName}, true
	}
}

// bridgeFrame builds the BridgeFrame ADT for one data frame. Binary frames
// arrive as bytes (the frame's own slice, not a copy) — G5 for the bridge.
func bridgeFrame(fromClient, text bool, evt streamEvent, data []byte) eval.Value {
	ctor := "Up"
	if fromClient {
		ctor = "Client"
	}
	if text {
		return &eval.TaggedValue{CtorName: ctor + "Text", Fields: []eval.Value{&eval.StringValue{Value: evt.text}}}
	}
	return &eval.TaggedValue{CtorName: ctor + "Bin", Fields: []eval.Value{&eval.BytesValue{Value: data}}}
}

// callStep invokes step(state, frame) with panic recovery and unpacks the
// (state, Verdict) tuple.
func (b *bridgeRun) callStep(state, frame eval.Value) (next eval.Value, verdict *eval.TaggedValue, err error) {
	var res eval.Value
	func() {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("step panic: %v", r)
			}
		}()
		res, err = b.ctx.FnCallerN(b.step, []eval.Value{state, frame})
	}()
	if err != nil {
		return nil, nil, fmt.Errorf("step failed: %w", err)
	}
	tup, ok := res.(*eval.TupleValue)
	if !ok || len(tup.Elements) != 2 {
		return nil, nil, fmt.Errorf("step must return (state, Verdict), got %T", res)
	}
	v, ok := tup.Elements[1].(*eval.TaggedValue)
	if !ok {
		return nil, nil, fmt.Errorf("step must return (state, Verdict), got verdict %T", tup.Elements[1])
	}
	return tup.Elements[0], v, nil
}

// send writes one frame to dest, charging Stream.send. A write failure ends
// the bridge as that leg closing.
func (b *bridgeRun) send(dest *bridgeLeg, text bool, data []byte) (bridgeEnd, bool) {
	if err := b.charge("stream.send"); err != nil {
		return bridgeEnd{ctor: "StepFailed", reason: "BudgetExhausted: " + err.Error()}, true
	}
	if int64(len(data)) > b.ctx.Stream.MaxMessageSize {
		return bridgeEnd{ctor: "StepFailed", reason: fmt.Sprintf("MessageTooLarge: frame of %d bytes exceeds limit %d", len(data), b.ctx.Stream.MaxMessageSize)}, true
	}
	c := dest.conn
	c.mu.Lock()
	err := c.writeWS(text, data)
	if err == nil {
		c.messagesSent++
		c.bytesSent += int64(len(data))
	}
	c.mu.Unlock()
	if err != nil {
		return b.legClosed(dest, 1006, "write failed: "+err.Error()), true
	}
	return bridgeEnd{}, false
}

// charge charges one Stream budget unit for a per-frame op. The builtin
// wrapper already opened a charge scope for the bridge call itself, which
// would swallow nested charges (M-BUDGET-SCOPING-BUG guard); each frame is a
// logical op of its own, so the scope is reset around the charge.
func (b *bridgeRun) charge(position string) error {
	prev := b.ctx.SaveAndResetBudgetChargeScope()
	defer b.ctx.RestoreBudgetChargeScope(prev)
	return b.ctx.RequireCapWithBudget("Stream", position)
}

// closeVerdict validates a CloseBridge(code, reason) verdict. Only codes an
// application may send are accepted (RFC 6455 §7.4): 1000, 1001, 1003,
// 1007–1011, and 3000–4999. Anything else is a StepFailed, not a quiet
// substitution.
func closeVerdict(v *eval.TaggedValue) bridgeEnd {
	if len(v.Fields) != 2 {
		return bridgeEnd{ctor: "StepFailed", reason: "CloseBridge: expected (int, string)"}
	}
	iv, ok1 := v.Fields[0].(*eval.IntValue)
	reason, ok2 := fieldString(v, 1)
	if !ok1 || !ok2 {
		return bridgeEnd{ctor: "StepFailed", reason: "CloseBridge: expected (int, string)"}
	}
	code := iv.Value
	valid := code == 1000 || code == 1001 || code == 1003 || (code >= 1007 && code <= 1011) || (code >= 3000 && code <= 4999)
	if !valid {
		return bridgeEnd{ctor: "StepFailed", reason: fmt.Sprintf("CloseBridge: %d is not a close code an application may send", code)}
	}
	return bridgeEnd{ctor: "ClosedByVerdict", code: code, reason: reason}
}

func fieldString(v *eval.TaggedValue, i int) (string, bool) {
	if i >= len(v.Fields) {
		return "", false
	}
	s, ok := v.Fields[i].(*eval.StringValue)
	if !ok {
		return "", false
	}
	return s.Value, true
}

// finish closes the legs the end calls for and releases them.
func (b *bridgeRun) finish(end bridgeEnd) {
	client, up := b.legs[0], b.legs[1]
	switch end.ctor {
	case "UpstreamClosed":
		// The handler keeps the client and may dial a new upstream.
	case "ClosedByVerdict":
		client.conn.CloseWithCode(end.code, end.reason)
	case "StepFailed":
		client.conn.CloseWithCode(1011, "bridge step failed")
	case "TimedOut":
		client.conn.CloseWithCode(1001, end.reason)
	default: // ClientClosed
		client.conn.Close()
	}
	up.conn.Close()
	b.ctx.Stream.ReleaseConnection(up.id)
	if end.ctor != "UpstreamClosed" {
		b.ctx.Stream.ReleaseConnection(client.id)
	}
}

// log emits one decision line when a decision log is attached.
func (b *bridgeRun) log(leg *bridgeLeg, text bool, n int, verdict string, stepUS int64) {
	if b.logFn == nil {
		return
	}
	dir, kind := "up", "bin"
	if leg.client {
		dir = "client"
	}
	if text {
		kind = "text"
	}
	b.logFn(BridgeDecision{Seq: b.seq, Dir: dir, Kind: kind, Bytes: n, Verdict: verdict, StepUS: stepUS})
}

func init() {
	RegisterOp("Stream", "bridge", StreamBridge)
}
