package effects_test

// Test harness for the bridge (M-SERVEAPI-WS-BRIDGE): a scripted fake
// upstream (plain or TLS), a "browser" that reaches the bridge's client leg
// through a real server-side upgrade, and a Go step function standing in for
// the AILANG one. No network beyond 127.0.0.1; CI never needs Vertex.

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/platform/streamws"
)

// wireFrame is one frame on a real socket.
type wireFrame struct {
	Binary bool
	Data   []byte
}

// fakeUpstream is a scripted WebSocket upstream. It records the handshake
// headers and every frame it receives, sends its script on connect (with an
// optional pause before a given index), and can close after the script.
type fakeUpstream struct {
	srv        *httptest.Server
	mu         sync.Mutex
	headers    http.Header
	received   []wireFrame
	script     []wireFrame
	closeAfter bool
	pauseAt    int           // index in script to wait at (0 = none)
	resume     chan struct{} // closed to continue past pauseAt
	gone       chan struct{} // closed when the upstream's read loop ends
}

func newFakeUpstream(t *testing.T, useTLS bool, script []wireFrame) *fakeUpstream {
	t.Helper()
	f := &fakeUpstream{script: script, resume: make(chan struct{}), gone: make(chan struct{})}
	up := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		f.headers = r.Header.Clone()
		f.mu.Unlock()
		conn, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		go f.sendScript(conn)
		defer close(f.gone)
		for {
			mt, data, err := conn.ReadMessage()
			if err != nil {
				return
			}
			f.mu.Lock()
			f.received = append(f.received, wireFrame{Binary: mt == websocket.BinaryMessage, Data: data})
			f.mu.Unlock()
		}
	})
	if useTLS {
		f.srv = httptest.NewTLSServer(h)
	} else {
		f.srv = httptest.NewServer(h)
	}
	t.Cleanup(f.srv.Close)
	return f
}

func (f *fakeUpstream) sendScript(conn *websocket.Conn) {
	for i, fr := range f.script {
		if f.pauseAt > 0 && i == f.pauseAt {
			<-f.resume
		}
		mt := websocket.TextMessage
		if fr.Binary {
			mt = websocket.BinaryMessage
		}
		if err := conn.WriteMessage(mt, fr.Data); err != nil {
			return
		}
	}
	if f.closeAfter {
		_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "script done"))
	}
}

func (f *fakeUpstream) url() string {
	if strings.HasPrefix(f.srv.URL, "https") {
		return "wss" + strings.TrimPrefix(f.srv.URL, "https")
	}
	return "ws" + strings.TrimPrefix(f.srv.URL, "http")
}

func (f *fakeUpstream) tlsConfig() *tls.Config {
	pool := x509.NewCertPool()
	pool.AddCert(f.srv.Certificate())
	return &tls.Config{RootCAs: pool}
}

func (f *fakeUpstream) got() ([]wireFrame, http.Header) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]wireFrame(nil), f.received...), f.headers
}

// stepFunc is the Go stand-in for an AILANG step: (state, frame) -> (state, verdict).
type stepFunc func(state, frame eval.Value) (eval.Value, eval.Value)

// bridgeHarness wires a browser <-> client leg <-> bridge <-> upstream.
type bridgeHarness struct {
	ctx      *effects.EffContext
	browser  *websocket.Conn
	clientID int
	upID     int
}

