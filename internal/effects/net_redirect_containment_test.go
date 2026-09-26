package effects

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/sunholo-data/ailang/internal/eval"
)

// M-EXECUTOR-POLICY-HARDENING M2 — permanent denial tests for the audited
// counterexample N1: with only one host allowlisted, an HTTP redirect to a
// host that is NOT allowlisted was followed. The domain allowlist was checked
// once before the first request; per-hop the round tripper re-validated the
// IP but never the domain, and the Stream transports (SSE, NDJSON) had no
// redirect check at all.
//
// Written BEFORE the fix; every case below FAILED on the audited baseline
// ("/final reached"). The recording handler is the oracle — never the
// response text.

// redirectFixture is one hermetic listener on 127.0.0.1 whose /start
// redirects to /final on a DIFFERENT hostname for the same listener.
// finalHits counts arrivals at /final: the forbidden connection.
type redirectFixture struct {
	srv       *httptest.Server
	finalHits atomic.Int32
	// finalHost is the hostname the redirect points at.
	finalHost string
}

func newRedirectFixture(t *testing.T, finalHost string, finalContentType string) *redirectFixture {
	t.Helper()
	f := &redirectFixture{finalHost: finalHost}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/start":
			http.Redirect(w, r, fmt.Sprintf("http://%s/final", net.JoinHostPort(finalHost, portOf(f.srv))), http.StatusFound)
		case "/final":
			f.finalHits.Add(1)
			w.Header().Set("Content-Type", finalContentType)
			_, _ = w.Write([]byte("data: leaked\n\n"))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(f.srv.Close)
	return f
}

func (f *redirectFixture) startURL() string { return f.srv.URL + "/start" }

// --- Net: the design-doc recipe — injected resolver + routed dialer ---
//
// Every hostname resolves to 203.0.113.10 and every dial to that address is
// routed to the local listener, so the request is hermetic and the oracle is
// the /final handler, never a dial failure. (A "localhost" alias would be
// ambiguous: LookupIP("localhost") returns ::1 first on macOS and the v4-only
// listener refuses it, which looks like a denial and is not one.)

func newRecipeNetCtx(t *testing.T, f *redirectFixture) *EffContext {
	t.Helper()
	probe := &netProbe{routes: map[string]string{}}
	ctx := newNetProbeCtx(t, probe)
	forceNoProxy(ctx)
	ctx.Net.AllowedDomains = []string{"allowed.example"}
	probe.resolve = func(string) ([]net.IP, error) { return []net.IP{net.ParseIP("203.0.113.10")}, nil }
	port := portOf(f.srv)
	probe.routes["203.0.113.10:"+port] = "127.0.0.1:" + port
	return ctx
}

func (f *redirectFixture) allowedStartURL() string {
	return "http://allowed.example:" + portOf(f.srv) + "/start"
}

func one(s string) []eval.Value { return []eval.Value{&eval.StringValue{Value: s}} }

func TestNetRedirectContainment_HTTPGet(t *testing.T) {
	f := newRedirectFixture(t, "denied.example", "text/plain")
	ctx := newRecipeNetCtx(t, f)
	_, err := Call(ctx, "Net", "httpGet", one(f.allowedStartURL()))
	if f.finalHits.Load() != 0 {
		t.Errorf("ALLOWLIST FAILURE: redirect reached denied.example (/final hit %d times)", f.finalHits.Load())
	}
	if err == nil || !strings.Contains(err.Error(), "denied.example") {
		t.Errorf("expected an error naming denied.example, got %v", err)
	}
}

func TestNetRedirectContainment_HTTPPost(t *testing.T) {
	f := newRedirectFixture(t, "denied.example", "text/plain")
	ctx := newRecipeNetCtx(t, f)
	_, err := Call(ctx, "Net", "httpPost", []eval.Value{&eval.StringValue{Value: f.allowedStartURL()}, &eval.StringValue{Value: "{}"}})
	if f.finalHits.Load() != 0 {
		t.Errorf("ALLOWLIST FAILURE: httpPost followed a redirect to denied.example")
	}
	if err == nil || !strings.Contains(err.Error(), "denied.example") {
		t.Errorf("expected an error naming denied.example, got %v", err)
	}
}

func TestNetRedirectContainment_HTTPRequest(t *testing.T) {
	f := newRedirectFixture(t, "denied.example", "text/plain")
	ctx := newRecipeNetCtx(t, f)
	res, err := NetHTTPRequest(ctx, []eval.Value{
		&eval.StringValue{Value: "GET"},
		&eval.StringValue{Value: f.allowedStartURL()},
		&eval.ListValue{Elements: []eval.Value{}},
		&eval.StringValue{Value: ""},
	})
	if err != nil {
		t.Fatal(err)
	}
	if f.finalHits.Load() != 0 {
		t.Errorf("ALLOWLIST FAILURE: httpRequest followed a redirect to denied.example")
	}
	if tv, ok := res.(*eval.TaggedValue); !ok || tv.CtorName != "Err" {
		t.Errorf("expected Err, got %v", res)
	}
}

