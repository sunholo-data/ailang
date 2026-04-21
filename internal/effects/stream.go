package effects

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sunholo-data/ailang/internal/eval"
)

// StreamStatus represents the connection state
type StreamStatus int

const (
	StreamStatusConnecting StreamStatus = iota
	StreamStatusOpen
	StreamStatusClosing
	StreamStatusClosed
)

func (s StreamStatus) String() string {
	switch s {
	case StreamStatusConnecting:
		return "Connecting"
	case StreamStatusOpen:
		return "Open"
	case StreamStatusClosing:
		return "Closing"
	case StreamStatusClosed:
		return "StreamClosed"
	default:
		return "Unknown"
	}
}

// StreamConnection holds per-connection state for a WebSocket or SSE connection.
type StreamConnection struct {
	mu           sync.Mutex
	conn         *websocket.Conn // WebSocket only
	httpResp     *http.Response  // SSE only: HTTP response for body close
	protocol     string          // "WebSocket" or "SSE"
	lastEventID  string          // SSE only: last received id: field
	status       StreamStatus
	handler      eval.Value // AILANG handler function (StreamEvent -> bool ! {e})
	eventBuffer  chan streamEvent
	done         chan struct{} // Signals read goroutine to stop
	idleTimeout  time.Duration
	maxDuration  time.Duration
	subprotocol  string
	messagesSent int
	messagesRecv int
	bytesSent    int64
	bytesRecv    int64
}

// streamEvent is an internal representation of a stream event before conversion to AILANG ADT.
type streamEvent struct {
	kind         string // "message", "binary", "opened", "closed", "error", "ping", "sse_data", "source_text", "source_bytes"
	text         string
	data         []byte
	code         int
	reason       string
	errType      string // "ConnectionFailed", "Timeout", "BudgetExhausted", "ProtocolError", etc.
	sseEventType string // SSE event: field (e.g. "content_block_delta", "message_stop")
	sseID        string // SSE id: field
	sourceName   string // M-ASYNC-IO: source tag for SourceText/SourceBytes events
}

// Close gracefully shuts down the connection.
func (sc *StreamConnection) Close() {
	sc.mu.Lock()
	if sc.status == StreamStatusClosed || sc.status == StreamStatusClosing {
		sc.mu.Unlock()
		return
	}
	sc.status = StreamStatusClosing
	sc.mu.Unlock()

	// Protocol-specific close
	if sc.protocol == "SSE" {
		// SSE: close the HTTP response body
		if sc.httpResp != nil && sc.httpResp.Body != nil {
			_ = sc.httpResp.Body.Close()
		}
	} else {
		// WebSocket: send close frame with deadline
		if sc.conn != nil {
			_ = sc.conn.WriteControl(
				websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
				time.Now().Add(3*time.Second),
			)
			_ = sc.conn.Close()
		}
	}

	// Signal read goroutine to stop
	select {
	case <-sc.done:
	default:
		close(sc.done)
	}

	sc.mu.Lock()
	sc.status = StreamStatusClosed
	sc.mu.Unlock()
}

// Status returns the current connection status.
func (sc *StreamConnection) Status() StreamStatus {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	return sc.status
}

func init() {
	RegisterOp("Stream", "connect", StreamConnect)
	RegisterOp("Stream", "send", StreamSend)
	RegisterOp("Stream", "onEvent", StreamOnEvent)
	RegisterOp("Stream", "runEventLoop", StreamRunEventLoop)
	RegisterOp("Stream", "close", StreamClose)
	RegisterOp("Stream", "status", StreamGetStatus)

	// M-ASYNC-IO: Multi-source event multiplexing
	RegisterOp("Stream", "sourceOfConn", StreamSourceOfConn)
	RegisterOp("Stream", "asyncReadStdinLines", StreamAsyncReadStdinLines)
	RegisterOp("Stream", "selectEvents", StreamSelectEvents)
	RegisterOp("Stream", "asyncExecProcess", StreamAsyncExecProcess)
}

