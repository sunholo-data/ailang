package riggate

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/riglock"
)

// LeaseHeader is the explicit lease header. It wins over Authorization, which
// carries the token only for clients whose API key is templated from the lease.
const LeaseHeader = "X-Rig-Lease"

// Gate is the admission proxy.
type Gate struct {
	upstream *url.URL
	proxy    *httputil.ReverseProxy
	decider  Decider
	lease    func() riglock.Lease
	ledger   *Ledger
	now      func() time.Time
}

// NewGate builds a gate to upstream. lease is normally riglock.CurrentLease;
// ledger may be nil (no ledger).
func NewGate(upstream *url.URL, decider Decider, lease func() riglock.Lease, ledger *Ledger) *Gate {
	g := &Gate{upstream: upstream, decider: decider, lease: lease, ledger: ledger, now: time.Now}
	rp := httputil.NewSingleHostReverseProxy(upstream)
	// Flush every write: chat completions stream as SSE/NDJSON and a buffering
	// proxy would turn a live stream into one late blob (and break tool-calling
	// clients that read incrementally). Go 1.26's ReverseProxy already flushes
	// immediately for text/event-stream and for any response of unknown length,
	// which covers every streamed ollama reply (mutation-checked 2026-09-27: 0
	// here still streams); this states the requirement rather than relying on it.
	rp.FlushInterval = -1
	rp.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		// 502, never 503: opencode retries a 503 indefinitely (design doc V11).
		// A client that went away is not an upstream failure; nothing to write.
		if errors.Is(r.Context().Err(), context.Canceled) {
			return
		}
		writeJSONError(w, http.StatusBadGateway, "rig_gate_upstream", fmt.Sprintf("ollama unreachable behind the rig gate: %v", err))
	}
	g.proxy = rp
	return g
}

// ServeHTTP gathers facts, asks the policy, then proxies or refuses.
func (g *Gate) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := g.now()
	l := g.lease()
	tok := requestToken(r)
	f := Facts{
		Path:          r.URL.Path,
		LeaseHeld:     l.Held,
		LeaseHasToken: l.Token != "",
		TokenPresent:  tok != "",
		TokenMatches:  tok != "" && l.Token != "" && subtle.ConstantTimeCompare([]byte(tok), []byte(l.Token)) == 1,
		Holder:        l.Holder,
	}
	v, err := g.decider.Decide(f)
	if err != nil {
		// No Go copy of the rule to fall back on (CLAUDE.md: fail loudly).
		writeJSONError(w, http.StatusInternalServerError, "rig_gate_policy", err.Error())
		g.record(r, f, Verdict{Decision: "policy-error", Reason: err.Error()}, http.StatusInternalServerError, start)
		return
	}
	if !v.Admit {
		writeJSONError(w, v.Status, "rig_lease", v.Reason)
		g.record(r, f, v, v.Status, start)
		return
	}
	sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
	g.proxy.ServeHTTP(sw, r)
	g.record(r, f, v, sw.status, start)
}

// requestToken extracts the lease token: X-Rig-Lease first, then a Bearer key.
// The placeholder values clients carry when they have no lease ("none", and pi's
// literal ollama key "ollama") count as no token.
func requestToken(r *http.Request) string {
	t := strings.TrimSpace(r.Header.Get(LeaseHeader))
	if t == "" {
		if a := r.Header.Get("Authorization"); len(a) > 7 && strings.EqualFold(a[:7], "bearer ") {
			t = strings.TrimSpace(a[7:])
		}
	}
	switch strings.ToLower(t) {
	case "", config.RigLeaseNone, "ollama":
		return ""
	}
	return t
}

func (g *Gate) record(r *http.Request, f Facts, v Verdict, status int, start time.Time) {
	if g.ledger == nil {
		return
	}
	g.ledger.Write(Entry{
		TS:         start.UTC().Format(time.RFC3339Nano),
		Method:     r.Method,
		Path:       r.URL.Path,
		Decision:   v.Decision,
		Status:     status,
		DurationMS: g.now().Sub(start).Milliseconds(),
		Holder:     f.Holder,
		TokenOK:    f.TokenMatches,
		Cancelled:  r.Context().Err() != nil,
		Reason:     v.Reason,
	})
}

func writeJSONError(w http.ResponseWriter, status int, typ, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"error": map[string]string{"type": typ, "message": msg},
	})
}

// statusWriter records the upstream status and keeps streaming working.
type statusWriter struct {
	http.ResponseWriter
	status int
	wrote  bool
}

func (s *statusWriter) WriteHeader(code int) {
	if !s.wrote {
		s.status, s.wrote = code, true
	}
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusWriter) Write(b []byte) (int, error) {
	s.wrote = true
	return s.ResponseWriter.Write(b)
}

// Flush forwards to the underlying writer so FlushInterval=-1 reaches the client.
func (s *statusWriter) Flush() {
	if f, ok := s.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Unwrap lets http.ResponseController reach the underlying writer.
func (s *statusWriter) Unwrap() http.ResponseWriter { return s.ResponseWriter }
