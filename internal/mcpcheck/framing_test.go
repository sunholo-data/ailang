package mcpcheck

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestAuthorizationFraming(t *testing.T) {
	for _, tc := range []struct {
		name, target, xfo, csp, content string
		code                            int
		want                            string
	}{
		{"xfo", "anthropic", "DENY", "", "text/html; charset=utf-8", 200, Pass},
		{"csp", "both", "", "default-src 'self'; frame-ancestors 'none'", "text/html", 200, Pass},
		{"missing", "anthropic", "", "", "text/html", 200, Fail},
		{"both", "both", "", "", "text/html", 200, Fail},
		{"openai", "openai", "", "", "text/html", 200, Warn},
		{"default", "", "", "", "text/html", 200, Fail},
		{"report only", "anthropic", "", "", "text/html", 200, Fail},
		{"unrelated csp", "anthropic", "", "default-src 'none'", "text/html", 200, Fail},
		{"bad xfo", "anthropic", "ALLOWALL", "", "text/html", 200, Fail},
		{"rejected", "both", "", "", "text/html", 400, Warn},
		{"json", "both", "DENY", "", "application/json", 200, Warn},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/authorize" {
					q := r.URL.Query()
					// Pin every dummy parameter: these are safe, deliberately
					// nonregistered credentials and a real S256 verifier digest.
					digest := sha256.Sum256([]byte("ailang-mcp-check-dummy-verifier-01234567890123456789"))
					for key, want := range map[string]string{
						"client_id":             "ailang-mcp-check-dummy",
						"redirect_uri":          "https://example.invalid/ailang-mcp-check/callback",
						"state":                 "ailang-mcp-check-dummy-state",
						"response_type":         "code",
						"code_challenge_method": "S256",
						"code_challenge":        base64.RawURLEncoding.EncodeToString(digest[:]),
					} {
						if got := q.Get(key); got != want {
							t.Errorf("%s = %q, want %q", key, got, want)
						}
					}
					if q.Get("existing") != "kept" {
						t.Error("existing query lost")
					}
					w.Header().Set("X-Frame-Options", "DENY") // only the final page determines the verdict
					http.Redirect(w, r, "/consent", http.StatusFound)
					return
				}
				w.Header().Set("Content-Type", tc.content)
				w.Header().Set("X-Frame-Options", tc.xfo)
				w.Header().Set("Content-Security-Policy", tc.csp)
				w.Header().Set("Content-Security-Policy-Report-Only", "frame-ancestors 'none'")
				w.WriteHeader(tc.code)
			}))
			defer srv.Close()
			s := session{opts: Options{Target: tc.target, Client: srv.Client()}}
			f := s.checkAuthorizationFraming(context.Background(), srv.URL+"/authorize?existing=kept")
			if f.Status != tc.want {
				t.Fatalf("%+v want %s", f, tc.want)
			}
		})
	}
}

func TestAuthorizationFramingInconclusive(t *testing.T) {
	s := session{opts: Options{Target: "both", Client: http.DefaultClient}}
	for _, endpoint := range []string{"", "://invalid"} {
		f := s.checkAuthorizationFraming(context.Background(), endpoint)
		want := Warn
		if endpoint == "" {
			want = Skip
		}
		if f.Status != want {
			t.Fatalf("%+v want %s", f, want)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if f := s.checkAuthorizationFraming(ctx, "http://127.0.0.1:1"); f.Status != Warn || !strings.Contains(f.Message, "canceled") {
		t.Fatalf("%+v", f)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, "/loop", 302) }))
	defer srv.Close()
	s.opts.Client = srv.Client()
	if f := s.checkAuthorizationFraming(context.Background(), srv.URL); f.Status != Warn || !strings.Contains(f.Message, "redirect") {
		t.Fatalf("%+v", f)
	}
}

func TestAuthorizationMetadataIncludesFraming(t *testing.T) {
	var endpoint string
	var probes atomic.Int32
	missingPKCE := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/.well-known/") {
			w.Header().Set("Content-Type", "application/json")
			pkce := `["S256"]`
			if missingPKCE {
				pkce = `[]`
			}
			_, _ = w.Write([]byte(`{"code_challenge_methods_supported":` + pkce + `,"registration_endpoint":"https://example.com/register","authorization_endpoint":"` + endpoint + `"}`))
			return
		}
		probes.Add(1)
		if r.URL.Path != "/authorize" {
			t.Errorf("unexpected probe path %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "text/html")
		w.Header().Set("X-Frame-Options", "SAMEORIGIN")
	}))
	defer srv.Close()
	endpoint = srv.URL + "/authorize"
	s := session{opts: Options{Client: srv.Client()}}
	fs := s.checkAuthorizationServer(context.Background(), srv.URL)
	if len(fs) != 2 || fs[0].Status != Pass || fs[1].Check != "authorization-framing" || fs[1].Status != Pass {
		t.Fatalf("%+v", fs)
	}
	if probes.Load() != 1 {
		t.Fatalf("discovered endpoint probed %d times", probes.Load())
	}
	// Framing still composes with existing metadata failures.
	missingPKCE = true
	fs = s.checkAuthorizationServer(context.Background(), srv.URL)
	if len(fs) != 2 || fs[0].Check != "authorization-server" || fs[0].Status != Fail || fs[1].Status != Pass || probes.Load() != 2 {
		t.Fatalf("metadata failure must retain probe finding: %+v, probes=%d", fs, probes.Load())
	}
	missingPKCE = false
	endpoint = ""
	fs = s.checkAuthorizationServer(context.Background(), srv.URL)
	if len(fs) != 2 || fs[0].Status != Pass || fs[1].Status != Skip || probes.Load() != 2 {
		t.Fatalf("%+v", fs)
	}
	fs = s.checkAuthorizationServer(context.Background(), "")
	if len(fs) != 2 || fs[0].Status != Skip || fs[1].Status != Skip {
		t.Fatalf("%+v", fs)
	}
}

type framingTransport func(*http.Request) (*http.Response, error)

func (f framingTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestAuthorizationFramingBoundsContextAndClientFailure(t *testing.T) {
	s := session{opts: Options{Client: &http.Client{Transport: framingTransport(func(r *http.Request) (*http.Response, error) {
		deadline, ok := r.Context().Deadline()
		if !ok || time.Until(deadline) > 10*time.Second {
			t.Error("probe requires a bounded context")
		}
		return nil, errors.New("endpoint unreachable")
	})}}}
	f := s.checkAuthorizationFraming(context.Background(), "https://example.invalid/authorize")
	if f.Status != Warn || !strings.Contains(f.Message, "endpoint unreachable") {
		t.Fatalf("%+v", f)
	}
}
