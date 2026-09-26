package effects_test

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
)

// M2-A1: a Forward-all step relays 1,000 mixed text and binary frames each
// way, byte-identical and in order.
func TestBridge_ForwardAllRelaysMixedFramesInOrder(t *testing.T) {
	const n = 1000
	frames := func(prefix string) []wireFrame {
		out := make([]wireFrame, n)
		for i := range out {
			if i%3 == 0 {
				out[i] = wireFrame{Binary: true, Data: []byte{byte(i), byte(i >> 8), 0x00, 0xff, prefix[0]}}
			} else {
				out[i] = wireFrame{Data: []byte(fmt.Sprintf("%s-%d", prefix, i))}
			}
		}
		return out
	}
	downFrames, upFrames := frames("down"), frames("up")
	up := newFakeUpstream(t, false, downFrames)
	h := newBridgeHarness(t, up, nil)
	done := h.run(&eval.IntValue{Value: 0}, forwardAll)

	sendErr := make(chan error, 1)
	go func() {
		for _, f := range upFrames {
			mt := websocket.TextMessage
			if f.Binary {
				mt = websocket.BinaryMessage
			}
			if err := h.browser.WriteMessage(mt, f.Data); err != nil {
				sendErr <- err
				return
			}
		}
		sendErr <- nil
	}()
	got := readBrowser(t, h.browser, n)
	if err := <-sendErr; err != nil {
		t.Fatalf("browser send: %v", err)
	}
	assertFrames(t, "upstream->client", got, downFrames)

	deadline := time.Now().Add(10 * time.Second)
	for {
		rec, _ := up.got()
		if len(rec) == n || time.Now().After(deadline) {
			assertFrames(t, "client->upstream", rec, upFrames)
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	_ = h.browser.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "bye"))
	r := waitBridge(t, done)
	if r.end.CtorName != "ClientClosed" {
		t.Fatalf("end = %v, want ClientClosed", r.end)
	}
	if got := r.state.(*eval.IntValue).Value; got != 2*n {
		t.Fatalf("step ran %d times, want %d (once per data frame)", got, 2*n)
	}
}

func assertFrames(t *testing.T, dir string, got, want []wireFrame) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s: got %d frames, want %d", dir, len(got), len(want))
	}
	for i := range want {
		if got[i].Binary != want[i].Binary || !bytes.Equal(got[i].Data, want[i].Data) {
			t.Fatalf("%s: frame %d = %v %q, want %v %q", dir, i, got[i].Binary, got[i].Data, want[i].Binary, want[i].Data)
		}
	}
}

// gateStep is the speech gate in Go: model audio (UpBin) reaches the client
// only while the user has addressed the assistant.
func gateStep(state, frame eval.Value) (eval.Value, eval.Value) {
	addressed := state.(*eval.BoolValue).Value
	f := frame.(*eval.TaggedValue)
	switch f.CtorName {
	case "UpBin":
		if addressed {
			return state, verdict("Forward")
		}
		return state, verdict("Drop")
	case "UpText":
		msg := f.Fields[0].(*eval.StringValue).Value
		switch {
		case strings.Contains(msg, "inputTranscription") && strings.Contains(strings.ToLower(msg), "daneel"):
			return &eval.BoolValue{Value: true}, verdict("Forward")
		case strings.Contains(msg, "turnComplete"):
			return &eval.BoolValue{Value: false}, verdict("Forward")
		case strings.Contains(msg, "modelTurn") && !addressed:
			return state, verdict("Drop")
		}
	}
	return state, verdict("Forward")
}

// M2-A2/A3: the speech-gate fixture delivers exactly the expected frames,
// a strict subset of what the upstream sent (the non-trivial verdict the
// 2026-02 rejection asked for).
func TestBridge_SpeechGateDeliversStrictSubset(t *testing.T) {
	bin := func(b byte) wireFrame { return wireFrame{Binary: true, Data: []byte{b, b, b}} }
	txt := func(s string) wireFrame { return wireFrame{Data: []byte(s)} }
	script := []wireFrame{
		txt(`{"serverContent":{"modelTurn":{"parts":[]}}}`),
		bin(1), bin(2), bin(3),
		txt(`{"serverContent":{"inputTranscription":{"text":"hey Daneel"}}}`),
		bin(4), bin(5),
		txt(`{"serverContent":{"turnComplete":true}}`),
		bin(6), bin(7),
	}
	want := []wireFrame{script[4], script[5], script[6], script[7]}
	up := newFakeUpstream(t, false, script)
	h := newBridgeHarness(t, up, nil)
	done := h.run(&eval.BoolValue{Value: false}, gateStep)

	got := readBrowser(t, h.browser, len(want))
	assertFrames(t, "gate", got, want)
	if len(want) >= len(script) {
		t.Fatal("fixture must be a strict subset")
	}
	// Nothing else may follow: the last two frames were dropped.
	_ = h.browser.SetReadDeadline(time.Now().Add(300 * time.Millisecond))
	if _, data, err := h.browser.ReadMessage(); err == nil {
		t.Fatalf("unexpected extra frame %q", data)
	}
	h.ctx.Stream.CloseAll()
	waitBridge(t, done)
}

