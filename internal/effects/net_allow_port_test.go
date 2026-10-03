package effects

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
)

// #1558 — port-qualified net_allow entries. A host serving a local mock had
// to open ALL of loopback to a confined program; a `127.0.0.1:PORT` entry
// now opens exactly that port. The recording handlers are the oracle: a
// refusal is proven by the forbidden listener never being reached, not by
// the error text alone.

// hitServer is a 127.0.0.1 listener that counts arrivals.
type hitServer struct {
	srv  *httptest.Server
	hits atomic.Int32
}

func newHitServer(t *testing.T, h func(w http.ResponseWriter, r *http.Request)) *hitServer {
	t.Helper()
	s := &hitServer{}
	s.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.hits.Add(1)
		if h != nil {
			h(w, r)
			return
		}
		_, _ = io.WriteString(w, "mock-ok")
	}))
	t.Cleanup(s.srv.Close)
	return s
}

func (s *hitServer) port() string { return portOf(s.srv) }

func (s *hitServer) loopbackURL(path string) string {
	return "http://127.0.0.1:" + s.port() + path
}

// policyNetCtx is the EffContext a --policy run builds for Net: the policy's
// net_allow becomes AllowedDomains, net_allow_http becomes AllowHTTP, and
// restricted mode never sets AllowLocalhost. Real resolver, real dialer.
func policyNetCtx(allow ...string) *EffContext {
	ctx := NewEffContext(nil)
	ctx.Grant(NewCapability("Net"))
	ctx.Net = NewNetContext()
	ctx.Net.AllowHTTP = true
	ctx.Net.AllowLocalhost = false
	ctx.Net.AllowedDomains = allow
	forceNoProxy(ctx)
	return ctx
}

func TestParseNetAllowEntry(t *testing.T) {
	ok := []struct{ in, host, port string }{
		{"api.example.com", "api.example.com", ""},
		{"API.Example.com.", "api.example.com", ""},
		{"*.example.com", "*.example.com", ""},
		{"api.example.com:8443", "api.example.com", "8443"},
		{"127.0.0.1", "127.0.0.1", ""},
		{"127.0.0.1:7655", "127.0.0.1", "7655"},
		{"localhost:7655", "localhost", "7655"},
		{"::1", "::1", ""},
		{"[::1]", "::1", ""},
		{"[::1]:7655", "::1", "7655"},
		{" 127.0.0.1:1 ", "127.0.0.1", "1"},
		{"127.0.0.1:65535", "127.0.0.1", "65535"},
	}
	for _, c := range ok {
		h, p, err := ParseNetAllowEntry(c.in)
		if err != nil || h != c.host || p != c.port {
			t.Errorf("ParseNetAllowEntry(%q) = (%q, %q, %v), want (%q, %q, nil)", c.in, h, p, err, c.host, c.port)
		}
	}
	bad := []string{"", ":7655", "127.0.0.1:", "127.0.0.1:0", "127.0.0.1:65536", "127.0.0.1:http",
		"127.0.0.1:-1", "[::1", "[::1]x", "[example.com]:80", "a:b:c", "host:80:90", "[::1]:", "http://x"}
	for _, in := range bad {
		if h, p, err := ParseNetAllowEntry(in); err == nil {
			t.Errorf("ParseNetAllowEntry(%q) = (%q, %q, nil), want an error", in, h, p)
		}
	}
}

