package observatory

import (
	"crypto/subtle"
	"net/http"
	"os"
	"strings"
)

// The OTLP ingest endpoints accept writes into the observatory database. In
// production (`ailang-dashboard` on Cloud Run) they are reachable from the
// public internet, so anyone who learns the URL can inject spans.
//
// This gates them behind a shared secret. The secret is a static header value
// rather than a signed token or Cloud Run IAM because the producer this exists
// for — OpenRouter Broadcast — can only attach static custom headers to a
// destination; IAM would exclude it.
//
// M-OPENROUTER-BROADCAST-INGEST M1.

const (
	// OTLPIngestTokenEnv holds the shared secret. When it is UNSET OR EMPTY,
	// authentication is DISABLED and ingest is open.
	//
	// That default is deliberate and load-bearing: Broadcast is already
	// streaming to production, and the rig posts to localhost:1957 with no
	// credential. Defaulting to "enforce" would break both the moment this
	// deploys. Enabling it requires setting this variable AND adding the
	// matching header on every producer — both sides, or ingest stops.
	OTLPIngestTokenEnv = "AILANG_OTLP_INGEST_TOKEN"

	// OTLPIngestTokenHeader is the primary header carrying the shared secret.
	OTLPIngestTokenHeader = "X-AILANG-Ingest-Token"
)

// otlpIngestToken returns the configured shared secret, or "" when ingest auth
// is disabled.
func otlpIngestToken() string {
	return strings.TrimSpace(os.Getenv(OTLPIngestTokenEnv))
}

// presentedIngestToken extracts the caller's credential.
//
// Both the AILANG-specific header and the standard `Authorization: Bearer`
// form are accepted, because OTLP exporters (including the Go SDK via
// OTEL_EXPORTER_OTLP_HEADERS) conventionally use the latter.
func presentedIngestToken(r *http.Request) string {
	if v := strings.TrimSpace(r.Header.Get(OTLPIngestTokenHeader)); v != "" {
		return v
	}
	if auth := strings.TrimSpace(r.Header.Get("Authorization")); auth != "" {
		if rest, ok := strings.CutPrefix(auth, "Bearer "); ok {
			return strings.TrimSpace(rest)
		}
	}
	return ""
}

// requireOTLPIngestAuth wraps an OTLP handler with the shared-secret check.
//
// The token is read per-request rather than captured at registration so that
// tests (and a redeploy) can change it without rebuilding the mux.
func requireOTLPIngestAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		expected := otlpIngestToken()
		if expected == "" {
			// Auth disabled — the documented default.
			next(w, r)
			return
		}

		presented := presentedIngestToken(r)
		if subtle.ConstantTimeCompare([]byte(presented), []byte(expected)) != 1 {
			http.Error(w, "unauthorized: OTLP ingest requires a valid "+OTLPIngestTokenHeader, http.StatusUnauthorized)
			return
		}

		next(w, r)
	}
}
