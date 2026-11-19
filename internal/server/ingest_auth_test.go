package server

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/config"
)

const (
	testOTLPToken = "test-otlp-ingest-token"
	testHubToken  = "test-hub-token"
)

// Public write routes that are NOT ingest, each with why it needs no ingest
// secret. Every other write publicRoutes admits must have an ingestRoutes
// class (TestIngestAuth_EveryPublicWriteHasAClass).
var nonIngestPublicWrites = map[string]string{
	"/":              "static UI / banner: serves no write",
	"/health":        "health probe: serves no write",
	"/benchmarks/":   "read-through of public benchmark data: serves no write",
	"/api/approvals": "secret-approval intake: its own fail-closed ID-token / AILANG_APPROVAL_TOKEN check",
}

func isWriteMethod(m string) bool {
	return m == "*" || m == http.MethodPost || m == http.MethodPut || m == http.MethodPatch || m == http.MethodDelete
}

// ingestGate is the server's two gates (ingest, then read) over a stub mux
// holding the real patterns, each answering 200 "reached".
func ingestGate(s *Server, patterns []string) http.Handler {
	stub := http.NewServeMux()
	for _, p := range patterns {
		stub.HandleFunc(p, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("reached"))
		})
	}
	return s.ingestAuthMiddleware(stub, s.readAuthMiddleware(stub))
}

func enforcingServer(t *testing.T, addr string, opts ...ServerOption) *Server {
	t.Helper()
	base := []ServerOption{
		WithAPIKey(testAPIKey),
		WithIngestAuthMode(config.IngestAuthEnforce),
		WithOTLPIngestToken(testOTLPToken),
		WithHookToken(testHubToken),
	}
	return fullRouteServer(t, addr, append(base, opts...)...)
}

// ingestHeader is canonicalised as a real server would: serve() copies the map
// verbatim.
func ingestHeader(tok string) http.Header {
	return http.Header{http.CanonicalHeaderKey(config.IngestTokenHeader): {tok}}
}

// The classification guard: every write the read gate lets through
// anonymously (publicRoutes) is either an ingest route with a class, or a
// named non-ingest exemption — and an ingest class names only registered,
// allow-listed routes.
func TestIngestAuth_EveryPublicWriteHasAClass(t *testing.T) {
	s := enforcingServer(t, "0.0.0.0:0")
	registered := map[string]bool{}
	for _, p := range registeredPatterns(t, s) {
		registered[p] = true
	}

	var unclassified []string
	for pattern, methods := range publicRoutes {
		for _, m := range methods {
			if !isWriteMethod(m) {
				continue
			}
			_, ingest := ingestRoutes[pattern]
			_, exempt := nonIngestPublicWrites[pattern]
			if ingest == exempt {
				unclassified = append(unclassified, pattern)
			}
		}
	}
	sort.Strings(unclassified)
	for _, p := range unclassified {
		t.Errorf("public write route %q needs exactly one of: an ingestRoutes class (ingest_auth.go) or a nonIngestPublicWrites entry", p)
	}

	for pattern, class := range ingestRoutes {
		if !registered[pattern] {
			t.Errorf("ingestRoutes[%q] (%s) is not a registered route: remove it", pattern, class)
		}
		if !isWantPublic(pattern, http.MethodPost) {
			t.Errorf("ingestRoutes[%q] is not an allow-listed POST in publicRoutes", pattern)
		}
		if ingestTokenEnv[class] == "" {
			t.Errorf("class %q has no token variable", class)
		}
	}
	for pattern := range nonIngestPublicWrites {
		if !registered[pattern] {
			t.Errorf("nonIngestPublicWrites[%q] is not a registered route: remove it", pattern)
		}
	}
}

// Behaviourally, under enforce: an anonymous write to every ingest route is
// refused by the ingest gate, and the right secret — and only that class's
// secret — gets through, in either header form.
func TestIngestAuth_Enforce_RouteTable(t *testing.T) {
	s := enforcingServer(t, "0.0.0.0:0")
	h := ingestGate(s, registeredPatterns(t, s))

	for pattern, class := range ingestRoutes {
		target := strings.TrimPrefix(pattern, "POST ")
		good, other := testHubToken, testOTLPToken
		if class == ingestOTLP {
			good, other = testOTLPToken, testHubToken
		}
		refusals := map[string]http.Header{
			"no token":         nil,
			"wrong token":      ingestHeader("wrong"),
			"other class":      ingestHeader(other),
			"dashboard key":    bearer(testAPIKey),
			"non-bearer basic": {"Authorization": {"Basic " + good}},
		}
		for name, hdr := range refusals {
			w := serve(h, http.MethodPost, target, hdr)
			if w.Code != http.StatusUnauthorized || w.Header().Get("WWW-Authenticate") != ingestAuthRealm {
				t.Errorf("POST %s (%s), %s: got %d %q, want 401 from the ingest gate", target, class, name, w.Code, w.Body.String())
			}
		}
		for name, hdr := range map[string]http.Header{"header": ingestHeader(good), "bearer": bearer(good)} {
			if w := serve(h, http.MethodPost, target, hdr); w.Body.String() != "reached" {
				t.Errorf("POST %s (%s) with the %s token form: got %d %q, want reached", target, class, name, w.Code, w.Body.String())
			}
		}
	}
}

