package effects

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"

	"github.com/sunholo-data/ailang/internal/eval"
)

// This file contains the M1 security-regression tests for the request-aware
// Net proxy/direct RoundTripper (net_proxy.go). They assert the OBSERVED
// dial/CONNECT destination and injected resolver/dial call counters rather
// than merely that a request succeeded.
//
// Go's http.ProxyFromEnvironment caches the proxy config once per process, so
// in-process tests inject a deterministic proxySelector hook (nil in
// production) to choose proxy/no-proxy mode order-independently. One subprocess
// test additionally proves the PRODUCTION default (http.ProxyFromEnvironment)
// honors a real process-level proxy.

// netProxyEnvBody is the body the fake proxy returns so the subprocess helper
// can recognise a proxy-served (not direct) response.
const netProxyEnvBody = "env-proxy-ok"

// netProbe carries the injected resolver/dial call counters and certificate of
// observed dial destinations for a request.
type netProbe struct {
	mu            sync.Mutex
	resolverCalls int
	dialCalls     int
	dialAddrs     []string
	// resolve returns the DNS answer the injected resolver should hand to
	// resolveAndValidateIP for the given hostname.
	resolve func(hostname string) ([]net.IP, error)
	// routes redirects a requested dial address to a real local listener. It
	// keeps the test hermetic even when production is MUTATED into dialing a
	// hostname (which would otherwise escape to the real resolver), and it is
	// what makes "the request reached the alternate endpoint" observable
	// instead of merely "the request failed".
	routes map[string]string
}

type probeSnapshot struct {
	resolverCalls int
	dialCalls     int
	dialAddrs     []string
}

func (p *netProbe) snapshot() probeSnapshot {
	p.mu.Lock()
	defer p.mu.Unlock()
	return probeSnapshot{
		resolverCalls: p.resolverCalls,
		dialCalls:     p.dialCalls,
		dialAddrs:     append([]string(nil), p.dialAddrs...),
	}
}

// newNetProbeCtx builds an EffContext with injected resolver/dial instrumentation
// (AllowHTTP + AllowLocalhost so local httptest servers are reachable on the
// direct route). The caller supplies probe.resolve before issuing a request.
func newNetProbeCtx(t *testing.T, probe *netProbe) *EffContext {
	t.Helper()
	ctx := NewEffContext(nil)
	ctx.Grant(NewCapability("Net"))
	ctx.Net = NewNetContext()
	ctx.Net.AllowHTTP = true
	ctx.Net.AllowLocalhost = true
	ctx.Net.lookupIP = func(hostname string) ([]net.IP, error) {
		probe.mu.Lock()
		probe.resolverCalls++
		probe.mu.Unlock()
		return probe.resolve(hostname)
	}
	ctx.Net.dialContext = func(c context.Context, network, addr string) (net.Conn, error) {
		probe.mu.Lock()
		probe.dialCalls++
		probe.dialAddrs = append(probe.dialAddrs, addr)
		dest, routed := probe.routes[addr]
		probe.mu.Unlock()
		if routed {
			return (&net.Dialer{}).DialContext(c, network, dest)
		}
		return (&net.Dialer{}).DialContext(c, network, addr)
	}
	return ctx
}

// forceProxy makes the request-aware RoundTripper treat every request as a
// proxy-selected request to proxyURL.
func forceProxy(ctx *EffContext, proxyURL *url.URL) {
	ctx.Net.proxySelector = func(*http.Request) (*url.URL, error) { return proxyURL, nil }
}

// forceNoProxy makes the request-aware RoundTripper treat every request as a
// direct (no-proxy) request.
func forceNoProxy(ctx *EffContext) {
	ctx.Net.proxySelector = func(*http.Request) (*url.URL, error) { return nil, nil }
}

// noProxyFor makes the RoundTripper return nil (direct) only for reqs whose
// host matches hostName (simulating a NO_PROXY bypass) and proxyURL otherwise.
func noProxyOnlyFor(ctx *EffContext, hostName string, proxyURL *url.URL) {
	ctx.Net.proxySelector = func(req *http.Request) (*url.URL, error) {
		if req.URL.Hostname() == hostName {
			return nil, nil
		}
		return proxyURL, nil
	}
}

