// Package streamws is the gorilla/websocket transport for the Stream effect.
//
// The language core (internal/effects) defines the StreamTransport seam and
// never links a websocket library; this package implements the seam and the
// binary registers it once at startup via Register (cmd/ailang/platform_init.go).
// Part of M-V1-SIMPLIFICATION-PROGRAM Phase 1.3: the core is a leaf of the
// platform.
package streamws

import (
	"errors"
	"fmt"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sunholo-data/ailang/internal/effects"
)

// Register installs this transport for ws:// and wss:// URLs.
func Register() {
	effects.RegisterStreamTransport("ws", Open)
}

// Open dials a websocket per cfg. On a failed handshake the error carries the
// HTTP status when the server sent one.
func Open(cfg effects.StreamDialConfig) (effects.StreamTransport, error) {
	dialer := websocket.Dialer{
		HandshakeTimeout: cfg.HandshakeTimeout,
		Subprotocols:     cfg.Subprotocols,
		ReadBufferSize:   1024,
		WriteBufferSize:  1024,
	}
	conn, resp, err := dialer.Dial(cfg.URL, cfg.Headers)
	if err != nil {
		if resp != nil {
			return nil, fmt.Errorf("HTTP %d: %w", resp.StatusCode, err)
		}
		return nil, err
	}
	conn.SetReadLimit(cfg.MaxFrameSize)
	return &transport{conn: conn}, nil
}

// transport adapts one *websocket.Conn to effects.StreamTransport.
type transport struct {
	conn *websocket.Conn
}

func (t *transport) Subprotocol() string { return t.conn.Subprotocol() }

// Recv blocks for the next frame. A clean peer close (normal closure or
// going-away) is returned as *effects.StreamCloseError; any other error is
// returned unchanged for the core to report as a ProtocolError.
func (t *transport) Recv() (effects.StreamFrame, error) {
	for {
		frame, ok, err := t.recvOne()
		if err != nil || ok {
			return frame, err
		}
		// Unrecognised message type: the core's read loop skipped these; keep reading.
	}
}

func (t *transport) recvOne() (effects.StreamFrame, bool, error) {
	msgType, data, err := t.conn.ReadMessage()
	if err != nil {
		if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
			ce := &effects.StreamCloseError{Code: websocket.CloseNormalClosure}
			var wsErr *websocket.CloseError
			if errors.As(err, &wsErr) {
				ce.Code = wsErr.Code
				ce.Reason = wsErr.Text
			}
			return effects.StreamFrame{}, false, ce
		}
		return effects.StreamFrame{}, false, err
	}
	switch msgType {
	case websocket.TextMessage:
		return effects.StreamFrame{Kind: effects.StreamFrameText, Data: data}, true, nil
	case websocket.BinaryMessage:
		return effects.StreamFrame{Kind: effects.StreamFrameBinary, Data: data}, true, nil
	case websocket.PingMessage:
		return effects.StreamFrame{Kind: effects.StreamFramePing, Data: data}, true, nil
	default:
		return effects.StreamFrame{}, false, nil
	}
}

// Send writes one text or binary frame.
func (t *transport) Send(frame effects.StreamFrame) error {
	msgType := websocket.BinaryMessage
	if frame.Kind == effects.StreamFrameText {
		msgType = websocket.TextMessage
	}
	return t.conn.WriteMessage(msgType, frame.Data)
}

// Close sends a normal-closure frame with a 3s deadline, then closes the socket.
func (t *transport) Close() error {
	_ = t.conn.WriteControl(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
		time.Now().Add(3*time.Second),
	)
	return t.conn.Close()
}
