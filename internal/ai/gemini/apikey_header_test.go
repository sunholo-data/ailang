package gemini

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ai"
)

// #1499: an AI Studio key is sent as the x-goog-api-key header (also to a
// custom base URL, which used to receive no key at all) and never appears in
// the request URL or in an error message.
func TestClient_APIKeyInHeaderNeverInURLOrError(t *testing.T) {
	const secret = "sk-very-secret-1499"
	var gotHeader, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Get("x-goog-api-key")
		gotQuery = r.URL.RawQuery
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"message":"API key not valid"}}`))
	}))
	defer srv.Close()

	client := NewClient(secret, WithBaseURL(srv.URL))
	_, err := client.Generate(context.Background(), &ai.Request{Model: "gemini-2.5-flash", UserPrompt: "hi"})
	if err == nil {
		t.Fatal("expected a 401 error")
	}
	if gotHeader != secret {
		t.Errorf("x-goog-api-key header = %q, want the key", gotHeader)
	}
	if strings.Contains(gotQuery, secret) {
		t.Errorf("key leaked into the query string: %q", gotQuery)
	}
	if strings.Contains(err.Error(), secret) {
		t.Errorf("key leaked into the error: %v", err)
	}

	// Transport failure: net/http quotes the URL in the error, so it must not
	// hold the key. Closed server = connection refused, no network needed.
	closed := httptest.NewServer(http.NotFoundHandler())
	closedURL := closed.URL
	closed.Close()
	unreachable := NewClient(secret, WithBaseURL(closedURL))
	_, err = unreachable.Generate(context.Background(), &ai.Request{Model: "gemini-2.5-flash", UserPrompt: "hi"})
	if err == nil || strings.Contains(err.Error(), secret) {
		t.Errorf("transport error must not echo the key: %v", err)
	}

	prod := NewClient(secret)
	for _, build := range []func(string) (string, error){prod.buildURL, prod.buildStreamURL} {
		if got, _ := build("m"); strings.Contains(got, secret) {
			t.Errorf("production URL carries the key: %q", got)
		}
	}
}