func TestIsAllowedTarget_PortScoping(t *testing.T) {
	allowed := []string{"any.example", "api.example:8443", "*.svc.example:9000", "127.0.0.1:7655", "[::1]:7655"}
	cases := []struct {
		host, port string
		want       bool
	}{
		{"any.example", "443", true}, // a bare entry admits every port
		{"any.example", "8080", true},
		{"api.example", "8443", true},
		{"api.example", "443", false}, // port-qualified: that port only
		{"a.svc.example", "9000", true},
		{"a.svc.example", "9001", false},
		{"127.0.0.1", "7655", true},
		{"127.0.0.1", "7656", false},
		{"::1", "7655", true},
		{"::1", "80", false},
		{"localhost", "7655", false}, // names are exact: 127.0.0.1:P does not admit localhost:P
	}
	for _, c := range cases {
		if got := isAllowedTarget(c.host, c.port, allowed); got != c.want {
			t.Errorf("isAllowedTarget(%q, %q) = %v, want %v", c.host, c.port, got, c.want)
		}
	}
}

// The issue's end-to-end shape: restricted-mode Net, net_allow names one
// loopback port, a second loopback listener stands beside it.
func TestNetAllowPortScopedLoopback_EndToEnd(t *testing.T) {
	mock := newHitServer(t, nil)
	other := newHitServer(t, nil)

	t.Run("allowed_port_reachable", func(t *testing.T) {
		ctx := policyNetCtx("127.0.0.1:" + mock.port())
		before := mock.hits.Load()
		v, err := Call(ctx, "Net", "httpGet", one(mock.loopbackURL("/data")))
		if err != nil {
			t.Fatalf("allowed loopback port refused: %v", err)
		}
		if got := v.String(); !strings.Contains(got, "mock-ok") {
			t.Errorf("body = %q, want mock-ok", got)
		}
		if mock.hits.Load() != before+1 {
			t.Errorf("mock not reached")
		}
	})

	t.Run("other_loopback_port_refused", func(t *testing.T) {
		ctx := policyNetCtx("127.0.0.1:" + mock.port())
		_, err := Call(ctx, "Net", "httpGet", one(other.loopbackURL("/data")))
		if err == nil {
			t.Fatal("a loopback port outside net_allow was reachable")
		}
		if other.hits.Load() != 0 {
			t.Errorf("CONTAINMENT FAILURE: other loopback port reached %d times", other.hits.Load())
		}
	})

	t.Run("bare_loopback_still_refused_in_restricted", func(t *testing.T) {
		// The issue's repro: net_allow = ["127.0.0.1"] without a localhost
		// grant. Restricted mode never sets AllowLocalhost, so this stays
		// E_NET_IP_BLOCKED — a bare entry is not a port-scoped grant.
		ctx := policyNetCtx("127.0.0.1")
		before := mock.hits.Load()
		_, err := Call(ctx, "Net", "httpGet", one(mock.loopbackURL("/data")))
		if err == nil || !strings.Contains(err.Error(), "E_NET_IP_BLOCKED") {
			t.Fatalf("bare loopback entry: want E_NET_IP_BLOCKED, got %v", err)
		}
		if mock.hits.Load() != before {
			t.Errorf("bare loopback entry reached the mock")
		}
	})

	t.Run("grant_is_port_scoped_even_when_host_is_listed_bare", func(t *testing.T) {
		// Outside a policy (--net-allow-domains) a bare entry can sit beside
		// a port-qualified one. The bare entry admits the host on any port
		// for the allowlist, but only the port-qualified one grants loopback.
		ctx := policyNetCtx("127.0.0.1", "127.0.0.1:"+mock.port())
		_, err := Call(ctx, "Net", "httpGet", one(other.loopbackURL("/data")))
		if err == nil || !strings.Contains(err.Error(), "E_NET_IP_BLOCKED") {
			t.Errorf("want E_NET_IP_BLOCKED for an ungranted loopback port, got %v", err)
		}
		if other.hits.Load() != 0 {
			t.Errorf("CONTAINMENT FAILURE: loopback grant leaked to port %s", other.port())
		}
	})

	t.Run("redirect_to_other_loopback_port_refused", func(t *testing.T) {
		redir := newHitServer(t, func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, other.loopbackURL("/final"), http.StatusFound)
		})
		ctx := policyNetCtx("127.0.0.1:" + redir.port())
		_, err := Call(ctx, "Net", "httpGet", one(redir.loopbackURL("/start")))
		if redir.hits.Load() != 1 {
			t.Fatalf("fixture: the allowed port must be reached once, got %d", redir.hits.Load())
		}
		if err == nil {
			t.Error("redirect to another loopback port succeeded")
		}
		if other.hits.Load() != 0 {
			t.Errorf("CONTAINMENT FAILURE: redirect reached another loopback port %d times", other.hits.Load())
		}
	})

	t.Run("redirect_even_to_allowed_port_refused", func(t *testing.T) {
		// Redirect hops never reach loopback (#1522 part (i)); a port grant
		// is for the request the program made, not for where a server sends it.
		redir := newHitServer(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/start" {
				http.Redirect(w, r, "/final", http.StatusFound)
				return
			}
			_, _ = io.WriteString(w, "final")
		})
		ctx := policyNetCtx("127.0.0.1:" + redir.port())
		_, err := Call(ctx, "Net", "httpGet", one(redir.loopbackURL("/start")))
		if err == nil || redir.hits.Load() != 1 {
			t.Errorf("redirect hop reached loopback: err=%v hits=%d", err, redir.hits.Load())
		}
	})

	t.Run("public_scope_overrides_port_grant", func(t *testing.T) {
		ctx := policyNetCtx("127.0.0.1:" + mock.port())
		ctx.PushNetScope("public")
		defer ctx.PopNetScope("public")
		before := mock.hits.Load()
		_, err := Call(ctx, "Net", "httpGet", one(mock.loopbackURL("/data")))
		if err == nil || mock.hits.Load() != before {
			t.Errorf("Net[scope=public] reached a port-granted loopback listener: err=%v", err)
		}
	})
}

