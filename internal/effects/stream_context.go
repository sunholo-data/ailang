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
	sc.mu.Lock()
	conns := make([]*StreamConnection, 0, len(sc.connections))
	for _, c := range sc.connections {
		conns = append(conns, c)
	}
	sc.mu.Unlock()

	for _, c := range conns {
		c.Close()
	}
}
