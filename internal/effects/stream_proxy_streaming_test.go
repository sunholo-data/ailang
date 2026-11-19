package effects

import (
	"bufio"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/eval"
)

// M3 of m-net-effect-proxy-boundary (design D-4): once SSE honors a proxy,
// does a token stream still arrive token-by-token through it, stay open across
// keep-alive traffic, and clean up on idle timeout? Measured against a local
// proxy in both HTTP proxy modes — absolute-form (http:// target) and CONNECT
// (https:// target, a minimal tunnel to a TLS origin).
//
// The synchronisation is causal, not timed: the origin sends event "one" and
// then WAITS for the client to signal that it has observed "one" before it
// sends "two". If anything between them buffers, the client never sees "one"
// in time, the origin's wait expires, and releasedByClient is false. The
// buffering_proxy_control subtest proves the instrument detects exactly that.

type streamingOrigin struct {
	release          chan struct{}
	releaseOnce      sync.Once
	releasedByClient chan bool
	// closed is closed only when the origin observed the CLIENT going away.
	closed chan struct{}
	// stop releases a held handler at test end, so a cleanup failure is
	// reported by the assertion instead of hanging httptest.Server.Close.
	stop     chan struct{}
	stopOnce sync.Once
	// served admits exactly one request to the stream. The handler is
	// single-use (one release verdict, one close of `closed`): a second
	// request used to block forever on the verdict send, so the deferred
	// httptest.Server.Close waited forever too — the 10-minute hang.
	served atomic.Bool
}

// streamPath is the only path the origin streams on; the client under test
// always requests it.
const streamPath = "/sse"

// streamingOriginHandler serves: "one"; wait (≤ releaseWait) for the client's
// release; "two"; keepalives × `ka` every kaEvery; then, when holdOpen, hold
// the stream until the client goes away and record the disconnect.
func newStreamingOrigin(releaseWait time.Duration, ka int, kaEvery time.Duration, holdOpen bool) (*streamingOrigin, http.Handler) {
	o := &streamingOrigin{
		release:          make(chan struct{}),
		releasedByClient: make(chan bool, 1),
		closed:           make(chan struct{}),
		stop:             make(chan struct{}),
	}
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Anything else that reaches this loopback port (a foreign client —
		// another test or process — whose old port the kernel handed to this
		// listener) is not the stream under test and must not consume the
		// single-use handler.
		if r.URL.Path != streamPath || !o.served.CompareAndSwap(false, true) {
			http.Error(w, "not the stream under test", http.StatusNotFound)
			return
		}
		fl := w.(http.Flusher)
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		send := func(data string) {
			fmt.Fprintf(w, "data: %s\n\n", data)
			fl.Flush()
		}
		send("one")
		select {
		case <-o.release:
			o.releasedByClient <- true
		case <-time.After(releaseWait):
			o.releasedByClient <- false
		case <-r.Context().Done():
			o.releasedByClient <- false
			return
		case <-o.stop:
			o.releasedByClient <- false
			return
		}
		send("two")
		for i := 0; i < ka; i++ {
			select {
			case <-time.After(kaEvery):
				send("ka")
			case <-r.Context().Done():
				return
			case <-o.stop:
				return
			}
		}
		if holdOpen {
			select {
			case <-r.Context().Done():
				close(o.closed)
			case <-o.stop:
			}
		}
	})
	return o, h
}

func (o *streamingOrigin) clientSawOne() { o.releaseOnce.Do(func() { close(o.release) }) }
func (o *streamingOrigin) shutdown()     { o.stopOnce.Do(func() { close(o.stop) }) }

// released reads the origin's release verdict, bounded so a broken path
// fails rather than hangs.
func (o *streamingOrigin) released(t *testing.T) bool {
	t.Helper()
	select {
	case v := <-o.releasedByClient:
		return v
	case <-time.After(30 * time.Second):
		t.Fatal("origin never reached its release decision")
		return false
	}
}

// recordingProxy is a forward proxy for both modes. buffer=true makes the
// absolute-form path read the whole upstream body before answering — the
// misbehaving proxy the experiment must detect.
//
// It is credential-gated. Its listener is an ordinary 127.0.0.1 ephemeral
// port, and under a parallel `make test` foreign clients (other tests or
// processes) do reach such ports: a port they were using, or "reserved" by
// listen-then-close, is handed to this listener. Before the gate, one such stray "GET /" was
// recorded as a proxy observation and failed the exact-match assertion. Only
// a request carrying this proxy's per-test Proxy-Authorization — which the
// client under test sends because its proxy URL holds the credential — counts
// as an observation; anything else is answered 407, kept in strays, and never
// forwarded or tunnelled. Direct routing still produces zero observations.
type recordingProxy struct {
	mu       sync.Mutex
	seen     []string // authenticated proxy requests: the client under test
	strays   []string // everything else that reached the listener
	upstream string   // origin listener address (host:port)
	buffer   bool
	auth     string // the Proxy-Authorization value the client must send
}