func argsGet(u string) []eval.Value {
	return []eval.Value{
		&eval.StringValue{Value: "GET"},
		&eval.StringValue{Value: u},
		&eval.ListValue{Elements: []eval.Value{}},
		&eval.StringValue{Value: ""},
	}
}

// unwrapErrCtor extracts the NetError constructor name and message from a
// Result.Err value.
func unwrapErrCtor(t *testing.T, v eval.Value) (ctor, msg string) {
	t.Helper()
	tagged, ok := v.(*eval.TaggedValue)
	if !ok {
		t.Fatalf("expected TaggedValue, got %T", v)
	}
	if tagged.CtorName != "Err" {
		t.Fatalf("expected Result.Err, got %s", tagged.CtorName)
	}
	errVal, ok := tagged.Fields[0].(*eval.TaggedValue)
	if !ok {
		t.Fatalf("expected NetError TaggedValue, got %T", tagged.Fields[0])
	}
	ctor = errVal.CtorName
	if len(errVal.Fields) > 0 {
		if s, ok := errVal.Fields[0].(*eval.StringValue); ok {
			msg = s.Value
		}
	}
	return ctor, msg
}

func unwrapOkBody(t *testing.T, v eval.Value) string {
	t.Helper()
	rec := unwrapOkRecord(t, v)
	return rec.Fields["body"].(*eval.StringValue).Value
}

// observingServer starts a local endpoint that serves body and reports whether
// it was ever reached. "Reached / not reached" is the observation that makes
// "the request went to the right place" falsifiable rather than incidental.
func observingServer(t *testing.T, body string) (srv *httptest.Server, seen func() bool) {
	t.Helper()
	var hit bool
	var mu sync.Mutex
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		hit = true
		mu.Unlock()
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(srv.Close)
	return srv, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return hit
	}
}

// portOf returns the port an httptest server is listening on.
func portOf(srv *httptest.Server) string {
	_, port, _ := net.SplitHostPort(srv.Listener.Addr().String())
	return port
}

// --- TestNetProxyBoundary (AC-M1.1) ---

