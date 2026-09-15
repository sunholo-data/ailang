package effects

import (
	"errors"
	"fmt"
	"net/url"
	"sync"
	"time"
)

// ErrBackendNotRegistered is returned when an effect needs a platform backend
// (a persistent SharedCache, a websocket transport) that the binary never
// registered. The language core defines the seam; cmd/ailang registers the
// implementation at startup (platform_init.go). Callers wrap it with the
// backend name — test with errors.Is.
var ErrBackendNotRegistered = errors.New("backend not registered")

// StreamFrameKind classifies one inbound or outbound websocket frame.
type StreamFrameKind int

const (
	StreamFrameText StreamFrameKind = iota
	StreamFrameBinary
	StreamFramePing
)

// StreamFrame is one websocket message as the core sees it — the transport
// hides the wire library's message types behind this.
type StreamFrame struct {
	Kind StreamFrameKind
	Data []byte
}

// StreamCloseError is returned by StreamTransport.Recv when the peer closed
// the connection cleanly (normal closure or going-away). Any other Recv error
// is surfaced to the program as a ProtocolError.
type StreamCloseError struct {
	Code   int
	Reason string
}

func (e *StreamCloseError) Error() string {
	return fmt.Sprintf("websocket closed (%d): %s", e.Code, e.Reason)
}

// StreamDialConfig is everything a transport needs to open a connection; the
// values come from the StreamContext security policy and the connect call.
type StreamDialConfig struct {
	URL              string
	Headers          map[string][]string
	Subprotocols     []string
	HandshakeTimeout time.Duration
	MaxFrameSize     int64 // inbound read limit per frame
}

// StreamTransport is a bidirectional message transport opened by a registered
// backend. Recv blocks for the next frame; Send writes one; Close sends the
// close handshake and releases the socket. Send is called under the
// connection mutex; Recv runs on the connection's read goroutine.
type StreamTransport interface {
	Subprotocol() string
	Recv() (StreamFrame, error)
	Send(frame StreamFrame) error
	Close() error
}

// StreamTransportOpener dials one connection.
type StreamTransportOpener func(cfg StreamDialConfig) (StreamTransport, error)

var (
	streamTransportsMu sync.RWMutex
	streamTransports   = map[string]StreamTransportOpener{}
)

// RegisterStreamTransport installs the opener for a transport name ("ws"
// covers both ws:// and wss://). Registering nil removes it. The platform
// registers at binary start; tests register per test.
func RegisterStreamTransport(name string, open StreamTransportOpener) {
	streamTransportsMu.Lock()
	defer streamTransportsMu.Unlock()
	if open == nil {
		delete(streamTransports, name)
		return
	}
	streamTransports[name] = open
}

// streamTransportName maps a URL scheme to the registered transport name.
// Returns "" for schemes Stream.connect does not serve (http/https are SSE).
func streamTransportName(scheme string) string {
	switch scheme {
	case "ws", "wss":
		return "ws"
	default:
		return ""
	}
}

// openStreamTransport resolves the transport for rawURL and dials it.
// A missing registration is a harness error (wraps ErrBackendNotRegistered);
// a dial failure is returned as-is for the caller to surface as a
// ConnectionFailed result.
func openStreamTransport(rawURL string, cfg StreamDialConfig) (StreamTransport, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}
	name := streamTransportName(u.Scheme)
	if name == "" {
		return nil, fmt.Errorf("Stream.connect expects ws:// or wss://, got %q (http(s) is served by Stream.sseConnect)", u.Scheme)
	}
	streamTransportsMu.RLock()
	open, ok := streamTransports[name]
	streamTransportsMu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("stream transport %q for %s://: %w", name, u.Scheme, ErrBackendNotRegistered)
	}
	return open(cfg)
}
