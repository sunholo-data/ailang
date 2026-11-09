package managed_agents

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/testutil"
)

// AC-M2.2 of m-net-effect-proxy-boundary: the production Vertex SSE client
// honors the process proxy environment and keeps its streaming timeouts.
//
// Go caches the proxy environment once per process, so the routing half runs
// in a subprocess whose HTTP(S)_PROXY points at a fake proxy started here. The
// target is a non-loopback hostname that only the fake proxy can answer: if
// defaultHTTPClient ignores the proxy, the helper fails instead of reaching the
// network (vertex.invalid never resolves).

const (
	maProxyHelperEnv = "AILANG_MA_PROXY_HELPER"
	maProxyBody      = "ma-via-proxy"
	maProxyTarget    = "http://vertex.invalid/v1/agents:stream"
)

func TestDefaultHTTPClientProxyBoundary(t *testing.T) {
	t.Run("timeouts_retained", func(t *testing.T) {
		c := defaultHTTPClient()
		if c.Timeout != 0 {
			t.Errorf("client.Timeout = %v, want 0 (no global timeout: the SSE stream runs as long as the Task allows)", c.Timeout)
		}
		tr, ok := c.Transport.(*http.Transport)
		if !ok {
			t.Fatalf("Transport = %T, want *http.Transport", c.Transport)
		}
		if tr.ResponseHeaderTimeout != 60*time.Second {
			t.Errorf("ResponseHeaderTimeout = %v, want 60s", tr.ResponseHeaderTimeout)
		}
		if tr.IdleConnTimeout != 90*time.Second {
			t.Errorf("IdleConnTimeout = %v, want 90s", tr.IdleConnTimeout)
		}
		if tr.Proxy == nil {
			t.Error("Transport.Proxy is nil: the Vertex client bypasses HTTP_PROXY/HTTPS_PROXY")
		}
	})

	t.Run("env_proxy_selected_and_observed", func(t *testing.T) {
		if os.Getenv(maProxyHelperEnv) != "" {
			t.Fatal("parent test must not run inside the helper subprocess")
		}
		var mu sync.Mutex
		var seen []string
		proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			seen = append(seen, r.Method+" "+r.URL.String())
			mu.Unlock()
			_, _ = io.WriteString(w, maProxyBody)
		}))
		defer proxy.Close()

		hctx, cancel := testutil.HangGuardContext(t, 90*time.Second)
		defer cancel()
		cmd := exec.CommandContext(hctx, os.Args[0], "-test.run=^TestDefaultHTTPClientProxyEnvHelper$", "-test.v", "-test.count=1")
		cmd.WaitDelay = 5 * time.Second
		cmd.Env = append(stripProxyEnvMA(os.Environ()),
			maProxyHelperEnv+"=1",
			"HTTP_PROXY="+proxy.URL,
			"HTTPS_PROXY="+proxy.URL,
			"NO_PROXY=",
		)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("helper failed: %v\n%s", err, out)
		}
		if !bytes.Contains(out, []byte("--- PASS: TestDefaultHTTPClientProxyEnvHelper")) {
			t.Fatalf("helper did not report PASS:\n%s", out)
		}
		mu.Lock()
		defer mu.Unlock()
		want := "POST " + maProxyTarget
		for _, s := range seen {
			if s == want {
				return
			}
		}
		t.Fatalf("fake proxy did not observe %q; saw %q", want, seen)
	})
}

// TestDefaultHTTPClientProxyEnvHelper is the subprocess half: proxy selection
// and the round trip both go through the PRODUCTION defaultHTTPClient().
func TestDefaultHTTPClientProxyEnvHelper(t *testing.T) {
	if os.Getenv(maProxyHelperEnv) == "" {
		t.Skip("managed-agents proxy-boundary environment helper (subprocess only)")
	}
	c := defaultHTTPClient()
	tr := c.Transport.(*http.Transport)
	req, err := http.NewRequest(http.MethodPost, maProxyTarget, strings.NewReader("{}"))
	if err != nil {
		t.Fatal(err)
	}
	if tr.Proxy == nil {
		t.Fatal("Transport.Proxy is nil: no proxy can be selected")
	}
	selected, err := tr.Proxy(req)
	if err != nil || selected == nil || selected.Host != strings.TrimPrefix(os.Getenv("HTTPS_PROXY"), "http://") {
		t.Fatalf("Proxy(req) = %v, %v; want the environment proxy %s", selected, err, os.Getenv("HTTPS_PROXY"))
	}
	resp, err := c.Do(req)
	if err != nil {
		t.Fatalf("POST through the environment proxy: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if string(body) != maProxyBody {
		t.Fatalf("body = %q, want the proxy-served %q", body, maProxyBody)
	}
}

// stripProxyEnvMA removes proxy variables (any case) so the subprocess starts
// with only the proxy this test sets.
func stripProxyEnvMA(base []string) []string {
	out := make([]string, 0, len(base))
	for _, kv := range base {
		name, _, _ := strings.Cut(kv, "=")
		switch strings.ToUpper(name) {
		case "HTTP_PROXY", "HTTPS_PROXY", "ALL_PROXY", "NO_PROXY":
			continue
		}
		out = append(out, kv)
	}
	return out
}