// TestNetProxyBoundary asserts the proxy boundary is observed by the production
// Net constructors: a proxy-selected request dials the proxy (never the target),
// performs zero local target resolution, and — via the production
// http.ProxyFromEnvironment default in a subprocess — honors the process proxy
// environment.
func TestNetProxyBoundary(t *testing.T) {
	t.Run("proxy_selected_dials_proxy_not_target", func(t *testing.T) {
		proxy, proxySeen := observingServer(t, "proxied-ok")
		pu, err := url.Parse(proxy.URL)
		if err != nil {
			t.Fatalf("parse proxy url: %v", err)
		}

		probe := &netProbe{resolve: func(h string) ([]net.IP, error) {
			return nil, fmt.Errorf("must not resolve proxied target %q", h)
		}}
		ctx := newNetProbeCtx(t, probe)
		forceProxy(ctx, pu)

		// Production structured constructor, proxy target hostname differs from
		// the fake proxy; would fail to resolve if the direct route were taken.
		result, err := NetHTTPRequest(ctx, argsGet("http://target.test/proxy-check"))
		if err != nil {
			t.Fatalf("NetHTTPRequest returned Go error: %v", err)
		}
		if body := unwrapOkBody(t, result); body != "proxied-ok" {
			t.Fatalf("body = %q, want %q (proxy-served)", body, "proxied-ok")
		}
		if !proxySeen() {
			t.Error("fake proxy did not observe the request (production route did not use it)")
		}
		s := probe.snapshot()
		if s.resolverCalls != 0 {
			t.Errorf("resolver calls = %d, want 0 for proxy-selected request", s.resolverCalls)
		}
		if s.dialCalls < 1 {
			t.Fatalf("dial calls = %d, want >= 1", s.dialCalls)
		}
		// Observed dial destination must be the proxy addr, NOT the target host.
		wantDialHost := proxy.Listener.Addr().String()
		if s.dialAddrs[0] != wantDialHost {
			t.Errorf("observed dial destination = %q, want proxy %q (proxy addr must never be rewritten)", s.dialAddrs[0], wantDialHost)
		}
	})

	t.Run("proxy_selected_https_connect_destination", func(t *testing.T) {
		// Minimal CONNECT proxy: records the CONNECT target and responds 200.
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatalf("connect proxy listen: %v", err)
		}
		defer ln.Close()
		var connects []string
		var cmu sync.Mutex
		go func() {
			for {
				conn, err := ln.Accept()
				if err != nil {
					return
				}
				go func(c net.Conn) {
					defer c.Close()
					br := bufio.NewReader(c)
					line, err := br.ReadString('\n')
					if err != nil {
						return
					}
					cmu.Lock()
					connects = append(connects, strings.TrimSpace(line))
					cmu.Unlock()
					_, _ = io.WriteString(c, "HTTP/1.1 200 Connection Established\r\n\r\n")
				}(conn)
			}
		}()

		proxyAddr := ln.Addr().String()
		pu := &url.URL{Scheme: "http", Host: proxyAddr}

		probe := &netProbe{resolve: func(h string) ([]net.IP, error) {
			return nil, fmt.Errorf("must not resolve proxied HTTPS target %q", h)
		}}
		ctx := newNetProbeCtx(t, probe)
		forceProxy(ctx, pu)

		// HTTPS request through the CONNECT proxy; the TLS handshake will fail
		// (no tunnel target), which is expected — we assert the CONNECT/dial
		// destination and zero resolution, not success.
		_, _ = NetHTTPRequest(ctx, argsGet("https://target.test/secure"))

		cmu.Lock()
		cList := append([]string(nil), connects...)
		cmu.Unlock()
		s := probe.snapshot()

		if s.resolverCalls != 0 {
			t.Errorf("resolver calls = %d, want 0 for proxy-selected HTTPS request", s.resolverCalls)
		}
		if s.dialCalls < 1 {
			t.Fatalf("dial calls = %d, want >= 1", s.dialCalls)
		}
		if s.dialAddrs[0] != proxyAddr {
			t.Errorf("observed dial destination = %q, want CONNECT proxy %q", s.dialAddrs[0], proxyAddr)
		}
		foundConnect := false
		for _, line := range cList {
			if strings.HasPrefix(line, "CONNECT target.test:443 ") {
				foundConnect = true
			}
		}
		if !foundConnect {
			t.Errorf("fake CONNECT proxy did not observe CONNECT to target.test:443; saw %q", cList)
		}
	})

	t.Run("proxy_selected_from_environment", func(t *testing.T) {
		// Proves the PRODUCTION default (http.ProxyFromEnvironment) honors a
		// process-level proxy, driven in a subprocess so the env is captured at
		// process start (Go caches proxy config once per process).
		if os.Getenv("AILANG_M1_PROXY_HELPER") == "1" {
			t.Fatal("proxy_selected_from_environment should not run inside the helper subprocess")
		}
		var proxied bool
		var pmu sync.Mutex
		proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			pmu.Lock()
			proxied = true
			pmu.Unlock()
			_, _ = w.Write([]byte(netProxyEnvBody))
		}))
		defer proxy.Close()

		cmd := exec.Command(os.Args[0], "-test.run=^TestNetProxyEnvProxyHelper$", "-test.v")
		env := stripProxyEnv(os.Environ())
		env = append(env,
			"AILANG_M1_PROXY_HELPER=1",
			"HTTP_PROXY="+proxy.URL,
			"HTTPS_PROXY="+proxy.URL,
			"NO_PROXY=",
		)
		cmd.Env = env
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("proxy-env helper subprocess failed: %v\n%s", err, out)
		}
		if !bytes.Contains(out, []byte("--- PASS: TestNetProxyEnvProxyHelper")) {
			t.Fatalf("proxy-env helper did not report PASS:\n%s", out)
		}
		pmu.Lock()
		observed := proxied
		pmu.Unlock()
		if !observed {
			t.Error("fake proxy did not observe the request through the production http.ProxyFromEnvironment path")
		}
	})
}

