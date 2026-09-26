package apiserver

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/platform/streamws"
)

// WebSocket routes (M-SERVEAPI-WS-BRIDGE M1).
//
// @route("WS", "/path") makes serve-api accept the upgrade and call the
// handler ONCE for the life of the connection, with the browser leg as an
// ordinary std/stream StreamConn. Every check runs BEFORE the upgrade:
//
//	plain GET (no Upgrade)          426
//	Origin missing, null or foreign 403
//	API key configured and absent   401
//	over --ws-max-sessions          503
//
// Each connection gets a child StreamContext (fresh connection map, IDs and
// limits), installed on the call's own cloned EffContext, so sessions cannot
// see each other's connections and MaxDuration is per connection (G6).

// WSConfig holds the WebSocket session limits.
type WSConfig struct {
	MaxSessions int  // concurrent WS sessions server-wide (0 = DefaultWSMaxSessions)
	QueueFrames int  // per-connection inbound queue, frames (0 = DefaultWSQueueFrames)
	DecisionLog bool // log one payload-free line per bridged frame
}

const (
	// DefaultWSMaxSessions is the server-wide concurrent WS session cap.
	DefaultWSMaxSessions = 4
	// DefaultWSQueueFrames bounds each connection's inbound queue. With the
	// default 64 KiB frame limit this is at most 4 MiB queued per direction.
	DefaultWSQueueFrames = 64

	// wsKeyProtocolPrefix carries an API key from a browser, which cannot set
	// headers on new WebSocket(): Sec-WebSocket-Protocol: ailang.v1, ailang.key.<key>.
	wsKeyProtocolPrefix = "ailang.key."
	// wsProtocol is the subprotocol selected when the key rode in a
	// subprotocol entry — the key itself is never echoed.
	wsProtocol = "ailang.v1"
)

// wsState is the server's WS session bookkeeping.
type wsState struct {
	cfg      WSConfig
	sem      chan struct{}
	nextID   atomic.Int64
	mu       sync.Mutex
	sessions map[*effects.StreamContext]struct{}
}

func newWSState(cfg WSConfig) *wsState {
	if cfg.MaxSessions <= 0 {
		cfg.MaxSessions = DefaultWSMaxSessions
	}
	if cfg.QueueFrames <= 0 {
		cfg.QueueFrames = DefaultWSQueueFrames
	}
	return &wsState{
		cfg:      cfg,
		sem:      make(chan struct{}, cfg.MaxSessions),
		sessions: make(map[*effects.StreamContext]struct{}),
	}
}

// wsRoutes returns the loaded WS routes.
func (s *Server) wsRoutes() []RouteEntry {
	var out []RouteEntry
	for _, r := range s.getCustomRoutes() {
		if r.IsWS {
			out = append(out, r)
		}
	}
	return out
}

// ValidateWSRoutes fails serve-api startup when a WS route cannot be served
// safely or correctly. Checked: handler shape, effect row and --caps, no
// @raw/@nowrap, and the bind rule — a WS route on a non-loopback address
// needs an Origin allowlist (--cors-origin); --cors (every origin) is not one.
func (s *Server) ValidateWSRoutes() error {
	routes := s.wsRoutes()
	if len(routes) == 0 {
		return nil
	}
	var problems []string
	for _, r := range routes {
		problems = append(problems, s.wsRouteProblems(r)...)
	}
	if !isLoopbackHost(s.bindHost()) && !s.origins.HasAllowlist() {
		problems = append(problems, fmt.Sprintf(
			"serve-api binds %s (not loopback) and has no --cors-origin allowlist: a WebSocket route there would accept any page that can reach the address. Pass --cors-origin https://your.page (repeatable), or --bind 127.0.0.1",
			s.bindHost()))
	}
	if len(problems) > 0 {
		return fmt.Errorf("serve-api refuses to register WebSocket routes:\n  - %s", strings.Join(problems, "\n  - "))
	}
	return nil
}

