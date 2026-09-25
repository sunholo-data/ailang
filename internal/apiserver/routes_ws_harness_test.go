package apiserver

// Harness for the WebSocket route tests (M-SERVEAPI-WS-BRIDGE): a real
// serve-api Server over httptest, AILANG fixtures from testdata/ws, a
// scripted fake upstream, and a gorilla "browser". CI never needs Vertex.

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/platform/streamws"
)

type wsFixture struct {
	srv  *Server
	http *httptest.Server
	eff  *effects.EffContext
}

// newWSFixture copies testdata/ws/<modules> into a temp project (replacing
// __UPSTREAM__), loads them into a Server with --caps Stream, and serves
// its routes. cfg.EffCtx is filled in; configure may adjust the Stream
// context before loading.
func newWSFixture(t testing.TB, cfg Config, upstream string, configure func(*effects.EffContext), modules ...string) *wsFixture {
	t.Helper()
	streamws.Register()
	t.Cleanup(func() { effects.RegisterStreamTransport("ws", nil) })

	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("AILANG_STDLIB_PATH", repoRoot)
	root := t.TempDir()
	var paths []string
	for _, m := range modules {
		src, err := os.ReadFile(filepath.Join("testdata", "ws", m+".ail"))
		if err != nil {
			t.Fatal(err)
		}
		dst := filepath.Join(root, "ws", m+".ail")
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			t.Fatal(err)
		}
		body := strings.ReplaceAll(string(src), "__UPSTREAM__", upstream)
		if err := os.WriteFile(dst, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		paths = append(paths, dst)
	}

	eff := effects.NewEffContext(nil)
	eff.Grant(effects.NewCapability("Stream"))
	eff.Stream = effects.NewStreamContext()
	eff.Stream.AllowHTTP = true
	eff.Stream.AllowLocalhost = true
	eff.Stream.IdleTimeout = 5 * time.Second
	eff.Stream.MaxDuration = 20 * time.Second
	if configure != nil {
		configure(eff)
	}
	cfg.EffCtx = eff
	if cfg.Port == "" {
		cfg.Port = "0"
	}
	srv := New(root, cfg)
	t.Cleanup(func() { _ = srv.Close() })
	if err := srv.LoadModules(paths); err != nil {
		t.Fatalf("LoadModules: %v", err)
	}
	eff.FnCallerN = srv.GetEngine().GetCallValueN()
	eff.FnCaller = srv.GetEngine().GetCallValue()
	if err := srv.ValidateWSRoutes(); err != nil {
		t.Fatalf("ValidateWSRoutes: %v", err)
	}
	ts := httptest.NewServer(srv.buildRoutes())
	t.Cleanup(ts.Close)
	return &wsFixture{srv: srv, http: ts, eff: eff}
}

func (f *wsFixture) wsURL(path string) string {
	return "ws" + strings.TrimPrefix(f.http.URL, "http") + path
}

// sameOrigin is the Origin a page served by this server would send.
func (f *wsFixture) sameOrigin() string { return f.http.URL }

// dial opens a browser connection with the given Origin (and subprotocols).
func (f *wsFixture) dial(t testing.TB, path, origin string, protocols ...string) (*websocket.Conn, *http.Response, error) {
	t.Helper()
	hdr := http.Header{}
	if origin != "" {
		hdr.Set("Origin", origin)
	}
	d := websocket.Dialer{Subprotocols: protocols, HandshakeTimeout: 5 * time.Second}
	c, resp, err := d.Dial(f.wsURL(path), hdr)
	if c != nil {
		t.Cleanup(func() { _ = c.Close() })
	}
	return c, resp, err
}

func (f *wsFixture) mustDial(t testing.TB, path string) *websocket.Conn {
	t.Helper()
	c, resp, err := f.dial(t, path, f.sameOrigin())
	if err != nil {
		status := 0
		if resp != nil {
			status = resp.StatusCode
		}
		t.Fatalf("dial %s: %v (HTTP %d)", path, err, status)
	}
	return c
}

type frame struct {
	bin  bool
	data string
}

func readFrames(t testing.TB, c *websocket.Conn, n int) []frame {
	t.Helper()
	_ = c.SetReadDeadline(time.Now().Add(10 * time.Second))
	var out []frame
	for len(out) < n {
		mt, data, err := c.ReadMessage()
		if err != nil {
			t.Fatalf("read after %d/%d frames: %v", len(out), n, err)
		}
		out = append(out, frame{bin: mt == websocket.BinaryMessage, data: string(data)})
	}
	return out
}

// upstream is a scripted fake of a BidiGenerateContent-shaped endpoint:
// it records the handshake headers and what it receives, and sends its
// script once the client's first (setup) message arrives.
type upstream struct {
	srv      *httptest.Server
	mu       sync.Mutex
	headers  http.Header
	received []frame
	script   []frame
	echo     bool // write every received frame back (benchmarks)
}

func newUpstream(t testing.TB, useTLS bool, script []frame) *upstream {
	return newUpstreamMode(t, useTLS, script, false)
}

// newUpstreamMode is newUpstream with echo mode chosen up front.
func newUpstreamMode(t testing.TB, useTLS bool, script []frame, echo bool) *upstream {
	t.Helper()
	u := &upstream{script: script, echo: echo}
	up := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u.mu.Lock()
		u.headers = r.Header.Clone()
		u.mu.Unlock()
		conn, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		first := true
		for {
			mt, data, err := conn.ReadMessage()
			if err != nil {
				return
			}
			if u.echo {
				if err := conn.WriteMessage(mt, data); err != nil {
					return
				}
				continue
			}
			u.mu.Lock()
			u.received = append(u.received, frame{bin: mt == websocket.BinaryMessage, data: string(data)})
			u.mu.Unlock()
			if first {
				first = false
				for _, f := range u.script {
					mt := websocket.TextMessage
					if f.bin {
						mt = websocket.BinaryMessage
					}
					_ = conn.WriteMessage(mt, []byte(f.data))
				}
			}
		}
	})
	if useTLS {
		u.srv = httptest.NewTLSServer(h)
	} else {
		u.srv = httptest.NewServer(h)
	}
	t.Cleanup(u.srv.Close)
	return u
}

func (u *upstream) url() string {
	if strings.HasPrefix(u.srv.URL, "https") {
		return "wss" + strings.TrimPrefix(u.srv.URL, "https") + "/ws/BidiGenerateContent"
	}
	return "ws" + strings.TrimPrefix(u.srv.URL, "http") + "/ws/BidiGenerateContent"
}

func (u *upstream) tlsConfig() *tls.Config {
	pool := x509.NewCertPool()
	pool.AddCert(u.srv.Certificate())
	return &tls.Config{RootCAs: pool}
}

func (u *upstream) got() ([]frame, http.Header) {
	u.mu.Lock()
	defer u.mu.Unlock()
	return append([]frame(nil), u.received...), u.headers
}