// TestNetProxyEnvProxyHelper is a subprocess helper: it runs the production
// default proxy selector (http.ProxyFromEnvironment) against the parent's fake
// proxy (passed via HTTP_PROXY/HTTPS_PROXY at process start). It asserts the
// proxy route is taken and that zero local resolution happens.
func TestNetProxyEnvProxyHelper(t *testing.T) {
	if os.Getenv("AILANG_M1_PROXY_HELPER") != "1" {
		t.Skip("proxy-boundary environment helper (subprocess only)")
	}
	probe := &netProbe{resolve: func(h string) ([]net.IP, error) {
		return nil, fmt.Errorf("no such host: %s", h)
	}}
	ctx := newNetProbeCtx(t, probe)
	// NB: proxySelector intentionally left nil — the production
	// http.ProxyFromEnvironment must route via the process-level proxy.
	result, err := NetHTTPRequest(ctx, argsGet("http://target.test/env-check"))
	if err != nil {
		t.Fatalf("NetHTTPRequest returned Go error: %v", err)
	}
	if body := unwrapOkBody(t, result); body != netProxyEnvBody {
		t.Fatalf("body = %q, want %q (proxy-served)", body, netProxyEnvBody)
	}
	if s := probe.snapshot(); s.resolverCalls != 0 {
		t.Errorf("resolver calls = %d, want 0 for env-selected proxy route", s.resolverCalls)
	}
}

// --- TestNetProxyNoProxy (AC-M1.1 / direct route) ---

// TestNetProxyNoProxy asserts the NO_PROXY case: a proxy IS configured for the
// world, but the target is bypassed, so the request must take the DIRECT pinned
// route. The fake proxy is a real listener, so "the proxy observed nothing" is
// an observation rather than an assumption; the alternate endpoint is a real
// listener too, so "not the hostname endpoint" is likewise observable.
func TestNetProxyNoProxy(t *testing.T) {
	proxy, proxySeen := observingServer(t, "proxy-served")
	pu, err := url.Parse(proxy.URL)
	if err != nil {
		t.Fatalf("parse proxy url: %v", err)
	}
	alternate, altSeen := observingServer(t, "alternate")
	server, _ := observingServer(t, "direct-ok")
	port := portOf(server)

	probe := &netProbe{
		resolve: func(h string) ([]net.IP, error) {
			return []net.IP{net.ParseIP("127.0.0.1")}, nil
		},
		// A dial to the hostname (which production must never emit) lands on the
		// alternate endpoint, keeping the test hermetic under mutation.
		routes: map[string]string{
			net.JoinHostPort("pinned.test", port): alternate.Listener.Addr().String(),
		},
	}
	ctx := newNetProbeCtx(t, probe)
	// HTTP_PROXY-equivalent for everything except the NO_PROXY-matched target.
	noProxyOnlyFor(ctx, "pinned.test", pu)

	result, err := NetHTTPRequest(ctx, argsGet("http://pinned.test:"+port+"/direct"))
	if err != nil {
		t.Fatalf("NetHTTPRequest returned Go error: %v", err)
	}
	if body := unwrapOkBody(t, result); body != "direct-ok" {
		t.Fatalf("body = %q, want %q (the pinned direct endpoint)", body, "direct-ok")
	}
	s := probe.snapshot()
	if s.resolverCalls != 1 {
		t.Errorf("resolver calls = %d, want exactly 1 for direct route", s.resolverCalls)
	}
	if s.dialCalls != 1 {
		t.Fatalf("dial calls = %d, want exactly 1, saw %v", s.dialCalls, s.dialAddrs)
	}
	wantDial := net.JoinHostPort("127.0.0.1", port)
	if s.dialAddrs[0] != wantDial {
		t.Errorf("observed dial destination = %q, want validated IP %q (hostname must not be re-resolved)", s.dialAddrs[0], wantDial)
	}
	if proxySeen() {
		t.Error("the fake proxy observed the request; the NO_PROXY-matched target must take the direct route")
	}
	if altSeen() {
		t.Error("the alternate (hostname) endpoint received the request; pinning failed")
	}
}

