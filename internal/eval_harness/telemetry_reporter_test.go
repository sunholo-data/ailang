package eval_harness

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/config"
)

// /api/telemetry is a hub-class ingest route: the reporter presents
// AILANG_HUB_TOKEN when it is set and no ingest header when unset.
func TestTelemetryReporter_HubToken(t *testing.T) {
	for _, tc := range []struct{ name, token string }{
		{"set", "hub-test-token"},
		{"unset", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(config.EnvHubToken, tc.token)
			got := make(chan http.Header, 1)
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/api/telemetry" {
					select {
					case got <- r.Header.Clone():
					default:
					}
				}
				w.WriteHeader(http.StatusOK)
			}))
			defer srv.Close()

			NewTelemetryReporter(srv.URL, "test-instance").SetStatus("running")

			var h http.Header
			select {
			case h = <-got:
			case <-time.After(5 * time.Second):
				t.Fatal("no POST reached /api/telemetry")
			}
			if v := h.Get(config.IngestTokenHeader); v != tc.token {
				t.Errorf("%s = %q, want %q", config.IngestTokenHeader, v, tc.token)
			}
			if _, present := h[http.CanonicalHeaderKey(config.IngestTokenHeader)]; tc.token == "" && present {
				t.Errorf("token unset: %s header must be absent", config.IngestTokenHeader)
			}
		})
	}
}