// newRecordingProxy starts a gated proxy for upstream and returns it, its
// server (the caller closes it), and the credentialed proxy URL to hand to
// the client under test.
func newRecordingProxy(t *testing.T, upstream string, buffer bool) (*recordingProxy, *httptest.Server, *url.URL) {
	t.Helper()
	token := make([]byte, 16)
	if _, err := rand.Read(token); err != nil {
		t.Fatal(err)
	}
	user, pass := "stream-proxy-test", hex.EncodeToString(token)
	p := &recordingProxy{
		upstream: upstream,
		buffer:   buffer,
		auth:     "Basic " + base64.StdEncoding.EncodeToString([]byte(user+":"+pass)),
	}
	ps := httptest.NewServer(p)
	pu, err := url.Parse(ps.URL)
	if err != nil {
		ps.Close()
		t.Fatal(err)
	}
	pu.User = url.UserPassword(user, pass)
	return p, ps, pu
}

func (p *recordingProxy) observed() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]string(nil), p.seen...)
}

func (p *recordingProxy) strayRequests() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]string(nil), p.strays...)
}

func (p *recordingProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	line := r.Method + " " + r.URL.String()
	if r.Method == http.MethodConnect {
		line = "CONNECT " + r.Host
	}
	if r.Header.Get("Proxy-Authorization") != p.auth {
		p.mu.Lock()
		p.strays = append(p.strays, fmt.Sprintf("%s (from %s, User-Agent %q)", line, r.RemoteAddr, r.UserAgent()))
		p.mu.Unlock()
		w.Header().Set("Proxy-Authenticate", `Basic realm="stream-proxy-test"`)
		http.Error(w, "not the client under test", http.StatusProxyAuthRequired)
		return
	}
	p.mu.Lock()
	p.seen = append(p.seen, line)
	p.mu.Unlock()
	if r.Method == http.MethodConnect {
		p.tunnel(w)
		return
	}
	out, _ := http.NewRequestWithContext(r.Context(), r.Method, "http://"+p.upstream+r.URL.RequestURI(), r.Body)
	out.Header = r.Header.Clone()
	out.Header.Del("Proxy-Authorization") // hop-by-hop: never forwarded
	resp, err := (&http.Transport{DisableKeepAlives: true}).RoundTrip(out)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	for k, vs := range resp.Header {
		for _, v := range vs {
			w.Header().Add(k, v)
		}
	}
	if p.buffer {
		body, _ := io.ReadAll(resp.Body)
		w.WriteHeader(resp.StatusCode)
		_, _ = w.Write(body)
		return
	}
	w.WriteHeader(resp.StatusCode)
	fl := w.(http.Flusher)
	buf := make([]byte, 512)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			_, _ = w.Write(buf[:n])
			fl.Flush()
		}
		if err != nil {
			return
		}
	}
}

// tunnel is a minimal CONNECT implementation: 200, then splice bytes both ways
// until either side closes.
func (p *recordingProxy) tunnel(w http.ResponseWriter) {
	up, err := net.Dial("tcp", p.upstream)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	hj, _ := w.(http.Hijacker)
	client, rw, err := hj.Hijack()
	if err != nil {
		up.Close()
		return
	}
	_, _ = rw.WriteString("HTTP/1.1 200 Connection Established\r\n\r\n")
	_ = rw.Flush()
	done := make(chan struct{}, 2)
	go func() { _, _ = io.Copy(up, rw); done <- struct{}{} }()
	go func() { _, _ = io.Copy(client, up); done <- struct{}{} }()
	<-done
	client.Close()
	up.Close()
}

type streamRun struct {
	data       []string
	sawOneAt   time.Time
	sawTwoAt   time.Time
	errorAt    time.Time
	sawTimeout bool
	resolver   int
}

