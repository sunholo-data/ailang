package pkg

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/testutil"
)

type observedRequestBody struct {
	*bytes.Reader
	reads  int
	closed int
}

func (b *observedRequestBody) Read(p []byte) (int, error) {
	b.reads++
	return b.Reader.Read(p)
}

func (b *observedRequestBody) Close() error {
	b.closed++
	return nil
}

func TestConfinedHTTPClosesBodyWithoutReadingOrSending(t *testing.T) {
	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { hits.Add(1) }))
	defer server.Close()
	confine(t)
	body := &observedRequestBody{Reader: bytes.NewReader([]byte("private package metadata"))}
	req, err := http.NewRequest(http.MethodPost, server.URL, body)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := HTTPClient("publishing package metadata", time.Second).Do(req)
	if resp != nil {
		resp.Body.Close()
	}
	if !IsConfinedError(err) {
		t.Fatalf("POST must preserve the confinement refusal: %v", err)
	}
	if body.closed != 1 || body.reads != 0 || hits.Load() != 0 {
		t.Fatalf("refused body: closes=%d reads=%d server requests=%d", body.closed, body.reads, hits.Load())
	}
}

func TestPackageAuthorityFailsClosedWithoutHome(t *testing.T) {
	testutil.SetHomeDir(t, "")
	t.Setenv(config.EnvAgentPolicy, "policy.toml")
	if _, err := UserDir("cache/git"); err == nil {
		t.Fatal("missing home must not resolve a fallback cache")
	}
	outside := filepath.Join(t.TempDir(), "cache")
	if err := ensureDir(outside, "creating cache", GitRemedy); !IsConfinedError(err) {
		t.Fatalf("unknown home location must refuse writes: %v", err)
	}
	if _, err := os.Stat(outside); !os.IsNotExist(err) {
		t.Fatalf("refused cache was created: %v", err)
	}
	if _, err := Git(outside, "checkout", "v1"); !IsConfinedError(err) {
		t.Fatalf("unknown home location must refuse git writes before exec: %v", err)
	}
	t.Setenv(config.EnvAgentPolicy, "")
	if _, err := EnsureUserDir("cache/git"); err == nil {
		t.Fatal("operator cache creation must report missing home, not fall back")
	}
}

func TestEnsureUserDirReportsBlockedParent(t *testing.T) {
	home := t.TempDir()
	testutil.SetHomeDir(t, home)
	t.Setenv(config.EnvAgentPolicy, "")
	if err := os.WriteFile(filepath.Join(home, ".ailang"), []byte("not a directory"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := EnsureUserDir("cache/git"); err == nil {
		t.Fatal("cache creation must report a blocked parent")
	}
}
