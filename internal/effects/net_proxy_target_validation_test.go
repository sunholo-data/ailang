package effects

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
)

// forceProxyCounted is forceProxy plus a selection counter. The counter is what
// makes an arm DISCRIMINATE the proxy route: a blocked literal IP is refused by
// the DIRECT route too (resolveAndValidateIP's raw-IP branch), with the same
// error text and the same zero resolver/dial counts — so without this, an arm
// named for the proxy route passes identically when the proxy is never selected,
// and survives having its own precondition removed. Found by the iteration-235
// evaluator's precondition-neutering drill.
func forceProxyCounted(ctx *EffContext, proxyURL *url.URL, calls *int32) {
	ctx.Net.proxySelector = func(*http.Request) (*url.URL, error) {
		atomic.AddInt32(calls, 1)
		return proxyURL, nil
	}
}

// installProbeResponseDial replaces the probe dialer with a socket-free
// net.Pipe responder while preserving dial call/address instrumentation.
func installProbeResponseDial(ctx *EffContext, probe *netProbe, body string) {
	ctx.Net.dialContext = func(_ context.Context, _ string, stringAddr string) (net.Conn, error) {
		probe.mu.Lock()
		probe.dialCalls++
		probe.dialAddrs = append(probe.dialAddrs, stringAddr)
		probe.mu.Unlock()
		clientConn, serverConn := net.Pipe()
		go func() {
			defer serverConn.Close()
			if req, err := http.ReadRequest(bufio.NewReader(serverConn)); err == nil {
				_ = req.Body.Close()
			}
			_, _ = io.WriteString(serverConn, "HTTP/1.1 200 OK\r\nContent-Length: "+fmt.Sprint(len(body))+"\r\nConnection: close\r\n\r\n"+body)
		}()
		return clientConn, nil
	}
}

