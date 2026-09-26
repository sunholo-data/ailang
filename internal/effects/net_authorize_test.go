package effects

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/sunholo-data/ailang/internal/eval"
)

// M-EXECUTOR-POLICY-HARDENING M2 — the destination authorizer table (AC4):
// every Net and Stream entrypoint against resolved private IPs, IP literal
// forms, TLS SNI, and the restricted-mode proxy refusal.

// entrypoint is one network operation and how to invoke it with a URL.
type entrypoint struct {
	name   string
	stream bool
	call   func(ctx *EffContext, u string) (eval.Value, error)
}

var netEntrypoints = []entrypoint{
	{"httpGet", false, func(ctx *EffContext, u string) (eval.Value, error) {
		return Call(ctx, "Net", "httpGet", one(u))
	}},
	{"httpPost", false, func(ctx *EffContext, u string) (eval.Value, error) {
		return Call(ctx, "Net", "httpPost", []eval.Value{&eval.StringValue{Value: u}, &eval.StringValue{Value: "{}"}})
	}},
	{"httpRequest", false, func(ctx *EffContext, u string) (eval.Value, error) {
		return NetHTTPRequest(ctx, []eval.Value{&eval.StringValue{Value: "GET"}, &eval.StringValue{Value: u}, &eval.ListValue{}, &eval.StringValue{Value: ""}})
	}},
	{"httpRequestBytes", false, func(ctx *EffContext, u string) (eval.Value, error) {
		return NetHTTPRequestBytes(ctx, []eval.Value{&eval.StringValue{Value: "GET"}, &eval.StringValue{Value: u}, &eval.ListValue{}, &eval.BytesValue{}})
	}},
	{"sseConnect", true, func(ctx *EffContext, u string) (eval.Value, error) {
		return StreamSSEConnect(ctx, []eval.Value{&eval.StringValue{Value: u}, &eval.RecordValue{Fields: map[string]eval.Value{}}})
	}},
	{"ssePost", true, func(ctx *EffContext, u string) (eval.Value, error) {
		return StreamSSEPost(ctx, []eval.Value{&eval.StringValue{Value: u}, &eval.StringValue{Value: "{}"}, &eval.RecordValue{Fields: map[string]eval.Value{}}})
	}},
	{"ndjsonPost", true, func(ctx *EffContext, u string) (eval.Value, error) {
		return StreamNDJSONPost(ctx, []eval.Value{&eval.StringValue{Value: u}, &eval.StringValue{Value: "{}"}, &eval.RecordValue{Fields: map[string]eval.Value{}}})
	}},
	{"wsConnect", true, func(ctx *EffContext, u string) (eval.Value, error) {
		return StreamConnect(ctx, []eval.Value{&eval.StringValue{Value: u}, &eval.RecordValue{Fields: map[string]eval.Value{}}})
	}},
}

// hookCtx builds a Net or Stream context with an injected resolver and a
// dial counter. dialTo, when set, routes every dial to that address.
func hookCtx(t *testing.T, stream bool, resolve func(string) ([]net.IP, error), dialTo string, dials *atomic.Int32) *EffContext {
	t.Helper()
	ctx := NewEffContext(nil)
	dial := func(c context.Context, network, addr string) (net.Conn, error) {
		dials.Add(1)
		if dialTo != "" {
			addr = dialTo
		}
		return (&net.Dialer{}).DialContext(c, network, addr)
	}
	if stream {
		ctx.Grant(NewCapability("Stream"))
		ctx.Stream = NewStreamContext()
		ctx.Stream.AllowHTTP = true
		ctx.Stream.lookupIP = resolve
		ctx.Stream.dialContext = dial
		t.Cleanup(ctx.Stream.CloseAll)
		// wsConnect needs a registered transport; a fake that records the dial
		// through the supplied dialer is enough for these tables.
		RegisterStreamTransport("ws", func(cfg StreamDialConfig) (StreamTransport, error) {
			if cfg.DialContext == nil {
				t.Fatal("core handed the transport no dialer")
			}
			conn, err := cfg.DialContext(context.Background(), "tcp", "ignored:1")
			if err != nil {
				return nil, err
			}
			conn.Close()
			return nil, context.Canceled // the dial happened; refuse the rest
		})
		t.Cleanup(func() { RegisterStreamTransport("ws", nil) })
	} else {
		ctx.Grant(NewCapability("Net"))
		ctx.Net = NewNetContext()
		ctx.Net.AllowHTTP = true
		ctx.Net.lookupIP = resolve
		ctx.Net.dialContext = dial
		forceNoProxy(ctx)
	}
	return ctx
}