// A port-qualified non-loopback host is scoped to that port. Hermetic: the
// name resolves to a public documentation address that the dialer routes to
// the local listener, so loopback rules play no part.
func TestNetAllowPortScopedHost(t *testing.T) {
	srv := newHitServer(t, nil)
	probe := &netProbe{routes: map[string]string{}}
	ctx := newNetProbeCtx(t, probe)
	forceNoProxy(ctx)
	ctx.Net.AllowLocalhost = false
	probe.resolve = func(string) ([]net.IP, error) { return []net.IP{net.ParseIP("203.0.113.10")}, nil }
	port := srv.port()
	probe.routes["203.0.113.10:"+port] = "127.0.0.1:" + port
	ctx.Net.AllowedDomains = []string{"api.example:" + port}

	if _, err := Call(ctx, "Net", "httpGet", one("http://api.example:"+port+"/x")); err != nil {
		t.Fatalf("allowed host:port refused: %v", err)
	}
	if srv.hits.Load() != 1 {
		t.Fatalf("allowed host:port not reached")
	}
	otherPort := "1"
	if port == otherPort {
		otherPort = "2"
	}
	_, err := Call(ctx, "Net", "httpGet", one("http://api.example:"+otherPort+"/x"))
	if err == nil || !strings.Contains(err.Error(), "E_NET_DOMAIN_BLOCKED") {
		t.Errorf("host on another port: want E_NET_DOMAIN_BLOCKED, got %v", err)
	}
	// The default port is the one the scheme implies.
	_, err = Call(ctx, "Net", "httpGet", one("http://api.example/x"))
	if err == nil || !strings.Contains(err.Error(), "E_NET_DOMAIN_BLOCKED") {
		t.Errorf("host on the implied port 80: want E_NET_DOMAIN_BLOCKED, got %v", err)
	}
}

