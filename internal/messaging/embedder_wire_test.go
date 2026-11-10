package messaging

import (
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/testutil/wiregolden"
)

// TestOllamaEmbedderWireParity pins what OllamaEmbedder puts on the wire and
// returns, against goldens captured while it still used
// github.com/ollama/ollama/api (v0.40.15). See internal/testutil/wiregolden.
func TestOllamaEmbedderWireParity(t *testing.T) {
	ok := func(v string) wiregolden.Reply {
		return wiregolden.Reply{Body: `{"model":"nomic-embed-text","embeddings":[[` + v + `]]}`}
	}
	long := strings.Repeat("Paragraph one is here.\n\n", 300) // > MaxChunkSize: chunked + averaged
	type tc struct {
		name     string
		endpoint string // "" = the server URL; "quoted" = the server URL in quotes
		refused  bool
		text     string
		replies  []wiregolden.Reply
	}
	cases := []tc{
		{name: "single", text: "hello <world>", replies: []wiregolden.Reply{ok("0.5,-1.25,3")}},
		{name: "quoted_endpoint", endpoint: "quoted", text: "q", replies: []wiregolden.Reply{ok("1")}},
		{name: "chunked_average", text: long, replies: []wiregolden.Reply{ok("1,2"), ok("3,4")}},
		{name: "chunked_one_fails", text: long, replies: []wiregolden.Reply{{Status: 500, Body: `{"error":"x"}`}, ok("3,4")}},
		{name: "err_404", text: "x", replies: []wiregolden.Reply{{Status: 404, Body: `{"error":"model not found"}`}}},
		{name: "err_empty", text: "x", replies: []wiregolden.Reply{{Body: `{"embeddings":[]}`}}},
		{name: "err_refused", refused: true, text: "x"},
	}
	got := map[string]wiregolden.Outcome{}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Setenv(config.EnvRigLease, "")
			t.Setenv(config.EnvOllamaHost, "") // NewOllamaEmbedder exports it; restore after
			srv := wiregolden.NewServer(t, c.replies...)
			if c.refused {
				srv.Close()
			}
			endpoint := srv.URL
			if c.endpoint == "quoted" {
				endpoint = `'` + srv.URL + `'`
			}
			e, err := NewOllamaEmbedder(OllamaConfig{Model: "nomic-embed-text", Endpoint: endpoint, Timeout: 10 * time.Second})
			if err != nil {
				t.Fatalf("NewOllamaEmbedder: %v", err)
			}
			v, err := e.Embed(c.text)
			out := wiregolden.Outcome{Requests: srv.Exchanges()}
			if err != nil {
				out.Error = wiregolden.ErrorText(err, srv.URL)
			} else {
				out.Result = v
			}
			got[c.name] = out
		})
	}
	wiregolden.Check(t, "testdata/ollama_embedder_wire.json", got)
}