func TestNetRedirectContainment_HTTPRequestBytes(t *testing.T) {
	f := newRedirectFixture(t, "denied.example", "text/plain")
	ctx := newRecipeNetCtx(t, f)
	res, err := NetHTTPRequestBytes(ctx, []eval.Value{
		&eval.StringValue{Value: "GET"},
		&eval.StringValue{Value: f.allowedStartURL()},
		&eval.ListValue{Elements: []eval.Value{}},
		&eval.BytesValue{Value: nil},
	})
	if err != nil {
		t.Fatal(err)
	}
	if f.finalHits.Load() != 0 {
		t.Errorf("ALLOWLIST FAILURE: httpRequestBytes followed a redirect to denied.example")
	}
	if tv, ok := res.(*eval.TaggedValue); !ok || tv.CtorName != "Err" {
		t.Errorf("expected Err, got %v", res)
	}
}

// --- Stream entrypoints: allowlist = the literal 127.0.0.1; redirect to "localhost" ---
//
// The Stream transports have no resolver/dial hooks on the baseline, so this
// uses two names for one listener; Go's dialer reaches the v4 listener via
// happy-eyeballs, which is why the baseline DID reach /final here.

func newLocalStreamCtx(t *testing.T) *EffContext {
	t.Helper()
	ctx := NewEffContext(nil)
	ctx.Grant(NewCapability("Stream"))
	ctx.Stream = NewStreamContext()
	ctx.Stream.AllowHTTP = true
	ctx.Stream.AllowLocalhost = true
	ctx.Stream.AllowedDomains = []string{"127.0.0.1"}
	t.Cleanup(ctx.Stream.CloseAll)
	return ctx
}

func TestNetRedirectContainment_SSEConnect(t *testing.T) {
	f := newRedirectFixture(t, "localhost", "text/event-stream")
	ctx := newLocalStreamCtx(t)
	res, err := StreamSSEConnect(ctx, []eval.Value{&eval.StringValue{Value: f.startURL()}, &eval.RecordValue{Fields: map[string]eval.Value{}}})
	if err != nil {
		t.Fatal(err)
	}
	if f.finalHits.Load() != 0 {
		t.Errorf("ALLOWLIST FAILURE: sseConnect followed a redirect to a non-allowlisted host")
	}
	if tv, ok := res.(*eval.TaggedValue); !ok || tv.CtorName != "Err" {
		t.Errorf("expected Err, got %v", res)
	}
}

func TestNetRedirectContainment_SSEPost(t *testing.T) {
	f := newRedirectFixture(t, "localhost", "text/event-stream")
	ctx := newLocalStreamCtx(t)
	res, err := StreamSSEPost(ctx, []eval.Value{&eval.StringValue{Value: f.startURL()}, &eval.StringValue{Value: "{}"}, &eval.RecordValue{Fields: map[string]eval.Value{}}})
	if err != nil {
		t.Fatal(err)
	}
	if f.finalHits.Load() != 0 {
		t.Errorf("ALLOWLIST FAILURE: ssePost followed a redirect to a non-allowlisted host")
	}
	if tv, ok := res.(*eval.TaggedValue); !ok || tv.CtorName != "Err" {
		t.Errorf("expected Err, got %v", res)
	}
}

func TestNetRedirectContainment_NDJSONPost(t *testing.T) {
	f := newRedirectFixture(t, "localhost", "application/x-ndjson")
	ctx := newLocalStreamCtx(t)
	res, err := StreamNDJSONPost(ctx, []eval.Value{&eval.StringValue{Value: f.startURL()}, &eval.StringValue{Value: "{}"}, &eval.RecordValue{Fields: map[string]eval.Value{}}})
	if err != nil {
		t.Fatal(err)
	}
	if f.finalHits.Load() != 0 {
		t.Errorf("ALLOWLIST FAILURE: ndjsonPost followed a redirect to a non-allowlisted host")
	}
	if tv, ok := res.(*eval.TaggedValue); !ok || tv.CtorName != "Err" {
		t.Errorf("expected Err, got %v", res)
	}
}

// Positive control: the same fixture with BOTH names allowlisted is followed.
func TestNetRedirectContainment_PositiveControl(t *testing.T) {
	f := newRedirectFixture(t, "denied.example", "text/plain")
	ctx := newRecipeNetCtx(t, f)
	ctx.Net.AllowedDomains = []string{"allowed.example", "denied.example"}
	if _, err := Call(ctx, "Net", "httpGet", one(f.allowedStartURL())); err != nil {
		t.Fatalf("allowlisted redirect must be followed: %v", err)
	}
	if f.finalHits.Load() != 1 {
		t.Errorf("expected /final to be reached once, got %d", f.finalHits.Load())
	}
}

// Cancellation: a GoCtx that is already cancelled stops the request before
// any dial.
func TestNetRedirectContainment_CancelledContextNoDial(t *testing.T) {
	probe := &netProbe{routes: map[string]string{}}
	ctx := newNetProbeCtx(t, probe)
	forceNoProxy(ctx)
	probe.resolve = func(string) ([]net.IP, error) { return []net.IP{net.ParseIP("203.0.113.10")}, nil }
	c, cancel := context.WithCancel(context.Background())
	cancel()
	ctx.GoCtx = c
	_, err := Call(ctx, "Net", "httpGet", one("http://allowed.example/"))
	if err == nil {
		t.Fatal("expected an error from a cancelled context")
	}
	if probe.snapshot().dialCalls != 0 {
		t.Errorf("a cancelled request must not dial; dialed %v", probe.snapshot().dialAddrs)
	}
}
