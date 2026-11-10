package server

import (
	"net/http"
	"testing"
)

// The dashboard serves no UI: "/" is a one-line plain-text banner, and any
// path no route claims is a plain 404 — never an HTML page, never a 200
// (an SPA fallback would answer a mistyped API path with 200 + index.html).
func TestRoot_HeadlessBannerAndPlain404(t *testing.T) {
	s := fullRouteServer(t, "0.0.0.0:0", WithAPIKey(testAPIKey))
	h, _, err := s.httpHandler()
	if err != nil {
		t.Fatal(err)
	}

	w := serve(h, http.MethodGet, "/", nil)
	if w.Code != http.StatusOK || w.Body.String() != rootBanner {
		t.Errorf("GET /: got %d %q, want 200 %q", w.Code, w.Body.String(), rootBanner)
	}
	if ct := w.Header().Get("Content-Type"); ct != "text/plain; charset=utf-8" {
		t.Errorf("GET /: Content-Type %q, want text/plain", ct)
	}
	if w := serve(h, http.MethodHead, "/", nil); w.Code != http.StatusOK {
		t.Errorf("HEAD /: got %d, want 200", w.Code)
	}
	if w := serve(h, http.MethodPost, "/", nil); w.Code != http.StatusMethodNotAllowed || w.Header().Get("Allow") != "GET, HEAD" {
		t.Errorf("POST /: got %d Allow=%q, want 405 GET, HEAD", w.Code, w.Header().Get("Allow"))
	}

	// What the deleted SPA used to serve, and paths it would have swallowed.
	for _, p := range []string{"/index.html", "/assets/index.js", "/logo.png", "/control-plane", "/v1/bogus", "/benchmarkz"} {
		w := serve(h, http.MethodGet, p, nil)
		if w.Code != http.StatusNotFound {
			t.Errorf("GET %s: got %d, want 404", p, w.Code)
		}
		if ct := w.Header().Get("Content-Type"); ct != "text/plain; charset=utf-8" {
			t.Errorf("GET %s: Content-Type %q, want text/plain", p, ct)
		}
	}
}