// StreamConnect establishes a WebSocket connection.
//
// Args: [url: string, config: record{protocol, headers, subprotocols}]
// Returns: Result[StreamConn(int), StreamError]
func StreamConnect(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("_stream_connect: expected 2 arguments, got %d", len(args))
	}

	urlVal, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("_stream_connect: expected String for url, got %T", args[0])
	}

	if ctx.Stream == nil {
		return nil, fmt.Errorf("E_STREAM_NO_CONTEXT: Stream effect not configured (missing --caps Stream)")
	}

	// Budget check: Stream.connect consumes one budget unit (after capability check)
	if err := ctx.RequireCapWithBudget("Stream", "stream.connect"); err != nil {
		return makeStreamErr("BudgetExhausted", err.Error()), nil
	}

	// Validate URL against security policy
	if err := ctx.Stream.ValidateURL(urlVal.Value); err != nil {
		return makeStreamErr("ConnectionFailed", err.Error()), nil
	}

	// Parse config record for headers and subprotocols
	headers := make(map[string][]string)
	var subprotocols []string

	if configRec, ok := args[1].(*eval.RecordValue); ok {
		if hdrs, ok := configRec.Fields["headers"]; ok {
			if hdrList, ok := hdrs.(*eval.ListValue); ok {
				for _, hdr := range hdrList.Elements {
					if hdrRec, ok := hdr.(*eval.RecordValue); ok {
						nameVal, _ := hdrRec.Fields["name"].(*eval.StringValue)
						valVal, _ := hdrRec.Fields["value"].(*eval.StringValue)
						if nameVal != nil && valVal != nil {
							headers[nameVal.Value] = append(headers[nameVal.Value], valVal.Value)
						}
					}
				}
			}
		}
		if subs, ok := configRec.Fields["subprotocols"]; ok {
			if subList, ok := subs.(*eval.ListValue); ok {
				for _, s := range subList.Elements {
					if sv, ok := s.(*eval.StringValue); ok {
						subprotocols = append(subprotocols, sv.Value)
					}
				}
			}
		}
	}

	// Dial WebSocket
	dialer := websocket.Dialer{
		HandshakeTimeout: ctx.Stream.ConnectTimeout,
		Subprotocols:     subprotocols,
		ReadBufferSize:   1024,
		WriteBufferSize:  1024,
	}

	wsConn, resp, err := dialer.Dial(urlVal.Value, headers)
	if err != nil {
		msg := fmt.Sprintf("WebSocket dial failed: %s", err.Error())
		if resp != nil {
			msg = fmt.Sprintf("WebSocket dial failed (HTTP %d): %s", resp.StatusCode, err.Error())
		}
		return makeStreamErr("ConnectionFailed", msg), nil
	}

	// Set read limit
	wsConn.SetReadLimit(ctx.Stream.MaxFrameSize)

	// Create connection
	conn := &StreamConnection{
		conn:        wsConn,
		protocol:    "WebSocket",
		status:      StreamStatusOpen,
		eventBuffer: make(chan streamEvent, ctx.Stream.EventBufferSize),
		done:        make(chan struct{}),
		idleTimeout: ctx.Stream.IdleTimeout,
		maxDuration: ctx.Stream.MaxDuration,
		subprotocol: wsConn.Subprotocol(),
	}

	// Register connection
	id, err := ctx.Stream.AcquireConnection(conn)
	if err != nil {
		wsConn.Close()
		return makeStreamErr("ConnectionFailed", err.Error()), nil
	}

	// Start read goroutine
	go conn.readLoop()

	// Deliver Opened event
	conn.eventBuffer <- streamEvent{
		kind: "opened",
		text: conn.subprotocol,
	}

	// Return Ok(StreamConn(id))
	return makeStreamOk(makeStreamConn(id)), nil
}

