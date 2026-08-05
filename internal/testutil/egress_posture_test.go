package testutil

import (
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
)

const egressPoisonProxy = "http://127.0.0.1:9"

func TestEgressPosture(t *testing.T) {
	t.Run("poison_sentinel_denies_HTTP_egress", testPoisonSentinel)
	t.Run("loopback_bypasses_lane_poison", testLoopbackBypass)
	t.Run("raw_TCP_remains_open", testRawTCPResidual)
	t.Run("effects_nil_proxy_remains_open", testEffectsProxyResidual)
}

func testPoisonSentinel(t *testing.T) {
	proxyURL, err := url.Parse(egressPoisonProxy)
	if err != nil {
		t.Fatalf("parse poison proxy: %v", err)
	}
	transport := &http.Transport{Proxy: http.ProxyURL(proxyURL)}
	t.Cleanup(transport.CloseIdleConnections)
	client := &http.Client{Transport: transport, Timeout: 5 * time.Second}

	_, err = client.Get("https://example.com")
	assertPoisonProxyError(t, err)
}

func testLoopbackBypass(t *testing.T) {
	if !laneIsPoisoned() {
		t.Skip("TestEgressPosture/loopback_bypasses_lane_poison requires HTTP_PROXY or HTTPS_PROXY=http://127.0.0.1:9")
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	transport := &http.Transport{Proxy: http.ProxyFromEnvironment}
	t.Cleanup(transport.CloseIdleConnections)
	response, err := (&http.Client{Transport: transport, Timeout: 5 * time.Second}).Get(server.URL)
	if err != nil {
		t.Fatalf("loopback GET under poisoned proxy: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusNoContent {
		t.Fatalf("loopback status = %d, want %d", response.StatusCode, http.StatusNoContent)
	}
}

func testRawTCPResidual(t *testing.T) {
	requireLiveEgressPosture(t, "raw_TCP_remains_open")
	connection, err := net.DialTimeout("tcp", "example.com:443", 10*time.Second)
	if err != nil {
		t.Fatalf("raw TCP residual unexpectedly closed: %v", err)
	}
	connection.Close()
}

func testEffectsProxyResidual(t *testing.T) {
	requireLiveEgressPosture(t, "effects_nil_proxy_remains_open")
	t.Setenv("HTTP_PROXY", egressPoisonProxy)
	t.Setenv("HTTPS_PROXY", egressPoisonProxy)

	// This first request intentionally trips red when Option B adds
	// ProxyFromEnvironment to internal/effects. That red means the residual has
	// closed; retire this tripwire and its Non-Goals text instead of "fixing" it.
	effectsTransport := &http.Transport{}
	t.Cleanup(effectsTransport.CloseIdleConnections)
	response, err := (&http.Client{Transport: effectsTransport, Timeout: 10 * time.Second}).Get("https://example.com")
	if err != nil {
		t.Fatalf("nil-Proxy transport should bypass poison while D5 Option A remains: %v", err)
	}
	response.Body.Close()

	controlTransport := &http.Transport{Proxy: http.ProxyFromEnvironment}
	t.Cleanup(controlTransport.CloseIdleConnections)
	_, err = (&http.Client{Transport: controlTransport, Timeout: 5 * time.Second}).Get("https://example.com")
	assertPoisonProxyError(t, err)
}

func requireLiveEgressPosture(t *testing.T, leg string) {
	t.Helper()
	if os.Getenv("AILANG_LIVE_NET") != "1" {
		t.Skipf("TestEgressPosture/%s requires AILANG_LIVE_NET=1", leg)
	}
}

func laneIsPoisoned() bool {
	return os.Getenv("HTTP_PROXY") == egressPoisonProxy || os.Getenv("HTTPS_PROXY") == egressPoisonProxy
}

func assertPoisonProxyError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("request through poison proxy unexpectedly succeeded")
	}
	message := err.Error()
	for _, required := range []string{"proxyconnect", "127.0.0.1:9", "connection refused"} {
		if !strings.Contains(strings.ToLower(message), required) {
			t.Fatalf("poison error %q does not contain %q", message, required)
		}
	}
	t.Logf("observed poison sentinel error: %v", err)
}
