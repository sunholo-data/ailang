package effects_test

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/platform/streamcred"
	"github.com/sunholo-data/ailang/internal/trace"
)

// M3: host-side credential binding (M-SERVEAPI-WS-BRIDGE D3). The fake
// upstream runs over TLS because a binding applies to wss:// only.

const canaryToken = "CANARY-7f3a91d0c2b84e56-ya29.not-a-real-token"

type fixedSource struct{ tok string }

func (f fixedSource) Token(context.Context) (string, error) { return f.tok, nil }
func (f fixedSource) Describe() string                      { return "fixed (test)" }

// bindTo binds the canary to the upstream's exact host:port.
func bindTo(t *testing.T, rawURL string) func(*effects.StreamContext) {
	t.Helper()
	u, err := url.Parse(rawURL)
	if err != nil {
		t.Fatal(err)
	}
	binder, err := streamcred.Binder([]streamcred.Binding{{Host: u.Hostname(), Port: u.Port(), Source: fixedSource{canaryToken}}})
	if err != nil {
		t.Fatal(err)
	}
	return func(sc *effects.StreamContext) { sc.Credentials = binder }
}

func connectWith(ctx *effects.EffContext, rawURL string, headers ...[2]string) eval.Value {
	var hs []eval.Value
	for _, h := range headers {
		hs = append(hs, &eval.RecordValue{Fields: map[string]eval.Value{
			"name": &eval.StringValue{Value: h[0]}, "value": &eval.StringValue{Value: h[1]},
		}})
	}
	res, err := effects.Call(ctx, "Stream", "connect", []eval.Value{
		&eval.StringValue{Value: rawURL},
		&eval.RecordValue{Fields: map[string]eval.Value{"headers": &eval.ListValue{Elements: hs}}},
	})
	if err != nil {
		return &eval.StringValue{Value: "go error: " + err.Error()}
	}
	return res
}

// M3-A2: the binding fires — the fake TLS upstream RECEIVED the bearer — and
// M3-A1 at unit level: the program's view (connect args/result, the trace)
// never contains it.
func TestCredentialBinding_FiresForBoundWSS(t *testing.T) {
	up := newFakeUpstream(t, true, nil)
	h := newBridgeHarness(t, nil, bindTo(t, up.url()))
	h.ctx.Stream.SetTLSClientConfig(up.tlsConfig())
	h.ctx.Trace = trace.NewCollectorWithTier(trace.TierDeep)

	res := connectWith(h.ctx, up.url())
	if tag, ok := res.(*eval.TaggedValue); !ok || tag.CtorName != "Ok" {
		t.Fatalf("connect: %v", res)
	}
	_, hdr := up.got()
	if got := hdr.Get("Authorization"); got != "Bearer "+canaryToken {
		t.Fatalf("upstream Authorization = %q, want the bound bearer", got)
	}
	if dump := traceText(h.ctx.Trace); strings.Contains(dump, canaryToken) {
		t.Fatalf("trace contains the credential:\n%s", dump)
	}
}

// M3-A3: a binding for host:port X is not applied to ws://X, nor to the same
// host on another port. (Host-name mismatch and redirects: a binding matches
// the exact dial URL only — streamcred's Matches tests cover names, and a
// WebSocket handshake never follows a redirect.)
func TestCredentialBinding_ExactHostPortSchemeOnly(t *testing.T) {
	t.Run("ws:// to the bound host:port", func(t *testing.T) {
		plain := newFakeUpstream(t, false, nil)
		h := newBridgeHarness(t, nil, bindTo(t, plain.url())) // same host:port, ws scheme
		if res := connectWith(h.ctx, plain.url()); res.(*eval.TaggedValue).CtorName != "Ok" {
			t.Fatalf("connect: %v", res)
		}
		if _, hdr := plain.got(); hdr.Get("Authorization") != "" {
			t.Fatalf("binding applied to ws://: %q", hdr.Get("Authorization"))
		}
	})
	t.Run("wss:// to another port", func(t *testing.T) {
		bound := newFakeUpstream(t, true, nil)
		other := newFakeUpstream(t, true, nil)
		h := newBridgeHarness(t, nil, bindTo(t, bound.url()))
		h.ctx.Stream.SetTLSClientConfig(other.tlsConfig())
		if res := connectWith(h.ctx, other.url()); res.(*eval.TaggedValue).CtorName != "Ok" {
			t.Fatalf("connect: %v", res)
		}
		if _, hdr := other.got(); hdr.Get("Authorization") != "" {
			t.Fatalf("binding applied to another port: %q", hdr.Get("Authorization"))
		}
	})
}

// M3-A4: a program-supplied Authorization to a bound host fails with the
// named error, so the credential has exactly one source.
func TestCredentialBinding_RefusesProgramAuthorizationForBoundHost(t *testing.T) {
	up := newFakeUpstream(t, true, nil)
	h := newBridgeHarness(t, nil, bindTo(t, up.url()))
	h.ctx.Stream.SetTLSClientConfig(up.tlsConfig())

	res := connectWith(h.ctx, up.url(), [2]string{"authorization", "Bearer program-token"})
	tag, _ := res.(*eval.TaggedValue)
	if tag == nil || tag.CtorName != "Err" {
		t.Fatalf("connect = %v, want Err(ConnectionFailed(credential binding ...))", res)
	}
	msg := tag.Fields[0].(*eval.TaggedValue).Fields[0].(*eval.StringValue).Value
	if !strings.Contains(msg, "credential binding: Authorization header supplied by program for bound host") {
		t.Fatalf("error = %q", msg)
	}
	if _, hdr := up.got(); hdr != nil {
		t.Fatal("the refused dial reached the upstream")
	}
}

// M3-A5 / G7: without a binding, a program-supplied Authorization header is
// [REDACTED] in the trace — at the effect site AND the function-call site —
// while the upstream still receives it.
func TestTrace_RedactsProgramAuthorizationHeader(t *testing.T) {
	const programToken = "prog-9c1e-token-value-abcdef"
	up := newFakeUpstream(t, false, nil)
	h := newBridgeHarness(t, nil, nil)
	h.ctx.Trace = trace.NewCollectorWithTier(trace.TierDeep)

	res := connectWith(h.ctx, up.url(), [2]string{"Authorization", "Bearer " + programToken}, [2]string{"X-Trace-Id", "keep-me"})
	if tag, ok := res.(*eval.TaggedValue); !ok || tag.CtorName != "Ok" {
		t.Fatalf("connect: %v", res)
	}
	if _, hdr := up.got(); hdr.Get("Authorization") != "Bearer "+programToken {
		t.Fatal("redaction must not change what goes on the wire")
	}
	dump := traceText(h.ctx.Trace)
	if strings.Contains(dump, programToken) {
		t.Fatalf("trace contains the program's bearer:\n%s", dump)
	}
	if !strings.Contains(dump, eval.RedactedMarker) || !strings.Contains(dump, "keep-me") {
		t.Fatalf("want the Authorization value redacted and other headers kept:\n%s", dump)
	}
}

// traceText renders every recorded trace event, effect payloads included.
func traceText(c *trace.Collector) string {
	var b strings.Builder
	for _, ev := range c.Events() {
		fmt.Fprintf(&b, "%+v", ev)
		if ev.Effect != nil {
			fmt.Fprintf(&b, " effect=%+v", *ev.Effect)
		}
		b.WriteByte('\n')
	}
	return b.String()
}