// Fail closed: under enforce a class whose secret is unset refuses every
// write with 503 naming the variable, while the configured class still works.
func TestIngestAuth_Enforce_UnsetTokenFailsClosed(t *testing.T) {
	s := fullRouteServer(t, "0.0.0.0:0", WithAPIKey(testAPIKey),
		WithIngestAuthMode(config.IngestAuthEnforce), WithHookToken(testHubToken))
	h := ingestGate(s, registeredPatterns(t, s))

	w := serve(h, http.MethodPost, "/v1/traces", ingestHeader("anything"))
	if w.Code != http.StatusServiceUnavailable || !strings.Contains(w.Body.String(), config.EnvOTLPIngestToken) {
		t.Errorf("OTLP with %s unset: got %d %q, want 503 naming it", config.EnvOTLPIngestToken, w.Code, w.Body.String())
	}
	if w := serve(h, http.MethodPost, "/v1/traces", nil); w.Code != http.StatusServiceUnavailable {
		t.Errorf("anonymous OTLP with %s unset: got %d, want 503", config.EnvOTLPIngestToken, w.Code)
	}
	if w := serve(h, http.MethodPost, "/api/hooks/claude", bearer(testHubToken)); w.Body.String() != "reached" {
		t.Errorf("hub class with its token: got %d, want reached", w.Code)
	}
}

// Warn (the default when no mode is set) accepts everything and counts what
// lacked a valid token; a valid token is not counted.
func TestIngestAuth_Warn_AcceptsAndCounts(t *testing.T) {
	s := fullRouteServer(t, "0.0.0.0:0", WithAPIKey(testAPIKey), WithOTLPIngestToken(testOTLPToken))
	h := ingestGate(s, registeredPatterns(t, s))

	for _, hdr := range []http.Header{nil, ingestHeader("wrong"), ingestHeader(testOTLPToken)} {
		if w := serve(h, http.MethodPost, "/v1/traces", hdr); w.Body.String() != "reached" {
			t.Errorf("warn mode: POST /v1/traces got %d, want reached", w.Code)
		}
	}
	// Hub token unset: every hub write is unauthenticated, and still accepted.
	if w := serve(h, http.MethodPost, "/api/exec/events", bearer("anything")); w.Body.String() != "reached" {
		t.Errorf("warn mode, hub token unset: got %d, want reached", w.Code)
	}
	if got := s.ingestWarnings.count(ingestOTLP); got != 2 {
		t.Errorf("otlp warn count = %d, want 2 (missing + invalid; the valid write is not counted)", got)
	}
	if got := s.ingestWarnings.count(ingestHub); got != 1 {
		t.Errorf("hub warn count = %d, want 1", got)
	}
	// The read gate is untouched: warn mode opens no read.
	if w := serve(h, http.MethodGet, "/api/inbox", nil); w.Code != http.StatusUnauthorized {
		t.Errorf("warn mode: anonymous GET /api/inbox got %d, want 401", w.Code)
	}
}

func TestIngestAuth_Off_AcceptsWithoutCounting(t *testing.T) {
	s := fullRouteServer(t, "0.0.0.0:0", WithAPIKey(testAPIKey), WithIngestAuthMode(config.IngestAuthOff))
	h := ingestGate(s, registeredPatterns(t, s))
	if w := serve(h, http.MethodPost, "/v1/traces", nil); w.Body.String() != "reached" {
		t.Errorf("off: got %d, want reached", w.Code)
	}
	if s.ingestWarnings != nil {
		t.Error("off mode must not install the warn counter")
	}
}

// The rig's local `ailang serve` receives every Claude Code hook with no
// credential: a loopback bind is exempt even under enforce with no secrets.
func TestIngestAuth_LoopbackExempt(t *testing.T) {
	for _, addr := range []string{"127.0.0.1:1957", "[::1]:1957", "localhost:1957"} {
		s := fullRouteServer(t, addr, WithIngestAuthMode(config.IngestAuthEnforce))
		h := ingestGate(s, registeredPatterns(t, s))
		for _, target := range []string{"/api/hooks/claude", "/api/observatory/hooks", "/v1/traces", "/v1/logs"} {
			if w := serve(h, http.MethodPost, target, nil); w.Body.String() != "reached" {
				t.Errorf("bind %s: anonymous POST %s got %d, want reached (loopback is exempt)", addr, target, w.Code)
			}
		}
	}
}

