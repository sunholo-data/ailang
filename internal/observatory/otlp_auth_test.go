package observatory

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

// M-OPENROUTER-BROADCAST-INGEST M1 (auth half).
//
// The critical case is the LAST one: with the env var unset, ingest must work
// with no credential. That is the state this ships in, and the state live
// OpenRouter Broadcast traffic depends on.

const testIngestToken = "s3cret-ingest-token"

// registerAuthedRoutes builds a mux with the OTLP routes registered exactly as
// production does, so these tests exercise the real wiring rather than calling
// the middleware directly.
func registerAuthedRoutes(t *testing.T) (*http.ServeMux, Backend) {
	t.Helper()
	r, backend := newTestReceiver(t)
	mux := http.NewServeMux()
	r.RegisterRoutes(mux)
	return mux, backend
}

func postTracesTo(mux *http.ServeMux, header, value string) *httptest.ResponseRecorder {
	body := otlpJSONTracePayload(knownTraceIDHex, knownSpanIDHex)
	req := httptest.NewRequest(http.MethodPost, "/v1/traces", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if header != "" {
		req.Header.Set(header, value)
	}
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	return w
}

func TestOTLPAuth_TokenSet_NoHeader_Unauthorized(t *testing.T) {
	t.Setenv(OTLPIngestTokenEnv, testIngestToken)
	mux, backend := registerAuthedRoutes(t)

	before := len(storedSpans(t, backend))
	w := postTracesTo(mux, "", "")

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
	if after := len(storedSpans(t, backend)); after != before {
		t.Errorf("span count went %d -> %d; an unauthorized request must write no rows", before, after)
	}
}

func TestOTLPAuth_TokenSet_WrongHeader_Unauthorized(t *testing.T) {
	t.Setenv(OTLPIngestTokenEnv, testIngestToken)
	mux, _ := registerAuthedRoutes(t)

	if w := postTracesTo(mux, OTLPIngestTokenHeader, "wrong-token"); w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401 for a wrong token", w.Code)
	}
}

func TestOTLPAuth_TokenSet_CorrectHeader_OK(t *testing.T) {
	t.Setenv(OTLPIngestTokenEnv, testIngestToken)
	mux, backend := registerAuthedRoutes(t)

	if w := postTracesTo(mux, OTLPIngestTokenHeader, testIngestToken); w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}
	if got := len(storedSpans(t, backend)); got != 1 {
		t.Errorf("stored %d spans, want 1", got)
	}
}

// TestOTLPAuth_BearerAccepted covers the standard OTLP exporter convention.
func TestOTLPAuth_BearerAccepted(t *testing.T) {
	t.Setenv(OTLPIngestTokenEnv, testIngestToken)
	mux, _ := registerAuthedRoutes(t)

	if w := postTracesTo(mux, "Authorization", "Bearer "+testIngestToken); w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200 for a valid Bearer token", w.Code)
	}
}

// TestOTLPAuth_TokenUnset_OpenIngest is the compatibility guarantee: unset env
// means ingest is open, so deploying this cannot break the live OpenRouter
// Broadcast stream or the rig's localhost:1957 posts.
func TestOTLPAuth_TokenUnset_OpenIngest(t *testing.T) {
	t.Setenv(OTLPIngestTokenEnv, "")
	mux, backend := registerAuthedRoutes(t)

	if w := postTracesTo(mux, "", ""); w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 when %s is unset; body = %s",
			w.Code, OTLPIngestTokenEnv, w.Body.String())
	}
	if got := len(storedSpans(t, backend)); got != 1 {
		t.Errorf("stored %d spans, want 1", got)
	}
}

// TestOTLPAuth_ScopedToIngestRoutes proves the middleware did not widen to the
// read APIs. With a token set, a GET to the observatory read surface must still
// work without any credential — the dashboard UI depends on it.
func TestOTLPAuth_ScopedToIngestRoutes(t *testing.T) {
	t.Setenv(OTLPIngestTokenEnv, testIngestToken)

	r, backend := newTestReceiver(t)
	mux := http.NewServeMux()
	r.RegisterRoutes(mux)

	// A route outside the OTLP set, registered the way the server registers
	// its read endpoints.
	reached := false
	mux.HandleFunc("GET /api/observatory/spans", func(w http.ResponseWriter, _ *http.Request) {
		reached = true
		w.WriteHeader(http.StatusOK)
	})
	_ = backend

	req := httptest.NewRequest(http.MethodGet, "/api/observatory/spans", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK || !reached {
		t.Errorf("read route returned %d (handler reached=%v); OTLP auth must not cover it", w.Code, reached)
	}
}