// M2-A4: barge-in. The step runs at DEQUEUE, so after an interruption frame
// every model frame already queued behind it is dropped: 0 stale frames reach
// the client, and fresh audio after the next turn start does.
func TestBridge_BargeInDropsQueuedStaleFrames(t *testing.T) {
	script := []wireFrame{{Data: []byte(`{"serverContent":{"interrupted":true}}`)}}
	for i := 0; i < 50; i++ {
		script = append(script, wireFrame{Binary: true, Data: []byte("stale")})
	}
	script = append(script, wireFrame{Data: []byte(`{"turnStart":true}`)})
	for i := 0; i < 3; i++ {
		script = append(script, wireFrame{Binary: true, Data: []byte(fmt.Sprintf("fresh-%d", i))})
	}
	up := newFakeUpstream(t, false, script)
	h := newBridgeHarness(t, up, nil)
	step := func(state, frame eval.Value) (eval.Value, eval.Value) {
		interrupted := state.(*eval.BoolValue).Value
		f := frame.(*eval.TaggedValue)
		if f.CtorName == "UpText" {
			msg := f.Fields[0].(*eval.StringValue).Value
			if strings.Contains(msg, "interrupted") {
				return &eval.BoolValue{Value: true}, verdict("Forward")
			}
			if strings.Contains(msg, "turnStart") {
				return &eval.BoolValue{Value: false}, verdict("Forward")
			}
		}
		if f.CtorName == "UpBin" && interrupted {
			return state, verdict("Drop")
		}
		return state, verdict("Forward")
	}
	done := h.run(&eval.BoolValue{Value: false}, step)
	got := readBrowser(t, h.browser, 5)
	for i, f := range got {
		if bytes.Equal(f.Data, []byte("stale")) {
			t.Fatalf("stale frame reached the client at position %d", i)
		}
	}
	if string(got[4].Data) != "fresh-2" {
		t.Fatalf("last frame = %q, want fresh-2", got[4].Data)
	}
	h.ctx.Stream.CloseAll()
	waitBridge(t, done)
}

// M2-A5: a step that raises ends the bridge with StepFailed, and the client
// sees close 1011.
func TestBridge_StepFailureClosesClient1011(t *testing.T) {
	up := newFakeUpstream(t, false, []wireFrame{{Data: []byte("hello")}})
	h := newBridgeHarness(t, up, nil)
	h.ctx.FnCallerN = func(eval.Value, []eval.Value) (eval.Value, error) {
		return nil, errors.New("division by zero")
	}
	out := make(chan bridgeResult, 1)
	go func() {
		res, err := effects.StreamBridge(h.ctx, []eval.Value{streamConn(h.clientID), streamConn(h.upID), &eval.UnitValue{}, &eval.UnitValue{}})
		r := bridgeResult{err: err}
		if tup, ok := res.(*eval.TupleValue); ok {
			r.end = tup.Elements[1].(*eval.TaggedValue)
		}
		out <- r
	}()
	r := waitBridge(t, out)
	if r.end.CtorName != "StepFailed" || !strings.Contains(r.end.Fields[0].(*eval.StringValue).Value, "division by zero") {
		t.Fatalf("end = %v, want StepFailed(...division by zero...)", r.end)
	}
	assertBrowserClose(t, h.browser, 1011)
	select {
	case <-up.gone:
	case <-time.After(3 * time.Second):
		t.Fatal("upstream was not closed")
	}
}

func assertBrowserClose(t *testing.T, c *websocket.Conn, code int) string {
	t.Helper()
	_ = c.SetReadDeadline(time.Now().Add(5 * time.Second))
	for {
		_, _, err := c.ReadMessage()
		if err == nil {
			continue
		}
		var ce *websocket.CloseError
		if !errors.As(err, &ce) {
			t.Fatalf("browser read error %v, want close %d", err, code)
		}
		if ce.Code != code {
			t.Fatalf("browser close code %d (%q), want %d", ce.Code, ce.Text, code)
		}
		return ce.Text
	}
}

// M2-A6: a client close mid-stream returns ClientClosed within 100 ms and
// closes the upstream.
func TestBridge_ClientCloseEndsQuicklyAndClosesUpstream(t *testing.T) {
	up := newFakeUpstream(t, false, nil)
	h := newBridgeHarness(t, up, nil)
	done := h.run(&eval.IntValue{Value: 0}, forwardAll)
	if err := h.browser.WriteMessage(websocket.TextMessage, []byte("mid-stream")); err != nil {
		t.Fatal(err)
	}
	time.Sleep(50 * time.Millisecond)
	closedAt := time.Now()
	_ = h.browser.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseGoingAway, "tab closed"))
	r := waitBridge(t, done)
	if r.end.CtorName != "ClientClosed" {
		t.Fatalf("end = %v, want ClientClosed", r.end)
	}
	if code := r.end.Fields[0].(*eval.IntValue).Value; code != 1001 {
		t.Fatalf("ClientClosed code = %d, want 1001", code)
	}
	if lag := r.at.Sub(closedAt); lag > 100*time.Millisecond {
		t.Fatalf("bridge returned %v after the client closed, want < 100ms", lag)
	}
	select {
	case <-up.gone:
	case <-time.After(3 * time.Second):
		t.Fatal("upstream was not closed")
	}
	if _, ok := h.ctx.Stream.GetConnection(h.upID); ok {
		t.Fatal("upstream still registered after the bridge returned")
	}
}

