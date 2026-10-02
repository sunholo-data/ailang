package effects

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

// M-NET-SCOPE-PUBLIC (#1522, S0 of m-mcp-oauth-package).
//
// (i)  a redirect hop never lands on loopback, link-local or private
//      addresses, whatever AllowLocalhost/AllowMetadata say;
// (ii) under Net[scope=public] every Net call (hop 0 and redirects) runs
//      with allowLocalhost/allowMetadata forced off, checked at dial time
//      after DNS (resolvePinned).
//
// The oracle is the /final handler's hit counter, never the error text: a
// refusal that still reached /final is a failure.
//
// Fixture: one listener on 127.0.0.1:P. The injected resolver maps names
// (public.example → 203.0.113.10, evil.example → the case's target IP), and
// the injected dialer routes 203.0.113.10:P and 169.254.169.254:P to the
// listener, so an allowed connection is observable instead of a dial error.

type scopeFixture struct {
	srv       *httptest.Server
	finalHits atomic.Int32
	location  string // where /start redirects; set per case
	port      string
	probe     *netProbe
	ctx       *EffContext
}

const (
	publicTestIP   = "203.0.113.10"
	metadataTestIP = "169.254.169.254"
)

func newScopeFixture(t *testing.T, evilIP string) *scopeFixture {
	t.Helper()
	f := &scopeFixture{}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/start":
			http.Redirect(w, r, f.location, http.StatusFound)
		case "/final":
			f.finalHits.Add(1)
			_, _ = w.Write([]byte("secret"))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(f.srv.Close)
	f.port = portOf(f.srv)

	f.probe = &netProbe{routes: map[string]string{
		publicTestIP + ":" + f.port:   "127.0.0.1:" + f.port,
		metadataTestIP + ":" + f.port: "127.0.0.1:" + f.port,
	}}
	f.probe.resolve = func(host string) ([]net.IP, error) {
		switch host {
		case "public.example", "other.example":
			return []net.IP{net.ParseIP(publicTestIP)}, nil
		case "localhost":
			return []net.IP{net.ParseIP("127.0.0.1")}, nil
		case "evil.example", "metadata.google.internal":
			return []net.IP{net.ParseIP(evilIP)}, nil
		}
		return nil, fmt.Errorf("no such host %s", host)
	}
	f.ctx = newNetProbeCtx(t, f.probe)
	forceNoProxy(f.ctx)
	// serve-api's permissive settings (cmd/ailang/serve_api.go).
	f.ctx.Net.AllowHTTP = true
	f.ctx.Net.AllowLocalhost = true
	f.ctx.Net.AllowMetadata = true
	return f
}

func (f *scopeFixture) url(host, path string) string {
	return "http://" + net.JoinHostPort(host, f.port) + path
}

func (f *scopeFixture) get(rawURL string) error {
	_, err := Call(f.ctx, "Net", "httpGet", one(rawURL))
	return err
}

func (f *scopeFixture) assertRefused(t *testing.T, err error) {
	t.Helper()
	if hits := f.finalHits.Load(); hits != 0 {
		t.Fatalf("SSRF: the forbidden target was reached (/final hit %d times, err=%v)", hits, err)
	}
	if err == nil || !strings.Contains(err.Error(), "E_NET_") {
		t.Fatalf("expected a typed E_NET_* refusal, got %v", err)
	}
}

func (f *scopeFixture) assertReached(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("expected the request to succeed, got %v", err)
	}
	if hits := f.finalHits.Load(); hits != 1 {
		t.Fatalf("expected /final to be reached once, got %d", hits)
	}
}

// --- (i) redirect hops ---

func TestRedirectHop_HostnameToLoopbackRefused(t *testing.T) {
	f := newScopeFixture(t, "127.0.0.1")
	f.location = f.url("evil.example", "/final")
	f.assertRefused(t, f.get(f.url("public.example", "/start")))
}

func TestRedirectHop_HostnameToMetadataRefused(t *testing.T) {
	f := newScopeFixture(t, metadataTestIP)
	f.location = f.url("evil.example", "/final")
	f.assertRefused(t, f.get(f.url("public.example", "/start")))
}

func TestRedirectHop_LiteralLinkLocalRefused(t *testing.T) {
	f := newScopeFixture(t, metadataTestIP)
	f.location = f.url(metadataTestIP, "/final")
	f.assertRefused(t, f.get(f.url("public.example", "/start")))
}

