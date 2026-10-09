package apiserver

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStaticSecurityDefaultsWire(t *testing.T) {
	ts := staticServer(t, "immutable")
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	first := head(t, ts.URL+"/clip-2026-09-26.mp4", nil)
	for _, tc := range []struct {
		path     string
		status   int
		modified string
	}{
		{"/clip-2026-09-26.mp4", 200, ""}, {"/missing", 404, ""}, {"/index.html", 301, ""},
		{"/./clip-2026-09-26.mp4", 307, ""},
		{"//clip-2026-09-26.mp4", 307, ""},
		{"/clip-2026-09-26.mp4", 304, first.Header.Get("Last-Modified")},
	} {
		req, _ := http.NewRequest("GET", ts.URL+tc.path, nil)
		if tc.modified != "" {
			req.Header.Set("If-Modified-Since", tc.modified)
		}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != tc.status {
			t.Fatalf("%s: %d", tc.path, resp.StatusCode)
		}
		for name, want := range map[string]string{"X-Content-Type-Options": "nosniff", "X-Frame-Options": "DENY", "Content-Security-Policy": "frame-ancestors 'none'", "Referrer-Policy": "no-referrer"} {
			if got := resp.Header.Get(name); got != want {
				t.Errorf("%s %s=%q, want %q", tc.path, name, got, want)
			}
		}
	}
}

func TestStaticHeaderOptions(t *testing.T) {
	for _, tc := range []struct {
		off   bool
		flags []string
		want  string
	}{
		{false, nil, "DENY"}, {false, []string{"X-Frame-Options: SAMEORIGIN", "x-frame-options: DENY"}, "DENY"},
		{true, nil, ""}, {true, []string{"X-Frame-Options: SAMEORIGIN"}, "SAMEORIGIN"},
	} {
		headers, err := ParseStaticHeaders("assets", tc.off, tc.flags)
		if err != nil {
			t.Fatal(err)
		}
		w := httptest.NewRecorder()
		staticHeadersHandler(headers, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(404) })).ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
		if got := w.Result().Header.Get("X-Frame-Options"); got != tc.want {
			t.Errorf("got %q want %q", got, tc.want)
		}
	}
	for _, s := range []string{"bad", "Bad Name: x", ": x", "X: ", "X: a\nb", "X: a\rb", "X: a\tb", "X: a\x7fb"} {
		if _, err := ParseStaticHeaders("assets", false, []string{s}); err == nil {
			t.Errorf("accepted %q", s)
		}
	}
	for _, tc := range []struct {
		off   bool
		flags []string
	}{{true, nil}, {false, []string{"X: y"}}} {
		if _, err := ParseStaticHeaders("", tc.off, tc.flags); err == nil {
			t.Error("accepted flags without --static")
		}
	}
}

func TestStaticHeadersDoNotReachAPI(t *testing.T) {
	ts := staticServer(t, "")
	for _, path := range []string{"/api/_health", "/api/_meta/modules"} {
		resp := head(t, ts.URL+path, nil)
		for _, name := range []string{"X-Frame-Options", "Content-Security-Policy", "Referrer-Policy"} {
			if got := resp.Header.Get(name); got != "" {
				t.Errorf("%s: %s=%q", path, name, got)
			}
		}
	}
}

func TestStaticConfigOverridesWire(t *testing.T) {
	for _, tc := range []struct {
		off   bool
		flags []string
		want  string
	}{
		{false, []string{"X-Frame-Options: SAMEORIGIN"}, "SAMEORIGIN"},
		{false, []string{"X-Frame-Options: SAMEORIGIN", "x-frame-options: DENY"}, "DENY"},
		{true, nil, ""}, {true, []string{"x-frame-options: SAMEORIGIN"}, "SAMEORIGIN"},
	} {
		srv := New(t.TempDir(), Config{StaticPath: t.TempDir(), StaticHeaders: StaticHeaderOptions{Disabled: tc.off, Headers: tc.flags}})
		ts := httptest.NewServer(srv.buildRoutes())
		resp := head(t, ts.URL+"/missing", nil)
		ts.Close()
		srv.Close()
		if resp.StatusCode != 404 || resp.Header.Get("X-Frame-Options") != tc.want {
			t.Fatalf("%+v: %d %v", tc, resp.StatusCode, resp.Header)
		}
	}
}

func TestStaticHeadersProtocolAndProxyIsolation(t *testing.T) {
	srv := New(t.TempDir(), Config{StaticPath: t.TempDir(), MCP: true, A2A: true})
	t.Cleanup(func() { srv.Close() })
	mux := srv.buildRoutes()
	for _, path := range []string{"/mcp/", "/a2a/", "/.well-known/agent.json"} {
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest("HEAD", path, nil))
		for _, name := range []string{"X-Frame-Options", "Content-Security-Policy", "Referrer-Policy"} {
			if got := w.Result().Header.Get(name); got != "" {
				t.Errorf("%s: %s=%q", path, name, got)
			}
		}
	}
	// Proxy responses do not pass through the static wrapper, even on errors.
	proxy := New(t.TempDir(), Config{FrontendPath: t.TempDir()})
	defer proxy.Close()
	w := httptest.NewRecorder()
	proxy.buildRoutes().ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	if w.Result().Header.Get("X-Frame-Options") != "" || w.Result().Header.Get("Content-Security-Policy") != "" {
		t.Fatal("static defaults reached proxy")
	}
}

func TestStaticConfigValidationBeforeServing(t *testing.T) {
	for _, cfg := range []Config{
		{StaticHeaders: StaticHeaderOptions{Disabled: true}},
		{StaticHeaders: StaticHeaderOptions{Headers: []string{"X: y"}}},
		{StaticPath: "assets", StaticHeaders: StaticHeaderOptions{Headers: []string{"Bad Name: y"}}},
	} {
		srv := New(t.TempDir(), cfg)
		err := srv.Start()
		srv.Close()
		if err == nil {
			t.Fatal("invalid static config reached serving")
		}
	}
}

func TestStaticHeadersExcludeAPICleanPathRedirect(t *testing.T) {
	ts := staticServer(t, "")
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := client.Get(ts.URL + "/api//_health")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode < 300 || resp.StatusCode >= 400 {
		t.Fatalf("expected redirect: %d", resp.StatusCode)
	}
	if resp.Header.Get("X-Frame-Options") != "" {
		t.Fatalf("static default reached API redirect: %v", resp.Header)
	}
}