// CloseBridge closes the client with the step's code and reason; an invalid
// code is a StepFailed, not a quiet substitution.
func TestBridge_CloseVerdict(t *testing.T) {
	cases := []struct {
		name     string
		code     int
		wantEnd  string
		wantCode int
	}{
		{"policy violation", 1008, "ClosedByVerdict", 1008},
		{"reserved code refused", 1005, "StepFailed", 1011},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			up := newFakeUpstream(t, false, nil)
			h := newBridgeHarness(t, up, nil)
			done := h.run(&eval.UnitValue{}, func(state, frame eval.Value) (eval.Value, eval.Value) {
				return state, verdict("CloseBridge", &eval.IntValue{Value: tc.code}, &eval.StringValue{Value: "setup is server-owned"})
			})
			_ = h.browser.WriteMessage(websocket.TextMessage, []byte(`{"setup":{}}`))
			r := waitBridge(t, done)
			if r.end.CtorName != tc.wantEnd {
				t.Fatalf("end = %v, want %s", r.end, tc.wantEnd)
			}
			reason := assertBrowserClose(t, h.browser, tc.wantCode)
			if tc.wantEnd == "ClosedByVerdict" && reason != "setup is server-owned" {
				t.Fatalf("close reason %q", reason)
			}
			if frames, _ := up.got(); len(frames) != 0 {
				t.Fatalf("closed frame reached the upstream: %v", frames)
			}
		})
	}
}

// Replace verdicts send the replacement, as text or binary regardless of the
// input kind; UpstreamClosed leaves the client open for the handler.
func TestBridge_ReplaceAndUpstreamClosedKeepsClient(t *testing.T) {
	up := newFakeUpstream(t, false, []wireFrame{{Data: []byte("secret-ish text")}, {Binary: true, Data: []byte{9, 9}}})
	up.closeAfter = true
	h := newBridgeHarness(t, up, nil)
	done := h.run(&eval.UnitValue{}, func(state, frame eval.Value) (eval.Value, eval.Value) {
		if frame.(*eval.TaggedValue).CtorName == "UpText" {
			return state, verdict("ReplaceBin", &eval.BytesValue{Value: []byte{1, 2, 3}})
		}
		return state, verdict("ReplaceText", &eval.StringValue{Value: "was binary"})
	})
	got := readBrowser(t, h.browser, 2)
	assertFrames(t, "replace", got, []wireFrame{{Binary: true, Data: []byte{1, 2, 3}}, {Data: []byte("was binary")}})
	r := waitBridge(t, done)
	if r.end.CtorName != "UpstreamClosed" || r.end.Fields[1].(*eval.StringValue).Value != "script done" {
		t.Fatalf("end = %v, want UpstreamClosed(1000, script done)", r.end)
	}
	res, err := effects.StreamSend(h.ctx, []eval.Value{streamConn(h.clientID), &eval.StringValue{Value: "still here"}})
	if err != nil || res.(*eval.TaggedValue).CtorName != "Ok" {
		t.Fatalf("client should stay open after UpstreamClosed: %v %v", res, err)
	}
	if f := readBrowser(t, h.browser, 1); string(f[0].Data) != "still here" {
		t.Fatalf("got %q", f[0].Data)
	}
}

// Every data frame charges Stream.recv and every frame sent charges
// Stream.send; an exhausted budget ends the bridge typed, not as a Go error.
func TestBridge_ChargesRecvAndSendBudget(t *testing.T) {
	up := newFakeUpstream(t, false, []wireFrame{{Data: []byte("a")}, {Data: []byte("b")}, {Data: []byte("c")}})
	h := newBridgeHarness(t, up, nil)
	limit := 3 // frame a: recv+send = 2; frame b: recv = 3, send = over
	h.ctx.SetBudget(effects.NewBudgetContext(map[string]*int{"Stream": &limit}))
	done := h.run(&eval.IntValue{Value: 0}, forwardAll)
	r := waitBridge(t, done)
	if r.end.CtorName != "StepFailed" || !strings.HasPrefix(r.end.Fields[0].(*eval.StringValue).Value, "BudgetExhausted") {
		t.Fatalf("end = %v, want StepFailed(BudgetExhausted...)", r.end)
	}
	if got := r.state.(*eval.IntValue).Value; got != 2 {
		t.Fatalf("step ran %d times, want 2", got)
	}
	f := readBrowser(t, h.browser, 1)
	if string(f[0].Data) != "a" {
		t.Fatalf("got %q", f[0].Data)
	}
	assertBrowserClose(t, h.browser, 1011)
}
