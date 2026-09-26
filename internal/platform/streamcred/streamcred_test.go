package streamcred

import (
	"context"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func mustURL(t *testing.T, s string) *url.URL {
	t.Helper()
	u, err := url.Parse(s)
	if err != nil {
		t.Fatal(err)
	}
	return u
}

func TestParse(t *testing.T) {
	dir := t.TempDir()
	tok := filepath.Join(dir, "tok")
	if err := os.WriteFile(tok, []byte("  abc  \n"), 0o600); err != nil {
		t.Fatal(err)
	}
	b, err := Parse("US-Central1-aiplatform.googleapis.com=bearer-file:" + tok)
	if err != nil {
		t.Fatal(err)
	}
	if b.Host != "us-central1-aiplatform.googleapis.com" || b.Port != "" {
		t.Fatalf("host/port = %q/%q", b.Host, b.Port)
	}
	if got, err := b.Source.Token(context.Background()); err != nil || got != "abc" {
		t.Fatalf("token = %q, %v", got, err)
	}
	if b, err := Parse("127.0.0.1:8443=gcp-metadata"); err != nil || b.Port != "8443" {
		t.Fatalf("port parse: %+v %v", b, err)
	}

	for _, bad := range []string{
		"host-only",
		"=gcp-metadata",
		"wss://host/path=gcp-metadata",
		"host=gcp-adc",             // no implicit ADC source
		"host=gcp-key-file:",       // path required
		"host=gcp-metadata:extra",  // takes no argument
		"host=bearer-file:",        // path required
		"host=gcp-key-file:/nope/", // unreadable
	} {
		if _, err := Parse(bad); err == nil {
			t.Errorf("Parse(%q) = nil error", bad)
		}
	}
}

// A gcloud user login (authorized_user) is refused: the relay runs as its
// own identity, never as whoever last ran gcloud (D3).
func TestParse_RefusesUserCredentials(t *testing.T) {
	path := filepath.Join(t.TempDir(), "adc.json")
	body := `{"type":"authorized_user","client_id":"x","client_secret":"y","refresh_token":"z"}`
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := Parse("h=gcp-key-file:" + path)
	if err == nil || !strings.Contains(err.Error(), `"authorized_user" is not accepted`) {
		t.Fatalf("err = %v", err)
	}
	if strings.Contains(err.Error(), "refresh_token") || strings.Contains(err.Error(), `"z"`) {
		t.Fatalf("error leaks file content: %v", err)
	}
}

func TestMatches(t *testing.T) {
	b := Binding{Host: "api.example.com"}
	for raw, want := range map[string]bool{
		"wss://api.example.com/ws":        true,
		"wss://API.example.com:443/ws":    true,
		"ws://api.example.com/ws":         false,
		"https://api.example.com/ws":      false,
		"wss://api.example.com:8443/ws":   false,
		"wss://evil.example.com/ws":       false,
		"wss://api.example.com.evil.io/x": false,
	} {
		if got := b.Matches(mustURL(t, raw)); got != want {
			t.Errorf("Matches(%s) = %v, want %v", raw, got, want)
		}
	}
	p := Binding{Host: "127.0.0.1", Port: "9443"}
	if !p.Matches(mustURL(t, "wss://127.0.0.1:9443/")) || p.Matches(mustURL(t, "wss://127.0.0.1/")) {
		t.Error("explicit-port binding must match that port only")
	}
}

type staticSource string

func (s staticSource) Token(context.Context) (string, error) { return string(s), nil }
func (s staticSource) Describe() string                      { return "static" }

func TestBinder(t *testing.T) {
	binder, err := Binder([]Binding{{Host: "a.example", Source: staticSource("T")}})
	if err != nil {
		t.Fatal(err)
	}
	hdr, bound, err := binder(mustURL(t, "wss://a.example/x"))
	if err != nil || !bound || hdr["Authorization"][0] != "Bearer T" {
		t.Fatalf("bound dial: %v %v %v", hdr, bound, err)
	}
	if _, bound, _ := binder(mustURL(t, "wss://b.example/x")); bound {
		t.Fatal("unbound host reported bound")
	}
	if _, err := Binder([]Binding{{Host: "a", Source: staticSource("1")}, {Host: "a", Source: staticSource("2")}}); err == nil {
		t.Fatal("duplicate binding accepted")
	}
}
