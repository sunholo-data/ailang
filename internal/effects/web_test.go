package effects

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/eval"
)

// std/web (M-DANEEL-AILANG-EXECUTOR M2): search and fetch through a fixed
// backend, {Net} only, the key read in Go and never visible to the program.

const webSentinelKey = "sk-ollama-SENTINEL-do-not-leak-4d7c"

func webTestCtx(t *testing.T) *EffContext {
	t.Helper()
	ctx := NewEffContext([]string{})
	ctx.Grant(NewCapability("Net"))
	ctx.Net.AllowHTTP = true
	ctx.Net.AllowLocalhost = true
	return ctx
}

// webFixtureServer serves the recorded Ollama responses and records what it
// was sent. The sentinel key is required on every request, exactly as the
// real API requires the real one.
func webFixtureServer(t *testing.T) (*httptest.Server, *[]*http.Request, *[][]byte) {
	t.Helper()
	var seen []*http.Request
	var bodies [][]byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		seen = append(seen, r)
		bodies = append(bodies, b)
		if r.Header.Get("Authorization") != "Bearer "+webSentinelKey {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"Unauthorized"}`))
			return
		}
		var fixture string
		switch r.URL.Path {
		case "/api/web_search":
			fixture = "ollama_web_search.json"
		case "/api/web_fetch":
			fixture = "ollama_web_fetch.json"
		default:
			w.WriteHeader(http.StatusNotFound)
			return
		}
		data, err := os.ReadFile(filepath.Join("testdata", fixture))
		if err != nil {
			t.Fatalf("fixture %s: %v", fixture, err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(data)
	}))
	t.Cleanup(srv.Close)
	return srv, &seen, &bodies
}

func pointWebAt(t *testing.T, base string) {
	t.Helper()
	old := ollamaWebBaseURL
	ollamaWebBaseURL = base
	t.Cleanup(func() { ollamaWebBaseURL = old })
}

func resultOf(t *testing.T, v eval.Value) (ctor string, payload eval.Value) {
	t.Helper()
	tv, ok := v.(*eval.TaggedValue)
	if !ok || tv.TypeName != "Result" {
		t.Fatalf("not a Result: %#v", v)
	}
	return tv.CtorName, tv.Fields[0]
}

func netErrOf(t *testing.T, v eval.Value) (ctor, msg string) {
	t.Helper()
	tv, ok := v.(*eval.TaggedValue)
	if !ok || tv.TypeName != "NetError" {
		t.Fatalf("not a NetError: %#v", v)
	}
	return tv.CtorName, tv.Fields[0].(*eval.StringValue).Value
}

func str(t *testing.T, r *eval.RecordValue, k string) string {
	t.Helper()
	s, ok := r.Fields[k].(*eval.StringValue)
	if !ok {
		t.Fatalf("field %s missing or not a string: %#v", k, r.Fields[k])
	}
	return s.Value
}

func TestWebSearch_RecordedFixture(t *testing.T) {
	t.Setenv(config.EnvOllamaAPIKey, webSentinelKey)
	srv, seen, bodies := webFixtureServer(t)
	pointWebAt(t, srv.URL)
	ctx := webTestCtx(t)

	v, err := WebSearch(ctx, []eval.Value{&eval.StringValue{Value: "AILANG programming language"}, &eval.IntValue{Value: 2}})
	if err != nil {
		t.Fatal(err)
	}
	ctor, payload := resultOf(t, v)
	if ctor != "Ok" {
		t.Fatalf("want Ok, got %s %#v", ctor, payload)
	}
	list := payload.(*eval.ListValue)
	if len(list.Elements) != 2 {
		t.Fatalf("want 2 results, got %d", len(list.Elements))
	}
	first := list.Elements[0].(*eval.RecordValue)
	if str(t, first, "url") != "https://ailang.sunholo.com/" || str(t, first, "title") == "" || str(t, first, "content") == "" {
		t.Errorf("first result not decoded: %#v", first.Fields)
	}
	// the wire: fixed endpoint, documented payload, the key as a bearer header
	if len(*seen) != 1 || (*seen)[0].URL.Path != "/api/web_search" || (*seen)[0].Method != "POST" {
		t.Fatalf("unexpected request(s): %d %v", len(*seen), *seen)
	}
	var sent map[string]any
	_ = json.Unmarshal((*bodies)[0], &sent)
	if sent["query"] != "AILANG programming language" || sent["max_results"] != float64(2) {
		t.Errorf("payload: %s", (*bodies)[0])
	}
}

func TestWebFetch_RecordedFixture(t *testing.T) {
	t.Setenv(config.EnvOllamaAPIKey, webSentinelKey)
	srv, seen, bodies := webFixtureServer(t)
	pointWebAt(t, srv.URL)
	ctx := webTestCtx(t)

	v, err := WebFetch(ctx, []eval.Value{&eval.StringValue{Value: "https://ailang.sunholo.com/"}})
	if err != nil {
		t.Fatal(err)
	}
	ctor, payload := resultOf(t, v)
	if ctor != "Ok" {
		t.Fatalf("want Ok, got %s %#v", ctor, payload)
	}
	page := payload.(*eval.RecordValue)
	if !strings.Contains(str(t, page, "title"), "AILANG") || str(t, page, "content") == "" {
		t.Errorf("page not decoded: %#v", page.Fields)
	}
	if links := page.Fields["links"].(*eval.ListValue); len(links.Elements) != 3 {
		t.Errorf("want 3 links, got %d", len(links.Elements))
	}
	// the supplied URL travels in the payload to the FIXED fetch endpoint —
	// the handler never requests the page itself
	if (*seen)[0].URL.Path != "/api/web_fetch" || !strings.Contains(string((*bodies)[0]), `"url":"https://ailang.sunholo.com/"`) {
		t.Errorf("fetch must go through the backend endpoint: %s %s", (*seen)[0].URL.Path, (*bodies)[0])
	}
}

func TestWeb_MissingKeyIsATypedErrThatNamesTheVariable(t *testing.T) {
	t.Setenv(config.EnvOllamaAPIKey, "")
	srv, seen, _ := webFixtureServer(t)
	pointWebAt(t, srv.URL)
	v, err := WebSearch(webTestCtx(t), []eval.Value{&eval.StringValue{Value: "x"}, &eval.IntValue{Value: 1}})
	if err != nil {
		t.Fatal(err)
	}
	ctor, e := resultOf(t, v)
	if ctor != "Err" {
		t.Fatalf("want Err, got %s", ctor)
	}
	if c, msg := netErrOf(t, e); c != "Transport" || !strings.Contains(msg, config.EnvOllamaAPIKey) {
		t.Errorf("want Transport naming the variable, got %s %q", c, msg)
	}
	if len(*seen) != 0 {
		t.Errorf("no request may be made without a key; saw %d", len(*seen))
	}
}

func TestWeb_Non2xxAndMalformedAreTransportErrs(t *testing.T) {
	t.Setenv(config.EnvOllamaAPIKey, "wrong-key")
	srv, _, _ := webFixtureServer(t) // 401 for a wrong key
	pointWebAt(t, srv.URL)
	v, _ := WebSearch(webTestCtx(t), []eval.Value{&eval.StringValue{Value: "x"}, &eval.IntValue{Value: 1}})
	ctor, e := resultOf(t, v)
	c, msg := netErrOf(t, e)
	if ctor != "Err" || c != "Transport" || !strings.Contains(msg, "HTTP 401") {
		t.Errorf("401: want Err Transport 'HTTP 401', got %s %s %q", ctor, c, msg)
	}

	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(`{"nope":`)) }))
	t.Cleanup(bad.Close)
	t.Setenv(config.EnvOllamaAPIKey, webSentinelKey)
	pointWebAt(t, bad.URL)
	v, _ = WebSearch(webTestCtx(t), []eval.Value{&eval.StringValue{Value: "x"}, &eval.IntValue{Value: 1}})
	ctor, e = resultOf(t, v)
	c, msg = netErrOf(t, e)
	if ctor != "Err" || c != "Transport" || !strings.Contains(msg, "malformed") {
		t.Errorf("malformed: want Err Transport 'malformed', got %s %s %q", ctor, c, msg)
	}
}

func TestWeb_InvalidInputFailsBeforeAnyRequest(t *testing.T) {
	t.Setenv(config.EnvOllamaAPIKey, webSentinelKey)
	srv, seen, _ := webFixtureServer(t)
	pointWebAt(t, srv.URL)
	ctx := webTestCtx(t)
	for _, tc := range []struct {
		name string
		args []eval.Value
	}{
		{"max 0", []eval.Value{&eval.StringValue{Value: "x"}, &eval.IntValue{Value: 0}}},
		{"max over bound", []eval.Value{&eval.StringValue{Value: "x"}, &eval.IntValue{Value: WebSearchMaxResults + 1}}},
		{"empty query", []eval.Value{&eval.StringValue{Value: ""}, &eval.IntValue{Value: 1}}},
	} {
		if _, err := WebSearch(ctx, tc.args); err == nil || !strings.Contains(err.Error(), "E_WEB_INVALID_INPUT") {
			t.Errorf("%s: want E_WEB_INVALID_INPUT, got %v", tc.name, err)
		}
	}
	if _, err := WebFetch(ctx, []eval.Value{&eval.StringValue{Value: ""}}); err == nil || !strings.Contains(err.Error(), "E_WEB_INVALID_INPUT") {
		t.Errorf("empty url: want E_WEB_INVALID_INPUT, got %v", err)
	}
	if len(*seen) != 0 {
		t.Errorf("invalid input must fail before the request; saw %d requests", len(*seen))
	}
}

func TestWeb_RequiresNetCapAndHonoursTheAllowlist(t *testing.T) {
	t.Setenv(config.EnvOllamaAPIKey, webSentinelKey)
	srv, _, _ := webFixtureServer(t)
	pointWebAt(t, srv.URL)
	noCap := NewEffContext([]string{})
	if _, err := WebSearch(noCap, []eval.Value{&eval.StringValue{Value: "x"}, &eval.IntValue{Value: 1}}); err == nil {
		t.Error("without Net the call must be a capability error")
	}
	// net_allow that does not list the backend host: the fixed endpoint is
	// still subject to the operator's allowlist
	ctx := webTestCtx(t)
	ctx.Net.AllowedDomains = []string{"example.com"}
	v, err := WebSearch(ctx, []eval.Value{&eval.StringValue{Value: "x"}, &eval.IntValue{Value: 1}})
	if err != nil {
		t.Fatal(err)
	}
	if ctor, e := resultOf(t, v); ctor != "Err" {
		t.Errorf("want Err from the allowlist, got %s", ctor)
	} else if c, _ := netErrOf(t, e); c != "DisallowedHost" {
		t.Errorf("want DisallowedHost, got %s", c)
	}
}

// The key must never reach the program: not in a value, not in an error, not
// in the stringified result. Every path that can carry text is exercised with
// the sentinel set and searched for it.
func TestWeb_SecretNeverLeaks(t *testing.T) {
	t.Setenv(config.EnvOllamaAPIKey, webSentinelKey)
	srv, _, _ := webFixtureServer(t)
	pointWebAt(t, srv.URL)
	ctx := webTestCtx(t)

	var carriers []string
	collect := func(v eval.Value, err error) {
		if err != nil {
			carriers = append(carriers, err.Error())
		}
		if v != nil {
			carriers = append(carriers, fmt.Sprintf("%#v", v), v.String())
		}
	}
	collect(WebSearch(ctx, []eval.Value{&eval.StringValue{Value: "AILANG"}, &eval.IntValue{Value: 2}}))
	collect(WebFetch(ctx, []eval.Value{&eval.StringValue{Value: "https://ailang.sunholo.com/"}}))
	collect(WebSearch(ctx, []eval.Value{&eval.StringValue{Value: ""}, &eval.IntValue{Value: 1}}))
	blocked := webTestCtx(t)
	blocked.Net.AllowedDomains = []string{"example.com"}
	collect(WebSearch(blocked, []eval.Value{&eval.StringValue{Value: "x"}, &eval.IntValue{Value: 1}}))
	// non-2xx WITH the sentinel on the wire — the error text must not echo it
	five := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(500); _, _ = w.Write([]byte("boom")) }))
	t.Cleanup(five.Close)
	pointWebAt(t, five.URL)
	collect(WebSearch(ctx, []eval.Value{&eval.StringValue{Value: "x"}, &eval.IntValue{Value: 1}}))
	collect(WebFetch(ctx, []eval.Value{&eval.StringValue{Value: "https://x.test/"}}))
	// malformed body with the sentinel on the wire
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(`{"nope":`)) }))
	t.Cleanup(bad.Close)
	pointWebAt(t, bad.URL)
	collect(WebSearch(ctx, []eval.Value{&eval.StringValue{Value: "x"}, &eval.IntValue{Value: 1}}))
	// connection refused
	dead := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(500) }))
	dead.Close()
	pointWebAt(t, dead.URL)
	collect(WebSearch(ctx, []eval.Value{&eval.StringValue{Value: "x"}, &eval.IntValue{Value: 1}}))

	if len(carriers) < 12 {
		t.Fatalf("non-leak assertion is vacuous: only %d carriers", len(carriers))
	}
	for i, c := range carriers {
		if strings.Contains(c, webSentinelKey) || strings.Contains(c, "SENTINEL") {
			t.Fatalf("secret leaked via carrier %d: %s", i, c)
		}
	}
	// positive control: the sentinel DID go on the wire
	var sawKey bool
	probe := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawKey = r.Header.Get("Authorization") == "Bearer "+webSentinelKey
		_, _ = w.Write([]byte(`{"results":[]}`))
	}))
	t.Cleanup(probe.Close)
	pointWebAt(t, probe.URL)
	_, _ = WebSearch(ctx, []eval.Value{&eval.StringValue{Value: "x"}, &eval.IntValue{Value: 1}})
	if !sawKey {
		t.Fatal("positive control failed: the key never reached the wire, so the non-leak assertion proves nothing")
	}
}

// Live positive control: opt-in, never blocks the suite.
func TestWeb_LiveOllama(t *testing.T) {
	if os.Getenv("AILANG_WEB_LIVE_TEST") != "1" || config.OllamaAPIKey() == "" {
		t.Skip("set AILANG_WEB_LIVE_TEST=1 and OLLAMA_API_KEY to run the live control")
	}
	ctx := NewEffContext([]string{})
	ctx.Grant(NewCapability("Net"))
	v, err := WebSearch(ctx, []eval.Value{&eval.StringValue{Value: "AILANG programming language"}, &eval.IntValue{Value: 1}})
	if err != nil {
		t.Fatal(err)
	}
	ctor, payload := resultOf(t, v)
	if ctor != "Ok" || len(payload.(*eval.ListValue).Elements) < 1 {
		t.Fatalf("live search: %s %#v", ctor, payload)
	}
}

func TestWebCredentialVars(t *testing.T) {
	cases := []struct {
		allow []string
		want  bool
	}{
		{[]string{"ollama.com"}, true},
		{[]string{"OLLAMA.COM."}, true},
		{[]string{"example.org"}, false},
		{[]string{"*.ollama.com"}, false}, // wildcard excludes the apex
		{nil, false},                      // no list admits nothing here
	}
	for _, tc := range cases {
		got := WebCredentialVars(tc.allow)
		if (len(got) == 1 && got[0] == "OLLAMA_API_KEY") != tc.want || (!tc.want && got != nil) {
			t.Errorf("WebCredentialVars(%v) = %v, want key=%v", tc.allow, got, tc.want)
		}
	}
}
