package wiregolden

import (
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestServerRecordsAndReplies(t *testing.T) {
	s := NewServer(t,
		Reply{Lines: []string{"a", "b"}},
		Reply{Status: 418, Body: "teapot"},
	)
	resp, err := http.Post(s.URL+"/x?q=1", "application/json", strings.NewReader(`{"k":1}`))
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if string(body) != "a\nb\n" {
		t.Errorf("lines body = %q", body)
	}
	resp, err = http.Get(s.URL + "/y")
	if err != nil {
		t.Fatal(err)
	}
	body, _ = io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != 418 || string(body) != "teapot" {
		t.Errorf("second reply = %d %q", resp.StatusCode, body)
	}
	resp, err = http.Get(s.URL + "/z")
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusTeapot {
		t.Errorf("an unscripted request must be refused, got %d", resp.StatusCode)
	}
	ex := s.Exchanges()
	if len(ex) != 3 || ex[0].Method != "POST" || ex[0].Path != "/x" || ex[0].Query != "q=1" || ex[0].Body != `{"k":1}` {
		t.Fatalf("exchanges = %+v", ex)
	}
	if ua := ex[0].Header["User-Agent"]; ua == "" {
		t.Error("headers not recorded")
	}
}

func TestStallHoldsUntilClientLeaves(t *testing.T) {
	s := NewServer(t, Reply{Lines: []string{"first"}, Stall: true})
	c := &http.Client{Timeout: 200 * time.Millisecond}
	resp, err := c.Get(s.URL)
	if err != nil {
		t.Fatal(err)
	}
	_, err = io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err == nil {
		t.Fatal("a stalled reply must not end on its own")
	}
}

func TestNormalizeHeaderUserAgent(t *testing.T) {
	h := http.Header{"User-Agent": {userAgentForTest()}, "Accept": {"a", "b"}}
	got := normalizeHeader(h)
	if got["User-Agent"] != "ollama/0.0.0 (GOARCH GOOS) Go/GOVERSION" || got["Accept"] != "a, b" {
		t.Errorf("normalized = %v", got)
	}
}

func TestErrorText(t *testing.T) {
	err := errors.New(`Post "http://127.0.0.1:4242/api": dial tcp 127.0.0.1:4242: ` + windowsRefused)
	want := `Post "http://SERVER/api": dial tcp SERVER: connect: connection refused`
	if got := ErrorText(err, "http://127.0.0.1:4242"); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestCheckMatchesGolden(t *testing.T) {
	path := filepath.Join(t.TempDir(), "g.json")
	got := map[string]Outcome{"s": {Requests: []Exchange{{Method: "GET", Path: "/", Header: map[string]string{}}}, Result: 1}}
	*update = true
	Check(t, path, got)
	*update = false
	Check(t, path, got)
}

func userAgentForTest() string {
	return "ollama/0.0.0 (" + runtime.GOARCH + " " + runtime.GOOS + ") Go/" + runtime.Version()
}