// Through the server's REAL handler chain: dropping ingestAuthMiddleware from
// httpHandler fails this.
func TestIngestAuth_EndToEnd(t *testing.T) {
	s := enforcingServer(t, "0.0.0.0:0")
	h, _, err := s.httpHandler()
	if err != nil {
		t.Fatal(err)
	}
	post := func(target string, hdr http.Header) *httptest.ResponseRecorder {
		r := httptest.NewRequest(http.MethodPost, target, strings.NewReader(`{"resourceSpans":[]}`))
		r.Header.Set("Content-Type", "application/json")
		for k, v := range hdr {
			r.Header[k] = v
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	for _, target := range []string{"/v1/traces", "/api/hooks/claude", "/api/exec/sessions"} {
		if w := post(target, nil); w.Code != http.StatusUnauthorized || w.Header().Get("WWW-Authenticate") != ingestAuthRealm {
			t.Errorf("anonymous POST %s: got %d, want 401 from the ingest gate", target, w.Code)
		}
	}
	if w := post("/v1/traces", ingestHeader(testOTLPToken)); w.Code != http.StatusOK {
		t.Errorf("POST /v1/traces with the token: got %d %q, want 200 from the receiver", w.Code, w.Body.String())
	}
}

// What an OTLP exporter configured with
// OTEL_EXPORTER_OTLP_HEADERS=x-ailang-ingest-token=<token> puts on the wire
// (lower-case name) is accepted, over a real connection.
func TestIngestAuth_ExporterHeaderOverTheWire(t *testing.T) {
	s := enforcingServer(t, "0.0.0.0:0")
	h, _, err := s.httpHandler()
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(h)
	defer srv.Close()
	for tok, want := range map[string]int{testOTLPToken: http.StatusOK, "wrong": http.StatusUnauthorized} {
		req, _ := http.NewRequest(http.MethodPost, srv.URL+"/v1/traces", strings.NewReader(`{"resourceSpans":[]}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header["x-ailang-ingest-token"] = []string{tok} // sent as written
		resp, err := srv.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		_ = resp.Body.Close()
		if resp.StatusCode != want {
			t.Errorf("token %q in a lower-case header: got %d, want %d", tok, resp.StatusCode, want)
		}
	}
}

// The warn log names every distinct sender once, then summarises per class
// per interval, carrying the count it held back.
func TestIngestWarnLog_RateLimited(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	clock := time.Unix(1_700_000_000, 0)
	l := newIngestWarnLog(time.Minute)
	l.now = func() time.Time { return clock }
	req := func(ua string) *http.Request {
		r := httptest.NewRequest(http.MethodPost, "/v1/traces", nil)
		r.Header.Set("User-Agent", ua)
		r.Header.Set("X-Forwarded-For", "203.0.113.7, 10.0.0.1")
		return r
	}
	lines := func() int { return strings.Count(buf.String(), "ingest auth WARN") }

	l.note(ingestOTLP, "missing", req("OTel-OTLP-Exporter-Go/1.47.0"))
	if lines() != 1 || !strings.Contains(buf.String(), "203.0.113.7") {
		t.Fatalf("first write from a sender must log with its client address; log:\n%s", buf.String())
	}
	for i := 0; i < 5; i++ {
		l.note(ingestOTLP, "missing", req("OTel-OTLP-Exporter-Go/1.47.0"))
	}
	if lines() != 1 {
		t.Fatalf("repeat writes inside the interval must not log; got %d lines", lines())
	}
	l.note(ingestOTLP, "missing", req("curl/8.7.1"))
	if lines() != 2 {
		t.Fatalf("a new sender must log at once; got %d lines", lines())
	}
	clock = clock.Add(time.Minute)
	l.note(ingestOTLP, "missing", req("OTel-OTLP-Exporter-Go/1.47.0"))
	if lines() != 3 || !strings.Contains(buf.String(), "accepted 1 unauthenticated otlp write(s) since the last report (8 total)") {
		t.Fatalf("interval summary missing or miscounted; log:\n%s", buf.String())
	}
	if l.count(ingestOTLP) != 8 || l.count(ingestHub) != 0 {
		t.Errorf("counts otlp=%d hub=%d, want 8 and 0", l.count(ingestOTLP), l.count(ingestHub))
	}
}
