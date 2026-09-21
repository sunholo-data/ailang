package effects

import (
	"errors"
	"fmt"
	"net/url"

	"github.com/sunholo-data/ailang/internal/eval"
)

// WebSocket side of the Stream effect. The wire library lives in
// internal/platform/streamws behind the StreamTransport seam
// (stream_transport.go); this file owns the connect op, the read goroutine
// and the frame accounting. stream.go holds the protocol-neutral connection
// state, the op surface and the SSE-shared helpers.

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

	// Authorize + resolve + validate the destination ONCE here and hand the
	// transport a dialer pinned to that address: the platform package never
	// resolves the name itself (M-EXECUTOR-POLICY-HARDENING M2).
	wsURL, err := url.Parse(urlVal.Value)
	if err != nil {
		return makeStreamErr("ConnectionFailed", fmt.Sprintf("E_STREAM_INVALID_URL: %v", err)), nil
	}
	dial, err := streamPolicy(ctx).pinnedDialer(wsURL)
	if err != nil {
		return makeStreamErr("ConnectionFailed", err.Error()), nil
	}

	// Dial through the registered transport. No registration is a harness
	// error (the binary forgot platform_init), not a program-visible result.
	transport, err := openStreamTransport(urlVal.Value, StreamDialConfig{
		URL:              urlVal.Value,
		Headers:          headers,
		Subprotocols:     subprotocols,
		HandshakeTimeout: ctx.Stream.ConnectTimeout,
		MaxFrameSize:     ctx.Stream.MaxFrameSize,
		DialContext:      dial,
	})
	if err != nil {
		if errors.Is(err, ErrBackendNotRegistered) {
			return nil, fmt.Errorf("_stream_connect: %w", err)
		}
		return makeStreamErr("ConnectionFailed", "WebSocket dial failed: "+err.Error()), nil
	}

	// Create connection
	conn := &StreamConnection{
		transport:   transport,
		protocol:    "WebSocket",
		status:      StreamStatusOpen,
		eventBuffer: make(chan streamEvent, ctx.Stream.EventBufferSize),
		done:        make(chan struct{}),
		idleTimeout: ctx.Stream.IdleTimeout,
		maxDuration: ctx.Stream.MaxDuration,
		subprotocol: transport.Subprotocol(),
	}

	// Register connection
	id, err := ctx.Stream.AcquireConnection(conn)
	if err != nil {
		_ = transport.Close()
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

		frame, err := sc.transport.Recv()
		if err != nil {
			var ce *StreamCloseError
			if errors.As(err, &ce) {
				sc.eventBuffer <- streamEvent{kind: "closed", code: ce.Code, reason: ce.Reason}
				return
			}
			sc.eventBuffer <- streamEvent{kind: "error", errType: "ProtocolError", text: err.Error()}
			return
		}

		switch frame.Kind {
		case StreamFrameText:
			sc.mu.Lock()
			sc.messagesRecv++
			sc.bytesRecv += int64(len(frame.Data))
			sc.mu.Unlock()
			sc.eventBuffer <- streamEvent{kind: "message", text: string(frame.Data)}
		case StreamFrameBinary:
			sc.mu.Lock()
			sc.messagesRecv++
			sc.bytesRecv += int64(len(frame.Data))
			sc.mu.Unlock()
			sc.eventBuffer <- streamEvent{kind: "binary", data: frame.Data}
		case StreamFramePing:
			sc.eventBuffer <- streamEvent{kind: "ping", data: frame.Data}
		}
	}
}

// closeWS runs the transport's close handshake and releases the socket.
func (sc *StreamConnection) closeWS() {
	if sc.transport == nil {
		return
	}
	_ = sc.transport.Close()
}

// writeWS writes one text (text=true) or binary frame. Caller holds sc.mu.
func (sc *StreamConnection) writeWS(text bool, data []byte) error {
	kind := StreamFrameBinary
	if text {
		kind = StreamFrameText
	}
	return sc.transport.Send(StreamFrame{Kind: kind, Data: data})
}
