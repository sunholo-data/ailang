package notify

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

// fakeDoer records requests and returns a canned status/error — no network.
type fakeDoer struct {
	urls    []string
	bodies  []string
	status  int
	err     error
	callCnt int
}

func (f *fakeDoer) Do(req *http.Request) (*http.Response, error) {
	f.callCnt++
	if f.err != nil {
		return nil, f.err
	}
	b, _ := io.ReadAll(req.Body)
	f.urls = append(f.urls, req.URL.String())
	f.bodies = append(f.bodies, string(b))
	st := f.status
	if st == 0 {
		st = http.StatusNoContent // Discord returns 204 on success
	}
	return &http.Response{StatusCode: st, Body: io.NopCloser(strings.NewReader(""))}, nil
}

func newDiscordWithDoer(url string, d httpDoer) *DiscordChannel {
	return &DiscordChannel{webhookURL: url, http: d}
}

func TestDiscordSendHappyPath(t *testing.T) {
	doer := &fakeDoer{}
	ch := newDiscordWithDoer("https://discord.test/webhook/abc", doer)

	err := ch.Send(context.Background(), Notification{Title: "Approval pending", Body: "doc X awaits", URL: "https://gh/issue/1"})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if doer.callCnt != 1 {
		t.Fatalf("expected 1 POST, got %d", doer.callCnt)
	}
	if doer.urls[0] != "https://discord.test/webhook/abc" {
		t.Errorf("posted to %q, want the webhook URL", doer.urls[0])
	}
	if !strings.Contains(doer.bodies[0], "Approval pending") || !strings.Contains(doer.bodies[0], "doc X awaits") {
		t.Errorf("body missing rendered fields: %s", doer.bodies[0])
	}
	if !strings.Contains(doer.bodies[0], `"content"`) {
		t.Errorf("body should be a discord content payload: %s", doer.bodies[0])
	}
}

func TestDiscordSendChunks(t *testing.T) {
	doer := &fakeDoer{}
	ch := newDiscordWithDoer("https://discord.test/webhook", doer)

	// Body well over the 2000-char limit -> multiple POSTs.
	long := strings.Repeat("x", 4500)
	if err := ch.Send(context.Background(), Notification{Body: long}); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if doer.callCnt != 3 { // ceil(~4500 / 2000) accounting for rendering = 3 chunks
		t.Fatalf("expected 3 chunked POSTs, got %d", doer.callCnt)
	}
}

func TestDiscordSendNon2xxErrors(t *testing.T) {
	doer := &fakeDoer{status: http.StatusInternalServerError}
	ch := newDiscordWithDoer("https://discord.test/webhook", doer)
	if err := ch.Send(context.Background(), Notification{Title: "hi"}); err == nil {
		t.Fatal("expected error on 500 response")
	}
}

func TestDiscordSendTransportErrors(t *testing.T) {
	doer := &fakeDoer{err: errors.New("boom")}
	ch := newDiscordWithDoer("https://discord.test/webhook", doer)
	if err := ch.Send(context.Background(), Notification{Title: "hi"}); err == nil {
		t.Fatal("expected error on transport failure")
	}
}

func TestChunkMessage(t *testing.T) {
	if got := chunkMessage("short", 2000); len(got) != 1 || got[0] != "short" {
		t.Fatalf("short string should be one chunk, got %v", got)
	}
	if got := chunkMessage(strings.Repeat("a", 10), 4); len(got) != 3 {
		t.Fatalf("expected 3 chunks (4+4+2), got %d", len(got))
	}
	// Rune-safety: multi-byte runes are never split.
	multi := strings.Repeat("é", 5) // 5 runes, 10 bytes
	chunks := chunkMessage(multi, 2)
	if len(chunks) != 3 {
		t.Fatalf("expected 3 chunks for 5 runes at limit 2, got %d", len(chunks))
	}
	rejoined := strings.Join(chunks, "")
	if rejoined != multi {
		t.Fatalf("rejoined chunks %q != original %q", rejoined, multi)
	}
}
