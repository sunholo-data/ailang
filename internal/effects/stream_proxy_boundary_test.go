package effects

import (
	"bytes"
	"context"
	"fmt"
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

// M2 of m-net-effect-proxy-boundary: the HTTP Stream transports (SSE GET, SSE
// POST, NDJSON POST) honor the process proxy environment.
//
// Before this milestone streamPolicy hard-coded a no-proxy selector, so a
// Stream request to a non-loopback host dialed straight past HTTP_PROXY /
// HTTPS_PROXY — including CI's poison proxy. The env-driven tests below run the
// PRODUCTION constructors in a subprocess (Go caches the proxy environment once
// per process) with nil proxy seams, and assert the parent's fake proxy
// observed each request. On the pre-fix code the helper's direct route asks the
// injected resolver for stream.test, gets "no such host", and the fake proxy
// observes nothing — the escape, made hermetic.

const (
	streamProxyHelperEnv = "AILANG_STREAM_PROXY_HELPER"
	streamProxyExpectEnv = "AILANG_STREAM_PROXY_EXPECT"
	streamProxyBody      = "via-proxy"
)

// fakeForwardProxy is an absolute-form HTTP proxy that serves Stream payloads
// itself and records "<METHOD> <absolute URL>" for every request it receives.
type fakeForwardProxy struct {
	mu   sync.Mutex
	seen []string
	srv  *httptest.Server
}

func newFakeForwardProxy(t *testing.T) *fakeForwardProxy {
	t.Helper()
	p := &fakeForwardProxy{}
	p.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p.mu.Lock()
		p.seen = append(p.seen, r.Method+" "+r.URL.String())
		p.mu.Unlock()
		if strings.HasSuffix(r.URL.Path, "/ndjson") {
			w.Header().Set("Content-Type", "application/x-ndjson")
			fmt.Fprintf(w, "{\"response\":%q}\n", streamProxyBody)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintf(w, "data: %s\n\n", streamProxyBody)
	}))
	t.Cleanup(p.srv.Close)
	return p
}

func (p *fakeForwardProxy) observed() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]string(nil), p.seen...)
}

// streamProxyOps are the three production HTTP Stream constructors, keyed by
// the helper selector. Every target is a non-loopback hostname, so Go's
// ProxyFromEnvironment does not apply its loopback bypass.
var streamProxyOps = map[string]struct {
	want string // what the fake proxy must observe
	call func(ctx *EffContext) (eval.Value, error)
}{
	"sse_get": {
		want: "GET http://stream.test/sse",
		call: func(ctx *EffContext) (eval.Value, error) {
			return StreamSSEConnect(ctx, []eval.Value{
				&eval.StringValue{Value: "http://stream.test/sse"},
				&eval.StringValue{Value: "{}"},
			})
		},
	},
	"sse_post": {
		want: "POST http://stream.test/sse-post",
		call: func(ctx *EffContext) (eval.Value, error) {
			return StreamSSEPost(ctx, []eval.Value{
				&eval.StringValue{Value: "http://stream.test/sse-post"},
				&eval.StringValue{Value: "{}"},
				emptyConfig(),
			})
		},
	},
	"ndjson_post": {
		want: "POST http://stream.test/ndjson",
		call: func(ctx *EffContext) (eval.Value, error) {
			return StreamNDJSONPost(ctx, []eval.Value{
				&eval.StringValue{Value: "http://stream.test/ndjson"},
				&eval.StringValue{Value: "{}"},
				emptyConfig(),
			})
		},
	},
}