// The pinned direct transport dials only the port the URL was authorized
// for, whatever address the HTTP machinery asks it to dial.
func TestNetAllowPort_DialEnforcesAuthorizedPort(t *testing.T) {
	mock := newHitServer(t, nil)
	other := newHitServer(t, nil)
	ctx := policyNetCtx("127.0.0.1:" + mock.port())
	rt := newNetRoundTripper(ctx)
	u, _ := url.Parse(mock.loopbackURL("/"))
	tr := rt.directTransport("127.0.0.1", u)
	conn, err := tr.DialContext(context.Background(), "tcp", "127.0.0.1:"+other.port())
	if err == nil {
		conn.Close()
		t.Fatal("pinned dialer connected to a port the URL was not authorized for")
	}
	conn, err = tr.DialContext(context.Background(), "tcp", "127.0.0.1:"+mock.port())
	if err != nil {
		t.Fatalf("pinned dialer refused the authorized port: %v", err)
	}
	conn.Close()
}

// Stream shares the destination authorizer: the same entry opens the same
// one port for SSE/NDJSON (http) and for the WebSocket pinned dialer.
func TestStreamAllowPortScopedLoopback(t *testing.T) {
	mock := newHitServer(t, nil)
	other := newHitServer(t, nil)
	newCtx := func(allow ...string) *EffContext {
		ctx := NewEffContext(nil)
		ctx.Grant(NewCapability("Stream"))
		ctx.Stream = NewStreamContext()
		ctx.Stream.AllowHTTP = true
		ctx.Stream.AllowLocalhost = false
		ctx.Stream.AllowedDomains = allow
		return ctx
	}
	ctx := newCtx("127.0.0.1:" + mock.port())

	if err := ctx.Stream.ValidateURL("ws://127.0.0.1:" + mock.port() + "/ws"); err != nil {
		t.Errorf("ValidateURL allowed port: %v", err)
	}
	if err := ctx.Stream.ValidateURL("ws://127.0.0.1:" + other.port() + "/ws"); err == nil {
		t.Error("ValidateURL admitted another loopback port")
	}

	// WebSocket: the pinned dialer connects to the allowed port only.
	wsOK, _ := url.Parse("ws://127.0.0.1:" + mock.port() + "/ws")
	dial, err := streamPolicy(ctx).pinnedDialer(wsOK)
	if err != nil {
		t.Fatalf("pinnedDialer allowed port: %v", err)
	}
	conn, err := dial(context.Background(), "tcp", "ignored:1")
	if err != nil {
		t.Fatalf("pinned dial to the allowed port failed: %v", err)
	}
	conn.Close()
	wsBad, _ := url.Parse("ws://127.0.0.1:" + other.port() + "/ws")
	if _, err := streamPolicy(ctx).pinnedDialer(wsBad); err == nil {
		t.Error("pinnedDialer admitted another loopback port")
	}

	// SSE/NDJSON client: allowed port reached; another port, a bare entry and
	// a redirect to another loopback port are not.
	get := func(ctx *EffContext, raw string) error {
		c, err := streamHTTPClient(ctx, raw)
		if err != nil {
			return err
		}
		resp, err := c.Get(raw)
		if err != nil {
			return err
		}
		resp.Body.Close()
		return nil
	}
	if err := get(ctx, mock.loopbackURL("/sse")); err != nil || mock.hits.Load() != 1 {
		t.Errorf("Stream http allowed port: err=%v hits=%d", err, mock.hits.Load())
	}
	if err := get(ctx, other.loopbackURL("/sse")); err == nil {
		t.Error("Stream http admitted another loopback port")
	}
	if err := get(newCtx("127.0.0.1"), mock.loopbackURL("/sse")); err == nil {
		t.Error("Stream http admitted loopback through a bare entry")
	}
	redir := newHitServer(t, func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, other.loopbackURL("/final"), http.StatusFound)
	})
	if err := get(newCtx("127.0.0.1:"+redir.port()), redir.loopbackURL("/start")); err == nil {
		t.Error("Stream http followed a redirect to another loopback port")
	}
	if other.hits.Load() != 0 {
		t.Errorf("CONTAINMENT FAILURE: Stream reached another loopback port %d times", other.hits.Load())
	}
}
