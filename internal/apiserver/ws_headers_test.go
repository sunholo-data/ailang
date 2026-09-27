package apiserver

import (
	"bytes"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/trace"
)

// M-SERVEAPI-OPERATOR-SURFACE D7: --ws-pass-header.

func TestValidatePassHeaders(t *testing.T) {
	got, err := ValidatePassHeaders([]string{"Tailscale-User-Login", "tailscale-user-login", "X-Forwarded-For"}, "")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(got, ",") != "tailscale-user-login,x-forwarded-for" {
		t.Fatalf("want lower-cased, de-duplicated, flag order; got %v", got)
	}
	refused := []struct {
		name, apiKeyHeader string
	}{
		{"Authorization", ""},
		{"proxy-authorization", ""},
		{"Cookie", ""},
		{"Sec-WebSocket-Protocol", ""},
		{"Sec-WebSocket-Key", ""},
		{"X-Api-Key", "x-api-key"}, // the configured --api-key-header
		{"", ""},
		{"Bad Header", ""},
		{"x:y", ""},
	}
	for _, c := range refused {
		if _, err := ValidatePassHeaders([]string{c.name}, c.apiKeyHeader); err == nil {
			t.Errorf("--ws-pass-header %q (api key header %q) must be refused", c.name, c.apiKeyHeader)
		}
	}
}