// readLoop runs in a goroutine, reading WebSocket frames into the event buffer.
func (sc *StreamConnection) readLoop() {
	defer func() {
		// Deliver closed event if we exit normally
		sc.mu.Lock()
		status := sc.status
		sc.mu.Unlock()
		if status != StreamStatusClosed && status != StreamStatusClosing {
			sc.eventBuffer <- streamEvent{kind: "closed", code: 1006, reason: "connection lost"}
		}
	}()

	for {
		select {
		case <-sc.done:
			return
		default:
		}

		msgType, data, err := sc.conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				closeCode := websocket.CloseNormalClosure
				closeReason := ""
				if ce, ok := err.(*websocket.CloseError); ok {
					closeCode = ce.Code
					closeReason = ce.Text
				}
				sc.eventBuffer <- streamEvent{kind: "closed", code: closeCode, reason: closeReason}
				return
			}
			sc.eventBuffer <- streamEvent{kind: "error", errType: "ProtocolError", text: err.Error()}
			return
		}

		switch msgType {
		case websocket.TextMessage:
			sc.mu.Lock()
			sc.messagesRecv++
			sc.bytesRecv += int64(len(data))
			sc.mu.Unlock()
			sc.eventBuffer <- streamEvent{kind: "message", text: string(data)}
		case websocket.BinaryMessage:
			sc.mu.Lock()
			sc.messagesRecv++
			sc.bytesRecv += int64(len(data))
			sc.mu.Unlock()
			sc.eventBuffer <- streamEvent{kind: "binary", data: data}
		case websocket.PingMessage:
			sc.eventBuffer <- streamEvent{kind: "ping", data: data}
		}
	}
}

// streamSend sends a message on a connection.
//
// Args: [conn: StreamConn(int), msg: StreamMessage(Text(string)|Bin(bytes))]
// Returns: Result[unit, StreamError]
func StreamSend(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("_stream_send: expected 2 arguments, got %d", len(args))
	}

	connID, err := extractConnID(args[0])
	if err != nil {
		return nil, err
	}

	conn, ok := ctx.Stream.GetConnection(connID)
	if !ok {
		return makeStreamErr("ConnectionFailed", "connection not found"), nil
	}

	if conn.Status() != StreamStatusOpen {
		return makeStreamErr("ConnectionFailed", "connection not open"), nil
	}

	// SSE connections are read-only — no silent fallback
	if conn.protocol == "SSE" {
		return makeStreamErr("ProtocolError", "SSE connections are read-only; use WebSocket for bidirectional messaging"), nil
	}

	// Budget check: Stream.send consumes one budget unit (after connection validation)
	if err := ctx.RequireCapWithBudget("Stream", "stream.send"); err != nil {
		return makeStreamErr("BudgetExhausted", err.Error()), nil
	}

	// Extract message variant — auto-wrap plain strings as Text(string)
	var adt *eval.TaggedValue
	if sv, ok := args[1].(*eval.StringValue); ok {
		adt = &eval.TaggedValue{
			CtorName: "Text",
			Fields:   []eval.Value{sv},
		}
	} else if tagged, ok := args[1].(*eval.TaggedValue); ok {
		adt = tagged
	} else {
		return nil, fmt.Errorf("_stream_send: expected string or StreamMessage ADT (Text/Bin), got %T", args[1])
	}

	conn.mu.Lock()
	defer conn.mu.Unlock()

	switch adt.CtorName {
	case "Text":
		if len(adt.Fields) < 1 {
			return nil, fmt.Errorf("_stream_send: Text requires 1 field")
		}
		sv, ok := adt.Fields[0].(*eval.StringValue)
		if !ok {
			return nil, fmt.Errorf("_stream_send: Text field must be string")
		}
		data := []byte(sv.Value)
		if int64(len(data)) > ctx.Stream.MaxMessageSize {
			return makeStreamErr("MessageTooLarge",
				fmt.Sprintf("message size %d exceeds limit %d", len(data), ctx.Stream.MaxMessageSize)), nil
		}
		if err := conn.conn.WriteMessage(websocket.TextMessage, data); err != nil {
			return makeStreamErr("ProtocolError", err.Error()), nil
		}
		conn.messagesSent++
		conn.bytesSent += int64(len(data))

	case "Bin":
		if len(adt.Fields) < 1 {
			return nil, fmt.Errorf("_stream_send: Bin requires 1 field")
		}
		bv, ok := adt.Fields[0].(*eval.BytesValue)
		if !ok {
			return nil, fmt.Errorf("_stream_send: Bin field must be bytes")
		}
		if int64(len(bv.Value)) > ctx.Stream.MaxMessageSize {
			return makeStreamErr("MessageTooLarge",
				fmt.Sprintf("message size %d exceeds limit %d", len(bv.Value), ctx.Stream.MaxMessageSize)), nil
		}
		if err := conn.conn.WriteMessage(websocket.BinaryMessage, bv.Value); err != nil {
			return makeStreamErr("ProtocolError", err.Error()), nil
		}
		conn.messagesSent++
		conn.bytesSent += int64(len(bv.Value))

	default:
		return nil, fmt.Errorf("_stream_send: unknown StreamMessage variant: %s", adt.CtorName)
	}

	return makeStreamOkUnit(), nil
}

