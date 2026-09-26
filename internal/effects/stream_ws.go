package effects

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

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

	// Host-side credential binding (M-SERVEAPI-WS-BRIDGE D3): applied only
	// here, only after the authorizer passed, and only in Go.
	headers, err = applyStreamCredential(ctx.Stream, wsURL, headers)
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
		TLSClientConfig:  ctx.Stream.tlsClientConfig,
	})
	if err != nil {
		if errors.Is(err, ErrBackendNotRegistered) {
			return nil, fmt.Errorf("_stream_connect: %w", err)
		}
		return makeStreamErr("ConnectionFailed", "WebSocket dial failed: "+err.Error()), nil
	}

	id, err := registerWSConnection(ctx.Stream, transport)
	if err != nil {
		return makeStreamErr("ConnectionFailed", err.Error()), nil
	}
	return makeStreamOk(makeStreamConn(id)), nil
}

// AdoptStreamConnection registers an already-open transport — the server
// side of an accepted WebSocket upgrade — in sc and starts its read loop, so
// the program sees it as an ordinary StreamConn: same event buffer, same
// frame accounting, same transmit/onEvent/bridge ops as an outbound
// connection (M-SERVEAPI-WS-BRIDGE M1). On error the transport is closed.
func AdoptStreamConnection(sc *StreamContext, t StreamTransport) (int, error) {
	if sc == nil {
		_ = t.Close()
		return 0, fmt.Errorf("E_STREAM_NO_CONTEXT: Stream effect not configured (missing --caps Stream)")
	}
	return registerWSConnection(sc, t)
}

// registerWSConnection wraps transport as a StreamConnection, registers it in
// sc, delivers Opened and starts the read loop. Closes transport on error.
func registerWSConnection(sc *StreamContext, transport StreamTransport) (int, error) {
	conn := &StreamConnection{
		transport:   transport,
		protocol:    "WebSocket",
		status:      StreamStatusOpen,
		eventBuffer: make(chan streamEvent, sc.EventBufferSize),
		done:        make(chan struct{}),
		idleTimeout: sc.IdleTimeout,
		maxDuration: sc.MaxDuration,
		subprotocol: transport.Subprotocol(),
	}

	id, err := sc.AcquireConnection(conn)
	if err != nil {
		_ = transport.Close()
		return 0, err
	}

	// Deliver Opened BEFORE starting the read goroutine so it is always the
	// first event in the buffer (buffered channel → never blocks); otherwise
	// the reader can race an inbound message ahead of Opened.
	conn.eventBuffer <- streamEvent{
		kind: "opened",
		text: conn.subprotocol,
	}

	go conn.readLoop()
	return id, nil
}

// applyStreamCredential merges the host-side credential binding into the
// program's dial headers. The binding is consulted for wss:// only — never
// ws://, whatever the binder would say — and a program that supplies its own
// Authorization for a bound host is refused, so the credential has exactly
// one source. The returned map is a copy; the program's headers are not
// mutated. Error text never carries a header value.
func applyStreamCredential(sc *StreamContext, u *url.URL, headers map[string][]string) (map[string][]string, error) {
	if sc.Credentials == nil || u.Scheme != "wss" {
		return headers, nil
	}
	hdr, bound, err := sc.Credentials(u)
	if !bound {
		return headers, nil
	}
	for name := range headers {
		if strings.EqualFold(name, "Authorization") {
			return nil, fmt.Errorf("credential binding: Authorization header supplied by program for bound host %s (the operator's --stream-credential owns it; remove the header)", u.Host)
		}
	}
	if err != nil {
		return nil, fmt.Errorf("credential binding for %s: %v", u.Host, err)
	}
	merged := make(map[string][]string, len(headers)+len(hdr))
	for k, v := range headers {
		merged[k] = v
	}
	for k, v := range hdr {
		merged[k] = v
	}
	return merged, nil
}

// readLoop runs in a goroutine, reading WebSocket frames into the event buffer.
func (sc *StreamConnection) readLoop() {
	defer func() {
		// Deliver closed event if we exit normally
		sc.mu.Lock()
		status := sc.status
		sc.mu.Unlock()
		if status != StreamStatusClosed && status != StreamStatusClosing {
			sc.deliver(streamEvent{kind: "closed", code: 1006, reason: "connection lost"})
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
				sc.deliver(streamEvent{kind: "closed", code: ce.Code, reason: ce.Reason})
				return
			}
			sc.deliver(streamEvent{kind: "error", errType: "ProtocolError", text: err.Error()})
			return
		}

		switch frame.Kind {
		case StreamFrameText:
			sc.mu.Lock()
			sc.messagesRecv++
			sc.bytesRecv += int64(len(frame.Data))
			sc.mu.Unlock()
			if !sc.deliver(streamEvent{kind: "message", text: string(frame.Data), data: frame.Data}) {
				return
			}
		case StreamFrameBinary:
			sc.mu.Lock()
			sc.messagesRecv++
			sc.bytesRecv += int64(len(frame.Data))
			sc.mu.Unlock()
			if !sc.deliver(streamEvent{kind: "binary", data: frame.Data}) {
				return
			}
		case StreamFramePing:
			if !sc.deliver(streamEvent{kind: "ping", data: frame.Data}) {
				return
			}
		}
	}
}

// deliver queues evt for the consumer, blocking while the buffer is full —
// that block IS the backpressure — but never past Close: a reader stuck on a
// full buffer after its consumer went away used to leak forever. Reports
// false when the connection was closed instead.
func (sc *StreamConnection) deliver(evt streamEvent) bool {
	select {
	case sc.eventBuffer <- evt:
		return true
	case <-sc.done:
		return false
	}
}

// closeWS runs the transport's close handshake and releases the socket. A
// code of 0 means a normal closure; a transport that implements
// StreamCodeCloser sends the given code and reason.
func (sc *StreamConnection) closeWS(code int, reason string) {
	if sc.transport == nil {
		return
	}
	if code != 0 {
		if cc, ok := sc.transport.(StreamCodeCloser); ok {
			_ = cc.CloseWithCode(code, reason)
			return
		}
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