// runStreamProxyHelper re-executes this test binary for op with the process
// proxy environment pointing at proxyURL, and returns the helper's output.
func runStreamProxyHelper(t *testing.T, op, proxyURL string, extraEnv ...string) []byte {
	t.Helper()
	if os.Getenv(streamProxyHelperEnv) != "" {
		t.Fatal("env-proxy parent test must not run inside the helper subprocess")
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestStreamProxyBoundaryEnvHelper$", "-test.v", "-test.count=1")
	cmd.Env = append(stripProxyEnv(os.Environ()),
		streamProxyHelperEnv+"="+op,
		"HTTP_PROXY="+proxyURL,
		"HTTPS_PROXY="+proxyURL,
		"NO_PROXY=",
	)
	cmd.Env = append(cmd.Env, extraEnv...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("stream proxy helper (%s) failed: %v\n%s", op, err, out)
	}
	if !bytes.Contains(out, []byte("--- PASS: TestStreamProxyBoundaryEnvHelper")) {
		t.Fatalf("stream proxy helper (%s) did not report PASS:\n%s", op, out)
	}
	return out
}

func assertProxyObserved(t *testing.T, p *fakeForwardProxy, want string) {
	t.Helper()
	got := p.observed()
	for _, s := range got {
		if s == want {
			return
		}
	}
	t.Fatalf("fake proxy did not observe %q (the Stream request bypassed the process proxy); saw %q", want, got)
}

// TestStreamSSEProxyBoundary (AC-M2.1): SSE GET and SSE POST, through the
// production constructors, are observed at the process proxy.
func TestStreamSSEProxyBoundary(t *testing.T) {
	for _, op := range []string{"sse_get", "sse_post"} {
		t.Run(op+"_observed_at_env_proxy", func(t *testing.T) {
			p := newFakeForwardProxy(t)
			runStreamProxyHelper(t, op, p.srv.URL)
			assertProxyObserved(t, p, streamProxyOps[op].want)
		})
	}
	// The CI lane's poison (HTTP(S)_PROXY=http://127.0.0.1:9): a non-loopback
	// Stream request must now hit the poison instead of escaping.
	t.Run("poisoned_lane_denies_non_loopback", func(t *testing.T) {
		runStreamProxyHelper(t, "sse_get", "http://127.0.0.1:9", streamProxyExpectEnv+"=poison")
	})
}

// TestStreamNDJSONProxyBoundary (AC-M2.1): NDJSON POST, through the production
// constructor, is observed at the process proxy.
func TestStreamNDJSONProxyBoundary(t *testing.T) {
	t.Run("ndjson_post_observed_at_env_proxy", func(t *testing.T) {
		p := newFakeForwardProxy(t)
		runStreamProxyHelper(t, "ndjson_post", p.srv.URL)
		assertProxyObserved(t, p, streamProxyOps["ndjson_post"].want)
	})
}

// TestStreamProxyBoundaryEnvHelper is the subprocess half: it runs one
// production Stream constructor with NO proxy seam, so routing comes from
// http.ProxyFromEnvironment. The injected resolver refuses every name, which
// keeps the direct route hermetic: if the constructor ignores the proxy it
// fails here instead of reaching the network.
func TestStreamProxyBoundaryEnvHelper(t *testing.T) {
	op := os.Getenv(streamProxyHelperEnv)
	if op == "" {
		t.Skip("stream proxy-boundary environment helper (subprocess only)")
	}
	spec, ok := streamProxyOps[op]
	if !ok {
		t.Fatalf("unknown helper op %q", op)
	}
	var resolverCalls int
	ctx := newSSETestContext(t)
	ctx.Stream.lookupIP = func(hostname string) ([]net.IP, error) {
		resolverCalls++
		return nil, fmt.Errorf("no such host: %s (hermetic resolver)", hostname)
	}
	result, err := spec.call(ctx)
	if err != nil {
		t.Fatalf("%s returned Go error: %v", op, err)
	}
	if os.Getenv(streamProxyExpectEnv) == "poison" {
		_, msg := unwrapStreamErr(t, result)
		for _, want := range []string{"proxyconnect", "127.0.0.1:9", "refused"} {
			if !strings.Contains(strings.ToLower(msg), want) {
				t.Fatalf("%s under the poison proxy: %q does not carry %q — the request did not hit the poison", op, msg, want)
			}
		}
		if resolverCalls != 0 {
			t.Errorf("resolver calls = %d, want 0 under the poison proxy", resolverCalls)
		}
		return
	}
	if got := firstStreamData(t, ctx, result); got != streamProxyBody && !strings.Contains(got, streamProxyBody) {
		t.Fatalf("%s first data = %q, want the proxy-served %q", op, got, streamProxyBody)
	}
	if resolverCalls != 0 {
		t.Errorf("resolver calls = %d, want 0: a proxied hostname target is resolved by the proxy (D-1)", resolverCalls)
	}
}

// firstStreamData requires an Ok connection, runs its event loop until the
// first data event and returns that event's text.
func firstStreamData(t *testing.T, ctx *EffContext, result eval.Value) string {
	t.Helper()
	tag, ok := result.(*eval.TaggedValue)
	if !ok || tag.CtorName != "Ok" {
		t.Fatalf("stream connect did not return Ok: %v", result)
	}
	conn := tag.Fields[0]
	var data string
	ctx.FnCaller = func(_ eval.Value, arg eval.Value) (eval.Value, error) {
		ev, ok := arg.(*eval.TaggedValue)
		if ok && ev.CtorName == "SSEData" && len(ev.Fields) >= 2 {
			if s, ok := ev.Fields[1].(*eval.StringValue); ok {
				data = s.Value
			}
			return &eval.BoolValue{Value: false}, nil
		}
		return &eval.BoolValue{Value: true}, nil
	}
	if _, err := StreamOnEvent(ctx, []eval.Value{conn, &eval.UnitValue{}}); err != nil {
		t.Fatalf("onEvent: %v", err)
	}
	if _, err := StreamRunEventLoop(ctx, []eval.Value{conn}); err != nil {
		t.Fatalf("runEventLoop: %v", err)
	}
	_, _ = StreamClose(ctx, []eval.Value{conn})
	return data
}

// streamProbeCtx is a Stream context with a counting resolver and dialer, for
// the in-process route tests below.
func streamProbeCtx(t *testing.T, probe *netProbe) *EffContext {
	t.Helper()
	ctx := newSSETestContext(t)
	ctx.Stream.lookupIP = func(hostname string) ([]net.IP, error) {
		probe.mu.Lock()
		probe.resolverCalls++
		probe.mu.Unlock()
		return probe.resolve(hostname)
	}
	ctx.Stream.dialContext = func(c context.Context, network, addr string) (net.Conn, error) {
		probe.mu.Lock()
		probe.dialCalls++
		probe.dialAddrs = append(probe.dialAddrs, addr)
		dest, routed := probe.routes[addr]
		probe.mu.Unlock()
		if routed {
			addr = dest
		}
		// Hermetic: only loopback listeners are ever dialed. A mutated
		// production route that dials an unrouted target address fails
		// here instead of reaching the network.
		if host, _, _ := net.SplitHostPort(addr); !net.ParseIP(host).IsLoopback() {
			return nil, fmt.Errorf("hermetic probe: unrouted non-loopback dial to %s", addr)
		}
		return (&net.Dialer{}).DialContext(c, network, addr)
	}
	return ctx
}

// TestStreamSSEProxyBoundaryRoutes pins the Stream route split to Net's (D-1):
// direct requests stay pinned, a proxied literal IP is still validated with
// zero DNS, and restricted mode refuses a proxy instead of falling back.
func TestStreamSSEProxyBoundaryRoutes(t *testing.T) {
	sseOrigin := func(t *testing.T, body string) *httptest.Server {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/event-stream")
			fmt.Fprintf(w, "data: %s\n\n", body)
		}))
		t.Cleanup(srv.Close)
		return srv
	}
	connect := func(ctx *EffContext, u string) (eval.Value, error) {
		return StreamSSEConnect(ctx, []eval.Value{&eval.StringValue{Value: u}, &eval.StringValue{Value: "{}"}})
	}

	t.Run("no_proxy_route_dials_pinned_ip", func(t *testing.T) {
		pinned := sseOrigin(t, "pinned")
		p := newFakeForwardProxy(t)
		proxyURL, _ := url.Parse(p.srv.URL)
		probe := &netProbe{
			resolve: func(string) ([]net.IP, error) { return []net.IP{net.ParseIP("203.0.113.10")}, nil },
			routes:  map[string]string{"203.0.113.10:80": pinned.Listener.Addr().String()},
		}
		ctx := streamProbeCtx(t, probe)
		// NO_PROXY analogue: a proxy is configured, but not for this host.
		ctx.Stream.proxySelector = func(r *http.Request) (*url.URL, error) {
			if r.URL.Hostname() == "stream.test" {
				return nil, nil
			}
			return proxyURL, nil
		}
		result, err := connect(ctx, "http://stream.test/sse")
		if err != nil {
			t.Fatal(err)
		}
		if got := firstStreamData(t, ctx, result); got != "pinned" {
			t.Fatalf("data = %q, want the pinned endpoint's", got)
		}
		s := probe.snapshot()
		if s.resolverCalls != 1 || len(s.dialAddrs) != 1 || s.dialAddrs[0] != "203.0.113.10:80" {
			t.Fatalf("direct route: resolver=%d dials=%q, want exactly one resolve and one dial to 203.0.113.10:80", s.resolverCalls, s.dialAddrs)
		}
		if got := p.observed(); len(got) != 0 {
			t.Fatalf("NO_PROXY request reached the proxy: %q", got)
		}
	})

	t.Run("proxied_hostname_zero_dns", func(t *testing.T) {
		p := newFakeForwardProxy(t)
		proxyURL, _ := url.Parse(p.srv.URL)
		probe := &netProbe{resolve: func(h string) ([]net.IP, error) { return nil, fmt.Errorf("no such host: %s", h) }}
		ctx := streamProbeCtx(t, probe)
		ctx.Stream.proxySelector = func(*http.Request) (*url.URL, error) { return proxyURL, nil }
		result, err := connect(ctx, "http://stream.test/sse")
		if err != nil {
			t.Fatal(err)
		}
		if got := firstStreamData(t, ctx, result); got != streamProxyBody {
			t.Fatalf("data = %q, want proxy-served", got)
		}
		s := probe.snapshot()
		if s.resolverCalls != 0 {
			t.Errorf("resolver calls = %d, want 0 on the proxy route", s.resolverCalls)
		}
		if len(s.dialAddrs) != 1 || s.dialAddrs[0] != proxyURL.Host {
			t.Errorf("dials = %q, want exactly one dial to the proxy %s (never a substituted target IP)", s.dialAddrs, proxyURL.Host)
		}
	})

	t.Run("proxied_literal_private_ip_refused_before_dial", func(t *testing.T) {
		p := newFakeForwardProxy(t)
		proxyURL, _ := url.Parse(p.srv.URL)
		probe := &netProbe{resolve: func(string) ([]net.IP, error) { return nil, nil }}
		ctx := streamProbeCtx(t, probe)
		ctx.Stream.proxySelector = func(*http.Request) (*url.URL, error) { return proxyURL, nil }
		result, err := connect(ctx, "http://10.1.2.3/sse")
		if err != nil {
			t.Fatal(err)
		}
		ctor, msg := unwrapStreamErr(t, result)
		if ctor != "ConnectionFailed" || !strings.Contains(msg, "E_STREAM_DISALLOWED_HOST") || !strings.Contains(msg, "private IP blocked") {
			t.Fatalf("got %s(%q), want ConnectionFailed carrying E_STREAM_DISALLOWED_HOST private IP blocked", ctor, msg)
		}
		if s := probe.snapshot(); s.resolverCalls != 0 || s.dialCalls != 0 {
			t.Errorf("resolver=%d dials=%d, want zero DNS and zero dials", s.resolverCalls, s.dialCalls)
		}
		if got := p.observed(); len(got) != 0 {
			t.Errorf("blocked literal IP reached the proxy: %q", got)
		}
	})

	t.Run("restricted_refuses_proxy_without_fallback", func(t *testing.T) {
		p := newFakeForwardProxy(t)
		proxyURL, _ := url.Parse(p.srv.URL)
		probe := &netProbe{resolve: func(string) ([]net.IP, error) { return []net.IP{net.ParseIP("203.0.113.10")}, nil }}
		ctx := streamProbeCtx(t, probe)
		ctx.Stream.RefuseProxy = true
		ctx.Stream.proxySelector = func(*http.Request) (*url.URL, error) { return proxyURL, nil }
		result, err := connect(ctx, "http://stream.test/sse")
		if err != nil {
			t.Fatal(err)
		}
		_, msg := unwrapStreamErr(t, result)
		if !strings.Contains(msg, "E_STREAM_PROXY_REFUSED") {
			t.Fatalf("message %q, want E_STREAM_PROXY_REFUSED", msg)
		}
		if s := probe.snapshot(); s.resolverCalls != 0 || s.dialCalls != 0 {
			t.Errorf("resolver=%d dials=%d, want a refusal with no direct fallback", s.resolverCalls, s.dialCalls)
		}
		if got := p.observed(); len(got) != 0 {
			t.Errorf("refused proxy was still used: %q", got)
		}
	})

	t.Run("child_context_keeps_proxy_policy", func(t *testing.T) {
		sc := NewStreamContext()
		sc.RefuseProxy = true
		sc.proxySelector = func(*http.Request) (*url.URL, error) { return nil, nil }
		c := sc.Child()
		if !c.RefuseProxy || c.proxySelector == nil {
			t.Fatalf("Child() dropped proxy policy: RefuseProxy=%v selector=%v", c.RefuseProxy, c.proxySelector != nil)
		}
	})
}

// unwrapStreamErr extracts the StreamError constructor and message from a
// Result.Err stream value.
func unwrapStreamErr(t *testing.T, v eval.Value) (ctor, msg string) {
	t.Helper()
	tag, ok := v.(*eval.TaggedValue)
	if !ok || tag.CtorName != "Err" {
		t.Fatalf("expected Result.Err, got %v", v)
	}
	inner, ok := tag.Fields[0].(*eval.TaggedValue)
	if !ok {
		t.Fatalf("expected StreamError TaggedValue, got %T", tag.Fields[0])
	}
	if len(inner.Fields) > 0 {
		if s, ok := inner.Fields[0].(*eval.StringValue); ok {
			msg = s.Value
		}
	}
	return inner.CtorName, msg
}
