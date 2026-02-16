package effects

import (
	"fmt"
	"net"
	"net/url"
	"strings"
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

// ValidateURL checks a URL against the stream security policy.
// Returns nil if the URL passes all checks.
func (sc *StreamContext) ValidateURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("E_STREAM_INVALID_URL: %w", err)
	}

	// Protocol check
	switch u.Scheme {
	case "wss", "https":
		// Always allowed
	case "ws", "http":
		if !sc.AllowHTTP {
			return fmt.Errorf("E_STREAM_PROTOCOL_ERROR: insecure protocol %q not allowed (use wss:// or set --stream-allow-http)", u.Scheme)
		}
	default:
		return fmt.Errorf("E_STREAM_PROTOCOL_ERROR: unsupported protocol %q (expected wss:// or ws://)", u.Scheme)
	}

	hostname := u.Hostname()
	if hostname == "" {
		return fmt.Errorf("E_STREAM_INVALID_URL: missing hostname")
	}

	// Domain allowlist check
	if len(sc.AllowedDomains) > 0 {
		if !isStreamAllowedDomain(hostname, sc.AllowedDomains) {
			return fmt.Errorf("E_STREAM_DISALLOWED_HOST: domain not in allowlist: %s", hostname)
		}
	}

	// Localhost check
	if !sc.AllowLocalhost && isLocalhost(hostname) {
		return fmt.Errorf("E_STREAM_DISALLOWED_HOST: localhost connections not allowed")
	}

	// Private IP check
	if sc.BlockPrivateIPs {
		ip := net.ParseIP(hostname)
		if ip != nil && isPrivateIP(ip) {
			return fmt.Errorf("E_STREAM_DISALLOWED_HOST: private IP addresses not allowed: %s", hostname)
		}
	}

	return nil
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

// isStreamAllowedDomain checks if a hostname matches the allowlist.
func isStreamAllowedDomain(hostname string, allowed []string) bool {
	hostname = strings.ToLower(hostname)
	for _, d := range allowed {
		d = strings.ToLower(d)
		if d == hostname {
			return true
		}
		// Wildcard prefix match: *.example.com matches sub.example.com
		if strings.HasPrefix(d, "*.") {
			suffix := d[1:] // ".example.com"
			if strings.HasSuffix(hostname, suffix) {
				return true
			}
		}
	}
	return false
}

// isLocalhost checks if a hostname resolves to a loopback address.
func isLocalhost(hostname string) bool {
	hostname = strings.ToLower(hostname)
	if hostname == "localhost" || hostname == "127.0.0.1" || hostname == "::1" {
		return true
	}
	if strings.HasPrefix(hostname, "127.") {
		return true
	}
	return false
}

// isPrivateIP checks if an IP is in RFC1918 or link-local ranges.
func isPrivateIP(ip net.IP) bool {
	privateRanges := []struct {
		network string
	}{
		{"10.0.0.0/8"},
		{"172.16.0.0/12"},
		{"192.168.0.0/16"},
		{"169.254.0.0/16"}, // Link-local IPv4
		{"fc00::/7"},       // IPv6 unique local
		{"fe80::/10"},      // IPv6 link-local
	}
	for _, r := range privateRanges {
		_, cidr, err := net.ParseCIDR(r.network)
		if err != nil {
			continue
		}
		if cidr.Contains(ip) {
			return true
		}
	}
	return false
}
