package managed_agents

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	gcpauth "github.com/sunholo-data/ailang/internal/auth/gcp"
)

const (
	// vertexAIHost is the GCP host serving the Vertex AI Managed Agents API.
	vertexAIHost = "https://aiplatform.googleapis.com"

	// apiVersion is the URL version segment. The Managed Agents endpoint is
	// only available under v1beta1 as of 2026-05-20 — earlier prefixes
	// (v1, v1beta) return HTML 404.
	apiVersion = "v1beta1"

	// apiRevisionHeader pins the API behaviour to a specific revision. This
	// header is required on every request to /interactions and protects us
	// against schema drift.
	apiRevisionHeader = "Api-Revision"
	apiRevision       = "2026-05-20"

	// defaultLocation is the only currently-supported region for the Managed
	// Agents endpoint.
	defaultLocation = "global"

	// defaultAgent is the public agent name as of 2026-05-20. New agents will
	// appear over time and can be specified via Task.Model.
	defaultAgent = "antigravity-preview-05-2026"
)

// httpClient is the type used for HTTP transport. Defined as an interface so
// tests can swap in a stub server without spinning up TLS termination.
type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// tokenSource produces ADC bearer tokens. Defined as a function type so tests
// can substitute deterministic tokens without invoking the shared GCP helper.
type tokenSource func(context.Context) (string, error)

// defaultTokenSource is the production token source: the shared GCP ADC helper
// from internal/auth/gcp, which prefers the metadata server (Cloud Run, GKE,
// GCE) and falls back to gcloud CLI (local dev / CI).
func defaultTokenSource(ctx context.Context) (string, error) {
	tok, err := gcpauth.AccessToken(ctx)
	if err != nil {
		return "", fmt.Errorf("managed_agents: %w", err)
	}
	return tok, nil
}

// buildInteractionsURL constructs the POST URL for the project + location.
func buildInteractionsURL(project, location string) string {
	if location == "" {
		location = defaultLocation
	}
	return fmt.Sprintf("%s/%s/projects/%s/locations/%s/interactions",
		vertexAIHost, apiVersion, project, location)
}

// sendInteraction POSTs an interaction request and returns the raw response
// body for SSE parsing. The body is the live response stream — callers must
// close it (this function does NOT close it for them).
//
// On non-2xx responses the body is drained and an error returned containing
// the API's error message (parsed as errorPayload when possible).
func sendInteraction(
	ctx context.Context,
	client httpClient,
	tokens tokenSource,
	project, location string,
	body *interactionRequest,
) (io.ReadCloser, error) {
	if project == "" {
		return nil, fmt.Errorf("managed_agents: GCP project not set (Task.GCPProject or executor config)")
	}

	url := buildInteractionsURL(project, location)

	buf, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("managed_agents: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(buf))
	if err != nil {
		return nil, fmt.Errorf("managed_agents: build request: %w", err)
	}

	token, err := tokens(ctx)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set(apiRevisionHeader, apiRevision)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("managed_agents: HTTP do: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		// Drain + parse the JSON error body so the message bubbles up.
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		var ep errorPayload
		if json.Unmarshal(b, &ep) == nil && ep.Error.Message != "" {
			return nil, fmt.Errorf("managed_agents: HTTP %d: %s (code=%s)",
				resp.StatusCode, ep.Error.Message, ep.Error.Code)
		}
		preview := string(b)
		if len(preview) > 500 {
			preview = preview[:500] + "..."
		}
		return nil, fmt.Errorf("managed_agents: HTTP %d: %s", resp.StatusCode, preview)
	}

	return resp.Body, nil
}

// defaultHTTPClient is the production HTTP client. The streaming-friendly
// timeouts come from the Task itself — we don't impose a request-level
// timeout, because a long agentic run may stream events for minutes.
func defaultHTTPClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			ResponseHeaderTimeout: 60 * time.Second, // initial headers
			IdleConnTimeout:       90 * time.Second,
		},
		// No global Timeout — let the SSE stream run as long as Task.Timeout
		// allows. The caller's context handles cancellation.
	}
}
