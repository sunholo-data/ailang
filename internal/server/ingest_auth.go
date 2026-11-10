package server

// Authentication for the dashboard's machine-ingest WRITE routes.
//
// read_auth.go default-denies /api/ and /ws but allow-lists the ingest routes
// (and OTLP /v1/* sits outside its namespace altogether), because their
// senders are machines that never held the dashboard credential. This gate
// gives each of those routes its own shared secret instead:
//
//	class   routes                                                  secret
//	otlp    POST /v1/traces, /v1/logs, /v1/metrics                  AILANG_OTLP_INGEST_TOKEN
//	hub     POST /api/hooks/claude, /api/observatory/hooks,         AILANG_HUB_TOKEN
//	        /api/exec/sessions, /api/exec/events,
//	        /api/coordinator/events, /api/telemetry
//
// A sender presents it as `X-AILANG-Ingest-Token: <token>` (what an OTLP
// exporter sends from OTEL_EXPORTER_OTLP_HEADERS=x-ailang-ingest-token=<token>,
// and the only form OpenRouter Broadcast's static custom headers need) or as
// `Authorization: Bearer <token>`; compared in constant time.
//
// ROLLOUT MODE, AILANG_INGEST_AUTH. Enforcing the moment this deploys would
// drop every write from a sender not yet configured, so the mode is explicit:
//
//	warn     (default) accept everything; count and log each write that lacks
//	         a valid token, naming the class, reason and sender — deploy, add
//	         the token to every sender, watch the log go quiet, then enforce
//	enforce  refuse a write without a valid token (401); a class whose token
//	         is unset refuses everything (503) — fail closed, never open
//	off      accept everything; logged at startup (the rollback lever)
//
// Any unrecognised mode enforces (config.IngestAuthMode).
//
// Loopback exemption, as read_auth.go: a server bound to loopback (the rig's
// `ailang serve`, which receives every local Claude Code hook) skips the gate.
//
// TestIngestAuth_EveryPublicWriteHasAClass fails when an allow-listed write
// route has no class here.

import (
	"crypto/subtle"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/sunholo-data/ailang/internal/config"
)

// ingestAuthRealm is sent in WWW-Authenticate on this gate's 401s, distinct
// from the dashboard's and the approval intake's realms.
const ingestAuthRealm = `Bearer realm="ailang-ingest"`

// ingestClass names which shared secret an ingest route takes.
type ingestClass string

const (
	ingestOTLP ingestClass = "otlp"
	ingestHub  ingestClass = "hub"
)

// ingestRoutes maps each ingest route's registered pattern to its class. Only
// POST is ingest: every other method on these patterns is either refused by
// the pattern itself (/v1/*) or by read_auth.go's gate.
var ingestRoutes = map[string]ingestClass{
	"POST /v1/traces":         ingestOTLP,
	"POST /v1/logs":           ingestOTLP,
	"POST /v1/metrics":        ingestOTLP,
	"/api/hooks/claude":       ingestHub,
	"/api/observatory/hooks":  ingestHub,
	"/api/exec/sessions":      ingestHub,
	"/api/exec/events":        ingestHub,
	"/api/coordinator/events": ingestHub,
	"/api/telemetry":          ingestHub,
}

// ingestTokenEnv names the variable that holds each class's secret, for logs
// and error bodies.
var ingestTokenEnv = map[ingestClass]string{
	ingestOTLP: config.EnvOTLPIngestToken,
	ingestHub:  config.EnvHubToken,
}

// WithOTLPIngestToken sets the OTLP ingest secret (AILANG_OTLP_INGEST_TOKEN).
func WithOTLPIngestToken(token string) ServerOption {
	return func(s *Server) { s.otlpIngestToken = token }
}

// WithIngestAuthMode sets the ingest gate's mode: config.IngestAuthOff, Warn
// or Enforce. Unset means warn, the registered default.
func WithIngestAuthMode(mode string) ServerOption {
	return func(s *Server) { s.ingestAuthMode = mode }
}

func (s *Server) ingestToken(c ingestClass) string {
	if c == ingestOTLP {
		return s.otlpIngestToken
	}
	return s.hookToken
}

func (s *Server) ingestMode() string {
	if s.ingestAuthMode == "" {
		return config.IngestAuthWarn
	}
	return s.ingestAuthMode
}

// ingestAuthMiddleware gates the ingest routes of mux, then hands every
// request to next (the read-auth gate). Like readAuthMiddleware it keys on the
// pattern the request will be routed to, not on a path guess.
func (s *Server) ingestAuthMiddleware(mux *http.ServeMux, next http.Handler) http.Handler {
	if isLoopbackBind(s.httpAddr) {
		log.Printf("Ingest auth: EXEMPT (bound to loopback %s)", s.httpAddr)
		return next
	}
	mode := s.ingestMode()
	s.logIngestPosture(mode)
	if mode == config.IngestAuthOff {
		return next
	}
	warnings := newIngestWarnLog(time.Minute)
	s.ingestWarnings = warnings
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			next.ServeHTTP(w, r)
			return
		}
		_, pattern := mux.Handler(r)
		class, ok := ingestRoutes[pattern]
		if !ok {
			next.ServeHTTP(w, r)
			return
		}
		reason := s.checkIngestToken(class, r)
		if reason == "" {
			next.ServeHTTP(w, r)
			return
		}
		if mode == config.IngestAuthWarn {
			warnings.note(class, reason, r)
			next.ServeHTTP(w, r)
			return
		}
		refuseIngest(w, r, class, reason)
	})
}