// streamOnEvent registers an event handler.
//
// Args: [conn: StreamConn(int), handler: StreamEvent -> bool]
// Returns: unit
func StreamOnEvent(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("_stream_onEvent: expected 2 arguments, got %d", len(args))
	}

	connID, err := extractConnID(args[0])
	if err != nil {
		return nil, err
	}

	conn, ok := ctx.Stream.GetConnection(connID)
	if !ok {
		return nil, fmt.Errorf("_stream_onEvent: connection %d not found", connID)
	}

	conn.mu.Lock()
	conn.handler = args[1]
	conn.mu.Unlock()

	return &eval.UnitValue{}, nil
}

// streamRunEventLoop blocks, dispatching events to the registered handler.
//
// M-ASYNC-IO: Now delegates to selectEventsLoop with a single connSource.
// This makes runEventLoop sugar over the general-purpose multiplexer,
// preserving backward compatibility while sharing the same implementation.
//
// Args: [conn: StreamConn(int)]
// Returns: unit
func StreamRunEventLoop(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("_stream_runEventLoop: expected 1 argument, got %d", len(args))
	}

	connID, err := extractConnID(args[0])
	if err != nil {
		return nil, err
	}

	conn, ok := ctx.Stream.GetConnection(connID)
	if !ok {
		return nil, fmt.Errorf("_stream_runEventLoop: connection %d not found", connID)
	}

	conn.mu.Lock()
	handler := conn.handler
	conn.mu.Unlock()

	if handler == nil {
		return nil, fmt.Errorf("_stream_runEventLoop: no handler registered (call onEvent first)")
	}

	if ctx.FnCaller == nil {
		return nil, fmt.Errorf("_stream_runEventLoop: FnCaller not set on EffContext (evaluator not wired)")
	}

	// Wrap the connection as a single EventSource and delegate to selectEventsLoop
	source := NewConnSource(conn, fmt.Sprintf("conn:%d", connID), 0)
	sources := []EventSource{source}

	err = selectEventsLoop(sources, handler, ctx.FnCaller, conn.idleTimeout, conn.maxDuration)
	if err != nil {
		return nil, fmt.Errorf("_stream_runEventLoop: %w", err)
	}

	return &eval.UnitValue{}, nil
}

// streamClose closes a connection.
func StreamClose(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("_stream_close: expected 1 argument, got %d", len(args))
	}

	connID, err := extractConnID(args[0])
	if err != nil {
		return nil, err
	}

	conn, ok := ctx.Stream.GetConnection(connID)
	if !ok {
		return &eval.UnitValue{}, nil // Already closed
	}

	conn.Close()
	ctx.Stream.ReleaseConnection(connID)

	return &eval.UnitValue{}, nil
}