// denied reports whether result/err is a refusal (Go error or Err variant).
func denied(result eval.Value, err error) bool {
	if err != nil {
		return true
	}
	tv, ok := result.(*eval.TaggedValue)
	return ok && tv.CtorName == "Err"
}

func TestDestinationPolicy_ResolvedPrivateIP_NoDial(t *testing.T) {
	for _, ep := range netEntrypoints {
		for _, ip := range []string{"10.0.0.1", "192.168.1.1", "172.16.0.1", "169.254.169.254", "127.0.0.1", "::1", "fd00::1", "fe80::1", "0.0.0.0", "::ffff:10.0.0.1"} {
			t.Run(ep.name+"/"+ip, func(t *testing.T) {
				var dials atomic.Int32
				ctx := hookCtx(t, ep.stream, func(string) ([]net.IP, error) { return []net.IP{net.ParseIP(ip)}, nil }, "", &dials)
				scheme := "http"
				if ep.name == "wsConnect" {
					scheme = "ws"
				}
				res, err := ep.call(ctx, scheme+"://public.example/")
				if !denied(res, err) {
					t.Errorf("%s resolving to %s must be denied, got %v %v", ep.name, ip, res, err)
				}
				if dials.Load() != 0 {
					t.Errorf("%s dialed %d times for a host resolving to %s", ep.name, dials.Load(), ip)
				}
			})
		}
	}
}

func TestDestinationPolicy_LiteralForms_NoDial(t *testing.T) {
	for _, ep := range netEntrypoints {
		for _, host := range []string{"10.0.0.1", "[::ffff:10.0.0.1]", "[fd00::1]", "169.254.169.254", "[fe80::1%25eth0]", "0.0.0.0", "[::]", "224.0.0.1"} {
			t.Run(ep.name+"/"+host, func(t *testing.T) {
				var dials atomic.Int32
				ctx := hookCtx(t, ep.stream, func(string) ([]net.IP, error) { t.Fatal("a literal must not be resolved"); return nil, nil }, "", &dials)
				scheme := "http"
				if ep.name == "wsConnect" {
					scheme = "ws"
				}
				res, err := ep.call(ctx, scheme+"://"+host+"/")
				if !denied(res, err) {
					t.Errorf("%s to literal %s must be denied, got %v %v", ep.name, host, res, err)
				}
				if dials.Load() != 0 {
					t.Errorf("%s dialed for literal %s", ep.name, host)
				}
			})
		}
	}
}

func TestDestinationPolicy_AllowedAndDeniedHost(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: ok\n\n"))
	}))
	defer srv.Close()
	for _, ep := range netEntrypoints {
		if ep.name == "wsConnect" {
			continue // the fake transport refuses after dialing; HTTP shapes only here
		}
		t.Run(ep.name, func(t *testing.T) {
			var dials atomic.Int32
			pub := func(string) ([]net.IP, error) { return []net.IP{net.ParseIP("203.0.113.10")}, nil }
			ctx := hookCtx(t, ep.stream, pub, strings.TrimPrefix(srv.URL, "http://"), &dials)
			if ep.stream {
				ctx.Stream.AllowedDomains = []string{"allowed.example"}
			} else {
				ctx.Net.AllowedDomains = []string{"allowed.example"}
			}
			res, err := ep.call(ctx, "http://denied.example/")
			if !denied(res, err) || dials.Load() != 0 {
				t.Errorf("denied host: got %v %v, dials=%d", res, err, dials.Load())
			}
			res, err = ep.call(ctx, "http://allowed.example/")
			if denied(res, err) {
				t.Errorf("allowed host must succeed: %v %v", res, err)
			}
			if dials.Load() != 1 {
				t.Errorf("allowed host must dial exactly once, got %d", dials.Load())
			}
		})
	}
}

// TLS: the pinned dial connects to the validated IP while the handshake
// verifies the certificate against the URL's hostname (SNI = the name).
func TestDestinationPolicy_TLSServerNameIsHostname(t *testing.T) {
	var sni atomic.Value
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.TLS != nil {
			sni.Store(r.TLS.ServerName)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: ok\n\n"))
	}))
	srv.StartTLS()
	defer srv.Close()
	pool := x509.NewCertPool()
	pool.AddCert(srv.Certificate())
	// httptest's certificate is valid for example.com; resolve it to a public
	// IP that the dialer routes to the listener.
	for _, ep := range netEntrypoints {
		if ep.name == "wsConnect" {
			continue
		}
		t.Run(ep.name, func(t *testing.T) {
			var dials atomic.Int32
			ctx := hookCtx(t, ep.stream, func(string) ([]net.IP, error) { return []net.IP{net.ParseIP("203.0.113.10")}, nil }, strings.TrimPrefix(srv.URL, "https://"), &dials)
			if ep.stream {
				ctx.Stream.tlsClientConfig = &tls.Config{RootCAs: pool}
			} else {
				ctx.Net.tlsClientConfig = &tls.Config{RootCAs: pool}
			}
			res, err := ep.call(ctx, "https://example.com/")
			if denied(res, err) {
				t.Fatalf("TLS request must succeed against the trusted test cert: %v %v", res, err)
			}
			if got, _ := sni.Load().(string); got != "example.com" {
				t.Errorf("server saw SNI %q, want example.com (the pinned IP must not leak into the handshake)", got)
			}
		})
	}
}