func (s *Server) wsRouteProblems(r RouteEntry) []string {
	name := r.Module + "." + r.Function
	var p []string
	if r.IsRaw || r.IsNowrap {
		p = append(p, fmt.Sprintf("%s: @raw/@nowrap do not apply to a WS route", name))
	}
	if n := len(r.ParamTypes); n < 1 || n > 2 || r.ParamTypes[0] != "StreamConn" {
		p = append(p, fmt.Sprintf("%s: a WS handler takes (client: StreamConn) or (client: StreamConn, req: {path: string, query: string, origin: string})", name))
	}
	hasStream := false
	for _, e := range r.Effects {
		if e == "Stream" {
			hasStream = true
		}
		if s.effCtx == nil || !s.effCtx.HasCap(e) {
			p = append(p, fmt.Sprintf("%s: effect %s is not granted (add it to --caps)", name, e))
		}
	}
	if !hasStream {
		p = append(p, fmt.Sprintf("%s: a WS handler's effect row must include Stream", name))
	}
	return p
}

// isLoopbackHost reports whether a bind host only listens on loopback.
func isLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(strings.Trim(host, "[]"))
	return ip != nil && ip.IsLoopback()
}

// wsHandler serves one WS route.
func (s *Server) wsHandler(route RouteEntry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !streamws.IsUpgrade(r) {
			w.Header().Set("Upgrade", "websocket")
			http.Error(w, "this route is a WebSocket endpoint: connect with an Upgrade: websocket request", http.StatusUpgradeRequired)
			return
		}
		if err := s.checkWSOrigin(r); err != nil {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		subprotocol, ok := s.checkWSKey(r)
		if !ok {
			http.Error(w, "unauthorized: invalid or missing API key (browsers: Sec-WebSocket-Protocol ailang.v1, ailang.key.<key>)", http.StatusUnauthorized)
			return
		}
		select {
		case s.ws.sem <- struct{}{}:
			defer func() { <-s.ws.sem }()
		default:
			http.Error(w, fmt.Sprintf("too many WebSocket sessions (serve-api --ws-max-sessions %d)", s.ws.cfg.MaxSessions), http.StatusServiceUnavailable)
			return
		}
		s.serveWSSession(w, r, route, subprotocol)
	}
}

// serveWSSession upgrades, adopts the connection into a fresh child
// StreamContext and runs the handler for the connection's lifetime.
func (s *Server) serveWSSession(w http.ResponseWriter, r *http.Request, route RouteEntry, subprotocol string) {
	if s.effCtx == nil || s.effCtx.Stream == nil {
		http.Error(w, "WebSocket routes need --caps Stream", http.StatusServiceUnavailable)
		return
	}
	callID := fmt.Sprintf("ws-%d", s.ws.nextID.Add(1))
	child := s.effCtx.Stream.Child()
	child.EventBufferSize = s.ws.cfg.QueueFrames
	if s.ws.cfg.DecisionLog {
		child.BridgeLog = decisionLogger(callID)
	}

	t, err := streamws.Accept(w, r, streamws.AcceptOptions{Subprotocol: subprotocol, MaxFrameSize: child.MaxFrameSize})
	if err != nil {
		log.Printf("[ws] %s %s: upgrade failed: %v", callID, route.Path, err)
		return // gorilla wrote the HTTP error
	}
	id, err := effects.AdoptStreamConnection(child, t)
	if err != nil {
		log.Printf("[ws] %s %s: %v", callID, route.Path, err)
		return
	}
	s.trackWSSession(child, true)
	defer s.trackWSSession(child, false)

	args := []interface{}{&eval.TaggedValue{CtorName: "StreamConn", Fields: []eval.Value{&eval.IntValue{Value: id}}}}
	if len(route.ParamTypes) == 2 {
		args = append(args, &eval.RecordValue{Fields: map[string]eval.Value{
			"path":   &eval.StringValue{Value: r.URL.Path},
			"query":  &eval.StringValue{Value: r.URL.RawQuery},
			"origin": &eval.StringValue{Value: r.Header.Get("Origin")},
		}})
	}
	_, callErr := s.engine.CallPrepared(func(ec interface{}) {
		if eff, ok := ec.(*effects.EffContext); ok {
			eff.Stream = child
		}
	}, route.Module, route.Function, args...)

	// The handler call IS the connection: close the client (1011 on failure)
	// and everything else the session opened.
	if conn, ok := child.GetConnection(id); ok {
		if callErr != nil {
			conn.CloseWithCode(1011, "handler failed")
		} else {
			conn.Close()
		}
	}
	child.CloseAll()
	if callErr != nil {
		log.Printf("[ws] %s %s -> %s.%s failed: %v", callID, route.Path, route.Module, route.Function, callErr)
	}
}