// StreamGetStatus returns connection status as a StreamStatus ADT value.
func StreamGetStatus(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("_stream_status: expected 1 argument, got %d", len(args))
	}

	connID, err := extractConnID(args[0])
	if err != nil {
		return nil, err
	}

	conn, ok := ctx.Stream.GetConnection(connID)
	if !ok {
		return &eval.TaggedValue{CtorName: "StreamClosed"}, nil
	}

	status := conn.Status()
	return &eval.TaggedValue{CtorName: status.String()}, nil
}

// --- Helper functions ---

// extractConnID extracts the integer ID from a StreamConn(int) ADT value.
// Handles both direct StreamConn(int) and Result-wrapped Ok(StreamConn(int)).
// Returns a descriptive error if Err(...) is passed.
func extractConnID(v eval.Value) (int, error) {
	adt, ok := v.(*eval.TaggedValue)
	if !ok {
		return 0, fmt.Errorf("expected StreamConn(int) or Ok(StreamConn(int)), got %T", v)
	}

	// Layer 1: Direct StreamConn(int)
	if adt.CtorName == "StreamConn" && len(adt.Fields) >= 1 {
		return extractIntFromStreamConn(adt)
	}

	// Layer 2: Ok(StreamConn(int)) — auto-unwrap Result
	if adt.CtorName == "Ok" && len(adt.Fields) >= 1 {
		inner, ok := adt.Fields[0].(*eval.TaggedValue)
		if !ok {
			return 0, fmt.Errorf("expected Ok(StreamConn(int)), got Ok(%T)", adt.Fields[0])
		}
		if inner.CtorName == "StreamConn" && len(inner.Fields) >= 1 {
			return extractIntFromStreamConn(inner)
		}
		return 0, fmt.Errorf("expected Ok(StreamConn(int)), got Ok(%s(...))", inner.CtorName)
	}

	// Layer 3: Err(...) — provide a descriptive error
	if adt.CtorName == "Err" && len(adt.Fields) >= 1 {
		if errAdt, ok := adt.Fields[0].(*eval.TaggedValue); ok {
			errMsg := errAdt.CtorName
			if len(errAdt.Fields) > 0 {
				if sv, ok := errAdt.Fields[0].(*eval.StringValue); ok {
					errMsg = fmt.Sprintf("%s(%s)", errAdt.CtorName, sv.Value)
				}
			}
			return 0, fmt.Errorf("stream connection failed: %s", errMsg)
		}
		return 0, fmt.Errorf("stream connection failed: Err(%v)", adt.Fields[0])
	}

	return 0, fmt.Errorf("expected StreamConn(int) or Ok(StreamConn(int)), got %s", adt.CtorName)
}

// extractIntFromStreamConn extracts the int ID from a StreamConn(int) TaggedValue.
func extractIntFromStreamConn(adt *eval.TaggedValue) (int, error) {
	intVal, ok := adt.Fields[0].(*eval.IntValue)
	if !ok {
		return 0, fmt.Errorf("expected StreamConn(int), got StreamConn(%T)", adt.Fields[0])
	}
	return int(intVal.Value), nil
}

// makeStreamConn creates a StreamConn(id) ADT value.
func makeStreamConn(id int) eval.Value {
	return &eval.TaggedValue{
		CtorName: "StreamConn",
		Fields:   []eval.Value{&eval.IntValue{Value: id}},
	}
}

// makeStreamErr creates an Err(StreamError) Result value.
func makeStreamErr(variant, msg string) eval.Value {
	return &eval.TaggedValue{
		CtorName: "Err",
		Fields: []eval.Value{
			&eval.TaggedValue{
				CtorName: variant,
				Fields:   []eval.Value{&eval.StringValue{Value: msg}},
			},
		},
	}
}