func TestNetProxyTargetValidation(t *testing.T) {
	t.Run("proxy_literal_blocked_before_dial", func(t *testing.T) {
		proxyURL, err := url.Parse("http://127.0.0.1:3128")
		if err != nil {
			t.Fatalf("parse proxy URL: %v", err)
		}
		probe := &netProbe{resolve: func(string) ([]net.IP, error) {
			t.Fatal("proxy-selected literal IP must not use the resolver")
			return nil, nil
		}}
		ctx := newNetProbeCtx(t, probe)
		var proxySelections int32
		forceProxyCounted(ctx, proxyURL, &proxySelections)

		result, err := NetHTTPRequest(ctx, argsGet("http://10.0.0.1/x"))
		if err != nil {
			t.Fatalf("NetHTTPRequest returned Go error: %v", err)
		}
		if got := atomic.LoadInt32(&proxySelections); got != 1 {
			t.Fatalf("proxy selections = %d, want 1 — this arm must exercise the PROXY route, "+
				"not the direct route, which refuses the same literal for a different reason", got)
		}
		ctor, msg := unwrapErrCtor(t, result)
		if ctor != "Transport" || !strings.Contains(msg, "E_NET_IP_BLOCKED") {
			t.Fatalf("expected Err(Transport(E_NET_IP_BLOCKED...)), got Err(%s(%q))", ctor, msg)
		}
		s := probe.snapshot()
		if s.resolverCalls != 0 {
			t.Errorf("resolver calls = %d, want 0", s.resolverCalls)
		}
		if s.dialCalls != 0 {
			t.Errorf("dial calls = %d, want 0 before refusal; saw %v", s.dialCalls, s.dialAddrs)
		}
	})

	t.Run("proxy_literal_public_dials_proxy", func(t *testing.T) {
		proxyURL, err := url.Parse("http://proxy.test:3128")
		if err != nil {
			t.Fatalf("parse proxy URL: %v", err)
		}
		probe := &netProbe{resolve: func(string) ([]net.IP, error) {
			t.Fatal("proxy-selected literal IP must not use the resolver")
			return nil, nil
		}}
		ctx := newNetProbeCtx(t, probe)
		installProbeResponseDial(ctx, probe, "proxied-public-ip")
		forceProxy(ctx, proxyURL)

		result, err := NetHTTPRequest(ctx, argsGet("http://93.184.216.34/x"))
		if err != nil {
			t.Fatalf("NetHTTPRequest returned Go error: %v", err)
		}
		if body := unwrapOkBody(t, result); body != "proxied-public-ip" {
			t.Fatalf("body = %q, want proxy response", body)
		}
		s := probe.snapshot()
		if s.resolverCalls != 0 {
			t.Errorf("resolver calls = %d, want 0", s.resolverCalls)
		}
		if s.dialCalls != 1 {
			t.Fatalf("dial calls = %d, want 1; saw %v", s.dialCalls, s.dialAddrs)
		}
		if want := proxyURL.Host; s.dialAddrs[0] != want {
			t.Errorf("dial address = %q, want proxy %q (not target)", s.dialAddrs[0], want)
		}
	})

	t.Run("proxy_hostname_remains_unresolved", func(t *testing.T) {
		proxyURL, err := url.Parse("http://proxy.test:3128")
		if err != nil {
			t.Fatalf("parse proxy URL: %v", err)
		}
		probe := &netProbe{resolve: func(string) ([]net.IP, error) {
			t.Fatal("proxy-selected hostname must remain unresolved locally")
			return nil, nil
		}}
		ctx := newNetProbeCtx(t, probe)
		installProbeResponseDial(ctx, probe, "proxied-hostname")
		forceProxy(ctx, proxyURL)

		result, err := NetHTTPRequest(ctx, argsGet("http://target.test/x"))
		if err != nil {
			t.Fatalf("NetHTTPRequest returned Go error: %v", err)
		}
		if body := unwrapOkBody(t, result); body != "proxied-hostname" {
			t.Fatalf("body = %q, want proxy response", body)
		}
		s := probe.snapshot()
		if s.resolverCalls != 0 {
			t.Errorf("resolver calls = %d, want 0", s.resolverCalls)
		}
		if s.dialCalls != 1 {
			t.Fatalf("dial calls = %d, want 1; saw %v", s.dialCalls, s.dialAddrs)
		}
		if want := proxyURL.Host; s.dialAddrs[0] != want {
			t.Errorf("dial address = %q, want proxy %q", s.dialAddrs[0], want)
		}
	})

	t.Run("direct_hostname_still_resolves_once", func(t *testing.T) {
		probe := &netProbe{
			resolve: func(hostname string) ([]net.IP, error) {
				if hostname != "direct.test" {
					t.Fatalf("resolved hostname = %q, want direct.test", hostname)
				}
				return []net.IP{net.ParseIP("93.184.216.34")}, nil
			},
		}
		ctx := newNetProbeCtx(t, probe)
		installProbeResponseDial(ctx, probe, "direct-origin")
		forceNoProxy(ctx)

		result, err := NetHTTPRequest(ctx, argsGet("http://direct.test:8080/x"))
		if err != nil {
			t.Fatalf("NetHTTPRequest returned Go error: %v", err)
		}
		if body := unwrapOkBody(t, result); body != "direct-origin" {
			t.Fatalf("body = %q, want direct response", body)
		}
		if s := probe.snapshot(); s.resolverCalls != 1 {
			t.Errorf("resolver calls = %d, want 1 for direct route", s.resolverCalls)
		}
	})

	// Regression arm for the iteration-235 evaluator finding: url.Hostname() keeps
	// the RFC 4007 zone ("fe80::1%eth0") and net.ParseIP returns nil for that form,
	// so before literalHost() this link-local target reached the proxy UNVALIDATED.
	t.Run("proxy_zone_qualified_literal_blocked", func(t *testing.T) {
		proxyURL, err := url.Parse("http://proxy.test:3128")
		if err != nil {
			t.Fatalf("parse proxy URL: %v", err)
		}
		probe := &netProbe{resolve: func(string) ([]net.IP, error) {
			t.Fatal("zone-qualified literal must not reach the resolver")
			return nil, nil
		}}
		ctx := newNetProbeCtx(t, probe)
		var proxySelections int32
		forceProxyCounted(ctx, proxyURL, &proxySelections)

		result, err := NetHTTPRequest(ctx, argsGet("http://[fe80::1%25eth0]/x"))
		if err != nil {
			t.Fatalf("NetHTTPRequest returned Go error: %v", err)
		}
		if got := atomic.LoadInt32(&proxySelections); got != 1 {
			t.Fatalf("proxy selections = %d, want 1 (arm must exercise the proxy route)", got)
		}
		ctor, msg := unwrapErrCtor(t, result)
		if ctor != "Transport" || !strings.Contains(msg, "E_NET_IP_BLOCKED") {
			t.Fatalf("expected Err(Transport(E_NET_IP_BLOCKED...)) for fe80::1%%eth0, got Err(%s(%q))", ctor, msg)
		}
		s := probe.snapshot()
		if s.resolverCalls != 0 {
			t.Errorf("resolver calls = %d, want 0", s.resolverCalls)
		}
		if s.dialCalls != 0 {
			t.Errorf("dial calls = %d, want 0 before refusal; saw %v", s.dialCalls, s.dialAddrs)
		}
	})
}
