package coordinator

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/websocket"
)

// /api/coordinator/events is a hub-class ingest route: the broadcaster
// presents AILANG_HUB_TOKEN when it is set and no ingest header when unset.
func TestHTTPBroadcaster_HubToken(t *testing.T) {
	for _, tc := range []struct{ name, token string }{
		{"set", "hub-test-token"},
		{"unset", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(config.EnvHubToken, tc.token)
			var got http.Header
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/api/coordinator/events" {
					got = r.Header.Clone()
				}
				w.WriteHeader(http.StatusOK)
			}))
			defer srv.Close()

			NewHTTPBroadcaster(srv.URL, nil).Broadcast(&websocket.TaskStreamEvent{TaskID: "t1", StreamType: "status"})

			if got == nil {
				t.Fatal("no POST reached /api/coordinator/events")
			}
			if v := got.Get(config.IngestTokenHeader); v != tc.token {
				t.Errorf("%s = %q, want %q", config.IngestTokenHeader, v, tc.token)
			}
			if _, present := got[http.CanonicalHeaderKey(config.IngestTokenHeader)]; tc.token == "" && present {
				t.Errorf("token unset: %s header must be absent", config.IngestTokenHeader)
			}
			if ct := got.Get("Content-Type"); ct != "application/json" {
				t.Errorf("Content-Type = %q, want application/json", ct)
			}
		})
	}
}
