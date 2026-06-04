package eval_harness

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

func TestStartHTTPMock_PostReturns200AndEchoes(t *testing.T) {
	srv := StartHTTPMock()
	defer srv.Close()

	body := []byte(`{"message":"Hello from ailang","count":42}`)
	req, err := http.NewRequest("POST", srv.URL+"/post", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-Header", "value123")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Deterministic 200 — the whole point of the fixture.
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var out map[string]any
	raw, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("response not JSON: %v (body=%s)", err, raw)
	}

	// Custom header echoed back.
	headers, _ := out["headers"].(map[string]any)
	if headers["X-Test-Header"] != "value123" {
		t.Errorf("X-Test-Header echo = %v, want value123", headers["X-Test-Header"])
	}

	// Posted JSON echoed back.
	echoed, _ := out["json"].(map[string]any)
	if echoed["message"] != "Hello from ailang" {
		t.Errorf("json.message echo = %v, want 'Hello from ailang'", echoed["message"])
	}
	if echoed["count"] != float64(42) {
		t.Errorf("json.count echo = %v, want 42", echoed["count"])
	}
}

func TestStartHTTPMock_GetAlsoReturns200(t *testing.T) {
	srv := StartHTTPMock()
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/anything")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
}

func TestStartHTTPMock_EphemeralPortsDistinct(t *testing.T) {
	// Concurrency safety: two servers must not share a port.
	a := StartHTTPMock()
	defer a.Close()
	b := StartHTTPMock()
	defer b.Close()
	if a.URL == b.URL {
		t.Fatalf("two mock servers share URL %s — ephemeral binding broken", a.URL)
	}
}

func TestPromptUsesHTTPMock(t *testing.T) {
	if !PromptUsesHTTPMock("POST to " + MockHTTPURLToken + " please") {
		t.Error("expected token to be detected")
	}
	if PromptUsesHTTPMock("no token here") {
		t.Error("false positive on prompt without token")
	}
}