// checkIngestToken returns "" when r carries class's secret, else why not:
// token_unset (the server has none), missing or invalid.
func (s *Server) checkIngestToken(class ingestClass, r *http.Request) string {
	expected := s.ingestToken(class)
	if expected == "" {
		return "token_unset"
	}
	presented := presentedIngestToken(r)
	if presented == "" {
		return "missing"
	}
	if subtle.ConstantTimeCompare([]byte(presented), []byte(expected)) != 1 {
		return "invalid"
	}
	return ""
}

// presentedIngestToken is the sender's credential: X-AILANG-Ingest-Token,
// else an Authorization bearer token.
func presentedIngestToken(r *http.Request) string {
	if v := strings.TrimSpace(r.Header.Get(config.IngestTokenHeader)); v != "" {
		return v
	}
	return strings.TrimSpace(extractBearerToken(r))
}

func refuseIngest(w http.ResponseWriter, r *http.Request, class ingestClass, reason string) {
	env := ingestTokenEnv[class]
	log.Printf("ingest auth: refused %s %s from %s: class=%s reason=%s", logField(r.Method), logField(r.URL.Path), logField(ingestSender(r)), class, reason)
	if reason == "token_unset" {
		writeErrorResponse(w, http.StatusServiceUnavailable, "ingest_auth_not_configured",
			"Ingest authentication is enforced but "+env+" is not set on this server")
		return
	}
	w.Header().Set("WWW-Authenticate", ingestAuthRealm)
	writeErrorResponse(w, http.StatusUnauthorized, reason+"_token",
		"Ingest requires the shared secret in "+config.IngestTokenHeader+" or Authorization: Bearer ("+env+")")
}

// logIngestPosture states, once at startup, what the gate will do.
func (s *Server) logIngestPosture(mode string) {
	switch mode {
	case config.IngestAuthOff:
		log.Printf("SECURITY: ingest auth is OFF (%s=off) on non-loopback bind %s — OTLP /v1/* and the hook/exec/telemetry routes accept anonymous writes", config.EnvIngestAuth, s.httpAddr)
		return
	case config.IngestAuthWarn:
		log.Printf("Ingest auth: WARN on %s — writes without a valid token are ACCEPTED and logged as \"ingest auth WARN\"; set %s=enforce once those lines stop", s.httpAddr, config.EnvIngestAuth)
	default:
		log.Printf("Ingest auth: ENFORCED on %s", s.httpAddr)
	}
	for _, c := range []ingestClass{ingestOTLP, ingestHub} {
		if s.ingestToken(c) != "" {
			continue
		}
		if mode == config.IngestAuthEnforce {
			log.Printf("SECURITY: %s is not set on non-loopback bind %s — %s ingest will return 503 until it is", ingestTokenEnv[c], s.httpAddr, c)
		} else {
			log.Printf("Ingest auth: %s is not set — every %s ingest write will be logged as unauthenticated", ingestTokenEnv[c], c)
		}
	}
}

// ingestSender identifies a caller for the log: the client address Cloud Run's
// front end records in X-Forwarded-For, else the peer address.
func ingestSender(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		first, _, _ := strings.Cut(xff, ",")
		return strings.TrimSpace(first)
	}
	return r.RemoteAddr
}

// ingestWarnLog counts the writes warn mode let through without a valid
// token, and logs them without flooding: the first write from each distinct
// (class, reason, user agent) immediately — so every unconfigured sender is
// named at least once — and otherwise one summary per class per interval.
type ingestWarnLog struct {
	interval time.Duration
	now      func() time.Time

	mu      sync.Mutex
	total   map[ingestClass]int64
	pending map[ingestClass]int64
	last    map[ingestClass]time.Time
	seen    map[string]bool
}

// maxIngestSendersLogged bounds the distinct-sender set, so a caller rotating
// its user agent cannot grow it without limit; past it, summaries only.
const maxIngestSendersLogged = 256

func newIngestWarnLog(interval time.Duration) *ingestWarnLog {
	return &ingestWarnLog{
		interval: interval,
		now:      time.Now,
		total:    map[ingestClass]int64{},
		pending:  map[ingestClass]int64{},
		last:     map[ingestClass]time.Time{},
		seen:     map[string]bool{},
	}
}

func (l *ingestWarnLog) note(class ingestClass, reason string, r *http.Request) {
	ua := r.UserAgent()
	if len(ua) > 80 {
		ua = ua[:80]
	}
	key := string(class) + "|" + reason + "|" + ua

	l.mu.Lock()
	l.total[class]++
	l.pending[class]++
	total, pending := l.total[class], l.pending[class]
	now := l.now()
	newSender := !l.seen[key] && len(l.seen) < maxIngestSendersLogged
	if newSender {
		l.seen[key] = true
	}
	due := now.Sub(l.last[class]) >= l.interval
	emit := newSender || due
	if emit {
		l.last[class] = now
		l.pending[class] = 0
	}
	l.mu.Unlock()

	if emit {
		log.Printf("ingest auth WARN: accepted %d unauthenticated %s write(s) since the last report (%d total); latest %s %s from %s user_agent=%q reason=%s — refused once %s=enforce",
			pending, class, total, logField(r.Method), logField(r.URL.Path), logField(ingestSender(r)), logField(ua), reason, config.EnvIngestAuth)
	}
}

// count reports how many unauthenticated writes of class warn mode accepted.
func (l *ingestWarnLog) count(class ingestClass) int64 {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.total[class]
}