// newBridgeHarness dials the upstream through Stream.connect and brings a
// browser in through a real server-side upgrade adopted into the context.
func newBridgeHarness(t *testing.T, up *fakeUpstream, configure func(*effects.StreamContext)) *bridgeHarness {
	t.Helper()
	useWSTransport(t)
	ctx := effects.NewEffContext(nil)
	ctx.Grant(effects.NewCapability("Stream"))
	ctx.Stream = effects.NewStreamContext()
	ctx.Stream.AllowHTTP = true
	ctx.Stream.AllowLocalhost = true
	ctx.Stream.IdleTimeout = 5 * time.Second
	ctx.Stream.MaxDuration = 20 * time.Second
	if up != nil && strings.HasPrefix(up.srv.URL, "https") {
		ctx.Stream.SetTLSClientConfig(up.tlsConfig())
	}
	if configure != nil {
		configure(ctx.Stream)
	}

	accepted := make(chan effects.StreamTransport, 1)
	front := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tr, err := streamws.Accept(w, r, streamws.AcceptOptions{MaxFrameSize: ctx.Stream.MaxFrameSize})
		if err != nil {
			t.Errorf("accept: %v", err)
			return
		}
		accepted <- tr
	}))
	t.Cleanup(front.Close)
	browser, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(front.URL, "http"), nil)
	if err != nil {
		t.Fatalf("browser dial: %v", err)
	}
	t.Cleanup(func() { _ = browser.Close() })
	clientID, err := effects.AdoptStreamConnection(ctx.Stream, <-accepted)
	if err != nil {
		t.Fatalf("adopt: %v", err)
	}
	h := &bridgeHarness{ctx: ctx, browser: browser, clientID: clientID}
	if up != nil {
		res, err := effects.StreamConnect(ctx, []eval.Value{
			&eval.StringValue{Value: up.url()},
			&eval.RecordValue{Fields: map[string]eval.Value{}},
		})
		if err != nil {
			t.Fatalf("connect: %v", err)
		}
		h.upID = connIDOf(t, res)
	}
	return h
}

func connIDOf(t *testing.T, res eval.Value) int {
	t.Helper()
	ok, isTag := res.(*eval.TaggedValue)
	if !isTag || ok.CtorName != "Ok" {
		t.Fatalf("connect failed: %v", res)
	}
	sc := ok.Fields[0].(*eval.TaggedValue)
	return sc.Fields[0].(*eval.IntValue).Value
}

func streamConn(id int) eval.Value {
	return &eval.TaggedValue{CtorName: "StreamConn", Fields: []eval.Value{&eval.IntValue{Value: id}}}
}

// bridgeResult is what StreamBridge returned.
type bridgeResult struct {
	state eval.Value
	end   *eval.TaggedValue
	err   error
	at    time.Time
}

// run starts the bridge with a Go step and returns a channel for its result.
func (h *bridgeHarness) run(init eval.Value, step stepFunc) <-chan bridgeResult {
	h.ctx.FnCallerN = func(_ eval.Value, args []eval.Value) (eval.Value, error) {
		s, v := step(args[0], args[1])
		return &eval.TupleValue{Elements: []eval.Value{s, v}}, nil
	}
	out := make(chan bridgeResult, 1)
	go func() {
		res, err := effects.StreamBridge(h.ctx, []eval.Value{streamConn(h.clientID), streamConn(h.upID), init, &eval.UnitValue{}})
		r := bridgeResult{err: err, at: time.Now()}
		if tup, ok := res.(*eval.TupleValue); ok {
			r.state = tup.Elements[0]
			r.end, _ = tup.Elements[1].(*eval.TaggedValue)
		}
		out <- r
	}()
	return out
}

func waitBridge(t *testing.T, ch <-chan bridgeResult) bridgeResult {
	t.Helper()
	select {
	case r := <-ch:
		if r.err != nil {
			t.Fatalf("bridge error: %v", r.err)
		}
		return r
	case <-time.After(15 * time.Second):
		t.Fatal("bridge did not return")
	}
	return bridgeResult{}
}

// verdict builds a Verdict value.
func verdict(ctor string, fields ...eval.Value) eval.Value {
	return &eval.TaggedValue{CtorName: ctor, Fields: fields}
}

// forwardAll counts frames in an IntValue state and forwards everything.
func forwardAll(state, _ eval.Value) (eval.Value, eval.Value) {
	n := state.(*eval.IntValue).Value
	return &eval.IntValue{Value: n + 1}, verdict("Forward")
}

// readBrowser reads n frames from the browser with a deadline.
func readBrowser(t *testing.T, c *websocket.Conn, n int) []wireFrame {
	t.Helper()
	_ = c.SetReadDeadline(time.Now().Add(10 * time.Second))
	out := make([]wireFrame, 0, n)
	for len(out) < n {
		mt, data, err := c.ReadMessage()
		if err != nil {
			t.Fatalf("browser read after %d/%d frames: %v", len(out), n, err)
		}
		out = append(out, wireFrame{Binary: mt == websocket.BinaryMessage, Data: data})
	}
	return out
}