// runProxiedSSE connects through proxyURL with the production SSE constructor
// and runs the event loop until the idle timeout ends it.
func runProxiedSSE(t *testing.T, target string, proxyURL *url.URL, origin *streamingOrigin, tlsCfg *tls.Config, idle time.Duration) streamRun {
	t.Helper()
	var run streamRun
	ctx := newSSETestContext(t)
	ctx.Stream.IdleTimeout = idle
	ctx.Stream.MaxDuration = 30 * time.Second
	ctx.Stream.tlsClientConfig = tlsCfg
	ctx.Stream.proxySelector = func(*http.Request) (*url.URL, error) { return proxyURL, nil }
	ctx.Stream.lookupIP = func(h string) ([]net.IP, error) {
		run.resolver++
		return nil, fmt.Errorf("no such host: %s (hermetic resolver)", h)
	}
	result, err := StreamSSEConnect(ctx, []eval.Value{&eval.StringValue{Value: target}, &eval.StringValue{Value: "{}"}})
	if err != nil {
		t.Fatal(err)
	}
	tag, ok := result.(*eval.TaggedValue)
	if !ok || tag.CtorName != "Ok" {
		t.Fatalf("SSE connect through the proxy: %v", result)
	}
	conn := tag.Fields[0]
	ctx.FnCaller = func(_ eval.Value, arg eval.Value) (eval.Value, error) {
		ev, ok := arg.(*eval.TaggedValue)
		if !ok {
			return &eval.BoolValue{Value: true}, nil
		}
		switch ev.CtorName {
		case "SSEData":
			s := ev.Fields[1].(*eval.StringValue).Value
			run.data = append(run.data, s)
			switch s {
			case "one":
				run.sawOneAt = time.Now()
				origin.clientSawOne()
			case "two":
				run.sawTwoAt = time.Now()
			}
		case "StreamError":
			run.errorAt = time.Now()
			run.sawTimeout = strings.Contains(fmt.Sprint(ev), "Timeout")
		}
		return &eval.BoolValue{Value: true}, nil
	}
	if _, err := StreamOnEvent(ctx, []eval.Value{conn, &eval.UnitValue{}}); err != nil {
		t.Fatal(err)
	}
	if _, err := StreamRunEventLoop(ctx, []eval.Value{conn}); err != nil {
		t.Fatal(err)
	}
	_, _ = StreamClose(ctx, []eval.Value{conn})
	return run
}

func TestStreamSSEProxyBoundaryStreaming(t *testing.T) {
	const (
		idle    = time.Second
		ka      = 3
		kaEvery = 300 * time.Millisecond
	)
	wantData := "one two ka ka ka"

	assertStreamed := func(t *testing.T, run streamRun, origin *streamingOrigin, proxy *recordingProxy, wantSeen string) {
		t.Helper()
		if got := proxy.observed(); len(got) != 1 || got[0] != wantSeen {
			t.Fatalf("proxy observed %q, want exactly [%q] (direct routing produces zero observations); rejected strays: %q", got, wantSeen, proxy.strayRequests())
		}
		if strays := proxy.strayRequests(); len(strays) > 1 {
			// Only the deliberate stray is expected; more is real-world
			// traffic on a reused port, harmless now but worth seeing.
			t.Logf("proxy rejected %d unauthenticated requests: %q", len(strays), strays)
		}
		if !origin.released(t) {
			t.Fatal("event one was NOT observable before event two was released: the path buffered the stream")
		}
		if got := strings.Join(run.data, " "); got != wantData {
			t.Fatalf("events = %q, want %q", got, wantData)
		}
		if run.resolver != 0 {
			t.Errorf("resolver calls = %d, want 0 (the proxy resolves a hostname target)", run.resolver)
		}
		// Keep-alive: the stream outlived more than one idle interval because
		// the keepalive events crossed the proxy; only silence ended it.
		if !run.sawTimeout || run.errorAt.Sub(run.sawTwoAt) < idle+2*kaEvery {
			t.Fatalf("idle timeout: sawTimeout=%v after %v, want a Timeout no sooner than %v after event two", run.sawTimeout, run.errorAt.Sub(run.sawTwoAt), idle+2*kaEvery)
		}
		// Cleanup: closing the stream tears down the proxied connection, so
		// the origin observes the disconnect.
		select {
		case <-origin.closed:
		case <-time.After(5 * time.Second):
			t.Fatal("origin never observed the client disconnect through the proxy: cleanup leaked the connection")
		}
	}

	t.Run("absolute_form_unbuffered", func(t *testing.T) {
		boundSubtest(t)
		origin, h := newStreamingOrigin(15*time.Second, ka, kaEvery, true)
		up := httptest.NewServer(h)
		defer up.Close()
		proxy, ps, pu := newRecordingProxy(t, up.Listener.Addr().String(), false)
		defer ps.Close()
		defer origin.shutdown() // runs first: frees a held origin before the servers close
		// The stray clients a parallel `make test` produced, made deterministic:
		// one at the proxy, one at the origin, before the client connects.
		sendStray(t, ps.Listener.Addr().String(), "407")
		sendStray(t, up.Listener.Addr().String(), "404")
		run := runProxiedSSE(t, "http://stream.test"+streamPath, pu, origin, nil, idle)
		assertStreamed(t, run, origin, proxy, "GET http://stream.test"+streamPath)
	})

	t.Run("connect_tunnel_unbuffered", func(t *testing.T) {
		boundSubtest(t)
		origin, h := newStreamingOrigin(15*time.Second, ka, kaEvery, true)
		up := httptest.NewTLSServer(h) // certificate covers example.com
		defer up.Close()
		pool := x509.NewCertPool()
		pool.AddCert(up.Certificate())
		proxy, ps, pu := newRecordingProxy(t, up.Listener.Addr().String(), false)
		defer ps.Close()
		defer origin.shutdown() // runs first: frees a held origin before the servers close
		// The observed flake, made deterministic: a stray "GET /" at the proxy.
		sendStray(t, ps.Listener.Addr().String(), "407")
		run := runProxiedSSE(t, "https://example.com"+streamPath, pu, origin, &tls.Config{RootCAs: pool}, idle)
		assertStreamed(t, run, origin, proxy, "CONNECT example.com:443")
	})

	t.Run("buffering_proxy_control", func(t *testing.T) {
		boundSubtest(t)
		// A proxy that buffers until EOF must make the experiment fail: the
		// origin's wait for the client's release expires, so "one" was not
		// observable before "two" — the condition the subtests above reject.
		origin, h := newStreamingOrigin(500*time.Millisecond, 0, 0, false)
		up := httptest.NewServer(h)
		defer up.Close()
		proxy, ps, pu := newRecordingProxy(t, up.Listener.Addr().String(), true)
		defer ps.Close()
		defer origin.shutdown() // runs first: frees a held origin before the servers close
		run := runProxiedSSE(t, "http://stream.test"+streamPath, pu, origin, nil, idle)
		if got := proxy.observed(); len(got) != 1 {
			t.Fatalf("buffering proxy observed %q, want exactly one request from the client under test", got)
		}
		if origin.released(t) {
			t.Fatal("buffering proxy: the client released event two — the instrument cannot see buffering")
		}
		if got := strings.Join(run.data, " "); got != "one two" {
			t.Fatalf("buffering proxy still delivers everything at EOF: events = %q, want %q", got, "one two")
		}
	})
}

