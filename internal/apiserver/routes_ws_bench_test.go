package apiserver

import (
	"bytes"
	"net/http"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// M5 measurement (M-SERVEAPI-WS-BRIDGE): the latency a relayed frame pays
// for going through serve-api and an AILANG step, against the same echo
// upstream reached directly. One iteration = one 22 KB frame client ->
// bridge -> upstream (echo) -> bridge -> client, i.e. TWO AILANG step calls
// and two relay hops. Recorded, not gated: shared CI runners are too noisy.
//
//	go test ./internal/apiserver/ -run '^$' -bench 'WSRelay' -benchtime 2000x

func BenchmarkWSRelay_Direct(b *testing.B) {
	up := newUpstreamMode(b, false, nil, true)
	c, _, err := websocket.DefaultDialer.Dial(up.url(), nil)
	if err != nil {
		b.Fatal(err)
	}
	defer c.Close()
	measureRTT(b, c, websocket.BinaryMessage, 22*1024)
}

func BenchmarkWSRelay_AILANGGateBinary(b *testing.B) {
	benchRelay(b, websocket.BinaryMessage)
}

func BenchmarkWSRelay_AILANGGateText(b *testing.B) {
	benchRelay(b, websocket.TextMessage)
}

func benchRelay(b *testing.B, kind int) {
	up := newUpstreamMode(b, false, nil, true)
	f := newWSFixture(b, Config{}, up.url(), nil, "relay")
	hdr := http.Header{"Origin": {f.sameOrigin()}}
	c, _, err := websocket.DefaultDialer.Dial(f.wsURL("/relay"), hdr)
	if err != nil {
		b.Fatal(err)
	}
	defer c.Close()
	// The setup echo arrives first; then address the gate so model audio
	// (UpBin) is forwarded rather than dropped.
	if err := c.WriteMessage(websocket.TextMessage, []byte("hey daneel")); err != nil {
		b.Fatal(err)
	}
	readFrames(b, c, 2)
	measureRTT(b, c, kind, 22*1024)
}

func measureRTT(b *testing.B, c *websocket.Conn, kind, size int) {
	payload := bytes.Repeat([]byte("a"), size)
	if kind == websocket.TextMessage {
		payload = []byte(strings.Repeat(`{"serverContent":{"modelTurn":{"parts":[{"text":"x"}]}}} `, size/56+1)[:size])
	}
	rtts := make([]time.Duration, 0, b.N)
	b.SetBytes(int64(size))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		start := time.Now()
		if err := c.WriteMessage(kind, payload); err != nil {
			b.Fatal(err)
		}
		_ = c.SetReadDeadline(time.Now().Add(10 * time.Second))
		_, got, err := c.ReadMessage()
		if err != nil {
			b.Fatal(err)
		}
		if len(got) != size {
			b.Fatalf("frame %d: got %d bytes, want %d", i, len(got), size)
		}
		rtts = append(rtts, time.Since(start))
	}
	b.StopTimer()
	sort.Slice(rtts, func(i, j int) bool { return rtts[i] < rtts[j] })
	pct := func(p float64) float64 { return float64(rtts[int(p*float64(len(rtts)-1))].Microseconds()) }
	b.ReportMetric(pct(0.50), "p50-rtt-us")
	b.ReportMetric(pct(0.95), "p95-rtt-us")
}
