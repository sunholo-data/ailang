package effects

import (
	"fmt"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sunholo-data/ailang/internal/eval"
)

// WebSocket transport for the Stream effect. Everything that touches
// gorilla/websocket lives in this file; stream.go holds the protocol-neutral
// connection state, the op surface and the SSE-shared helpers.

// wsConn is the WebSocket handle held by StreamConnection.
type wsConn = *websocket.Conn

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

	// Deliver Opened BEFORE starting the read goroutine so it is always the
	// first event in the buffer (buffered channel → never blocks); otherwise
	// the reader can race an inbound message ahead of Opened.
	conn.eventBuffer <- streamEvent{
		kind: "opened",
		text: conn.subprotocol,
	}

	// Start read goroutine
	go conn.readLoop()

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

// closeWS sends a close frame with a deadline, then closes the socket.
func (sc *StreamConnection) closeWS() {
	if sc.conn == nil {
		return
	}
	_ = sc.conn.WriteControl(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
		time.Now().Add(3*time.Second),
	)
	_ = sc.conn.Close()
}

// writeWS writes one text (text=true) or binary frame. Caller holds sc.mu.
func (sc *StreamConnection) writeWS(text bool, data []byte) error {
	msgType := websocket.BinaryMessage
	if text {
		msgType = websocket.TextMessage
	}
	return sc.conn.WriteMessage(msgType, data)
}