// --- TestNetProxyDirectPin (AC-M1.2) ---

// TestNetProxyDirectPin asserts a direct/NO_PROXY route pins the accepted
// connection to the injected validated IP — not the request hostname's
// alternate endpoint — even when a proxy is configured for other hosts.
func TestNetProxyDirectPin(t *testing.T) {
	// The "alternate endpoint" server that must NOT receive the request.
	alternate, altSeen := observingServer(t, "alternate")
	// The validated-IP server the request SHOULD reach.
	target, _ := observingServer(t, "pinned-ok")
	port := portOf(target)

	// hostRoute makes a dial to the *hostname* land on the alternate endpoint.
	// Production must never produce such a dial: it must dial the validated IP.
	// Without this route a mutated production would escape to the real resolver
	// (non-hermetic) and the alternate-endpoint assertion would be vacuous.
	hostRoute := map[string]string{
		net.JoinHostPort("pinned.test", port): alternate.Listener.Addr().String(),
	}

	t.Run("no_proxy_pin_reaches_validated_ip", func(t *testing.T) {
		// A proxy is configured for the world, but NO_PROXY bypasses
		// "pinned.test", so the direct pinned route must win.
		proxyURL, _ := url.Parse("http://proxy.invalid:3128")
		probe := &netProbe{
			resolve: func(h string) ([]net.IP, error) {
				return []net.IP{net.ParseIP("127.0.0.1")}, nil
			},
			routes: hostRoute,
		}
		ctx := newNetProbeCtx(t, probe)
		noProxyOnlyFor(ctx, "pinned.test", proxyURL)

		result, err := NetHTTPRequest(ctx, argsGet("http://pinned.test:"+port+"/pinned"))
		if err != nil {
			t.Fatalf("NetHTTPRequest returned Go error: %v", err)
		}
		if body := unwrapOkBody(t, result); body != "pinned-ok" {
			t.Fatalf("body = %q, want %q", body, "pinned-ok")
		}
		s := probe.snapshot()
		if s.resolverCalls != 1 {
			t.Errorf("resolver calls = %d, want exactly 1 (direct+NO_PROXY resolves once)", s.resolverCalls)
		}
		if len(s.dialAddrs) == 0 {
			t.Fatal("no dial observed")
		}
		wantDial := net.JoinHostPort("127.0.0.1", port)
		if s.dialAddrs[0] != wantDial {
			t.Errorf("observed dial destination = %q, want validated IP %q (NOT the request hostname alternate endpoint)", s.dialAddrs[0], wantDial)
		}
		if altSeen() {
			t.Error("the alternate (hostname) endpoint received the request; pinning failed")
		}
	})

	t.Run("direct_pin_reaches_validated_ip", func(t *testing.T) {
		probe := &netProbe{
			resolve: func(h string) ([]net.IP, error) {
				return []net.IP{net.ParseIP("127.0.0.1")}, nil
			},
			routes: hostRoute,
		}
		ctx := newNetProbeCtx(t, probe)
		forceNoProxy(ctx)

		result, err := NetHTTPRequest(ctx, argsGet("http://pinned.test:"+port+"/pinned"))
		if err != nil {
			t.Fatalf("NetHTTPRequest returned Go error: %v", err)
		}
		if body := unwrapOkBody(t, result); body != "pinned-ok" {
			t.Fatalf("body = %q, want %q (the pinned/validated-IP endpoint)", body, "pinned-ok")
		}
		s := probe.snapshot()
		if s.dialCalls != 1 {
			t.Fatalf("dial calls = %d, want exactly 1", s.dialCalls)
		}
		if s.dialAddrs[0] != net.JoinHostPort("127.0.0.1", port) {
			t.Errorf("observed dial destination = %q, want validated IP", s.dialAddrs[0])
		}
		if s.resolverCalls != 1 {
			t.Errorf("resolver calls = %d, want exactly 1", s.resolverCalls)
		}
		if altSeen() {
			t.Error("the alternate (hostname) endpoint received the request; pinning failed")
		}
	})

	// --- AC-M1.4: direct-route failure shape -------------------------------
	//
	// These live inside TestNetProxyDirectPin (rather than a separate top-level
	// test) so that the graded AC-M1.1 -run filter actually executes them.
	// Direct-route DNS/IP rejection must perform exactly 1 resolver call, exactly
	// 0 dials, and surface the original category text — NOT the *url.Error
	// wrapper that http.Client.Do adds around it.

	t.Run("AC-M1.4_legacy_dns_failure_exact_category", func(t *testing.T) {
		probe := &netProbe{resolve: func(h string) ([]net.IP, error) {
			return nil, errInjectedDNS
		}}
		ctx := newNetProbeCtx(t, probe)
		forceNoProxy(ctx)

		_, err := netHTTPGet(ctx, []eval.Value{&eval.StringValue{Value: "http://dnsfail.test/x"}})
		if err == nil {
			t.Fatal("expected E_NET_DNS_FAILED error, got nil")
		}
		// Exact, not Contains: a Contains check would still pass if the
		// *url.Error wrapper leaked through unwrapTargetValidation.
		if want := "E_NET_DNS_FAILED: " + errInjectedDNS.Error(); err.Error() != want {
			t.Errorf("legacy error = %q, want exactly %q (unwrapTargetValidation must strip *url.Error)", err.Error(), want)
		}
		assertOneResolveNoDial(t, probe)
	})

	t.Run("AC-M1.4_legacy_ip_blocked_exact_category", func(t *testing.T) {
		probe := &netProbe{resolve: func(h string) ([]net.IP, error) {
			return []net.IP{net.ParseIP("192.168.1.5")}, nil // private, always blocked
		}}
		ctx := newNetProbeCtx(t, probe)
		forceNoProxy(ctx)

		_, err := netHTTPGet(ctx, []eval.Value{&eval.StringValue{Value: "http://blocked.test/x"}})
		if err == nil {
			t.Fatal("expected E_NET_IP_BLOCKED error, got nil")
		}
		assertRawCategory(t, err.Error(), "E_NET_IP_BLOCKED")
		assertOneResolveNoDial(t, probe)
	})

	t.Run("AC-M1.4_structured_dns_failure_transport_category", func(t *testing.T) {
		probe := &netProbe{resolve: func(h string) ([]net.IP, error) {
			return nil, errInjectedDNS
		}}
		ctx := newNetProbeCtx(t, probe)
		forceNoProxy(ctx)

		result, err := NetHTTPRequest(ctx, argsGet("http://dnsfail.test/x"))
		if err != nil {
			t.Fatalf("NetHTTPRequest returned Go error: %v", err)
		}
		ctor, msg := unwrapErrCtor(t, result)
		if ctor != "Transport" {
			t.Errorf("expected Err(Transport), got Err(%s)", ctor)
		}
		if want := "E_NET_DNS_FAILED: " + errInjectedDNS.Error(); msg != want {
			t.Errorf("Transport message = %q, want exactly %q (transportMessage must strip *url.Error)", msg, want)
		}
		assertOneResolveNoDial(t, probe)
	})

	t.Run("AC-M1.4_structured_ip_blocked_transport_category", func(t *testing.T) {
		probe := &netProbe{resolve: func(h string) ([]net.IP, error) {
			return []net.IP{net.ParseIP("10.0.0.1")}, nil // private, always blocked
		}}
		ctx := newNetProbeCtx(t, probe)
		forceNoProxy(ctx)

		result, err := NetHTTPRequest(ctx, argsGet("http://blocked.test/x"))
		if err != nil {
			t.Fatalf("NetHTTPRequest returned Go error: %v", err)
		}
		ctor, msg := unwrapErrCtor(t, result)
		if ctor != "Transport" {
			t.Errorf("expected Err(Transport), got Err(%s)", ctor)
		}
		assertRawCategory(t, msg, "E_NET_IP_BLOCKED")
		assertOneResolveNoDial(t, probe)
	})
}