// dialWithHeaders opens a same-origin browser connection carrying extra
// request headers; values of a repeated name arrive as separate lines.
func dialWithHeaders(t *testing.T, f *wsFixture, path string, extra http.Header) *websocket.Conn {
	t.Helper()
	hdr := http.Header{}
	for k, vs := range extra {
		for _, v := range vs {
			hdr.Add(k, v)
		}
	}
	hdr.Set("Origin", f.sameOrigin())
	d := websocket.Dialer{HandshakeTimeout: 5 * time.Second}
	c, resp, err := d.Dial(f.wsURL(path), hdr)
	if err != nil {
		status := 0
		if resp != nil {
			status = resp.StatusCode
		}
		t.Fatalf("dial %s: %v (HTTP %d)", path, err, status)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c
}

func readText(t *testing.T, c *websocket.Conn) string {
	t.Helper()
	_ = c.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, b, err := c.ReadMessage()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	return string(b)
}

const headerCanary = "CANARY-cookie-5e1f0c8a9d7b4e32"

func browserHeaders() http.Header {
	return http.Header{
		"Tailscale-User-Login": {"mark@example.com"},
		"X-Multi":              {"one", "two, three"},
		"X-Unlisted":           {"never"},
		"Cookie":               {"session=" + headerCanary},
		"Authorization":        {"Bearer " + headerCanary},
	}
}

func TestWS_PassHeaderReachesHandlerExactly(t *testing.T) {
	cfg := Config{WS: WSConfig{PassHeaders: []string{"tailscale-user-login", "x-multi", "x-absent"}}}
	f := newWSFixture(t, cfg, "", nil, "headers")

	c := dialWithHeaders(t, f, "/who", browserHeaders())
	got := readText(t, c)
	want := "/who|tailscale-user-login=mark@example.com;x-multi=one;x-multi=two, three;"
	if got != want {
		t.Fatalf("handler saw %q\nwant          %q", got, want)
	}
}

func TestWS_NoPassHeaderMeansEmptyList(t *testing.T) {
	f := newWSFixture(t, Config{}, "", nil, "headers")
	c := dialWithHeaders(t, f, "/who", browserHeaders())
	if got := readText(t, c); got != "/who|" {
		t.Fatalf("without --ws-pass-header the handler must see no headers, got %q", got)
	}
}

func TestWS_ReqRecordIsExactlyTheDeclaredFields(t *testing.T) {
	cfg := Config{WS: WSConfig{PassHeaders: []string{"tailscale-user-login"}}}
	f := newWSFixture(t, cfg, "", nil, "headers")

	if got := readText(t, dialWithHeaders(t, f, "/query?x=1", browserHeaders())); got != "q=x=1" {
		t.Fatalf("one-field req: got %q", got)
	}
	if got := readText(t, dialWithHeaders(t, f, "/legacy?y=2", browserHeaders())); got != "/legacy?y=2" {
		t.Fatalf("three-field req: got %q", got)
	}
}

// The session-level test above passes on the tree-walking evaluator whatever
// the record carries; this pins the builder itself to the declared fields.
func TestWSReqRecord_OnlyDeclaredFields(t *testing.T) {
	r, _ := http.NewRequest(http.MethodGet, "http://x/who?a=1", nil)
	r.Header.Set("Origin", "http://x")
	r.Header.Set("Tailscale-User-Login", "mark@example.com")
	rec := wsReqRecord(r, []string{"query"}, []string{"tailscale-user-login"})
	if len(rec.Fields) != 1 || rec.Fields["query"] == nil {
		t.Fatalf("declared {query} must yield exactly {query}, got %v", rec.Fields)
	}
	rec = wsReqRecord(r, legacyWSReqFields, nil)
	if len(rec.Fields) != 3 || rec.Fields["headers"] != nil {
		t.Fatalf("legacy shape must stay {origin, path, query}, got %v", rec.Fields)
	}
}

func TestWS_ReqShapeValidated(t *testing.T) {
	cases := map[string]string{
		"unknown field": `req: { path: string, cookie: string }`,
		"headers type":  `req: { headers: [string] }`,
		"headers shape": `req: { headers: [{ name: string }] }`,
	}
	for name, param := range cases {
		t.Run(name, func(t *testing.T) {
			src := "module ws/bad\n\nimport std/stream (transmit, StreamConn)\n\n@route(\"WS\", \"/bad\")\nexport func bad(client: StreamConn, " +
				param + ") -> unit ! {Stream} {\n  let _ = transmit(client, \"x\");\n  ()\n}\n"
			err := loadWSSource(t, src)
			if err == nil || !strings.Contains(err.Error(), "headers: [{name: string, value: string}]") {
				t.Fatalf("want a startup refusal naming the accepted shape, got %v", err)
			}
		})
	}
}

// loadWSSource loads one WS module and returns ValidateWSRoutes' verdict.
func loadWSSource(t *testing.T, src string) error {
	t.Helper()
	repoRoot, _ := filepath.Abs(filepath.Join("..", ".."))
	t.Setenv("AILANG_STDLIB_PATH", filepath.Join(repoRoot, "std"))
	root := t.TempDir()
	p := filepath.Join(root, "ws", "bad.ail")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	eff := effects.NewEffContext(nil)
	eff.Grant(effects.NewCapability("Stream"))
	eff.Stream = effects.NewStreamContext()
	srv := New(root, Config{Port: "0", EffCtx: eff})
	t.Cleanup(func() { _ = srv.Close() })
	if err := srv.LoadModules([]string{p}); err != nil {
		t.Fatalf("LoadModules: %v", err)
	}
	return srv.ValidateWSRoutes()
}

// The M-SERVEAPI-WS-BRIDGE canary, extended to request headers: a secret in
// Cookie and Authorization never reaches the handler, the deep trace or the
// server log, while the operator-named header does (positive control).
func TestWS_PassHeaderCanary(t *testing.T) {
	collector := trace.NewCollectorWithTier(trace.TierDeep)
	var got string
	logs := captureStderr(t, func() {
		cfg := Config{WS: WSConfig{PassHeaders: []string{"tailscale-user-login"}, DecisionLog: true}}
		f := newWSFixture(t, cfg, "", func(eff *effects.EffContext) { eff.Trace = collector }, "headers")
		got = readText(t, dialWithHeaders(t, f, "/who", browserHeaders()))
	})
	if !strings.Contains(got, "mark@example.com") {
		t.Fatalf("instrument check: the named header must reach the handler, got %q", got)
	}
	var traceBuf bytes.Buffer
	if err := trace.WriteJSONL(&traceBuf, collector.Events()); err != nil {
		t.Fatal(err)
	}
	if traceBuf.Len() == 0 {
		t.Fatal("instrument check: the deep trace recorded nothing")
	}
	for sink, content := range map[string]string{"handler": got, "trace": traceBuf.String(), "log": logs} {
		if strings.Contains(content, headerCanary) {
			t.Errorf("canary leaked into the %s", sink)
		}
		if strings.Contains(content, "never") && sink == "handler" {
			t.Errorf("an unlisted header reached the handler: %q", content)
		}
	}
}