func TestRedirectHop_LocalhostNameRefused(t *testing.T) {
	f := newScopeFixture(t, "127.0.0.1")
	f.location = f.url("localhost", "/final")
	f.assertRefused(t, f.get(f.url("public.example", "/start")))
}

func TestRedirectHop_PrivateHostnameRefused(t *testing.T) {
	f := newScopeFixture(t, "10.0.0.5")
	f.location = f.url("evil.example", "/final")
	f.assertRefused(t, f.get(f.url("public.example", "/start")))
}

func TestRedirectHop_PublicTargetStillFollowed(t *testing.T) {
	f := newScopeFixture(t, "127.0.0.1")
	f.location = f.url("other.example", "/final")
	f.assertReached(t, f.get(f.url("public.example", "/start")))
}

// gcp_auth regression: a DIRECT (hop-0) metadata call with bare Net and
// AllowMetadata keeps working.
func TestDirectMetadataCall_BareNetStillAllowed(t *testing.T) {
	f := newScopeFixture(t, metadataTestIP)
	f.assertReached(t, f.get(f.url("metadata.google.internal", "/final")))
}

// A direct localhost call with bare Net + AllowLocalhost keeps working.
func TestDirectLocalhostCall_BareNetStillAllowed(t *testing.T) {
	f := newScopeFixture(t, "127.0.0.1")
	f.assertReached(t, f.get(f.url("evil.example", "/final")))
}

// --- (ii) Net[scope=public] ---

func TestNetScopePublic_HostnameToLoopbackRefused(t *testing.T) {
	f := newScopeFixture(t, "127.0.0.1")
	f.ctx.PushNetScope("public")
	defer f.ctx.PopNetScope("public")
	f.assertRefused(t, f.get(f.url("evil.example", "/final")))
}

func TestNetScopePublic_HostnameToMetadataRefused(t *testing.T) {
	f := newScopeFixture(t, metadataTestIP)
	f.ctx.PushNetScope("public")
	defer f.ctx.PopNetScope("public")
	f.assertRefused(t, f.get(f.url("metadata.google.internal", "/final")))
}

func TestNetScopePublic_LiteralLoopbackRefused(t *testing.T) {
	f := newScopeFixture(t, "127.0.0.1")
	f.ctx.PushNetScope("public")
	defer f.ctx.PopNetScope("public")
	f.assertRefused(t, f.get(f.url("127.0.0.1", "/final")))
}

func TestNetScopePublic_PublicHostStillAllowed(t *testing.T) {
	f := newScopeFixture(t, "127.0.0.1")
	f.ctx.PushNetScope("public")
	defer f.ctx.PopNetScope("public")
	f.assertReached(t, f.get(f.url("public.example", "/final")))
}

func TestNetScopePublic_PopRestoresBareBehaviour(t *testing.T) {
	f := newScopeFixture(t, metadataTestIP)
	f.ctx.PushNetScope("public")
	f.ctx.PushNetScope("public") // nested public frame
	f.ctx.PopNetScope("public")
	if !f.ctx.NetScopePublic() {
		t.Fatal("an outer public frame must survive the inner frame's pop")
	}
	f.ctx.PopNetScope("public")
	if f.ctx.NetScopePublic() {
		t.Fatal("scope still public after every frame popped")
	}
	f.assertReached(t, f.get(f.url("metadata.google.internal", "/final")))
}

func TestNetScopePublic_EmptyScopeIsNoop(t *testing.T) {
	ctx := NewEffContext(nil)
	ctx.PushNetScope("")
	if ctx.NetScopePublic() {
		t.Fatal("bare Net must push nothing")
	}
	ctx.PopNetScope("") // must not underflow
	if ctx.NetScopePublic() {
		t.Fatal("pop of bare Net changed the scope")
	}
}

func TestNetScopePublic_CloneResets(t *testing.T) {
	ctx := NewEffContext(nil)
	ctx.PushNetScope("public")
	clone := ctx.Clone().(*EffContext)
	if clone.NetScopePublic() {
		t.Fatal("a cloned (per-request) context inherited another request's public scope")
	}
	clone.PushNetScope("public")
	ctx.PopNetScope("public")
	if !clone.NetScopePublic() {
		t.Fatal("the original context's pop leaked into the clone")
	}
}