// errInjectedDNS is the resolver failure injected by the AC-M1.4 DNS arms. It is
// a fixed value so the surfaced message can be asserted byte-for-byte.
var errInjectedDNS = errors.New("injected dns failure")

// assertOneResolveNoDial pins AC-M1.4's call-count half: the direct route must
// resolve exactly once and must never open a socket for a rejected target.
func assertOneResolveNoDial(t *testing.T, probe *netProbe) {
	t.Helper()
	s := probe.snapshot()
	if s.resolverCalls != 1 {
		t.Errorf("resolver calls = %d, want exactly 1", s.resolverCalls)
	}
	if s.dialCalls != 0 {
		t.Errorf("dial calls = %d, want exactly 0 (no socket before validation), saw %v", s.dialCalls, s.dialAddrs)
	}
}

// assertRawCategory checks a surfaced message carries the stable category AND
// has not been re-wrapped by http.Client.Do's *url.Error (which would prefix it
// with `Get "…": `). Contains() alone cannot tell those two apart.
func assertRawCategory(t *testing.T, msg, category string) {
	t.Helper()
	if !strings.Contains(msg, category) {
		t.Errorf("message %q does not carry the %s category", msg, category)
	}
	if strings.Contains(msg, `Get "`) || strings.Contains(msg, `Post "`) {
		t.Errorf("message %q leaked the *url.Error wrapper; the target-validation unwrapping regressed", msg)
	}
}

