package configdriven

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/sunholo-data/ailang/internal/ai"
	"github.com/sunholo-data/ailang/internal/pkg"
)

func TestFrozenAuthEnvironment_MissingDoesNotAdoptLateCredential(t *testing.T) {
	t.Setenv("FROZEN_AUTH_TEST_KEY", "")
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.Write([]byte(`{"text":"unexpected"}`))
	}))
	defer server.Close()
	spec := &pkg.AIProviderSpec{Name: "frozen", Endpoint: server.URL, RequestShape: "simple_completion", ResponsePath: "$.text", Auth: pkg.AIProviderAuth{Type: "bearer", Env: "FROZEN_AUTH_TEST_KEY"}}
	provider := New(spec, WithFrozenAuthEnvironment())
	t.Setenv("FROZEN_AUTH_TEST_KEY", "late-key")
	if _, err := provider.Generate(context.Background(), &ai.Request{Model: "test", UserPrompt: "hi"}); err == nil {
		t.Fatal("adopted late credential")
	}
	if requests.Load() != 0 {
		t.Fatal("missing auth contacted provider")
	}
	// An explicit eager provider retains its original per-call environment behavior.
	if _, err := New(spec).Generate(context.Background(), &ai.Request{Model: "test", UserPrompt: "hi"}); err != nil {
		t.Fatal(err)
	}
	if requests.Load() != 1 {
		t.Fatalf("requests=%d", requests.Load())
	}
}

func TestFrozenAuthEnvironment_StreamRequestUsesSameSnapshot(t *testing.T) {
	t.Setenv("FROZEN_STREAM_KEY", "selected-stream-key")
	spec := &pkg.AIProviderSpec{Name: "frozen", Endpoint: "http://127.0.0.1:1", RequestShape: "simple_completion", ResponsePath: "$.text", AuthHeaders: map[string]string{"X-Auth": "prefix ${FROZEN_STREAM_KEY}"}, Capabilities: pkg.AIProviderCapabilities{Streaming: true}, Streaming: pkg.AIProviderStreaming{Enabled: true}}
	provider := New(spec, WithFrozenAuthEnvironment())
	t.Setenv("FROZEN_STREAM_KEY", "late-stream-key")
	messages := `[{"role":"user","content":"hi"}]`
	frozen, err := provider.BuildStreamRequest("test", messages)
	if err != nil {
		t.Fatal(err)
	}
	eager, err := BuildStreamRequest(spec, "test", messages)
	if err != nil {
		t.Fatal(err)
	}
	find := func(req *StreamRequest) string {
		for _, header := range req.Headers {
			if header.Name == "X-Auth" {
				return header.Value
			}
		}
		return ""
	}
	if find(frozen) != "prefix selected-stream-key" || find(eager) != "prefix late-stream-key" {
		t.Fatalf("frozen=%q eager=%q", find(frozen), find(eager))
	}
}
