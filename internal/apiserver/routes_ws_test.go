package apiserver

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/sunholo-data/ailang/internal/effects"
)

// M1-A1: a WS route whose handler uses only the pre-existing Stream ops
// round-trips text AND binary frames.
func TestWS_EchoRoundTripsTextAndBinary(t *testing.T) {
	f := newWSFixture(t, Config{}, "", nil, "echo")
	c := f.mustDial(t, "/echo")
	if err := c.WriteMessage(websocket.TextMessage, []byte("hello")); err != nil {
		t.Fatal(err)
	}
	if err := c.WriteMessage(websocket.BinaryMessage, []byte{0x00, 0xff, 0x10, 0x80}); err != nil {
		t.Fatal(err)
	}
	got := readFrames(t, c, 2)
	if got[0] != (frame{data: "hello"}) || got[1] != (frame{bin: true, data: "\x00\xff\x10\x80"}) {
		t.Fatalf("echo = %+v", got)
	}
}

// M1-A2: plain GET 426, bad Origin 403, over-cap session 503 — all BEFORE the
// upgrade (the handler never runs).
func TestWS_RefusedBeforeUpgrade(t *testing.T) {
	f := newWSFixture(t, Config{WS: WSConfig{MaxSessions: 1}}, "", nil, "echo")

	resp, err := http.Get(f.http.URL + "/echo")
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusUpgradeRequired || resp.Header.Get("Upgrade") != "websocket" {
		t.Fatalf("plain GET: %d (Upgrade=%q), want 426", resp.StatusCode, resp.Header.Get("Upgrade"))
	}

	for name, origin := range map[string]string{
		"missing": "",
		"null":    "null",
		"foreign": "https://evil.example",
		"port":    strings.Replace(f.sameOrigin(), "127.0.0.1:", "127.0.0.1:1", 1),
	} {
		_, resp, err := f.dial(t, "/echo", origin)
		if err == nil || resp == nil || resp.StatusCode != http.StatusForbidden {
			t.Errorf("origin %s (%q): want 403, got %v %v", name, origin, resp, err)
		}
	}

	c := f.mustDial(t, "/echo") // takes the only session
	_ = c.WriteMessage(websocket.TextMessage, []byte("x"))
	readFrames(t, c, 1) // session is live inside the handler
	_, resp, err = f.dial(t, "/echo", f.sameOrigin())
	if err == nil || resp == nil || resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("second session: want 503, got %v %v", resp, err)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "--ws-max-sessions 1") {
		t.Fatalf("503 body should name the flag: %s", body)
	}
}

// --cors-origin entries are the WS Origin allowlist.
func TestWS_AllowlistedOriginAccepted(t *testing.T) {
	f := newWSFixture(t, Config{CORSOrigins: []string{"https://studio.tailnet.ts.net"}}, "", nil, "echo")
	if _, resp, err := f.dial(t, "/echo", "https://studio.tailnet.ts.net"); err != nil {
		t.Fatalf("allowlisted origin refused: %v %v", resp, err)
	}
}