// makeStreamOk creates an Ok(value) Result value.
func makeStreamOk(val eval.Value) eval.Value {
	return &eval.TaggedValue{
		CtorName: "Ok",
		Fields:   []eval.Value{val},
	}
}

// makeStreamOkUnit creates an Ok(()) Result value.
func makeStreamOkUnit() eval.Value {
	return &eval.TaggedValue{
		CtorName: "Ok",
		Fields:   []eval.Value{&eval.UnitValue{}},
	}
}

// eventToADT converts an internal streamEvent to an AILANG StreamEvent ADT value.
func eventToADT(evt streamEvent) eval.Value {
	switch evt.kind {
	case "message":
		return &eval.TaggedValue{
			CtorName: "Message",
			Fields:   []eval.Value{&eval.StringValue{Value: evt.text}},
		}
	case "binary":
		return &eval.TaggedValue{
			CtorName: "Binary",
			Fields:   []eval.Value{&eval.StringValue{Value: string(evt.data)}},
		}
	case "opened":
		return &eval.TaggedValue{
			CtorName: "Opened",
			Fields:   []eval.Value{&eval.StringValue{Value: evt.text}},
		}
	case "closed":
		return &eval.TaggedValue{
			CtorName: "Closed",
			Fields: []eval.Value{
				&eval.IntValue{Value: evt.code},
				&eval.StringValue{Value: evt.reason},
			},
		}
	case "error":
		return &eval.TaggedValue{
			CtorName: "StreamError",
			Fields: []eval.Value{
				&eval.TaggedValue{
					CtorName: evt.errType,
					Fields:   []eval.Value{&eval.StringValue{Value: evt.text}},
				},
			},
		}
	case "ping":
		return &eval.TaggedValue{
			CtorName: "Ping",
			Fields:   []eval.Value{&eval.StringValue{Value: string(evt.data)}},
		}
	case "sse_data":
		// SSE data event: SSEData(eventType, data)
		// eventType carries the SSE event: field (e.g. "content_block_delta")
		return &eval.TaggedValue{
			CtorName: "SSEData",
			Fields: []eval.Value{
				&eval.StringValue{Value: evt.sseEventType},
				&eval.StringValue{Value: evt.text},
			},
		}
	case "source_text":
		// M-ASYNC-IO: SourceText(sourceName, text)
		return &eval.TaggedValue{
			CtorName: "SourceText",
			Fields: []eval.Value{
				&eval.StringValue{Value: evt.sourceName},
				&eval.StringValue{Value: evt.text},
			},
		}
	case "source_bytes":
		// M-ASYNC-IO: SourceBytes(sourceName, bytes)
		return &eval.TaggedValue{
			CtorName: "SourceBytes",
			Fields: []eval.Value{
				&eval.StringValue{Value: evt.sourceName},
				&eval.BytesValue{Value: evt.data},
			},
		}
	default:
		return &eval.TaggedValue{
			CtorName: "StreamError",
			Fields: []eval.Value{
				&eval.TaggedValue{
					CtorName: "ProtocolError",
					Fields:   []eval.Value{&eval.StringValue{Value: "unknown event: " + evt.kind}},
				},
			},
		}
	}
}

// callHandlerSafe calls an AILANG handler with panic recovery.
// Uses the FnCaller callback set on EffContext to invoke the handler function.
// Returns (shouldContinue, error).
func callHandlerSafe(fnCaller func(eval.Value, eval.Value) (eval.Value, error), handler eval.Value, event eval.Value) (bool, error) {
	var result eval.Value
	var callErr error

	func() {
		defer func() {
			if r := recover(); r != nil {
				callErr = fmt.Errorf("handler panic: %v", r)
			}
		}()
		result, callErr = fnCaller(handler, event)
	}()

	if callErr != nil {
		return false, callErr
	}

	// Check if handler returned true (continue) or false (stop)
	if bv, ok := result.(*eval.BoolValue); ok {
		return bv.Value, nil
	}

	// If not a bool, treat as continue
	return true, nil
}
