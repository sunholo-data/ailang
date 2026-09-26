package apiserver

import (
	"bytes"
	"context"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/platform/streamcred"
	"github.com/sunholo-data/ailang/internal/trace"
)

// The M3 canary (DONE WHEN, part 2): a full relay session through serve-api
// to a TLS fake upstream whose credential is bound host-side. The planted
// token must reach the upstream (the binding fired) and appear in NONE of:
// the frames the client received, the deep trace file, stderr/the server log,
// and the per-frame decision log.

const wsCanary = "CANARY-ya29.c.b0AXv0zTOP-5e1f0c8a9d7b4e32"

type canarySource struct{}

func (canarySource) Token(context.Context) (string, error) { return wsCanary, nil }
func (canarySource) Describe() string                      { return "canary (test)" }

func TestWS_CredentialCanary(t *testing.T) {
	up := newUpstream(t, true, []frame{
		{data: `{"setupComplete":{}}`},
		{data: `{"serverContent":{"inputTranscription":{"text":"hi daneel"}}}`},
		{bin: true, data: "audio-1"},
		{data: `{"serverContent":{"note":"drop-me"}}`},
		{bin: true, data: "audio-2"},
	})
	u, _ := url.Parse(up.url())
	binder, err := streamcred.Binder([]streamcred.Binding{{Host: u.Hostname(), Port: u.Port(), Source: canarySource{}}})
	if err != nil {
		t.Fatal(err)
	}
	collector := trace.NewCollectorWithTier(trace.TierDeep)

	var clientFrames []frame
	logs := captureStderr(t, func() {
		f := newWSFixture(t, Config{WS: WSConfig{DecisionLog: true}}, up.url(), func(eff *effects.EffContext) {
			eff.Stream.SetTLSClientConfig(up.tlsConfig())
			eff.Stream.Credentials = binder
			eff.Trace = collector
		}, "relay")
		c := f.mustDial(t, "/relay")
		clientFrames = readFrames(t, c, 4)
		_ = c.WriteMessage(websocket.TextMessage, []byte(`{"realtimeInput":{"text":"hello"}}`))
		waitFor(t, func() bool { r, _ := up.got(); return len(r) == 2 })
		_ = c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "done"))
		waitFor(t, func() bool { f.srv.ws.mu.Lock(); defer f.srv.ws.mu.Unlock(); return len(f.srv.ws.sessions) == 0 })
	})

	// Positive controls: the binding fired, and every sink we search is live.
	if _, hdr := up.got(); hdr.Get("Authorization") != "Bearer "+wsCanary {
		t.Fatalf("instrument check: upstream did not receive the bound credential (got %q)", hdr.Get("Authorization"))
	}
	tracePath := filepath.Join(t.TempDir(), "session.trace.jsonl")
	var traceBuf bytes.Buffer
	if err := trace.WriteJSONL(&traceBuf, collector.Events()); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tracePath, traceBuf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	traceFile, _ := os.ReadFile(tracePath)
	if !bytes.Contains(traceFile, []byte("connect")) || !bytes.Contains(traceFile, []byte("bridge")) {
		t.Fatalf("instrument check: the deep trace must record the connect and bridge effects:\n%s", traceFile)
	}
	decisions := decisionLines(logs)
	if strings.Count(decisions, "\n") < 5 {
		t.Fatalf("instrument check: expected a decision line per frame, got:\n%s", decisions)
	}
	if len(clientFrames) != 4 {
		t.Fatalf("instrument check: client frames %v", clientFrames)
	}

	// The canary: zero hits in all four sinks.
	for i, fr := range clientFrames {
		if strings.Contains(fr.data, wsCanary) || strings.Contains(fr.data, "ya29") {
			t.Errorf("client frame %d carries the credential: %q", i, fr.data)
		}
	}
	if bytes.Contains(traceFile, []byte(wsCanary)) {
		t.Errorf("deep trace file carries the credential")
	}
	if strings.Contains(logs, wsCanary) {
		t.Errorf("stderr / server log carries the credential")
	}
	if strings.Contains(decisions, wsCanary) {
		t.Errorf("decision log carries the credential")
	}
	// And the decision log carries no payloads either.
	for _, payload := range []string{"setupComplete", "daneel", "audio-1", "realtimeInput"} {
		if strings.Contains(decisions, payload) {
			t.Errorf("decision log carries payload %q", payload)
		}
	}
}

// decisionLines extracts the [ws-bridge] decision lines from the log.
func decisionLines(logs string) string {
	var b strings.Builder
	for _, line := range strings.Split(logs, "\n") {
		if strings.Contains(line, "[ws-bridge] ") {
			b.WriteString(line)
			b.WriteByte('\n')
		}
	}
	return b.String()
}