// subtestDeadline bounds one subtest. A healthy subtest takes about 2s; every
// wait inside it is bounded well below this, so only a genuine hang (a held
// handler, a blocked Server.Close, a stream that never ends) reaches it.
const subtestDeadline = 45 * time.Second

// boundSubtest makes a hang fail in seconds, naming the subtest and dumping
// every goroutine, instead of reaching the 10-minute package timeout. A hang
// cannot be failed from the test goroutine (it is the one stuck — often in a
// deferred Close), so the watchdog panics, as the package timeout would.
func boundSubtest(t *testing.T) {
	t.Helper()
	name := t.Name()
	timer := time.AfterFunc(subtestDeadline, func() {
		buf := make([]byte, 1<<20)
		buf = buf[:runtime.Stack(buf, true)]
		panic(fmt.Sprintf("%s did not finish within %v: hung (look for a blocked httptest.Server.Close, origin handler or stream read below)\n\n%s", name, subtestDeadline, buf))
	})
	t.Cleanup(func() { timer.Stop() }) // cleanups run after the subtest's defers
}

// sendStray plays a foreign client that reaches addr, a loopback port the
// kernel reassigned: a plain origin-form "GET /". It asserts the status
// the gate answers with, so a stray can neither count as an observation nor
// consume the single-use origin.
func sendStray(t *testing.T, addr, wantStatus string) {
	t.Helper()
	c, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		t.Fatalf("stray client: %v", err)
	}
	defer c.Close()
	_ = c.SetDeadline(time.Now().Add(10 * time.Second))
	if _, err := io.WriteString(c, "GET / HTTP/1.1\r\nHost: "+addr+"\r\nConnection: close\r\n\r\n"); err != nil {
		t.Fatalf("stray client: %v", err)
	}
	status, err := bufio.NewReader(c).ReadString('\n')
	if err != nil {
		t.Fatalf("stray client got no status line from %s: %v", addr, err)
	}
	if f := strings.Fields(status); len(f) < 2 || f[1] != wantStatus {
		t.Fatalf("stray GET / to %s answered %q, want %s", addr, strings.TrimSpace(status), wantStatus)
	}
}
