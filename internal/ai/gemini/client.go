package gemini

import (
	"context"
	"net/http"
	"os/exec"
	"strings"

	"github.com/sunholo/ailang/internal/ai"
)

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
	return c.generateContent(ctx, req)
}

// Name implements ai.Provider.
func (c *Client) Name() string {
	return "gemini"
}

// NewHandler creates an ai.Handler wrapping this client.
func (c *Client) NewHandler(model string, opts ...ai.HandlerOption) *ai.Handler {
	return ai.NewHandler(c, model, opts...)
}

// getAccessToken retrieves an access token from gcloud ADC.
func getAccessToken() (string, error) {
	cmd := exec.Command("gcloud", "auth", "application-default", "print-access-token")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	token := strings.TrimSpace(string(output))
	if token == "" {
		return "", ai.NewProviderError("gemini", 0, "empty token from gcloud", nil)
	}

	return token, nil
}

// getGCPProject gets the current GCP project ID from gcloud config.
func getGCPProject() (string, error) {
	cmd := exec.Command("gcloud", "config", "get-value", "project")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	project := strings.TrimSpace(string(output))
	if project == "" {
		return "", ai.NewProviderError("gemini", 0, "no GCP project set", nil)
	}

	return project, nil
}
