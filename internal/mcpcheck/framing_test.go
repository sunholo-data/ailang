package mcpcheck

import (
	"context"
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
		// A CIMD AS that still rejects the probe client: no page to judge.
		{"rejected", "both", "", "", "text/html", 400, Warn},
		{"json", "both", "DENY", "", "application/json", 200, Warn},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/authorize" {
					q := r.URL.Query()
					// M3c: the CIMD probe client, and a fresh 43-character
					// S256 challenge (the verifier is never sent).
					for key, want := range map[string]string{
						"client_id":             ProbeClientURL,
						"redirect_uri":          ProbeRedirectURI,
						"state":                 "ailang-mcp-check-dummy-state",
						"response_type":         "code",
						"code_challenge_method": "S256",
					} {
						if got := q.Get(key); got != want {
							t.Errorf("%s = %q, want %q", key, got, want)
						}
					}
					if c := q.Get("code_challenge"); len(c) != 43 || !unreservedRe.MatchString(c) {
						t.Errorf("code_challenge = %q, want 43 base64url characters", c)
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
			f := s.checkAuthorizationFraming(context.Background(), cimdMetadata(srv.URL+"/authorize?existing=kept"))
			if f.Status != tc.want {
				t.Fatalf("%+v want %s", f, tc.want)
			}
		})
	}
}

// cimdMetadata is AS metadata that advertises CIMD for public clients and
// S256, so the probe presents ProbeClientURL.
func cimdMetadata(endpoint string) authServerMetadata {
	return authServerMetadata{AuthorizationEndpoint: endpoint, PKCE: []string{"S256"}, CIMD: true, AuthMeth: []string{"none"}}
}

func TestAuthorizationFramingInconclusive(t *testing.T) {
	s := session{opts: Options{Target: "both", Client: http.DefaultClient}}
	for _, endpoint := range []string{"", "://invalid"} {
		f := s.checkAuthorizationFraming(context.Background(), cimdMetadata(endpoint))
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
	if f := s.checkAuthorizationFraming(ctx, cimdMetadata("http://127.0.0.1:1")); f.Status != Warn || !strings.Contains(f.Message, "canceled") {
		t.Fatalf("%+v", f)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, "/loop", 302) }))
	defer srv.Close()
	s.opts.Client = srv.Client()
	if f := s.checkAuthorizationFraming(context.Background(), cimdMetadata(srv.URL)); f.Status != Warn || !strings.Contains(f.Message, "redirect") {
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
			_, _ = w.Write([]byte(`{"code_challenge_methods_supported":` + pkce + `,"client_id_metadata_document_supported":true,"token_endpoint_auth_methods_supported":["none"],"authorization_endpoint":"` + endpoint + `"}`))
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
	// Without S256 the metadata check FAILs and the probe is skipped (M3c):
	// it would send a request the AS cannot accept.
	missingPKCE = true
	fs = s.checkAuthorizationServer(context.Background(), srv.URL)
	if len(fs) != 2 || fs[0].Check != "authorization-server" || fs[0].Status != Fail || fs[1].Status != Skip || probes.Load() != 1 {
		t.Fatalf("metadata failure must retain a (skipped) probe finding: %+v, probes=%d", fs, probes.Load())
	}
	missingPKCE = false
	endpoint = ""
	fs = s.checkAuthorizationServer(context.Background(), srv.URL)
	if len(fs) != 2 || fs[0].Status != Pass || fs[1].Status != Skip || probes.Load() != 1 {
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
	f := s.checkAuthorizationFraming(context.Background(), cimdMetadata("https://example.invalid/authorize"))
	if f.Status != Warn || !strings.Contains(f.Message, "endpoint unreachable") {
		t.Fatalf("%+v", f)
	}
}