// checkWSOrigin enforces the Origin rule before any upgrade. CORS does not
// apply to WebSockets, so this is the only thing stopping another page in
// the user's browser from opening the socket. Allowed: same-origin (Origin's
// host equals the request Host) and exact --cors-origin entries. A missing
// Origin or "null" (sandboxed/file pages) is refused.
// The rule itself is originpolicy.CheckWebSocket, shared with `ailang server`;
// serve-api passes allowMissing=false (its WS routes are browser-facing).
func (s *Server) checkWSOrigin(r *http.Request) error {
	return s.origins.CheckWebSocket(r, false)
}

// checkWSKey applies --api-key-header/--api-key-env to a WS upgrade. With no
// key configured it accepts and selects no subprotocol. A non-browser client
// may send the configured header (or Authorization: Bearer); a browser sends
// Sec-WebSocket-Protocol entries "ailang.v1" and "ailang.key.<key>", and the
// server selects ailang.v1 so the key is never echoed. Query-string keys are
// not accepted: they leak into logs and history.
func (s *Server) checkWSKey(r *http.Request) (string, bool) {
	if s.apiKeyHeader == "" || s.apiKeyEnv == "" {
		return "", true
	}
	expected := config.Raw(s.apiKeyEnv)
	if expected == "" {
		return "", false
	}
	match := func(k string) bool {
		return k != "" && subtle.ConstantTimeCompare([]byte(k), []byte(expected)) == 1
	}
	if match(r.Header.Get(s.apiKeyHeader)) || match(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")) {
		return "", true
	}
	offeredV1 := false
	keyOK := false
	for _, p := range strings.Split(r.Header.Get("Sec-WebSocket-Protocol"), ",") {
		p = strings.TrimSpace(p)
		if p == wsProtocol {
			offeredV1 = true
		}
		if strings.HasPrefix(p, wsKeyProtocolPrefix) && match(strings.TrimPrefix(p, wsKeyProtocolPrefix)) {
			keyOK = true
		}
	}
	if keyOK && offeredV1 {
		return wsProtocol, true
	}
	return "", false
}

func (s *Server) trackWSSession(sc *effects.StreamContext, add bool) {
	s.ws.mu.Lock()
	defer s.ws.mu.Unlock()
	if add {
		s.ws.sessions[sc] = struct{}{}
	} else {
		delete(s.ws.sessions, sc)
	}
}

// closeWSSessions sends 1001 (going away) on every live session. Registered
// with http.Server.RegisterOnShutdown: Shutdown does not close hijacked
// connections by itself.
func (s *Server) closeWSSessions() {
	s.ws.mu.Lock()
	live := make([]*effects.StreamContext, 0, len(s.ws.sessions))
	for sc := range s.ws.sessions {
		live = append(live, sc)
	}
	s.ws.mu.Unlock()
	for _, sc := range live {
		sc.CloseAllWithCode(1001, "server shutting down")
	}
}

// decisionLogger writes one JSON line per bridged frame to the serve-api log:
// call id, sequence, direction, kind, size, verdict and step cost. Never a
// payload, so the log is safe to keep; replaying the (dir, kind) sequence
// through the step offline reproduces the verdicts.
func decisionLogger(callID string) func(effects.BridgeDecision) {
	return func(d effects.BridgeDecision) {
		line, err := json.Marshal(struct {
			CallID string `json:"call_id"`
			effects.BridgeDecision
		}{callID, d})
		if err != nil {
			return
		}
		log.Printf("[ws-bridge] %s", line)
	}
}
