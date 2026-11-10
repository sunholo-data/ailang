package builtins

import (
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/testutil/wiregolden"
)

// TestOllamaEmbedWireParity pins what _ollama_embed / _ollama_embed_media put
// on the wire and return, against goldens captured while they still used
// github.com/ollama/ollama/api (v0.40.15). See internal/testutil/wiregolden.
func TestOllamaEmbedWireParity(t *testing.T) {
	const okBody = `{"model":"nomic-embed-text","embeddings":[[0.1,-0.25,3.4028235e38,1e-46,7]],"total_duration":5,"load_duration":1,"prompt_eval_count":3,"extra":true}`
	type tc struct {
		name    string
		quoted  bool
		refused bool
		input   any
		replies []wiregolden.Reply
	}
	cases := []tc{
		{name: "text", input: "hello <world> &  ", replies: []wiregolden.Reply{{Body: okBody}}},
		{name: "quoted_host_env", quoted: true, input: "q", replies: []wiregolden.Reply{{Body: okBody}}},
		{name: "media_map", input: ollamaMediaInput("caption", []byte("PNG\x00\xff"), []byte("RIFF")), replies: []wiregolden.Reply{{Body: okBody}}},
		{name: "media_text_only", input: ollamaMediaInput("only text", nil, nil), replies: []wiregolden.Reply{{Body: okBody}}},
		{name: "two_embeddings_takes_first", input: "x", replies: []wiregolden.Reply{{Body: `{"embeddings":[[1,2],[3,4]]}`}}},
		{name: "err_no_embeddings", input: "x", replies: []wiregolden.Reply{{Body: `{"model":"m","embeddings":[]}`}}},
		{name: "err_null_embeddings", input: "x", replies: []wiregolden.Reply{{Body: `{"model":"m"}`}}},
		{name: "err_float32_overflow", input: "x", replies: []wiregolden.Reply{{Body: `{"embeddings":[[1e39]]}`}}},
		{name: "err_404_json", input: "x", replies: []wiregolden.Reply{{Status: 404, Body: `{"error":"model \"nomic-embed-text\" not found, try pulling it first"}`}}},
		{name: "err_500_text", input: "x", replies: []wiregolden.Reply{{Status: 500, Body: "kaput"}}},
		{name: "err_400_empty", input: "x", replies: []wiregolden.Reply{{Status: 400, Body: ""}}},
		{name: "err_401", input: "x", replies: []wiregolden.Reply{{Status: 401, Body: `{"error":"unauthorized"}`}}},
		{name: "err_200_garbage", input: "x", replies: []wiregolden.Reply{{Body: "<html>"}}},
		{name: "ok_200_empty_body", input: "x", replies: []wiregolden.Reply{{Body: ""}}},
		{name: "err_refused", refused: true, input: "x"},
	}
	got := map[string]wiregolden.Outcome{}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Setenv(config.EnvRigLease, "")
			srv := wiregolden.NewServer(t, c.replies...)
			if c.refused {
				srv.Close()
			}
			host := srv.URL
			if c.quoted {
				host = `"` + host + `"`
			}
			t.Setenv(config.EnvOllamaHost, host)
			v, err := ollamaEmbedRequest("_ollama_embed", "nomic-embed-text", c.input, 10*time.Second)
			out := wiregolden.Outcome{Requests: srv.Exchanges()}
			if err != nil {
				out.Error = wiregolden.ErrorText(err, srv.URL)
			} else {
				var floats []float64
				for _, e := range v.(*eval.ListValue).Elements {
					floats = append(floats, e.(*eval.FloatValue).Value)
				}
				out.Result = floats
			}
			got[c.name] = out
		})
	}
	wiregolden.Check(t, "testdata/ollama_embed_wire.json", got)
}
