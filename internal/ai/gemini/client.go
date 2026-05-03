package gemini

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/ai"
	"github.com/sunholo-data/ailang/internal/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var geminiAITracer = telemetry.Tracer("ai.gemini")

const (
	// AI Studio base URL (uses API key)
	aiStudioBaseURL = "https://generativelanguage.googleapis.com/v1beta"

	// Vertex AI base URL (uses ADC)
	vertexAIBaseURL = "https://aiplatform.googleapis.com/v1"
)

// Client implements ai.Provider for Google's Gemini API.
// It supports both AI Studio and Vertex AI endpoints.
type Client struct {
	apiKey     string   // API key for AI Studio
	projectID  string   // GCP project for Vertex AI
	location   string   // GCP location (default: "global")
	authType   AuthType // Authentication type
	httpClient *http.Client
	baseURL    string // Override base URL (for testing)
}

// ClientOption configures a Client.
type ClientOption func(*Client)

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(client *http.Client) ClientOption {
	return func(c *Client) {
		c.httpClient = client
	}
}

// WithLocation sets the GCP location for Vertex AI.
func WithLocation(location string) ClientOption {
	return func(c *Client) {
		c.location = location
	}
}

// WithBaseURL sets a custom base URL (useful for testing).
func WithBaseURL(url string) ClientOption {
	return func(c *Client) {
		c.baseURL = url
	}
}

// NewClient creates a new Gemini client using API key (AI Studio).
func NewClient(apiKey string, opts ...ClientOption) *Client {
	c := &Client{
		apiKey:     apiKey,
		authType:   AuthAPIKey,
		location:   "global",
		httpClient: http.DefaultClient,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// NewVertexAIClient creates a new Gemini client using ADC (Vertex AI).
// If projectID is empty, it will be fetched from gcloud config.
func NewVertexAIClient(projectID string, opts ...ClientOption) (*Client, error) {
	c := &Client{
		projectID:  projectID,
		authType:   AuthADC,
		location:   "global",
		httpClient: http.DefaultClient,
	}
	for _, opt := range opts {
		opt(c)
	}

	// Get project ID from gcloud if not provided
	if c.projectID == "" {
		project, err := getGCPProject()
		if err != nil {
			return nil, ai.NewProviderError("gemini", 0, "failed to get GCP project", err)
		}
		c.projectID = project
	}

	return c, nil
}

// Generate implements ai.Provider.
func (c *Client) Generate(ctx context.Context, req *ai.Request) (*ai.Response, error) {
	if req.Routing != nil && req.Routing.HasRouting() {
		return nil, ai.NewProviderError("gemini", 0,
			"this provider does not support AIRoutingPolicy; use openrouter instead",
			ai.ErrRoutingNotSupported)
	}
	// Start OTEL span
	ctx, span := telemetry.StartSpan(ctx, geminiAITracer, "gemini.generate",
		trace.WithAttributes(
			attribute.String("ai.provider", "gemini"),
			attribute.String("ai.model", req.Model),
			attribute.String("ai.auth_type", string(c.authType)),
			attribute.String("ai.prompt_preview", telemetry.Truncate(req.UserPrompt, 100)),
		),
	)
	defer span.End()

	resp, err := c.generateContent(ctx, req)
	if err != nil {
		span.SetAttributes(
			attribute.String("error.message", telemetry.Truncate(err.Error(), 200)),
			attribute.String("error.category", telemetry.CategorizeError(err)),
		)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	// Record success metrics on span
	span.SetAttributes(
		attribute.Int("ai.tokens_in", resp.InputTokens),
		attribute.Int("ai.tokens_out", resp.OutputTokens),
		attribute.Int("ai.tokens_total", resp.TotalTokens),
		attribute.String("ai.response_preview", telemetry.Truncate(resp.Text, 100)),
	)

	return resp, nil
}

// Name implements ai.Provider.
func (c *Client) Name() string {
	return "gemini"
}

// NewHandler creates an ai.Handler wrapping this client.
func (c *Client) NewHandler(model string, opts ...ai.HandlerOption) *ai.Handler {
	return ai.NewHandler(c, model, opts...)
}

// metadataClient is a short-timeout HTTP client for the GCE/Cloud Run metadata server.
var metadataClient = &http.Client{Timeout: 2 * time.Second}

// getAccessToken retrieves an access token for Vertex AI.
// Tries: (1) GCE/Cloud Run metadata server, (2) gcloud CLI fallback.
func getAccessToken() (string, error) {
	// 1. Try metadata server (Cloud Run, GKE, GCE)
	if token, err := getTokenFromMetadata(); err == nil && token != "" {
		return token, nil
	}

	// 2. Fall back to gcloud CLI (local dev)
	cmd := exec.Command("gcloud", "auth", "application-default", "print-access-token")
	output, err := cmd.Output()
	if err != nil {
		return "", ai.NewProviderError("gemini", 0,
			"failed to get access token: metadata server unavailable and gcloud failed", err)
	}

	token := strings.TrimSpace(string(output))
	if token == "" {
		return "", ai.NewProviderError("gemini", 0, "empty token from gcloud", nil)
	}

	return token, nil
}

// getTokenFromMetadata fetches an access token from the GCE/Cloud Run metadata server.
func getTokenFromMetadata() (string, error) {
	req, err := http.NewRequest("GET",
		"http://metadata.google.internal/computeMetadata/v1/instance/service-accounts/default/token", nil)
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

	var tokenResp struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", err
	}
	return tokenResp.AccessToken, nil
}

// getGCPProject gets the current GCP project ID.
// Tries: (1) GOOGLE_CLOUD_PROJECT env var, (2) GCP_PROJECT env var,
// (3) GCE/Cloud Run metadata server, (4) gcloud CLI fallback.
func getGCPProject() (string, error) {
	// 1. GOOGLE_CLOUD_PROJECT env var (set by Cloud Run, GKE, App Engine)
	if project := os.Getenv("GOOGLE_CLOUD_PROJECT"); project != "" {
		return project, nil
	}

	// 2. GCP_PROJECT env var (alternate convention)
	if project := os.Getenv("GCP_PROJECT"); project != "" {
		return project, nil
	}

	// 3. Metadata server (Cloud Run, GKE, GCE)
	if project, err := getProjectFromMetadata(); err == nil && project != "" {
		return project, nil
	}

	// 4. Fall back to gcloud CLI (local dev)
	cmd := exec.Command("gcloud", "config", "get-value", "project")
	output, err := cmd.Output()
	if err != nil {
		return "", ai.NewProviderError("gemini", 0,
			"no GCP project: set GOOGLE_CLOUD_PROJECT env var, or run 'gcloud config set project PROJECT'", err)
	}

	project := strings.TrimSpace(string(output))
	if project == "" {
		return "", ai.NewProviderError("gemini", 0, "no GCP project set", nil)
	}

	return project, nil
}

// getProjectFromMetadata fetches the project ID from the GCE/Cloud Run metadata server.
func getProjectFromMetadata() (string, error) {
	req, err := http.NewRequest("GET",
		"http://metadata.google.internal/computeMetadata/v1/project/project-id", nil)
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
	return strings.TrimSpace(string(body)), nil
}
