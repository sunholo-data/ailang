package embed

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestEmbedRequest(t *testing.T) {
	for _, tc := range []struct {
		name     string
		input    any
		response string
		status   int
		wantErr  string
	}{
		{"text", "hello", `{"embeddings":[[0.5,-1.25]]}`, 200, ""},
		{"empty text", "", `{"embeddings":[[0.5,-1.25]]}`, 200, ""},
		{"media", MediaInput("caption", []byte{1}, []byte{2}), `{"embeddings":[[0.5,-1.25]]}`, 200, ""},
		{"empty response", "hi", `{"embeddings":[]}`, 200, "no embeddings returned"},
		{"server error", "hi", `{"error":"model missing"}`, 404, "Ollama embed failed: model missing"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/embed" {
					t.Errorf("path %s", r.URL.Path)
				}
				var body struct {
					Model string `json:"model"`
					Input any    `json:"input"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
				}
				if body.Model != "test" || !reflect.DeepEqual(body.Input, toJSON(t, tc.input)) {
					t.Errorf("payload: %+v", body)
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.response))
			}))
			defer server.Close()
			t.Setenv("OLLAMA_HOST", server.URL)
			got, err := EmbedRequest("_ollama_embed", "test", tc.input, time.Second)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("error %v", err)
				}
				return
			}
			if err != nil || !reflect.DeepEqual(got, []float64{0.5, -1.25}) {
				t.Fatalf("got %v, %v", got, err)
			}
		})
	}
}
func toJSON(t *testing.T, v any) any {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	var out any
	if err = json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestMediaInput(t *testing.T) {
	if got := MediaInput("caption", []byte{1}, []byte{2}); !reflect.DeepEqual(got, map[string]string{"text": "caption", "image": "AQ==", "audio": "Ag=="}) {
		t.Fatal(got)
	}
	if got := MediaInput("", nil, []byte{}); len(got) != 0 {
		t.Fatal(got)
	}
}
