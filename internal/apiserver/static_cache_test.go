package apiserver

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestParseStaticCache(t *testing.T) {
	cases := []struct {
		in, want string
		err      bool
	}{
		{"", "", false},
		{"immutable", "public, max-age=31536000, immutable", false},
		{"IMMUTABLE", "public, max-age=31536000, immutable", false},
		{"3600", "public, max-age=3600", false},
		{"31536000", "public, max-age=31536000", false},
		{"0", "", true},
		{"-5", "", true},
		{"31536001", "", true},
		{"1h", "", true},
		{"forever", "", true},
	}
	for _, c := range cases {
		got, err := ParseStaticCache(c.in)
		if (err != nil) != c.err || got != c.want {
			t.Errorf("ParseStaticCache(%q) = %q, %v; want %q, err=%v", c.in, got, err, c.want, c.err)
		}
	}
}

func staticServer(t *testing.T, cacheValue string) *httptest.Server {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "clip-2026-09-26.mp4"), []byte("0123456789"), 0o644); err != nil {
		t.Fatal(err)
	}
	cc, err := ParseStaticCache(cacheValue)
	if err != nil {
		t.Fatal(err)
	}
	srv := New(t.TempDir(), Config{Port: "0", StaticPath: dir, StaticCache: cc})
	t.Cleanup(func() { _ = srv.Close() })
	ts := httptest.NewServer(srv.buildRoutes())
	t.Cleanup(ts.Close)
	return ts
}

func head(t *testing.T, url string, hdr map[string]string) *http.Response {
	t.Helper()
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	return resp
}

func TestStaticCache_SetOnSuccessOnly(t *testing.T) {
	ts := staticServer(t, "immutable")
	want := "public, max-age=31536000, immutable"

	ok := head(t, ts.URL+"/clip-2026-09-26.mp4", nil)
	if ok.StatusCode != 200 || ok.Header.Get("Cache-Control") != want {
		t.Fatalf("200: status %d Cache-Control %q", ok.StatusCode, ok.Header.Get("Cache-Control"))
	}
	partial := head(t, ts.URL+"/clip-2026-09-26.mp4", map[string]string{"Range": "bytes=0-3"})
	if partial.StatusCode != 206 || partial.Header.Get("Cache-Control") != want {
		t.Fatalf("206: status %d Cache-Control %q", partial.StatusCode, partial.Header.Get("Cache-Control"))
	}
	notMod := head(t, ts.URL+"/clip-2026-09-26.mp4", map[string]string{"If-Modified-Since": ok.Header.Get("Last-Modified")})
	if notMod.StatusCode != 304 || notMod.Header.Get("Cache-Control") != want {
		t.Fatalf("304: status %d Cache-Control %q", notMod.StatusCode, notMod.Header.Get("Cache-Control"))
	}
	listing := head(t, ts.URL+"/", nil)
	if listing.StatusCode != 200 || listing.Header.Get("Cache-Control") != "" {
		t.Fatalf("a directory listing changes as files are added; must not be cached: status %d Cache-Control %q", listing.StatusCode, listing.Header.Get("Cache-Control"))
	}
	missing := head(t, ts.URL+"/nope.mp4", nil)
	if missing.StatusCode != 404 || missing.Header.Get("Cache-Control") != "" {
		t.Fatalf("404 must not be cacheable: status %d Cache-Control %q", missing.StatusCode, missing.Header.Get("Cache-Control"))
	}
}

func TestStaticCache_NoneWithoutFlag(t *testing.T) {
	ts := staticServer(t, "")
	resp := head(t, ts.URL+"/clip-2026-09-26.mp4", nil)
	if resp.StatusCode != 200 {
		t.Fatalf("instrument check: HTTP %d", resp.StatusCode)
	}
	if cc := resp.Header.Get("Cache-Control"); cc != "" {
		t.Fatalf("no --static-cache must mean no Cache-Control, got %q", cc)
	}
}
