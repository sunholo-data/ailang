package effects

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/url"
	"sync"
	"time"
)

// StreamContext provides configuration for Stream effect security
//
// M-STREAM-BIDI: The stream context holds security settings for persistent
// WebSocket connections, following the same patterns as NetContext.
//
// Security features:
//   - Protocol validation (wss:// enforced by default, ws:// requires flag)
//   - Domain allowlist (optional)
//   - Private IP blocking (RFC1918 + link-local, default: on)
//   - Connection count limits (default: 4 concurrent)
//   - Message size limits (default: 1MB per message, 64KB per frame)
//   - Idle timeout (default: 60s)
//   - Hard duration ceiling (default: 5min)
type StreamContext struct {
	// Connection limits
	MaxConnections int   // Default: 4 (prevent resource exhaustion)
	MaxMessageSize int64 // Default: 1MB per message (reassembled from frames)
	MaxFrameSize   int64 // Default: 64KB per frame (gorilla ReadLimit)

	// Timeouts
	ConnectTimeout time.Duration // Default: 30s
	IdleTimeout    time.Duration // Default: 60s (close if no messages; reset on any activity)
	MaxDuration    time.Duration // Default: 5min (hard ceiling)

	// Security
	AllowHTTP       bool     // Default: false (wss:// only)
	AllowLocalhost  bool     // Default: false
	BlockPrivateIPs bool     // Default: true (RFC1918 + link-local)
	AllowedDomains  []string // Domain allowlist (empty = all allowed)

	// Event buffer
	EventBufferSize int // Default: 1000 (bounded; backpressure when full)

	// Credentials is the host-side credential binding (M-SERVEAPI-WS-BRIDGE
	// D3). Nil = no binding. It is consulted ONLY for wss:// dials, after the
	// destination authorizer has passed, and returns the headers to add for
	// that exact URL (bound=false when no binding matches). The returned
	// values never become AILANG values: they are merged into the dial
	// headers in Go and nowhere else. The binder must not put the credential
	// into err.
	Credentials func(u *url.URL) (hdr map[string][]string, bound bool, err error)

	// BridgeLog, when set, receives one payload-free decision per bridged
	// frame (M-SERVEAPI-WS-BRIDGE M4). serve-api sets it per session.
	BridgeLog func(BridgeDecision)

	// Runtime state
	mu          sync.Mutex
	connections map[int]*StreamConnection
	nextID      int

	// Event source management (M-ASYNC-IO)
	sources      map[int]EventSource
	nextSourceID int

	// Test hooks, nil in production — the same seam NetContext has, so the
	// shared destination authorizer (net_authorize.go) is falsifiable for
	// Stream transports with an injected resolver/dialer.
	lookupIP        func(hostname string) ([]net.IP, error)
	dialContext     func(ctx context.Context, network, addr string) (net.Conn, error)
	tlsClientConfig *tls.Config
}

// NewStreamContext creates a new stream context with secure defaults
func NewStreamContext() *StreamContext {
	return &StreamContext{
		MaxConnections:  4,
		MaxMessageSize:  1 * 1024 * 1024, // 1MB
		MaxFrameSize:    64 * 1024,       // 64KB
		ConnectTimeout:  30 * time.Second,
		IdleTimeout:     60 * time.Second,
		MaxDuration:     5 * time.Minute,
		AllowHTTP:       false,
		AllowLocalhost:  false,
		BlockPrivateIPs: true,
		AllowedDomains:  []string{},
		EventBufferSize: 1000,
		connections:     make(map[int]*StreamConnection),
		nextID:          1,
	}
}

// Child returns a context with this context's POLICY (limits, timeouts,
// destination rules, credential binding, test seams) and FRESH runtime state:
// its own connection map, IDs and sources. serve-api gives every WebSocket
// session one, so MaxConnections, MaxDuration and connection IDs are per
// connection rather than process-wide (M-SERVEAPI-WS-BRIDGE G6), and one
// session cannot reach another session's connections by ID.
func (sc *StreamContext) Child() *StreamContext {
	c := &StreamContext{
		MaxConnections:  sc.MaxConnections,
		MaxMessageSize:  sc.MaxMessageSize,
		MaxFrameSize:    sc.MaxFrameSize,
		ConnectTimeout:  sc.ConnectTimeout,
		IdleTimeout:     sc.IdleTimeout,
		MaxDuration:     sc.MaxDuration,
		AllowHTTP:       sc.AllowHTTP,
		AllowLocalhost:  sc.AllowLocalhost,
		BlockPrivateIPs: sc.BlockPrivateIPs,
		AllowedDomains:  append([]string(nil), sc.AllowedDomains...),
		EventBufferSize: sc.EventBufferSize,
		Credentials:     sc.Credentials,
		BridgeLog:       sc.BridgeLog,
		connections:     make(map[int]*StreamConnection),
		nextID:          1,
		lookupIP:        sc.lookupIP,
		dialContext:     sc.dialContext,
		tlsClientConfig: sc.tlsClientConfig,
	}
	return c
}

// SetTLSClientConfig sets the TLS client configuration Stream transports use
// (the RootCAs a wss:// or https:// dial trusts). Nil restores the system
// roots. Used by tests that stand up an httptest TLS upstream and by hosts
// that pin a private CA; it never relaxes verification by itself.
func (sc *StreamContext) SetTLSClientConfig(cfg *tls.Config) {
	sc.tlsClientConfig = cfg
}

// ValidateURL checks a URL against the stream security policy — the shared
// destination authorizer (net_authorize.go), which the SSE/NDJSON transports
// re-run on every redirect hop and the WebSocket dialer pins on.
// Returns nil if the URL passes all checks.
func (sc *StreamContext) ValidateURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("E_STREAM_INVALID_URL: %w", err)
	}
	return sc.policy().authorizeURL(u)
}

// policy is the destination policy for this context.
func (sc *StreamContext) policy() destinationPolicy {
	return streamPolicy(&EffContext{Stream: sc})
}

// AcquireConnection registers a new connection and returns its ID.
// Returns an error if the connection limit is reached.
func (sc *StreamContext) AcquireConnection(conn *StreamConnection) (int, error) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	if len(sc.connections) >= sc.MaxConnections {
		return 0, fmt.Errorf("E_STREAM_CONNECTION_LIMIT: maximum %d concurrent connections reached", sc.MaxConnections)
	}

	id := sc.nextID
	sc.nextID++
	sc.connections[id] = conn
	return id, nil
}

// ReleaseConnection removes a connection from tracking.
func (sc *StreamContext) ReleaseConnection(id int) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	delete(sc.connections, id)
}

// GetConnection retrieves a connection by ID.
func (sc *StreamContext) GetConnection(id int) (*StreamConnection, bool) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	conn, ok := sc.connections[id]
	return conn, ok
}

// ConnectionCount returns the number of active connections.
func (sc *StreamContext) ConnectionCount() int {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	return len(sc.connections)
}

// CloseAll closes all active connections. Used for graceful shutdown.
func (sc *StreamContext) CloseAll() {
	sc.CloseAllWithCode(0, "")
}

// CloseAllWithCode closes all active connections, sending code and reason on
// WebSocket ones (0 = normal closure). serve-api sends 1001 on shutdown.
func (sc *StreamContext) CloseAllWithCode(code int, reason string) {
	sc.mu.Lock()
	conns := make([]*StreamConnection, 0, len(sc.connections))
	for _, c := range sc.connections {
		conns = append(conns, c)
	}
	sc.mu.Unlock()

	for _, c := range conns {
		c.CloseWithCode(code, reason)
	}
}