// --- TestNetProxyRedirectControls (AC-M1.3) ---

// TestNetProxyRedirectControls asserts the pre-transport security controls
// (capability, initial-domain) and the redirect controls (protocol, count) all
// still fire when a proxy is selected, and that the injected resolver counter
// stays exactly 0 for proxy-selected initial AND redirect requests.
func TestNetProxyRedirectControls(t *testing.T) {
	// fakeProxy returns resp for every absolute-form request it observes.
	fakeProxy := func(resp func(w http.ResponseWriter, r *http.Request)) (*httptest.Server, *url.URL, func() bool) {
		var seen bool
		var mu sync.Mutex
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			seen = true
			mu.Unlock()
			resp(w, r)
		}))
		pu, _ := url.Parse(srv.URL)
		seenFn := func() bool {
			mu.Lock()
			defer mu.Unlock()
			return seen
		}
		return srv, pu, seenFn
	}

	t.Run("capability_denial_proxy_selected", func(t *testing.T) {
		srv, pu, _ := fakeProxy(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
		})
		defer srv.Close()

		probe := &netProbe{resolve: func(h string) ([]net.IP, error) { return nil, fmt.Errorf("no") }}
		ctx := newNetProbeCtx(t, probe)
		forceProxy(ctx, pu)
		// Remove the Net grant to force capability denial.
		ctx.Caps = map[string]Capability{}

		urlVal := &eval.StringValue{Value: "http://target.test/start"}
		_, err := netHTTPGet(ctx, []eval.Value{urlVal})
		if err == nil {
			t.Fatal("expected capability error, got nil")
		}
		if _, ok := err.(*CapabilityError); !ok {
			t.Fatalf("expected CapabilityError, got %T", err)
		}
		if s := probe.snapshot(); s.resolverCalls != 0 {
			t.Errorf("resolver calls = %d, want 0 (capability check precedes transport)", s.resolverCalls)
		}
	})

	t.Run("initial_domain_rejection_proxy_selected", func(t *testing.T) {
		srv, pu, _ := fakeProxy(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
		})
		defer srv.Close()

		probe := &netProbe{resolve: func(h string) ([]net.IP, error) { return nil, fmt.Errorf("no") }}
		ctx := newNetProbeCtx(t, probe)
		forceProxy(ctx, pu)
		ctx.Net.AllowedDomains = []string{"allowed.test"}

		// Domain allowlist rejects before any transport / proxy contact.
		result, err := NetHTTPRequest(ctx, argsGet("http://evil.test/x"))
		if err != nil {
			t.Fatalf("NetHTTPRequest returned Go error: %v", err)
		}
		if ctor, msg := unwrapErrCtor(t, result); ctor != "DisallowedHost" || !strings.Contains(msg, "evil.test") {
			t.Errorf("expected Err(DisallowedHost(evil.test)), got Err(%s(%q))", ctor, msg)
		}
		if s := probe.snapshot(); s.resolverCalls != 0 {
			t.Errorf("resolver calls = %d, want 0 (allowlist precedes transport)", s.resolverCalls)
		}
	})

	t.Run("redirect_protocol_rejection_proxy_selected", func(t *testing.T) {
		// Proxy serves a redirect to a blocked protocol. validateRedirect must
		// reject without resolving, and the proxy route never resolves either.
		srv, pu, seen := fakeProxy(func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "file:///etc/passwd", http.StatusFound)
		})
		defer srv.Close()

		probe := &netProbe{resolve: func(h string) ([]net.IP, error) { return nil, fmt.Errorf("no") }}
		ctx := newNetProbeCtx(t, probe)
		forceProxy(ctx, pu)

		urlVal := &eval.StringValue{Value: "http://target.test/start"}
		got, err := netHTTPGet(ctx, []eval.Value{urlVal})
		if err == nil {
			t.Fatalf("expected protocol-blocked error, got result %v", got)
		}
		if !strings.Contains(err.Error(), "E_NET_PROTOCOL_BLOCKED") {
			t.Errorf("expected E_NET_PROTOCOL_BLOCKED in %q", err.Error())
		}
		if !seen() {
			t.Error("fake proxy did not observe the initial request; redirect-control ran in the wrong mode")
		}
		if s := probe.snapshot(); s.resolverCalls != 0 {
			t.Errorf("resolver calls = %d, want 0 for proxy initial AND redirect protocol rejection", s.resolverCalls)
		}
	})

	t.Run("redirect_count_rejection_proxy_selected", func(t *testing.T) {
		srv, pu, seen := fakeProxy(func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/loop", http.StatusFound)
		})
		defer srv.Close()

		probe := &netProbe{resolve: func(h string) ([]net.IP, error) { return nil, fmt.Errorf("no") }}
		ctx := newNetProbeCtx(t, probe)
		forceProxy(ctx, pu)
		ctx.Net.MaxRedirects = 1

		urlVal := &eval.StringValue{Value: "http://target.test/start"}
		got, err := netHTTPGet(ctx, []eval.Value{urlVal})
		if err == nil {
			t.Fatalf("expected too-many-redirects error, got result %v", got)
		}
		if !strings.Contains(err.Error(), "E_NET_TOO_MANY_REDIRECTS") {
			t.Errorf("expected E_NET_TOO_MANY_REDIRECTS in %q", err.Error())
		}
		if !seen() {
			t.Error("fake proxy did not observe the request; redirect-count control ran in the wrong mode")
		}
		if s := probe.snapshot(); s.resolverCalls != 0 {
			t.Errorf("resolver calls = %d, want 0 for proxy redirect-count rejection", s.resolverCalls)
		}
	})
}

// stripProxyEnv removes all proxy/no-proxy variables (any case) from an
// environment slice so a subprocess starts with a clean egress posture.
func stripProxyEnv(base []string) []string {
	out := make([]string, 0, len(base))
	for _, kv := range base {
		name := kv
		if i := strings.IndexByte(kv, '='); i >= 0 {
			name = kv[:i]
		}
		switch strings.ToUpper(name) {
		case "HTTP_PROXY", "HTTPS_PROXY", "ALL_PROXY", "NO_PROXY":
			continue
		default:
			out = append(out, kv)
		}
	}
	return out
}
