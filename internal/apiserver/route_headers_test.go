package apiserver

import (
	"io"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestRouteHeadersWireMatrix(t *testing.T) {
	setHeaderTestStdlib(t)
	path, err := filepath.Abs("../../examples/runnable/serve_api_response_headers.ail")
	if err != nil {
		t.Fatal(err)
	}
	srv := New(filepath.Dir(path), Config{CORS: true})
	t.Cleanup(func() { srv.Close() })
	if err := srv.LoadModules([]string{path}); err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(srv.buildRoutes())
	t.Cleanup(ts.Close)
	for _, name := range []string{"raw-record", "raw-json", "nowrap-record", "nowrap-json", "result"} {
		t.Run(name, func(t *testing.T) {
			resp, err := ts.Client().Get(ts.URL + "/headers/" + name)
			if err != nil {
				t.Fatal(err)
			}
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			if resp.StatusCode != 200 {
				t.Fatalf("status %d: %s", resp.StatusCode, body)
			}
			if resp.Header.Get("X-Frame-Options") != "DENY" || resp.Header.Get("Content-Security-Policy") != "frame-ancestors 'none'" {
				t.Errorf("missing framing headers: %v", resp.Header)
			}
			if strings.Contains(string(body), "_headers") {
				t.Errorf("metadata leaked: %s", body)
			}
			want := "text/plain; charset=utf-8"
			if strings.HasPrefix(name, "nowrap") {
				want = "application/json"
			}
			if resp.Header.Get("Content-Type") != want {
				t.Errorf("Content-Type=%q want %q", resp.Header.Get("Content-Type"), want)
			}
		})
	}
}
