// Package gcp provides Application Default Credentials (ADC) helpers shared
// by every AILANG consumer of Google Cloud APIs (currently: internal/ai/gemini
// for direct generateContent calls, and internal/executor/managed_agents for
// the Vertex Managed Agents API).
//
// The helpers prefer the GCE/Cloud Run metadata server (works in cloud
// without bundled credentials) and fall back to the gcloud CLI (local dev,
// CI). Both paths are exercised by the tests in adc_test.go.
//
// Extracted as part of M-MANAGED-AGENTS (v0.22.0) when the second consumer
// of ADC tokens forced the choice between duplication and a shared helper.
package gcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

// metadataClient is a short-timeout HTTP client for the GCE/Cloud Run
// metadata server. Two seconds matches the existing pattern in
// internal/ai/gemini/client.go — long enough to succeed in cloud, short
// enough to fall through quickly on a developer laptop where the metadata
// service is unreachable.
var metadataClient = &http.Client{Timeout: 2 * time.Second}

// AccessToken returns an ADC access token suitable for `Authorization: Bearer`
// headers when calling Google Cloud APIs. It prefers the GCE/Cloud Run
// metadata server (zero-config in cloud) and falls back to `gcloud auth
// application-default print-access-token` (local dev / CI).
//
// The context is honoured by the gcloud subprocess invocation; the metadata
// server call uses its own 2s timeout because the metadata service is meant
// to be fast or absent.
func AccessToken(ctx context.Context) (string, error) {
	// 1. Try the metadata server (Cloud Run, GKE, GCE).
	if token, err := accessTokenFromMetadata(ctx); err == nil && token != "" {
		return token, nil
	}

	// 2. Fall back to gcloud CLI (local dev).
	cmd := exec.CommandContext(ctx, "gcloud", "auth", "application-default", "print-access-token")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf(
			"gcp ADC: metadata server unavailable and gcloud failed (run `gcloud auth application-default login`): %w",
			err,
		)
	}

	token := strings.TrimSpace(string(output))
	if token == "" {
		return "", fmt.Errorf("gcp ADC: empty token from gcloud")
	}
	return token, nil
}

// accessTokenFromMetadata fetches a token from the GCE/Cloud Run metadata server.
func accessTokenFromMetadata(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		"http://metadata.google.internal/computeMetadata/v1/instance/service-accounts/default/token",
		nil,
	)
	if err != nil {
		return "", err
	}
	req.Header.Set("Metadata-Flavor", "Google")

	resp, err := metadataClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var payload struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", err
	}
	return payload.AccessToken, nil
}