func TestDestinationPolicy_RefuseProxy_NoFallback(t *testing.T) {
	proxyURL, _ := url.Parse("http://127.0.0.1:3128")
	var dials atomic.Int32
	var resolves atomic.Int32
	ctx := hookCtx(t, false, func(string) ([]net.IP, error) { resolves.Add(1); return []net.IP{net.ParseIP("203.0.113.10")}, nil }, "", &dials)
	ctx.Net.RefuseProxy = true
	forceProxy(ctx, proxyURL)
	for _, ep := range netEntrypoints[:4] {
		t.Run(ep.name, func(t *testing.T) {
			res, err := ep.call(ctx, "https://public.example/")
			if !denied(res, err) {
				t.Fatalf("expected refusal, got %v %v", res, err)
			}
			msg := ""
			if err != nil {
				msg = err.Error()
			} else {
				msg = res.(*eval.TaggedValue).Fields[0].(*eval.TaggedValue).Fields[0].(*eval.StringValue).Value
			}
			if !strings.Contains(msg, "E_NET_PROXY_REFUSED") {
				t.Errorf("refusal must be named E_NET_PROXY_REFUSED, got %q", msg)
			}
		})
	}
	if dials.Load() != 0 || resolves.Load() != 0 {
		t.Errorf("a refused proxy must neither dial (%d) nor resolve (%d) — no direct fallback", dials.Load(), resolves.Load())
	}
}

// Cross-origin redirects strip Authorization, Cookie, Proxy-Authorization
// and operator-designated headers; same-origin redirects keep them.
func TestDestinationPolicy_CrossOriginRedirectStripsSensitiveHeaders(t *testing.T) {
	var gotAuth, gotCookie, gotCustom, gotSafe atomic.Value
	var port string // assigned once the listener exists; the handler runs later
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/start":
			http.Redirect(w, r, "http://b.example:"+port+"/final", http.StatusFound)
		case "/final":
			gotAuth.Store(r.Header.Get("Authorization"))
			gotCookie.Store(r.Header.Get("Cookie"))
			gotCustom.Store(r.Header.Get("X-Api-Key"))
			gotSafe.Store(r.Header.Get("X-Trace"))
			_, _ = w.Write([]byte("ok"))
		}
	}))
	defer srv.Close()
	port = portOf(srv)
	var dials atomic.Int32
	ctx := hookCtx(t, false, func(string) ([]net.IP, error) { return []net.IP{net.ParseIP("203.0.113.10")}, nil }, strings.TrimPrefix(srv.URL, "http://"), &dials)
	ctx.Net.AllowedDomains = []string{"a.example", "b.example"}
	ctx.Net.SensitiveHeaders = []string{"X-Api-Key"}
	headers := &eval.ListValue{Elements: []eval.Value{
		hdr("Authorization", "Bearer secret"), hdr("Cookie", "s=1"), hdr("X-Api-Key", "k"), hdr("X-Trace", "t"),
	}}
	res, err := NetHTTPRequest(ctx, []eval.Value{&eval.StringValue{Value: "GET"}, &eval.StringValue{Value: "http://a.example:" + portOf(srv) + "/start"}, headers, &eval.StringValue{Value: ""}})
	if err != nil || denied(res, err) {
		t.Fatalf("redirect within the allowlist must succeed: %v %v", res, err)
	}
	if v, _ := gotAuth.Load().(string); v != "" {
		t.Errorf("Authorization leaked across origins: %q", v)
	}
	if v, _ := gotCookie.Load().(string); v != "" {
		t.Errorf("Cookie leaked across origins: %q", v)
	}
	if v, _ := gotCustom.Load().(string); v != "" {
		t.Errorf("operator-designated X-Api-Key leaked across origins: %q", v)
	}
	if v, _ := gotSafe.Load().(string); v != "t" {
		t.Errorf("non-sensitive header must survive the redirect, got %q", v)
	}
}

func hdr(name, value string) eval.Value {
	return &eval.RecordValue{Fields: map[string]eval.Value{"name": &eval.StringValue{Value: name}, "value": &eval.StringValue{Value: value}}}
}
