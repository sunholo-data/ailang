package originpolicy

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const (
	host     = "127.0.0.1:1957"
	same     = "http://127.0.0.1:1957"
	listed   = "http://localhost:3000"
	foreign  = "https://evil.example"
	nullOrig = "null"
)

func req(method, origin string) *http.Request {
	r := httptest.NewRequest(method, "http://"+host+"/api/x", nil)
	r.Host = host
	if origin != "" {
		r.Header.Set("Origin", origin)
	}
	return r
}

func apply(p *Policy, method, origin string) (*httptest.ResponseRecorder, bool) {
	w := httptest.NewRecorder()
	ok := p.Apply(w, req(method, origin))
	return w, ok
}

// The default (no flags) policy is same-origin only: a foreign page gets no
// ACAO on reads and a 403 before dispatch on anything state-changing.
func TestApply_DefaultRefusesForeignStateChange(t *testing.T) {
	p := New(false, nil, "GET, POST")
	for _, m := range []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch, http.MethodOptions} {
		w, ok := apply(p, m, foreign)
		if ok || w.Code != http.StatusForbidden {
			t.Errorf("%s from foreign origin: ran=%v status=%d, want refused 403", m, ok, w.Code)
		}
		if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
			t.Errorf("%s from foreign origin: ACAO=%q, want none", m, got)
		}
	}
	w, ok := apply(p, http.MethodPost, nullOrig)
	if ok || w.Code != http.StatusForbidden {
		t.Errorf("POST from Origin: null ran=%v status=%d, want refused", ok, w.Code)
	}
	// Safe methods run (a browser would load them anyway via <img>/<script>),
	// but with no ACAO the foreign page cannot read the answer.
	for _, m := range []string{http.MethodGet, http.MethodHead} {
		w, ok := apply(p, m, foreign)
		if !ok {
			t.Errorf("%s from foreign origin refused; want run without ACAO", m)
		}
		if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
			t.Errorf("%s from foreign origin: ACAO=%q, want none", m, got)
		}
	}
}

func TestApply_SameOriginAndNoOriginPass(t *testing.T) {
	p := New(false, nil, "GET, POST")
	for _, origin := range []string{same, "HTTP://127.0.0.1:1957", ""} {
		w, ok := apply(p, http.MethodPost, origin)
		if !ok || w.Code != http.StatusOK {
			t.Errorf("POST with Origin %q: ran=%v status=%d, want run", origin, ok, w.Code)
		}
	}
	// Same host on a different port is a different origin.
	if _, ok := apply(p, http.MethodPost, "http://127.0.0.1:3000"); ok {
		t.Error("POST from a different port on the same host ran; want refused")
	}
}

func TestApply_Allowlisted(t *testing.T) {
	p := New(false, []string{listed}, "GET, POST")
	w, ok := apply(p, http.MethodOptions, listed)
	if ok || w.Code != http.StatusNoContent {
		t.Fatalf("preflight from listed origin: ran=%v status=%d, want 204 without dispatch", ok, w.Code)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != listed {
		t.Fatalf("preflight ACAO=%q, want %q", got, listed)
	}
	if got := w.Header().Get("Access-Control-Allow-Methods"); got != "GET, POST" {
		t.Fatalf("Allow-Methods=%q", got)
	}
	w, ok = apply(p, http.MethodPost, listed)
	if !ok || w.Header().Get("Access-Control-Allow-Origin") != listed {
		t.Fatalf("POST from listed origin: ran=%v ACAO=%q", ok, w.Header().Get("Access-Control-Allow-Origin"))
	}
	if !strings.Contains(w.Header().Get("Vary"), "Origin") {
		t.Fatalf("Vary=%q, want Origin", w.Header().Get("Vary"))
	}
	if _, ok := apply(p, http.MethodPost, foreign); ok {
		t.Fatal("POST from unlisted origin ran with an allowlist set")
	}
}

func TestApply_AnyOrigin(t *testing.T) {
	p := New(true, nil, "GET")
	w, ok := apply(p, http.MethodOptions, foreign)
	if ok || w.Code != http.StatusNoContent || w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("any-origin preflight: ran=%v status=%d ACAO=%q", ok, w.Code, w.Header().Get("Access-Control-Allow-Origin"))
	}
	w, ok = apply(p, http.MethodPost, foreign)
	if !ok || w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("any-origin POST: ran=%v ACAO=%q", ok, w.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestCheckWebSocket(t *testing.T) {
	p := New(false, []string{listed}, "GET")
	cases := []struct {
		origin       string
		allowMissing bool
		ok           bool
	}{
		{same, false, true},
		{listed, false, true},
		{foreign, false, false},
		{foreign, true, false},
		{nullOrig, true, false},
		{"", true, true},   // CLI / server-to-server client
		{"", false, false}, // serve-api's stricter rule
	}
	for _, c := range cases {
		err := p.CheckWebSocket(req(http.MethodGet, c.origin), c.allowMissing)
		if (err == nil) != c.ok {
			t.Errorf("CheckWebSocket(origin=%q, allowMissing=%v) err=%v, want ok=%v", c.origin, c.allowMissing, err, c.ok)
		}
	}
	// --cors (any origin) is NOT a WebSocket grant: CORS does not govern WS.
	if err := New(true, nil, "GET").CheckWebSocket(req(http.MethodGet, foreign), true); err == nil {
		t.Error("any-origin CORS mode admitted a foreign WebSocket")
	}
}

func TestValidate(t *testing.T) {
	if err := Validate(true, []string{listed}); err == nil {
		t.Error("any + allowlist: want error")
	}
	for _, ok := range []string{"https://daneel.ts.net", "http://localhost:5173", "http://127.0.0.1:8080"} {
		if err := Validate(false, []string{ok}); err != nil {
			t.Errorf("origin %q rejected: %v", ok, err)
		}
	}
	for _, bad := range []string{"daneel.ts.net", "https://daneel.ts.net/", "https://daneel.ts.net/app",
		"*", "null", "ftp://x.example", "https://", "https://x.example?q=1", "https://u@x.example", ""} {
		if err := Validate(false, []string{bad}); err == nil {
			t.Errorf("malformed origin %q accepted", bad)
		}
	}
}