// M1-A3: a WS export is on no HTTP surface: not in OpenAPI, not in MCP
// tools/list, not callable through the catch-all.
func TestWS_ExcludedFromHTTPAndMCPSurfaces(t *testing.T) {
	f := newWSFixture(t, Config{}, "", nil, "echo")

	resp, err := http.Get(f.http.URL + "/api/_meta/openapi.json")
	if err != nil {
		t.Fatal(err)
	}
	spec, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if !strings.Contains(string(spec), "/api/v1/inc") {
		t.Fatal("instrument check: the HTTP route must be in the spec")
	}
	for _, leak := range []string{`"/echo"`, `"/probe"`, "ws/echo.echo", "ws/echo.probe"} {
		if strings.Contains(string(spec), leak) {
			t.Errorf("OpenAPI spec contains WS route %q", leak)
		}
	}

	resp, err = http.Post(f.http.URL+"/api/ws/echo/echo", "application/json", strings.NewReader(`{"args":[1]}`))
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		t.Error("WS export callable through the /api catch-all")
	}

	ms := NewMCPServer(f.srv)
	ctx := context.Background()
	st, ct := mcp.NewInMemoryTransports()
	ss, err := ms.mcpServer.Connect(ctx, st, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer ss.Close()
	cs, err := mcp.NewClient(&mcp.Implementation{Name: "t", Version: "0"}, nil).Connect(ctx, ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer cs.Close()
	tools, err := cs.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	names := toolNames(tools.Tools)
	if !strings.Contains(strings.Join(names, ","), "inc") {
		t.Fatalf("instrument check: inc missing from tools/list: %v", names)
	}
	for _, n := range names {
		if n == "echo" || n == "probe" {
			t.Errorf("WS export %q in MCP tools/list", n)
		}
	}
}

// M1-A4 (G6): two concurrent sessions cannot see each other's connections.
// Each session gets its own StreamContext, so the probe sees its own client
// as connection 1 and nothing at 2 — although session A is still open. With
// one shared context the probe's client would be connection 2 and A's
// connection 1 would be visible ("Open Open").
func TestWS_SessionsAreIsolated(t *testing.T) {
	f := newWSFixture(t, Config{}, "", nil, "echo")
	a := f.mustDial(t, "/echo")
	_ = a.WriteMessage(websocket.TextMessage, []byte("a"))
	readFrames(t, a, 1)

	b := f.mustDial(t, "/probe?x=1")
	got := readFrames(t, b, 1)
	if got[0].data != "Open StreamClosed /probe?x=1" {
		t.Fatalf("probe = %q, want %q", got[0].data, "Open StreamClosed /probe?x=1")
	}
	if n := f.eff.Stream.ConnectionCount(); n != 0 {
		t.Fatalf("server-wide Stream context holds %d connections; sessions must use their own", n)
	}
}

// Startup refuses WS routes it cannot serve safely or correctly.
func TestWS_ValidateRoutes(t *testing.T) {
	f := newWSFixture(t, Config{}, "", nil, "echo")

	f.srv.bind = "0.0.0.0"
	err := f.srv.ValidateWSRoutes()
	if err == nil || !strings.Contains(err.Error(), "not loopback") || !strings.Contains(err.Error(), "--cors-origin") {
		t.Fatalf("non-loopback bind without allowlist: %v", err)
	}
	f.srv.corsOrigins = originSet([]string{"https://studio.example"})
	if err := f.srv.ValidateWSRoutes(); err != nil {
		t.Fatalf("non-loopback bind WITH allowlist: %v", err)
	}
	for _, h := range []string{"127.0.0.1", "::1", "localhost"} {
		if !isLoopbackHost(h) {
			t.Errorf("%s should be loopback", h)
		}
	}

	bad := RouteEntry{Module: "m", Function: "f", IsRaw: true, ParamTypes: []string{"string"}, Effects: []string{"IO"}}
	probs := strings.Join(f.srv.wsRouteProblems(bad), "\n")
	for _, want := range []string{"@raw/@nowrap", "takes (client: StreamConn)", "effect IO is not granted", "must include Stream"} {
		if !strings.Contains(probs, want) {
			t.Errorf("problems %q missing %q", probs, want)
		}
	}
}

// With an API key configured, a browser presents it as a subprotocol entry;
// the server selects ailang.v1 and never echoes the key.
func TestWS_APIKeyViaSubprotocol(t *testing.T) {
	t.Setenv("WS_TEST_KEY", "k-123")
	f := newWSFixture(t, Config{APIKeyHeader: "x-api-key", APIKeyEnv: "WS_TEST_KEY"}, "", nil, "echo")

	if _, resp, err := f.dial(t, "/echo", f.sameOrigin()); err == nil || resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("no key: want 401, got %v %v", resp, err)
	}
	if _, resp, err := f.dial(t, "/echo", f.sameOrigin(), "ailang.v1", "ailang.key.wrong"); err == nil || resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("wrong key: want 401, got %v %v", resp, err)
	}
	c, resp, err := f.dial(t, "/echo", f.sameOrigin(), "ailang.v1", "ailang.key.k-123")
	if err != nil {
		t.Fatalf("key subprotocol: %v %v", resp, err)
	}
	if c.Subprotocol() != "ailang.v1" || strings.Contains(resp.Header.Get("Sec-WebSocket-Protocol"), "k-123") {
		t.Fatalf("selected %q (header %q)", c.Subprotocol(), resp.Header.Get("Sec-WebSocket-Protocol"))
	}
}

// Shutdown sends 1001 on live sessions (Shutdown does not close hijacked
// connections by itself).
func TestWS_ShutdownSendsGoingAway(t *testing.T) {
	f := newWSFixture(t, Config{}, "", nil, "echo")
	c := f.mustDial(t, "/echo")
	_ = c.WriteMessage(websocket.TextMessage, []byte("x"))
	readFrames(t, c, 1)
	f.srv.closeWSSessions()
	_ = c.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, _, err := c.ReadMessage()
	var ce *websocket.CloseError
	if !errors.As(err, &ce) || ce.Code != websocket.CloseGoingAway {
		t.Fatalf("want close 1001, got %v", err)
	}
}

// DONE WHEN (part 1): an AILANG function decides each upstream->client
// frame of a relayed BidiGenerateContent-shaped session.
func TestWS_RelayAILANGStepDecidesEachFrame(t *testing.T) {
	up := newUpstream(t, false, []frame{
		{data: `{"setupComplete":{}}`},
		{bin: true, data: "audio-before"},
		{data: `{"serverContent":{"note":"drop-me"}}`},
		{data: `{"serverContent":{"inputTranscription":{"text":"ok daneel"}}}`},
		{bin: true, data: "audio-after"},
	})
	f := newWSFixture(t, Config{}, up.url(), nil, "relay")
	c := f.mustDial(t, "/relay")
	got := readFrames(t, c, 3)
	want := []frame{{data: `{"setupComplete":{}}`}, {data: `{"addressed":true}`}, {bin: true, data: "audio-after"}}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("frame %d = %+v, want %+v (all: %+v)", i, got[i], want[i], got)
		}
	}
	if err := c.WriteMessage(websocket.TextMessage, []byte(`{"realtimeInput":{}}`)); err != nil {
		t.Fatal(err)
	}
	waitFor(t, func() bool { r, _ := up.got(); return len(r) == 2 })
	rec, _ := up.got()
	if rec[0].data != `{"setup":{"model":"fake"}}` || rec[1].data != `{"realtimeInput":{}}` {
		t.Fatalf("upstream received %+v", rec)
	}
}

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatal("condition not reached")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

var _ = effects.BridgeDecision{}
